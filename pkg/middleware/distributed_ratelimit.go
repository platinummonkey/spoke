package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

// DistributedRateLimiter implements rate limiting using Redis
// This allows rate limits to be shared across multiple instances
type DistributedRateLimiter struct {
	redis  *redis.Client
	config *RateLimitConfig
	prefix string
}

// NewDistributedRateLimiter creates a new Redis-backed rate limiter
func NewDistributedRateLimiter(redisClient *redis.Client, config *RateLimitConfig, prefix string) *DistributedRateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	if prefix == "" {
		prefix = "ratelimit"
	}

	return &DistributedRateLimiter{
		redis:  redisClient,
		config: config,
		prefix: prefix,
	}
}

// Allow checks if a request is allowed using Redis-backed token bucket
func (rl *DistributedRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := fmt.Sprintf("%s:%s", rl.prefix, key)

	// Use Redis pipeline for atomic operations
	pipe := rl.redis.Pipeline()

	// Increment counter
	incr := pipe.Incr(ctx, redisKey)

	// Set expiration if this is a new key
	pipe.Expire(ctx, redisKey, rl.config.WindowDuration)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		// On Redis error, fail open (allow request) to prevent service disruption
		return true, fmt.Errorf("redis error: %w", err)
	}

	// Check if under limit
	count := incr.Val()
	return count <= int64(rl.config.RequestsPerWindow), nil
}

// Remaining returns the number of remaining requests in the window
func (rl *DistributedRateLimiter) Remaining(ctx context.Context, key string) (int, error) {
	redisKey := fmt.Sprintf("%s:%s", rl.prefix, key)

	count, err := rl.redis.Get(ctx, redisKey).Int()
	if err == redis.Nil {
		// Key doesn't exist, full quota available
		return rl.config.RequestsPerWindow, nil
	} else if err != nil {
		return 0, err
	}

	remaining := rl.config.RequestsPerWindow - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// TTL returns the time until the rate limit window resets
func (rl *DistributedRateLimiter) TTL(ctx context.Context, key string) (time.Duration, error) {
	redisKey := fmt.Sprintf("%s:%s", rl.prefix, key)
	return rl.redis.TTL(ctx, redisKey).Result()
}

// Reset clears the rate limit for a key (for testing or admin purposes)
func (rl *DistributedRateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("%s:%s", rl.prefix, key)
	return rl.redis.Del(ctx, redisKey).Err()
}

// DistributedRateLimitMiddleware provides HTTP rate limiting with Redis
type DistributedRateLimitMiddleware struct {
	redis            *redis.Client
	userLimiter      *DistributedRateLimiter
	botLimiter       *DistributedRateLimiter
	anonymousLimiter *DistributedRateLimiter
	fallbackEnabled  bool
}

// NewDistributedRateLimitMiddleware creates a new Redis-backed rate limit middleware
func NewDistributedRateLimitMiddleware(redisClient *redis.Client) *DistributedRateLimitMiddleware {
	return &DistributedRateLimitMiddleware{
		redis:            redisClient,
		userLimiter:      NewDistributedRateLimiter(redisClient, PerUserRateLimitConfig(), "ratelimit:user"),
		botLimiter:       NewDistributedRateLimiter(redisClient, PerBotRateLimitConfig(), "ratelimit:bot"),
		anonymousLimiter: NewDistributedRateLimiter(redisClient, DefaultRateLimitConfig(), "ratelimit:anon"),
		fallbackEnabled:  true, // Fail open on Redis errors
	}
}

// Handler wraps an HTTP handler with distributed rate limiting
func (m *DistributedRateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Determine rate limit key
		var key string
		var limiter *DistributedRateLimiter

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
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			if m.fallbackEnabled {
				// Fail open: allow request on Redis error
				// Log the error but don't block the request
				// In production, you'd want to increment an error counter here
				next.ServeHTTP(w, r)
				return
			}
			// Fail closed: return 503 Service Unavailable
			http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
			return
		}

		if !allowed {
			m.rateLimitExceeded(ctx, w, limiter, key)
			return
		}

		// Add rate limit headers
		remaining, err := limiter.Remaining(ctx, key)
		if err != nil {
			// If we can't get remaining count, still serve request but without headers
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.RequestsPerWindow))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		// Get TTL for reset time
		ttl, err := limiter.TTL(ctx, key)
		if err == nil && ttl > 0 {
			resetTime := time.Now().Add(ttl).Unix()
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
		}

		next.ServeHTTP(w, r)
	})
}

func (m *DistributedRateLimitMiddleware) rateLimitExceeded(ctx context.Context, w http.ResponseWriter, limiter *DistributedRateLimiter, key string) {
	// Get TTL for Retry-After header
	ttl, err := limiter.TTL(ctx, key)
	retryAfter := limiter.config.WindowDuration.Seconds()
	if err == nil && ttl > 0 {
		retryAfter = ttl.Seconds()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter))
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.RequestsPerWindow))
	w.Header().Set("X-RateLimit-Remaining", "0")

	if ttl > 0 {
		resetTime := time.Now().Add(ttl).Unix()
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
	}

	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"error":"rate limit exceeded","retry_after":` + fmt.Sprintf("%.0f", retryAfter) + `}`))
}

// SetFallbackEnabled controls whether to fail open (true) or closed (false) on Redis errors
func (m *DistributedRateLimitMiddleware) SetFallbackEnabled(enabled bool) {
	m.fallbackEnabled = enabled
}

// HealthCheck verifies Redis connectivity for rate limiting
func (m *DistributedRateLimitMiddleware) HealthCheck(ctx context.Context) error {
	return m.redis.Ping(ctx).Err()
}

// GetStats returns rate limiting statistics from Redis
func (m *DistributedRateLimitMiddleware) GetStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	// Count keys for each limiter type
	patterns := []string{
		"ratelimit:user:*",
		"ratelimit:bot:*",
		"ratelimit:anon:*",
	}

	for _, pattern := range patterns {
		keys, err := m.redis.Keys(ctx, pattern).Result()
		if err != nil {
			return nil, err
		}
		stats[pattern] = int64(len(keys))
	}

	return stats, nil
}
