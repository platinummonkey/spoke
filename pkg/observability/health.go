package observability

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

// HealthChecker provides health check functionality
type HealthChecker struct {
	db    *sql.DB
	redis *redis.Client
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *sql.DB, redis *redis.Client) *HealthChecker {
	return &HealthChecker{
		db:    db,
		redis: redis,
	}
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status      string                   `json:"status"`
	Timestamp   time.Time                `json:"timestamp"`
	Version     string                   `json:"version,omitempty"`
	Dependencies map[string]DependencyStatus `json:"dependencies,omitempty"`
}

// DependencyStatus represents the health of a single dependency
type DependencyStatus struct {
	Status    string        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Latency   time.Duration `json:"latency_ms,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

const (
	StatusHealthy   = "healthy"
	StatusDegraded  = "degraded"
	StatusUnhealthy = "unhealthy"
)

// Liveness returns a simple liveness probe (always returns 200 if server is running)
func (h *HealthChecker) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    StatusHealthy,
		"timestamp": time.Now(),
	})
}

// Readiness returns a readiness probe (checks all dependencies)
func (h *HealthChecker) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := h.Check(ctx)

	w.Header().Set("Content-Type", "application/json")

	// Return 503 if unhealthy, 200 if healthy or degraded
	if status.Status == StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(status)
}

// Check performs a comprehensive health check
func (h *HealthChecker) Check(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Status:       StatusHealthy,
		Timestamp:    time.Now(),
		Version:      "1.0.0", // TODO: Get from build info
		Dependencies: make(map[string]DependencyStatus),
	}

	// Check database
	if h.db != nil {
		dbStatus := h.checkDatabase(ctx)
		status.Dependencies["database"] = dbStatus
		if dbStatus.Status == StatusUnhealthy {
			status.Status = StatusUnhealthy
		} else if dbStatus.Status == StatusDegraded && status.Status != StatusUnhealthy {
			status.Status = StatusDegraded
		}
	}

	// Check Redis
	if h.redis != nil {
		redisStatus := h.checkRedis(ctx)
		status.Dependencies["redis"] = redisStatus
		if redisStatus.Status == StatusUnhealthy {
			// Redis is optional - degraded if Redis is down
			if status.Status != StatusUnhealthy {
				status.Status = StatusDegraded
			}
		}
	}

	// Check S3 would go here
	// status.Dependencies["s3"] = h.checkS3(ctx)

	return status
}

// checkDatabase checks PostgreSQL health
func (h *HealthChecker) checkDatabase(ctx context.Context) DependencyStatus {
	start := time.Now()
	status := DependencyStatus{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
	}

	// Ping database with context
	err := h.db.PingContext(ctx)
	status.Latency = time.Since(start)

	if err != nil {
		status.Status = StatusUnhealthy
		status.Message = err.Error()
		return status
	}

	// Check if we can run a simple query
	var count int
	err = h.db.QueryRowContext(ctx, "SELECT 1").Scan(&count)
	if err != nil {
		status.Status = StatusUnhealthy
		status.Message = "query failed: " + err.Error()
		return status
	}

	// Check connection pool stats
	stats := h.db.Stats()
	if stats.OpenConnections >= stats.MaxOpenConnections {
		status.Status = StatusDegraded
		status.Message = "connection pool exhausted"
	}

	return status
}

// checkRedis checks Redis health
func (h *HealthChecker) checkRedis(ctx context.Context) DependencyStatus {
	start := time.Now()
	status := DependencyStatus{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
	}

	// Ping Redis
	err := h.redis.Ping(ctx).Err()
	status.Latency = time.Since(start)

	if err != nil {
		status.Status = StatusUnhealthy
		status.Message = err.Error()
		return status
	}

	// Check memory usage (optional)
	info, err := h.redis.Info(ctx, "memory").Result()
	if err == nil {
		// Parse memory info if needed
		_ = info
	}

	return status
}

// RegisterHealthRoutes registers health check endpoints
func RegisterHealthRoutes(mux *http.ServeMux, checker *HealthChecker) {
	mux.HandleFunc("/health", checker.Readiness)
	mux.HandleFunc("/health/live", checker.Liveness)
	mux.HandleFunc("/health/ready", checker.Readiness)
}
