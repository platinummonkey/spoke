package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/async"
	"github.com/platinummonkey/spoke/pkg/search"
)

// EnhancedSearchHandlers provides HTTP handlers for advanced search
type EnhancedSearchHandlers struct {
	service *search.SearchService
}

// NewEnhancedSearchHandlers creates new enhanced search handlers
func NewEnhancedSearchHandlers(db *sql.DB) *EnhancedSearchHandlers {
	return &EnhancedSearchHandlers{
		service: search.NewSearchService(db),
	}
}

// RegisterRoutes registers enhanced search routes
func (h *EnhancedSearchHandlers) RegisterRoutes(router *mux.Router) {
	// v2 API endpoints
	router.HandleFunc("/api/v2/search", h.search).Methods("GET")
	router.HandleFunc("/api/v2/search/suggestions", h.getSuggestions).Methods("GET")

	// Also register under /search for convenience
	router.HandleFunc("/search/advanced", h.search).Methods("GET")
	router.HandleFunc("/search/suggest", h.getSuggestions).Methods("GET")
}

// search handles GET /api/v2/search
// Query parameters:
//   - q: search query with filters (e.g., "user entity:message type:string")
//   - limit: max results (default: 50, max: 1000)
//   - offset: pagination offset (default: 0)
func (h *EnhancedSearchHandlers) search(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	// Parse limit
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Parse offset
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Execute search
	req := search.SearchRequest{
		Query:  query,
		Limit:  limit,
		Offset: offset,
	}

	response, err := h.service.Search(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record search in history (async, don't block response)
	async.SafeGo(r.Context(), 5*time.Second, "record search", func(ctx context.Context) error {
		durationMs := int(time.Since(startTime).Milliseconds())
		return h.service.RecordSearch(ctx, query, response.TotalCount, durationMs)
	})

	// Return results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getSuggestions handles GET /api/v2/search/suggestions
// Query parameters:
//   - prefix: search query prefix (e.g., "user")
//   - limit: max suggestions (default: 5, max: 20)
func (h *EnhancedSearchHandlers) getSuggestions(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	if prefix == "" {
		http.Error(w, "missing query parameter 'prefix'", http.StatusBadRequest)
		return
	}

	// Parse limit
	limit := 5
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Get suggestions
	suggestions, err := h.service.GetSuggestions(r.Context(), prefix, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return suggestions
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"prefix":      prefix,
		"suggestions": suggestions,
	})
}
