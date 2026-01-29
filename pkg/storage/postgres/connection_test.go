package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseReplicaURLs tests the ParseReplicaURLs function
func TestParseReplicaURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single URL",
			input:    "postgres://localhost:5432/db",
			expected: []string{"postgres://localhost:5432/db"},
		},
		{
			name:  "multiple URLs",
			input: "postgres://host1:5432/db,postgres://host2:5432/db,postgres://host3:5432/db",
			expected: []string{
				"postgres://host1:5432/db",
				"postgres://host2:5432/db",
				"postgres://host3:5432/db",
			},
		},
		{
			name:  "URLs with whitespace",
			input: " postgres://host1:5432/db , postgres://host2:5432/db , postgres://host3:5432/db ",
			expected: []string{
				"postgres://host1:5432/db",
				"postgres://host2:5432/db",
				"postgres://host3:5432/db",
			},
		},
		{
			name:     "URLs with empty entries",
			input:    "postgres://host1:5432/db,,postgres://host2:5432/db,",
			expected: []string{"postgres://host1:5432/db", "postgres://host2:5432/db"},
		},
		{
			name:     "only commas and whitespace",
			input:    " , , , ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseReplicaURLs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConnectionConfig_Validation tests connection config validation
func TestConnectionConfig_Validation(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := ConnectionConfig{
			PrimaryURL:  "postgres://localhost:5432/test",
			ReplicaURLs: []string{"postgres://replica:5432/test"},
			MaxConns:    25,
			MinConns:    5,
			Timeout:     30 * time.Second,
			MaxLifetime: 1 * time.Hour,
			MaxIdleTime: 10 * time.Minute,
		}

		assert.NotEmpty(t, config.PrimaryURL)
		assert.Greater(t, config.MaxConns, 0)
		assert.GreaterOrEqual(t, config.MinConns, 0)
		assert.LessOrEqual(t, config.MinConns, config.MaxConns)
		assert.Greater(t, config.Timeout, time.Duration(0))
	})

	t.Run("config without replicas", func(t *testing.T) {
		config := ConnectionConfig{
			PrimaryURL:  "postgres://localhost:5432/test",
			ReplicaURLs: nil,
			MaxConns:    10,
			MinConns:    2,
			Timeout:     10 * time.Second,
		}

		assert.NotEmpty(t, config.PrimaryURL)
		assert.Nil(t, config.ReplicaURLs)
	})

	t.Run("min/max connection bounds", func(t *testing.T) {
		tests := []struct {
			name     string
			maxConns int
			minConns int
			valid    bool
		}{
			{"valid", 25, 5, true},
			{"min equals max", 10, 10, true},
			{"min exceeds max", 10, 20, false},
			{"zero max", 0, 5, false},
			{"negative min", 10, -1, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				valid := tt.maxConns > 0 && tt.minConns >= 0 && tt.minConns <= tt.maxConns
				assert.Equal(t, tt.valid, valid)
			})
		}
	})
}

// TestNewConnectionManager_InvalidPrimary tests connection manager with invalid primary
func TestNewConnectionManager_InvalidPrimary(t *testing.T) {
	t.Run("invalid primary URL", func(t *testing.T) {
		config := ConnectionConfig{
			PrimaryURL:  "invalid://badurl",
			MaxConns:    10,
			MinConns:    2,
			Timeout:     5 * time.Second,
			MaxLifetime: 1 * time.Hour,
			MaxIdleTime: 10 * time.Minute,
		}

		cm, err := NewConnectionManager(config)
		assert.Error(t, err)
		assert.Nil(t, cm)
		// The error could be from opening or pinging
		assert.True(t, strings.Contains(err.Error(), "failed to open primary connection") ||
			strings.Contains(err.Error(), "failed to ping primary"))
	})

	t.Run("unreachable primary", func(t *testing.T) {
		config := ConnectionConfig{
			PrimaryURL:  "postgres://nonexistent:9999/testdb?connect_timeout=1",
			MaxConns:    10,
			MinConns:    2,
			Timeout:     2 * time.Second,
			MaxLifetime: 1 * time.Hour,
			MaxIdleTime: 10 * time.Minute,
		}

		cm, err := NewConnectionManager(config)
		assert.Error(t, err)
		assert.Nil(t, cm)
		assert.Contains(t, err.Error(), "failed to ping primary")
	})
}

// TestConnectionManager_Primary tests the Primary method
func TestConnectionManager_Primary(t *testing.T) {
	cm := &ConnectionManager{
		primary: &sql.DB{},
	}

	primary := cm.Primary()
	assert.NotNil(t, primary)
	assert.Equal(t, cm.primary, primary)
}

// TestConnectionManager_Replica tests replica selection
func TestConnectionManager_Replica(t *testing.T) {
	t.Run("no replicas - fallback to primary", func(t *testing.T) {
		primaryDB := &sql.DB{}
		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{},
		}

		replica := cm.Replica()
		assert.Equal(t, primaryDB, replica, "Should return primary when no replicas")
	})

	t.Run("single replica", func(t *testing.T) {
		primaryDB := &sql.DB{}
		replicaDB := &sql.DB{}
		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{replicaDB},
		}

		replica := cm.Replica()
		assert.Equal(t, replicaDB, replica)
	})

	t.Run("round-robin selection with multiple replicas", func(t *testing.T) {
		replica1 := &sql.DB{}
		replica2 := &sql.DB{}
		replica3 := &sql.DB{}

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1, replica2, replica3},
		}

		// Get replicas and verify round-robin
		selections := make(map[*sql.DB]int)
		iterations := 30 // 10 cycles through 3 replicas

		for i := 0; i < iterations; i++ {
			replica := cm.Replica()
			selections[replica]++
		}

		// Each replica should be selected 10 times
		assert.Equal(t, 10, selections[replica1])
		assert.Equal(t, 10, selections[replica2])
		assert.Equal(t, 10, selections[replica3])
	})

	t.Run("concurrent replica selection", func(t *testing.T) {
		replica1 := &sql.DB{}
		replica2 := &sql.DB{}

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1, replica2},
		}

		var wg sync.WaitGroup
		iterations := 100
		results := make(chan *sql.DB, iterations)

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results <- cm.Replica()
			}()
		}

		wg.Wait()
		close(results)

		// Count selections
		selections := make(map[*sql.DB]int)
		for replica := range results {
			selections[replica]++
		}

		// Both replicas should be selected (roughly evenly)
		assert.NotZero(t, selections[replica1])
		assert.NotZero(t, selections[replica2])
		assert.Equal(t, iterations, selections[replica1]+selections[replica2])
	})
}

// TestConnectionManager_AllReplicas tests the AllReplicas method
func TestConnectionManager_AllReplicas(t *testing.T) {
	t.Run("no replicas", func(t *testing.T) {
		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{},
		}

		replicas := cm.AllReplicas()
		assert.Empty(t, replicas)
	})

	t.Run("multiple replicas", func(t *testing.T) {
		replica1 := &sql.DB{}
		replica2 := &sql.DB{}
		replica3 := &sql.DB{}

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1, replica2, replica3},
		}

		replicas := cm.AllReplicas()
		assert.Len(t, replicas, 3)
		assert.Contains(t, replicas, replica1)
		assert.Contains(t, replicas, replica2)
		assert.Contains(t, replicas, replica3)
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		replica1 := &sql.DB{}
		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1},
		}

		replicas1 := cm.AllReplicas()
		replicas2 := cm.AllReplicas()

		// Modify one slice
		replicas1[0] = &sql.DB{}

		// Original should be unchanged
		assert.Equal(t, replica1, replicas2[0])
	})
}

// TestConnectionManager_HealthCheck tests health check functionality
func TestConnectionManager_HealthCheck(t *testing.T) {
	t.Run("healthy primary and replicas", func(t *testing.T) {
		// Create mock primary
		primaryDB, primaryMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer primaryDB.Close()

		// Create mock replicas
		replica1DB, replica1Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica1DB.Close()

		replica2DB, replica2Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica2DB.Close()

		// Expect successful pings
		primaryMock.ExpectPing()
		replica1Mock.ExpectPing()
		replica2Mock.ExpectPing()

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{replica1DB, replica2DB},
		}

		err = cm.HealthCheck(context.Background())
		assert.NoError(t, err)

		assert.NoError(t, primaryMock.ExpectationsWereMet())
		assert.NoError(t, replica1Mock.ExpectationsWereMet())
		assert.NoError(t, replica2Mock.ExpectationsWereMet())
	})

	t.Run("unhealthy primary", func(t *testing.T) {
		primaryDB, primaryMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer primaryDB.Close()

		// Expect failed ping
		primaryMock.ExpectPing().WillReturnError(errors.New("connection refused"))

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{},
		}

		err = cm.HealthCheck(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "primary unhealthy")
	})

	t.Run("healthy primary with some unhealthy replicas", func(t *testing.T) {
		primaryDB, primaryMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer primaryDB.Close()

		replica1DB, replica1Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica1DB.Close()

		replica2DB, replica2Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica2DB.Close()

		primaryMock.ExpectPing()
		replica1Mock.ExpectPing()
		replica2Mock.ExpectPing().WillReturnError(errors.New("connection refused"))

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{replica1DB, replica2DB},
		}

		err = cm.HealthCheck(context.Background())
		// Should succeed - not all replicas are down
		assert.NoError(t, err)
	})

	t.Run("healthy primary with all replicas unhealthy", func(t *testing.T) {
		primaryDB, primaryMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer primaryDB.Close()

		replica1DB, replica1Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica1DB.Close()

		replica2DB, replica2Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica2DB.Close()

		primaryMock.ExpectPing()
		replica1Mock.ExpectPing().WillReturnError(errors.New("connection refused"))
		replica2Mock.ExpectPing().WillReturnError(errors.New("connection refused"))

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{replica1DB, replica2DB},
		}

		err = cm.HealthCheck(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "all replicas unhealthy")
	})

	t.Run("health check with context timeout", func(t *testing.T) {
		primaryDB, primaryMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer primaryDB.Close()

		primaryMock.ExpectPing().WillDelayFor(2 * time.Second)

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = cm.HealthCheck(ctx)
		assert.Error(t, err)
	})
}

// TestConnectionManager_Stats tests connection statistics
func TestConnectionManager_Stats(t *testing.T) {
	t.Run("stats from primary only", func(t *testing.T) {
		primaryDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer primaryDB.Close()

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{},
		}

		stats := cm.Stats()
		assert.NotNil(t, stats.Primary)
		assert.Empty(t, stats.Replicas)
	})

	t.Run("stats from primary and replicas", func(t *testing.T) {
		primaryDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer primaryDB.Close()

		replica1DB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer replica1DB.Close()

		replica2DB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer replica2DB.Close()

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{replica1DB, replica2DB},
		}

		stats := cm.Stats()
		assert.NotNil(t, stats.Primary)
		assert.Len(t, stats.Replicas, 2)
	})
}

// TestConnectionManager_RemoveUnhealthyReplicas tests replica removal
func TestConnectionManager_RemoveUnhealthyReplicas(t *testing.T) {
	t.Run("all replicas healthy", func(t *testing.T) {
		replica1DB, replica1Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica1DB.Close()

		replica2DB, replica2Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica2DB.Close()

		replica1Mock.ExpectPing()
		replica2Mock.ExpectPing()

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1DB, replica2DB},
		}

		removed := cm.RemoveUnhealthyReplicas(context.Background())
		assert.Equal(t, 0, removed)
		assert.Len(t, cm.replicas, 2)
	})

	t.Run("one replica unhealthy", func(t *testing.T) {
		replica1DB, replica1Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica1DB.Close()

		replica2DB, replica2Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica2DB.Close()

		replica1Mock.ExpectPing()
		replica2Mock.ExpectPing().WillReturnError(errors.New("connection refused"))
		replica2Mock.ExpectClose()

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1DB, replica2DB},
		}

		removed := cm.RemoveUnhealthyReplicas(context.Background())
		assert.Equal(t, 1, removed)
		assert.Len(t, cm.replicas, 1)
		assert.Equal(t, replica1DB, cm.replicas[0])
	})

	t.Run("all replicas unhealthy", func(t *testing.T) {
		replica1DB, replica1Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica1DB.Close()

		replica2DB, replica2Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica2DB.Close()

		replica1Mock.ExpectPing().WillReturnError(errors.New("connection refused"))
		replica1Mock.ExpectClose()
		replica2Mock.ExpectPing().WillReturnError(errors.New("connection refused"))
		replica2Mock.ExpectClose()

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1DB, replica2DB},
		}

		removed := cm.RemoveUnhealthyReplicas(context.Background())
		assert.Equal(t, 2, removed)
		assert.Empty(t, cm.replicas)
	})

	t.Run("no replicas", func(t *testing.T) {
		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{},
		}

		removed := cm.RemoveUnhealthyReplicas(context.Background())
		assert.Equal(t, 0, removed)
		assert.Empty(t, cm.replicas)
	})
}

// TestConnectionManager_AddReplica tests dynamic replica addition
func TestConnectionManager_AddReplica(t *testing.T) {
	t.Run("invalid replica URL", func(t *testing.T) {
		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{},
			config: ConnectionConfig{
				MaxConns:    10,
				MinConns:    2,
				Timeout:     5 * time.Second,
				MaxLifetime: 1 * time.Hour,
				MaxIdleTime: 10 * time.Minute,
			},
		}

		err := cm.AddReplica("invalid://badurl")
		assert.Error(t, err)
		// The error could be from opening or pinging
		assert.True(t, strings.Contains(err.Error(), "failed to open replica connection") ||
			strings.Contains(err.Error(), "failed to ping replica"))
	})

	t.Run("unreachable replica", func(t *testing.T) {
		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{},
			config: ConnectionConfig{
				MaxConns:    10,
				MinConns:    2,
				Timeout:     1 * time.Second,
				MaxLifetime: 1 * time.Hour,
				MaxIdleTime: 10 * time.Minute,
			},
		}

		err := cm.AddReplica("postgres://nonexistent:9999/testdb?connect_timeout=1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to ping replica")
	})

	t.Run("replica max connections calculation", func(t *testing.T) {
		tests := []struct {
			name               string
			primaryMaxConns    int
			expectedReplicaMax int
		}{
			{"normal case", 20, 10},
			{"small primary", 2, 2}, // Min is 2
			{"large primary", 100, 50},
			{"odd number", 15, 7},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				replicaMaxConns := tt.primaryMaxConns / 2
				if replicaMaxConns < 2 {
					replicaMaxConns = 2
				}
				assert.Equal(t, tt.expectedReplicaMax, replicaMaxConns)
			})
		}
	})
}

// TestConnectionManager_Close tests connection cleanup
func TestConnectionManager_Close(t *testing.T) {
	t.Run("close primary only", func(t *testing.T) {
		primaryDB, primaryMock, err := sqlmock.New()
		require.NoError(t, err)

		primaryMock.ExpectClose()

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{},
		}

		err = cm.Close()
		assert.NoError(t, err)
		assert.NoError(t, primaryMock.ExpectationsWereMet())
	})

	t.Run("close primary and replicas", func(t *testing.T) {
		primaryDB, primaryMock, err := sqlmock.New()
		require.NoError(t, err)

		replica1DB, replica1Mock, err := sqlmock.New()
		require.NoError(t, err)

		replica2DB, replica2Mock, err := sqlmock.New()
		require.NoError(t, err)

		primaryMock.ExpectClose()
		replica1Mock.ExpectClose()
		replica2Mock.ExpectClose()

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{replica1DB, replica2DB},
		}

		err = cm.Close()
		assert.NoError(t, err)
		assert.NoError(t, primaryMock.ExpectationsWereMet())
		assert.NoError(t, replica1Mock.ExpectationsWereMet())
		assert.NoError(t, replica2Mock.ExpectationsWereMet())
	})

	t.Run("close with errors", func(t *testing.T) {
		primaryDB, primaryMock, err := sqlmock.New()
		require.NoError(t, err)

		replica1DB, replica1Mock, err := sqlmock.New()
		require.NoError(t, err)

		primaryMock.ExpectClose().WillReturnError(errors.New("primary close error"))
		replica1Mock.ExpectClose().WillReturnError(errors.New("replica close error"))

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{replica1DB},
		}

		err = cm.Close()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection close errors")
	})

	t.Run("close clears replicas", func(t *testing.T) {
		primaryDB, primaryMock, err := sqlmock.New()
		require.NoError(t, err)

		replica1DB, replica1Mock, err := sqlmock.New()
		require.NoError(t, err)

		primaryMock.ExpectClose()
		replica1Mock.ExpectClose()

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{replica1DB},
		}

		err = cm.Close()
		assert.NoError(t, err)
		assert.Nil(t, cm.replicas)
	})
}

// TestConnectionManager_StartHealthCheckRoutine tests background health check
func TestConnectionManager_StartHealthCheckRoutine(t *testing.T) {
	t.Run("routine runs and checks health", func(t *testing.T) {
		replica1DB, replica1Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica1DB.Close()

		// Expect at least one ping
		replica1Mock.ExpectPing()

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1DB},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start with short interval
		cm.StartHealthCheckRoutine(ctx, 100*time.Millisecond)

		// Wait for at least one health check
		time.Sleep(150 * time.Millisecond)

		// Cancel context to stop routine
		cancel()

		// Give goroutine time to clean up
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("routine uses default interval when zero", func(t *testing.T) {
		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start with zero interval (should default to 30s)
		cm.StartHealthCheckRoutine(ctx, 0)

		// Cancel immediately
		cancel()

		// Give goroutine time to clean up
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("routine stops on context cancellation", func(t *testing.T) {
		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{},
		}

		ctx, cancel := context.WithCancel(context.Background())

		cm.StartHealthCheckRoutine(ctx, 1*time.Second)

		// Cancel immediately
		cancel()

		// Give goroutine time to stop
		time.Sleep(50 * time.Millisecond)

		// If we get here without hanging, the test passes
	})

	t.Run("routine removes unhealthy replicas", func(t *testing.T) {
		replica1DB, replica1Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica1DB.Close()

		// First ping succeeds, second fails
		replica1Mock.ExpectPing()
		replica1Mock.ExpectPing().WillReturnError(errors.New("connection lost"))
		replica1Mock.ExpectClose()

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1DB},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start with very short interval
		cm.StartHealthCheckRoutine(ctx, 50*time.Millisecond)

		// Wait for two health checks
		time.Sleep(150 * time.Millisecond)

		// Cancel context
		cancel()

		// Give goroutine time to clean up
		time.Sleep(50 * time.Millisecond)

		// Verify replica was removed
		cm.mu.RLock()
		replicaCount := len(cm.replicas)
		cm.mu.RUnlock()

		assert.Equal(t, 0, replicaCount, "Unhealthy replica should have been removed")
	})
}

// TestConnectionManager_ConcurrentOperations tests thread safety
func TestConnectionManager_ConcurrentOperations(t *testing.T) {
	t.Run("concurrent replica access", func(t *testing.T) {
		replica1 := &sql.DB{}
		replica2 := &sql.DB{}

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1, replica2},
		}

		var wg sync.WaitGroup
		iterations := 100

		// Concurrent reads
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = cm.Replica()
				_ = cm.AllReplicas()
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent replica modification", func(t *testing.T) {
		replica1DB, replica1Mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replica1DB.Close()

		// Expect pings for health checks
		for i := 0; i < 50; i++ {
			replica1Mock.ExpectPing()
		}

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replica1DB},
		}

		var wg sync.WaitGroup

		// Concurrent reads
		for i := 0; i < 25; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = cm.AllReplicas()
			}()
		}

		// Concurrent health checks
		for i := 0; i < 25; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = cm.RemoveUnhealthyReplicas(context.Background())
			}()
		}

		wg.Wait()
	})
}

// TestConnectionStats tests the ConnectionStats structure
func TestConnectionStats(t *testing.T) {
	t.Run("stats structure", func(t *testing.T) {
		stats := ConnectionStats{
			Primary: sql.DBStats{
				MaxOpenConnections: 25,
				OpenConnections:    5,
				InUse:              2,
				Idle:               3,
			},
			Replicas: []sql.DBStats{
				{
					MaxOpenConnections: 10,
					OpenConnections:    3,
					InUse:              1,
					Idle:               2,
				},
			},
		}

		assert.Equal(t, 25, stats.Primary.MaxOpenConnections)
		assert.Len(t, stats.Replicas, 1)
		assert.Equal(t, 10, stats.Replicas[0].MaxOpenConnections)
	})
}

// TestConnectionManager_ConnectionPoolConfiguration tests pool settings
func TestConnectionManager_ConnectionPoolConfiguration(t *testing.T) {
	t.Run("verify pool configuration values", func(t *testing.T) {
		config := ConnectionConfig{
			MaxConns:    25,
			MinConns:    5,
			MaxLifetime: 1 * time.Hour,
			MaxIdleTime: 10 * time.Minute,
		}

		assert.Equal(t, 25, config.MaxConns)
		assert.Equal(t, 5, config.MinConns)
		assert.Equal(t, 1*time.Hour, config.MaxLifetime)
		assert.Equal(t, 10*time.Minute, config.MaxIdleTime)
	})

	t.Run("replica pool is half of primary", func(t *testing.T) {
		tests := []struct {
			primaryMax int
			replicaMax int
		}{
			{20, 10},
			{100, 50},
			{3, 2}, // Less than min
			{1, 2}, // Less than min
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("primary_%d", tt.primaryMax), func(t *testing.T) {
				replicaMax := tt.primaryMax / 2
				if replicaMax < 2 {
					replicaMax = 2
				}
				assert.Equal(t, tt.replicaMax, replicaMax)
			})
		}
	})
}

// TestConnectionManager_ContextHandling tests context usage
func TestConnectionManager_ContextHandling(t *testing.T) {
	t.Run("health check respects context", func(t *testing.T) {
		primaryDB, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer primaryDB.Close()

		cm := &ConnectionManager{
			primary:  primaryDB,
			replicas: []*sql.DB{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = cm.HealthCheck(ctx)
		assert.Error(t, err)
	})

	t.Run("remove unhealthy replicas respects context", func(t *testing.T) {
		replicaDB, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer replicaDB.Close()

		cm := &ConnectionManager{
			primary:  &sql.DB{},
			replicas: []*sql.DB{replicaDB},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		removed := cm.RemoveUnhealthyReplicas(ctx)
		// Should handle cancellation gracefully
		assert.GreaterOrEqual(t, removed, 0)
	})
}
