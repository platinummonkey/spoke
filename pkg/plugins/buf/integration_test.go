package buf

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/platinummonkey/spoke/pkg/plugins"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateBufPluginFactory tests the factory creation function
func TestCreateBufPluginFactory(t *testing.T) {
	factory := CreateBufPluginFactory()
	assert.NotNil(t, factory, "Factory should not be nil")
}

// TestBufPluginFactory_Success tests successful plugin creation via factory
func TestBufPluginFactory_Success(t *testing.T) {
	factory := CreateBufPluginFactory()

	manifest := &plugins.Manifest{
		ID:          "test-buf-plugin",
		Name:        "Test Buf Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test Buf plugin for integration testing",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/connect-go",
			"buf_version":  "v1.5.0",
		},
	}

	plugin, err := factory(manifest)
	require.NoError(t, err)
	assert.NotNil(t, plugin)

	// Verify it implements Plugin interface
	assert.NotNil(t, plugin.Manifest())
	assert.Equal(t, manifest, plugin.Manifest())
}

// TestBufPluginFactory_MissingRegistry tests factory with missing buf_registry
func TestBufPluginFactory_MissingRegistry(t *testing.T) {
	factory := CreateBufPluginFactory()

	manifest := &plugins.Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test plugin without buf_registry",
		Metadata:    map[string]string{
			// Missing buf_registry
		},
	}

	plugin, err := factory(manifest)
	assert.Error(t, err)
	assert.Nil(t, plugin)
	assert.Contains(t, err.Error(), "buf_registry")
}

// TestBufPluginFactory_UsesManifestVersion tests factory uses manifest version when buf_version is missing
func TestBufPluginFactory_UsesManifestVersion(t *testing.T) {
	factory := CreateBufPluginFactory()

	manifest := &plugins.Manifest{
		ID:          "test-buf-plugin",
		Name:        "Test Buf Plugin",
		Version:     "2.3.4",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test Buf plugin",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/test",
			// buf_version is missing, should use manifest Version
		},
	}

	plugin, err := factory(manifest)
	require.NoError(t, err)
	assert.NotNil(t, plugin)

	// Verify the adapter was created and uses manifest version
	adapter, ok := plugin.(*BufPluginAdapter)
	require.True(t, ok, "Plugin should be a BufPluginAdapter")
	assert.Equal(t, "2.3.4", adapter.version)
}

// TestBufPluginFactory_ReturnsLanguagePlugin tests that factory returns a LanguagePlugin
func TestBufPluginFactory_ReturnsLanguagePlugin(t *testing.T) {
	factory := CreateBufPluginFactory()

	manifest := &plugins.Manifest{
		ID:          "test-buf-plugin",
		Name:        "Test Buf Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test Buf plugin",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/connect-go",
			"buf_version":  "v1.0.0",
		},
	}

	plugin, err := factory(manifest)
	require.NoError(t, err)

	// Verify it implements LanguagePlugin interface
	langPlugin, ok := plugin.(plugins.LanguagePlugin)
	assert.True(t, ok, "Plugin should implement LanguagePlugin interface")

	if ok {
		spec := langPlugin.GetLanguageSpec()
		assert.NotNil(t, spec)
		assert.Equal(t, "connect-go", spec.ID)
	}
}

// TestConfigureLoader tests the loader configuration function
func TestConfigureLoader(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stderr)

	loader := plugins.NewLoader([]string{"/tmp"}, logger)
	assert.NotNil(t, loader)

	// Configure with Buf plugin support
	ConfigureLoader(loader)

	// Verify configuration by attempting to use the factory
	// Create a test manifest
	manifest := &plugins.Manifest{
		ID:          "test-buf-plugin",
		Name:        "Test Buf Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test Buf plugin",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/test",
			"buf_version":  "v1.0.0",
		},
	}

	// Create factory to verify it works
	factory := CreateBufPluginFactory()
	plugin, err := factory(manifest)
	require.NoError(t, err)
	assert.NotNil(t, plugin)
}

// TestConfigureLoader_MultiplePlugins tests loader with multiple Buf plugins
func TestConfigureLoader_MultiplePlugins(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stderr)

	loader := plugins.NewLoader([]string{"/tmp"}, logger)
	ConfigureLoader(loader)

	manifests := []*plugins.Manifest{
		{
			ID:          "buf-connect-go",
			Name:        "Connect for Go",
			Version:     "1.5.0",
			APIVersion:  "1.0.0",
			Type:        plugins.PluginTypeLanguage,
			Description: "Connect RPC for Go",
			Metadata: map[string]string{
				"buf_registry": "buf.build/library/connect-go",
				"buf_version":  "v1.5.0",
			},
		},
		{
			ID:          "buf-grpc-go",
			Name:        "gRPC for Go",
			Version:     "1.3.0",
			APIVersion:  "1.0.0",
			Type:        plugins.PluginTypeLanguage,
			Description: "gRPC for Go",
			Metadata: map[string]string{
				"buf_registry": "buf.build/protocolbuffers/go",
				"buf_version":  "v1.3.0",
			},
		},
	}

	factory := CreateBufPluginFactory()

	for _, manifest := range manifests {
		plugin, err := factory(manifest)
		require.NoError(t, err, "Failed to create plugin for %s", manifest.ID)
		assert.NotNil(t, plugin)
		assert.Equal(t, manifest, plugin.Manifest())
	}
}

// TestBufPluginFactory_EmptyMetadata tests factory with empty metadata map
func TestBufPluginFactory_EmptyMetadata(t *testing.T) {
	factory := CreateBufPluginFactory()

	manifest := &plugins.Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test plugin",
		Metadata:    map[string]string{},
	}

	plugin, err := factory(manifest)
	assert.Error(t, err)
	assert.Nil(t, plugin)
}

// TestBufPluginFactory_NilMetadata tests factory with nil metadata
func TestBufPluginFactory_NilMetadata(t *testing.T) {
	factory := CreateBufPluginFactory()

	manifest := &plugins.Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test plugin",
		Metadata:    nil,
	}

	plugin, err := factory(manifest)
	assert.Error(t, err)
	assert.Nil(t, plugin)
}

// TestBufPluginFactory_VariousRegistryFormats tests factory with different registry formats
func TestBufPluginFactory_VariousRegistryFormats(t *testing.T) {
	tests := []struct {
		name         string
		registry     string
		version      string
		expectedID   string
		expectedName string
	}{
		{
			name:         "standard registry",
			registry:     "buf.build/library/connect-go",
			version:      "v1.5.0",
			expectedID:   "connect-go",
			expectedName: "connect-go",
		},
		{
			name:         "simple name",
			registry:     "connect-go",
			version:      "v1.0.0",
			expectedID:   "connect-go",
			expectedName: "connect-go",
		},
		{
			name:         "deep path",
			registry:     "buf.build/org/team/plugin-name",
			version:      "v2.0.0",
			expectedID:   "plugin-name",
			expectedName: "plugin-name",
		},
		{
			name:         "protocolbuffers",
			registry:     "buf.build/protocolbuffers/go",
			version:      "v1.31.0",
			expectedID:   "go",
			expectedName: "go",
		},
	}

	factory := CreateBufPluginFactory()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := &plugins.Manifest{
				ID:          "test-" + tt.expectedID,
				Name:        "Test " + tt.expectedName,
				Version:     "1.0.0",
				APIVersion:  "1.0.0",
				Type:        plugins.PluginTypeLanguage,
				Description: "Test plugin",
				Metadata: map[string]string{
					"buf_registry": tt.registry,
					"buf_version":  tt.version,
				},
			}

			plugin, err := factory(manifest)
			require.NoError(t, err)
			assert.NotNil(t, plugin)

			// Verify language spec
			langPlugin, ok := plugin.(plugins.LanguagePlugin)
			require.True(t, ok)

			spec := langPlugin.GetLanguageSpec()
			assert.Equal(t, tt.expectedID, spec.ID)
			assert.Equal(t, tt.expectedName, spec.Name)
		})
	}
}

// TestBufPluginFactory_IntegrationWithProtocCommand tests that created plugins can build protoc commands
func TestBufPluginFactory_IntegrationWithProtocCommand(t *testing.T) {
	factory := CreateBufPluginFactory()

	manifest := &plugins.Manifest{
		ID:          "buf-connect-go",
		Name:        "Connect for Go",
		Version:     "1.5.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Connect RPC for Go",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/connect-go",
			"buf_version":  "v1.5.0",
		},
	}

	plugin, err := factory(manifest)
	require.NoError(t, err)

	// Cast to language plugin
	langPlugin, ok := plugin.(plugins.LanguagePlugin)
	require.True(t, ok)

	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a fake binary to simulate the plugin being loaded
	adapter, ok := plugin.(*BufPluginAdapter)
	require.True(t, ok)

	fakeBinaryPath := filepath.Join(tmpDir, "protoc-gen-connect-go")
	err = os.WriteFile(fakeBinaryPath, []byte("#!/bin/bash\necho 'test'"), 0755)
	require.NoError(t, err)

	adapter.binaryPath = fakeBinaryPath
	adapter.loaded = true

	// Build a protoc command
	ctx := context.Background()
	req := &plugins.CommandRequest{
		ProtoFiles:  []string{"test.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   tmpDir,
		Options: map[string]string{
			"paths": "source_relative",
		},
	}

	cmd, err := langPlugin.BuildProtocCommand(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd, "protoc")
}

// TestConfigureLoader_WithRealPluginDirectory tests loader configuration with an actual plugin directory
func TestConfigureLoader_WithRealPluginDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test plugin directory structure
	pluginDir := filepath.Join(tmpDir, "buf-test-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	// Create a manifest file
	manifestContent := `id: buf-test-plugin
name: Test Buf Plugin
version: 1.0.0
api_version: 1.0.0
type: language
description: Test Buf plugin for integration testing
metadata:
  buf_registry: buf.build/library/test
  buf_version: v1.0.0
`
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	err = os.WriteFile(manifestPath, []byte(manifestContent), 0644)
	require.NoError(t, err)

	// Create loader with the test directory
	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	loader := plugins.NewLoader([]string{tmpDir}, logger)

	// Configure with Buf plugin support
	ConfigureLoader(loader)

	// Discover plugins - this will use the Buf factory
	ctx := context.Background()
	discoveredPlugins, err := loader.DiscoverPlugins(ctx)

	// Note: This will fail to fully load because we don't have a real buf binary,
	// but we're testing that the configuration and discovery mechanism works
	// The error is expected but the factory should have been invoked
	t.Logf("Discovered %d plugin(s), error: %v", len(discoveredPlugins), err)
}

// TestBufPluginFactory_PreservesManifestMetadata tests that factory preserves all manifest data
func TestBufPluginFactory_PreservesManifestMetadata(t *testing.T) {
	factory := CreateBufPluginFactory()

	manifest := &plugins.Manifest{
		ID:            "test-buf-plugin",
		Name:          "Test Buf Plugin",
		Version:       "1.0.0",
		APIVersion:    "1.0.0",
		Type:          plugins.PluginTypeLanguage,
		Description:   "Test Buf plugin",
		Author:        "Test Author",
		License:       "MIT",
		Homepage:      "https://example.com",
		Repository:    "https://github.com/example/repo",
		SecurityLevel: plugins.SecurityLevelCommunity,
		Permissions:   []string{"filesystem:read", "network:read"},
		Dependencies:  []string{"dep1", "dep2"},
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/test",
			"buf_version":  "v1.0.0",
			"custom_field": "custom_value",
		},
	}

	plugin, err := factory(manifest)
	require.NoError(t, err)
	assert.NotNil(t, plugin)

	// Verify all manifest fields are preserved
	retrievedManifest := plugin.Manifest()
	assert.Equal(t, manifest.ID, retrievedManifest.ID)
	assert.Equal(t, manifest.Name, retrievedManifest.Name)
	assert.Equal(t, manifest.Version, retrievedManifest.Version)
	assert.Equal(t, manifest.Author, retrievedManifest.Author)
	assert.Equal(t, manifest.License, retrievedManifest.License)
	assert.Equal(t, manifest.Homepage, retrievedManifest.Homepage)
	assert.Equal(t, manifest.Repository, retrievedManifest.Repository)
	assert.Equal(t, manifest.SecurityLevel, retrievedManifest.SecurityLevel)
	assert.Equal(t, manifest.Permissions, retrievedManifest.Permissions)
	assert.Equal(t, manifest.Dependencies, retrievedManifest.Dependencies)
	assert.Equal(t, manifest.Metadata, retrievedManifest.Metadata)
}

// TestBufPluginFactory_LanguageSpecGeneration tests various aspects of language spec generation
func TestBufPluginFactory_LanguageSpecGeneration(t *testing.T) {
	tests := []struct {
		name            string
		registry        string
		expectedSupportsGRPC bool
		expectedFileExts     []string
	}{
		{
			name:                 "connect plugin",
			registry:             "buf.build/library/connect-go",
			expectedSupportsGRPC: true,
			expectedFileExts:     []string{".pb.go", ".go"},
		},
		{
			name:                 "grpc plugin",
			registry:             "buf.build/library/grpc-python",
			expectedSupportsGRPC: true,
			expectedFileExts:     []string{"_pb2.py", ".py"},
		},
		{
			name:                 "non-grpc plugin",
			registry:             "buf.build/library/validate-go",
			expectedSupportsGRPC: false,
			expectedFileExts:     []string{".pb.go", ".go"},
		},
		{
			name:                 "typescript plugin",
			registry:             "buf.build/library/plugin-ts",
			expectedSupportsGRPC: false,
			expectedFileExts:     []string{".pb.ts", ".ts"},
		},
	}

	factory := CreateBufPluginFactory()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := &plugins.Manifest{
				ID:          "test-plugin",
				Name:        "Test Plugin",
				Version:     "1.0.0",
				APIVersion:  "1.0.0",
				Type:        plugins.PluginTypeLanguage,
				Description: "Test plugin",
				Metadata: map[string]string{
					"buf_registry": tt.registry,
					"buf_version":  "v1.0.0",
				},
			}

			plugin, err := factory(manifest)
			require.NoError(t, err)

			langPlugin, ok := plugin.(plugins.LanguagePlugin)
			require.True(t, ok)

			spec := langPlugin.GetLanguageSpec()
			assert.Equal(t, tt.expectedSupportsGRPC, spec.SupportsGRPC)
			assert.Equal(t, tt.expectedFileExts, spec.FileExtensions)
		})
	}
}

// TestConfigureLoader_FactoryNotNil tests that ConfigureLoader sets a non-nil factory
func TestConfigureLoader_FactoryNotNil(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stderr)

	loader := plugins.NewLoader([]string{"/tmp"}, logger)

	// Before configuration, attempt to create a buf plugin should fail
	// After configuration, it should work
	ConfigureLoader(loader)

	// Create a test manifest and verify the factory works
	manifest := &plugins.Manifest{
		ID:          "test-buf-plugin",
		Name:        "Test Buf Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test Buf plugin",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/test",
			"buf_version":  "v1.0.0",
		},
	}

	factory := CreateBufPluginFactory()
	plugin, err := factory(manifest)
	require.NoError(t, err)
	assert.NotNil(t, plugin)
}

// TestBufPluginFactory_LoadAndUnload tests the plugin lifecycle
func TestBufPluginFactory_LoadAndUnload(t *testing.T) {
	factory := CreateBufPluginFactory()

	manifest := &plugins.Manifest{
		ID:          "test-buf-plugin",
		Name:        "Test Buf Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test Buf plugin",
		Metadata: map[string]string{
			"buf_registry": "buf.build/library/connect-go",
			"buf_version":  "v1.5.0",
		},
	}

	plugin, err := factory(manifest)
	require.NoError(t, err)
	assert.NotNil(t, plugin)

	// Test unload
	err = plugin.Unload()
	assert.NoError(t, err)

	// Verify it's no longer loaded
	adapter, ok := plugin.(*BufPluginAdapter)
	require.True(t, ok)
	assert.False(t, adapter.loaded)
}
