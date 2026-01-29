package plugins

import (
	"context"
	"time"
)

// Plugin is the base interface all plugins must implement
type Plugin interface {
	Manifest() *Manifest
	Load() error
	Unload() error
}

// Manifest describes plugin metadata
type Manifest struct {
	ID          string            `yaml:"id"`          // Unique ID (e.g., "rust-language")
	Name        string            `yaml:"name"`        // Display name
	Version     string            `yaml:"version"`     // Semver
	APIVersion  string            `yaml:"api_version"` // SDK API version
	Description string            `yaml:"description"` // Short description
	Author      string            `yaml:"author"`      // Author name
	License     string            `yaml:"license"`     // License (e.g., MIT, Apache-2.0)
	Homepage    string            `yaml:"homepage"`    // Homepage URL
	Repository  string            `yaml:"repository"`  // Repository URL
	Type        PluginType        `yaml:"type"`        // Plugin type
	Metadata    map[string]string `yaml:"metadata"`    // Additional metadata
}

// PluginType defines the category of plugin
type PluginType string

const (
	PluginTypeLanguage PluginType = "language"
)

// PluginInfo contains runtime information about a loaded plugin
type PluginInfo struct {
	Manifest  *Manifest
	LoadedAt  time.Time
	IsEnabled bool
	Source    string // filesystem, marketplace, buf
}

// PluginRegistry manages loaded plugins
type PluginRegistry interface {
	Register(plugin Plugin) error
	Unregister(id string) error
	Get(id string) (Plugin, error)
	List() []Plugin
	ListByType(t PluginType) []Plugin
}

// ValidationError represents a manifest validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// PluginLoader is responsible for discovering and loading plugins
type PluginLoader interface {
	DiscoverPlugins(ctx context.Context) ([]Plugin, error)
	LoadPlugin(ctx context.Context, path string) (Plugin, error)
	UnloadPlugin(ctx context.Context, id string) error
}
