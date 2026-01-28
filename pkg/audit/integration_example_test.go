package audit

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultIntegrationConfig(t *testing.T) {
	db := &sql.DB{}

	config := DefaultIntegrationConfig(db)

	assert.NotNil(t, config.DB)
	assert.Equal(t, db, config.DB)
	assert.True(t, config.FileLoggingEnabled)
	assert.Equal(t, "/var/log/spoke/audit", config.FileLogPath)
	assert.True(t, config.FileLogRotate)
	assert.Equal(t, int64(100*1024*1024), config.FileLogMaxSize)
	assert.Equal(t, 10, config.FileLogMaxFiles)
	assert.True(t, config.DBLoggingEnabled)
	assert.False(t, config.LogAllRequests)
	assert.NotNil(t, config.RetentionPolicy)
	assert.Equal(t, 90, config.RetentionPolicy.RetentionDays)
}

func TestDefaultIntegrationConfig_NilDB(t *testing.T) {
	config := DefaultIntegrationConfig(nil)

	assert.Nil(t, config.DB)
	assert.True(t, config.FileLoggingEnabled)
	assert.True(t, config.DBLoggingEnabled)
}

func TestSetupAuditLogging_FileLoggingOnly(t *testing.T) {
	// Create a temporary directory for audit logs
	tempDir, err := os.MkdirTemp("", "audit-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	config := IntegrationConfig{
		FileLoggingEnabled: true,
		FileLogPath:        tempDir,
		FileLogRotate:      true,
		FileLogMaxSize:     100 * 1024 * 1024,
		FileLogMaxFiles:    10,
		DBLoggingEnabled:   false,
		LogAllRequests:     true,
	}

	middleware, handlers, err := SetupAuditLogging(config)

	require.NoError(t, err)
	assert.NotNil(t, middleware)
	assert.Nil(t, handlers, "Handlers should be nil when DB logging is disabled")

	// Verify that the file logger was created successfully
	logFile := filepath.Join(tempDir, "audit.log")
	_, err = os.Stat(logFile)
	assert.NoError(t, err, "Audit log file should be created")
}

func TestSetupAuditLogging_FileLoggingError(t *testing.T) {
	config := IntegrationConfig{
		FileLoggingEnabled: true,
		FileLogPath:        "/invalid/path/that/does/not/exist/and/cannot/be/created",
		DBLoggingEnabled:   false,
	}

	middleware, handlers, err := SetupAuditLogging(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create file logger")
	assert.Nil(t, middleware)
	assert.Nil(t, handlers)
}

func TestSetupAuditLogging_DBLoggingError(t *testing.T) {
	// Create a closed database connection to trigger an error
	db, err := sql.Open("postgres", "invalid-connection-string")
	require.NoError(t, err)
	db.Close() // Close immediately to make it invalid

	config := IntegrationConfig{
		FileLoggingEnabled: false,
		DBLoggingEnabled:   true,
		DB:                 db,
	}

	middleware, handlers, err := SetupAuditLogging(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create database logger")
	assert.Nil(t, middleware)
	assert.Nil(t, handlers)
}

func TestSetupAuditLogging_BothLoggers(t *testing.T) {
	// Note: This test will fail to create DB logger without a real database
	// But we can test the configuration and partial setup
	tempDir, err := os.MkdirTemp("", "audit-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test with nil DB - should skip DB logger creation
	config := IntegrationConfig{
		FileLoggingEnabled: true,
		FileLogPath:        tempDir,
		FileLogRotate:      false,
		FileLogMaxSize:     50 * 1024 * 1024,
		FileLogMaxFiles:    5,
		DBLoggingEnabled:   true,
		DB:                 nil, // Nil DB will skip DB logger
		LogAllRequests:     false,
	}

	middleware, handlers, err := SetupAuditLogging(config)

	require.NoError(t, err)
	assert.NotNil(t, middleware)
	assert.Nil(t, handlers, "Handlers should be nil when DB is nil")
}

func TestSetupAuditLogging_NoLoggers(t *testing.T) {
	config := IntegrationConfig{
		FileLoggingEnabled: false,
		DBLoggingEnabled:   false,
	}

	middleware, handlers, err := SetupAuditLogging(config)

	require.NoError(t, err)
	assert.NotNil(t, middleware, "Middleware should be created even without loggers")
	assert.Nil(t, handlers, "Handlers should be nil without DB logging")
}

func TestWrapRouterWithAudit(t *testing.T) {
	// Create a simple mock logger
	logger := &mockLogger{}
	middleware := NewMiddleware(logger, true)

	// Create a router with a test handler
	router := mux.NewRouter()
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap the router
	wrapped := WrapRouterWithAudit(router, middleware)

	// Test the wrapped handler
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "test response", rec.Body.String())
	assert.Len(t, logger.GetEvents(), 1)
	assert.Equal(t, "/test", logger.GetEvents()[0].Path)
}

func TestWrapRouterWithAudit_MultipleRequests(t *testing.T) {
	logger := &mockLogger{}
	middleware := NewMiddleware(logger, true)

	router := mux.NewRouter()
	router.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	router.HandleFunc("/api/modules", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	wrapped := WrapRouterWithAudit(router, middleware)

	// First request
	req1 := httptest.NewRequest("GET", "/api/users", nil)
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)

	// Second request
	req2 := httptest.NewRequest("POST", "/api/modules", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	events := logger.GetEvents()
	assert.Len(t, events, 2)
	assert.Equal(t, "/api/users", events[0].Path)
	assert.Equal(t, "/api/modules", events[1].Path)
}

func TestAddAuditRoutes_WithHandlers(t *testing.T) {
	// Create a mock store
	mockStore := &mockStore{}
	handlers := NewHandlers(mockStore)

	router := mux.NewRouter()
	AddAuditRoutes(router, handlers)

	// Verify that routes are registered
	routes := []string{
		"/audit/events",
		"/audit/export",
		"/audit/stats",
	}

	for _, route := range routes {
		req := httptest.NewRequest("GET", route, nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		// We expect either 200 or 500 (not 404), indicating the route is registered
		assert.NotEqual(t, http.StatusNotFound, rec.Code, "Route %s should be registered", route)
	}
}

func TestAddAuditRoutes_NilHandlers(t *testing.T) {
	router := mux.NewRouter()

	// Should not panic with nil handlers
	assert.NotPanics(t, func() {
		AddAuditRoutes(router, nil)
	})

	// Routes should not be registered
	req := httptest.NewRequest("GET", "/audit/events", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestIntegrationConfig_CustomValues(t *testing.T) {
	db := &sql.DB{}

	config := IntegrationConfig{
		DB:                 db,
		FileLoggingEnabled: false,
		FileLogPath:        "/custom/path",
		FileLogRotate:      false,
		FileLogMaxSize:     50 * 1024 * 1024,
		FileLogMaxFiles:    5,
		DBLoggingEnabled:   true,
		LogAllRequests:     true,
		RetentionPolicy: RetentionPolicy{
			RetentionDays:   30,
			ArchiveEnabled:  false,
			ArchivePath:     "/custom/archive",
			CompressArchive: false,
		},
	}

	assert.Equal(t, db, config.DB)
	assert.False(t, config.FileLoggingEnabled)
	assert.Equal(t, "/custom/path", config.FileLogPath)
	assert.False(t, config.FileLogRotate)
	assert.Equal(t, int64(50*1024*1024), config.FileLogMaxSize)
	assert.Equal(t, 5, config.FileLogMaxFiles)
	assert.True(t, config.DBLoggingEnabled)
	assert.True(t, config.LogAllRequests)
	assert.Equal(t, 30, config.RetentionPolicy.RetentionDays)
}

// mockStore is already defined in handlers_test.go and will be reused here
