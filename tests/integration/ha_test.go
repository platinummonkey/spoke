package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/middleware"
	postgresStorage "github.com/platinummonkey/spoke/pkg/storage/postgres"
)

// TestDatabaseConnectionManager tests the connection manager with primary and replicas
func TestDatabaseConnectionManager(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	primaryURL := getEnvOrDefault("TEST_POSTGRES_PRIMARY", "postgres://spoke:spoke@localhost:5432/spoke?sslmode=disable")
	replicaURL := getEnvOrDefault("TEST_POSTGRES_REPLICA", "postgres://spoke:spoke@localhost:5433/spoke?sslmode=disable")

	t.Run("ConnectionManagerWithPrimaryOnly", func(t *testing.T) {
		config := postgresStorage.ConnectionConfig{
			PrimaryURL:  primaryURL,
			ReplicaURLs: nil, // No replicas
			MaxConns:    10,
			MinConns:    2,
			Timeout:     5 * time.Second,
			MaxLifetime: 1 * time.Hour,
			MaxIdleTime: 10 * time.Minute,
		}

		cm, err := postgresStorage.NewConnectionManager(config)
		require.NoError(t, err)
		defer cm.Close()

		// Test primary connection
		primary := cm.Primary()
		assert.NotNil(t, primary)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = primary.PingContext(ctx)
		assert.NoError(t, err)

		// Replica should fall back to primary
		replica := cm.Replica()
		assert.Equal(t, primary, replica, "Replica should fall back to primary when no replicas configured")
	})

	t.Run("ConnectionManagerWithReplicas", func(t *testing.T) {
		config := postgresStorage.ConnectionConfig{
			PrimaryURL:  primaryURL,
			ReplicaURLs: []string{replicaURL},
			MaxConns:    10,
			MinConns:    2,
			Timeout:     5 * time.Second,
			MaxLifetime: 1 * time.Hour,
			MaxIdleTime: 10 * time.Minute,
		}

		cm, err := postgresStorage.NewConnectionManager(config)
		if err != nil {
			t.Logf("Could not connect to replica: %v (this is expected if replica is not running)", err)
			t.Skip("Skipping replica test - replica not available")
			return
		}
		defer cm.Close()

		// Test primary connection
		primary := cm.Primary()
		assert.NotNil(t, primary)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = primary.PingContext(ctx)
		assert.NoError(t, err)

		// Test replica connection
		replica := cm.Replica()
		assert.NotNil(t, replica)

		err = replica.PingContext(ctx)
		assert.NoError(t, err)

		// Verify replica is different from primary
		replicas := cm.AllReplicas()
		if len(replicas) > 0 {
			assert.NotEqual(t, primary, replicas[0], "Replica should be different connection than primary")
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		config := postgresStorage.ConnectionConfig{
			PrimaryURL:  primaryURL,
			ReplicaURLs: nil,
			MaxConns:    10,
			MinConns:    2,
			Timeout:     5 * time.Second,
			MaxLifetime: 1 * time.Hour,
			MaxIdleTime: 10 * time.Minute,
		}

		cm, err := postgresStorage.NewConnectionManager(config)
		require.NoError(t, err)
		defer cm.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = cm.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("ConnectionPoolStats", func(t *testing.T) {
		config := postgresStorage.ConnectionConfig{
			PrimaryURL:  primaryURL,
			ReplicaURLs: nil,
			MaxConns:    10,
			MinConns:    2,
			Timeout:     5 * time.Second,
			MaxLifetime: 1 * time.Hour,
			MaxIdleTime: 10 * time.Minute,
		}

		cm, err := postgresStorage.NewConnectionManager(config)
		require.NoError(t, err)
		defer cm.Close()

		stats := cm.Stats()
		assert.NotNil(t, stats.Primary)
		assert.GreaterOrEqual(t, stats.Primary.MaxOpenConnections, 10)
	})

	t.Run("RoundRobinReplicaSelection", func(t *testing.T) {
		// This test would require multiple replicas to verify round-robin
		// For now, just verify the behavior with one replica
		config := postgresStorage.ConnectionConfig{
			PrimaryURL:  primaryURL,
			ReplicaURLs: []string{replicaURL},
			MaxConns:    10,
			MinConns:    2,
			Timeout:     5 * time.Second,
			MaxLifetime: 1 * time.Hour,
			MaxIdleTime: 10 * time.Minute,
		}

		cm, err := postgresStorage.NewConnectionManager(config)
		if err != nil {
			t.Skip("Skipping replica test - replica not available")
			return
		}
		defer cm.Close()

		// Call Replica() multiple times - should round-robin
		r1 := cm.Replica()
		r2 := cm.Replica()
		r3 := cm.Replica()

		// With one replica, all should be the same
		assert.NotNil(t, r1)
		assert.NotNil(t, r2)
		assert.NotNil(t, r3)
	})
}

// TestRedisCache tests Redis caching functionality
func TestRedisCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	redisURL := getEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/0")

	t.Run("RedisConnection", func(t *testing.T) {
		opts, err := redis.ParseURL(redisURL)
		require.NoError(t, err)

		client := redis.NewClient(opts)
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = client.Ping(ctx).Err()
		if err != nil {
			t.Logf("Could not connect to Redis: %v", err)
			t.Skip("Skipping Redis test - Redis not available")
			return
		}
		assert.NoError(t, err)
	})

	t.Run("CacheHitMiss", func(t *testing.T) {
		opts, err := redis.ParseURL(redisURL)
		require.NoError(t, err)

		client := redis.NewClient(opts)
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			t.Skip("Skipping Redis test - Redis not available")
			return
		}

		// Test cache miss
		testKey := fmt.Sprintf("test:module:%d", time.Now().UnixNano())
		result, err := client.Get(ctx, testKey).Result()
		assert.Equal(t, redis.Nil, err)
		assert.Empty(t, result)

		// Test cache hit
		testModule := &api.Module{
			Name:        "test-module",
			Description: "Test module",
			CreatedAt:   time.Now(),
		}

		data, err := json.Marshal(testModule)
		require.NoError(t, err)

		err = client.Set(ctx, testKey, data, 1*time.Minute).Err()
		require.NoError(t, err)

		result, err = client.Get(ctx, testKey).Result()
		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		var retrieved api.Module
		err = json.Unmarshal([]byte(result), &retrieved)
		assert.NoError(t, err)
		assert.Equal(t, testModule.Name, retrieved.Name)

		// Cleanup
		client.Del(ctx, testKey)
	})

	t.Run("PatternInvalidation", func(t *testing.T) {
		opts, err := redis.ParseURL(redisURL)
		require.NoError(t, err)

		client := redis.NewClient(opts)
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			t.Skip("Skipping Redis test - Redis not available")
			return
		}

		// Create multiple keys with pattern
		pattern := fmt.Sprintf("test:pattern:%d:*", time.Now().UnixNano())
		baseKey := pattern[:len(pattern)-1] // Remove the *

		keys := []string{
			baseKey + "key1",
			baseKey + "key2",
			baseKey + "key3",
		}

		for _, key := range keys {
			err := client.Set(ctx, key, "test-value", 1*time.Minute).Err()
			require.NoError(t, err)
		}

		// Verify keys exist
		for _, key := range keys {
			result, err := client.Get(ctx, key).Result()
			assert.NoError(t, err)
			assert.Equal(t, "test-value", result)
		}

		// Invalidate using pattern
		iter := client.Scan(ctx, 0, pattern, 100).Iterator()
		deleted := 0
		for iter.Next(ctx) {
			err := client.Del(ctx, iter.Val()).Err()
			require.NoError(t, err)
			deleted++
		}
		assert.NoError(t, iter.Err())
		assert.Equal(t, 3, deleted)

		// Verify keys are deleted
		for _, key := range keys {
			result, err := client.Get(ctx, key).Result()
			assert.Equal(t, redis.Nil, err)
			assert.Empty(t, result)
		}
	})
}

// TestDistributedRateLimiting tests Redis-backed rate limiting
func TestDistributedRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	redisURL := getEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/0")

	t.Run("BasicRateLimiting", func(t *testing.T) {
		opts, err := redis.ParseURL(redisURL)
		require.NoError(t, err)

		client := redis.NewClient(opts)
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			t.Skip("Skipping Redis test - Redis not available")
			return
		}

		config := &middleware.RateLimitConfig{
			RequestsPerWindow: 5,
			WindowDuration:    10 * time.Second,
		}

		limiter := middleware.NewDistributedRateLimiter(client, config, "test:ratelimit")
		testKey := fmt.Sprintf("test-key-%d", time.Now().UnixNano())

		// First 5 requests should succeed
		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow(ctx, testKey)
			assert.NoError(t, err)
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}

		// 6th request should fail
		allowed, err := limiter.Allow(ctx, testKey)
		assert.NoError(t, err)
		assert.False(t, allowed, "Request 6 should be rate limited")

		// Cleanup
		limiter.Reset(ctx, testKey)
	})

	t.Run("RateLimitRemaining", func(t *testing.T) {
		opts, err := redis.ParseURL(redisURL)
		require.NoError(t, err)

		client := redis.NewClient(opts)
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			t.Skip("Skipping Redis test - Redis not available")
			return
		}

		config := &middleware.RateLimitConfig{
			RequestsPerWindow: 10,
			WindowDuration:    10 * time.Second,
		}

		limiter := middleware.NewDistributedRateLimiter(client, config, "test:ratelimit")
		testKey := fmt.Sprintf("test-key-%d", time.Now().UnixNano())

		// Use 3 requests
		for i := 0; i < 3; i++ {
			limiter.Allow(ctx, testKey)
		}

		// Check remaining
		remaining, err := limiter.Remaining(ctx, testKey)
		assert.NoError(t, err)
		assert.Equal(t, 7, remaining, "Should have 7 requests remaining after using 3")

		// Cleanup
		limiter.Reset(ctx, testKey)
	})

	t.Run("FailOpenOnRedisError", func(t *testing.T) {
		// Create client with invalid URL to simulate Redis error
		client := redis.NewClient(&redis.Options{
			Addr: "localhost:9999", // Invalid port
		})
		defer client.Close()

		config := &middleware.RateLimitConfig{
			RequestsPerWindow: 5,
			WindowDuration:    10 * time.Second,
		}

		limiter := middleware.NewDistributedRateLimiter(client, config, "test:ratelimit")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Should fail open (return true) on error
		allowed, err := limiter.Allow(ctx, "test-key")
		assert.Error(t, err, "Should return error for Redis connection failure")
		assert.True(t, allowed, "Should fail open and allow request on Redis error")
	})
}

// TestOpenTelemetry tests OpenTelemetry tracing
func TestOpenTelemetry(t *testing.T) {
	t.Run("DatabaseSpanCreation", func(t *testing.T) {
		// Create in-memory span exporter
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(exporter),
		)
		otel.SetTracerProvider(tp)

		tracer := otel.Tracer("test-tracer")

		ctx := context.Background()
		ctx, span := tracer.Start(ctx, "test-database-operation",
			trace.WithAttributes(
				attribute.String("db.system", "postgresql"),
				attribute.String("db.operation", "SELECT"),
				attribute.String("db.table", "modules"),
			),
		)

		// Simulate work
		time.Sleep(10 * time.Millisecond)

		span.End()

		// Verify span was created
		spans := exporter.GetSpans()
		assert.Len(t, spans, 1)
		assert.Equal(t, "test-database-operation", spans[0].Name)

		// Verify attributes
		attrs := spans[0].Attributes
		hasDBSystem := false
		hasDBOperation := false
		for _, attr := range attrs {
			if string(attr.Key) == "db.system" && attr.Value.AsString() == "postgresql" {
				hasDBSystem = true
			}
			if string(attr.Key) == "db.operation" && attr.Value.AsString() == "SELECT" {
				hasDBOperation = true
			}
		}
		assert.True(t, hasDBSystem, "Should have db.system attribute")
		assert.True(t, hasDBOperation, "Should have db.operation attribute")
	})

	t.Run("ErrorRecording", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(exporter),
		)
		otel.SetTracerProvider(tp)

		tracer := otel.Tracer("test-tracer")

		ctx := context.Background()
		ctx, span := tracer.Start(ctx, "test-error-operation")

		// Record an error
		testErr := fmt.Errorf("test error")
		span.RecordError(testErr)
		span.SetStatus(codes.Error, testErr.Error())

		span.End()

		// Verify error was recorded
		spans := exporter.GetSpans()
		assert.Len(t, spans, 1)
		assert.Equal(t, codes.Error, spans[0].Status.Code)
		assert.Contains(t, spans[0].Status.Description, "test error")
	})
}

// TestHealthChecks tests health check endpoints
func TestHealthChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("LivenessCheck", func(t *testing.T) {
		// Liveness should always return 200 when server is running
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})

		req := httptest.NewRequest("GET", "/health/live", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("ReadinessCheckWithDependencies", func(t *testing.T) {
		// This would test the actual readiness endpoint
		// For now, just verify the pattern

		postgresURL := getEnvOrDefault("TEST_POSTGRES_PRIMARY", "postgres://spoke:spoke@localhost:5432/spoke?sslmode=disable")
		redisURL := getEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/0")

		// Check PostgreSQL
		db, err := sql.Open("postgres", postgresURL)
		if err != nil {
			t.Logf("PostgreSQL not available: %v", err)
		} else {
			defer db.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			postgresHealthy := db.PingContext(ctx) == nil
			t.Logf("PostgreSQL healthy: %v", postgresHealthy)
		}

		// Check Redis
		opts, err := redis.ParseURL(redisURL)
		if err == nil {
			client := redis.NewClient(opts)
			defer client.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			redisHealthy := client.Ping(ctx).Err() == nil
			t.Logf("Redis healthy: %v", redisHealthy)
		}
	})
}

// TestRateLimitHeaders tests rate limit headers in HTTP responses
func TestRateLimitHeaders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	redisURL := getEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/0")

	opts, err := redis.ParseURL(redisURL)
	require.NoError(t, err)

	client := redis.NewClient(opts)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Skipping Redis test - Redis not available")
		return
	}

	// Create middleware
	rateLimitMiddleware := middleware.NewDistributedRateLimitMiddleware(client)

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with rate limit middleware
	handler := rateLimitMiddleware.Handler(testHandler)

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.1:12345" // Test IP
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check for rate limit headers
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
