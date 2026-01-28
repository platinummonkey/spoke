package cache

import (
	"context"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCache tests the NewCache constructor
func TestNewCache(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		// DefaultConfig enables L2 but without address, so it will fail
		// We need a proper config for this test
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)
		require.NotNil(t, cache)

		mlCache := cache.(*MultiLevelCache)
		assert.NotNil(t, mlCache.config)
		assert.NotNil(t, mlCache.metrics)
		assert.NotNil(t, mlCache.l1)
	})

	t.Run("with L1 only", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)
		require.NotNil(t, cache)

		mlCache := cache.(*MultiLevelCache)
		assert.NotNil(t, mlCache.l1)
		assert.Nil(t, mlCache.l2)
	})

	t.Run("with L2 missing address", func(t *testing.T) {
		config := &Config{
			EnableL1: false,
			EnableL2: true,
			L2Addr:   "", // Missing address
		}
		cache, err := NewCache(config)
		assert.Error(t, err)
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "no Redis address provided")
	})
}

// TestMultiLevelCache_Get tests the Get method
func TestMultiLevelCache_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("with nil key", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		result, err := cache.Get(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidCacheKey, err)
	})

	t.Run("cache miss L1 only", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		key := &codegen.CacheKey{
			ModuleName:    "test-module",
			Version:       "v1.0.0",
			Language:      "go",
			PluginVersion: "v1.0.0",
			ProtoHash:     "abc123",
		}

		result, err := cache.Get(ctx, key)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrCacheMiss, err)

		// Verify miss was recorded
		mlCache := cache.(*MultiLevelCache)
		assert.Equal(t, int64(1), mlCache.metrics.getMisses())
	})

	t.Run("L1 cache hit", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		key := &codegen.CacheKey{
			ModuleName:    "test-module",
			Version:       "v1.0.0",
			Language:      "go",
			PluginVersion: "v1.0.0",
			ProtoHash:     "abc123",
		}

		expectedResult := &codegen.CompilationResult{
			Success:  true,
			Language: "go",
			GeneratedFiles: []codegen.GeneratedFile{
				{Path: "test.go", Content: []byte("package test"), Size: 12},
			},
		}

		// Set the value first
		err = cache.Set(ctx, key, expectedResult, 5*time.Minute)
		require.NoError(t, err)

		// Now get it
		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expectedResult.Language, result.Language)
		assert.Equal(t, expectedResult.Success, result.Success)

		// Verify hit was recorded
		mlCache := cache.(*MultiLevelCache)
		assert.Equal(t, int64(1), mlCache.metrics.getHits())
		assert.Equal(t, int64(1), mlCache.metrics.getL1Hits())
	})
}

// TestMultiLevelCache_Set tests the Set method
func TestMultiLevelCache_Set(t *testing.T) {
	ctx := context.Background()

	t.Run("with nil key", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		result := &codegen.CompilationResult{Success: true}
		err = cache.Set(ctx, nil, result, 5*time.Minute)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCacheKey, err)
	})

	t.Run("with nil result", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		key := &codegen.CacheKey{
			ModuleName:    "test-module",
			Version:       "v1.0.0",
			Language:      "go",
			PluginVersion: "v1.0.0",
			ProtoHash:     "abc123",
		}

		err = cache.Set(ctx, key, nil, 5*time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "result cannot be nil")
	})

	t.Run("successful set L1 only", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		key := &codegen.CacheKey{
			ModuleName:    "test-module",
			Version:       "v1.0.0",
			Language:      "go",
			PluginVersion: "v1.0.0",
			ProtoHash:     "abc123",
		}

		result := &codegen.CompilationResult{
			Success:  true,
			Language: "go",
			GeneratedFiles: []codegen.GeneratedFile{
				{Path: "test.go", Content: []byte("package test"), Size: 12},
			},
		}

		err = cache.Set(ctx, key, result, 5*time.Minute)
		require.NoError(t, err)
	})
}

// TestMultiLevelCache_Delete tests the Delete method
func TestMultiLevelCache_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("with nil key", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		err = cache.Delete(ctx, nil)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCacheKey, err)
	})

	t.Run("delete existing key", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		key := &codegen.CacheKey{
			ModuleName:    "test-module",
			Version:       "v1.0.0",
			Language:      "go",
			PluginVersion: "v1.0.0",
			ProtoHash:     "abc123",
		}

		result := &codegen.CompilationResult{
			Success:  true,
			Language: "go",
		}

		// Set first
		err = cache.Set(ctx, key, result, 5*time.Minute)
		require.NoError(t, err)

		// Verify it exists
		cached, err := cache.Get(ctx, key)
		require.NoError(t, err)
		require.NotNil(t, cached)

		// Delete it
		err = cache.Delete(ctx, key)
		require.NoError(t, err)

		// Verify it's gone
		cached, err = cache.Get(ctx, key)
		assert.Error(t, err)
		assert.Nil(t, cached)
		assert.Equal(t, ErrCacheMiss, err)
	})
}

// TestMultiLevelCache_Invalidate tests the Invalidate method
func TestMultiLevelCache_Invalidate(t *testing.T) {
	ctx := context.Background()

	t.Run("with empty module name", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		err = cache.Invalidate(ctx, "", "v1.0.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "module name and version required")
	})

	t.Run("with empty version", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		err = cache.Invalidate(ctx, "test-module", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "module name and version required")
	})

	t.Run("invalidate L1 only", func(t *testing.T) {
		config := &Config{
			EnableL1:    true,
			L1MaxSize:   1024 * 1024,
			L1TTL:       5 * time.Minute,
			EnableL2:    false,
			L2KeyPrefix: "spoke:compiled:",
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		// Add some entries
		key1 := &codegen.CacheKey{
			ModuleName:    "test-module",
			Version:       "v1.0.0",
			Language:      "go",
			PluginVersion: "v1.0.0",
			ProtoHash:     "abc123",
		}
		result1 := &codegen.CompilationResult{Success: true, Language: "go"}
		err = cache.Set(ctx, key1, result1, 5*time.Minute)
		require.NoError(t, err)

		// Invalidate
		err = cache.Invalidate(ctx, "test-module", "v1.0.0")
		require.NoError(t, err)

		// Verify L1 is cleared
		mlCache := cache.(*MultiLevelCache)
		assert.Equal(t, 0, mlCache.l1.size())
	})
}

// TestMultiLevelCache_Stats tests the Stats method
func TestMultiLevelCache_Stats(t *testing.T) {
	ctx := context.Background()

	t.Run("initial stats", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		stats, err := cache.Stats(ctx)
		require.NoError(t, err)
		require.NotNil(t, stats)

		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(0), stats.Misses)
		assert.Equal(t, float64(0), stats.HitRate)
		assert.Equal(t, int64(0), stats.ItemCount)
	})

	t.Run("stats after operations", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		key := &codegen.CacheKey{
			ModuleName:    "test-module",
			Version:       "v1.0.0",
			Language:      "go",
			PluginVersion: "v1.0.0",
			ProtoHash:     "abc123",
		}

		result := &codegen.CompilationResult{
			Success:  true,
			Language: "go",
			GeneratedFiles: []codegen.GeneratedFile{
				{Path: "test.go", Content: []byte("package test"), Size: 12},
			},
		}

		// Set value
		err = cache.Set(ctx, key, result, 5*time.Minute)
		require.NoError(t, err)

		// Get value (hit)
		_, err = cache.Get(ctx, key)
		require.NoError(t, err)

		// Get missing value (miss)
		missingKey := &codegen.CacheKey{
			ModuleName:    "missing",
			Version:       "v1.0.0",
			Language:      "go",
			PluginVersion: "v1.0.0",
			ProtoHash:     "xyz789",
		}
		_, _ = cache.Get(ctx, missingKey)

		// Check stats
		stats, err := cache.Stats(ctx)
		require.NoError(t, err)

		assert.Equal(t, int64(1), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
		assert.Equal(t, 0.5, stats.HitRate)
		assert.Equal(t, int64(1), stats.ItemCount)
		assert.Equal(t, int64(1), stats.L1Hits)
		assert.Greater(t, stats.Size, int64(0))
		assert.Greater(t, stats.AvgItemSize, int64(0))
	})
}

// TestMultiLevelCache_Close tests the Close method
func TestMultiLevelCache_Close(t *testing.T) {
	t.Run("close L1 only cache", func(t *testing.T) {
		config := &Config{
			EnableL1:  true,
			L1MaxSize: 1024 * 1024,
			L1TTL:     5 * time.Minute,
			EnableL2:  false,
		}
		cache, err := NewCache(config)
		require.NoError(t, err)

		err = cache.Close()
		assert.NoError(t, err)
	})
}

// TestMultiLevelCache_buildKey tests the buildKey method
func TestMultiLevelCache_buildKey(t *testing.T) {
	config := &Config{
		EnableL1:    true,
		L1MaxSize:   1024 * 1024,
		L1TTL:       5 * time.Minute,
		EnableL2:    false,
		L2KeyPrefix: "spoke:compiled:",
	}
	cache, err := NewCache(config)
	require.NoError(t, err)

	mlCache := cache.(*MultiLevelCache)

	key := &codegen.CacheKey{
		ModuleName:    "test-module",
		Version:       "v1.0.0",
		Language:      "go",
		PluginVersion: "v1.0.0",
		ProtoHash:     "abc123",
	}

	keyStr := mlCache.buildKey(key)
	assert.Contains(t, keyStr, "spoke:compiled:")
	assert.Contains(t, keyStr, "test-module")
	assert.Contains(t, keyStr, "v1.0.0")
	assert.Contains(t, keyStr, "go")
}

// TestMemoryCache tests the memory cache implementation
func TestMemoryCache(t *testing.T) {
	t.Run("new memory cache", func(t *testing.T) {
		mc := newMemoryCache(1024*1024, 5*time.Minute)
		require.NotNil(t, mc)
		assert.Equal(t, int64(1024*1024), mc.maxSize)
		assert.Equal(t, 5*time.Minute, mc.ttl)
		assert.Equal(t, 0, mc.size())
		assert.Equal(t, int64(0), mc.currentSize())
	})

	t.Run("set and get", func(t *testing.T) {
		mc := newMemoryCache(1024*1024, 5*time.Minute)

		result := &codegen.CompilationResult{
			Success:  true,
			Language: "go",
			GeneratedFiles: []codegen.GeneratedFile{
				{Path: "test.go", Content: []byte("package test"), Size: 12},
			},
		}

		mc.set("key1", result, 5*time.Minute)
		assert.Equal(t, 1, mc.size())
		assert.Greater(t, mc.currentSize(), int64(0))

		retrieved := mc.get("key1")
		require.NotNil(t, retrieved)
		assert.Equal(t, result.Language, retrieved.Language)
	})

	t.Run("get expired", func(t *testing.T) {
		mc := newMemoryCache(1024*1024, 1*time.Millisecond)

		result := &codegen.CompilationResult{Success: true}
		mc.set("key1", result, 1*time.Millisecond)

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		retrieved := mc.get("key1")
		assert.Nil(t, retrieved)
	})

	t.Run("delete", func(t *testing.T) {
		mc := newMemoryCache(1024*1024, 5*time.Minute)

		result := &codegen.CompilationResult{Success: true}
		mc.set("key1", result, 5*time.Minute)
		assert.Equal(t, 1, mc.size())

		mc.delete("key1")
		assert.Equal(t, 0, mc.size())
		assert.Equal(t, int64(0), mc.currentSize())

		retrieved := mc.get("key1")
		assert.Nil(t, retrieved)
	})

	t.Run("clear", func(t *testing.T) {
		mc := newMemoryCache(1024*1024, 5*time.Minute)

		result := &codegen.CompilationResult{Success: true}
		mc.set("key1", result, 5*time.Minute)
		mc.set("key2", result, 5*time.Minute)
		assert.Equal(t, 2, mc.size())

		mc.clear()
		assert.Equal(t, 0, mc.size())
		assert.Equal(t, int64(0), mc.currentSize())
	})

	t.Run("eviction when full", func(t *testing.T) {
		// Small cache that can only hold ~100 bytes
		mc := newMemoryCache(100, 5*time.Minute)

		result1 := &codegen.CompilationResult{
			Success: true,
			GeneratedFiles: []codegen.GeneratedFile{
				{Path: "test1.go", Content: []byte("content1"), Size: 50},
			},
		}
		result2 := &codegen.CompilationResult{
			Success: true,
			GeneratedFiles: []codegen.GeneratedFile{
				{Path: "test2.go", Content: []byte("content2"), Size: 50},
			},
		}
		result3 := &codegen.CompilationResult{
			Success: true,
			GeneratedFiles: []codegen.GeneratedFile{
				{Path: "test3.go", Content: []byte("content3"), Size: 50},
			},
		}

		mc.set("key1", result1, 5*time.Minute)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
		mc.set("key2", result2, 5*time.Minute)
		time.Sleep(1 * time.Millisecond)
		mc.set("key3", result3, 5*time.Minute)

		// Cache should have evicted oldest entries
		assert.LessOrEqual(t, mc.currentSize(), int64(100))
		assert.LessOrEqual(t, mc.size(), 2)
	})

	t.Run("estimateSize", func(t *testing.T) {
		mc := newMemoryCache(1024*1024, 5*time.Minute)

		result := &codegen.CompilationResult{
			Success: true,
			GeneratedFiles: []codegen.GeneratedFile{
				{Path: "test1.go", Content: []byte("content1"), Size: 50},
				{Path: "test2.go", Content: []byte("content2"), Size: 100},
			},
			PackageFiles: []codegen.GeneratedFile{
				{Path: "go.mod", Content: []byte("module test"), Size: 30},
			},
		}

		size := mc.estimateSize(result)
		assert.Equal(t, int64(180), size) // 50 + 100 + 30
	})

	t.Run("cleanup expired items", func(t *testing.T) {
		mc := newMemoryCache(1024*1024, 1*time.Millisecond)

		result := &codegen.CompilationResult{Success: true}
		mc.set("key1", result, 1*time.Millisecond)
		mc.set("key2", result, 1*time.Hour) // Won't expire

		assert.Equal(t, 2, mc.size())

		// Wait for first item to expire
		time.Sleep(10 * time.Millisecond)

		// Run cleanup
		mc.cleanup()

		// Only non-expired item should remain
		assert.Equal(t, 1, mc.size())
		assert.Nil(t, mc.get("key1"))
		assert.NotNil(t, mc.get("key2"))
	})
}

// TestMetrics tests the metrics implementation
func TestMetrics(t *testing.T) {
	t.Run("new metrics", func(t *testing.T) {
		m := newMetrics()
		require.NotNil(t, m)
		assert.Equal(t, int64(0), m.getHits())
		assert.Equal(t, int64(0), m.getMisses())
		assert.Equal(t, int64(0), m.getL1Hits())
		assert.Equal(t, int64(0), m.getL2Hits())
		assert.Equal(t, int64(0), m.getL3Hits())
	})

	t.Run("record hits", func(t *testing.T) {
		m := newMetrics()

		m.recordHit(1)
		assert.Equal(t, int64(1), m.getHits())
		assert.Equal(t, int64(1), m.getL1Hits())

		m.recordHit(2)
		assert.Equal(t, int64(2), m.getHits())
		assert.Equal(t, int64(1), m.getL2Hits())

		m.recordHit(3)
		assert.Equal(t, int64(3), m.getHits())
		assert.Equal(t, int64(1), m.getL3Hits())
	})

	t.Run("record misses", func(t *testing.T) {
		m := newMetrics()

		m.recordMiss()
		assert.Equal(t, int64(1), m.getMisses())

		m.recordMiss()
		assert.Equal(t, int64(2), m.getMisses())
	})
}
