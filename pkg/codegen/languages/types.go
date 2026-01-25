package languages

// LanguageSpec defines the configuration for a language compiler
type LanguageSpec struct {
	// Identification
	ID              string `json:"id"`               // "go", "python", "java"
	Name            string `json:"name"`             // "Go", "Python", "Java"
	DisplayName     string `json:"display_name"`     // "Go (Protocol Buffers)"

	// Plugin information
	ProtocPlugin    string   `json:"protoc_plugin"`    // "protoc-gen-go"
	PluginVersion   string   `json:"plugin_version"`   // "v1.31.0"
	ProtocFlags     []string `json:"protoc_flags"`     // ["--go_opt=paths=source_relative"]

	// Docker configuration
	DockerImage     string `json:"docker_image"`     // "spoke/compiler-go:1.31.0"
	DockerTag       string `json:"docker_tag"`       // "1.31.0"

	// gRPC support
	SupportsGRPC    bool   `json:"supports_grpc"`
	GRPCPlugin      string `json:"grpc_plugin"`      // "protoc-gen-go-grpc"
	GRPCPluginVersion string `json:"grpc_plugin_version"` // "v1.3.0"
	GRPCFlags       []string `json:"grpc_flags"`      // ["--go-grpc_opt=paths=source_relative"]

	// Package manager
	PackageManager  *PackageManagerSpec `json:"package_manager,omitempty"`

	// File extensions
	FileExtensions  []string `json:"file_extensions"` // [".pb.go", "_pb2.py"]

	// Status
	Enabled         bool `json:"enabled"`
	Stable          bool `json:"stable"`
	Experimental    bool `json:"experimental"`

	// Documentation
	Description     string `json:"description"`
	DocumentationURL string `json:"documentation_url"`
}

// PackageManagerSpec defines package manager configuration
type PackageManagerSpec struct {
	Name            string            `json:"name"`              // "go-modules", "pip", "npm", "maven"
	ConfigFiles     []string          `json:"config_files"`      // ["go.mod", "go.sum"]
	TemplateDir     string            `json:"template_dir"`      // Path to template files
	DependencyMap   map[string]string `json:"dependency_map"`    // Map proto imports to package dependencies
	DefaultVersions map[string]string `json:"default_versions"`  // Default package versions
}

// Validate checks if the language spec is valid
func (ls *LanguageSpec) Validate() error {
	if ls.ID == "" {
		return ErrInvalidLanguageID
	}
	if ls.Name == "" {
		return ErrInvalidLanguageName
	}
	if ls.DockerImage == "" {
		return ErrInvalidDockerImage
	}
	if ls.ProtocPlugin == "" {
		return ErrInvalidProtocPlugin
	}
	return nil
}

// GetFullDockerImage returns the full Docker image reference
func (ls *LanguageSpec) GetFullDockerImage() string {
	if ls.DockerTag != "" {
		return ls.DockerImage + ":" + ls.DockerTag
	}
	return ls.DockerImage
}

// Common language IDs
const (
	LanguageGo         = "go"
	LanguagePython     = "python"
	LanguageJava       = "java"
	LanguageCPP        = "cpp"
	LanguageCSharp     = "csharp"
	LanguageRust       = "rust"
	LanguageTypeScript = "typescript"
	LanguageJavaScript = "javascript"
	LanguageDart       = "dart"
	LanguageSwift      = "swift"
	LanguageKotlin     = "kotlin"
	LanguageObjectiveC = "objc"
	LanguageRuby       = "ruby"
	LanguagePHP        = "php"
	LanguageScala      = "scala"
)
