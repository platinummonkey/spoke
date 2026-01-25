package orchestrator

import (
	"context"

	"github.com/platinummonkey/spoke/pkg/codegen"
)

// Orchestrator coordinates the compilation process
type Orchestrator interface {
	// CompileSingle compiles proto files for a single language
	CompileSingle(ctx context.Context, req *CompileRequest) (*codegen.CompilationResult, error)

	// CompileAll compiles proto files for multiple languages in parallel
	CompileAll(ctx context.Context, req *CompileRequest, languages []string) ([]*codegen.CompilationResult, error)

	// GetStatus returns the status of a compilation job
	GetStatus(ctx context.Context, jobID string) (*codegen.CompilationJob, error)

	// Close releases resources
	Close() error
}

// CompileRequest represents a compilation request
type CompileRequest struct {
	// Module information
	ModuleName    string
	Version       string
	VersionID     int64

	// Proto files (already fetched from storage)
	ProtoFiles    []codegen.ProtoFile

	// Dependencies (already resolved)
	Dependencies  []codegen.Dependency

	// Language (for single language compilation)
	Language      string

	// Compilation options
	IncludeGRPC   bool
	Options       map[string]string

	// Storage configuration
	StorageDir    string // Local storage directory
	S3Bucket      string // S3 bucket for artifacts
}

// Config holds orchestrator configuration
type Config struct {
	// Parallel execution
	MaxParallelWorkers int // Maximum number of parallel compilations (default: 5)

	// Feature flags
	EnableCache        bool
	EnableMetrics      bool
	CodeGenVersion     string // "v1" or "v2"

	// Storage
	StorageDir         string
	S3Bucket           string
	S3Prefix           string
	S3Region           string

	// Cache configuration
	RedisAddr          string // Redis address for L2 cache
	RedisPassword      string
	RedisDB            int

	// Timeouts
	CompilationTimeout int // Seconds (default: 300)
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxParallelWorkers: 5,
		EnableCache:        true,
		EnableMetrics:      true,
		CodeGenVersion:     "v2",
		CompilationTimeout: 300,
	}
}
