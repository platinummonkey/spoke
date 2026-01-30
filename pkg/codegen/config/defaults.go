// Package config provides default configuration values for the codegen system
//
// CENTRALIZED DEFAULTS: All magic constants should be defined here
//
// This file serves as the single source of truth for default values across
// the entire codegen system. Extracting these constants:
// 1. Makes defaults discoverable and easy to adjust
// 2. Documents the rationale for each default value
// 3. Prevents drift when defaults are duplicated across files
// 4. Makes it clear which values are configurable vs hardcoded
package config

import (
	"time"
)

// Cache Configuration Defaults
const (
	// DefaultCacheMaxSize is the maximum size for the in-memory compilation cache
	// Default: 100MB
	//
	// Rationale: Balances memory usage with cache effectiveness. Average compiled
	// artifact is 1-5MB, so 100MB allows caching ~20-100 recent compilations.
	// Increase for higher cache hit rates, decrease for memory-constrained environments.
	DefaultCacheMaxSize = 100 * 1024 * 1024

	// DefaultCacheTTL is the time-to-live for cache entries
	// Default: 5 minutes
	//
	// Rationale: Proto files change infrequently in production but can change rapidly
	// during development. 5 minutes provides a good balance - caching repeated builds
	// during CI runs while ensuring developers see changes after a short delay.
	DefaultCacheTTL = 5 * time.Minute
)

// Orchestrator Configuration Defaults
const (
	// DefaultMaxParallelWorkers is the maximum number of parallel compilations
	// Default: 5
	//
	// Rationale: Balances throughput with resource usage. Each compilation spawns
	// a Docker container (if using Docker runner), consuming CPU and memory.
	// 5 allows reasonable parallelism on modern hardware without overwhelming the system.
	DefaultMaxParallelWorkers = 5

	// DefaultCompilationTimeout is the maximum time allowed for a single compilation
	// Default: 300 seconds (5 minutes)
	//
	// Rationale: Most proto compilations complete in seconds. 5 minutes provides
	// generous headroom for large proto files or slow environments while preventing
	// runaway compilations from blocking the queue indefinitely.
	DefaultCompilationTimeout = 300 * time.Second

	// DefaultCodeGenVersion is the default code generation version
	// Default: "v2"
	//
	// Rationale: v2 is the current stable version. v1 is legacy and maintained
	// only for backward compatibility.
	DefaultCodeGenVersion = "v2"
)

// Docker Execution Defaults
const (
	// DefaultDockerMemoryLimit is the memory limit for Docker containers
	// Default: 512MB
	//
	// Rationale: Protoc is relatively memory-efficient. 512MB is sufficient for
	// most proto compilations including large files. Smaller limits may cause OOM
	// errors on complex schemas.
	DefaultDockerMemoryLimit = int64(512 * 1024 * 1024)

	// DefaultDockerCPULimit is the CPU limit for Docker containers
	// Default: 1.0 (1 CPU core)
	//
	// Rationale: Protoc is CPU-bound but single-threaded. Allocating more than
	// 1 core provides no benefit. Use orchestrator parallelism instead.
	DefaultDockerCPULimit = 1.0

	// DefaultDockerTimeout is the execution timeout for Docker containers
	// Default: 5 minutes
	//
	// Rationale: Matches compilation timeout. Container execution includes image
	// pull time, so this should be at least as long as compilation timeout.
	DefaultDockerTimeout = 5 * time.Minute
)
