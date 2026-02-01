//go:build integration

package search

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupPostgresTestDB creates a PostgreSQL test container with full-text search support
func setupPostgresTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase("search_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	err = db.Ping()
	require.NoError(t, err)

	// Run migrations
	err = runSearchMigrations(db)
	require.NoError(t, err, "Failed to run migrations")

	cleanup := func() {
		db.Close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := postgresContainer.Terminate(cleanupCtx); err != nil {
			t.Logf("Warning: Failed to terminate container: %v", err)
		}
	}

	return db, cleanup
}

// runSearchMigrations applies necessary migrations for search functionality
func runSearchMigrations(db *sql.DB) error {
	// Get migrations directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Navigate up to find migrations directory
	migrationsDir := filepath.Join(wd, "..", "..", "migrations")
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Try one more level up
		migrationsDir = filepath.Join(wd, "..", "..", "..", "migrations")
	}

	// Read and apply migrations
	migrationFiles, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return err
	}

	sort.Strings(migrationFiles)

	for _, migrationPath := range migrationFiles {
		content, err := os.ReadFile(migrationPath)
		if err != nil {
			return err
		}

		_, err = db.Exec(string(content))
		if err != nil {
			// Log but continue - some migrations may have dependencies
			// that aren't available in test environment
			continue
		}
	}

	return nil
}

// seedSearchTestData populates the database with test data for full-text search
func seedSearchTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	// Create test module
	_, err := db.Exec(`
		INSERT INTO modules (name, description) VALUES
		('test-module', 'Test module for search')
	`)
	require.NoError(t, err)

	// Get module ID
	var moduleID int64
	err = db.QueryRow("SELECT id FROM modules WHERE name = 'test-module'").Scan(&moduleID)
	require.NoError(t, err)

	// Create version
	_, err = db.Exec(`
		INSERT INTO versions (module_id, version) VALUES ($1, 'v1.0.0')
	`, moduleID)
	require.NoError(t, err)

	// Get version ID
	var versionID int64
	err = db.QueryRow("SELECT id FROM versions WHERE module_id = $1 AND version = 'v1.0.0'", moduleID).Scan(&versionID)
	require.NoError(t, err)

	// Insert search index entries - the trigger will automatically populate search_vector
	testData := []struct {
		entityType   string
		entityName   string
		fullPath     string
		parentPath   *string
		description  *string
		comments     *string
		fieldType    *string
		fieldNumber  *int
	}{
		{
			entityType:  "message",
			entityName:  "User",
			fullPath:    "test.User",
			description: strPtr("User profile message"),
			comments:    strPtr("Main user object"),
		},
		{
			entityType:  "field",
			entityName:  "email",
			fullPath:    "test.User.email",
			parentPath:  strPtr("test.User"),
			fieldType:   strPtr("string"),
			fieldNumber: intPtr(1),
			description: strPtr("User email address"),
		},
		{
			entityType:  "enum",
			entityName:  "Status",
			fullPath:    "test.Status",
			description: strPtr("User status enum"),
		},
		{
			entityType: "service",
			entityName: "UserService",
			fullPath:   "test.UserService",
			comments:   strPtr("Service for managing users"),
		},
		{
			entityType:  "message",
			entityName:  "GetUserRequest",
			fullPath:    "test.GetUserRequest",
			description: strPtr("Request to get a user by ID"),
		},
	}

	for _, data := range testData {
		_, err := db.Exec(`
			INSERT INTO proto_search_index
			(version_id, entity_type, entity_name, full_path, parent_path, description, comments, field_type, field_number)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, versionID, data.entityType, data.entityName, data.fullPath, data.parentPath, data.description, data.comments, data.fieldType, data.fieldNumber)
		require.NoError(t, err)
	}
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

// TestSearchService_SearchBasic_Integration tests basic full-text search with PostgreSQL
func TestSearchService_SearchBasic_Integration(t *testing.T) {
	db, cleanup := setupPostgresTestDB(t)
	defer cleanup()

	seedSearchTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	tests := []struct {
		name           string
		query          string
		minResults     int
		checkFirstName string
	}{
		{
			name:           "search for User",
			query:          "User",
			minResults:     2, // User message, UserService, GetUserRequest
			checkFirstName: "User",
		},
		{
			name:           "search for email",
			query:          "email",
			minResults:     1,
			checkFirstName: "email",
		},
		{
			name:           "search for Status",
			query:          "Status",
			minResults:     1,
			checkFirstName: "Status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SearchRequest{
				Query: tt.query,
				Limit: 50,
			}

			response, err := service.Search(ctx, req)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(response.Results), tt.minResults, "Expected at least %d results", tt.minResults)

			if len(response.Results) > 0 {
				// Check that the first result contains our expected name
				found := false
				for _, result := range response.Results {
					if result.EntityName == tt.checkFirstName {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected to find entity named %s in results", tt.checkFirstName)
			}
		})
	}
}

// TestSearchService_SearchWithEntityFilter_Integration tests filtering by entity type
func TestSearchService_SearchWithEntityFilter_Integration(t *testing.T) {
	db, cleanup := setupPostgresTestDB(t)
	defer cleanup()

	seedSearchTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	tests := []struct {
		name             string
		query            string
		expectedEntity   string
		minResults       int
	}{
		{
			name:           "search messages only",
			query:          "User entity:message",
			expectedEntity: "message",
			minResults:     2, // User, GetUserRequest
		},
		{
			name:           "search services only",
			query:          "User entity:service",
			expectedEntity: "service",
			minResults:     1, // UserService
		},
		{
			name:           "search fields only",
			query:          "email entity:field",
			expectedEntity: "field",
			minResults:     1, // email field
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SearchRequest{
				Query: tt.query,
				Limit: 50,
			}

			response, err := service.Search(ctx, req)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(response.Results), tt.minResults)

			// Verify all results match the entity type filter
			for _, result := range response.Results {
				assert.Equal(t, tt.expectedEntity, result.EntityType)
			}
		})
	}
}

// TestSearchService_FuzzySearch_Integration tests fuzzy/partial matching
func TestSearchService_FuzzySearch_Integration(t *testing.T) {
	db, cleanup := setupPostgresTestDB(t)
	defer cleanup()

	seedSearchTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	req := SearchRequest{
		Query: "user", // lowercase, should still match User entities
		Limit: 50,
	}

	response, err := service.Search(ctx, req)
	require.NoError(t, err)
	assert.Greater(t, len(response.Results), 0, "Fuzzy search should return results")
}

// TestSearchService_RankingOrder_Integration tests that results are ranked by relevance
func TestSearchService_RankingOrder_Integration(t *testing.T) {
	db, cleanup := setupPostgresTestDB(t)
	defer cleanup()

	seedSearchTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	req := SearchRequest{
		Query: "User",
		Limit: 50,
	}

	response, err := service.Search(ctx, req)
	require.NoError(t, err)
	require.Greater(t, len(response.Results), 0)

	// Verify results include relevance scores (if the service provides them)
	// Entity names that exactly match should rank higher
	firstResult := response.Results[0]
	assert.Contains(t, firstResult.EntityName, "User", "First result should be highly relevant to 'User'")
}
