package rbac

import (
	"os"
	"testing"
)

func TestIsDatabaseAvailable(t *testing.T) {
	// Save original value
	original := os.Getenv("TEST_POSTGRES_PRIMARY")
	defer func() {
		if original != "" {
			os.Setenv("TEST_POSTGRES_PRIMARY", original)
		} else {
			os.Unsetenv("TEST_POSTGRES_PRIMARY")
		}
	}()

	t.Run("returns true when env var is set", func(t *testing.T) {
		os.Setenv("TEST_POSTGRES_PRIMARY", "postgres://test")
		if !IsDatabaseAvailable() {
			t.Error("Expected IsDatabaseAvailable to return true when env var is set")
		}
	})

	t.Run("returns false when env var is not set", func(t *testing.T) {
		os.Unsetenv("TEST_POSTGRES_PRIMARY")
		if IsDatabaseAvailable() {
			t.Error("Expected IsDatabaseAvailable to return false when env var is not set")
		}
	})
}

func TestSkipIfNoDatabaseOrShort(t *testing.T) {
	// This test just verifies the function exists and can be called
	// The actual skip logic is tested by integration tests
	original := os.Getenv("TEST_POSTGRES_PRIMARY")
	if original == "" {
		os.Setenv("TEST_POSTGRES_PRIMARY", "postgres://fake")
		defer os.Unsetenv("TEST_POSTGRES_PRIMARY")
	}

	// We can't actually test the skip behavior easily in a unit test
	// since t.Skip() would skip this test. But we can at least verify
	// the function compiles and runs in non-short, database-available mode.
	if !testing.Short() && IsDatabaseAvailable() {
		_ = SkipIfNoDatabaseOrShort(t)
	}
}
