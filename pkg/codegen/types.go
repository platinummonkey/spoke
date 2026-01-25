package codegen

import (
	"context"
	"time"
)

// CompilationRequest represents a request to compile proto files for a language
type CompilationRequest struct {
	// Module information
	ModuleName    string
	Version       string
	VersionID     int64

	// Proto files and dependencies
	ProtoFiles    []ProtoFile
	Dependencies  []Dependency

	// Language and options
	Language      string
	IncludeGRPC   bool
	Options       map[string]string

	// Context
	Context       context.Context
}

// ProtoFile represents a single proto file
type ProtoFile struct {
	Path     string // Relative path within the module
	Content  []byte
	Hash     string // SHA256 hash for cache key generation
}

// Dependency represents a module dependency
type Dependency struct {
	ModuleName string
	Version    string
	ProtoFiles []ProtoFile
}

// CompilationResult represents the result of a compilation
type CompilationResult struct {
	Success       bool
	Language      string
	GeneratedFiles []GeneratedFile
	PackageFiles  []GeneratedFile // go.mod, setup.py, package.json, etc.
	CacheHit      bool
	Duration      time.Duration
	Error         string

	// Storage information
	S3Key         string
	S3Bucket      string
	ArtifactHash  string
}

// GeneratedFile represents a single generated file
type GeneratedFile struct {
	Path    string // Relative path within output directory
	Content []byte
	Size    int64
}

// CompilationJob represents an async compilation job
type CompilationJob struct {
	ID            string
	VersionID     int64
	Language      string
	Status        JobStatus
	StartedAt     *time.Time
	CompletedAt   *time.Time
	Error         string
	CacheHit      bool
	Result        *CompilationResult
}

// JobStatus represents the status of a compilation job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// CacheKey represents a key for caching compiled artifacts
type CacheKey struct {
	ModuleName    string
	Version       string
	Language      string
	PluginVersion string
	ProtoHash     string // Combined hash of all proto files + dependencies
	Options       map[string]string
}

// String returns the cache key as a string
func (k *CacheKey) String() string {
	optionsHash := ""
	// TODO: Generate stable hash from options map
	return k.ModuleName + ":" + k.Version + ":" + k.Language + ":" + k.PluginVersion + ":" + k.ProtoHash + ":" + optionsHash
}

// CompilationMetrics tracks compilation performance
type CompilationMetrics struct {
	Language      string
	Duration      time.Duration
	CacheHit      bool
	GeneratedSize int64
	Success       bool
}
