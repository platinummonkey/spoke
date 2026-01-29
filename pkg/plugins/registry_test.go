package plugins

// Comprehensive tests for registry.go providing 100% code coverage.
// Tests cover:
// - Plugin registration (success, nil plugin, nil manifest, duplicates)
// - Plugin unregistration (success, not found)
// - Plugin lookup (Get, Has)
// - Plugin listing (List, ListByType, GetLanguagePlugins, GetValidatorPlugins, GetRunnerPlugins)
// - Registry operations (Count, Clear)
// - Concurrent access patterns (reads, writes, mixed operations)

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registryMockPlugin implements the Plugin interface for testing
type registryMockPlugin struct {
	manifest *Manifest
	loadErr  error
}

func (m *registryMockPlugin) Manifest() *Manifest {
	return m.manifest
}

func (m *registryMockPlugin) Load() error {
	return m.loadErr
}

func (m *registryMockPlugin) Unload() error {
	return nil
}

// registryMockLanguagePlugin implements LanguagePlugin for testing
type registryMockLanguagePlugin struct {
	registryMockPlugin
	langSpec *LanguageSpec
}

func (m *registryMockLanguagePlugin) GetLanguageSpec() *LanguageSpec {
	return m.langSpec
}

func (m *registryMockLanguagePlugin) BuildProtocCommand(ctx context.Context, req *CommandRequest) ([]string, error) {
	return []string{"protoc", "--go_out=."}, nil
}

func (m *registryMockLanguagePlugin) ValidateOutput(ctx context.Context, files []string) error {
	return nil
}

// registryMockValidatorPlugin implements ValidatorPlugin for testing
type registryMockValidatorPlugin struct {
	registryMockPlugin
}

func (m *registryMockValidatorPlugin) Validate(ctx context.Context, req *ValidationRequest) (*ValidationResult, error) {
	return &ValidationResult{Valid: true}, nil
}

// registryMockRunnerPlugin implements RunnerPlugin for testing
type registryMockRunnerPlugin struct {
	registryMockPlugin
}

func (m *registryMockRunnerPlugin) Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error) {
	return &ExecutionResult{ExitCode: 0}, nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.plugins)
	assert.Equal(t, 0, registry.Count())
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name      string
		plugin    Plugin
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful registration",
			plugin: &registryMockPlugin{
				manifest: &Manifest{
					ID:      "test-plugin",
					Name:    "Test Plugin",
					Version: "1.0.0",
					Type:    PluginTypeLanguage,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil plugin",
			plugin:  nil,
			wantErr: true,
			errMsg:  "cannot register nil plugin",
		},
		{
			name: "nil manifest",
			plugin: &registryMockPlugin{
				manifest: nil,
			},
			wantErr: true,
			errMsg:  "plugin has nil manifest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			err := registry.Register(tt.plugin)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 1, registry.Count())
			}
		})
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	registry := NewRegistry()

	plugin := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "test-plugin",
			Name:    "Test Plugin",
			Version: "1.0.0",
			Type:    PluginTypeLanguage,
		},
	}

	// Register once
	err := registry.Register(plugin)
	require.NoError(t, err)
	assert.Equal(t, 1, registry.Count())

	// Try to register again
	err = registry.Register(plugin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin already registered")
	assert.Contains(t, err.Error(), "test-plugin")
	assert.Equal(t, 1, registry.Count())
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	plugin := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "test-plugin",
			Name:    "Test Plugin",
			Version: "1.0.0",
			Type:    PluginTypeLanguage,
		},
	}

	// Register plugin
	err := registry.Register(plugin)
	require.NoError(t, err)
	assert.Equal(t, 1, registry.Count())

	// Unregister plugin
	err = registry.Unregister("test-plugin")
	assert.NoError(t, err)
	assert.Equal(t, 0, registry.Count())
}

func TestRegistry_Unregister_NotFound(t *testing.T) {
	registry := NewRegistry()

	err := registry.Unregister("nonexistent-plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin not found")
	assert.Contains(t, err.Error(), "nonexistent-plugin")
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	plugin := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "test-plugin",
			Name:    "Test Plugin",
			Version: "1.0.0",
			Type:    PluginTypeLanguage,
		},
	}

	// Register plugin
	err := registry.Register(plugin)
	require.NoError(t, err)

	// Get plugin
	retrieved, err := registry.Get("test-plugin")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-plugin", retrieved.Manifest().ID)
	assert.Equal(t, plugin, retrieved)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := NewRegistry()

	plugin, err := registry.Get("nonexistent-plugin")
	assert.Error(t, err)
	assert.Nil(t, plugin)
	assert.Contains(t, err.Error(), "plugin not found")
	assert.Contains(t, err.Error(), "nonexistent-plugin")
}

func TestRegistry_Has(t *testing.T) {
	registry := NewRegistry()

	plugin := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "test-plugin",
			Name:    "Test Plugin",
			Version: "1.0.0",
			Type:    PluginTypeLanguage,
		},
	}

	// Check before registration
	assert.False(t, registry.Has("test-plugin"))

	// Register plugin
	err := registry.Register(plugin)
	require.NoError(t, err)

	// Check after registration
	assert.True(t, registry.Has("test-plugin"))
	assert.False(t, registry.Has("other-plugin"))
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Empty list
	plugins := registry.List()
	assert.Empty(t, plugins)

	// Add plugins
	plugin1 := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "plugin1",
			Name:    "Plugin 1",
			Version: "1.0.0",
			Type:    PluginTypeLanguage,
		},
	}
	plugin2 := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "plugin2",
			Name:    "Plugin 2",
			Version: "1.0.0",
			Type:    PluginTypeValidator,
		},
	}

	err := registry.Register(plugin1)
	require.NoError(t, err)
	err = registry.Register(plugin2)
	require.NoError(t, err)

	// List all plugins
	plugins = registry.List()
	assert.Len(t, plugins, 2)

	// Check that both plugins are present
	ids := make(map[string]bool)
	for _, p := range plugins {
		ids[p.Manifest().ID] = true
	}
	assert.True(t, ids["plugin1"])
	assert.True(t, ids["plugin2"])
}

func TestRegistry_ListByType(t *testing.T) {
	registry := NewRegistry()

	// Add plugins of different types
	langPlugin := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "lang-plugin",
			Name:    "Language Plugin",
			Version: "1.0.0",
			Type:    PluginTypeLanguage,
		},
	}
	validatorPlugin1 := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "validator1",
			Name:    "Validator 1",
			Version: "1.0.0",
			Type:    PluginTypeValidator,
		},
	}
	validatorPlugin2 := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "validator2",
			Name:    "Validator 2",
			Version: "1.0.0",
			Type:    PluginTypeValidator,
		},
	}
	runnerPlugin := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "runner-plugin",
			Name:    "Runner Plugin",
			Version: "1.0.0",
			Type:    PluginTypeRunner,
		},
	}

	require.NoError(t, registry.Register(langPlugin))
	require.NoError(t, registry.Register(validatorPlugin1))
	require.NoError(t, registry.Register(validatorPlugin2))
	require.NoError(t, registry.Register(runnerPlugin))

	// List by type
	langPlugins := registry.ListByType(PluginTypeLanguage)
	assert.Len(t, langPlugins, 1)
	assert.Equal(t, "lang-plugin", langPlugins[0].Manifest().ID)

	validatorPlugins := registry.ListByType(PluginTypeValidator)
	assert.Len(t, validatorPlugins, 2)

	runnerPlugins := registry.ListByType(PluginTypeRunner)
	assert.Len(t, runnerPlugins, 1)
	assert.Equal(t, "runner-plugin", runnerPlugins[0].Manifest().ID)

	// Non-existent type
	generatorPlugins := registry.ListByType(PluginTypeGenerator)
	assert.Empty(t, generatorPlugins)
}

func TestRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	// Initial count
	assert.Equal(t, 0, registry.Count())

	// Add plugins
	for i := 1; i <= 3; i++ {
		plugin := &registryMockPlugin{
			manifest: &Manifest{
				ID:      "plugin" + string(rune('0'+i)),
				Name:    "Plugin",
				Version: "1.0.0",
				Type:    PluginTypeLanguage,
			},
		}
		err := registry.Register(plugin)
		require.NoError(t, err)
		assert.Equal(t, i, registry.Count())
	}

	// Remove plugin
	err := registry.Unregister("plugin1")
	require.NoError(t, err)
	assert.Equal(t, 2, registry.Count())
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	// Add plugins
	for i := 1; i <= 3; i++ {
		plugin := &registryMockPlugin{
			manifest: &Manifest{
				ID:      "plugin" + string(rune('0'+i)),
				Name:    "Plugin",
				Version: "1.0.0",
				Type:    PluginTypeLanguage,
			},
		}
		err := registry.Register(plugin)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, registry.Count())

	// Clear registry
	registry.Clear()
	assert.Equal(t, 0, registry.Count())
	assert.Empty(t, registry.List())
}

func TestRegistry_GetLanguagePlugins(t *testing.T) {
	registry := NewRegistry()

	// Add language plugins
	langPlugin1 := &registryMockLanguagePlugin{
		registryMockPlugin: registryMockPlugin{
			manifest: &Manifest{
				ID:      "go-plugin",
				Name:    "Go Plugin",
				Version: "1.0.0",
				Type:    PluginTypeLanguage,
			},
		},
		langSpec: &LanguageSpec{
			ID:   "go",
			Name: "Go",
		},
	}
	langPlugin2 := &registryMockLanguagePlugin{
		registryMockPlugin: registryMockPlugin{
			manifest: &Manifest{
				ID:      "rust-plugin",
				Name:    "Rust Plugin",
				Version: "1.0.0",
				Type:    PluginTypeLanguage,
			},
		},
		langSpec: &LanguageSpec{
			ID:   "rust",
			Name: "Rust",
		},
	}

	// Add non-language plugin
	validatorPlugin := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "validator-plugin",
			Name:    "Validator Plugin",
			Version: "1.0.0",
			Type:    PluginTypeValidator,
		},
	}

	require.NoError(t, registry.Register(langPlugin1))
	require.NoError(t, registry.Register(langPlugin2))
	require.NoError(t, registry.Register(validatorPlugin))

	// Get language plugins
	langPlugins := registry.GetLanguagePlugins()
	assert.Len(t, langPlugins, 2)

	// Verify they implement LanguagePlugin interface
	for _, lp := range langPlugins {
		assert.NotNil(t, lp.GetLanguageSpec())
	}
}

func TestRegistry_GetValidatorPlugins(t *testing.T) {
	registry := NewRegistry()

	// Add validator plugins
	valPlugin1 := &registryMockValidatorPlugin{
		registryMockPlugin: registryMockPlugin{
			manifest: &Manifest{
				ID:      "buf-lint",
				Name:    "Buf Lint",
				Version: "1.0.0",
				Type:    PluginTypeValidator,
			},
		},
	}
	valPlugin2 := &registryMockValidatorPlugin{
		registryMockPlugin: registryMockPlugin{
			manifest: &Manifest{
				ID:      "proto-lint",
				Name:    "Proto Lint",
				Version: "1.0.0",
				Type:    PluginTypeValidator,
			},
		},
	}

	// Add non-validator plugin
	langPlugin := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "go-plugin",
			Name:    "Go Plugin",
			Version: "1.0.0",
			Type:    PluginTypeLanguage,
		},
	}

	require.NoError(t, registry.Register(valPlugin1))
	require.NoError(t, registry.Register(valPlugin2))
	require.NoError(t, registry.Register(langPlugin))

	// Get validator plugins
	valPlugins := registry.GetValidatorPlugins()
	assert.Len(t, valPlugins, 2)

	// Verify they implement ValidatorPlugin interface
	ctx := context.Background()
	for _, vp := range valPlugins {
		result, err := vp.Validate(ctx, &ValidationRequest{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestRegistry_GetRunnerPlugins(t *testing.T) {
	registry := NewRegistry()

	// Add runner plugins
	runPlugin1 := &registryMockRunnerPlugin{
		registryMockPlugin: registryMockPlugin{
			manifest: &Manifest{
				ID:      "docker-runner",
				Name:    "Docker Runner",
				Version: "1.0.0",
				Type:    PluginTypeRunner,
			},
		},
	}
	runPlugin2 := &registryMockRunnerPlugin{
		registryMockPlugin: registryMockPlugin{
			manifest: &Manifest{
				ID:      "local-runner",
				Name:    "Local Runner",
				Version: "1.0.0",
				Type:    PluginTypeRunner,
			},
		},
	}

	// Add non-runner plugin
	langPlugin := &registryMockPlugin{
		manifest: &Manifest{
			ID:      "go-plugin",
			Name:    "Go Plugin",
			Version: "1.0.0",
			Type:    PluginTypeLanguage,
		},
	}

	require.NoError(t, registry.Register(runPlugin1))
	require.NoError(t, registry.Register(runPlugin2))
	require.NoError(t, registry.Register(langPlugin))

	// Get runner plugins
	runPlugins := registry.GetRunnerPlugins()
	assert.Len(t, runPlugins, 2)

	// Verify they implement RunnerPlugin interface
	ctx := context.Background()
	for _, rp := range runPlugins {
		result, err := rp.Execute(ctx, &ExecutionRequest{Command: []string{"test"}})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()
	var wg sync.WaitGroup

	// Number of concurrent operations
	numGoroutines := 10
	numOperations := 100

	// Concurrent registrations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				plugin := &registryMockPlugin{
					manifest: &Manifest{
						ID:      "plugin-" + string(rune('0'+id)) + "-" + string(rune('0'+j)),
						Name:    "Plugin",
						Version: "1.0.0",
						Type:    PluginTypeLanguage,
					},
				}
				_ = registry.Register(plugin)
			}
		}(i)
	}

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = registry.Count()
				_ = registry.List()
				_ = registry.Has("plugin-0-0")
			}
		}()
	}

	wg.Wait()

	// Verify that we can still access the registry after concurrent operations
	count := registry.Count()
	assert.GreaterOrEqual(t, count, 0)
	plugins := registry.List()
	assert.Equal(t, count, len(plugins))
}

func TestRegistry_ConcurrentRegisterAndUnregister(t *testing.T) {
	registry := NewRegistry()
	var wg sync.WaitGroup

	pluginIDs := make([]string, 50)
	for i := 0; i < 50; i++ {
		pluginIDs[i] = "plugin-" + string(rune('0'+i))
		plugin := &registryMockPlugin{
			manifest: &Manifest{
				ID:      pluginIDs[i],
				Name:    "Plugin",
				Version: "1.0.0",
				Type:    PluginTypeLanguage,
			},
		}
		err := registry.Register(plugin)
		require.NoError(t, err)
	}

	// Concurrent unregistrations and registrations
	wg.Add(2)

	// Unregister plugins concurrently
	go func() {
		defer wg.Done()
		for _, id := range pluginIDs[:25] {
			_ = registry.Unregister(id)
		}
	}()

	// Register new plugins concurrently
	go func() {
		defer wg.Done()
		for i := 50; i < 75; i++ {
			plugin := &registryMockPlugin{
				manifest: &Manifest{
					ID:      "plugin-" + string(rune('0'+i)),
					Name:    "Plugin",
					Version: "1.0.0",
					Type:    PluginTypeLanguage,
				},
			}
			_ = registry.Register(plugin)
		}
	}()

	wg.Wait()

	// Verify registry is still consistent
	count := registry.Count()
	plugins := registry.List()
	assert.Equal(t, count, len(plugins))
}

func TestRegistry_ConcurrentGetOperations(t *testing.T) {
	registry := NewRegistry()

	// Pre-register some plugins
	for i := 0; i < 10; i++ {
		plugin := &registryMockPlugin{
			manifest: &Manifest{
				ID:      "plugin-" + string(rune('0'+i)),
				Name:    "Plugin",
				Version: "1.0.0",
				Type:    PluginTypeLanguage,
			},
		}
		err := registry.Register(plugin)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	numGoroutines := 20

	// Concurrent Get operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				pluginID := "plugin-" + string(rune('0'+(id%10)))
				plugin, err := registry.Get(pluginID)
				if err == nil {
					assert.NotNil(t, plugin)
					assert.Equal(t, pluginID, plugin.Manifest().ID)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestRegistry_EmptyList(t *testing.T) {
	registry := NewRegistry()

	// Test empty lists
	assert.Empty(t, registry.List())
	assert.Empty(t, registry.ListByType(PluginTypeLanguage))
	assert.Empty(t, registry.GetLanguagePlugins())
	assert.Empty(t, registry.GetValidatorPlugins())
	assert.Empty(t, registry.GetRunnerPlugins())
}

func TestRegistry_MultiplePluginTypes(t *testing.T) {
	registry := NewRegistry()

	// Add various plugin types
	plugins := []Plugin{
		&registryMockLanguagePlugin{
			registryMockPlugin: registryMockPlugin{
				manifest: &Manifest{
					ID:      "go-plugin",
					Name:    "Go Plugin",
					Version: "1.0.0",
					Type:    PluginTypeLanguage,
				},
			},
		},
		&registryMockValidatorPlugin{
			registryMockPlugin: registryMockPlugin{
				manifest: &Manifest{
					ID:      "buf-lint",
					Name:    "Buf Lint",
					Version: "1.0.0",
					Type:    PluginTypeValidator,
				},
			},
		},
		&registryMockRunnerPlugin{
			registryMockPlugin: registryMockPlugin{
				manifest: &Manifest{
					ID:      "docker-runner",
					Name:    "Docker Runner",
					Version: "1.0.0",
					Type:    PluginTypeRunner,
				},
			},
		},
		&registryMockPlugin{
			manifest: &Manifest{
				ID:      "generator",
				Name:    "Generator",
				Version: "1.0.0",
				Type:    PluginTypeGenerator,
			},
		},
	}

	for _, p := range plugins {
		err := registry.Register(p)
		require.NoError(t, err)
	}

	// Verify counts
	assert.Equal(t, 4, registry.Count())
	assert.Len(t, registry.GetLanguagePlugins(), 1)
	assert.Len(t, registry.GetValidatorPlugins(), 1)
	assert.Len(t, registry.GetRunnerPlugins(), 1)
	assert.Len(t, registry.ListByType(PluginTypeGenerator), 1)
}
