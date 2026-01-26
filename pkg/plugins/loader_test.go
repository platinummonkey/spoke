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

func TestGetDefaultPluginDirectories(t *testing.T) {
	dirs := GetDefaultPluginDirectories()

	assert.NotEmpty(t, dirs)
	assert.Contains(t, dirs[0], ".spoke/plugins")
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
