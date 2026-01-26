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
	ID            string            `yaml:"id"`              // Unique ID (e.g., "rust-language")
	Name          string            `yaml:"name"`            // Display name
	Version       string            `yaml:"version"`         // Semver
	APIVersion    string            `yaml:"api_version"`     // SDK API version
	Description   string            `yaml:"description"`     // Short description
	Author        string            `yaml:"author"`          // Author name
	License       string            `yaml:"license"`         // License (e.g., MIT, Apache-2.0)
	Homepage      string            `yaml:"homepage"`        // Homepage URL
	Repository    string            `yaml:"repository"`      // Repository URL
	Type          PluginType        `yaml:"type"`            // Plugin type
	SecurityLevel SecurityLevel     `yaml:"security_level"`  // Security level
	Permissions   []string          `yaml:"permissions"`     // Required permissions
	Dependencies  []string          `yaml:"dependencies"`    // Other plugin IDs
	Metadata      map[string]string `yaml:"metadata"`        // Additional metadata
}

// PluginType defines the category of plugin
type PluginType string

const (
	PluginTypeLanguage  PluginType = "language"
	PluginTypeValidator PluginType = "validator"
	PluginTypeGenerator PluginType = "generator"
	PluginTypeRunner    PluginType = "runner"
	PluginTypeTransform PluginType = "transform"
)

// SecurityLevel defines the trust level of a plugin
type SecurityLevel string

const (
	SecurityLevelOfficial  SecurityLevel = "official"
	SecurityLevelVerified  SecurityLevel = "verified"
	SecurityLevelCommunity SecurityLevel = "community"
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
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // error, warning
}

// SecurityIssue represents a security concern found during scanning
type SecurityIssue struct {
	Severity       string `json:"severity"`        // critical, high, medium, low, warning
	Category       string `json:"category"`        // imports, hardcoded-secrets, sql-injection, etc.
	Description    string `json:"description"`     // Human-readable description
	File           string `json:"file,omitempty"`  // File path
	Line           int    `json:"line,omitempty"`  // Line number
	Column         int    `json:"column,omitempty"` // Column number
	Recommendation string `json:"recommendation,omitempty"` // How to fix
	CWEID          string `json:"cwe_id,omitempty"` // Common Weakness Enumeration ID
}

// PluginValidationResult contains the complete plugin validation results
// (distinct from ValidatorPlugin's ValidationResult which is for proto validation)
type PluginValidationResult struct {
	Valid            bool              `json:"valid"`
	ManifestErrors   []ValidationError `json:"manifest_errors,omitempty"`
	SecurityIssues   []SecurityIssue   `json:"security_issues,omitempty"`
	PermissionIssues []ValidationError `json:"permission_issues,omitempty"`
	ScanDuration     time.Duration     `json:"scan_duration"`
	Recommendations  []string          `json:"recommendations,omitempty"`
}

// PluginLoader is responsible for discovering and loading plugins
type PluginLoader interface {
	DiscoverPlugins(ctx context.Context) ([]Plugin, error)
	LoadPlugin(ctx context.Context, path string) (Plugin, error)
	UnloadPlugin(ctx context.Context, id string) error
}
