package plugins

import (
	"context"
)

// LanguagePlugin extends Plugin for language-specific plugins
type LanguagePlugin interface {
	Plugin
	GetLanguageSpec() *LanguageSpec
	BuildProtocCommand(ctx context.Context, req *CommandRequest) ([]string, error)
	ValidateOutput(ctx context.Context, files []string) error
}

// LanguageSpec defines a programming language supported for code generation
type LanguageSpec struct {
	ID               string            `yaml:"id" json:"id"`
	Name             string            `yaml:"name" json:"name"`
	DisplayName      string            `yaml:"display_name" json:"display_name"`
	SupportsGRPC     bool              `yaml:"supports_grpc" json:"supports_grpc"`
	FileExtensions   []string          `yaml:"file_extensions" json:"file_extensions"`
	Enabled          bool              `yaml:"enabled" json:"enabled"`
	Stable           bool              `yaml:"stable" json:"stable"`
	Description      string            `yaml:"description" json:"description"`
	DocumentationURL string            `yaml:"documentation_url" json:"documentation_url"`
	PluginVersion    string            `yaml:"plugin_version" json:"plugin_version"`
	ProtocPlugin     string            `yaml:"protoc_plugin" json:"protoc_plugin"`
	DockerImage      string            `yaml:"docker_image" json:"docker_image"`
	PackageManager   *PackageManager   `yaml:"package_manager,omitempty" json:"package_manager,omitempty"`
	CustomOptions    map[string]string `yaml:"custom_options,omitempty" json:"custom_options,omitempty"`
}

// PackageManager defines package management configuration for a language
type PackageManager struct {
	Name            string            `yaml:"name" json:"name"`
	ConfigFiles     []string          `yaml:"config_files" json:"config_files"`
	DependencyMap   map[string]string `yaml:"dependency_map" json:"dependency_map"`
	DefaultVersions map[string]string `yaml:"default_versions" json:"default_versions"`
}

// CommandRequest contains information needed to build a protoc command
type CommandRequest struct {
	ProtoFiles  []string          `json:"proto_files"`
	ImportPaths []string          `json:"import_paths"`
	OutputDir   string            `json:"output_dir"`
	Options     map[string]string `json:"options"`
	PluginPath  string            `json:"plugin_path,omitempty"`
}

// CommandResult contains the result of executing a protoc command
type CommandResult struct {
	ExitCode      int      `json:"exit_code"`
	Stdout        string   `json:"stdout"`
	Stderr        string   `json:"stderr"`
	GeneratedFiles []string `json:"generated_files"`
}
