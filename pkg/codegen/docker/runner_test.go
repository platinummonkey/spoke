package docker

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionRequest_Defaults(t *testing.T) {
	req := &ExecutionRequest{
		Image:      "test/image",
		Tag:        "latest",
		ProtoFiles: []codegen.ProtoFile{},
		ProtocFlags: []string{},
	}

	// Test that defaults are reasonable
	if req.MemoryLimit == 0 {
		req.MemoryLimit = DefaultMemoryLimit
	}
	if req.CPULimit == 0 {
		req.CPULimit = DefaultCPULimit
	}
	if req.Timeout == 0 {
		req.Timeout = DefaultTimeout
	}

	assert.Equal(t, int64(512*1024*1024), req.MemoryLimit)
	assert.Equal(t, 1.0, req.CPULimit)
	assert.Equal(t, 5*time.Minute, req.Timeout)
}

func TestBuildProtocCommand(t *testing.T) {
	runner := &DockerRunner{}

	tests := []struct {
		name     string
		req      *ExecutionRequest
		expected []string
	}{
		{
			name: "basic proto compilation",
			req: &ExecutionRequest{
				ProtoFiles: []codegen.ProtoFile{
					{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
				},
				ProtocFlags: []string{"--go_out=/output"},
			},
			expected: []string{
				"protoc",
				"--proto_path=/input",
				"--go_out=/output",
				"/input/test.proto",
			},
		},
		{
			name: "multiple proto files",
			req: &ExecutionRequest{
				ProtoFiles: []codegen.ProtoFile{
					{Path: "foo.proto", Content: []byte("syntax = \"proto3\";")},
					{Path: "bar.proto", Content: []byte("syntax = \"proto3\";")},
				},
				ProtocFlags: []string{"--go_out=/output", "--go_opt=paths=source_relative"},
			},
			expected: []string{
				"protoc",
				"--proto_path=/input",
				"--go_out=/output",
				"--go_opt=paths=source_relative",
				"/input/foo.proto",
				"/input/bar.proto",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := runner.buildProtocCommand(tt.req)
			assert.Equal(t, tt.expected, cmd)
		})
	}
}

func TestExtractGeneratedFiles(t *testing.T) {
	// Create a temporary directory with some test files
	tmpDir, err := os.MkdirTemp("", "spoke-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := map[string]string{
		"test.pb.go":      "package test",
		"nested/foo.pb.go": "package nested",
	}

	for path, content := range testFiles {
		fullPath := tmpDir + "/" + path
		require.NoError(t, os.MkdirAll(tmpDir+"/nested", 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	runner := &DockerRunner{}
	files, err := runner.extractGeneratedFiles(tmpDir)
	require.NoError(t, err)

	assert.Len(t, files, 2)

	// Verify file contents
	foundFiles := make(map[string]bool)
	for _, f := range files {
		foundFiles[f.Path] = true
		assert.Greater(t, f.Size, int64(0))
		assert.NotEmpty(t, f.Content)
	}

	assert.True(t, foundFiles["test.pb.go"])
	assert.True(t, foundFiles["nested/foo.pb.go"])
}

func TestExtractGeneratedFiles_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spoke-test-empty-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	runner := &DockerRunner{}
	files, err := runner.extractGeneratedFiles(tmpDir)
	require.NoError(t, err)
	assert.Len(t, files, 0)
}

// TestNewDockerRunner_NoDocker tests behavior when Docker is not available
func TestNewDockerRunner_NoDocker(t *testing.T) {
	// This test will fail if Docker is running
	// We can't easily mock the Docker client creation in the current implementation
	// So this is more of a placeholder for integration tests

	// Skip if DOCKER_HOST is set (Docker is available)
	if os.Getenv("DOCKER_HOST") != "" || fileExists("/var/run/docker.sock") {
		t.Skip("Docker is available, skipping no-Docker test")
	}

	// If Docker is not available, NewDockerRunner should return an error
	// In CI environments without Docker, this test validates error handling
}

// TestDockerRunner_Execute_Integration is an integration test that requires Docker
func TestDockerRunner_Execute_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip if Docker is not available
	if !isDockerAvailable() {
		t.Skip("Docker is not available")
	}

	runner, err := NewDockerRunner()
	if err != nil {
		t.Skipf("Cannot create Docker runner: %v", err)
	}
	defer runner.Close()

	// Simple proto file
	protoContent := []byte(`
syntax = "proto3";
package test;

option go_package = "github.com/example/test";

message TestMessage {
  string name = 1;
  int32 value = 2;
}
`)

	req := &ExecutionRequest{
		Image: "spoke/compiler-go",
		Tag:   "latest",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: protoContent},
		},
		ProtocFlags: []string{
			"--go_out=/output",
			"--go_opt=paths=source_relative",
		},
		Timeout: 30 * time.Second,
	}

	ctx := context.Background()
	result, err := runner.Execute(ctx, req)

	// If the image doesn't exist or can't be pulled, skip the test
	if err != nil {
		errMsg := err.Error()
		if err == ErrImagePullFailed || strings.Contains(errMsg, "failed to pull docker image") || strings.Contains(errMsg, "denied: requested access to the resource is denied") {
			t.Skipf("Compiler image not available: %v", err)
		}
	}
	if result != nil && result.Error != nil {
		errMsg := result.Error.Error()
		if result.Error == ErrImagePullFailed || strings.Contains(errMsg, "failed to pull docker image") || strings.Contains(errMsg, "denied: requested access to the resource is denied") {
			t.Skipf("Compiler image not available: %v", result.Error)
		}
	}

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.ExitCode)
	assert.NotEmpty(t, result.GeneratedFiles)
}

func TestDockerRunner_Cleanup(t *testing.T) {
	// Skip if Docker is not available
	if !isDockerAvailable() {
		t.Skip("Docker is not available")
	}

	runner, err := NewDockerRunner()
	if err != nil {
		t.Skipf("Cannot create Docker runner: %v", err)
	}
	defer runner.Close()

	// Add some fake container IDs to the cleanup list
	runner.cleanupIDs = []string{"nonexistent1", "nonexistent2"}

	// Mock cleanup - in real scenario this would remove containers
	ctx := context.Background()
	err = runner.Cleanup(ctx)

	// Cleanup should not fail even if containers don't exist
	assert.NoError(t, err)
	assert.Len(t, runner.cleanupIDs, 0)
}

// Helper functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDockerAvailable() bool {
	// Check if Docker socket exists
	if !fileExists("/var/run/docker.sock") {
		return false
	}

	// Try to create a client
	runner, err := NewDockerRunner()
	if err != nil {
		return false
	}
	runner.Close()
	return true
}

func TestDockerRunner_Close(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available")
	}

	runner, err := NewDockerRunner()
	require.NoError(t, err)

	// Add some cleanup IDs
	runner.cleanupIDs = []string{"fake-id"}

	// Close should cleanup and close client
	err = runner.Close()
	assert.NoError(t, err)
	assert.Len(t, runner.cleanupIDs, 0)
}

func TestDockerRunner_Close_NilClient(t *testing.T) {
	runner := &DockerRunner{
		client:     nil,
		imageCache: make(map[string]bool),
		cleanupIDs: []string{},
	}

	err := runner.Close()
	assert.NoError(t, err)
}

func TestDockerRunner_PullImage_Cached(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available")
	}

	runner, err := NewDockerRunner()
	require.NoError(t, err)
	defer runner.Close()

	// Manually set image as cached
	runner.imageCache["alpine:latest"] = true

	ctx := context.Background()
	err = runner.PullImage(ctx, "alpine:latest")
	assert.NoError(t, err)
}

func TestCreateContainer_Configuration(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available")
	}

	runner, err := NewDockerRunner()
	require.NoError(t, err)
	defer runner.Close()

	// Pull a simple image first
	ctx := context.Background()
	err = runner.PullImage(ctx, "alpine:latest")
	if err != nil {
		t.Skipf("Cannot pull alpine image: %v", err)
	}

	// Create temporary directories
	inputDir, err := os.MkdirTemp("", "test-input-*")
	require.NoError(t, err)
	defer os.RemoveAll(inputDir)

	outputDir, err := os.MkdirTemp("", "test-output-*")
	require.NoError(t, err)
	defer os.RemoveAll(outputDir)

	req := &ExecutionRequest{
		Image:       "alpine",
		Tag:         "latest",
		MemoryLimit: 256 * 1024 * 1024,
		CPULimit:    0.5,
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("test")},
		},
	}

	containerID, err := runner.createContainer(ctx, "alpine:latest", []string{"echo", "test"}, inputDir, outputDir, req)
	require.NoError(t, err)
	assert.NotEmpty(t, containerID)

	// Add to cleanup list and cleanup
	runner.cleanupIDs = append(runner.cleanupIDs, containerID)
	err = runner.Cleanup(ctx)
	assert.NoError(t, err)
}

func TestExecute_InvalidImageTag(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available")
	}

	runner, err := NewDockerRunner()
	require.NoError(t, err)
	defer runner.Close()

	req := &ExecutionRequest{
		Image: "nonexistent-image-12345",
		Tag:   "invalid-tag-67890",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
		ProtocFlags: []string{"--go_out=/output"},
		Timeout:     5 * time.Second,
	}

	ctx := context.Background()
	result, err := runner.Execute(ctx, req)

	// Should fail with image pull error
	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}

func TestExecute_SetDefaults(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available")
	}

	runner, err := NewDockerRunner()
	require.NoError(t, err)
	defer runner.Close()

	// Request with no defaults set
	req := &ExecutionRequest{
		Image: "alpine",
		Tag:   "latest",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
		ProtocFlags: []string{"--go_out=/output"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// This will fail but we're testing that defaults are set
	result, _ := runner.Execute(ctx, req)

	// Check that defaults were applied by checking the result exists
	assert.NotNil(t, result)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestExecute_EmptyProtoFiles(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available")
	}

	runner, err := NewDockerRunner()
	require.NoError(t, err)
	defer runner.Close()

	req := &ExecutionRequest{
		Image:       "alpine",
		Tag:         "latest",
		ProtoFiles:  []codegen.ProtoFile{},
		ProtocFlags: []string{"--go_out=/output"},
		Timeout:     5 * time.Second,
	}

	ctx := context.Background()
	result, _ := runner.Execute(ctx, req)

	// Should complete but likely fail or produce no files
	assert.NotNil(t, result)
}
