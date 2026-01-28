package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/platinummonkey/spoke/pkg/codegen"
)

// MultiLevelCache implements a multi-level cache (L1: Memory, L2: Redis)
type MultiLevelCache struct {
	config  *Config
	l1      *memoryCache
	l2      *redis.Client
	metrics *metrics
	mu      sync.RWMutex
}

// NewCache creates a new multi-level cache
func NewCache(config *Config) (Cache, error) {
	if config == nil {
		config = DefaultConfig()
	}

	c := &MultiLevelCache{
		config:  config,
		metrics: newMetrics(),
	}

	// Initialize L1 (memory) cache
	if config.EnableL1 {
		c.l1 = newMemoryCache(config.L1MaxSize, config.L1TTL)
	}

	// Initialize L2 (Redis) cache
	if config.EnableL2 {
		if config.L2Addr == "" {
			return nil, fmt.Errorf("L2 cache enabled but no Redis address provided")
		}

		c.l2 = redis.NewClient(&redis.Options{
			Addr:     config.L2Addr,
			Password: config.L2Password,
			DB:       config.L2DB,
		})

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.l2.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
	}

	return c, nil
}

// Get retrieves a cached compilation result
func (c *MultiLevelCache) Get(ctx context.Context, key *codegen.CacheKey) (*codegen.CompilationResult, error) {
	if key == nil {
		return nil, ErrInvalidCacheKey
	}

	keyStr := c.buildKey(key)

	// Try L1 (memory) cache first
	if c.config.EnableL1 && c.l1 != nil {
		if result := c.l1.get(keyStr); result != nil {
			c.metrics.recordHit(1) // L1 hit
			return result, nil
		}
	}

	// Try L2 (Redis) cache
	if c.config.EnableL2 && c.l2 != nil {
		data, err := c.l2.Get(ctx, keyStr).Bytes()
		if err == nil {
			var result codegen.CompilationResult
			if err := json.Unmarshal(data, &result); err == nil {
				c.metrics.recordHit(2) // L2 hit

				// Populate L1 cache
				if c.config.EnableL1 && c.l1 != nil {
					c.l1.set(keyStr, &result, c.config.L1TTL)
				}

				return &result, nil
			}
		} else if err != redis.Nil {
			// Redis error (not just missing key)
			return nil, fmt.Errorf("redis error: %w", err)
		}
	}

	// Cache miss
	c.metrics.recordMiss()
	return nil, ErrCacheMiss
}

// Set stores a compilation result in cache
func (c *MultiLevelCache) Set(ctx context.Context, key *codegen.CacheKey, result *codegen.CompilationResult, ttl time.Duration) error {
	if key == nil {
		return ErrInvalidCacheKey
	}
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}

	keyStr := c.buildKey(key)

	// Store in L1 (memory) cache
	if c.config.EnableL1 && c.l1 != nil {
		c.l1.set(keyStr, result, c.config.L1TTL)
	}

	// Store in L2 (Redis) cache
	if c.config.EnableL2 && c.l2 != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}

		if ttl == 0 {
			ttl = c.config.L2TTL
		}

		if err := c.l2.Set(ctx, keyStr, data, ttl).Err(); err != nil {
			return fmt.Errorf("failed to set Redis cache: %w", err)
		}
	}

	return nil
}

// Delete removes a cached result
func (c *MultiLevelCache) Delete(ctx context.Context, key *codegen.CacheKey) error {
	if key == nil {
		return ErrInvalidCacheKey
	}

	keyStr := c.buildKey(key)

	// Delete from L1
	if c.config.EnableL1 && c.l1 != nil {
		c.l1.delete(keyStr)
	}

	// Delete from L2
	if c.config.EnableL2 && c.l2 != nil {
		if err := c.l2.Del(ctx, keyStr).Err(); err != nil {
			return fmt.Errorf("failed to delete from Redis: %w", err)
		}
	}

	return nil
}

// Invalidate removes all cached results for a module/version
func (c *MultiLevelCache) Invalidate(ctx context.Context, moduleName, version string) error {
	if moduleName == "" || version == "" {
		return fmt.Errorf("module name and version required")
	}

	// Pattern: spoke:compiled:{moduleName}:{version}:*
	pattern := fmt.Sprintf("%s%s:%s:*", c.config.L2KeyPrefix, moduleName, version)

	// Invalidate L1 (clear all - no pattern matching in memory cache)
	if c.config.EnableL1 && c.l1 != nil {
		c.l1.clear()
	}

	// Invalidate L2 (Redis - use SCAN + DEL)
	if c.config.EnableL2 && c.l2 != nil {
		iter := c.l2.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			if err := c.l2.Del(ctx, iter.Val()).Err(); err != nil {
				return fmt.Errorf("failed to delete key %s: %w", iter.Val(), err)
			}
		}
		if err := iter.Err(); err != nil {
			return fmt.Errorf("scan error: %w", err)
		}
	}

	return nil
}

// Stats returns cache statistics
func (c *MultiLevelCache) Stats(ctx context.Context) (*Stats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &Stats{
		Hits:   c.metrics.getHits(),
		Misses: c.metrics.getMisses(),
	}

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	// L1 stats
	if c.config.EnableL1 && c.l1 != nil {
		stats.L1Hits = c.metrics.getL1Hits()
		stats.ItemCount = int64(c.l1.size())
		stats.Size = c.l1.currentSize()
		if stats.ItemCount > 0 {
			stats.AvgItemSize = stats.Size / stats.ItemCount
		}
	}

	// L2 stats
	if c.config.EnableL2 && c.l2 != nil {
		stats.L2Hits = c.metrics.getL2Hits()
		// Note: Could add Redis INFO command to get more stats
	}

	return stats, nil
}

// Close releases resources
func (c *MultiLevelCache) Close() error {
	if c.l2 != nil {
		return c.l2.Close()
	}
	return nil
}

// buildKey builds a cache key string from a CacheKey
func (c *MultiLevelCache) buildKey(key *codegen.CacheKey) string {
	return fmt.Sprintf("%s%s", c.config.L2KeyPrefix, key.String())
}

// memoryCache implements a simple in-memory LRU cache
type memoryCache struct {
	maxSize     int64
	ttl         time.Duration
	items       map[string]*cacheItem
	mu          sync.RWMutex
	sizeBytes   int64 // Current size in bytes
}

type cacheItem struct {
	result    *codegen.CompilationResult
	size      int64
	expiresAt time.Time
}

func newMemoryCache(maxSize int64, ttl time.Duration) *memoryCache {
	c := &memoryCache{
		maxSize: maxSize,
		ttl:     ttl,
		items:   make(map[string]*cacheItem),
	}

	// Start cleanup goroutine
	go c.cleanupLoop()

	return c
}

func (c *memoryCache) get(key string) *codegen.CompilationResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil
	}

	// Check expiration
	if time.Now().After(item.expiresAt) {
		return nil
	}

	return item.result
}

func (c *memoryCache) set(key string, result *codegen.CompilationResult, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Calculate size (rough estimate)
	size := c.estimateSize(result)

	// Check if we need to evict items
	for c.sizeBytes+size > c.maxSize && len(c.items) > 0 {
		c.evictOldest()
	}

	// If still too large, don't cache
	if c.sizeBytes+size > c.maxSize {
		return
	}

	c.items[key] = &cacheItem{
		result:    result,
		size:      size,
		expiresAt: time.Now().Add(ttl),
	}
	c.sizeBytes += size
}

func (c *memoryCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		c.sizeBytes -= item.size
		delete(c.items, key)
	}
}

func (c *memoryCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	c.sizeBytes = 0
}

func (c *memoryCache) size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *memoryCache) currentSize() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sizeBytes
}

func (c *memoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestKey == "" || item.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.expiresAt
		}
	}

	if oldestKey != "" {
		c.sizeBytes -= c.items[oldestKey].size
		delete(c.items, oldestKey)
	}
}

func (c *memoryCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *memoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			c.sizeBytes -= item.size
			delete(c.items, key)
		}
	}
}

func (c *memoryCache) estimateSize(result *codegen.CompilationResult) int64 {
	size := int64(0)
	for _, file := range result.GeneratedFiles {
		size += file.Size
	}
	for _, file := range result.PackageFiles {
		size += file.Size
	}
	return size
}

// metrics tracks cache metrics
type metrics struct {
	hits    int64
	misses  int64
	l1Hits  int64
	l2Hits  int64
	l3Hits  int64
	mu      sync.RWMutex
}

func newMetrics() *metrics {
	return &metrics{}
}

func (m *metrics) recordHit(level int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hits++
	switch level {
	case 1:
		m.l1Hits++
	case 2:
		m.l2Hits++
	case 3:
		m.l3Hits++
	}
}

func (m *metrics) recordMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.misses++
}

func (m *metrics) getHits() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hits
}

func (m *metrics) getMisses() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.misses
}

func (m *metrics) getL1Hits() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.l1Hits
}

func (m *metrics) getL2Hits() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.l2Hits
}

func (m *metrics) getL3Hits() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.l3Hits
}
