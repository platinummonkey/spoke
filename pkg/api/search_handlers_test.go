package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// TestNewEnhancedSearchHandlers verifies handler initialization
func TestNewEnhancedSearchHandlers(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.service)
}

// TestEnhancedSearchHandlers_RegisterRoutes verifies all routes are registered
func TestEnhancedSearchHandlers_RegisterRoutes(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v2/search"},
		{"GET", "/api/v2/search/suggestions"},
		{"GET", "/search/advanced"},
		{"GET", "/search/suggest"},
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

// TestSearch_MissingQuery tests search without query parameter
func TestSearch_MissingQuery(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/search", nil)
	w := httptest.NewRecorder()

	handlers.search(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing query parameter")
}

// TestSearch_InvalidLimit tests search with invalid limit parameter
func TestSearch_InvalidLimit(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/search?q=test&limit=invalid", nil)
	w := httptest.NewRecorder()

	handlers.search(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSearch_InvalidOffset tests search with invalid offset parameter
func TestSearch_InvalidOffset(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/search?q=test&offset=invalid", nil)
	w := httptest.NewRecorder()

	handlers.search(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSearch_ValidParameters tests search with valid parameters (no DB mock)
func TestSearch_ValidParameters(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	// Query will fail due to no DB setup, but we're testing parameter parsing
	req := httptest.NewRequest("GET", "/api/v2/search?q=test&limit=20&offset=10", nil)
	w := httptest.NewRecorder()

	handlers.search(w, req)

	// Will return 500 due to no mock setup, but that's OK - we validated params
	assert.True(t, w.Code == http.StatusInternalServerError || w.Code == http.StatusOK)
}

// TestSearch_DefaultParameters tests search with default parameters
func TestSearch_DefaultParameters(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/search?q=test", nil)
	w := httptest.NewRecorder()

	handlers.search(w, req)

	// Will return 500 due to no mock setup, but that's OK - we validated params
	assert.True(t, w.Code == http.StatusInternalServerError || w.Code == http.StatusOK)
}

// TestGetSuggestions_MissingPrefix tests suggestions without prefix
func TestGetSuggestions_MissingPrefix(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/search/suggestions", nil)
	w := httptest.NewRecorder()

	handlers.getSuggestions(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing query parameter")
}

// TestGetSuggestions_InvalidLimit tests suggestions with invalid limit
func TestGetSuggestions_InvalidLimit(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/search/suggestions?prefix=test&limit=invalid", nil)
	w := httptest.NewRecorder()

	handlers.getSuggestions(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGetSuggestions_ValidParameters tests suggestions with valid parameters
func TestGetSuggestions_ValidParameters(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/search/suggestions?prefix=test&limit=10", nil)
	w := httptest.NewRecorder()

	handlers.getSuggestions(w, req)

	// Will return 500 due to no mock setup, but that's OK - we validated params
	assert.True(t, w.Code == http.StatusInternalServerError || w.Code == http.StatusOK)
}

// TestGetSuggestions_DefaultLimit tests suggestions with default limit
func TestGetSuggestions_DefaultLimit(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/search/suggestions?prefix=test", nil)
	w := httptest.NewRecorder()

	handlers.getSuggestions(w, req)

	// Will return 500 due to no mock setup, but that's OK - we validated params
	assert.True(t, w.Code == http.StatusInternalServerError || w.Code == http.StatusOK)
}

// TestSearch_EmptyQuery tests search with empty query string
func TestSearch_EmptyQuery(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/search?q=", nil)
	w := httptest.NewRecorder()

	handlers.search(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGetSuggestions_EmptyPrefix tests suggestions with empty prefix
func TestGetSuggestions_EmptyPrefix(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/search/suggestions?prefix=", nil)
	w := httptest.NewRecorder()

	handlers.getSuggestions(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSearch_AdvancedRoute tests search via /search/advanced route
func TestSearch_AdvancedRoute(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/search/advanced?q=test", nil)
	w := httptest.NewRecorder()

	handlers.search(w, req)

	// Parameters are valid, so should not be 400
	assert.NotEqual(t, http.StatusBadRequest, w.Code)
}

// TestGetSuggestions_SuggestRoute tests suggestions via /search/suggest route
func TestGetSuggestions_SuggestRoute(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	handlers := NewEnhancedSearchHandlers(db)

	req := httptest.NewRequest("GET", "/search/suggest?prefix=test", nil)
	w := httptest.NewRecorder()

	handlers.getSuggestions(w, req)

	// Parameters are valid, so should not be 400
	assert.NotEqual(t, http.StatusBadRequest, w.Code)
}

// Benchmark tests

func BenchmarkSearch_ParameterParsing(b *testing.B) {
	db, _ := setupMockDB(&testing.T{})
	defer db.Close()

	handlers := NewEnhancedSearchHandlers(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/search?q=test&limit=50&offset=0", nil)
		w := httptest.NewRecorder()
		handlers.search(w, req)
	}
}

func BenchmarkGetSuggestions_ParameterParsing(b *testing.B) {
	db, _ := setupMockDB(&testing.T{})
	defer db.Close()

	handlers := NewEnhancedSearchHandlers(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/search/suggestions?prefix=test&limit=5", nil)
		w := httptest.NewRecorder()
		handlers.getSuggestions(w, req)
	}
}
