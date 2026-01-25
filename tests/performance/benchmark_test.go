package performance

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/storage"
	postgresStorage "github.com/platinummonkey/spoke/pkg/storage/postgres"
)

// BenchmarkModuleCreation benchmarks module creation in PostgreSQL
func BenchmarkModuleCreation(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	config := getTestStorageConfig()
	store, err := postgresStorage.NewPostgresStorage(config)
	if err != nil {
		b.Skipf("Could not create storage: %v", err)
		return
	}
	defer store.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		module := &api.Module{
			Name:        fmt.Sprintf("benchmark-module-%d", i),
			Description: "Benchmark test module",
			CreatedAt:   time.Now(),
		}

		if err := store.CreateModuleContext(ctx, module); err != nil {
			b.Errorf("Failed to create module: %v", err)
		}
	}
}

// BenchmarkModuleRetrievalWithCache benchmarks module retrieval with Redis cache
func BenchmarkModuleRetrievalWithCache(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	config := getTestStorageConfig()
	config.CacheEnabled = true

	store, err := postgresStorage.NewPostgresStorage(config)
	if err != nil {
		b.Skipf("Could not create storage: %v", err)
		return
	}
	defer store.Close()

	// Create a test module
	ctx := context.Background()
	module := &api.Module{
		Name:        "cache-benchmark-module",
		Description: "Cache benchmark test",
		CreatedAt:   time.Now(),
	}

	if err := store.CreateModuleContext(ctx, module); err != nil {
		b.Fatalf("Failed to create test module: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetModuleContext(ctx, "cache-benchmark-module")
		if err != nil {
			b.Errorf("Failed to get module: %v", err)
		}
	}
}

// BenchmarkModuleRetrievalWithoutCache benchmarks module retrieval without cache
func BenchmarkModuleRetrievalWithoutCache(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	config := getTestStorageConfig()
	config.CacheEnabled = false

	store, err := postgresStorage.NewPostgresStorage(config)
	if err != nil {
		b.Skipf("Could not create storage: %v", err)
		return
	}
	defer store.Close()

	// Create a test module
	ctx := context.Background()
	module := &api.Module{
		Name:        "nocache-benchmark-module",
		Description: "No cache benchmark test",
		CreatedAt:   time.Now(),
	}

	if err := store.CreateModuleContext(ctx, module); err != nil {
		b.Fatalf("Failed to create test module: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetModuleContext(ctx, "nocache-benchmark-module")
		if err != nil {
			b.Errorf("Failed to get module: %v", err)
		}
	}
}

// BenchmarkVersionCreation benchmarks version creation
func BenchmarkVersionCreation(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	config := getTestStorageConfig()
	store, err := postgresStorage.NewPostgresStorage(config)
	if err != nil {
		b.Skipf("Could not create storage: %v", err)
		return
	}
	defer store.Close()

	// Create a test module first
	ctx := context.Background()
	module := &api.Module{
		Name:        "version-benchmark-module",
		Description: "Version benchmark test",
		CreatedAt:   time.Now(),
	}

	if err := store.CreateModuleContext(ctx, module); err != nil {
		b.Fatalf("Failed to create test module: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		version := &api.Version{
			ModuleName: "version-benchmark-module",
			Version:    fmt.Sprintf("v1.0.%d", i),
			CreatedAt:  time.Now(),
		}

		if err := store.CreateVersionContext(ctx, version); err != nil {
			b.Errorf("Failed to create version: %v", err)
		}
	}
}

// BenchmarkRedisSet benchmarks Redis SET operations
func BenchmarkRedisSet(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	redisURL := getEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/0")
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		b.Skipf("Invalid Redis URL: %v", err)
		return
	}

	client := redis.NewClient(opts)
	defer client.Close()

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark:key:%d", i)
		if err := client.Set(ctx, key, "benchmark-value", 1*time.Minute).Err(); err != nil {
			b.Errorf("Failed to set key: %v", err)
		}
	}
}

// BenchmarkRedisGet benchmarks Redis GET operations
func BenchmarkRedisGet(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	redisURL := getEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/0")
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		b.Skipf("Invalid Redis URL: %v", err)
		return
	}

	client := redis.NewClient(opts)
	defer client.Close()

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
		return
	}

	// Pre-populate cache
	testKey := "benchmark:get:key"
	if err := client.Set(ctx, testKey, "benchmark-value", 5*time.Minute).Err(); err != nil {
		b.Fatalf("Failed to setup test: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.Get(ctx, testKey).Result(); err != nil && err != redis.Nil {
			b.Errorf("Failed to get key: %v", err)
		}
	}
}

// BenchmarkDatabaseQuery benchmarks direct database queries
func BenchmarkDatabaseQuery(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	config := getTestStorageConfig()
	store, err := postgresStorage.NewPostgresStorage(config)
	if err != nil {
		b.Skipf("Could not create storage: %v", err)
		return
	}
	defer store.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.ListModulesContext(ctx)
		if err != nil {
			b.Errorf("Failed to list modules: %v", err)
		}
	}
}

// BenchmarkConnectionPoolPerformance benchmarks connection pool behavior
func BenchmarkConnectionPoolPerformance(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	primaryURL := getEnvOrDefault("TEST_POSTGRES_PRIMARY", "postgres://spoke:spoke@localhost:5432/spoke?sslmode=disable")

	config := postgresStorage.ConnectionConfig{
		PrimaryURL:  primaryURL,
		ReplicaURLs: nil,
		MaxConns:    50,
		MinConns:    5,
		Timeout:     5 * time.Second,
		MaxLifetime: 1 * time.Hour,
		MaxIdleTime: 10 * time.Minute,
	}

	cm, err := postgresStorage.NewConnectionManager(config)
	if err != nil {
		b.Skipf("Could not create connection manager: %v", err)
		return
	}
	defer cm.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			db := cm.Primary()
			if err := db.PingContext(ctx); err != nil {
				b.Errorf("Ping failed: %v", err)
			}
		}
	})
}

// BenchmarkReplicaRoundRobin benchmarks replica selection performance
func BenchmarkReplicaRoundRobin(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	primaryURL := getEnvOrDefault("TEST_POSTGRES_PRIMARY", "postgres://spoke:spoke@localhost:5432/spoke?sslmode=disable")
	replicaURL := getEnvOrDefault("TEST_POSTGRES_REPLICA", "postgres://spoke:spoke@localhost:5433/spoke?sslmode=disable")

	config := postgresStorage.ConnectionConfig{
		PrimaryURL:  primaryURL,
		ReplicaURLs: []string{replicaURL},
		MaxConns:    50,
		MinConns:    5,
		Timeout:     5 * time.Second,
		MaxLifetime: 1 * time.Hour,
		MaxIdleTime: 10 * time.Minute,
	}

	cm, err := postgresStorage.NewConnectionManager(config)
	if err != nil {
		b.Skipf("Could not create connection manager: %v", err)
		return
	}
	defer cm.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cm.Replica()
		}
	})
}

// Helper functions

func getTestStorageConfig() storage.Config {
	return storage.Config{
		PostgresURL:         getEnvOrDefault("TEST_POSTGRES_PRIMARY", "postgres://spoke:spoke@localhost:5432/spoke?sslmode=disable"),
		PostgresReplicaURLs: getEnvOrDefault("TEST_POSTGRES_REPLICAS", ""),
		PostgresMaxConns:    25,
		PostgresMinConns:    5,
		PostgresTimeout:     5 * time.Second,
		CacheEnabled:        true,
		RedisURL:            getEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/0"),
		RedisMaxRetries:     3,
		RedisPoolSize:       10,
		CacheTTL: map[string]time.Duration{
			"module":  5 * time.Minute,
			"version": 10 * time.Minute,
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
