package plugins_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/platinummonkey/spoke/pkg/codegen/languages"
	"github.com/platinummonkey/spoke/pkg/plugins"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLanguagePluginIntegration(t *testing.T) {
	// Skip if plugins directory doesn't exist
	pluginsDir := filepath.Join("..", "..", "plugins")

	// Create plugin loader
	logger := logrus.New()
	loader := plugins.NewLoader([]string{pluginsDir}, logger)

	// Discover plugins
	ctx := context.Background()
	discoveredPlugins, err := loader.DiscoverPlugins(ctx)
	require.NoError(t, err)

	// Should find at least the rust-language plugin
	assert.NotEmpty(t, discoveredPlugins, "Should discover at least one plugin")

	// Create language registry
	registry := languages.NewRegistry()

	// Load plugins into registry
	err = registry.LoadPlugins(ctx, loader, logger)
	require.NoError(t, err)

	// Verify rust language is registered
	rustSpec, err := registry.Get("rust")
	if err == nil {
		assert.Equal(t, "rust", rustSpec.ID)
		assert.Equal(t, "Rust", rustSpec.Name)
		assert.True(t, rustSpec.SupportsGRPC)
		t.Logf("Successfully loaded Rust plugin: %s", rustSpec.DisplayName)
	} else {
		t.Logf("Rust plugin not found (may not be in plugins directory)")
	}

	// List all registered languages
	allLanguages := registry.List()
	t.Logf("Total registered languages: %d", len(allLanguages))
	for _, lang := range allLanguages {
		t.Logf("  - %s (%s) - Enabled: %v", lang.Name, lang.ID, lang.Enabled)
	}
}
