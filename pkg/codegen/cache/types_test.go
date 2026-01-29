package cache

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	expectedMaxSize := int64(100 * 1024 * 1024) // 100MB
	if config.MaxSize != expectedMaxSize {
		t.Errorf("Expected MaxSize to be %d, got %d", expectedMaxSize, config.MaxSize)
	}

	expectedTTL := 5 * time.Minute
	if config.TTL != expectedTTL {
		t.Errorf("Expected TTL to be %v, got %v", expectedTTL, config.TTL)
	}
}

func TestStats(t *testing.T) {
	stats := &Stats{
		Hits:      100,
		Misses:    50,
		HitRate:   0.666,
		ItemCount: 10,
	}

	if stats.Hits != 100 {
		t.Errorf("Expected Hits to be 100, got %d", stats.Hits)
	}
	if stats.Misses != 50 {
		t.Errorf("Expected Misses to be 50, got %d", stats.Misses)
	}
	if stats.HitRate != 0.666 {
		t.Errorf("Expected HitRate to be 0.666, got %f", stats.HitRate)
	}
	if stats.ItemCount != 10 {
		t.Errorf("Expected ItemCount to be 10, got %d", stats.ItemCount)
	}
}

func TestConfig(t *testing.T) {
	config := &Config{
		MaxSize: 50 * 1024 * 1024, // 50MB
		TTL:     10 * time.Minute,
	}

	if config.MaxSize != 50*1024*1024 {
		t.Errorf("Expected MaxSize to be %d, got %d", 50*1024*1024, config.MaxSize)
	}
	if config.TTL != 10*time.Minute {
		t.Errorf("Expected TTL to be %v, got %v", 10*time.Minute, config.TTL)
	}
}
