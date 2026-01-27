package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/httputil"
)

// UserFeaturesHandlers provides HTTP handlers for user features (saved searches, bookmarks)
type UserFeaturesHandlers struct {
	db *sql.DB
}

// NewUserFeaturesHandlers creates new user features handlers
func NewUserFeaturesHandlers(db *sql.DB) *UserFeaturesHandlers {
	return &UserFeaturesHandlers{db: db}
}

// RegisterRoutes registers user features routes
func (h *UserFeaturesHandlers) RegisterRoutes(router *mux.Router) {
	// Saved searches endpoints
	router.HandleFunc("/api/v2/saved-searches", h.listSavedSearches).Methods("GET")
	router.HandleFunc("/api/v2/saved-searches", h.createSavedSearch).Methods("POST")
	router.HandleFunc("/api/v2/saved-searches/{id}", h.getSavedSearch).Methods("GET")
	router.HandleFunc("/api/v2/saved-searches/{id}", h.updateSavedSearch).Methods("PUT")
	router.HandleFunc("/api/v2/saved-searches/{id}", h.deleteSavedSearch).Methods("DELETE")

	// Bookmarks endpoints
	router.HandleFunc("/api/v2/bookmarks", h.listBookmarks).Methods("GET")
	router.HandleFunc("/api/v2/bookmarks", h.createBookmark).Methods("POST")
	router.HandleFunc("/api/v2/bookmarks/{id}", h.getBookmark).Methods("GET")
	router.HandleFunc("/api/v2/bookmarks/{id}", h.updateBookmark).Methods("PUT")
	router.HandleFunc("/api/v2/bookmarks/{id}", h.deleteBookmark).Methods("DELETE")
}

// SavedSearch represents a saved search query
type SavedSearch struct {
	ID          int64                  `json:"id"`
	UserID      *int64                 `json:"user_id,omitempty"`
	Name        string                 `json:"name"`
	Query       string                 `json:"query"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Description string                 `json:"description,omitempty"`
	IsShared    bool                   `json:"is_shared"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// Bookmark represents a bookmarked module/entity
type Bookmark struct {
	ID         int64    `json:"id"`
	UserID     *int64   `json:"user_id,omitempty"`
	ModuleName string   `json:"module_name"`
	Version    string   `json:"version"`
	EntityPath string   `json:"entity_path,omitempty"`
	EntityType string   `json:"entity_type,omitempty"`
	Notes      string   `json:"notes,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

// listSavedSearches handles GET /api/v2/saved-searches
func (h *UserFeaturesHandlers) listSavedSearches(w http.ResponseWriter, r *http.Request) {
	// For now, return all saved searches (no user authentication)
	// Future: filter by user_id from authentication
	query := `
		SELECT id, user_id, name, query, filters, description, is_shared, created_at, updated_at
		FROM saved_searches
		WHERE user_id IS NULL
		ORDER BY created_at DESC
	`

	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}
	defer rows.Close()

	searches := make([]SavedSearch, 0)
	for rows.Next() {
		var search SavedSearch
		var filtersJSON []byte

		err := rows.Scan(
			&search.ID,
			&search.UserID,
			&search.Name,
			&search.Query,
			&filtersJSON,
			&search.Description,
			&search.IsShared,
			&search.CreatedAt,
			&search.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if len(filtersJSON) > 0 {
			json.Unmarshal(filtersJSON, &search.Filters)
		}

		searches = append(searches, search)
	}

	httputil.WriteSuccess(w, map[string]interface{}{
		"saved_searches": searches,
		"count":          len(searches),
	})
}

// createSavedSearch handles POST /api/v2/saved-searches
func (h *UserFeaturesHandlers) createSavedSearch(w http.ResponseWriter, r *http.Request) {
	var req SavedSearch
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	// Validate required fields
	if !httputil.RequireNonEmpty(w, req.Name, "name") {
		return
	}
	if !httputil.RequireNonEmpty(w, req.Query, "query") {
		return
	}

	filtersJSON, _ := json.Marshal(req.Filters)

	query := `
		INSERT INTO saved_searches (user_id, name, query, filters, description, is_shared)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := h.db.QueryRowContext(
		r.Context(),
		query,
		req.UserID,
		req.Name,
		req.Query,
		filtersJSON,
		req.Description,
		req.IsShared,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)

	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w, req)
}

// getSavedSearch handles GET /api/v2/saved-searches/{id}
func (h *UserFeaturesHandlers) getSavedSearch(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	query := `
		SELECT id, user_id, name, query, filters, description, is_shared, created_at, updated_at
		FROM saved_searches
		WHERE id = $1
	`

	var search SavedSearch
	var filtersJSON []byte

	err := h.db.QueryRowContext(r.Context(), query, id).Scan(
		&search.ID,
		&search.UserID,
		&search.Name,
		&search.Query,
		&filtersJSON,
		&search.Description,
		&search.IsShared,
		&search.CreatedAt,
		&search.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		httputil.WriteNotFoundError(w, "saved search not found")
		return
	}
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	if len(filtersJSON) > 0 {
		json.Unmarshal(filtersJSON, &search.Filters)
	}

	httputil.WriteSuccess(w, search)
}

// updateSavedSearch handles PUT /api/v2/saved-searches/{id}
func (h *UserFeaturesHandlers) updateSavedSearch(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	var req SavedSearch
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	filtersJSON, _ := json.Marshal(req.Filters)

	query := `
		UPDATE saved_searches
		SET name = $1, query = $2, filters = $3, description = $4, is_shared = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING updated_at
	`

	err := h.db.QueryRowContext(
		r.Context(),
		query,
		req.Name,
		req.Query,
		filtersJSON,
		req.Description,
		req.IsShared,
		id,
	).Scan(&req.UpdatedAt)

	if err == sql.ErrNoRows {
		httputil.WriteNotFoundError(w, "saved search not found")
		return
	}
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	req.ID = id
	httputil.WriteSuccess(w, req)
}

// deleteSavedSearch handles DELETE /api/v2/saved-searches/{id}
func (h *UserFeaturesHandlers) deleteSavedSearch(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	query := `DELETE FROM saved_searches WHERE id = $1`
	result, err := h.db.ExecContext(r.Context(), query, id)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		httputil.WriteNotFoundError(w, "saved search not found")
		return
	}

	httputil.WriteNoContent(w)
}

// listBookmarks handles GET /api/v2/bookmarks
func (h *UserFeaturesHandlers) listBookmarks(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, user_id, module_name, version, entity_path, entity_type, notes, tags, created_at, updated_at
		FROM bookmarks
		WHERE user_id IS NULL
		ORDER BY created_at DESC
	`

	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}
	defer rows.Close()

	bookmarks := make([]Bookmark, 0)
	for rows.Next() {
		var bookmark Bookmark
		var tags sql.NullString

		err := rows.Scan(
			&bookmark.ID,
			&bookmark.UserID,
			&bookmark.ModuleName,
			&bookmark.Version,
			&bookmark.EntityPath,
			&bookmark.EntityType,
			&bookmark.Notes,
			&tags,
			&bookmark.CreatedAt,
			&bookmark.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if tags.Valid && tags.String != "{}" {
			// Parse PostgreSQL array format
			bookmark.Tags = parsePostgresArray(tags.String)
		}

		bookmarks = append(bookmarks, bookmark)
	}

	httputil.WriteSuccess(w, map[string]interface{}{
		"bookmarks": bookmarks,
		"count":     len(bookmarks),
	})
}

// createBookmark handles POST /api/v2/bookmarks
func (h *UserFeaturesHandlers) createBookmark(w http.ResponseWriter, r *http.Request) {
	var req Bookmark
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ModuleName == "" || req.Version == "" {
		http.Error(w, "module_name and version are required", http.StatusBadRequest)
		return
	}

	// Convert tags to PostgreSQL array format
	tagsArray := formatPostgresArray(req.Tags)

	query := `
		INSERT INTO bookmarks (user_id, module_name, version, entity_path, entity_type, notes, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, module_name, version, entity_path) DO UPDATE
		SET notes = EXCLUDED.notes, tags = EXCLUDED.tags, updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	err := h.db.QueryRowContext(
		r.Context(),
		query,
		req.UserID,
		req.ModuleName,
		req.Version,
		req.EntityPath,
		req.EntityType,
		req.Notes,
		tagsArray,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)

	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w, req)
}

// getBookmark handles GET /api/v2/bookmarks/{id}
func (h *UserFeaturesHandlers) getBookmark(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	query := `
		SELECT id, user_id, module_name, version, entity_path, entity_type, notes, tags, created_at, updated_at
		FROM bookmarks
		WHERE id = $1
	`

	var bookmark Bookmark
	var tags sql.NullString

	err := h.db.QueryRowContext(r.Context(), query, id).Scan(
		&bookmark.ID,
		&bookmark.UserID,
		&bookmark.ModuleName,
		&bookmark.Version,
		&bookmark.EntityPath,
		&bookmark.EntityType,
		&bookmark.Notes,
		&tags,
		&bookmark.CreatedAt,
		&bookmark.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		httputil.WriteNotFoundError(w, "bookmark not found")
		return
	}
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	if tags.Valid && tags.String != "{}" {
		bookmark.Tags = parsePostgresArray(tags.String)
	}

	httputil.WriteSuccess(w, bookmark)
}

// updateBookmark handles PUT /api/v2/bookmarks/{id}
func (h *UserFeaturesHandlers) updateBookmark(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	var req Bookmark
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	tagsArray := formatPostgresArray(req.Tags)

	query := `
		UPDATE bookmarks
		SET notes = $1, tags = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at
	`

	err := h.db.QueryRowContext(r.Context(), query, req.Notes, tagsArray, id).Scan(&req.UpdatedAt)

	if err == sql.ErrNoRows {
		httputil.WriteNotFoundError(w, "bookmark not found")
		return
	}
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	req.ID = id
	httputil.WriteSuccess(w, req)
}

// deleteBookmark handles DELETE /api/v2/bookmarks/{id}
func (h *UserFeaturesHandlers) deleteBookmark(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	query := `DELETE FROM bookmarks WHERE id = $1`
	result, err := h.db.ExecContext(r.Context(), query, id)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		httputil.WriteNotFoundError(w, "bookmark not found")
		return
	}

	httputil.WriteNoContent(w)
}

// Helper functions for PostgreSQL array handling
func parsePostgresArray(s string) []string {
	if s == "" || s == "{}" {
		return []string{}
	}
	// Remove braces and split by comma
	s = s[1 : len(s)-1]
	if s == "" {
		return []string{}
	}
	return []string{s} // Simplified for now, would need proper parsing for complex arrays
}

func formatPostgresArray(tags []string) string {
	if len(tags) == 0 {
		return "{}"
	}
	result := "{"
	for i, tag := range tags {
		if i > 0 {
			result += ","
		}
		result += tag
	}
	result += "}"
	return result
}
