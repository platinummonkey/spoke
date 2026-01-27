package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/analytics"
	"github.com/platinummonkey/spoke/pkg/httputil"
)

// AnalyticsHandlers provides analytics API endpoints
type AnalyticsHandlers struct {
	service *analytics.Service
}

// NewAnalyticsHandlers creates a new analytics handlers instance
func NewAnalyticsHandlers(service *analytics.Service) *AnalyticsHandlers {
	return &AnalyticsHandlers{
		service: service,
	}
}

// RegisterRoutes registers analytics API routes
func (h *AnalyticsHandlers) RegisterRoutes(r *mux.Router) {
	// Overview and high-level metrics
	r.HandleFunc("/api/v2/analytics/overview", h.getOverview).Methods("GET")

	// Module analytics
	r.HandleFunc("/api/v2/analytics/modules/popular", h.getPopularModules).Methods("GET")
	r.HandleFunc("/api/v2/analytics/modules/trending", h.getTrendingModules).Methods("GET")
	r.HandleFunc("/api/v2/analytics/modules/{name}/stats", h.getModuleStats).Methods("GET")

	// Health scoring
	r.HandleFunc("/api/v2/analytics/modules/{name}/health", h.getModuleHealth).Methods("GET")

	// Performance analytics (to be implemented in Phase 6)
	// r.HandleFunc("/api/v2/analytics/performance/compilation", h.getCompilationPerformance).Methods("GET")
	// r.HandleFunc("/api/v2/analytics/languages", h.getLanguageStats).Methods("GET")
}

// getOverview handles GET /api/v2/analytics/overview
// Returns high-level KPIs for the analytics dashboard
func (h *AnalyticsHandlers) getOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	overview, err := h.service.GetOverview(ctx)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, overview)
}

// getPopularModules handles GET /api/v2/analytics/modules/popular
// Returns top modules by download count
// Query params:
//   - period: Time period (7d, 30d, 90d) - default: 30d
//   - limit: Number of results (1-100) - default: 100
func (h *AnalyticsHandlers) getPopularModules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	period := httputil.ParseQueryString(r, "period", "30d")

	limit, err := httputil.ParseQueryInt(r, "limit", 100)
	if err != nil {
		httputil.WriteBadRequest(w, err.Error())
		return
	}
	if limit > 100 {
		limit = 100
	}

	modules, err := h.service.GetPopularModules(ctx, period, limit)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, modules)
}

// getTrendingModules handles GET /api/v2/analytics/modules/trending
// Returns modules with highest growth rate (7d vs previous 7d)
// Query params:
//   - limit: Number of results (1-50) - default: 50
func (h *AnalyticsHandlers) getTrendingModules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit, err := httputil.ParseQueryInt(r, "limit", 50)
	if err != nil {
		httputil.WriteBadRequest(w, err.Error())
		return
	}
	if limit > 50 {
		limit = 50
	}

	modules, err := h.service.GetTrendingModules(ctx, limit)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, modules)
}

// getModuleStats handles GET /api/v2/analytics/modules/{name}/stats
// Returns detailed analytics for a specific module
// Query params:
//   - period: Time period (7d, 30d, 90d) - default: 30d
func (h *AnalyticsHandlers) getModuleStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := httputil.GetPathVars(r)
	moduleName := vars["name"]

	period := httputil.ParseQueryString(r, "period", "30d")

	stats, err := h.service.GetModuleStats(ctx, moduleName, period)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, stats)
}

// getModuleHealth handles GET /api/v2/analytics/modules/{name}/health
// Returns schema health assessment for a module version
// Query params:
//   - version: Module version - default: latest version
func (h *AnalyticsHandlers) getModuleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := httputil.GetPathVars(r)
	moduleName := vars["name"]

	version := httputil.ParseQueryString(r, "version", "")

	health, err := h.service.GetModuleHealth(ctx, moduleName, version)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, health)
}
