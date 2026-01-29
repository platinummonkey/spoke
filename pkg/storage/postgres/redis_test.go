package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/storage"
)

// setupRedisClientTest creates a miniredis instance and returns the client and cleanup function
func setupRedisClientTest(t *testing.T) (*RedisClient, *miniredis.Miniredis, func()) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	config := storage.Config{
		RedisURL: "redis://" + mr.Addr(),
		CacheTTL: map[string]time.Duration{
			"module":  1 * time.Hour,
			"version": 30 * time.Minute,
		},
		RedisDB:         0,
		RedisMaxRetries: 3,
		RedisPoolSize:   10,
	}

	client, err := NewRedisClient(config)
	if err != nil {
		mr.Close()
		t.Fatalf("Failed to create Redis client: %v", err)
	}

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return client, mr, cleanup
}

func TestNewRedisClient_Success(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	if client.client == nil {
		t.Fatal("Expected underlying redis client to be non-nil")
	}
}

func TestNewRedisClient_InvalidURL(t *testing.T) {
	config := storage.Config{
		RedisURL: "invalid://url",
	}

	_, err := NewRedisClient(config)
	if err == nil {
		t.Fatal("Expected error for invalid Redis URL")
	}
}

func TestNewRedisClient_ConnectionFailure(t *testing.T) {
	config := storage.Config{
		RedisURL: "redis://localhost:9999", // Non-existent server
		CacheTTL: map[string]time.Duration{
			"module": 1 * time.Hour,
		},
	}

	_, err := NewRedisClient(config)
	if err == nil {
		t.Fatal("Expected connection error")
	}
}

func TestNewRedisClient_WithCustomConfig(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	config := storage.Config{
		RedisURL:        "redis://" + mr.Addr(),
		RedisDB:         2,
		RedisMaxRetries: 5,
		RedisPoolSize:   20,
		CacheTTL: map[string]time.Duration{
			"module": 1 * time.Hour,
		},
	}

	client, err := NewRedisClient(config)
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	if client.config.RedisDB != 2 {
		t.Errorf("Expected RedisDB to be 2, got %d", client.config.RedisDB)
	}
	if client.config.RedisMaxRetries != 5 {
		t.Errorf("Expected RedisMaxRetries to be 5, got %d", client.config.RedisMaxRetries)
	}
	if client.config.RedisPoolSize != 20 {
		t.Errorf("Expected RedisPoolSize to be 20, got %d", client.config.RedisPoolSize)
	}
}

func TestRedisClient_Ping(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestRedisClient_GetClient(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	underlyingClient := client.GetClient()
	if underlyingClient == nil {
		t.Fatal("Expected GetClient to return non-nil client")
	}

	// Verify it's a working Redis client
	ctx := context.Background()
	if err := underlyingClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("Underlying client ping failed: %v", err)
	}
}

func TestRedisClient_GetPoolStats(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	stats := client.GetPoolStats()
	if stats == nil {
		t.Fatal("Expected GetPoolStats to return non-nil stats")
	}
}

func TestRedisClient_SetAndGetModule(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	module := &api.Module{
		Name:        "test.module",
		Description: "Test module for caching",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set the module
	err := client.SetModule(ctx, module)
	if err != nil {
		t.Fatalf("SetModule failed: %v", err)
	}

	// Get the module
	retrieved, err := client.GetModule(ctx, "test.module")
	if err != nil {
		t.Fatalf("GetModule failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected retrieved module to be non-nil")
	}

	if retrieved.Name != module.Name {
		t.Errorf("Expected name %s, got %s", module.Name, retrieved.Name)
	}

	if retrieved.Description != module.Description {
		t.Errorf("Expected description %s, got %s", module.Description, retrieved.Description)
	}
}

func TestRedisClient_GetModule_NotFound(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Get non-existent module
	retrieved, err := client.GetModule(ctx, "nonexistent.module")
	if err != nil {
		t.Fatalf("GetModule failed: %v", err)
	}

	if retrieved != nil {
		t.Errorf("Expected nil for non-existent module, got %v", retrieved)
	}
}

func TestRedisClient_GetModule_CorruptData(t *testing.T) {
	client, mr, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Set corrupt data directly in Redis
	mr.Set("module:corrupt.module", "invalid json data")

	// Try to get the corrupt module
	retrieved, err := client.GetModule(ctx, "corrupt.module")
	if err == nil {
		t.Fatal("Expected error for corrupt data")
	}

	if retrieved != nil {
		t.Errorf("Expected nil for corrupt module, got %v", retrieved)
	}

	// Verify that corrupt data was deleted
	exists := mr.Exists("module:corrupt.module")
	if exists {
		t.Error("Expected corrupt data to be deleted")
	}
}

func TestRedisClient_InvalidateModule(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	module := &api.Module{
		Name:        "test.module",
		Description: "Test module",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set the module
	err := client.SetModule(ctx, module)
	if err != nil {
		t.Fatalf("SetModule failed: %v", err)
	}

	// Verify it exists
	retrieved, err := client.GetModule(ctx, "test.module")
	if err != nil {
		t.Fatalf("GetModule failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected module to exist")
	}

	// Invalidate the module
	err = client.InvalidateModule(ctx, "test.module")
	if err != nil {
		t.Fatalf("InvalidateModule failed: %v", err)
	}

	// Verify it's gone
	retrieved, err = client.GetModule(ctx, "test.module")
	if err != nil {
		t.Fatalf("GetModule failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected module to be deleted, got %v", retrieved)
	}
}

func TestRedisClient_SetAndGetVersion(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	version := &api.Version{
		ModuleName: "test.module",
		Version:    "v1.0.0",
		Files: []api.File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
		CreatedAt: now,
		SourceInfo: api.SourceInfo{
			Repository: "github.com/test/repo",
			CommitSHA:  "abc123",
			Branch:     "main",
		},
	}

	// Set the version
	err := client.SetVersion(ctx, version)
	if err != nil {
		t.Fatalf("SetVersion failed: %v", err)
	}

	// Get the version
	retrieved, err := client.GetVersion(ctx, "test.module", "v1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected retrieved version to be non-nil")
	}

	if retrieved.ModuleName != version.ModuleName {
		t.Errorf("Expected module name %s, got %s", version.ModuleName, retrieved.ModuleName)
	}

	if retrieved.Version != version.Version {
		t.Errorf("Expected version %s, got %s", version.Version, retrieved.Version)
	}

	if len(retrieved.Files) != len(version.Files) {
		t.Errorf("Expected %d files, got %d", len(version.Files), len(retrieved.Files))
	}
}

func TestRedisClient_GetVersion_NotFound(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Get non-existent version
	retrieved, err := client.GetVersion(ctx, "nonexistent.module", "v1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if retrieved != nil {
		t.Errorf("Expected nil for non-existent version, got %v", retrieved)
	}
}

func TestRedisClient_GetVersion_CorruptData(t *testing.T) {
	client, mr, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Set corrupt data directly in Redis
	mr.Set("version:corrupt.module:v1.0.0", "invalid json data")

	// Try to get the corrupt version
	retrieved, err := client.GetVersion(ctx, "corrupt.module", "v1.0.0")
	if err == nil {
		t.Fatal("Expected error for corrupt data")
	}

	if retrieved != nil {
		t.Errorf("Expected nil for corrupt version, got %v", retrieved)
	}

	// Verify that corrupt data was deleted
	exists := mr.Exists("version:corrupt.module:v1.0.0")
	if exists {
		t.Error("Expected corrupt data to be deleted")
	}
}

func TestRedisClient_InvalidateVersion(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	version := &api.Version{
		ModuleName: "test.module",
		Version:    "v1.0.0",
		CreatedAt:  now,
	}

	// Set the version
	err := client.SetVersion(ctx, version)
	if err != nil {
		t.Fatalf("SetVersion failed: %v", err)
	}

	// Verify it exists
	retrieved, err := client.GetVersion(ctx, "test.module", "v1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected version to exist")
	}

	// Invalidate the version
	err = client.InvalidateVersion(ctx, "test.module", "v1.0.0")
	if err != nil {
		t.Fatalf("InvalidateVersion failed: %v", err)
	}

	// Verify it's gone
	retrieved, err = client.GetVersion(ctx, "test.module", "v1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected version to be deleted, got %v", retrieved)
	}
}

func TestRedisClient_InvalidatePatterns(t *testing.T) {
	client, mr, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create multiple modules and versions
	modules := []*api.Module{
		{Name: "test.module1", Description: "Module 1", CreatedAt: now, UpdatedAt: now},
		{Name: "test.module2", Description: "Module 2", CreatedAt: now, UpdatedAt: now},
		{Name: "other.module", Description: "Other", CreatedAt: now, UpdatedAt: now},
	}

	for _, m := range modules {
		if err := client.SetModule(ctx, m); err != nil {
			t.Fatalf("SetModule failed: %v", err)
		}
	}

	versions := []*api.Version{
		{ModuleName: "test.module1", Version: "v1.0.0", CreatedAt: now},
		{ModuleName: "test.module1", Version: "v1.1.0", CreatedAt: now},
		{ModuleName: "test.module2", Version: "v1.0.0", CreatedAt: now},
	}

	for _, v := range versions {
		if err := client.SetVersion(ctx, v); err != nil {
			t.Fatalf("SetVersion failed: %v", err)
		}
	}

	// Invalidate all test.module* patterns
	err := client.InvalidatePatterns(ctx, "module:test.module*", "version:test.module1:*")
	if err != nil {
		t.Fatalf("InvalidatePatterns failed: %v", err)
	}

	// Verify test.module1 and test.module2 are gone
	if mr.Exists("module:test.module1") {
		t.Error("Expected test.module1 to be deleted")
	}
	if mr.Exists("module:test.module2") {
		t.Error("Expected test.module2 to be deleted")
	}

	// Verify other.module still exists
	if !mr.Exists("module:other.module") {
		t.Error("Expected other.module to still exist")
	}

	// Verify test.module1 versions are gone
	if mr.Exists("version:test.module1:v1.0.0") {
		t.Error("Expected test.module1:v1.0.0 to be deleted")
	}
	if mr.Exists("version:test.module1:v1.1.0") {
		t.Error("Expected test.module1:v1.1.0 to be deleted")
	}

	// Verify test.module2 version still exists (not in pattern)
	if !mr.Exists("version:test.module2:v1.0.0") {
		t.Error("Expected test.module2:v1.0.0 to still exist")
	}
}

func TestRedisClient_InvalidatePatterns_NoMatches(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Invalidate pattern that matches nothing
	err := client.InvalidatePatterns(ctx, "nonexistent:*")
	if err != nil {
		t.Fatalf("InvalidatePatterns should not fail for non-matching pattern: %v", err)
	}
}

func TestRedisClient_Incr(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	key := "counter:test"

	// First increment
	val, err := client.Incr(ctx, key)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if val != 1 {
		t.Errorf("Expected 1, got %d", val)
	}

	// Second increment
	val, err = client.Incr(ctx, key)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if val != 2 {
		t.Errorf("Expected 2, got %d", val)
	}

	// Third increment
	val, err = client.Incr(ctx, key)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if val != 3 {
		t.Errorf("Expected 3, got %d", val)
	}
}

func TestRedisClient_Expire(t *testing.T) {
	client, mr, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:key"

	// Set a value
	mr.Set(key, "test value")

	// Set expiration
	err := client.Expire(ctx, key, 1*time.Second)
	if err != nil {
		t.Fatalf("Expire failed: %v", err)
	}

	// Check TTL is set
	ttl, err := client.TTL(ctx, key)
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	if ttl <= 0 {
		t.Errorf("Expected positive TTL, got %v", ttl)
	}

	if ttl > 1*time.Second {
		t.Errorf("Expected TTL <= 1 second, got %v", ttl)
	}
}

func TestRedisClient_TTL(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:ttl"

	// Set a value with TTL
	client.client.Set(ctx, key, "value", 1*time.Hour)

	// Get TTL
	ttl, err := client.TTL(ctx, key)
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	if ttl <= 0 {
		t.Errorf("Expected positive TTL, got %v", ttl)
	}

	if ttl > 1*time.Hour {
		t.Errorf("Expected TTL <= 1 hour, got %v", ttl)
	}
}

func TestRedisClient_TTL_NoExpiration(t *testing.T) {
	client, mr, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:no-ttl"

	// Set a value without TTL
	mr.Set(key, "value")

	// Get TTL
	ttl, err := client.TTL(ctx, key)
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	// Redis returns -1 for keys with no expiration
	if ttl != -1 {
		t.Errorf("Expected TTL -1 for key without expiration, got %v", ttl)
	}
}

func TestRedisClient_TTL_NonExistentKey(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Get TTL for non-existent key
	ttl, err := client.TTL(ctx, "nonexistent:key")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	// Redis returns -2 for keys that don't exist
	if ttl != -2 {
		t.Errorf("Expected TTL -2 for non-existent key, got %v", ttl)
	}
}

func TestRedisClient_SetNX(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	key := "lock:test"

	// First SetNX should succeed
	success, err := client.SetNX(ctx, key, "locked", 1*time.Hour)
	if err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if !success {
		t.Error("Expected first SetNX to succeed")
	}

	// Second SetNX should fail (key exists)
	success, err = client.SetNX(ctx, key, "locked-again", 1*time.Hour)
	if err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if success {
		t.Error("Expected second SetNX to fail")
	}
}

func TestRedisClient_GetDel(t *testing.T) {
	client, mr, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:getdel"
	value := "test value"

	// Set a value
	mr.Set(key, value)

	// GetDel should return the value and delete it
	retrieved, err := client.GetDel(ctx, key)
	if err != nil {
		t.Fatalf("GetDel failed: %v", err)
	}

	if retrieved != value {
		t.Errorf("Expected value %s, got %s", value, retrieved)
	}

	// Verify key is deleted
	exists := mr.Exists(key)
	if exists {
		t.Error("Expected key to be deleted after GetDel")
	}
}

func TestRedisClient_GetDel_NonExistentKey(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// GetDel on non-existent key should return error
	_, err := client.GetDel(ctx, "nonexistent:key")
	if err == nil {
		t.Fatal("Expected error for non-existent key")
	}

	// Should be redis.Nil error
	if err != redis.Nil {
		t.Errorf("Expected redis.Nil error, got %v", err)
	}
}

func TestRedisClient_Close(t *testing.T) {
	client, mr, _ := setupRedisClientTest(t)

	// Close the client
	err := client.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Clean up miniredis
	mr.Close()

	// Verify connection is closed by trying to ping
	ctx := context.Background()
	err = client.Ping(ctx)
	if err == nil {
		t.Error("Expected error after closing connection")
	}
}

func TestRedisClient_ConcurrentOperations(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create multiple modules concurrently
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			module := &api.Module{
				Name:        "test.module" + string(rune('0'+idx)),
				Description: "Concurrent test",
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			if err := client.SetModule(ctx, module); err != nil {
				errors <- err
				done <- false
				return
			}

			retrieved, err := client.GetModule(ctx, module.Name)
			if err != nil {
				errors <- err
				done <- false
				return
			}

			if retrieved == nil || retrieved.Name != module.Name {
				errors <- err
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case err := <-errors:
			t.Fatalf("Concurrent operation failed: %v", err)
		case success := <-done:
			if !success {
				t.Fatal("Concurrent operation failed")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}

func TestRedisClient_ContextCancellation(t *testing.T) {
	client, _, cleanup := setupRedisClientTest(t)
	defer cleanup()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	module := &api.Module{
		Name:        "test.module",
		Description: "Test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Operation should fail due to cancelled context
	err := client.SetModule(ctx, module)
	if err == nil {
		t.Fatal("Expected error with cancelled context")
	}
}

func TestRedisClient_ExpirationRespected(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Set very short TTL
	config := storage.Config{
		RedisURL: "redis://" + mr.Addr(),
		CacheTTL: map[string]time.Duration{
			"module": 1 * time.Millisecond,
		},
	}

	client, err := NewRedisClient(config)
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	now := time.Now()

	module := &api.Module{
		Name:        "test.module",
		Description: "Test expiration",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set the module
	err = client.SetModule(ctx, module)
	if err != nil {
		t.Fatalf("SetModule failed: %v", err)
	}

	// Fast-forward time in miniredis
	mr.FastForward(2 * time.Millisecond)

	// Get should return nil (cache miss)
	retrieved, err := client.GetModule(ctx, "test.module")
	if err != nil {
		t.Fatalf("GetModule failed: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected module to be expired")
	}
}

func TestRedisClient_KeyFormats(t *testing.T) {
	client, mr, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	module := &api.Module{
		Name:        "test.module",
		Description: "Test",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	version := &api.Version{
		ModuleName: "test.module",
		Version:    "v1.0.0",
		CreatedAt:  now,
	}

	// Set module and version
	client.SetModule(ctx, module)
	client.SetVersion(ctx, version)

	// Verify keys are formatted correctly
	if !mr.Exists("module:test.module") {
		t.Error("Expected module key to be 'module:test.module'")
	}

	if !mr.Exists("version:test.module:v1.0.0") {
		t.Error("Expected version key to be 'version:test.module:v1.0.0'")
	}
}

func TestRedisClient_ModuleSerialization(t *testing.T) {
	client, mr, cleanup := setupRedisClientTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()

	module := &api.Module{
		Name:        "complex.module",
		Description: "Module with special chars: <>&\"'",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set and get the module
	err := client.SetModule(ctx, module)
	if err != nil {
		t.Fatalf("SetModule failed: %v", err)
	}

	retrieved, err := client.GetModule(ctx, "complex.module")
	if err != nil {
		t.Fatalf("GetModule failed: %v", err)
	}

	// Verify special characters are preserved
	if retrieved.Description != module.Description {
		t.Errorf("Expected description %q, got %q", module.Description, retrieved.Description)
	}

	// Verify the raw data in Redis is valid JSON
	rawData, err := mr.Get("module:complex.module")
	if err != nil {
		t.Fatalf("Failed to get raw data: %v", err)
	}

	var decoded api.Module
	if err := json.Unmarshal([]byte(rawData), &decoded); err != nil {
		t.Fatalf("Raw data is not valid JSON: %v", err)
	}
}
