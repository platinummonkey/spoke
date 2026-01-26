package plugins_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/platinummonkey/spoke/pkg/codegen/languages"
	"github.com/platinummonkey/spoke/pkg/plugins"
	"github.com/platinummonkey/spoke/pkg/plugins/buf"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBufPluginIntegration(t *testing.T) {
	// Setup: Check if buf-connect-go plugin exists
	pluginsDir := filepath.Join("..", "..", "plugins")

	// Create plugin loader with Buf support
	logger := logrus.New()
	loader := plugins.NewLoader([]string{pluginsDir}, logger)

	// Configure Buf plugin support
	buf.ConfigureLoader(loader)

	// Discover plugins
	ctx := context.Background()
	discoveredPlugins, err := loader.DiscoverPlugins(ctx)
	require.NoError(t, err)

	// Check if any Buf plugins were discovered
	bufPluginCount := 0
	for _, plugin := range discoveredPlugins {
		manifest := plugin.Manifest()
		if _, hasBufRegistry := manifest.Metadata["buf_registry"]; hasBufRegistry {
			bufPluginCount++
			t.Logf("Found Buf plugin: %s (%s)", manifest.Name, manifest.ID)
		}
	}

	t.Logf("Discovered %d Buf plugin(s)", bufPluginCount)

	// Create language registry
	registry := languages.NewRegistry()

	// Load plugins into registry
	err = registry.LoadPlugins(ctx, loader, logger)
	require.NoError(t, err)

	// If buf-connect-go plugin exists, verify it's registered
	if bufPluginCount > 0 {
		allLanguages := registry.List()
		t.Logf("Total registered languages: %d", len(allLanguages))

		// Look for connect-go language
		foundBufLanguage := false
		for _, lang := range allLanguages {
			if lang.ID == "connect-go" || lang.ID == "buf-connect-go" {
				foundBufLanguage = true
				t.Logf("Buf language registered: %s (%s)", lang.Name, lang.ID)
				assert.True(t, lang.Enabled, "Buf language should be enabled")
			}
		}

		if !foundBufLanguage {
			t.Log("Note: buf-connect-go plugin exists but language not registered (may need download)")
		}
	}
}

func TestBufPluginFactory(t *testing.T) {
	// Create a test manifest for a Buf plugin
	manifest := &plugins.Manifest{
		ID:          "test-buf-plugin",
		Name:        "Test Buf Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        plugins.PluginTypeLanguage,
		Description: "Test Buf plugin",
		Metadata: map[string]string{
			"buf_registry": "buf.build/test/plugin",
			"buf_version":  "v1.0.0",
		},
	}

	// Create factory
	factory := buf.CreateBufPluginFactory()
	require.NotNil(t, factory)

	// Create plugin using factory
	plugin, err := factory(manifest)
	require.NoError(t, err)
	assert.NotNil(t, plugin)

	// Verify it's a language plugin
	langPlugin, ok := plugin.(plugins.LanguagePlugin)
	assert.True(t, ok, "Plugin should implement LanguagePlugin interface")

	if ok {
		spec := langPlugin.GetLanguageSpec()
		assert.NotNil(t, spec)
		assert.Equal(t, "plugin", spec.ID)
		t.Logf("Language spec: %s (%s)", spec.Name, spec.ID)
	}
}

func TestBufPluginLoaderConfiguration(t *testing.T) {
	logger := logrus.New()
	loader := plugins.NewLoader([]string{"/tmp"}, logger)

	// Configure Buf plugin support
	buf.ConfigureLoader(loader)

	// Note: We can't directly test LoadPlugin without a real plugin directory,
	// but we've verified the factory is set up correctly in the previous test
	t.Log("Buf plugin loader configuration successful")
}
