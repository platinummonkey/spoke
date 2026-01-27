package storage

import (
	"context"
	"io"
	"time"

	"github.com/platinummonkey/spoke/pkg/api"
)

// ModuleReader defines read operations for protobuf modules
type ModuleReader interface {
	GetModuleContext(ctx context.Context, name string) (*api.Module, error)
	ListModulesContext(ctx context.Context) ([]*api.Module, error)
	ListModulesPaginated(ctx context.Context, limit, offset int) ([]*api.Module, int64, error)
}

// ModuleWriter defines write operations for protobuf modules
type ModuleWriter interface {
	CreateModuleContext(ctx context.Context, module *api.Module) error
}

// VersionReader defines read operations for module versions
type VersionReader interface {
	GetVersionContext(ctx context.Context, moduleName, version string) (*api.Version, error)
	ListVersionsContext(ctx context.Context, moduleName string) ([]*api.Version, error)
	ListVersionsPaginated(ctx context.Context, moduleName string, limit, offset int) ([]*api.Version, int64, error)
	GetFileContext(ctx context.Context, moduleName, version, path string) (*api.File, error)
}

// VersionWriter defines write operations for module versions
type VersionWriter interface {
	CreateVersionContext(ctx context.Context, version *api.Version) error
	UpdateVersionContext(ctx context.Context, version *api.Version) error
}

// FileStorage defines content-addressed file storage operations
type FileStorage interface {
	GetFileContent(ctx context.Context, hash string) (io.ReadCloser, error)
	PutFileContent(ctx context.Context, content io.Reader, contentType string) (hash string, err error)
}

// ArtifactStorage defines compiled artifact storage operations
type ArtifactStorage interface {
	GetCompiledArtifact(ctx context.Context, moduleName, version, language string) (io.ReadCloser, error)
	PutCompiledArtifact(ctx context.Context, moduleName, version, language string, artifact io.Reader) error
}

// CacheManager defines cache invalidation operations
type CacheManager interface {
	InvalidateCache(ctx context.Context, patterns ...string) error
}

// HealthChecker defines health check operations
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// Storage is the canonical storage interface that combines all storage capabilities.
// It provides context-aware, composable operations for protobuf schema registry storage.
//
// This interface supersedes api.Storage and StorageV2, providing a unified API
// with proper context propagation, interface segregation, and modern Go practices.
type Storage interface {
	ModuleReader
	ModuleWriter
	VersionReader
	VersionWriter
	FileStorage
	ArtifactStorage
	CacheManager
	HealthChecker
}

// StorageV2 extends the base Storage interface with modern features
//
// DEPRECATED: Use Storage interface instead.
// StorageV2 will be removed in v2.0.0 (superseded by Storage).
type StorageV2 interface {
	api.Storage // Embed existing interface for backward compatibility

	// Context-aware operations
	CreateModuleContext(ctx context.Context, module *api.Module) error
	GetModuleContext(ctx context.Context, name string) (*api.Module, error)
	ListModulesContext(ctx context.Context) ([]*api.Module, error)

	// Batch operations with pagination
	ListModulesPaginated(ctx context.Context, limit, offset int) ([]*api.Module, int64, error)
	ListVersionsPaginated(ctx context.Context, moduleName string, limit, offset int) ([]*api.Version, int64, error)

	// Object storage operations
	GetFileContent(ctx context.Context, hash string) (io.ReadCloser, error)
	PutFileContent(ctx context.Context, content io.Reader, contentType string) (hash string, err error)

	// Compiled artifacts
	GetCompiledArtifact(ctx context.Context, moduleName, version, language string) (io.ReadCloser, error)
	PutCompiledArtifact(ctx context.Context, moduleName, version, language string, artifact io.Reader) error

	// Cache operations
	InvalidateCache(ctx context.Context, patterns ...string) error

	// Health checks
	HealthCheck(ctx context.Context) error
}

// Config for storage backend
type Config struct {
	Type string // "filesystem", "postgres", "hybrid"

	// Filesystem config
	FilesystemRoot string

	// PostgreSQL config
	PostgresURL         string
	PostgresReplicaURLs string // Comma-separated list of replica URLs
	PostgresMaxConns    int
	PostgresMinConns    int
	PostgresTimeout     time.Duration

	// S3 config
	S3Endpoint       string
	S3Region         string
	S3Bucket         string
	S3AccessKey      string
	S3SecretKey      string
	S3UsePathStyle   bool
	S3ForcePathStyle bool

	// Redis config
	RedisURL        string
	RedisPassword   string
	RedisDB         int
	RedisMaxRetries int
	RedisPoolSize   int

	// Cache config
	CacheEnabled bool
	CacheTTL     map[string]time.Duration
	L1CacheSize  int64 // Bytes
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() Config {
	return Config{
		Type:             "filesystem",
		FilesystemRoot:   "/tmp/spoke",
		PostgresMaxConns: 20,
		PostgresMinConns: 2,
		PostgresTimeout:  10 * time.Second,
		RedisDB:          0,
		RedisMaxRetries:  3,
		RedisPoolSize:    10,
		CacheEnabled:     true,
		CacheTTL: map[string]time.Duration{
			"module":          1 * time.Hour,
			"version":         1 * time.Hour,
			"version_full":    30 * time.Minute,
			"version_list":    5 * time.Minute,
			"latest":          1 * time.Minute,
			"compiled":        24 * time.Hour,
			"proto_content":   24 * time.Hour,
			"dependency_tree": 1 * time.Hour,
		},
		L1CacheSize: 10 * 1024 * 1024, // 10MB
	}
}
