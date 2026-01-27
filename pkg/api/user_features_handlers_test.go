package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMockDB creates a mock database for testing
func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock
}

// TestNewUserFeaturesHandlers verifies handler initialization
func TestNewUserFeaturesHandlers(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.db)
}

// TestUserFeaturesHandlers_RegisterRoutes verifies all routes are registered
func TestUserFeaturesHandlers_RegisterRoutes(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v2/saved-searches"},
		{"POST", "/api/v2/saved-searches"},
		{"GET", "/api/v2/saved-searches/123"},
		{"PUT", "/api/v2/saved-searches/123"},
		{"DELETE", "/api/v2/saved-searches/123"},
		{"GET", "/api/v2/bookmarks"},
		{"POST", "/api/v2/bookmarks"},
		{"GET", "/api/v2/bookmarks/123"},
		{"PUT", "/api/v2/bookmarks/123"},
		{"DELETE", "/api/v2/bookmarks/123"},
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

// Saved Searches Tests

func TestListSavedSearches_Empty(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "name", "query", "filters", "description", "is_shared", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT id, user_id, name, query, filters, description, is_shared, created_at, updated_at FROM saved_searches").
		WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/api/v2/saved-searches", nil)
	w := httptest.NewRecorder()

	handlers.listSavedSearches(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(0), response["count"])
}

func TestListSavedSearches_WithData(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "name", "query", "filters", "description", "is_shared", "created_at", "updated_at"}).
		AddRow(1, nil, "My Search", "test query", []byte(`{"type":"module"}`), "Test description", false, "2024-01-01", "2024-01-01").
		AddRow(2, nil, "Another Search", "another query", []byte(`{}`), "", true, "2024-01-02", "2024-01-02")

	mock.ExpectQuery("SELECT id, user_id, name, query, filters, description, is_shared, created_at, updated_at FROM saved_searches").
		WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/api/v2/saved-searches", nil)
	w := httptest.NewRecorder()

	handlers.listSavedSearches(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(2), response["count"])

	searches := response["saved_searches"].([]interface{})
	assert.Len(t, searches, 2)
}

func TestListSavedSearches_QueryError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	mock.ExpectQuery("SELECT id, user_id, name, query, filters, description, is_shared, created_at, updated_at FROM saved_searches").
		WillReturnError(errors.New("database error"))

	req := httptest.NewRequest("GET", "/api/v2/saved-searches", nil)
	w := httptest.NewRecorder()

	handlers.listSavedSearches(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateSavedSearch_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	search := SavedSearch{
		Name:        "Test Search",
		Query:       "test query",
		Description: "Test description",
		IsShared:    false,
	}
	body, err := json.Marshal(search)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(1, "2024-01-01", "2024-01-01")

	mock.ExpectQuery("INSERT INTO saved_searches").
		WithArgs(nil, "Test Search", "test query", sqlmock.AnyArg(), "Test description", false).
		WillReturnRows(rows)

	req := httptest.NewRequest("POST", "/api/v2/saved-searches", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.createSavedSearch(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response SavedSearch
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, int64(1), response.ID)
	assert.Equal(t, "Test Search", response.Name)
}

func TestCreateSavedSearch_MissingName(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	search := SavedSearch{
		Query: "test query",
	}
	body, err := json.Marshal(search)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v2/saved-searches", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.createSavedSearch(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "name")
}

func TestCreateSavedSearch_MissingQuery(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	search := SavedSearch{
		Name: "Test Search",
	}
	body, err := json.Marshal(search)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v2/saved-searches", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.createSavedSearch(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "query")
}

func TestCreateSavedSearch_InvalidJSON(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	req := httptest.NewRequest("POST", "/api/v2/saved-searches", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlers.createSavedSearch(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSavedSearch_DatabaseError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	search := SavedSearch{
		Name:  "Test Search",
		Query: "test query",
	}
	body, err := json.Marshal(search)
	require.NoError(t, err)

	mock.ExpectQuery("INSERT INTO saved_searches").
		WillReturnError(errors.New("database error"))

	req := httptest.NewRequest("POST", "/api/v2/saved-searches", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.createSavedSearch(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetSavedSearch_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "name", "query", "filters", "description", "is_shared", "created_at", "updated_at"}).
		AddRow(1, nil, "Test Search", "test query", []byte(`{"type":"module"}`), "Test description", false, "2024-01-01", "2024-01-01")

	mock.ExpectQuery("SELECT id, user_id, name, query, filters, description, is_shared, created_at, updated_at FROM saved_searches WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/api/v2/saved-searches/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.getSavedSearch(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SavedSearch
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, int64(1), response.ID)
	assert.Equal(t, "Test Search", response.Name)
}

func TestGetSavedSearch_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	mock.ExpectQuery("SELECT id, user_id, name, query, filters, description, is_shared, created_at, updated_at FROM saved_searches WHERE id = \\$1").
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/api/v2/saved-searches/999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	w := httptest.NewRecorder()

	handlers.getSavedSearch(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetSavedSearch_InvalidID(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	req := httptest.NewRequest("GET", "/api/v2/saved-searches/invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.getSavedSearch(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSavedSearch_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	search := SavedSearch{
		Name:        "Updated Search",
		Query:       "updated query",
		Description: "Updated description",
		IsShared:    true,
	}
	body, err := json.Marshal(search)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"updated_at"}).
		AddRow("2024-01-02")

	mock.ExpectQuery("UPDATE saved_searches SET name = \\$1, query = \\$2, filters = \\$3, description = \\$4, is_shared = \\$5, updated_at = NOW\\(\\) WHERE id = \\$6").
		WithArgs("Updated Search", "updated query", sqlmock.AnyArg(), "Updated description", true, int64(1)).
		WillReturnRows(rows)

	req := httptest.NewRequest("PUT", "/api/v2/saved-searches/1", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.updateSavedSearch(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SavedSearch
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, int64(1), response.ID)
	assert.Equal(t, "Updated Search", response.Name)
}

func TestUpdateSavedSearch_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	search := SavedSearch{
		Name:  "Updated Search",
		Query: "updated query",
	}
	body, err := json.Marshal(search)
	require.NoError(t, err)

	mock.ExpectQuery("UPDATE saved_searches").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), int64(999)).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("PUT", "/api/v2/saved-searches/999", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	w := httptest.NewRecorder()

	handlers.updateSavedSearch(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateSavedSearch_InvalidJSON(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	req := httptest.NewRequest("PUT", "/api/v2/saved-searches/1", bytes.NewReader([]byte("{")))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.updateSavedSearch(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteSavedSearch_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	mock.ExpectExec("DELETE FROM saved_searches WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest("DELETE", "/api/v2/saved-searches/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.deleteSavedSearch(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteSavedSearch_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	mock.ExpectExec("DELETE FROM saved_searches WHERE id = \\$1").
		WithArgs(int64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest("DELETE", "/api/v2/saved-searches/999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	w := httptest.NewRecorder()

	handlers.deleteSavedSearch(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSavedSearch_InvalidID(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	req := httptest.NewRequest("DELETE", "/api/v2/saved-searches/invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.deleteSavedSearch(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Bookmarks Tests

func TestListBookmarks_Empty(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "module_name", "version", "entity_path", "entity_type", "notes", "tags", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT id, user_id, module_name, version, entity_path, entity_type, notes, tags, created_at, updated_at FROM bookmarks").
		WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/api/v2/bookmarks", nil)
	w := httptest.NewRecorder()

	handlers.listBookmarks(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(0), response["count"])
}

func TestListBookmarks_WithData(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "module_name", "version", "entity_path", "entity_type", "notes", "tags", "created_at", "updated_at"}).
		AddRow(1, nil, "test-module", "v1.0.0", "proto.Message", "message", "Test note", "{tag1,tag2}", "2024-01-01", "2024-01-01").
		AddRow(2, nil, "another-module", "v2.0.0", "", "", "", "{}", "2024-01-02", "2024-01-02")

	mock.ExpectQuery("SELECT id, user_id, module_name, version, entity_path, entity_type, notes, tags, created_at, updated_at FROM bookmarks").
		WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/api/v2/bookmarks", nil)
	w := httptest.NewRecorder()

	handlers.listBookmarks(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(2), response["count"])
}

func TestListBookmarks_QueryError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	mock.ExpectQuery("SELECT id, user_id, module_name, version, entity_path, entity_type, notes, tags, created_at, updated_at FROM bookmarks").
		WillReturnError(errors.New("database error"))

	req := httptest.NewRequest("GET", "/api/v2/bookmarks", nil)
	w := httptest.NewRecorder()

	handlers.listBookmarks(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateBookmark_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	bookmark := Bookmark{
		ModuleName: "test-module",
		Version:    "v1.0.0",
		EntityPath: "proto.Message",
		EntityType: "message",
		Notes:      "Test note",
		Tags:       []string{"tag1", "tag2"},
	}
	body, err := json.Marshal(bookmark)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(1, "2024-01-01", "2024-01-01")

	mock.ExpectQuery("INSERT INTO bookmarks").
		WithArgs(nil, "test-module", "v1.0.0", "proto.Message", "message", "Test note", sqlmock.AnyArg()).
		WillReturnRows(rows)

	req := httptest.NewRequest("POST", "/api/v2/bookmarks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.createBookmark(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response Bookmark
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, int64(1), response.ID)
	assert.Equal(t, "test-module", response.ModuleName)
}

func TestCreateBookmark_MissingModuleName(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	bookmark := Bookmark{
		Version: "v1.0.0",
	}
	body, err := json.Marshal(bookmark)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v2/bookmarks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.createBookmark(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "module_name")
}

func TestCreateBookmark_MissingVersion(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	bookmark := Bookmark{
		ModuleName: "test-module",
	}
	body, err := json.Marshal(bookmark)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v2/bookmarks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.createBookmark(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "version")
}

func TestCreateBookmark_InvalidJSON(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	req := httptest.NewRequest("POST", "/api/v2/bookmarks", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	handlers.createBookmark(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetBookmark_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "module_name", "version", "entity_path", "entity_type", "notes", "tags", "created_at", "updated_at"}).
		AddRow(1, nil, "test-module", "v1.0.0", "proto.Message", "message", "Test note", "{tag1,tag2}", "2024-01-01", "2024-01-01")

	mock.ExpectQuery("SELECT id, user_id, module_name, version, entity_path, entity_type, notes, tags, created_at, updated_at FROM bookmarks WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/api/v2/bookmarks/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.getBookmark(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response Bookmark
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, int64(1), response.ID)
	assert.Equal(t, "test-module", response.ModuleName)
}

func TestGetBookmark_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	mock.ExpectQuery("SELECT id, user_id, module_name, version, entity_path, entity_type, notes, tags, created_at, updated_at FROM bookmarks WHERE id = \\$1").
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/api/v2/bookmarks/999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	w := httptest.NewRecorder()

	handlers.getBookmark(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateBookmark_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	bookmark := Bookmark{
		Notes: "Updated note",
		Tags:  []string{"tag3"},
	}
	body, err := json.Marshal(bookmark)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"updated_at"}).
		AddRow("2024-01-02")

	mock.ExpectQuery("UPDATE bookmarks SET notes = \\$1, tags = \\$2, updated_at = NOW\\(\\) WHERE id = \\$3").
		WithArgs("Updated note", sqlmock.AnyArg(), int64(1)).
		WillReturnRows(rows)

	req := httptest.NewRequest("PUT", "/api/v2/bookmarks/1", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.updateBookmark(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateBookmark_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	bookmark := Bookmark{
		Notes: "Updated note",
	}
	body, err := json.Marshal(bookmark)
	require.NoError(t, err)

	mock.ExpectQuery("UPDATE bookmarks").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(999)).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("PUT", "/api/v2/bookmarks/999", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	w := httptest.NewRecorder()

	handlers.updateBookmark(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteBookmark_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	mock.ExpectExec("DELETE FROM bookmarks WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest("DELETE", "/api/v2/bookmarks/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.deleteBookmark(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteBookmark_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	mock.ExpectExec("DELETE FROM bookmarks WHERE id = \\$1").
		WithArgs(int64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest("DELETE", "/api/v2/bookmarks/999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	w := httptest.NewRecorder()

	handlers.deleteBookmark(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Benchmark tests

func BenchmarkListSavedSearches(b *testing.B) {
	db, mock := setupMockDB(&testing.T{})
	defer db.Close()

	handlers := NewUserFeaturesHandlers(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "name", "query", "filters", "description", "is_shared", "created_at", "updated_at"}).
		AddRow(1, nil, "Test", "query", []byte(`{}`), "", false, "2024-01-01", "2024-01-01")

	for i := 0; i < b.N; i++ {
		mock.ExpectQuery("SELECT id, user_id, name, query, filters, description, is_shared, created_at, updated_at FROM saved_searches").
			WillReturnRows(rows)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/saved-searches", nil)
		w := httptest.NewRecorder()
		handlers.listSavedSearches(w, req)
	}
}
