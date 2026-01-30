package api

import (
	"errors"
	"time"
)

// Common errors
var (
	ErrNotFound = errors.New("not found")
)

// Module represents a protobuf module with its metadata
type Module struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SourceInfo represents information about the source code of a version
type SourceInfo struct {
	Repository string `json:"repository"` // Can be github.com URL or custom domain
	CommitSHA  string `json:"commit_sha"`
	Branch     string `json:"branch"`
}

// Language represents supported programming languages
type Language string

const (
	LanguageGo         Language = "go"
	LanguagePython     Language = "python"
	LanguageJava       Language = "java"
	LanguageCPP        Language = "cpp"
	LanguageCSharp     Language = "csharp"
	LanguageRust       Language = "rust"
	LanguageTypeScript Language = "typescript"
	LanguageJavaScript Language = "javascript"
	LanguageDart       Language = "dart"
	LanguageSwift      Language = "swift"
	LanguageKotlin     Language = "kotlin"
	LanguageObjectiveC Language = "objc"
	LanguageRuby       Language = "ruby"
	LanguagePHP        Language = "php"
	LanguageScala      Language = "scala"
)

// CompilationInfo contains information about compiled libraries
type CompilationInfo struct {
	Language    Language `json:"language"`
	PackageName string   `json:"package_name"`
	Version     string   `json:"version"`
	Files       []File   `json:"files"`
}

// Version represents a specific version of a protobuf module
type Version struct {
	ModuleName       string           `json:"module_name"`
	Version          string           `json:"version"` // Can be semantic version or commit hash
	Files            []File           `json:"files"`
	CreatedAt        time.Time        `json:"created_at"`
	Dependencies     []string         `json:"dependencies,omitempty"`
	SourceInfo       SourceInfo       `json:"source_info"`
	CompilationInfo  []CompilationInfo `json:"compilation_info,omitempty"`
}

// File represents a single protobuf file
type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// LanguageInfo represents information about a supported language
type LanguageInfo struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	DisplayName      string            `json:"display_name"`
	SupportsGRPC     bool              `json:"supports_grpc"`
	FileExtensions   []string          `json:"file_extensions"`
	Enabled          bool              `json:"enabled"`
	Stable           bool              `json:"stable"`
	Description      string            `json:"description"`
	DocumentationURL string            `json:"documentation_url"`
	PluginVersion    string            `json:"plugin_version"`
	PackageManager   *PackageManagerInfo `json:"package_manager,omitempty"`
}

// PackageManagerInfo represents package manager information
type PackageManagerInfo struct {
	Name        string   `json:"name"`
	ConfigFiles []string `json:"config_files"`
}

// CompileRequest represents a request to compile proto files
type CompileRequest struct {
	Languages   []string          `json:"languages"`   // List of language IDs to compile for
	IncludeGRPC bool              `json:"include_grpc"`
	Options     map[string]string `json:"options,omitempty"`
}

// CompileResponse represents the response from a compilation request
type CompileResponse struct {
	JobID   string                  `json:"job_id"`
	Results []CompilationJobInfo    `json:"results"`
}

// CompilationJobInfo represents information about a compilation job
type CompilationJobInfo struct {
	ID          string    `json:"id"`
	Language    string    `json:"language"`
	Status      string    `json:"status"` // "pending", "running", "completed", "failed"
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    int64     `json:"duration_ms,omitempty"` // Duration in milliseconds
	CacheHit    bool      `json:"cache_hit"`
	Error       string    `json:"error,omitempty"`
	S3Key       string    `json:"s3_key,omitempty"`
	S3Bucket    string    `json:"s3_bucket,omitempty"`
}

// Storage interface defines the methods required for storing and retrieving protobuf modules
//
// DEPRECATED: Use storage.Storage interface instead.
//
// DEPRECATION TIMELINE:
//   - Deprecated: 2025-01-15 (v1.8.0 release)
//   - Final removal: 2026-01-15 (v2.0.0 release)
//   - Migration period: 12 months
//
// STATUS (as of 2026-01-30):
//   ⚠️  15 days PAST removal deadline
//   ⚠️  This interface should be removed imminently
//   ⚠️  All production code MUST migrate to storage.Storage
//
// After 2026-01-15, this interface will be completely removed.
// Code using api.Storage will not compile. Plan migration before this date.
//
// Migration: Replace api.Storage with storage.Storage and use context-aware methods:
//   - CreateModule → CreateModuleContext(ctx, module)
//   - GetModule → GetModuleContext(ctx, name)
//   - ListModules → ListModulesContext(ctx)
//   - CreateVersion → CreateVersionContext(ctx, version)
//   - GetVersion → GetVersionContext(ctx, moduleName, version)
//   - ListVersions → ListVersionsContext(ctx, moduleName)
//   - UpdateVersion → UpdateVersionContext(ctx, version)
//   - GetFile → GetFileContext(ctx, moduleName, version, path)
//
// See pkg/storage/DEPRECATION.md for detailed migration guide.
//
// TODO(maintainers): Remove this interface after confirming all callers migrated.
// Check remaining usage: git grep -n "api\.Storage\b" | grep -v "Deprecated"
type Storage interface {
	// Module operations
	CreateModule(module *Module) error
	GetModule(name string) (*Module, error)
	ListModules() ([]*Module, error)

	// Version operations
	CreateVersion(version *Version) error
	GetVersion(moduleName, version string) (*Version, error)
	ListVersions(moduleName string) ([]*Version, error)
	UpdateVersion(version *Version) error

	// File operations
	GetFile(moduleName, version, path string) (*File, error)
} 