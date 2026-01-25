package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/storage"
)

// RedisClient handles caching operations
type RedisClient struct {
	client *redis.Client
	config storage.Config
}

// NewRedisClient creates a new Redis client
func NewRedisClient(config storage.Config) (*RedisClient, error) {
	// Parse Redis URL or use default options
	opts, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	// Override with config values if provided
	if config.RedisPassword != "" {
		opts.Password = config.RedisPassword
	}
	if config.RedisDB >= 0 {
		opts.DB = config.RedisDB
	}
	if config.RedisMaxRetries > 0 {
		opts.MaxRetries = config.RedisMaxRetries
	}
	if config.RedisPoolSize > 0 {
		opts.PoolSize = config.RedisPoolSize
	}

	// Set connection timeouts
	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second
	opts.PoolTimeout = 4 * time.Second

	// Create client
	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		client: client,
		config: config,
	}, nil
}

// GetModule retrieves a module from cache
func (c *RedisClient) GetModule(ctx context.Context, name string) (*api.Module, error) {
	key := fmt.Sprintf("module:%s", name)

	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	} else if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var module api.Module
	if err := json.Unmarshal([]byte(data), &module); err != nil {
		// If unmarshal fails, delete corrupt data
		c.client.Del(ctx, key)
		return nil, fmt.Errorf("failed to unmarshal module: %w", err)
	}

	return &module, nil
}

// SetModule stores a module in cache
func (c *RedisClient) SetModule(ctx context.Context, module *api.Module) error {
	key := fmt.Sprintf("module:%s", module.Name)
	ttl := c.config.CacheTTL["module"]

	data, err := json.Marshal(module)
	if err != nil {
		return fmt.Errorf("failed to marshal module: %w", err)
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

// InvalidateModule removes a module from cache
func (c *RedisClient) InvalidateModule(ctx context.Context, name string) error {
	key := fmt.Sprintf("module:%s", name)
	return c.client.Del(ctx, key).Err()
}

// GetVersion retrieves a version from cache
func (c *RedisClient) GetVersion(ctx context.Context, moduleName, version string) (*api.Version, error) {
	key := fmt.Sprintf("version:%s:%s", moduleName, version)

	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	} else if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var v api.Version
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		// If unmarshal fails, delete corrupt data
		c.client.Del(ctx, key)
		return nil, fmt.Errorf("failed to unmarshal version: %w", err)
	}

	return &v, nil
}

// SetVersion stores a version in cache
func (c *RedisClient) SetVersion(ctx context.Context, version *api.Version) error {
	key := fmt.Sprintf("version:%s:%s", version.ModuleName, version.Version)
	ttl := c.config.CacheTTL["version"]

	data, err := json.Marshal(version)
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

// InvalidateVersion removes a version from cache
func (c *RedisClient) InvalidateVersion(ctx context.Context, moduleName, version string) error {
	key := fmt.Sprintf("version:%s:%s", moduleName, version)
	return c.client.Del(ctx, key).Err()
}

// InvalidatePatterns removes keys matching patterns
func (c *RedisClient) InvalidatePatterns(ctx context.Context, patterns ...string) error {
	for _, pattern := range patterns {
		// Use SCAN to find matching keys
		iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
				return fmt.Errorf("failed to delete key %s: %w", iter.Val(), err)
			}
		}
		if err := iter.Err(); err != nil {
			return fmt.Errorf("scan failed for pattern %s: %w", pattern, err)
		}
	}
	return nil
}

// Ping checks Redis connectivity
func (c *RedisClient) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// GetClient returns the underlying Redis client for health checks
func (c *RedisClient) GetClient() *redis.Client {
	return c.client
}

// Close closes the Redis connection
func (c *RedisClient) Close() error {
	return c.client.Close()
}

// GetPoolStats returns connection pool statistics
func (c *RedisClient) GetPoolStats() *redis.PoolStats {
	return c.client.PoolStats()
}

// Incr increments a counter (for rate limiting)
func (c *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Expire sets a key's expiration
func (c *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// TTL returns the remaining time to live of a key
func (c *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// SetNX sets a key only if it doesn't exist (for distributed locks)
func (c *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, expiration).Result()
}

// GetDel atomically gets and deletes a key
func (c *RedisClient) GetDel(ctx context.Context, key string) (string, error) {
	return c.client.GetDel(ctx, key).Result()
}
