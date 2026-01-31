package rbac

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// SkipIfNoDatabase skips the test if TEST_POSTGRES_PRIMARY environment variable is not set.
// This allows tests to run in CI where the database is available, but skip locally if not configured.
func SkipIfNoDatabase(t *testing.T) string {
	t.Helper()

	dbURL := os.Getenv("TEST_POSTGRES_PRIMARY")
	if dbURL == "" {
		t.Skip("Skipping test: TEST_POSTGRES_PRIMARY environment variable not set (database not available)")
	}

	return dbURL
}

// SkipIfNoDatabaseOrShort skips the test if running in short mode OR if database is not available.
func SkipIfNoDatabaseOrShort(t *testing.T) string {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	return SkipIfNoDatabase(t)
}

// RequireDatabase gets the database connection or skips the test if not available.
// Returns a connected database instance.
func RequireDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dbURL := SkipIfNoDatabase(t)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Failed to connect to database: %v", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Database not reachable: %v", err)
	}

	return db
}

// IsDatabaseAvailable returns true if TEST_POSTGRES_PRIMARY is set (does not test connection).
func IsDatabaseAvailable() bool {
	return os.Getenv("TEST_POSTGRES_PRIMARY") != ""
}
