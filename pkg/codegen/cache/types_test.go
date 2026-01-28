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

	// Test L1 defaults
	if !config.EnableL1 {
		t.Error("Expected EnableL1 to be true")
	}
	expectedL1MaxSize := int64(10 * 1024 * 1024) // 10MB
	if config.L1MaxSize != expectedL1MaxSize {
		t.Errorf("Expected L1MaxSize to be %d, got %d", expectedL1MaxSize, config.L1MaxSize)
	}
	expectedL1TTL := 5 * time.Minute
	if config.L1TTL != expectedL1TTL {
		t.Errorf("Expected L1TTL to be %v, got %v", expectedL1TTL, config.L1TTL)
	}

	// Test L2 defaults
	if !config.EnableL2 {
		t.Error("Expected EnableL2 to be true")
	}
	expectedL2TTL := 24 * time.Hour
	if config.L2TTL != expectedL2TTL {
		t.Errorf("Expected L2TTL to be %v, got %v", expectedL2TTL, config.L2TTL)
	}
	expectedL2KeyPrefix := "spoke:compiled:"
	if config.L2KeyPrefix != expectedL2KeyPrefix {
		t.Errorf("Expected L2KeyPrefix to be %q, got %q", expectedL2KeyPrefix, config.L2KeyPrefix)
	}

	// Test L3 defaults
	if !config.EnableL3 {
		t.Error("Expected EnableL3 to be true")
	}

	// Test metrics default
	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}
}

func TestStats(t *testing.T) {
	stats := &Stats{
		Hits:        100,
		Misses:      50,
		HitRate:     0.666,
		Size:        1024 * 1024,
		ItemCount:   10,
		AvgItemSize: 1024 * 1024 / 10,
		L1Hits:      60,
		L2Hits:      30,
		L3Hits:      10,
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
	if stats.Size != 1024*1024 {
		t.Errorf("Expected Size to be %d, got %d", 1024*1024, stats.Size)
	}
	if stats.ItemCount != 10 {
		t.Errorf("Expected ItemCount to be 10, got %d", stats.ItemCount)
	}
	expectedAvgSize := int64(1024 * 1024 / 10)
	if stats.AvgItemSize != expectedAvgSize {
		t.Errorf("Expected AvgItemSize to be %d, got %d", expectedAvgSize, stats.AvgItemSize)
	}
	if stats.L1Hits != 60 {
		t.Errorf("Expected L1Hits to be 60, got %d", stats.L1Hits)
	}
	if stats.L2Hits != 30 {
		t.Errorf("Expected L2Hits to be 30, got %d", stats.L2Hits)
	}
	if stats.L3Hits != 10 {
		t.Errorf("Expected L3Hits to be 10, got %d", stats.L3Hits)
	}
}

func TestConfig(t *testing.T) {
	config := &Config{
		EnableL1:      false,
		L1MaxSize:     5 * 1024 * 1024, // 5MB
		L1TTL:         10 * time.Minute,
		EnableL2:      false,
		L2Addr:        "localhost:6379",
		L2Password:    "password",
		L2DB:          1,
		L2TTL:         12 * time.Hour,
		L2KeyPrefix:   "test:",
		EnableL3:      false,
		L3Bucket:      "test-bucket",
		L3Prefix:      "test-prefix/",
		EnableMetrics: false,
	}

	if config.EnableL1 {
		t.Error("Expected EnableL1 to be false")
	}
	if config.L1MaxSize != 5*1024*1024 {
		t.Errorf("Expected L1MaxSize to be %d, got %d", 5*1024*1024, config.L1MaxSize)
	}
	if config.L1TTL != 10*time.Minute {
		t.Errorf("Expected L1TTL to be %v, got %v", 10*time.Minute, config.L1TTL)
	}
	if config.EnableL2 {
		t.Error("Expected EnableL2 to be false")
	}
	if config.L2Addr != "localhost:6379" {
		t.Errorf("Expected L2Addr to be %q, got %q", "localhost:6379", config.L2Addr)
	}
	if config.L2Password != "password" {
		t.Errorf("Expected L2Password to be %q, got %q", "password", config.L2Password)
	}
	if config.L2DB != 1 {
		t.Errorf("Expected L2DB to be 1, got %d", config.L2DB)
	}
	if config.L2TTL != 12*time.Hour {
		t.Errorf("Expected L2TTL to be %v, got %v", 12*time.Hour, config.L2TTL)
	}
	if config.L2KeyPrefix != "test:" {
		t.Errorf("Expected L2KeyPrefix to be %q, got %q", "test:", config.L2KeyPrefix)
	}
	if config.EnableL3 {
		t.Error("Expected EnableL3 to be false")
	}
	if config.L3Bucket != "test-bucket" {
		t.Errorf("Expected L3Bucket to be %q, got %q", "test-bucket", config.L3Bucket)
	}
	if config.L3Prefix != "test-prefix/" {
		t.Errorf("Expected L3Prefix to be %q, got %q", "test-prefix/", config.L3Prefix)
	}
	if config.EnableMetrics {
		t.Error("Expected EnableMetrics to be false")
	}
}
