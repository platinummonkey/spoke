package observability

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestNewHealthChecker(t *testing.T) {
	t.Run("with nil dependencies", func(t *testing.T) {
		checker := NewHealthChecker(nil, nil)
		if checker == nil {
			t.Fatal("Expected non-nil checker")
		}
		if checker.db != nil {
			t.Error("Expected nil db")
		}
		if checker.redis != nil {
			t.Error("Expected nil redis")
		}
	})

	t.Run("with database", func(t *testing.T) {
		db, _, err := sqlmock.New()
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		checker := NewHealthChecker(db, nil)
		if checker.db == nil {
			t.Error("Expected non-nil db")
		}
	})

	t.Run("with redis", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("Failed to start miniredis: %v", err)
		}
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
		defer redisClient.Close()

		checker := NewHealthChecker(nil, redisClient)
		if checker.redis == nil {
			t.Error("Expected non-nil redis")
		}
	})
}

func TestHealthChecker_Liveness(t *testing.T) {
	checker := NewHealthChecker(nil, nil)

	req := httptest.NewRequest("GET", "/health/live", nil)
	rr := httptest.NewRecorder()

	checker.Liveness(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Liveness check returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != StatusHealthy {
		t.Errorf("Expected status %s, got %v", StatusHealthy, response["status"])
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected timestamp in response")
	}
}

func TestHealthChecker_Readiness(t *testing.T) {
	t.Run("healthy readiness", func(t *testing.T) {
		checker := NewHealthChecker(nil, nil)

		req := httptest.NewRequest("GET", "/health/ready", nil)
		rr := httptest.NewRecorder()

		checker.Readiness(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Readiness check returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("unhealthy readiness with failed database", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		mock.ExpectPing().WillReturnError(errors.New("connection failed"))

		checker := NewHealthChecker(db, nil)

		req := httptest.NewRequest("GET", "/health/ready", nil)
		rr := httptest.NewRecorder()

		checker.Readiness(rr, req)

		if status := rr.Code; status != http.StatusServiceUnavailable {
			t.Errorf("Expected status %v for unhealthy, got %v", http.StatusServiceUnavailable, status)
		}

		var response HealthStatus
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != StatusUnhealthy {
			t.Errorf("Expected status %s, got %s", StatusUnhealthy, response.Status)
		}
	})

	t.Run("degraded readiness with healthy database and failed redis", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Create a Redis client pointing to a non-existent server
		redisClient := redis.NewClient(&redis.Options{
			Addr: "localhost:9999",
		})
		defer redisClient.Close()

		checker := NewHealthChecker(db, redisClient)

		req := httptest.NewRequest("GET", "/health/ready", nil)
		rr := httptest.NewRecorder()

		checker.Readiness(rr, req)

		// Should return 200 for degraded, not 503
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status %v for degraded, got %v", http.StatusOK, status)
		}

		var response HealthStatus
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != StatusDegraded {
			t.Errorf("Expected status %s, got %s", StatusDegraded, response.Status)
		}
	})
}

func TestHealthChecker_Check(t *testing.T) {
	t.Run("no dependencies", func(t *testing.T) {
		checker := NewHealthChecker(nil, nil)
		ctx := context.Background()

		status := checker.Check(ctx)

		if status.Status != StatusHealthy {
			t.Errorf("Expected status %s, got %s", StatusHealthy, status.Status)
		}

		if len(status.Dependencies) != 0 {
			t.Errorf("Expected 0 dependencies, got %d", len(status.Dependencies))
		}

		if status.Version != "1.0.0" {
			t.Errorf("Expected version 1.0.0, got %s", status.Version)
		}

		if status.Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp")
		}
	})

	t.Run("with healthy database", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		// Set max open connections to avoid pool exhaustion
		db.SetMaxOpenConns(10)

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		checker := NewHealthChecker(db, nil)
		ctx := context.Background()

		status := checker.Check(ctx)

		if len(status.Dependencies) != 1 {
			t.Errorf("Expected 1 dependency, got %d", len(status.Dependencies))
		}

		dbStatus, ok := status.Dependencies["database"]
		if !ok {
			t.Fatal("Expected database dependency")
		}

		// Database should be healthy (or degraded due to pool, but not unhealthy)
		if dbStatus.Status == StatusUnhealthy {
			t.Errorf("Expected database not unhealthy, got %s with message: %s", dbStatus.Status, dbStatus.Message)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unmet expectations: %v", err)
		}
	})

	t.Run("with unhealthy database", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		mock.ExpectPing().WillReturnError(errors.New("connection refused"))

		checker := NewHealthChecker(db, nil)
		ctx := context.Background()

		status := checker.Check(ctx)

		if status.Status != StatusUnhealthy {
			t.Errorf("Expected status %s, got %s", StatusUnhealthy, status.Status)
		}

		dbStatus := status.Dependencies["database"]
		if dbStatus.Status != StatusUnhealthy {
			t.Errorf("Expected database status %s, got %s", StatusUnhealthy, dbStatus.Status)
		}

		if dbStatus.Message == "" {
			t.Error("Expected error message for unhealthy database")
		}
	})

	t.Run("with healthy redis", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("Failed to start miniredis: %v", err)
		}
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
		defer redisClient.Close()

		checker := NewHealthChecker(nil, redisClient)
		ctx := context.Background()

		status := checker.Check(ctx)

		if status.Status != StatusHealthy {
			t.Errorf("Expected status %s, got %s", StatusHealthy, status.Status)
		}

		redisStatus, ok := status.Dependencies["redis"]
		if !ok {
			t.Fatal("Expected redis dependency")
		}

		if redisStatus.Status != StatusHealthy {
			t.Errorf("Expected redis status %s, got %s", StatusHealthy, redisStatus.Status)
		}

		if redisStatus.Latency == 0 {
			t.Error("Expected non-zero latency")
		}
	})

	t.Run("with unhealthy redis causes degraded", func(t *testing.T) {
		redisClient := redis.NewClient(&redis.Options{
			Addr: "localhost:9999",
		})
		defer redisClient.Close()

		checker := NewHealthChecker(nil, redisClient)
		ctx := context.Background()

		status := checker.Check(ctx)

		// Redis failure causes degraded, not unhealthy
		if status.Status != StatusDegraded {
			t.Errorf("Expected status %s, got %s", StatusDegraded, status.Status)
		}

		redisStatus := status.Dependencies["redis"]
		if redisStatus.Status != StatusUnhealthy {
			t.Errorf("Expected redis status %s, got %s", StatusUnhealthy, redisStatus.Status)
		}
	})

	t.Run("with database and redis both healthy", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		// Set max open connections to avoid pool exhaustion
		db.SetMaxOpenConns(10)

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("Failed to start miniredis: %v", err)
		}
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
		defer redisClient.Close()

		checker := NewHealthChecker(db, redisClient)
		ctx := context.Background()

		status := checker.Check(ctx)

		if len(status.Dependencies) != 2 {
			t.Errorf("Expected 2 dependencies, got %d", len(status.Dependencies))
		}

		// Check that neither dependency is unhealthy
		if dbStatus, ok := status.Dependencies["database"]; ok && dbStatus.Status == StatusUnhealthy {
			t.Errorf("Database should not be unhealthy, got: %s", dbStatus.Message)
		}
		if redisStatus, ok := status.Dependencies["redis"]; ok && redisStatus.Status == StatusUnhealthy {
			t.Errorf("Redis should not be unhealthy, got: %s", redisStatus.Message)
		}
	})
}

func TestHealthChecker_checkDatabase(t *testing.T) {
	t.Run("successful ping and query", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		// Set max open connections to avoid pool exhaustion
		db.SetMaxOpenConns(10)

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		checker := NewHealthChecker(db, nil)
		ctx := context.Background()

		status := checker.checkDatabase(ctx)

		// Status should not be unhealthy
		if status.Status == StatusUnhealthy {
			t.Errorf("Expected status not unhealthy, got %s with message: %s", status.Status, status.Message)
		}

		if status.Latency == 0 {
			t.Error("Expected non-zero latency")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unmet expectations: %v", err)
		}
	})

	t.Run("ping fails", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		mock.ExpectPing().WillReturnError(errors.New("connection refused"))

		checker := NewHealthChecker(db, nil)
		ctx := context.Background()

		status := checker.checkDatabase(ctx)

		if status.Status != StatusUnhealthy {
			t.Errorf("Expected status %s, got %s", StatusUnhealthy, status.Status)
		}

		if status.Message == "" {
			t.Error("Expected error message")
		}

		if status.Message != "connection refused" {
			t.Errorf("Expected 'connection refused', got %s", status.Message)
		}
	})

	t.Run("query fails", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectQuery("SELECT 1").WillReturnError(errors.New("query timeout"))

		checker := NewHealthChecker(db, nil)
		ctx := context.Background()

		status := checker.checkDatabase(ctx)

		if status.Status != StatusUnhealthy {
			t.Errorf("Expected status %s, got %s", StatusUnhealthy, status.Status)
		}

		if !contains(status.Message, "query failed") {
			t.Errorf("Expected message to contain 'query failed', got %s", status.Message)
		}
	})

	t.Run("checks pool statistics", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		// Set max open connections
		db.SetMaxOpenConns(10)

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		checker := NewHealthChecker(db, nil)
		ctx := context.Background()

		status := checker.checkDatabase(ctx)

		// Verify the check completed
		if status.Status == StatusUnhealthy && status.Message != "" {
			t.Errorf("Unexpected unhealthy status: %s", status.Message)
		}

		// Verify latency was measured
		if status.Latency == 0 {
			t.Error("Expected non-zero latency")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unmet expectations: %v", err)
		}
	})
}

func TestHealthChecker_checkRedis(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("Failed to start miniredis: %v", err)
		}
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
		defer redisClient.Close()

		checker := NewHealthChecker(nil, redisClient)
		ctx := context.Background()

		status := checker.checkRedis(ctx)

		if status.Status != StatusHealthy {
			t.Errorf("Expected status %s, got %s", StatusHealthy, status.Status)
		}

		if status.Message != "" {
			t.Errorf("Expected empty message for healthy, got %s", status.Message)
		}

		if status.Latency == 0 {
			t.Error("Expected non-zero latency")
		}

		if status.Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp")
		}
	})

	t.Run("ping fails", func(t *testing.T) {
		redisClient := redis.NewClient(&redis.Options{
			Addr: "localhost:9999",
		})
		defer redisClient.Close()

		checker := NewHealthChecker(nil, redisClient)
		ctx := context.Background()

		status := checker.checkRedis(ctx)

		if status.Status != StatusUnhealthy {
			t.Errorf("Expected status %s, got %s", StatusUnhealthy, status.Status)
		}

		if status.Message == "" {
			t.Error("Expected error message")
		}
	})

	t.Run("info command succeeds", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("Failed to start miniredis: %v", err)
		}
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
		defer redisClient.Close()

		checker := NewHealthChecker(nil, redisClient)
		ctx := context.Background()

		status := checker.checkRedis(ctx)

		// Even if Info fails, status should still be healthy if Ping succeeded
		if status.Status != StatusHealthy {
			t.Errorf("Expected status %s, got %s", StatusHealthy, status.Status)
		}
	})
}

func TestRegisterHealthRoutes(t *testing.T) {
	t.Run("registers all routes", func(t *testing.T) {
		mux := http.NewServeMux()
		checker := NewHealthChecker(nil, nil)

		RegisterHealthRoutes(mux, checker)

		// Test /health route
		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("/health returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Test /health/live route
		req = httptest.NewRequest("GET", "/health/live", nil)
		rr = httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("/health/live returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Test /health/ready route
		req = httptest.NewRequest("GET", "/health/ready", nil)
		rr = httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("/health/ready returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("routes work with dependencies", func(t *testing.T) {
		mux := http.NewServeMux()

		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("Failed to create mock db: %v", err)
		}
		defer db.Close()

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		checker := NewHealthChecker(db, nil)
		RegisterHealthRoutes(mux, checker)

		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("/health with db returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response HealthStatus
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if _, ok := response.Dependencies["database"]; !ok {
			t.Error("Expected database dependency in response")
		}
	})
}

func TestHealthStatus_Values(t *testing.T) {
	t.Run("status constants", func(t *testing.T) {
		if StatusHealthy != "healthy" {
			t.Errorf("Expected StatusHealthy to be 'healthy', got %s", StatusHealthy)
		}
		if StatusDegraded != "degraded" {
			t.Errorf("Expected StatusDegraded to be 'degraded', got %s", StatusDegraded)
		}
		if StatusUnhealthy != "unhealthy" {
			t.Errorf("Expected StatusUnhealthy to be 'unhealthy', got %s", StatusUnhealthy)
		}
	})
}

func TestDependencyStatus_Latency(t *testing.T) {
	status := DependencyStatus{
		Status:    StatusHealthy,
		Latency:   50 * time.Millisecond,
		Timestamp: time.Now(),
	}

	if status.Latency != 50*time.Millisecond {
		t.Errorf("Expected latency 50ms, got %v", status.Latency)
	}
}

func TestHealthStatus_JSON(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		original := HealthStatus{
			Status:    StatusHealthy,
			Timestamp: time.Now().Round(time.Second),
			Version:   "1.0.0",
			Dependencies: map[string]DependencyStatus{
				"database": {
					Status:    StatusHealthy,
					Message:   "OK",
					Latency:   10 * time.Millisecond,
					Timestamp: time.Now().Round(time.Second),
				},
			},
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded HealthStatus
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Status != original.Status {
			t.Errorf("Status mismatch: got %s, want %s", decoded.Status, original.Status)
		}

		if decoded.Version != original.Version {
			t.Errorf("Version mismatch: got %s, want %s", decoded.Version, original.Version)
		}
	})
}

func TestDependencyStatus_JSON(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		original := DependencyStatus{
			Status:    StatusDegraded,
			Message:   "High latency",
			Latency:   500 * time.Millisecond,
			Timestamp: time.Now().Round(time.Second),
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded DependencyStatus
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Status != original.Status {
			t.Errorf("Status mismatch: got %s, want %s", decoded.Status, original.Status)
		}

		if decoded.Message != original.Message {
			t.Errorf("Message mismatch: got %s, want %s", decoded.Message, original.Message)
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		len(s) > len(substr)+1 && s[1:len(substr)+1] == substr ||
		len(s) > len(substr)*2 && s[len(substr):len(substr)*2] == substr ||
		findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
