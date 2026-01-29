package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/platinummonkey/spoke/pkg/codegen"
)

// MemoryCache implements a simple in-memory LRU cache
type MemoryCache struct {
	config  *Config
	cache   *lru.LRU[string, *codegen.CompilationResult]
	metrics *metrics
	mu      sync.RWMutex
}

// NewCache creates a new memory-only cache
func NewCache(config *Config) (*MemoryCache, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Calculate max entries based on max size and estimated average item size
	// Assume average item is ~100KB, so for 100MB cache we'd have ~1000 entries
	maxEntries := int(config.MaxSize / (100 * 1024))
	if maxEntries < 10 {
		maxEntries = 10 // Minimum 10 entries
	}

	// Create LRU cache with TTL support
	cache := lru.NewLRU[string, *codegen.CompilationResult](
		maxEntries,
		nil, // No eviction callback needed
		config.TTL,
	)

	return &MemoryCache{
		config:  config,
		cache:   cache,
		metrics: newMetrics(),
	}, nil
}

// Get retrieves a cached compilation result
func (c *MemoryCache) Get(ctx context.Context, key *codegen.CacheKey) (*codegen.CompilationResult, error) {
	if key == nil {
		return nil, ErrInvalidCacheKey
	}

	keyStr := key.String()

	result, ok := c.cache.Get(keyStr)
	if !ok {
		c.metrics.recordMiss()
		return nil, ErrCacheMiss
	}

	c.metrics.recordHit()
	return result, nil
}

// Set stores a compilation result in cache
func (c *MemoryCache) Set(ctx context.Context, key *codegen.CacheKey, result *codegen.CompilationResult, ttl time.Duration) error {
	if key == nil {
		return ErrInvalidCacheKey
	}
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}

	keyStr := key.String()
	c.cache.Add(keyStr, result)

	return nil
}

// Delete removes a cached result
func (c *MemoryCache) Delete(ctx context.Context, key *codegen.CacheKey) error {
	if key == nil {
		return ErrInvalidCacheKey
	}

	keyStr := key.String()
	c.cache.Remove(keyStr)

	return nil
}

// Invalidate removes all cached results for a module/version
func (c *MemoryCache) Invalidate(ctx context.Context, moduleName, version string) error {
	if moduleName == "" || version == "" {
		return fmt.Errorf("module name and version required")
	}

	// Since we can't pattern-match keys in the LRU cache, we'll just clear everything
	// This is acceptable for an in-memory cache
	c.cache.Purge()

	return nil
}

// Stats returns cache statistics
func (c *MemoryCache) Stats(ctx context.Context) (*Stats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &Stats{
		Hits:      c.metrics.getHits(),
		Misses:    c.metrics.getMisses(),
		ItemCount: int64(c.cache.Len()),
	}

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	return stats, nil
}

// Close releases resources
func (c *MemoryCache) Close() error {
	c.cache.Purge()
	return nil
}

// metrics tracks cache metrics
type metrics struct {
	hits   atomic.Int64
	misses atomic.Int64
}

func newMetrics() *metrics {
	return &metrics{}
}

func (m *metrics) recordHit() {
	m.hits.Add(1)
}

func (m *metrics) recordMiss() {
	m.misses.Add(1)
}

func (m *metrics) getHits() int64 {
	return m.hits.Load()
}

func (m *metrics) getMisses() int64 {
	return m.misses.Load()
}
