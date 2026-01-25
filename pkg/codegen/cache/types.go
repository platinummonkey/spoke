package cache

import (
	"context"
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen"
)

// Cache provides caching for compiled artifacts
type Cache interface {
	// Get retrieves a cached compilation result
	Get(ctx context.Context, key *codegen.CacheKey) (*codegen.CompilationResult, error)

	// Set stores a compilation result in cache
	Set(ctx context.Context, key *codegen.CacheKey, result *codegen.CompilationResult, ttl time.Duration) error

	// Delete removes a cached result
	Delete(ctx context.Context, key *codegen.CacheKey) error

	// Invalidate removes all cached results for a module/version
	Invalidate(ctx context.Context, moduleName, version string) error

	// Stats returns cache statistics
	Stats(ctx context.Context) (*Stats, error)

	// Close releases resources
	Close() error
}

// Stats represents cache statistics
type Stats struct {
	Hits              int64
	Misses            int64
	HitRate           float64
	Size              int64 // Bytes
	ItemCount         int64
	AvgItemSize       int64
	L1Hits            int64 // Memory cache hits
	L2Hits            int64 // Redis cache hits
	L3Hits            int64 // S3 cache hits
}

// Config holds cache configuration
type Config struct {
	// L1 (Memory) cache
	EnableL1          bool
	L1MaxSize         int64         // Max size in bytes (default: 10MB)
	L1TTL             time.Duration // TTL for L1 cache (default: 5 minutes)

	// L2 (Redis) cache
	EnableL2          bool
	L2Addr            string        // Redis address
	L2Password        string        // Redis password
	L2DB              int           // Redis database
	L2TTL             time.Duration // TTL for L2 cache (default: 24 hours)
	L2KeyPrefix       string        // Key prefix for Redis (default: "spoke:compiled:")

	// L3 (S3) cache
	EnableL3          bool
	L3Bucket          string        // S3 bucket name
	L3Prefix          string        // S3 key prefix

	// General
	EnableMetrics     bool
}

// DefaultConfig returns default cache configuration
func DefaultConfig() *Config {
	return &Config{
		EnableL1:      true,
		L1MaxSize:     10 * 1024 * 1024, // 10MB
		L1TTL:         5 * time.Minute,

		EnableL2:      true,
		L2TTL:         24 * time.Hour,
		L2KeyPrefix:   "spoke:compiled:",

		EnableL3:      true,

		EnableMetrics: true,
	}
}
