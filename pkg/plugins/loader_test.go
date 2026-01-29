package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	dirs := []string{"/tmp/plugins"}
	loader := NewLoader(dirs, nil)

	assert.NotNil(t, loader)
	assert.Equal(t, dirs, loader.pluginDirs)
	assert.NotNil(t, loader.log)
	assert.NotNil(t, loader.loadedPlugins)
}

func TestNewLoader_WithCustomLogger(t *testing.T) {
	customLogger := logrus.New()
	customLogger.SetLevel(logrus.DebugLevel)

	dirs := []string{"/tmp/plugins"}
	loader := NewLoader(dirs, customLogger)

	assert.NotNil(t, loader)
	assert.Equal(t, customLogger, loader.log)
}

func TestGetDefaultPluginDirectories(t *testing.T) {
	dirs := GetDefaultPluginDirectories()

	assert.NotEmpty(t, dirs)
	assert.Contains(t, dirs[0], ".spoke/plugins")
}

func TestSetBufPluginFactory(t *testing.T) {
	loader := NewLoader([]string{}, logrus.New())

	// Create a mock factory
	factoryCalled := false
	mockFactory := func(m *Manifest) (Plugin, error) {
		factoryCalled = true
		return NewBasicLanguagePlugin(m, "/tmp"), nil
	}

	// Set the factory
	loader.SetBufPluginFactory(mockFactory)

	// Verify it was set by trying to load a Buf plugin
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "buf-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	manifest := &Manifest{
		ID:          "buf-plugin",
		Name:        "Buf Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Description: "A Buf plugin",
		Author:      "Test Author",
		Metadata: map[string]string{
			"buf_registry": "buf.build",
		},
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	err = SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = loader.LoadPlugin(ctx, pluginDir)

	assert.NoError(t, err)
	assert.True(t, factoryCalled)
}

func TestDiscoverPlugins(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	// Create a valid manifest
	manifest := &Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Description: "A test plugin",
		Author:      "Test Author",
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	err = SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader([]string{tmpDir}, logrus.New())

	// Discover plugins
	ctx := context.Background()
	plugins, err := loader.DiscoverPlugins(ctx)

	assert.NoError(t, err)
	assert.Len(t, plugins, 1)
	assert.Equal(t, "test-plugin", plugins[0].Manifest().ID)
}

func TestDiscoverPlugins_InvalidManifest(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "invalid-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	// Create an invalid manifest (missing required fields)
	manifest := &Manifest{
		ID:   "invalid-plugin",
		Name: "Invalid Plugin",
		// Missing Version, APIVersion, Type
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	err = SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader([]string{tmpDir}, logrus.New())

	// Discover plugins - should skip invalid plugin
	ctx := context.Background()
	plugins, err := loader.DiscoverPlugins(ctx)

	assert.NoError(t, err)
	assert.Len(t, plugins, 0) // Invalid plugin should be skipped
}

func TestDiscoverPlugins_NonexistentDirectory(t *testing.T) {
	// Create loader with non-existent directory
	loader := NewLoader([]string{"/nonexistent/path"}, logrus.New())

	// Discover plugins - should not error
	ctx := context.Background()
	plugins, err := loader.DiscoverPlugins(ctx)

	assert.NoError(t, err)
	assert.Len(t, plugins, 0)
}

func TestDiscoverPlugins_MultipleDirectories(t *testing.T) {
	// Create two temporary plugin directories
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create plugin in first directory
	pluginDir1 := filepath.Join(tmpDir1, "plugin1")
	err := os.MkdirAll(pluginDir1, 0755)
	require.NoError(t, err)

	manifest1 := &Manifest{
		ID:          "plugin1",
		Name:        "Plugin 1",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Description: "First plugin",
		Author:      "Test Author",
	}

	err = SaveManifest(manifest1, filepath.Join(pluginDir1, "plugin.yaml"))
	require.NoError(t, err)

	// Create plugin in second directory
	pluginDir2 := filepath.Join(tmpDir2, "plugin2")
	err = os.MkdirAll(pluginDir2, 0755)
	require.NoError(t, err)

	manifest2 := &Manifest{
		ID:          "plugin2",
		Name:        "Plugin 2",
		Version:     "2.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Description: "Second plugin",
		Author:      "Test Author",
	}

	err = SaveManifest(manifest2, filepath.Join(pluginDir2, "plugin.yaml"))
	require.NoError(t, err)

	// Create loader with both directories
	loader := NewLoader([]string{tmpDir1, tmpDir2}, logrus.New())

	// Discover plugins
	ctx := context.Background()
	plugins, err := loader.DiscoverPlugins(ctx)

	assert.NoError(t, err)
	assert.Len(t, plugins, 2)

	// Verify both plugins were discovered
	pluginIDs := make(map[string]bool)
	for _, plugin := range plugins {
		pluginIDs[plugin.Manifest().ID] = true
	}
	assert.True(t, pluginIDs["plugin1"])
	assert.True(t, pluginIDs["plugin2"])
}

func TestDiscoverPlugins_SkipsFiles(t *testing.T) {
	// Create a temporary directory with files (not directories)
	tmpDir := t.TempDir()

	// Create a file (should be skipped)
	filePath := filepath.Join(tmpDir, "not-a-plugin.txt")
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader([]string{tmpDir}, logrus.New())

	// Discover plugins
	ctx := context.Background()
	plugins, err := loader.DiscoverPlugins(ctx)

	assert.NoError(t, err)
	assert.Len(t, plugins, 0) // File should be skipped
}

func TestDiscoverPlugins_WithMixedValidAndInvalid(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid plugin
	validDir := filepath.Join(tmpDir, "valid-plugin")
	err := os.MkdirAll(validDir, 0755)
	require.NoError(t, err)

	validManifest := &Manifest{
		ID:          "valid-plugin",
		Name:        "Valid Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Description: "A valid plugin",
		Author:      "Test Author",
	}
	err = SaveManifest(validManifest, filepath.Join(validDir, "plugin.yaml"))
	require.NoError(t, err)

	// Create invalid plugin (missing required fields)
	invalidDir := filepath.Join(tmpDir, "invalid-plugin")
	err = os.MkdirAll(invalidDir, 0755)
	require.NoError(t, err)

	invalidManifest := &Manifest{
		ID:   "invalid-plugin",
		Name: "Invalid Plugin",
		// Missing Version, APIVersion, Type
	}
	err = SaveManifest(invalidManifest, filepath.Join(invalidDir, "plugin.yaml"))
	require.NoError(t, err)

	// Create plugin with no manifest
	noManifestDir := filepath.Join(tmpDir, "no-manifest")
	err = os.MkdirAll(noManifestDir, 0755)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader([]string{tmpDir}, logrus.New())

	// Discover plugins
	ctx := context.Background()
	plugins, err := loader.DiscoverPlugins(ctx)

	// Should not error, but should only load valid plugin
	assert.NoError(t, err)
	assert.Len(t, plugins, 1)
	assert.Equal(t, "valid-plugin", plugins[0].Manifest().ID)
}

func TestLoadPlugin(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	// Create a valid manifest
	manifest := &Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Description: "A test plugin",
		Author:      "Test Author",
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	err = SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader([]string{tmpDir}, logrus.New())

	// Load plugin
	ctx := context.Background()
	plugin, err := loader.LoadPlugin(ctx, pluginDir)

	assert.NoError(t, err)
	assert.NotNil(t, plugin)
	assert.Equal(t, "test-plugin", plugin.Manifest().ID)
}

func TestLoadPlugin_MissingManifest(t *testing.T) {
	// Create a temporary plugin directory without manifest
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "no-manifest-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader([]string{tmpDir}, logrus.New())

	// Try to load plugin
	ctx := context.Background()
	_, err = loader.LoadPlugin(ctx, pluginDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load manifest")
}

func TestLoadPlugin_IncompatibleAPIVersion(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "incompatible-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	// Create a manifest with incompatible API version
	manifest := &Manifest{
		ID:          "incompatible-plugin",
		Name:        "Incompatible Plugin",
		Version:     "1.0.0",
		APIVersion:  "99.0.0", // Incompatible major version
		Type:        PluginTypeLanguage,
		Description: "A plugin with incompatible API",
		Author:      "Test Author",
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	err = SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader([]string{tmpDir}, logrus.New())

	// Try to load plugin
	ctx := context.Background()
	_, err = loader.LoadPlugin(ctx, pluginDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible API version")
}

func TestLoadPlugin_InvalidPluginType(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "invalid-type-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	// Create a manifest with invalid plugin type
	manifest := &Manifest{
		ID:         "invalid-type-plugin",
		Name:       "Invalid Type Plugin",
		Version:    "1.0.0",
		APIVersion: "1.0.0",
		Type:       PluginType("unknown"),
		Author:     "Test Author",
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	err = SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	// Create loader
	loader := NewLoader([]string{tmpDir}, logrus.New())

	// Try to load plugin
	ctx := context.Background()
	_, err = loader.LoadPlugin(ctx, pluginDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest validation failed")
}

func TestLoadPlugin_BufPluginWithoutFactory(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "buf-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	// Create a manifest with buf_registry metadata (indicates Buf plugin)
	manifest := &Manifest{
		ID:          "buf-plugin",
		Name:        "Buf Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Description: "A Buf plugin",
		Author:      "Test Author",
		Metadata: map[string]string{
			"buf_registry": "buf.build",
		},
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	err = SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	// Create loader without setting Buf factory
	loader := NewLoader([]string{tmpDir}, logrus.New())

	// Try to load plugin
	ctx := context.Background()
	_, err = loader.LoadPlugin(ctx, pluginDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "buf plugin factory not configured")
}

func TestUnloadPlugin(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	// Create a valid manifest
	manifest := &Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Description: "A test plugin",
		Author:      "Test Author",
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	err = SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	// Create loader and load plugin
	loader := NewLoader([]string{tmpDir}, logrus.New())
	ctx := context.Background()
	_, err = loader.LoadPlugin(ctx, pluginDir)
	require.NoError(t, err)

	// Verify plugin is loaded
	plugin, exists := loader.GetLoadedPlugin("test-plugin")
	assert.True(t, exists)
	assert.NotNil(t, plugin)

	// Unload plugin
	err = loader.UnloadPlugin(ctx, "test-plugin")
	assert.NoError(t, err)

	// Verify plugin is unloaded
	_, exists = loader.GetLoadedPlugin("test-plugin")
	assert.False(t, exists)
}

func TestUnloadPlugin_NotLoaded(t *testing.T) {
	loader := NewLoader([]string{}, logrus.New())
	ctx := context.Background()

	// Try to unload a plugin that was never loaded
	err := loader.UnloadPlugin(ctx, "nonexistent-plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin not loaded")
}

func TestGetLoadedPlugin(t *testing.T) {
	loader := NewLoader([]string{}, logrus.New())

	// Test non-existent plugin
	plugin, exists := loader.GetLoadedPlugin("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, plugin)
}

func TestListLoadedPlugins(t *testing.T) {
	// Create a temporary plugin directory with multiple plugins
	tmpDir := t.TempDir()

	for i := 1; i <= 3; i++ {
		pluginDir := filepath.Join(tmpDir, "test-plugin-"+string(rune('0'+i)))
		err := os.MkdirAll(pluginDir, 0755)
		require.NoError(t, err)

		manifest := &Manifest{
			ID:          "test-plugin-" + string(rune('0'+i)),
			Name:        "Test Plugin " + string(rune('0'+i)),
			Version:     "1.0.0",
			APIVersion:  "1.0.0",
			Type:        PluginTypeLanguage,
			Description: "A test plugin",
			Author:      "Test Author",
		}

		manifestPath := filepath.Join(pluginDir, "plugin.yaml")
		err = SaveManifest(manifest, manifestPath)
		require.NoError(t, err)
	}

	// Create loader and discover plugins
	loader := NewLoader([]string{tmpDir}, logrus.New())
	ctx := context.Background()
	_, err := loader.DiscoverPlugins(ctx)
	require.NoError(t, err)

	// List loaded plugins
	plugins := loader.ListLoadedPlugins()
	assert.Len(t, plugins, 3)
}

func TestListLoadedPlugins_Empty(t *testing.T) {
	loader := NewLoader([]string{}, logrus.New())

	plugins := loader.ListLoadedPlugins()

	assert.NotNil(t, plugins)
	assert.Len(t, plugins, 0)
}
