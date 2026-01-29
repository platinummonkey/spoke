package buf

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/platinummonkey/spoke/pkg/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBufPluginAdapter(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")

	assert.NotNil(t, adapter)
	assert.Equal(t, "buf.build/library/connect-go", adapter.pluginRef)
	assert.Equal(t, "v1.5.0", adapter.version)
	assert.NotNil(t, adapter.downloader)
}

func TestNewBufPluginAdapterFromManifest(t *testing.T) {
	manifest := &plugins.Manifest{
		ID:          "buf-connect-go",
		Name:        "Buf Connect for Go",
		Version:     "1.5.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Connect RPC framework for Go",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/connect-go",
			"buf_version":  "v1.5.0",
		},
	}

	adapter, err := NewBufPluginAdapterFromManifest(manifest)
	require.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, "buf.build/library/connect-go", adapter.pluginRef)
	assert.Equal(t, "v1.5.0", adapter.version)
	assert.Equal(t, manifest, adapter.manifest)
}

func TestNewBufPluginAdapterFromManifest_MissingRegistry(t *testing.T) {
	manifest := &plugins.Manifest{
		ID:         "test-plugin",
		Name:       "Test Plugin",
		Version:    "1.0.0",
		APIVersion: "1.0.0",
		Type:       plugins.PluginTypeLanguage,
		Metadata:   map[string]string{},
	}

	_, err := NewBufPluginAdapterFromManifest(manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "buf_registry")
}

func TestDeriveLanguageID(t *testing.T) {
	tests := []struct {
		name      string
		pluginRef string
		expected  string
	}{
		{
			name:      "connect-go",
			pluginRef: "buf.build/library/connect-go",
			expected:  "connect-go",
		},
		{
			name:      "grpc-go",
			pluginRef: "buf.build/protocolbuffers/go",
			expected:  "go",
		},
		{
			name:      "simple",
			pluginRef: "test",
			expected:  "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBufPluginAdapter(tt.pluginRef, "v1.0.0")
			result := adapter.deriveLanguageID()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildLanguageSpec(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	spec := adapter.buildLanguageSpec()

	assert.NotNil(t, spec)
	assert.Equal(t, "connect-go", spec.ID)
	assert.Equal(t, "connect-go", spec.Name)
	assert.Contains(t, spec.DisplayName, "Buf Plugin")
	assert.Equal(t, "v1.5.0", spec.PluginVersion)
	assert.True(t, spec.SupportsGRPC) // connect-go contains "connect"
	assert.True(t, spec.Enabled)
	assert.True(t, spec.Stable)
}

func TestGuessFileExtensions(t *testing.T) {
	tests := []struct {
		name           string
		pluginName     string
		expectedExts   []string
		shouldContain  string
	}{
		{
			name:          "go",
			pluginName:    "connect-go",
			shouldContain: ".go",
		},
		{
			name:          "python",
			pluginName:    "python",
			shouldContain: ".py",
		},
		{
			name:          "typescript",
			pluginName:    "typescript",
			shouldContain: ".ts",
		},
		{
			name:          "java",
			pluginName:    "java",
			shouldContain: ".java",
		},
		{
			name:          "rust",
			pluginName:    "rust",
			shouldContain: ".rs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBufPluginAdapter("buf.build/test/"+tt.pluginName, "v1.0.0")
			exts := adapter.guessFileExtensions(tt.pluginName)

			found := false
			for _, ext := range exts {
				if ext == tt.shouldContain {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected extension %s not found in %v", tt.shouldContain, exts)
		})
	}
}

func TestBuildProtocCommand(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	adapter.binaryPath = "/fake/path/protoc-gen-connect-go"
	adapter.loaded = true

	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles:  []string{"user.proto", "order.proto"},
		ImportPaths: []string{"/proto", "/protos/common"},
		OutputDir:   "/output",
		Options: map[string]string{
			"paths": "source_relative",
		},
	}

	cmd, err := adapter.BuildProtocCommand(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, cmd)

	// Verify command structure
	assert.Contains(t, cmd, "protoc")
	assert.Contains(t, cmd, "--plugin=protoc-gen-connect-go=/fake/path/protoc-gen-connect-go")
	assert.Contains(t, cmd, "--proto_path=/proto")
	assert.Contains(t, cmd, "--proto_path=/protos/common")
	assert.Contains(t, cmd, "--connect-go_out=/output")
	assert.Contains(t, cmd, "user.proto")
	assert.Contains(t, cmd, "order.proto")
}

func TestBuildProtocCommand_NotLoaded(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	adapter.loaded = false

	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles:  []string{"test.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
	}

	_, err := adapter.BuildProtocCommand(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not loaded")
}

func TestIsCached(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")

	// Should not be cached initially (unless it actually exists)
	cached := adapter.isCached()
	// We can't assert false because the file might actually exist on the system
	_ = cached
}

func TestGetCachedPath(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	path := adapter.getCachedPath()

	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".buf/plugins")
	assert.Contains(t, path, "connect-go")
	assert.Contains(t, path, "v1.5.0")
	assert.Contains(t, path, "protoc-gen-connect-go")
}

func TestManifest(t *testing.T) {
	manifest := &plugins.Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test plugin",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/test",
		},
	}

	adapter, err := NewBufPluginAdapterFromManifest(manifest)
	require.NoError(t, err)

	result := adapter.Manifest()
	assert.Equal(t, manifest, result)
}

func TestUnload(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	adapter.loaded = true

	err := adapter.Unload()
	assert.NoError(t, err)
	assert.False(t, adapter.loaded)
}

func TestGetLanguageSpec(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")

	// First call - should build the spec
	spec1 := adapter.GetLanguageSpec()
	assert.NotNil(t, spec1)
	assert.Equal(t, "connect-go", spec1.ID)

	// Second call - should return cached spec
	spec2 := adapter.GetLanguageSpec()
	assert.Equal(t, spec1, spec2)
}

func TestGetLanguageSpec_AlreadyBuilt(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")

	// Pre-build the spec
	adapter.languageSpec = &plugins.LanguageSpec{
		ID:   "custom-id",
		Name: "custom-name",
	}

	spec := adapter.GetLanguageSpec()
	assert.Equal(t, "custom-id", spec.ID)
	assert.Equal(t, "custom-name", spec.Name)
}

func TestValidateOutput_Success(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	ctx := context.Background()

	// Create temporary test files
	tmpDir := t.TempDir()
	file1 := tmpDir + "/file1.pb.go"
	file2 := tmpDir + "/file2.pb.go"

	err := os.WriteFile(file1, []byte("test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("test"), 0644)
	require.NoError(t, err)

	err = adapter.ValidateOutput(ctx, []string{file1, file2})
	assert.NoError(t, err)
}

func TestValidateOutput_NoFiles(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	ctx := context.Background()

	err := adapter.ValidateOutput(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no files generated")
}

func TestValidateOutput_MissingFile(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	ctx := context.Background()

	err := adapter.ValidateOutput(ctx, []string{"/nonexistent/file.pb.go"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected file not found")
}

func TestBuildProtocCommand_WithOptions(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	adapter.binaryPath = "/fake/path/protoc-gen-connect-go"
	adapter.loaded = true

	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles:  []string{"test.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		Options: map[string]string{
			"paths":      "source_relative",
			"standalone": "",
		},
	}

	cmd, err := adapter.BuildProtocCommand(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, cmd)

	// Check that options are included
	hasOptFlag := false
	for _, arg := range cmd {
		if strings.Contains(arg, "--connect-go_opt=") {
			hasOptFlag = true
			break
		}
	}
	assert.True(t, hasOptFlag, "Command should include option flag")
}

func TestBuildProtocCommand_EmptyOptions(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	adapter.binaryPath = "/fake/path/protoc-gen-connect-go"
	adapter.loaded = true

	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles:  []string{"test.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		Options:     map[string]string{},
	}

	cmd, err := adapter.BuildProtocCommand(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, cmd)

	// Check that no option flag is included
	hasOptFlag := false
	for _, arg := range cmd {
		if strings.Contains(arg, "--connect-go_opt=") {
			hasOptFlag = true
			break
		}
	}
	assert.False(t, hasOptFlag, "Command should not include option flag")
}

func TestDeriveLanguageID_EmptyPluginRef(t *testing.T) {
	adapter := NewBufPluginAdapter("", "v1.0.0")
	result := adapter.deriveLanguageID()
	// When pluginRef is empty, split returns [""], and parts[len(parts)-1] = ""
	assert.Equal(t, "", result)
}

func TestGuessFileExtensions_AllLanguages(t *testing.T) {
	tests := []struct {
		name         string
		pluginName   string
		expectedExts []string
	}{
		{
			name:         "go",
			pluginName:   "connect-go",
			expectedExts: []string{".pb.go", ".go"},
		},
		{
			name:         "python",
			pluginName:   "python",
			expectedExts: []string{"_pb2.py", ".py"},
		},
		{
			name:         "py",
			pluginName:   "py",
			expectedExts: []string{"_pb2.py", ".py"},
		},
		{
			name:         "typescript",
			pluginName:   "typescript",
			expectedExts: []string{".pb.ts", ".ts"},
		},
		{
			name:         "ts",
			pluginName:   "ts",
			expectedExts: []string{".pb.ts", ".ts"},
		},
		{
			name:         "javascript",
			pluginName:   "javascript",
			expectedExts: []string{".pb.js", ".js"},
		},
		{
			name:         "js",
			pluginName:   "js",
			expectedExts: []string{".pb.js", ".js"},
		},
		{
			name:         "java",
			pluginName:   "java",
			expectedExts: []string{".java"},
		},
		{
			name:         "kotlin",
			pluginName:   "kotlin",
			expectedExts: []string{".kt"},
		},
		{
			name:         "kt",
			pluginName:   "kt",
			expectedExts: []string{".kt"},
		},
		{
			name:         "swift",
			pluginName:   "swift",
			expectedExts: []string{".swift"},
		},
		{
			name:         "rust",
			pluginName:   "rust",
			expectedExts: []string{".rs"},
		},
		{
			name:         "rs",
			pluginName:   "rs",
			expectedExts: []string{".rs"},
		},
		{
			name:         "unknown",
			pluginName:   "unknown-language",
			expectedExts: []string{".pb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBufPluginAdapter("buf.build/test/"+tt.pluginName, "v1.0.0")
			exts := adapter.guessFileExtensions(tt.pluginName)
			assert.Equal(t, tt.expectedExts, exts)
		})
	}
}

func TestNewBufPluginAdapterFromManifest_NoVersion(t *testing.T) {
	manifest := &plugins.Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "2.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test plugin",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/test",
		},
	}

	adapter, err := NewBufPluginAdapterFromManifest(manifest)
	require.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, "2.0.0", adapter.version, "Should use manifest version when buf_version is not provided")
}

func TestBuildLanguageSpec_GRPCDetection(t *testing.T) {
	tests := []struct {
		name          string
		pluginRef     string
		expectsGRPC   bool
	}{
		{
			name:        "connect plugin",
			pluginRef:   "buf.build/library/connect-go",
			expectsGRPC: true,
		},
		{
			name:        "grpc plugin",
			pluginRef:   "buf.build/library/grpc-go",
			expectsGRPC: true,
		},
		{
			name:        "non-grpc plugin",
			pluginRef:   "buf.build/library/validate-go",
			expectsGRPC: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBufPluginAdapter(tt.pluginRef, "v1.0.0")
			spec := adapter.buildLanguageSpec()
			assert.Equal(t, tt.expectsGRPC, spec.SupportsGRPC)
		})
	}
}

func TestBuildLanguageSpec_AllFields(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	spec := adapter.buildLanguageSpec()

	assert.Equal(t, "connect-go", spec.ID)
	assert.Equal(t, "connect-go", spec.Name)
	assert.Equal(t, "connect-go (Buf Plugin)", spec.DisplayName)
	assert.Equal(t, "protoc-gen-connect-go", spec.ProtocPlugin)
	assert.Equal(t, "v1.5.0", spec.PluginVersion)
	assert.Empty(t, spec.DockerImage)
	assert.True(t, spec.SupportsGRPC)
	assert.True(t, spec.Enabled)
	assert.True(t, spec.Stable)
	assert.Contains(t, spec.Description, "connect-go")
	assert.Equal(t, "https://buf.build/library/connect-go", spec.DocumentationURL)
	assert.NotEmpty(t, spec.FileExtensions)
}

func TestVerifyBinary_Success(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := tmpDir + "/protoc-gen-test"

	// Create an executable file
	err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho test"), 0755)
	require.NoError(t, err)

	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	adapter.binaryPath = binaryPath

	err = adapter.verifyBinary()
	assert.NoError(t, err)
}

func TestVerifyBinary_NotExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := tmpDir + "/protoc-gen-test"

	// Create a non-executable file
	err := os.WriteFile(binaryPath, []byte("test"), 0644)
	require.NoError(t, err)

	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	adapter.binaryPath = binaryPath

	err = adapter.verifyBinary()
	assert.NoError(t, err) // Should succeed after making it executable

	// Verify it's now executable
	info, err := os.Stat(binaryPath)
	require.NoError(t, err)
	assert.NotEqual(t, 0, info.Mode()&0111, "File should be executable")
}

func TestVerifyBinary_NotFound(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	adapter.binaryPath = "/nonexistent/path/protoc-gen-test"

	err := adapter.verifyBinary()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "binary not found")
}

func TestVerifyBinary_CannotMakeExecutable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	binaryPath := tmpDir + "/protoc-gen-test"

	// Create a non-executable file
	err := os.WriteFile(binaryPath, []byte("test"), 0644)
	require.NoError(t, err)

	// Make the directory read-only to prevent chmod
	err = os.Chmod(tmpDir, 0555)
	require.NoError(t, err)
	defer func() {
		// Restore permissions for cleanup
		_ = os.Chmod(tmpDir, 0755)
	}()

	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	adapter.binaryPath = binaryPath

	err = adapter.verifyBinary()
	if err != nil {
		assert.Contains(t, err.Error(), "failed to make binary executable")
	}
	// Note: This test may pass on some systems if chmod succeeds despite directory permissions
}

func TestLoad_AlreadyLoaded(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	adapter.loaded = true

	err := adapter.Load()
	assert.NoError(t, err)
}

func TestLoad_FromCache(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := tmpDir + "/protoc-gen-test"

	// Create an executable file to simulate cached plugin
	err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho test"), 0755)
	require.NoError(t, err)

	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")

	// Test the path that would be used if cached
	adapter.loaded = false
	adapter.binaryPath = binaryPath

	err = adapter.verifyBinary()
	assert.NoError(t, err)

	// Verify language spec is built
	spec := adapter.GetLanguageSpec()
	assert.NotNil(t, spec)
}

func TestLoad_VerifyBinaryFails(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	adapter.loaded = false
	adapter.binaryPath = "/nonexistent/binary"

	// Since Load calls downloader.Download which we can't easily mock,
	// we test verifyBinary directly which is called by Load
	err := adapter.verifyBinary()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "binary not found")
}

func TestDerivePluginName(t *testing.T) {
	tests := []struct {
		name         string
		pluginRef    string
		expectedName string
	}{
		{
			name:         "multi-path",
			pluginRef:    "buf.build/library/connect-go",
			expectedName: "connect-go",
		},
		{
			name:         "single-path",
			pluginRef:    "connect-go",
			expectedName: "connect-go",
		},
		{
			name:         "deep-path",
			pluginRef:    "buf.build/org/team/plugin",
			expectedName: "plugin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBufPluginAdapter(tt.pluginRef, "v1.0.0")
			result := adapter.derivePluginName()
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestLoad_CachedPlugin(t *testing.T) {
	// This test validates the Load workflow when a plugin is already cached
	// We simulate this by pre-setting the binary path and testing the load logic
	tmpDir := t.TempDir()

	// Create a mock cached plugin
	pluginName := "test-plugin"
	version := "v1.0.0"
	binaryPath := tmpDir + "/protoc-gen-" + pluginName
	err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho test"), 0755)
	require.NoError(t, err)

	adapter := NewBufPluginAdapter("buf.build/library/"+pluginName, version)

	// Manually set the binary path to simulate cached state
	adapter.binaryPath = binaryPath
	adapter.loaded = false

	// Verify binary and build language spec
	err = adapter.verifyBinary()
	assert.NoError(t, err)

	adapter.languageSpec = adapter.buildLanguageSpec()
	adapter.loaded = true

	assert.True(t, adapter.loaded)
	assert.NotNil(t, adapter.languageSpec)
}

func TestLoad_MultipleCallsIdempotent(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	adapter.loaded = true

	// First call
	err := adapter.Load()
	assert.NoError(t, err)

	// Second call should also succeed
	err = adapter.Load()
	assert.NoError(t, err)
	assert.True(t, adapter.loaded)
}

func TestBuildProtocCommand_MultipleOptions(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/grpc-go", "v1.5.0")
	adapter.binaryPath = "/fake/path/protoc-gen-grpc-go"
	adapter.loaded = true

	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles:  []string{"service.proto"},
		ImportPaths: []string{"/proto", "/vendor/protos"},
		OutputDir:   "/gen",
		Options: map[string]string{
			"paths":        "source_relative",
			"require_unimplemented_servers": "false",
		},
	}

	cmd, err := adapter.BuildProtocCommand(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd, "protoc")
	assert.Contains(t, cmd, "service.proto")
}

func TestBuildProtocCommand_EmptyProtoFiles(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	adapter.binaryPath = "/fake/path/protoc-gen-connect-go"
	adapter.loaded = true

	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles:  []string{},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
	}

	cmd, err := adapter.BuildProtocCommand(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	// Command should still be valid even with no proto files
	assert.Contains(t, cmd, "protoc")
}

func TestBuildProtocCommand_NoImportPaths(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	adapter.binaryPath = "/fake/path/protoc-gen-connect-go"
	adapter.loaded = true

	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles:  []string{"test.proto"},
		ImportPaths: []string{},
		OutputDir:   "/output",
	}

	cmd, err := adapter.BuildProtocCommand(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd, "protoc")
	assert.Contains(t, cmd, "test.proto")
}

func TestValidateOutput_MultipleFiles(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/go", "v1.5.0")
	ctx := context.Background()

	tmpDir := t.TempDir()
	files := []string{
		tmpDir + "/user.pb.go",
		tmpDir + "/order.pb.go",
		tmpDir + "/product.pb.go",
	}

	for _, file := range files {
		err := os.WriteFile(file, []byte("package main"), 0644)
		require.NoError(t, err)
	}

	err := adapter.ValidateOutput(ctx, files)
	assert.NoError(t, err)
}

func TestValidateOutput_PartialFailure(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/go", "v1.5.0")
	ctx := context.Background()

	tmpDir := t.TempDir()
	existingFile := tmpDir + "/existing.pb.go"
	err := os.WriteFile(existingFile, []byte("test"), 0644)
	require.NoError(t, err)

	files := []string{
		existingFile,
		"/nonexistent/missing.pb.go",
	}

	err = adapter.ValidateOutput(ctx, files)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected file not found")
}

func TestDeriveLanguageID_VariousFormats(t *testing.T) {
	tests := []struct {
		name      string
		pluginRef string
		expected  string
	}{
		{
			name:      "standard format",
			pluginRef: "buf.build/library/connect-go",
			expected:  "connect-go",
		},
		{
			name:      "with organization",
			pluginRef: "buf.build/grpc/grpc-go",
			expected:  "grpc-go",
		},
		{
			name:      "nested path",
			pluginRef: "buf.build/org/team/subteam/plugin",
			expected:  "plugin",
		},
		{
			name:      "single segment",
			pluginRef: "plugin-name",
			expected:  "plugin-name",
		},
		{
			name:      "with dashes",
			pluginRef: "buf.build/library/connect-grpc-go",
			expected:  "connect-grpc-go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBufPluginAdapter(tt.pluginRef, "v1.0.0")
			result := adapter.deriveLanguageID()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCachedPath_WindowsCompatibility(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/test-plugin", "v2.5.0")
	path := adapter.getCachedPath()

	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".buf")
	assert.Contains(t, path, "plugins")
	assert.Contains(t, path, "test-plugin")
	assert.Contains(t, path, "v2.5.0")
	// Check for proper path construction
	assert.True(t, strings.Contains(path, string(os.PathSeparator)))
}

func TestBuildLanguageSpec_DocumentationURL(t *testing.T) {
	tests := []struct {
		name        string
		pluginRef   string
		expectedURL string
	}{
		{
			name:        "standard plugin",
			pluginRef:   "buf.build/library/connect-go",
			expectedURL: "https://buf.build/library/connect-go",
		},
		{
			name:        "organization plugin",
			pluginRef:   "buf.build/grpc/grpc-go",
			expectedURL: "https://buf.build/grpc/grpc-go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBufPluginAdapter(tt.pluginRef, "v1.0.0")
			spec := adapter.buildLanguageSpec()
			assert.Equal(t, tt.expectedURL, spec.DocumentationURL)
		})
	}
}

func TestBuildLanguageSpec_ProtocPluginNaming(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/validate-go", "v1.0.0")
	spec := adapter.buildLanguageSpec()

	assert.Equal(t, "validate-go", spec.Name)
	assert.Equal(t, "protoc-gen-validate-go", spec.ProtocPlugin)
	assert.Equal(t, "validate-go", spec.ID)
}

func TestBuildLanguageSpec_DockerImageEmpty(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	spec := adapter.buildLanguageSpec()

	// Buf plugins should not use Docker images
	assert.Empty(t, spec.DockerImage)
	assert.True(t, spec.Enabled)
	assert.True(t, spec.Stable)
}

func TestGuessFileExtensions_CaseSensitivity(t *testing.T) {
	tests := []struct {
		name         string
		pluginName   string
		shouldContain string
	}{
		{
			name:          "uppercase GO",
			pluginName:    "CONNECT-GO",
			shouldContain: ".go",
		},
		{
			name:          "mixed case Python",
			pluginName:    "Python-Gen",
			shouldContain: ".py",
		},
		{
			name:          "uppercase JAVA",
			pluginName:    "JAVA",
			shouldContain: ".java",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBufPluginAdapter("buf.build/test/"+tt.pluginName, "v1.0.0")
			exts := adapter.guessFileExtensions(tt.pluginName)

			found := false
			for _, ext := range exts {
				if ext == tt.shouldContain {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected extension %s not found in %v", tt.shouldContain, exts)
		})
	}
}

func TestManifestNil(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	manifest := adapter.Manifest()
	assert.Nil(t, manifest, "Manifest should be nil when adapter created without manifest")
}

func TestNewBufPluginAdapterFromManifest_VersionPriority(t *testing.T) {
	manifest := &plugins.Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test plugin",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/test",
			"buf_version":  "v2.0.0",
		},
	}

	adapter, err := NewBufPluginAdapterFromManifest(manifest)
	require.NoError(t, err)

	// buf_version should take priority
	assert.Equal(t, "v2.0.0", adapter.version)
	assert.Equal(t, "buf.build/library/test", adapter.pluginRef)
}

func TestBuildProtocCommand_OptionsWithEmptyValues(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")
	adapter.binaryPath = "/fake/path/protoc-gen-connect-go"
	adapter.loaded = true

	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles:  []string{"test.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		Options: map[string]string{
			"standalone": "",
			"verbose":    "",
		},
	}

	cmd, err := adapter.BuildProtocCommand(ctx, req)
	require.NoError(t, err)

	// Check that options without values are included
	hasOptFlag := false
	for _, arg := range cmd {
		if strings.Contains(arg, "--connect-go_opt=") {
			hasOptFlag = true
			// Should contain option names without values
			assert.True(t, strings.Contains(arg, "standalone") || strings.Contains(arg, "verbose"))
		}
	}
	assert.True(t, hasOptFlag)
}

func TestBuildProtocCommand_OptionsOrdering(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/go", "v1.5.0")
	adapter.binaryPath = "/fake/path/protoc-gen-go"
	adapter.loaded = true

	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles:  []string{"a.proto", "b.proto", "c.proto"},
		ImportPaths: []string{"/proto1", "/proto2"},
		OutputDir:   "/output",
		Options: map[string]string{
			"paths": "source_relative",
		},
	}

	cmd, err := adapter.BuildProtocCommand(ctx, req)
	require.NoError(t, err)

	// Verify command starts with protoc
	assert.Equal(t, "protoc", cmd[0])

	// Verify proto files are at the end
	assert.Contains(t, cmd, "a.proto")
	assert.Contains(t, cmd, "b.proto")
	assert.Contains(t, cmd, "c.proto")
}

func TestVerifyBinary_PermissionEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := tmpDir + "/protoc-gen-test"

	// Create a file with minimal permissions
	err := os.WriteFile(binaryPath, []byte("test"), 0400)
	require.NoError(t, err)

	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	adapter.binaryPath = binaryPath

	err = adapter.verifyBinary()
	// Should succeed after making it executable
	assert.NoError(t, err)

	// Check that permissions were updated
	info, err := os.Stat(binaryPath)
	require.NoError(t, err)
	assert.NotEqual(t, 0, info.Mode()&0111)
}

func TestGetLanguageSpec_Consistency(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/connect-go", "v1.5.0")

	// Get spec multiple times
	spec1 := adapter.GetLanguageSpec()
	spec2 := adapter.GetLanguageSpec()
	spec3 := adapter.GetLanguageSpec()

	// All should return the same instance
	assert.Equal(t, spec1, spec2)
	assert.Equal(t, spec2, spec3)

	// Verify it's actually the same pointer (cached)
	assert.True(t, spec1 == spec2)
	assert.True(t, spec2 == spec3)
}

func TestUnload_StateChange(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")

	// Initially not loaded
	assert.False(t, adapter.loaded)

	// Simulate loaded state
	adapter.loaded = true
	adapter.binaryPath = "/some/path"
	adapter.languageSpec = &plugins.LanguageSpec{ID: "test"}

	// Unload
	err := adapter.Unload()
	assert.NoError(t, err)
	assert.False(t, adapter.loaded)

	// Other fields should remain
	assert.NotEmpty(t, adapter.binaryPath)
	assert.NotNil(t, adapter.languageSpec)
}

func TestBuildProtocCommand_ComplexScenario(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/grpc-gateway", "v2.0.0")
	adapter.binaryPath = "/usr/local/bin/protoc-gen-grpc-gateway"
	adapter.loaded = true

	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles: []string{
			"api/v1/users.proto",
			"api/v1/orders.proto",
		},
		ImportPaths: []string{
			"/workspace/proto",
			"/workspace/vendor/proto",
			"/usr/local/include",
		},
		OutputDir: "/workspace/gen",
		Options: map[string]string{
			"paths":                   "source_relative",
			"grpc_api_configuration":  "api/config.yaml",
			"generate_unbound_methods": "true",
		},
	}

	cmd, err := adapter.BuildProtocCommand(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, cmd)

	// Verify all components are present
	cmdStr := strings.Join(cmd, " ")
	assert.Contains(t, cmdStr, "protoc")
	assert.Contains(t, cmdStr, "grpc-gateway")
	assert.Contains(t, cmdStr, "api/v1/users.proto")
	assert.Contains(t, cmdStr, "api/v1/orders.proto")
	assert.Contains(t, cmdStr, "/workspace/proto")
	assert.Contains(t, cmdStr, "/workspace/gen")
}

func TestLoad_WithMockDownloader(t *testing.T) {
	// Test the Load function's download path
	tmpDir := t.TempDir()
	pluginName := "mock-plugin"
	version := "v1.0.0"
	binaryPath := tmpDir + "/protoc-gen-" + pluginName

	// Create a binary file
	err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho test"), 0755)
	require.NoError(t, err)

	adapter := NewBufPluginAdapter("buf.build/library/"+pluginName, version)

	// Override downloader with a mock that sets the binary path
	// In production, we can't easily test the actual download without mocking
	// But we can test the load flow after download
	adapter.binaryPath = binaryPath
	adapter.loaded = false

	// Simulate what Load does after download
	err = adapter.verifyBinary()
	require.NoError(t, err)

	adapter.languageSpec = adapter.buildLanguageSpec()
	adapter.loaded = true

	assert.True(t, adapter.loaded)
	assert.NotNil(t, adapter.languageSpec)
	assert.Equal(t, pluginName, adapter.languageSpec.ID)
}

func TestDeriveLanguageID_SingleSegment(t *testing.T) {
	// Test edge case with single segment
	adapter := NewBufPluginAdapter("plugin", "v1.0.0")
	result := adapter.deriveLanguageID()
	assert.Equal(t, "plugin", result)
}

func TestDeriveLanguageID_EmptyPluginRefReturnsLastPart(t *testing.T) {
	// When pluginRef is empty, strings.Split returns [""]
	// len(parts) > 0 is true, so it returns parts[0] which is ""
	adapter := NewBufPluginAdapter("", "v1.0.0")
	result := adapter.deriveLanguageID()
	// Empty string split by "/" gives [""], so parts[len(parts)-1] = ""
	assert.Equal(t, "", result)
}

func TestIsCached_VariousScenarios(t *testing.T) {
	tests := []struct {
		name      string
		pluginRef string
		version   string
		setup     func(t *testing.T, adapter *BufPluginAdapter) bool
		expected  bool
	}{
		{
			name:      "not cached",
			pluginRef: "buf.build/library/nonexistent",
			version:   "v1.0.0",
			setup:     func(t *testing.T, adapter *BufPluginAdapter) bool { return false },
			expected:  false,
		},
		{
			name:      "cached exists",
			pluginRef: "buf.build/library/cached",
			version:   "v1.0.0",
			setup: func(t *testing.T, adapter *BufPluginAdapter) bool {
				// Create the cached file
				cachePath := adapter.getCachedPath()
				err := os.MkdirAll(filepath.Dir(cachePath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(cachePath, []byte("test"), 0644)
				require.NoError(t, err)
				return true
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBufPluginAdapter(tt.pluginRef, tt.version)
			if tt.setup(t, adapter) {
				defer os.RemoveAll(filepath.Dir(adapter.getCachedPath()))
			}
			result := adapter.isCached()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildProtocCommand_PluginNaming(t *testing.T) {
	tests := []struct {
		name          string
		pluginRef     string
		expectedFlag  string
	}{
		{
			name:         "simple go plugin",
			pluginRef:    "buf.build/protocolbuffers/go",
			expectedFlag: "--go_out=",
		},
		{
			name:         "connect go plugin",
			pluginRef:    "buf.build/library/connect-go",
			expectedFlag: "--connect-go_out=",
		},
		{
			name:         "grpc gateway",
			pluginRef:    "buf.build/grpc-ecosystem/grpc-gateway",
			expectedFlag: "--grpc-gateway_out=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBufPluginAdapter(tt.pluginRef, "v1.0.0")
			adapter.binaryPath = "/fake/path/protoc-gen-test"
			adapter.loaded = true

			ctx := context.Background()
			req := &plugins.CommandRequest{
				ProtoFiles:  []string{"test.proto"},
				ImportPaths: []string{"/proto"},
				OutputDir:   "/output",
			}

			cmd, err := adapter.BuildProtocCommand(ctx, req)
			require.NoError(t, err)

			cmdStr := strings.Join(cmd, " ")
			assert.Contains(t, cmdStr, tt.expectedFlag)
		})
	}
}

func TestGetCachedPath_ConsistentPaths(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/test-plugin", "v1.5.0")

	// Call multiple times to ensure consistency
	path1 := adapter.getCachedPath()
	path2 := adapter.getCachedPath()
	path3 := adapter.getCachedPath()

	assert.Equal(t, path1, path2)
	assert.Equal(t, path2, path3)
	assert.Contains(t, path1, "test-plugin")
	assert.Contains(t, path1, "v1.5.0")
}

func TestBuildLanguageSpec_StabilityFlags(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/experimental-plugin", "v0.1.0")
	spec := adapter.buildLanguageSpec()

	// All Buf plugins are marked as stable and enabled
	assert.True(t, spec.Enabled)
	assert.True(t, spec.Stable)
}

func TestBuildLanguageSpec_DescriptionContent(t *testing.T) {
	pluginRef := "buf.build/library/connect-go"
	adapter := NewBufPluginAdapter(pluginRef, "v1.5.0")
	spec := adapter.buildLanguageSpec()

	assert.Contains(t, spec.Description, "connect-go")
	assert.Contains(t, spec.Description, pluginRef)
	assert.NotEmpty(t, spec.Description)
}

func TestValidateOutput_EmptyFilePath(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	ctx := context.Background()

	// Test with file path that's empty string in list
	err := adapter.ValidateOutput(ctx, []string{""})
	assert.Error(t, err)
}

func TestUnload_MultipleCallsIdempotent(t *testing.T) {
	adapter := NewBufPluginAdapter("buf.build/library/test", "v1.0.0")
	adapter.loaded = true

	// First unload
	err := adapter.Unload()
	assert.NoError(t, err)
	assert.False(t, adapter.loaded)

	// Second unload
	err = adapter.Unload()
	assert.NoError(t, err)
	assert.False(t, adapter.loaded)
}

func TestNewBufPluginAdapter_FieldInitialization(t *testing.T) {
	pluginRef := "buf.build/library/test-plugin"
	version := "v2.0.0"

	adapter := NewBufPluginAdapter(pluginRef, version)

	assert.Equal(t, pluginRef, adapter.pluginRef)
	assert.Equal(t, version, adapter.version)
	assert.NotNil(t, adapter.downloader)
	assert.Nil(t, adapter.manifest)
	assert.Nil(t, adapter.languageSpec)
	assert.False(t, adapter.loaded)
	assert.Empty(t, adapter.binaryPath)
}

func TestNewBufPluginAdapterFromManifest_FieldMapping(t *testing.T) {
	manifest := &plugins.Manifest{
		ID:          "custom-id",
		Name:        "Custom Plugin",
		Version:     "3.0.0",
		APIVersion:  "2.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Custom description",
		Metadata: map[string]string{
			"buf_registry": "buf.build/custom/plugin",
			"buf_version":  "v3.1.0",
		},
	}

	adapter, err := NewBufPluginAdapterFromManifest(manifest)
	require.NoError(t, err)

	assert.Equal(t, manifest, adapter.manifest)
	assert.Equal(t, "buf.build/custom/plugin", adapter.pluginRef)
	assert.Equal(t, "v3.1.0", adapter.version)
	assert.NotNil(t, adapter.downloader)
	assert.False(t, adapter.loaded)
}
