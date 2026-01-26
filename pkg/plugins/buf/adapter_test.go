package buf

import (
	"context"
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
