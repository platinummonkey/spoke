// +build integration

package api

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestContainerCleanupOption configures container cleanup behavior
type TestContainerCleanupOption func(*testContainerCleanupConfig)

type testContainerCleanupConfig struct {
	removeVolumes  bool
	cleanupTimeout time.Duration
}

// WithRemoveVolumes ensures volumes are removed on cleanup (default: true)
func WithRemoveVolumes(remove bool) TestContainerCleanupOption {
	return func(c *testContainerCleanupConfig) {
		c.removeVolumes = remove
	}
}

// WithCleanupTimeout sets the timeout for cleanup operations (default: 30s)
func WithCleanupTimeout(timeout time.Duration) TestContainerCleanupOption {
	return func(c *testContainerCleanupConfig) {
		c.cleanupTimeout = timeout
	}
}

// SetupPostgresContainer creates a PostgreSQL test container with proper cleanup
// that removes both the container and its volumes.
//
// Usage:
//
//	db, cleanup := SetupPostgresContainer(t)
//	defer cleanup()
//
// The cleanup function will:
// 1. Close the database connection
// 2. Terminate the container with a fresh context (avoiding cancelled context issues)
// 3. Remove the container and its volumes automatically
func SetupPostgresContainer(t *testing.T, opts ...TestContainerCleanupOption) (*sql.DB, func()) {
	t.Helper()

	// Apply options
	config := &testContainerCleanupConfig{
		removeVolumes:  true,
		cleanupTimeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(config)
	}

	ctx := context.Background()

	// Check if Docker/Podman is available
	provider, err := testcontainers.ProviderDocker.GetProvider()
	if err != nil {
		t.Skip("Docker/Podman not available, skipping integration tests")
	}
	defer provider.Close()

	// Container options for automatic cleanup
	containerOpts := []testcontainers.ContainerCustomizer{
		postgres.WithDatabase("spoke_test"),
		postgres.WithUsername("spoke"),
		postgres.WithPassword("spoke_test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second)),
		postgres.BasicWaitStrategies(),
	}

	// Add AutoRemove if volume removal is requested
	if config.removeVolumes {
		containerOpts = append(containerOpts,
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					// AutoRemove ensures container and volumes are removed when terminated
					AutoRemove: true,
				},
			}),
		)
	}

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx, "postgres:15-alpine", containerOpts...)
	if err != nil {
		t.Skipf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	// Wait for connection
	err = db.Ping()
	require.NoError(t, err)

	// Run migrations
	err = runMigrations(db)
	require.NoError(t, err, "Failed to run migrations")

	// Cleanup function with proper volume and container removal
	cleanup := func() {
		// Close database connection first
		if err := db.Close(); err != nil {
			t.Logf("Warning: Failed to close database: %v", err)
		}

		// Use a fresh context for cleanup to avoid issues with cancelled contexts
		// This is important because the original ctx might be cancelled by test timeout
		cleanupCtx, cancel := context.WithTimeout(context.Background(), config.cleanupTimeout)
		defer cancel()

		// Terminate will remove the container and volumes if AutoRemove is set
		if err := postgresContainer.Terminate(cleanupCtx); err != nil {
			t.Errorf("Failed to terminate container: %v", err)
		}

		t.Log("Successfully cleaned up testcontainer and volumes")
	}

	return db, cleanup
}

// CleanupOrphanedTestContainers finds and removes any orphaned testcontainers
// that weren't properly cleaned up. This can be called in TestMain or as a
// utility function.
//
// Usage:
//
//	func TestMain(m *testing.M) {
//	    // Clean up any orphaned containers from previous test runs
//	    CleanupOrphanedTestContainers()
//	    os.Exit(m.Run())
//	}
func CleanupOrphanedTestContainers() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	provider, err := testcontainers.ProviderDocker.GetProvider()
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer provider.Close()

	// List containers with testcontainers label
	containers, err := provider.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	cleaned := 0
	for _, container := range containers {
		// Check if it's a testcontainer (they have specific labels)
		if _, ok := container.Labels["org.testcontainers"]; ok {
			if err := provider.RemoveContainer(ctx, container.ID); err != nil {
				// Log but don't fail - best effort cleanup
				fmt.Printf("Warning: Failed to remove orphaned container %s: %v\n", container.ID, err)
			} else {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		fmt.Printf("Cleaned up %d orphaned testcontainers\n", cleaned)
	}

	return nil
}

// runMigrations applies database migrations from the migrations directory
func runMigrations(db *sql.DB) error {
	// Get the migrations directory path
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Find migrations directory (go up until we find it)
	migrationsDir := filepath.Join(wd, "..", "..", "migrations")
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Try one more level up
		migrationsDir = filepath.Join(wd, "..", "..", "..", "migrations")
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			return fmt.Errorf("migrations directory not found")
		}
	}

	// Read and apply key migration files
	migrationFiles := []string{
		"001_create_base_schema.up.sql",
		"002_create_auth_schema.up.sql",
	}

	for _, filename := range migrationFiles {
		migrationPath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}
	}

	return nil
}
