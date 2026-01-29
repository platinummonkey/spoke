package cache

import (
	"context"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen"
)

func TestMemoryCache_GetSet(t *testing.T) {
	cache, err := NewCache(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Create a test cache key
	key := &codegen.CacheKey{
		ModuleName:    "test-module",
		Version:       "v1.0.0",
		Language:      "go",
		PluginVersion: "v1.0.0",
		ProtoHash:     "abc123",
		Options:       map[string]string{"opt1": "val1"},
	}

	// Create a test result
	result := &codegen.CompilationResult{
		Success:  true,
		Language: "go",
		GeneratedFiles: []codegen.GeneratedFile{
			{
				Path:    "test.go",
				Content: []byte("package test"),
				Size:    12,
			},
		},
	}

	// Test cache miss
	_, err = cache.Get(ctx, key)
	if err != ErrCacheMiss {
		t.Errorf("Expected cache miss, got: %v", err)
	}

	// Test set
	err = cache.Set(ctx, key, result, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Test cache hit
	cached, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get from cache: %v", err)
	}

	if cached.Language != result.Language {
		t.Errorf("Expected language %s, got %s", result.Language, cached.Language)
	}

	if len(cached.GeneratedFiles) != len(result.GeneratedFiles) {
		t.Errorf("Expected %d files, got %d", len(result.GeneratedFiles), len(cached.GeneratedFiles))
	}

	// Test delete
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete from cache: %v", err)
	}

	// Verify deletion
	_, err = cache.Get(ctx, key)
	if err != ErrCacheMiss {
		t.Errorf("Expected cache miss after delete, got: %v", err)
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	cache, err := NewCache(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Get initial stats
	stats, err := cache.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("Expected 0 hits and misses, got %d hits and %d misses", stats.Hits, stats.Misses)
	}

	// Create a test cache key and result
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

	// Cause a miss
	_, _ = cache.Get(ctx, key)

	// Set and hit
	_ = cache.Set(ctx, key, result, 5*time.Minute)
	_, _ = cache.Get(ctx, key)

	// Check stats
	stats, err = cache.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	if stats.HitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", stats.HitRate)
	}

	if stats.ItemCount != 1 {
		t.Errorf("Expected 1 item, got %d", stats.ItemCount)
	}
}

func TestMemoryCache_Invalidate(t *testing.T) {
	cache, err := NewCache(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Add multiple items
	for i := 0; i < 5; i++ {
		key := &codegen.CacheKey{
			ModuleName:    "test-module",
			Version:       "v1.0.0",
			Language:      "go",
			PluginVersion: "v1.0.0",
			ProtoHash:     string(rune('a' + i)),
		}

		result := &codegen.CompilationResult{
			Success:  true,
			Language: "go",
		}

		err = cache.Set(ctx, key, result, 5*time.Minute)
		if err != nil {
			t.Fatalf("Failed to set cache: %v", err)
		}
	}

	// Verify items are in cache
	stats, _ := cache.Stats(ctx)
	if stats.ItemCount != 5 {
		t.Errorf("Expected 5 items, got %d", stats.ItemCount)
	}

	// Invalidate
	err = cache.Invalidate(ctx, "test-module", "v1.0.0")
	if err != nil {
		t.Fatalf("Failed to invalidate cache: %v", err)
	}

	// Verify cache is empty
	stats, _ = cache.Stats(ctx)
	if stats.ItemCount != 0 {
		t.Errorf("Expected 0 items after invalidate, got %d", stats.ItemCount)
	}
}
