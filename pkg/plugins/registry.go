package plugins

import (
	"fmt"
	"sync"
)

var (
	// plugins is the package-level registry map
	plugins = make(map[string]Plugin)
	// mu protects concurrent access to plugins map
	mu sync.RWMutex
)

// Register adds a plugin to the registry
func Register(plugin Plugin) error {
	if plugin == nil {
		return fmt.Errorf("cannot register nil plugin")
	}

	manifest := plugin.Manifest()
	if manifest == nil {
		return fmt.Errorf("plugin has nil manifest")
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := plugins[manifest.ID]; exists {
		return fmt.Errorf("plugin already registered: %s", manifest.ID)
	}

	plugins[manifest.ID] = plugin
	return nil
}

// Unregister removes a plugin from the registry
func Unregister(id string) error {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := plugins[id]; !exists {
		return fmt.Errorf("plugin not found: %s", id)
	}

	delete(plugins, id)
	return nil
}

// Get retrieves a plugin by ID
func Get(id string) (Plugin, error) {
	mu.RLock()
	defer mu.RUnlock()

	plugin, exists := plugins[id]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", id)
	}

	return plugin, nil
}

// Has checks if a plugin is registered
func Has(id string) bool {
	mu.RLock()
	defer mu.RUnlock()

	_, exists := plugins[id]
	return exists
}

// List returns all registered plugins
func List() []Plugin {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]Plugin, 0, len(plugins))
	for _, plugin := range plugins {
		result = append(result, plugin)
	}

	return result
}

// ListByType returns all plugins of a specific type
func ListByType(t PluginType) []Plugin {
	mu.RLock()
	defer mu.RUnlock()

	var result []Plugin
	for _, plugin := range plugins {
		if plugin.Manifest().Type == t {
			result = append(result, plugin)
		}
	}

	return result
}

// Count returns the number of registered plugins
func Count() int {
	mu.RLock()
	defer mu.RUnlock()

	return len(plugins)
}

// Clear removes all plugins from the registry
func Clear() {
	mu.Lock()
	defer mu.Unlock()

	plugins = make(map[string]Plugin)
}

// GetLanguagePlugins returns all language plugins
func GetLanguagePlugins() []LanguagePlugin {
	mu.RLock()
	defer mu.RUnlock()

	var result []LanguagePlugin
	for _, plugin := range plugins {
		if langPlugin, ok := plugin.(LanguagePlugin); ok {
			result = append(result, langPlugin)
		}
	}

	return result
}
