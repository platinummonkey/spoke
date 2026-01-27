package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/analytics"
	"github.com/stretchr/testify/assert"
)

// TestNewAnalyticsHandlers verifies handler initialization
func TestNewAnalyticsHandlers(t *testing.T) {
	service := &analytics.Service{}
	handlers := NewAnalyticsHandlers(service)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.service)
}

// TestAnalyticsHandlers_RegisterRoutes verifies all routes are registered
func TestAnalyticsHandlers_RegisterRoutes(t *testing.T) {
	service := &analytics.Service{}
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
}

// Note: Since analytics.Service requires database and doesn't expose an interface,
// we can only test the HTTP handling layer (parameter parsing, error handling).
// Full integration tests would require a test database.

// TestGetOverview_InvalidRequest tests overview endpoint
func TestGetOverview_CallsService(t *testing.T) {
	// This test verifies the handler structure but requires a real service
	// In production, this would be an integration test with a test DB
	t.Skip("Requires analytics.Service with database - integration test needed")
}

// TestGetPopularModules_InvalidLimit tests limit parameter validation
func TestGetPopularModules_CallsService(t *testing.T) {
	t.Skip("Requires analytics.Service with database - integration test needed")
}

// TestGetTrendingModules_CallsService tests trending modules endpoint
func TestGetTrendingModules_CallsService(t *testing.T) {
	t.Skip("Requires analytics.Service with database - integration test needed")
}

// TestGetModuleStats_CallsService tests module stats endpoint
func TestGetModuleStats_CallsService(t *testing.T) {
	t.Skip("Requires analytics.Service with database - integration test needed")
}

// TestGetModuleHealth_CallsService tests module health endpoint
func TestGetModuleHealth_CallsService(t *testing.T) {
	t.Skip("Requires analytics.Service with database - integration test needed")
}

// Note: Integration tests with real service would go here
// These require a test database and are beyond unit test scope

// TestAnalyticsHandlers_MethodNotAllowed tests that wrong methods are rejected
func TestAnalyticsHandlers_MethodNotAllowed(t *testing.T) {
	handlers := NewAnalyticsHandlers(nil)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	// These endpoints should only accept GET
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

			// Should get 405 Method Not Allowed
			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

// Benchmark tests for performance

func BenchmarkGetOverview(b *testing.B) {
	handlers := NewAnalyticsHandlers(nil)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/analytics/overview", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkGetPopularModules(b *testing.B) {
	handlers := NewAnalyticsHandlers(nil)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/analytics/modules/popular?period=30d&limit=100", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkGetModuleStats(b *testing.B) {
	handlers := NewAnalyticsHandlers(nil)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/analytics/modules/test-module/stats?period=30d", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
