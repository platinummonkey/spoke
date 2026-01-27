package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// ConnectionManager manages PostgreSQL primary and read replica connections
type ConnectionManager struct {
	primary  *sql.DB
	replicas []*sql.DB
	current  uint32 // Atomic counter for round-robin selection
	mu       sync.RWMutex
	config   ConnectionConfig
}

// ConnectionConfig holds database connection configuration
type ConnectionConfig struct {
	PrimaryURL   string
	ReplicaURLs  []string
	MaxConns     int
	MinConns     int
	Timeout      time.Duration
	MaxLifetime  time.Duration
	MaxIdleTime  time.Duration
}

// NewConnectionManager creates a new connection manager with primary and replicas
func NewConnectionManager(config ConnectionConfig) (*ConnectionManager, error) {
	cm := &ConnectionManager{
		config:   config,
		replicas: make([]*sql.DB, 0),
	}

	// Connect to primary
	primary, err := sql.Open("postgres", config.PrimaryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open primary connection: %w", err)
	}

	// Configure primary connection pool
	primary.SetMaxOpenConns(config.MaxConns)
	primary.SetMaxIdleConns(config.MinConns)
	primary.SetConnMaxLifetime(config.MaxLifetime)
	primary.SetConnMaxIdleTime(config.MaxIdleTime)

	// Test primary connection
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	if err := primary.PingContext(ctx); err != nil {
		primary.Close()
		return nil, fmt.Errorf("failed to ping primary: %w", err)
	}

	cm.primary = primary

	// Connect to replicas (if configured)
	for i, replicaURL := range config.ReplicaURLs {
		replica, err := sql.Open("postgres", replicaURL)
		if err != nil {
			// Log error but continue (replicas are optional)
			fmt.Printf("Warning: failed to open replica %d: %v\n", i, err)
			continue
		}

		// Configure replica connection pool (slightly smaller than primary)
		replicaMaxConns := config.MaxConns / 2
		if replicaMaxConns < 2 {
			replicaMaxConns = 2
		}
		replica.SetMaxOpenConns(replicaMaxConns)
		replica.SetMaxIdleConns(config.MinConns)
		replica.SetConnMaxLifetime(config.MaxLifetime)
		replica.SetConnMaxIdleTime(config.MaxIdleTime)

		// Test replica connection
		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
		err = replica.PingContext(ctx)
		cancel()

		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to ping replica %d: %v\n", i, err)
			replica.Close()
			continue
		}

		cm.replicas = append(cm.replicas, replica)
	}

	if len(cm.replicas) > 0 {
		fmt.Printf("Connection manager initialized with 1 primary and %d replicas\n", len(cm.replicas))
	} else {
		fmt.Println("Connection manager initialized with primary only (no replicas)")
	}

	return cm, nil
}

// Primary returns the primary database connection (for writes)
func (cm *ConnectionManager) Primary() *sql.DB {
	return cm.primary
}

// Replica returns a read replica using round-robin selection
// Falls back to primary if no replicas are available
func (cm *ConnectionManager) Replica() *sql.DB {
	cm.mu.RLock()
	replicaCount := len(cm.replicas)
	cm.mu.RUnlock()

	if replicaCount == 0 {
		// No replicas available, use primary
		return cm.primary
	}

	// Round-robin selection using atomic counter
	index := atomic.AddUint32(&cm.current, 1)
	replicaIndex := int(index % uint32(replicaCount))

	cm.mu.RLock()
	replica := cm.replicas[replicaIndex]
	cm.mu.RUnlock()

	return replica
}

// AllReplicas returns all replica connections
func (cm *ConnectionManager) AllReplicas() []*sql.DB {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	replicas := make([]*sql.DB, len(cm.replicas))
	copy(replicas, cm.replicas)
	return replicas
}

// HealthCheck checks the health of primary and all replicas
func (cm *ConnectionManager) HealthCheck(ctx context.Context) error {
	// Check primary
	if err := cm.primary.PingContext(ctx); err != nil {
		return fmt.Errorf("primary unhealthy: %w", err)
	}

	// Check replicas
	cm.mu.RLock()
	replicas := make([]*sql.DB, len(cm.replicas))
	copy(replicas, cm.replicas)
	cm.mu.RUnlock()

	var unhealthy []string
	for i, replica := range replicas {
		if err := replica.PingContext(ctx); err != nil {
			unhealthy = append(unhealthy, fmt.Sprintf("replica-%d", i))
		}
	}

	if len(unhealthy) > 0 && len(unhealthy) == len(replicas) {
		// All replicas are down, but primary is up (degraded state)
		return fmt.Errorf("all replicas unhealthy: %s", strings.Join(unhealthy, ", "))
	}

	return nil
}

// Stats returns connection pool statistics for primary and replicas
func (cm *ConnectionManager) Stats() ConnectionStats {
	stats := ConnectionStats{
		Primary: cm.primary.Stats(),
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats.Replicas = make([]sql.DBStats, len(cm.replicas))
	for i, replica := range cm.replicas {
		stats.Replicas[i] = replica.Stats()
	}

	return stats
}

// ConnectionStats holds statistics for all database connections
type ConnectionStats struct {
	Primary  sql.DBStats
	Replicas []sql.DBStats
}

// RemoveUnhealthyReplicas removes replicas that fail health checks
// This is useful for automatic failover scenarios
func (cm *ConnectionManager) RemoveUnhealthyReplicas(ctx context.Context) int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	healthy := make([]*sql.DB, 0, len(cm.replicas))
	removed := 0

	for _, replica := range cm.replicas {
		if err := replica.PingContext(ctx); err != nil {
			// Close unhealthy replica
			replica.Close()
			removed++
		} else {
			healthy = append(healthy, replica)
		}
	}

	cm.replicas = healthy
	return removed
}

// AddReplica adds a new replica connection at runtime
func (cm *ConnectionManager) AddReplica(replicaURL string) error {
	replica, err := sql.Open("postgres", replicaURL)
	if err != nil {
		return fmt.Errorf("failed to open replica connection: %w", err)
	}

	// Configure connection pool
	replicaMaxConns := cm.config.MaxConns / 2
	if replicaMaxConns < 2 {
		replicaMaxConns = 2
	}
	replica.SetMaxOpenConns(replicaMaxConns)
	replica.SetMaxIdleConns(cm.config.MinConns)
	replica.SetConnMaxLifetime(cm.config.MaxLifetime)
	replica.SetConnMaxIdleTime(cm.config.MaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cm.config.Timeout)
	defer cancel()

	if err := replica.PingContext(ctx); err != nil {
		replica.Close()
		return fmt.Errorf("failed to ping replica: %w", err)
	}

	cm.mu.Lock()
	cm.replicas = append(cm.replicas, replica)
	cm.mu.Unlock()

	return nil
}

// Close closes all database connections
func (cm *ConnectionManager) Close() error {
	var errs []error

	// Close primary
	if err := cm.primary.Close(); err != nil {
		errs = append(errs, fmt.Errorf("primary close error: %w", err))
	}

	// Close replicas
	cm.mu.Lock()
	replicas := cm.replicas
	cm.replicas = nil
	cm.mu.Unlock()

	for i, replica := range replicas {
		if err := replica.Close(); err != nil {
			errs = append(errs, fmt.Errorf("replica-%d close error: %w", i, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("connection close errors: %v", errs)
	}

	return nil
}

// StartHealthCheckRoutine starts a background goroutine to check replica health
// and remove unhealthy replicas automatically
func (cm *ConnectionManager) StartHealthCheckRoutine(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()

		// Recover from panics to prevent crashing the process
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[HealthCheckRoutine] PANIC: %v\n%s\n", r, debug.Stack())
			}
		}()

		for {
			select {
			case <-ticker.C:
				checkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				removed := cm.RemoveUnhealthyReplicas(checkCtx)
				cancel()

				if removed > 0 {
					fmt.Printf("Removed %d unhealthy replicas\n", removed)
				}

			case <-ctx.Done():
				return
			}
		}
	}()
}

// ParseReplicaURLs parses a comma-separated list of replica URLs
func ParseReplicaURLs(replicaURLsStr string) []string {
	if replicaURLsStr == "" {
		return nil
	}

	urls := strings.Split(replicaURLsStr, ",")
	result := make([]string, 0, len(urls))

	for _, url := range urls {
		trimmed := strings.TrimSpace(url)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
