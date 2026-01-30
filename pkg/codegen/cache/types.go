package cache

import (
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen/config"
)

// Stats represents cache statistics
type Stats struct {
	Hits      int64
	Misses    int64
	HitRate   float64
	ItemCount int64
}

// Config holds cache configuration
type Config struct {
	MaxSize int64         // Max size in bytes (default: 100MB)
	TTL     time.Duration // TTL for cache entries (default: 5 minutes)
}

// DefaultConfig returns default cache configuration
func DefaultConfig() *Config {
	return &Config{
		MaxSize: config.DefaultCacheMaxSize,
		TTL:     config.DefaultCacheTTL,
	}
}
