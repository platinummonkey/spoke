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
// CacheKey represents a unique identifier for cached compilation results
//
// CRITICAL INVARIANT: Cache keys must be generated with sorted inputs.
// Use cache.GenerateCacheKey() to create keys - never construct manually.
//
// Cache Key Format Version: v1
// String format: {moduleName}:{version}:{language}:{pluginVersion}:{protoHash}:{optionsHash}
//
// WARNING: Changing the String() method or key generation algorithm
// INVALIDATES ALL CACHED COMPILATIONS system-wide.
//
// If you must modify:
// 1. Increment cache format version number
// 2. Clear all cached data or implement migration
// 3. Update cache package documentation
// 4. Test that identical inputs produce identical keys
type CacheKey struct {
	ModuleName    string
	Version       string
	Language      string
	PluginVersion string
	ProtoHash     string            // Combined hash of all proto files + dependencies (generated with sorted inputs)
	Options       map[string]string // Plugin options (hashed with sorted keys)
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
