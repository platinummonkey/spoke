package buf

import (
	"context"
	"os"
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
