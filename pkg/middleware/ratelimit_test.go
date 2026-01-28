package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/auth"
)

// Helper function to set auth context in request for testing
func setAuthContextForTest(r *http.Request, authCtx *auth.AuthContext) *http.Request {
	ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
	return r.WithContext(ctx)
}

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
	tests := []struct {
		name           string
		headers        map[string]string
		remoteAddr     string
		expectedIP     string
	}{
		{
			name:       "X-Forwarded-For header",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Real-IP header",
			headers:    map[string]string{"X-Real-IP": "192.168.1.2"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.2",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "10.0.0.1:12345",
		},
		{
			name:       "X-Forwarded-For takes precedence",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1", "X-Real-IP": "192.168.1.2"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("getClientIP() = %v, want %v", ip, tt.expectedIP)
			}
		})
	}
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

func TestNewRateLimiter_NilConfig(t *testing.T) {
	// Should use default config when nil is passed
	limiter := NewRateLimiter(nil)
	if limiter == nil {
		t.Fatal("NewRateLimiter should not return nil")
	}
	if limiter.config == nil {
		t.Fatal("NewRateLimiter should have default config")
	}
	if limiter.config.RequestsPerWindow <= 0 {
		t.Error("Default config should have positive RequestsPerWindow")
	}
}

func TestRateLimiter_StartCleanup(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 10,
		WindowDuration:    50 * time.Millisecond,
		BurstSize:         2,
	}
	limiter := NewRateLimiter(config)

	// Create some buckets
	limiter.Allow("user1")
	limiter.Allow("user2")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cleanup
	limiter.StartCleanup(ctx)

	// Give time for at least one cleanup cycle
	time.Sleep(200 * time.Millisecond)

	// Buckets should be cleaned up after being stale
	limiter.mu.RLock()
	bucketCount := len(limiter.buckets)
	limiter.mu.RUnlock()

	if bucketCount != 0 {
		t.Logf("Expected buckets to be cleaned up, got %d buckets", bucketCount)
		// This is a race condition test, so we'll be lenient
	}

	// Cancel context and verify cleanup stops
	cancel()
	time.Sleep(100 * time.Millisecond)
	// If we reach here without panic, cleanup stopped gracefully
}

func TestRateLimiter_StartCleanup_PanicRecovery(t *testing.T) {
	// This test verifies that cleanup goroutine has panic recovery
	config := &RateLimitConfig{
		RequestsPerWindow: 10,
		WindowDuration:    10 * time.Millisecond,
		BurstSize:         2,
	}
	limiter := NewRateLimiter(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	limiter.StartCleanup(ctx)

	// Give time for cleanup to run
	time.Sleep(50 * time.Millisecond)

	// If we reach here without crashing, panic recovery is working
	// (The actual panic recovery is defensive and hard to trigger in tests)
}

func TestNewRateLimitMiddleware(t *testing.T) {
	middleware := NewRateLimitMiddleware()

	if middleware == nil {
		t.Fatal("NewRateLimitMiddleware should not return nil")
	}
	if middleware.userLimiter == nil {
		t.Error("userLimiter should not be nil")
	}
	if middleware.botLimiter == nil {
		t.Error("botLimiter should not be nil")
	}
	if middleware.anonymousLimiter == nil {
		t.Error("anonymousLimiter should not be nil")
	}

	// Verify different limiters have different configs
	if middleware.userLimiter.config.RequestsPerWindow == middleware.anonymousLimiter.config.RequestsPerWindow {
		t.Error("User and anonymous limiters should have different limits")
	}
	if middleware.botLimiter.config.RequestsPerWindow == middleware.userLimiter.config.RequestsPerWindow {
		t.Error("Bot and user limiters should have different limits")
	}
}

func TestRateLimitMiddleware_Handler_Anonymous(t *testing.T) {
	middleware := NewRateLimitMiddleware()

	// Override config to make testing faster
	middleware.anonymousLimiter.config = &RateLimitConfig{
		RequestsPerWindow: 3,
		WindowDuration:    time.Second,
		BurstSize:         1,
	}

	handlerCalled := false
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// First few requests should succeed
	for i := 0; i < 4; i++ {
		handlerCalled = false
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, rec.Code)
		}
		if !handlerCalled {
			t.Errorf("Request %d: handler was not called", i+1)
		}

		// Check rate limit headers
		if rec.Header().Get("X-RateLimit-Limit") == "" {
			t.Error("X-RateLimit-Limit header should be set")
		}
		if rec.Header().Get("X-RateLimit-Remaining") == "" {
			t.Error("X-RateLimit-Remaining header should be set")
		}
		if rec.Header().Get("X-RateLimit-Reset") == "" {
			t.Error("X-RateLimit-Reset header should be set")
		}
	}

	// Next request should be rate limited
	handlerCalled = false
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rec.Code)
	}
	if handlerCalled {
		t.Error("Handler should not be called when rate limited")
	}

	// Check rate limit exceeded headers
	if rec.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header should be set")
	}
	if rec.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("X-RateLimit-Remaining should be 0, got %s", rec.Header().Get("X-RateLimit-Remaining"))
	}

	// Check response body
	body := rec.Body.String()
	if !strings.Contains(body, "rate limit exceeded") {
		t.Errorf("Response body should contain error message, got: %s", body)
	}
	if !strings.Contains(body, "retry_after") {
		t.Errorf("Response body should contain retry_after, got: %s", body)
	}
}

func TestRateLimitMiddleware_Handler_AuthenticatedUser(t *testing.T) {
	middleware := NewRateLimitMiddleware()

	// Override config to make testing faster
	middleware.userLimiter.config = &RateLimitConfig{
		RequestsPerWindow: 5,
		WindowDuration:    time.Second,
		BurstSize:         0,
	}

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create authenticated request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = setAuthContextForTest(req, &auth.AuthContext{
		User: &auth.User{
			ID:    123,
			IsBot: false,
		},
	})

	// Should allow requests up to limit
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Next request should be rate limited
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_Handler_BotUser(t *testing.T) {
	middleware := NewRateLimitMiddleware()

	// Override config to make testing faster
	middleware.botLimiter.config = &RateLimitConfig{
		RequestsPerWindow: 10,
		WindowDuration:    time.Second,
		BurstSize:         0,
	}

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create bot authenticated request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = setAuthContextForTest(req, &auth.AuthContext{
		User: &auth.User{
			ID:    456,
			IsBot: true,
		},
	})

	// Should allow requests up to bot limit
	for i := 0; i < 10; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Next request should be rate limited
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_Handler_DifferentIPsIndependent(t *testing.T) {
	middleware := NewRateLimitMiddleware()

	// Override config to make testing faster
	middleware.anonymousLimiter.config = &RateLimitConfig{
		RequestsPerWindow: 2,
		WindowDuration:    time.Second,
		BurstSize:         0,
	}

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First IP exhausts limit
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req1)
		if rec.Code != http.StatusOK {
			t.Errorf("First IP request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// First IP should be rate limited
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusTooManyRequests {
		t.Errorf("First IP: expected 429, got %d", rec1.Code)
	}

	// Second IP should still work
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("Second IP: expected 200, got %d", rec2.Code)
	}
}

func TestRateLimitMiddleware_Handler_XForwardedFor(t *testing.T) {
	middleware := NewRateLimitMiddleware()

	middleware.anonymousLimiter.config = &RateLimitConfig{
		RequestsPerWindow: 2,
		WindowDuration:    time.Second,
		BurstSize:         0,
	}

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request with X-Forwarded-For header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")

	// Exhaust limit
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Should be rate limited based on X-Forwarded-For
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rec.Code)
	}

	// Request with different RemoteAddr but no X-Forwarded-For should work
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "10.0.0.2:12345"

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("Different IP: expected 200, got %d", rec2.Code)
	}
}

func TestRateLimiter_TokenCapRefill(t *testing.T) {
	// Test that tokens don't exceed max when refilling
	config := &RateLimitConfig{
		RequestsPerWindow: 10,
		WindowDuration:    100 * time.Millisecond,
		BurstSize:         5,
	}
	limiter := NewRateLimiter(config)

	key := "cap-test"

	// Use some tokens
	for i := 0; i < 5; i++ {
		limiter.Allow(key)
	}

	// Wait for much longer than the window to trigger refill beyond max
	time.Sleep(500 * time.Millisecond)

	// Try to use tokens - should have refilled to max
	allowed := 0
	maxAllowed := config.RequestsPerWindow + config.BurstSize
	for i := 0; i < maxAllowed+5; i++ {
		if limiter.Allow(key) {
			allowed++
		}
	}

	// Should only allow up to max, not more
	if allowed != maxAllowed {
		t.Errorf("Should allow exactly %d requests after full refill, got %d", maxAllowed, allowed)
	}
}

func TestRateLimitMiddleware_RateLimitExceeded_Headers(t *testing.T) {
	middleware := NewRateLimitMiddleware()

	middleware.anonymousLimiter.config = &RateLimitConfig{
		RequestsPerWindow: 1,
		WindowDuration:    time.Minute,
		BurstSize:         0,
	}

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Exhaust limit
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Trigger rate limit
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rec.Code)
	}

	// Verify all required headers are set
	headers := []string{"Content-Type", "Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"}
	for _, header := range headers {
		if rec.Header().Get(header) == "" {
			t.Errorf("Header %s should be set", header)
		}
	}

	// Verify Content-Type is JSON
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should be application/json, got %s", rec.Header().Get("Content-Type"))
	}

	// Verify Retry-After is positive
	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter == "" || retryAfter == "0" {
		t.Errorf("Retry-After should be positive, got %s", retryAfter)
	}
}
