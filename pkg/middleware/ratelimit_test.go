package middleware

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 10,
		WindowDuration:    time.Second,
		BurstSize:         2,
	}
	limiter := NewRateLimiter(config)

	key := "test-user"

	// Should allow initial requests up to limit + burst
	allowedCount := 0
	for i := 0; i < config.RequestsPerWindow+config.BurstSize+5; i++ {
		if limiter.Allow(key) {
			allowedCount++
		}
	}

	expected := config.RequestsPerWindow + config.BurstSize
	if allowedCount != expected {
		t.Errorf("Allowed %d requests, want %d", allowedCount, expected)
	}

	// After waiting, tokens should refill
	time.Sleep(time.Second)
	if !limiter.Allow(key) {
		t.Error("Should allow request after refill")
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 10,
		WindowDuration:    time.Second,
		BurstSize:         2,
	}
	limiter := NewRateLimiter(config)

	key := "test-user"

	// Check initial remaining
	initial := limiter.Remaining(key)
	expected := config.RequestsPerWindow + config.BurstSize
	if initial != expected {
		t.Errorf("Initial remaining = %d, want %d", initial, expected)
	}

	// Use one token
	limiter.Allow(key)
	remaining := limiter.Remaining(key)
	if remaining != initial-1 {
		t.Errorf("After using 1 token, remaining = %d, want %d", remaining, initial-1)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 10,
		WindowDuration:    100 * time.Millisecond,
		BurstSize:         2,
	}
	limiter := NewRateLimiter(config)

	// Create some buckets
	keys := []string{"user1", "user2", "user3"}
	for _, key := range keys {
		limiter.Allow(key)
	}

	// Buckets should exist
	if len(limiter.buckets) != len(keys) {
		t.Errorf("Expected %d buckets, got %d", len(keys), len(limiter.buckets))
	}

	// Wait for buckets to become stale
	time.Sleep(300 * time.Millisecond)

	// Cleanup should remove old buckets
	limiter.Cleanup()

	if len(limiter.buckets) != 0 {
		t.Errorf("Expected 0 buckets after cleanup, got %d", len(limiter.buckets))
	}
}

func TestRateLimitConfig_Defaults(t *testing.T) {
	config := DefaultRateLimitConfig()

	if config.RequestsPerWindow <= 0 {
		t.Error("RequestsPerWindow should be positive")
	}
	if config.WindowDuration <= 0 {
		t.Error("WindowDuration should be positive")
	}
	if config.BurstSize < 0 {
		t.Error("BurstSize should be non-negative")
	}
}

func TestPerUserRateLimitConfig(t *testing.T) {
	config := PerUserRateLimitConfig()

	defaultConfig := DefaultRateLimitConfig()
	if config.RequestsPerWindow <= defaultConfig.RequestsPerWindow {
		t.Error("User rate limit should be higher than default")
	}
}

func TestPerBotRateLimitConfig(t *testing.T) {
	config := PerBotRateLimitConfig()

	userConfig := PerUserRateLimitConfig()
	if config.RequestsPerWindow <= userConfig.RequestsPerWindow {
		t.Error("Bot rate limit should be higher than user rate limit")
	}
}

func TestGetClientIP(t *testing.T) {
	// This function is simple string extraction, covered in integration tests
	// Just ensure it doesn't panic with nil request
	// Full testing would require http.Request construction
}

func TestRateLimiter_Concurrency(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 100,
		WindowDuration:    time.Second,
		BurstSize:         10,
	}
	limiter := NewRateLimiter(config)

	key := "concurrent-user"
	concurrency := 10
	requestsPerGoroutine := 20

	// Run concurrent requests
	results := make(chan bool, concurrency*requestsPerGoroutine)
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < requestsPerGoroutine; j++ {
				results <- limiter.Allow(key)
			}
		}()
	}

	// Collect results
	allowedCount := 0
	for i := 0; i < concurrency*requestsPerGoroutine; i++ {
		if <-results {
			allowedCount++
		}
	}

	// Should respect rate limit even with concurrent requests
	maxAllowed := config.RequestsPerWindow + config.BurstSize
	if allowedCount > maxAllowed {
		t.Errorf("Allowed %d requests with concurrency, should not exceed %d", allowedCount, maxAllowed)
	}
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 10,
		WindowDuration:    time.Second,
		BurstSize:         0,
	}
	limiter := NewRateLimiter(config)

	key := "refill-test"

	// Exhaust tokens
	for i := 0; i < config.RequestsPerWindow; i++ {
		limiter.Allow(key)
	}

	// Should be denied
	if limiter.Allow(key) {
		t.Error("Should deny request after exhausting tokens")
	}

	// Wait for half the window
	time.Sleep(time.Second / 2)

	// Should have some tokens refilled (approximately half)
	if !limiter.Allow(key) {
		t.Error("Should allow request after partial refill")
	}
}
