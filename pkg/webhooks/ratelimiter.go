package webhooks

import (
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting per webhook
type RateLimiter struct {
	buckets       map[string]*TokenBucket
	mutex         sync.RWMutex
	maxTokens     int
	refillPeriod  time.Duration
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens       int
	maxTokens    int
	refillPeriod time.Duration
	lastRefill   time.Time
	mutex        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, period time.Duration) *RateLimiter {
	return &RateLimiter{
		buckets:      make(map[string]*TokenBucket),
		maxTokens:    maxRequests,
		refillPeriod: period,
	}
}

// Allow checks if a request is allowed for the given webhook
func (rl *RateLimiter) Allow(webhookID string) bool {
	rl.mutex.Lock()
	bucket, exists := rl.buckets[webhookID]
	if !exists {
		bucket = &TokenBucket{
			tokens:       rl.maxTokens,
			maxTokens:    rl.maxTokens,
			refillPeriod: rl.refillPeriod,
			lastRefill:   time.Now(),
		}
		rl.buckets[webhookID] = bucket
	}
	rl.mutex.Unlock()

	return bucket.Take()
}

// Take attempts to take a token from the bucket
func (tb *TokenBucket) Take() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	if elapsed >= tb.refillPeriod {
		periods := int(elapsed / tb.refillPeriod)
		tb.tokens = min(tb.tokens+periods, tb.maxTokens)
		tb.lastRefill = tb.lastRefill.Add(time.Duration(periods) * tb.refillPeriod)
	}

	// Check if we have tokens available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// Reset resets the rate limiter for a webhook
func (rl *RateLimiter) Reset(webhookID string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	delete(rl.buckets, webhookID)
}

// GetRemaining returns the number of remaining tokens for a webhook
func (rl *RateLimiter) GetRemaining(webhookID string) int {
	rl.mutex.RLock()
	bucket, exists := rl.buckets[webhookID]
	rl.mutex.RUnlock()

	if !exists {
		return rl.maxTokens
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	// Refill first
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	if elapsed >= bucket.refillPeriod {
		periods := int(elapsed / bucket.refillPeriod)
		bucket.tokens = min(bucket.tokens+periods, bucket.maxTokens)
		bucket.lastRefill = bucket.lastRefill.Add(time.Duration(periods) * bucket.refillPeriod)
	}

	return bucket.tokens
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
