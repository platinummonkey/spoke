package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	// RequestsPerWindow is the max requests allowed in the time window
	RequestsPerWindow int
	// WindowDuration is the time window for rate limiting
	WindowDuration time.Duration
	// BurstSize allows temporary bursts above the rate
	BurstSize int
}

// DefaultRateLimitConfig returns default rate limit settings
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: 100,
		WindowDuration:    time.Minute,
		BurstSize:         10,
	}
}

// PerUserRateLimitConfig returns per-user rate limit settings
func PerUserRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: 1000,
		WindowDuration:    time.Minute,
		BurstSize:         50,
	}
}

// PerBotRateLimitConfig returns rate limits for bot users (more generous)
func PerBotRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: 5000,
		WindowDuration:    time.Minute,
		BurstSize:         100,
	}
}

// RateLimiter implements rate limiting using token bucket algorithm
type RateLimiter struct {
	config *RateLimitConfig
	// In-memory buckets (for simple implementation)
	// In production, use Redis for distributed rate limiting
	buckets map[string]*bucket
	mu      sync.RWMutex
}

type bucket struct {
	tokens     int
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	return &RateLimiter{
		config:  config,
		buckets: make(map[string]*bucket),
	}
}

// Allow checks if a request is allowed for the given key
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	b, exists := rl.buckets[key]
	if !exists {
		b = &bucket{
			tokens:     rl.config.RequestsPerWindow + rl.config.BurstSize,
			lastUpdate: time.Now(),
		}
		rl.buckets[key] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastUpdate)

	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed.Seconds() * float64(rl.config.RequestsPerWindow) / rl.config.WindowDuration.Seconds())
	if tokensToAdd > 0 {
		b.tokens += tokensToAdd
		maxTokens := rl.config.RequestsPerWindow + rl.config.BurstSize
		if b.tokens > maxTokens {
			b.tokens = maxTokens
		}
		b.lastUpdate = now
	}

	// Check if request is allowed
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// Remaining returns the number of remaining tokens for a key
func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.RLock()
	b, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if !exists {
		return rl.config.RequestsPerWindow + rl.config.BurstSize
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	return b.tokens
}

// Cleanup removes old buckets (should be called periodically)
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, b := range rl.buckets {
		b.mu.Lock()
		if now.Sub(b.lastUpdate) > rl.config.WindowDuration*2 {
			delete(rl.buckets, key)
		}
		b.mu.Unlock()
	}
}

// StartCleanup starts a background goroutine to cleanup old buckets
func (rl *RateLimiter) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(rl.config.WindowDuration)
	go func() {
		for {
			select {
			case <-ticker.C:
				rl.Cleanup()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// RateLimitMiddleware provides HTTP rate limiting
type RateLimitMiddleware struct {
	userLimiter      *RateLimiter
	botLimiter       *RateLimiter
	anonymousLimiter *RateLimiter
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware() *RateLimitMiddleware {
	return &RateLimitMiddleware{
		userLimiter:      NewRateLimiter(PerUserRateLimitConfig()),
		botLimiter:       NewRateLimiter(PerBotRateLimitConfig()),
		anonymousLimiter: NewRateLimiter(DefaultRateLimitConfig()),
	}
}

// Handler wraps an HTTP handler with rate limiting
func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Determine rate limit key
		var key string
		var limiter *RateLimiter

		authCtx := GetAuthContext(r)
		if authCtx != nil && authCtx.User != nil {
			// Authenticated user
			key = fmt.Sprintf("user:%d", authCtx.User.ID)
			if authCtx.User.IsBot {
				limiter = m.botLimiter
			} else {
				limiter = m.userLimiter
			}
		} else {
			// Anonymous/unauthenticated - use IP address
			key = "ip:" + getClientIP(r)
			limiter = m.anonymousLimiter
		}

		// Check rate limit
		if !limiter.Allow(key) {
			m.rateLimitExceeded(w, limiter, key)
			return
		}

		// Add rate limit headers
		remaining := limiter.Remaining(key)
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.RequestsPerWindow))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(limiter.config.WindowDuration).Unix()))

		next.ServeHTTP(w, r)
	})
}

func (m *RateLimitMiddleware) rateLimitExceeded(w http.ResponseWriter, limiter *RateLimiter, key string) {
	retryAfter := limiter.config.WindowDuration.Seconds()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter))
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.RequestsPerWindow))
	w.Header().Set("X-RateLimit-Remaining", "0")
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(limiter.config.WindowDuration).Unix()))
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"error":"rate limit exceeded","retry_after":` + fmt.Sprintf("%.0f", retryAfter) + `}`))
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (if behind proxy)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Use remote address
	return r.RemoteAddr
}
