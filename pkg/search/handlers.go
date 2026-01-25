package search

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/api"
)

// SearchHandlers provides HTTP handlers for search
type SearchHandlers struct {
	engine *SearchEngine
}

// NewSearchHandlers creates new search handlers
func NewSearchHandlers(storage api.Storage) *SearchHandlers {
	return &SearchHandlers{
		engine: NewSearchEngine(storage),
	}
}

// RegisterRoutes registers search routes
func (h *SearchHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/search", h.search).Methods("GET")
	router.HandleFunc("/search/modules", h.searchModules).Methods("GET")
	router.HandleFunc("/search/messages", h.searchMessages).Methods("GET")
	router.HandleFunc("/search/fields", h.searchFields).Methods("GET")
	router.HandleFunc("/search/services", h.searchServices).Methods("GET")
}

// search handles GET /search
func (h *SearchHandlers) search(w http.ResponseWriter, r *http.Request) {
	query := h.parseQuery(r)
	results, err := h.engine.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// searchModules handles GET /search/modules
func (h *SearchHandlers) searchModules(w http.ResponseWriter, r *http.Request) {
	query := h.parseQuery(r)
	query.Type = "package"
	results, err := h.engine.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// searchMessages handles GET /search/messages
func (h *SearchHandlers) searchMessages(w http.ResponseWriter, r *http.Request) {
	query := h.parseQuery(r)
	query.Type = "message"
	results, err := h.engine.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// searchFields handles GET /search/fields
func (h *SearchHandlers) searchFields(w http.ResponseWriter, r *http.Request) {
	query := h.parseQuery(r)
	query.Type = "field"
	results, err := h.engine.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// searchServices handles GET /search/services
func (h *SearchHandlers) searchServices(w http.ResponseWriter, r *http.Request) {
	query := h.parseQuery(r)
	query.Type = "service"
	results, err := h.engine.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// parseQuery extracts SearchQuery from HTTP request
func (h *SearchHandlers) parseQuery(r *http.Request) SearchQuery {
	q := SearchQuery{
		Query:   r.URL.Query().Get("q"),
		Type:    r.URL.Query().Get("type"),
		Module:  r.URL.Query().Get("module"),
		Version: r.URL.Query().Get("version"),
		Limit:   50, // Default limit
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			q.Limit = limit
		}
	}

	// Parse deprecated
	if depStr := r.URL.Query().Get("deprecated"); depStr != "" {
		q.Deprecated = depStr == "true" || depStr == "1"
	}

	return q
}
