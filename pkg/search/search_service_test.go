package search

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3" // SQLite driver for testing
)

// NOTE: These tests use SQLite for convenience, but the actual SearchService
// requires PostgreSQL for full-text search features (tsvector, @@, ts_rank).
// These tests verify the query building logic but not the actual FTS functionality.
// For full FTS testing, use PostgreSQL integration tests in tests/integration/

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE modules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT
		);

		CREATE TABLE versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			module_id INTEGER NOT NULL REFERENCES modules(id),
			version TEXT NOT NULL
		);

		CREATE TABLE proto_search_index (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version_id INTEGER NOT NULL REFERENCES versions(id),
			entity_type TEXT NOT NULL,
			entity_name TEXT NOT NULL,
			full_path TEXT NOT NULL,
			parent_path TEXT,
			proto_file_path TEXT,
			line_number INTEGER,
			description TEXT,
			comments TEXT,
			field_type TEXT,
			field_number INTEGER,
			is_repeated INTEGER DEFAULT 0,
			is_optional INTEGER DEFAULT 0,
			method_input_type TEXT,
			method_output_type TEXT,
			metadata TEXT DEFAULT '{}'
		);

		CREATE TABLE search_suggestions (
			query TEXT PRIMARY KEY,
			search_count INTEGER,
			last_searched_at TIMESTAMP
		);
	`)
	require.NoError(t, err)

	return db
}

// seedTestData inserts test data into the database
func seedTestData(t *testing.T, db *sql.DB) {
	// Insert module
	result, err := db.Exec(`INSERT INTO modules (name, description) VALUES ('user', 'User service')`)
	require.NoError(t, err)
	moduleID, _ := result.LastInsertId()

	// Insert version
	result, err = db.Exec(`INSERT INTO versions (module_id, version) VALUES (?, 'v1.0.0')`, moduleID)
	require.NoError(t, err)
	versionID, _ := result.LastInsertId()

	// Insert proto entities
	entities := []struct {
		entityType  string
		entityName  string
		fullPath    string
		parentPath  string
		description string
		fieldType   string
	}{
		{"message", "User", "user.v1.User", "user.v1", "User message", ""},
		{"field", "id", "user.v1.User.id", "user.v1.User", "User ID", "string"},
		{"field", "email", "user.v1.User.email", "user.v1.User", "Email address", "string"},
		{"field", "age", "user.v1.User.age", "user.v1.User", "User age", "int32"},
		{"enum", "Status", "user.v1.Status", "user.v1", "User status", ""},
		{"service", "UserService", "user.v1.UserService", "user.v1", "User service", ""},
		{"method", "GetUser", "user.v1.UserService.GetUser", "user.v1.UserService", "Get user by ID", ""},
	}

	for _, entity := range entities {
		_, err := db.Exec(`
			INSERT INTO proto_search_index (
				version_id, entity_type, entity_name, full_path, parent_path, description, field_type
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, versionID, entity.entityType, entity.entityName, entity.fullPath, entity.parentPath, entity.description, entity.fieldType)
		require.NoError(t, err)
	}
}

func TestSearchService_SearchBasic(t *testing.T) {
	t.Skip("Skipping: requires PostgreSQL full-text search (tsvector, @@, ts_rank). Use integration tests with PostgreSQL instead.")

	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	tests := []struct {
		name           string
		query          string
		expectedCount  int
		checkFirstName string
	}{
		{
			name:           "search for User",
			query:          "User",
			expectedCount:  3, // User message, UserService, GetUser
			checkFirstName: "User",
		},
		{
			name:           "search for email",
			query:          "email",
			expectedCount:  1,
			checkFirstName: "email",
		},
		{
			name:           "search for Status",
			query:          "Status",
			expectedCount:  1,
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
			assert.Len(t, response.Results, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, tt.checkFirstName, response.Results[0].EntityName)
			}
		})
	}
}

func TestSearchService_SearchWithEntityFilter(t *testing.T) {
	t.Skip("Skipping: requires PostgreSQL full-text search (tsvector, @@, ts_rank). Use integration tests with PostgreSQL instead.")

	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedType  string
	}{
		{
			name:          "search for message entities",
			query:         "entity:message",
			expectedCount: 1, // User message
			expectedType:  "message",
		},
		{
			name:          "search for field entities",
			query:         "entity:field",
			expectedCount: 3, // id, email, age
			expectedType:  "field",
		},
		{
			name:          "search for enum entities",
			query:         "entity:enum",
			expectedCount: 1, // Status
			expectedType:  "enum",
		},
		{
			name:          "search for service entities",
			query:         "entity:service",
			expectedCount: 1, // UserService
			expectedType:  "service",
		},
		{
			name:          "search for method entities",
			query:         "entity:method",
			expectedCount: 1, // GetUser
			expectedType:  "method",
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
			assert.Len(t, response.Results, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedType, response.Results[0].EntityType)
			}
		})
	}
}

func TestSearchService_SearchWithFieldTypeFilter(t *testing.T) {
t.Skip("Skipping: requires PostgreSQL full-text search (tsvector, @@, ts_rank). Use integration tests with PostgreSQL instead.")

	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedType  string
	}{
		{
			name:          "search for string fields",
			query:         "type:string",
			expectedCount: 2, // id, email
			expectedType:  "string",
		},
		{
			name:          "search for int32 fields",
			query:         "type:int32",
			expectedCount: 1, // age
			expectedType:  "int32",
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
			assert.Len(t, response.Results, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedType, response.Results[0].FieldType)
			}
		})
	}
}

func TestSearchService_SearchWithModuleFilter(t *testing.T) {
t.Skip("Skipping: requires PostgreSQL full-text search (tsvector, @@, ts_rank). Use integration tests with PostgreSQL instead.")

	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	req := SearchRequest{
		Query: "module:user",
		Limit: 50,
	}

	response, err := service.Search(ctx, req)
	require.NoError(t, err)
	assert.Len(t, response.Results, 7) // All entities in user module
	assert.Equal(t, "user", response.Results[0].ModuleName)
}

func TestSearchService_SearchCombinedFilters(t *testing.T) {
t.Skip("Skipping: requires PostgreSQL full-text search (tsvector, @@, ts_rank). Use integration tests with PostgreSQL instead.")

	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	// Search for "email entity:field type:string"
	req := SearchRequest{
		Query: "entity:field type:string",
		Limit: 50,
	}

	response, err := service.Search(ctx, req)
	require.NoError(t, err)
	assert.Len(t, response.Results, 2) // id, email (both string fields)
	assert.Equal(t, "field", response.Results[0].EntityType)
	assert.Equal(t, "string", response.Results[0].FieldType)
}

func TestSearchService_SearchPagination(t *testing.T) {
t.Skip("Skipping: requires PostgreSQL full-text search (tsvector, @@, ts_rank). Use integration tests with PostgreSQL instead.")

	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	// Get first page
	req1 := SearchRequest{
		Query:  "entity:field",
		Limit:  2,
		Offset: 0,
	}

	response1, err := service.Search(ctx, req1)
	require.NoError(t, err)
	assert.Len(t, response1.Results, 2)
	assert.Equal(t, 3, response1.TotalCount)

	// Get second page
	req2 := SearchRequest{
		Query:  "entity:field",
		Limit:  2,
		Offset: 2,
	}

	response2, err := service.Search(ctx, req2)
	require.NoError(t, err)
	assert.Len(t, response2.Results, 1) // Only 1 remaining
	assert.Equal(t, 3, response2.TotalCount)

	// Verify different results
	assert.NotEqual(t, response1.Results[0].EntityName, response2.Results[0].EntityName)
}

func TestSearchService_GetSuggestions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert test suggestions
	_, err := db.Exec(`
		INSERT INTO search_suggestions (query, search_count, last_searched_at)
		VALUES
			('user', 10, datetime('now')),
			('user email', 5, datetime('now')),
			('email', 3, datetime('now'))
	`)
	require.NoError(t, err)

	service := NewSearchService(db)
	ctx := context.Background()

	// Get suggestions for "user"
	suggestions, err := service.GetSuggestions(ctx, "user", 5)
	require.NoError(t, err)
	assert.Len(t, suggestions, 2) // "user", "user email"
	assert.Equal(t, "user", suggestions[0])
}

func TestSearchService_EmptyQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	service := NewSearchService(db)
	ctx := context.Background()

	req := SearchRequest{
		Query: "",
		Limit: 50,
	}

	response, err := service.Search(ctx, req)
	require.NoError(t, err)
	// Empty query should return all results (or none if filtering requires terms)
	assert.NotNil(t, response)
}
