package cache

import "errors"

var (
	// ErrCacheMiss is returned when a cache key is not found
	ErrCacheMiss = errors.New("cache miss")

	// ErrCacheUnavailable is returned when the cache is unavailable
	ErrCacheUnavailable = errors.New("cache unavailable")

	// ErrInvalidCacheKey is returned when a cache key is invalid
	ErrInvalidCacheKey = errors.New("invalid cache key")

	// ErrCacheFull is returned when the cache is full
	ErrCacheFull = errors.New("cache full")
)
