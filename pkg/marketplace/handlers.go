package marketplace

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for the marketplace API
type Handlers struct {
	service *Service
}

// NewHandlers creates new marketplace handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{
		service: service,
	}
}

// RegisterRoutes registers all marketplace routes
func (h *Handlers) RegisterRoutes(r *mux.Router) {
	// Plugin discovery
	r.HandleFunc("/api/v1/plugins", h.ListPlugins).Methods("GET")
	r.HandleFunc("/api/v1/plugins/search", h.SearchPlugins).Methods("GET")
	r.HandleFunc("/api/v1/plugins/trending", h.GetTrendingPlugins).Methods("GET")
	r.HandleFunc("/api/v1/plugins/{id}", h.GetPlugin).Methods("GET")

	// Plugin versions
	r.HandleFunc("/api/v1/plugins/{id}/versions", h.ListVersions).Methods("GET")
	r.HandleFunc("/api/v1/plugins/{id}/versions/{version}", h.GetVersion).Methods("GET")
	r.HandleFunc("/api/v1/plugins/{id}/versions/{version}/download", h.DownloadPlugin).Methods("GET")

	// Plugin reviews
	r.HandleFunc("/api/v1/plugins/{id}/reviews", h.ListReviews).Methods("GET")
	r.HandleFunc("/api/v1/plugins/{id}/reviews", h.CreateReview).Methods("POST")

	// Plugin installation tracking
	r.HandleFunc("/api/v1/plugins/{id}/install", h.RecordInstallation).Methods("POST")
	r.HandleFunc("/api/v1/plugins/{id}/uninstall", h.RecordUninstallation).Methods("POST")

	// Plugin submission (authenticated)
	r.HandleFunc("/api/v1/plugins", h.SubmitPlugin).Methods("POST")
	r.HandleFunc("/api/v1/plugins/{id}/versions", h.SubmitVersion).Methods("POST")

	// Plugin stats
	r.HandleFunc("/api/v1/plugins/{id}/stats", h.GetPluginStats).Methods("GET")
}

// ListPlugins handles GET /api/v1/plugins
func (h *Handlers) ListPlugins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	req := &PluginListRequest{
		Type:          r.URL.Query().Get("type"),
		SecurityLevel: r.URL.Query().Get("security_level"),
		Search:        r.URL.Query().Get("q"),
		SortBy:        r.URL.Query().Get("sort_by"),
		SortOrder:     r.URL.Query().Get("sort_order"),
	}

	// Parse limit and offset
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err == nil {
			req.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err == nil {
			req.Offset = offset
		}
	}

	// Get plugins
	resp, err := h.service.ListPlugins(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// SearchPlugins handles GET /api/v1/plugins/search
func (h *Handlers) SearchPlugins(w http.ResponseWriter, r *http.Request) {
	// For now, delegate to ListPlugins with search parameter
	h.ListPlugins(w, r)
}

// GetTrendingPlugins handles GET /api/v1/plugins/trending
func (h *Handlers) GetTrendingPlugins(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement trending logic based on growth rate
	// For now, return most downloaded plugins
	ctx := r.Context()

	req := &PluginListRequest{
		SortBy:    "downloads",
		SortOrder: "desc",
		Limit:     10,
	}

	resp, err := h.service.ListPlugins(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetPlugin handles GET /api/v1/plugins/{id}
func (h *Handlers) GetPlugin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	pluginID := vars["id"]

	plugin, err := h.service.GetPlugin(ctx, pluginID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plugin)
}

// ListVersions handles GET /api/v1/plugins/{id}/versions
func (h *Handlers) ListVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	pluginID := vars["id"]

	versions, err := h.service.ListPluginVersions(ctx, pluginID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

// GetVersion handles GET /api/v1/plugins/{id}/versions/{version}
func (h *Handlers) GetVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	pluginID := vars["id"]
	version := vars["version"]

	versions, err := h.service.ListPluginVersions(ctx, pluginID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Find specific version
	for _, v := range versions {
		if v.Version == version {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(v)
			return
		}
	}

	http.Error(w, "version not found", http.StatusNotFound)
}

// DownloadPlugin handles GET /api/v1/plugins/{id}/versions/{version}/download
func (h *Handlers) DownloadPlugin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	pluginID := vars["id"]
	version := vars["version"]

	// Record download
	if err := h.service.RecordDownload(ctx, pluginID, version); err != nil {
		// Log error but don't fail the download
		// TODO: Add proper logging
	}

	// Get download URL and redirect
	versions, err := h.service.ListPluginVersions(ctx, pluginID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Find version
	for _, v := range versions {
		if v.Version == version {
			http.Redirect(w, r, v.DownloadURL, http.StatusFound)
			return
		}
	}

	http.Error(w, "version not found", http.StatusNotFound)
}

// ListReviews handles GET /api/v1/plugins/{id}/reviews
func (h *Handlers) ListReviews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	pluginID := vars["id"]

	// Parse pagination
	limit := 20
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	reviews, err := h.service.ListReviews(ctx, pluginID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

// CreateReview handles POST /api/v1/plugins/{id}/reviews
func (h *Handlers) CreateReview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	pluginID := vars["id"]

	var req PluginReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Get user ID from authentication context
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	review := &PluginReview{
		PluginID: pluginID,
		UserID:   userID,
		Rating:   req.Rating,
		Review:   req.Review,
	}

	if err := h.service.CreateReview(ctx, review); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// RecordInstallation handles POST /api/v1/plugins/{id}/install
func (h *Handlers) RecordInstallation(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement installation tracking
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// RecordUninstallation handles POST /api/v1/plugins/{id}/uninstall
func (h *Handlers) RecordUninstallation(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement uninstallation tracking
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// SubmitPlugin handles POST /api/v1/plugins
func (h *Handlers) SubmitPlugin(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement plugin submission with authentication and validation
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// SubmitVersion handles POST /api/v1/plugins/{id}/versions
func (h *Handlers) SubmitVersion(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement version submission
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// GetPluginStats handles GET /api/v1/plugins/{id}/stats
func (h *Handlers) GetPluginStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement statistics aggregation
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
