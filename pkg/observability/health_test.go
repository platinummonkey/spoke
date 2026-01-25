package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthChecker_Liveness(t *testing.T) {
	checker := NewHealthChecker(nil, nil)

	req := httptest.NewRequest("GET", "/health/live", nil)
	rr := httptest.NewRecorder()

	checker.Liveness(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Liveness check returned wrong status code: got %v want %v", status, http.StatusOK)
	}
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
	})

	t.Run("with dependencies", func(t *testing.T) {
		// This would require mock database and Redis
		// In integration tests, use testcontainers
		t.Skip("Requires actual database and Redis")
	})
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
	})

	t.Run("unhealthy readiness", func(t *testing.T) {
		// This would require a failing dependency
		t.Skip("Requires mock failing dependency")
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
