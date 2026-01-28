package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMockAnalyticsDB creates a mock database for analytics testing
func setupMockAnalyticsDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock
}

// TestNewAnalyticsHandlers verifies handler initialization
func TestNewAnalyticsHandlers(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.service)

	mock.ExpectClose()
}

// TestAnalyticsHandlers_RegisterRoutes verifies all routes are registered
func TestAnalyticsHandlers_RegisterRoutes(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v2/analytics/overview"},
		{"GET", "/api/v2/analytics/modules/popular"},
		{"GET", "/api/v2/analytics/modules/trending"},
		{"GET", "/api/v2/analytics/modules/test-module/stats"},
		{"GET", "/api/v2/analytics/modules/test-module/health"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			var match mux.RouteMatch
			matched := router.Match(req, &match)
			assert.True(t, matched, "Route %s %s should be registered", tt.method, tt.path)
		})
	}

	mock.ExpectClose()
}

// TestGetOverview_Success tests successful overview retrieval
func TestGetOverview_Success(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	// Mock the database queries
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(500))
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"downloads_24h", "downloads_7d", "downloads_30d"}).
			AddRow(1000, 5000, 20000))
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"active_users_24h", "active_users_7d"}).
			AddRow(50, 200))
	mock.ExpectQuery("SELECT language").
		WillReturnRows(sqlmock.NewRows([]string{"language"}).AddRow("protobuf"))
	mock.ExpectQuery("SELECT AVG").
		WillReturnRows(sqlmock.NewRows([]string{"avg"}).AddRow(150.5))
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"rate"}).AddRow(0.85))

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/overview", nil)
	w := httptest.NewRecorder()

	handlers.getOverview(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response analytics.OverviewResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, int64(100), response.TotalModules)

	mock.ExpectClose()
}

// TestGetOverview_DatabaseError tests error handling for overview endpoint
func TestGetOverview_DatabaseError(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrConnDone)

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/overview", nil)
	w := httptest.NewRecorder()

	handlers.getOverview(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mock.ExpectClose()
}

// TestGetPopularModules_Success tests successful popular modules retrieval
func TestGetPopularModules_Success(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"module_name", "total_downloads", "total_views", "active_days", "avg_daily_downloads"}).
		AddRow("test-module-1", 1000, 2000, 30, 33.3).
		AddRow("test-module-2", 800, 1600, 25, 32.0)

	mock.ExpectQuery("SELECT").
		WillReturnRows(rows)

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/popular?period=30d&limit=10", nil)
	w := httptest.NewRecorder()

	handlers.getPopularModules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []analytics.PopularModule
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response, 2)
	assert.Equal(t, "test-module-1", response[0].ModuleName)

	mock.ExpectClose()
}

// TestGetPopularModules_DefaultParameters tests default parameter handling
func TestGetPopularModules_DefaultParameters(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"module_name", "total_downloads", "total_views", "active_days", "avg_daily_downloads"}))

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/popular", nil)
	w := httptest.NewRecorder()

	handlers.getPopularModules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mock.ExpectClose()
}

// TestGetPopularModules_InvalidLimit tests invalid limit parameter
func TestGetPopularModules_InvalidLimit(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/popular?limit=invalid", nil)
	w := httptest.NewRecorder()

	handlers.getPopularModules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mock.ExpectClose()
}

// TestGetPopularModules_LimitCapping tests that limit is capped at 100
func TestGetPopularModules_LimitCapping(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").
		WithArgs(30, 100).
		WillReturnRows(sqlmock.NewRows([]string{"module_name", "total_downloads", "total_views", "active_days", "avg_daily_downloads"}))

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/popular?limit=200", nil)
	w := httptest.NewRecorder()

	handlers.getPopularModules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mock.ExpectClose()
}

// TestGetPopularModules_DatabaseError tests error handling
func TestGetPopularModules_DatabaseError(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrConnDone)

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/popular", nil)
	w := httptest.NewRecorder()

	handlers.getPopularModules(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mock.ExpectClose()
}

// TestGetTrendingModules_Success tests successful trending modules retrieval
func TestGetTrendingModules_Success(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"module_name", "current_downloads", "previous_downloads", "growth_rate"}).
		AddRow("trending-module-1", 500, 100, 4.0).
		AddRow("trending-module-2", 300, 150, 1.0)

	mock.ExpectQuery("SELECT").
		WillReturnRows(rows)

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/trending?limit=20", nil)
	w := httptest.NewRecorder()

	handlers.getTrendingModules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []analytics.TrendingModule
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response, 2)
	assert.Equal(t, "trending-module-1", response[0].ModuleName)

	mock.ExpectClose()
}

// TestGetTrendingModules_InvalidLimit tests invalid limit parameter
func TestGetTrendingModules_InvalidLimit(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/trending?limit=invalid", nil)
	w := httptest.NewRecorder()

	handlers.getTrendingModules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mock.ExpectClose()
}

// TestGetTrendingModules_LimitCapping tests that limit is capped at 50
func TestGetTrendingModules_LimitCapping(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").
		WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"module_name", "current_downloads", "previous_downloads", "growth_rate"}))

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/trending?limit=100", nil)
	w := httptest.NewRecorder()

	handlers.getTrendingModules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mock.ExpectClose()
}

// TestGetTrendingModules_DatabaseError tests error handling
func TestGetTrendingModules_DatabaseError(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrConnDone)

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/trending", nil)
	w := httptest.NewRecorder()

	handlers.getTrendingModules(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mock.ExpectClose()
}

// TestGetModuleStats_Success tests module stats endpoint calls service correctly
func TestGetModuleStats_Success(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	// Mock all expected queries
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"v", "d", "u", "a"}).AddRow(1000, 500, 100, 120))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"date", "download_count"}))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"language", "count"}))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"version", "count"}))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"rate"}).AddRow(0.95))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"max"}))

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/analytics/modules/{name}/stats", handlers.getModuleStats)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/my-module/stats?period=7d", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mock.ExpectClose()
}

// TestGetModuleStats_DefaultPeriod tests default period parameter
func TestGetModuleStats_DefaultPeriod(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").WithArgs("test", 30).
		WillReturnRows(sqlmock.NewRows([]string{"v", "d", "u", "a"}).AddRow(0, 0, 0, nil))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"date", "download_count"}))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"language", "count"}))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"version", "count"}))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"rate"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"max"}))

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/analytics/modules/{name}/stats", handlers.getModuleStats)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/test/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mock.ExpectClose()
}

// TestGetModuleStats_DatabaseError tests error handling
func TestGetModuleStats_DatabaseError(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrConnDone)

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/analytics/modules/{name}/stats", handlers.getModuleStats)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/test/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mock.ExpectClose()
}

// TestGetModuleHealth_Success tests successful module health retrieval
func TestGetModuleHealth_Success(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	// Mock all health calculation queries
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"mc", "ec", "sc", "fc", "mtc"}).AddRow(10, 5, 2, 50, 10))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"entity_name"}))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/analytics/modules/{name}/health", handlers.getModuleHealth)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/my-module/health?version=v1.0.0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mock.ExpectClose()
}

// TestGetModuleHealth_DefaultVersion tests default version handling
func TestGetModuleHealth_DefaultVersion(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	// Expect query for latest version
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("v1.0.0"))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"mc", "ec", "sc", "fc", "mtc"}).AddRow(10, 5, 2, 50, 10))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"entity_name"}))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/analytics/modules/{name}/health", handlers.getModuleHealth)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/test/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mock.ExpectClose()
}

// TestGetModuleHealth_DatabaseError tests error handling
func TestGetModuleHealth_DatabaseError(t *testing.T) {
	db, mock := setupMockAnalyticsDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrNoRows)

	service := analytics.NewService(db)
	handlers := NewAnalyticsHandlers(service)

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/analytics/modules/{name}/health", handlers.getModuleHealth)

	req := httptest.NewRequest("GET", "/api/v2/analytics/modules/test/health?version=v1.0.0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mock.ExpectClose()
}

// TestAnalyticsHandlers_MethodNotAllowed tests that wrong methods are rejected
func TestAnalyticsHandlers_MethodNotAllowed(t *testing.T) {
	handlers := NewAnalyticsHandlers(nil)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v2/analytics/overview"},
		{"PUT", "/api/v2/analytics/modules/popular"},
		{"DELETE", "/api/v2/analytics/modules/trending"},
		{"PATCH", "/api/v2/analytics/modules/test/stats"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}
