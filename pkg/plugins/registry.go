package plugins

import (
	"fmt"
	"sync"
)

// Registry manages loaded plugins
type Registry struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
	}
}

// Register adds a plugin to the registry
func (r *Registry) Register(plugin Plugin) error {
	if plugin == nil {
		return fmt.Errorf("cannot register nil plugin")
	}

	manifest := plugin.Manifest()
	if manifest == nil {
		return fmt.Errorf("plugin has nil manifest")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[manifest.ID]; exists {
		return fmt.Errorf("plugin already registered: %s", manifest.ID)
	}

	r.plugins[manifest.ID] = plugin
	return nil
}

// Unregister removes a plugin from the registry
func (r *Registry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[id]; !exists {
		return fmt.Errorf("plugin not found: %s", id)
	}

	delete(r.plugins, id)
	return nil
}

// Get retrieves a plugin by ID
func (r *Registry) Get(id string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[id]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", id)
	}

	return plugin, nil
}

// Has checks if a plugin is registered
func (r *Registry) Has(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.plugins[id]
	return exists
}

// List returns all registered plugins
func (r *Registry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// ListByType returns all plugins of a specific type
func (r *Registry) ListByType(t PluginType) []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var plugins []Plugin
	for _, plugin := range r.plugins {
		if plugin.Manifest().Type == t {
			plugins = append(plugins, plugin)
		}
	}

	return plugins
}

// Count returns the number of registered plugins
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.plugins)
}

// Clear removes all plugins from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins = make(map[string]Plugin)
}

// GetLanguagePlugins returns all language plugins
func (r *Registry) GetLanguagePlugins() []LanguagePlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var plugins []LanguagePlugin
	for _, plugin := range r.plugins {
		if langPlugin, ok := plugin.(LanguagePlugin); ok {
			plugins = append(plugins, langPlugin)
		}
	}

	return plugins
}

// GetValidatorPlugins returns all validator plugins
func (r *Registry) GetValidatorPlugins() []ValidatorPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var plugins []ValidatorPlugin
	for _, plugin := range r.plugins {
		if valPlugin, ok := plugin.(ValidatorPlugin); ok {
			plugins = append(plugins, valPlugin)
		}
	}

	return plugins
}

// GetRunnerPlugins returns all runner plugins
func (r *Registry) GetRunnerPlugins() []RunnerPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var plugins []RunnerPlugin
	for _, plugin := range r.plugins {
		if runPlugin, ok := plugin.(RunnerPlugin); ok {
			plugins = append(plugins, runPlugin)
		}
	}

	return plugins
}
