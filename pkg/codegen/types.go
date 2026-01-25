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
	// Import cache package for proper key generation
	// For now, use simple concatenation
	// Full implementation in cache.FormatCacheKey()
	parts := []string{
		k.ModuleName,
		k.Version,
		k.Language,
		k.PluginVersion,
		k.ProtoHash,
	}

	// Add options hash if present (simple approach)
	if len(k.Options) > 0 {
		// Simple hash for options - full implementation in cache package
		optStr := ""
		for key, val := range k.Options {
			optStr += key + "=" + val + ";"
		}
		parts = append(parts, optStr[:min(16, len(optStr))])
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ":" + parts[i]
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CompilationMetrics tracks compilation performance
type CompilationMetrics struct {
	Language      string
	Duration      time.Duration
	CacheHit      bool
	GeneratedSize int64
	Success       bool
}
