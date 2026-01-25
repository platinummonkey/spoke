package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/platinummonkey/spoke/pkg/api"
)

// RedisCache provides a Redis-based caching layer for PostgreSQL storage
type RedisCache struct {
	storage *PostgresStorage
	redis   *redis.Client
	ttl     map[string]time.Duration
}

// NewRedisCache creates a new Redis cache layer
func NewRedisCache(storage *PostgresStorage, redisAddr string, password string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: password,
		DB:       0, // use default DB
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		storage: storage,
		redis:   client,
		ttl: map[string]time.Duration{
			"module":  15 * time.Minute,
			"version": 30 * time.Minute,
			"file":    1 * time.Hour,
			"list":    5 * time.Minute,
		},
	}, nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.redis.Close()
}

// CreateModule creates a new module and invalidates list cache
func (c *RedisCache) CreateModule(module *api.Module) error {
	if err := c.storage.CreateModule(module); err != nil {
		return err
	}

	// Invalidate module list cache
	ctx := context.Background()
	c.redis.Del(ctx, "modules:list")

	return nil
}

// GetModule gets a module with caching
func (c *RedisCache) GetModule(name string) (*api.Module, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("module:%s", name)

	// Try cache first
	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var module api.Module
		if err := json.Unmarshal([]byte(cached), &module); err == nil {
			return &module, nil
		}
	}

	// Cache miss - fetch from database
	module, err := c.storage.GetModule(name)
	if err != nil {
		return nil, err
	}

	// Store in cache
	data, err := json.Marshal(module)
	if err == nil {
		c.redis.Set(ctx, cacheKey, data, c.ttl["module"])
	}

	return module, nil
}

// ListModules lists modules with caching
func (c *RedisCache) ListModules() ([]*api.Module, error) {
	ctx := context.Background()
	cacheKey := "modules:list"

	// Try cache first
	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var modules []*api.Module
		if err := json.Unmarshal([]byte(cached), &modules); err == nil {
			return modules, nil
		}
	}

	// Cache miss - fetch from database
	modules, err := c.storage.ListModules()
	if err != nil {
		return nil, err
	}

	// Store in cache
	data, err := json.Marshal(modules)
	if err == nil {
		c.redis.Set(ctx, cacheKey, data, c.ttl["list"])
	}

	return modules, nil
}

// CreateVersion creates a new version and invalidates caches
func (c *RedisCache) CreateVersion(version *api.Version) error {
	if err := c.storage.CreateVersion(version); err != nil {
		return err
	}

	// Invalidate caches
	ctx := context.Background()
	c.redis.Del(ctx,
		fmt.Sprintf("versions:%s:list", version.ModuleName),
		fmt.Sprintf("module:%s", version.ModuleName),
		"modules:list",
	)

	return nil
}

// GetVersion gets a version with caching
func (c *RedisCache) GetVersion(moduleName, version string) (*api.Version, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("version:%s:%s", moduleName, version)

	// Try cache first
	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var ver api.Version
		if err := json.Unmarshal([]byte(cached), &ver); err == nil {
			return &ver, nil
		}
	}

	// Cache miss - fetch from database
	ver, err := c.storage.GetVersion(moduleName, version)
	if err != nil {
		return nil, err
	}

	// Store in cache
	data, err := json.Marshal(ver)
	if err == nil {
		c.redis.Set(ctx, cacheKey, data, c.ttl["version"])
	}

	return ver, nil
}

// ListVersions lists versions for a module with caching
func (c *RedisCache) ListVersions(moduleName string) ([]*api.Version, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("versions:%s:list", moduleName)

	// Try cache first
	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var versions []*api.Version
		if err := json.Unmarshal([]byte(cached), &versions); err == nil {
			return versions, nil
		}
	}

	// Cache miss - fetch from database
	versions, err := c.storage.ListVersions(moduleName)
	if err != nil {
		return nil, err
	}

	// Store in cache
	data, err := json.Marshal(versions)
	if err == nil {
		c.redis.Set(ctx, cacheKey, data, c.ttl["list"])
	}

	return versions, nil
}

// GetFile gets a file with caching
func (c *RedisCache) GetFile(moduleName, version, path string) (*api.File, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("file:%s:%s:%s", moduleName, version, path)

	// Try cache first
	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var file api.File
		if err := json.Unmarshal([]byte(cached), &file); err == nil {
			return &file, nil
		}
	}

	// Cache miss - fetch from database
	file, err := c.storage.GetFile(moduleName, version, path)
	if err != nil {
		return nil, err
	}

	// Store in cache
	data, err := json.Marshal(file)
	if err == nil {
		c.redis.Set(ctx, cacheKey, data, c.ttl["file"])
	}

	return file, nil
}

// InvalidateModule removes module from cache
func (c *RedisCache) InvalidateModule(name string) error {
	ctx := context.Background()
	return c.redis.Del(ctx, fmt.Sprintf("module:%s", name)).Err()
}

// InvalidateVersion removes version from cache
func (c *RedisCache) InvalidateVersion(moduleName, version string) error {
	ctx := context.Background()
	return c.redis.Del(ctx,
		fmt.Sprintf("version:%s:%s", moduleName, version),
		fmt.Sprintf("versions:%s:list", moduleName),
	).Err()
}

// InvalidateAll clears all cached data
func (c *RedisCache) InvalidateAll() error {
	ctx := context.Background()
	return c.redis.FlushDB(ctx).Err()
}

// GetCacheStats returns cache statistics
func (c *RedisCache) GetCacheStats() (map[string]interface{}, error) {
	ctx := context.Background()

	// Get Redis INFO
	info, err := c.redis.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	// Get key count
	dbSize, err := c.redis.DBSize(ctx).Result()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"keys":  dbSize,
		"info":  info,
		"connected": true,
	}, nil
}

// WarmupCache pre-loads frequently accessed data into cache
func (c *RedisCache) WarmupCache() error {
	// Load all modules
	modules, err := c.storage.ListModules()
	if err != nil {
		return fmt.Errorf("failed to load modules: %w", err)
	}

	ctx := context.Background()
	for _, module := range modules {
		// Cache module
		data, err := json.Marshal(module)
		if err != nil {
			continue
		}
		c.redis.Set(ctx, fmt.Sprintf("module:%s", module.Name), data, c.ttl["module"])

		// Cache versions for this module
		versions, err := c.storage.ListVersions(module.Name)
		if err != nil {
			continue
		}

		versionData, err := json.Marshal(versions)
		if err != nil {
			continue
		}
		c.redis.Set(ctx, fmt.Sprintf("versions:%s:list", module.Name), versionData, c.ttl["list"])

		// Cache individual versions (limit to latest 5)
		for i, version := range versions {
			if i >= 5 {
				break
			}
			verData, err := json.Marshal(version)
			if err != nil {
				continue
			}
			c.redis.Set(ctx,
				fmt.Sprintf("version:%s:%s", version.ModuleName, version.Version),
				verData,
				c.ttl["version"],
			)
		}
	}

	// Cache module list
	modulesData, err := json.Marshal(modules)
	if err == nil {
		c.redis.Set(ctx, "modules:list", modulesData, c.ttl["list"])
	}

	return nil
}

// SetTTL updates TTL for a specific cache type
func (c *RedisCache) SetTTL(cacheType string, ttl time.Duration) {
	c.ttl[cacheType] = ttl
}

// GetTTL returns TTL for a specific cache type
func (c *RedisCache) GetTTL(cacheType string) time.Duration {
	return c.ttl[cacheType]
}
