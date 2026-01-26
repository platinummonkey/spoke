package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	// CurrentSDKAPIVersion is the current version of the Plugin SDK API
	CurrentSDKAPIVersion = "1.0.0"
)

// BufPluginFactory is a function that creates a Buf plugin from a manifest
type BufPluginFactory func(*Manifest) (Plugin, error)

// Loader discovers and loads plugins from filesystem directories
type Loader struct {
	pluginDirs       []string
	loadedPlugins    map[string]Plugin
	bufPluginFactory BufPluginFactory
	mu               sync.RWMutex
	log              *logrus.Logger
}

// NewLoader creates a new plugin loader
func NewLoader(dirs []string, log *logrus.Logger) *Loader {
	if log == nil {
		log = logrus.New()
	}

	return &Loader{
		pluginDirs:       dirs,
		loadedPlugins:    make(map[string]Plugin),
		bufPluginFactory: nil, // Can be set later with SetBufPluginFactory
		log:              log,
	}
}

// SetBufPluginFactory sets the factory function for creating Buf plugins
func (l *Loader) SetBufPluginFactory(factory BufPluginFactory) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.bufPluginFactory = factory
}

// DiscoverPlugins scans plugin directories and returns discovered plugins
func (l *Loader) DiscoverPlugins(ctx context.Context) ([]Plugin, error) {
	var plugins []Plugin

	for _, dir := range l.pluginDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			l.log.Debugf("Plugin directory does not exist: %s", dir)
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			l.log.Warnf("Failed to read plugin directory %s: %v", dir, err)
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			pluginDir := filepath.Join(dir, entry.Name())
			plugin, err := l.loadPluginFromDir(ctx, pluginDir)
			if err != nil {
				l.log.Warnf("Failed to load plugin from %s: %v", pluginDir, err)
				continue
			}

			plugins = append(plugins, plugin)
		}
	}

	return plugins, nil
}

// LoadPlugin loads a single plugin from a path
func (l *Loader) LoadPlugin(ctx context.Context, path string) (Plugin, error) {
	return l.loadPluginFromDir(ctx, path)
}

// UnloadPlugin unloads a plugin by ID
func (l *Loader) UnloadPlugin(ctx context.Context, id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	plugin, exists := l.loadedPlugins[id]
	if !exists {
		return fmt.Errorf("plugin not loaded: %s", id)
	}

	if err := plugin.Unload(); err != nil {
		return fmt.Errorf("failed to unload plugin: %w", err)
	}

	delete(l.loadedPlugins, id)
	return nil
}

// GetLoadedPlugin returns a loaded plugin by ID
func (l *Loader) GetLoadedPlugin(id string) (Plugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugin, exists := l.loadedPlugins[id]
	return plugin, exists
}

// ListLoadedPlugins returns all loaded plugins
func (l *Loader) ListLoadedPlugins() []Plugin {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugins := make([]Plugin, 0, len(l.loadedPlugins))
	for _, plugin := range l.loadedPlugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// loadPluginFromDir loads a plugin from a directory
func (l *Loader) loadPluginFromDir(ctx context.Context, pluginDir string) (Plugin, error) {
	// Load manifest
	manifest, err := LoadManifestFromDir(pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	// Validate manifest
	if validationErrors := ValidateManifest(manifest); len(validationErrors) > 0 {
		return nil, fmt.Errorf("manifest validation failed: %v", validationErrors)
	}

	// Check API version compatibility
	if !IsCompatibleAPIVersion(manifest.APIVersion, CurrentSDKAPIVersion) {
		return nil, fmt.Errorf("incompatible API version: plugin requires %s, SDK is %s",
			manifest.APIVersion, CurrentSDKAPIVersion)
	}

	// Load plugin based on type
	var plugin Plugin

	switch manifest.Type {
	case PluginTypeLanguage:
		plugin, err = l.loadLanguagePlugin(ctx, pluginDir, manifest)
	case PluginTypeValidator:
		plugin, err = l.loadValidatorPlugin(ctx, pluginDir, manifest)
	case PluginTypeRunner:
		plugin, err = l.loadRunnerPlugin(ctx, pluginDir, manifest)
	default:
		return nil, fmt.Errorf("unsupported plugin type: %s", manifest.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	// Load the plugin
	if err := plugin.Load(); err != nil {
		return nil, fmt.Errorf("plugin load failed: %w", err)
	}

	// Store in loaded plugins map
	l.mu.Lock()
	l.loadedPlugins[manifest.ID] = plugin
	l.mu.Unlock()

	l.log.Infof("Loaded plugin: %s v%s (type: %s)", manifest.Name, manifest.Version, manifest.Type)

	return plugin, nil
}

// loadLanguagePlugin loads a language plugin
func (l *Loader) loadLanguagePlugin(ctx context.Context, pluginDir string, manifest *Manifest) (Plugin, error) {
	// Check if this is a Buf plugin
	if isBufPlugin(manifest) {
		return l.loadBufPlugin(ctx, manifest)
	}

	// Otherwise, create a basic language plugin wrapper
	return NewBasicLanguagePlugin(manifest, pluginDir), nil
}

// loadBufPlugin loads a Buf plugin
func (l *Loader) loadBufPlugin(ctx context.Context, manifest *Manifest) (Plugin, error) {
	bufAdapter, err := l.createBufAdapter(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to create Buf plugin adapter: %w", err)
	}

	return bufAdapter, nil
}

// isBufPlugin checks if a manifest describes a Buf plugin
func isBufPlugin(manifest *Manifest) bool {
	_, hasBufRegistry := manifest.Metadata["buf_registry"]
	return hasBufRegistry
}

// createBufAdapter creates a Buf plugin adapter using the factory
func (l *Loader) createBufAdapter(manifest *Manifest) (Plugin, error) {
	l.mu.RLock()
	factory := l.bufPluginFactory
	l.mu.RUnlock()

	if factory == nil {
		return nil, fmt.Errorf("Buf plugin factory not configured (Buf plugin support disabled)")
	}

	return factory(manifest)
}

// loadValidatorPlugin loads a validator plugin
func (l *Loader) loadValidatorPlugin(ctx context.Context, pluginDir string, manifest *Manifest) (Plugin, error) {
	// For now, return error - will implement in later phases
	return nil, fmt.Errorf("validator plugins not yet implemented")
}

// loadRunnerPlugin loads a runner plugin
func (l *Loader) loadRunnerPlugin(ctx context.Context, pluginDir string, manifest *Manifest) (Plugin, error) {
	// For now, return error - will implement in later phases
	return nil, fmt.Errorf("runner plugins not yet implemented")
}

// GetDefaultPluginDirectories returns the default plugin search directories
func GetDefaultPluginDirectories() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}

	return []string{
		filepath.Join(homeDir, ".spoke", "plugins"),
		"/etc/spoke/plugins",
		"./plugins", // Current directory
	}
}
