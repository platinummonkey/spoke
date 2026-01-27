package api

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/analytics"
	"github.com/platinummonkey/spoke/pkg/async"
	"github.com/platinummonkey/spoke/pkg/codegen/orchestrator"
	"github.com/platinummonkey/spoke/pkg/httputil"
	"github.com/platinummonkey/spoke/pkg/search"
)

// Server represents our API server
type Server struct {
	storage             Storage
	router              *mux.Router
	db                  *sql.DB
	authHandlers        *AuthHandlers
	compatHandlers      *CompatibilityHandlers
	validationHandlers  *ValidationHandlers
	orchestrator        orchestrator.Orchestrator // Code generation orchestrator (v2)
	searchIndexer       *search.Indexer            // Search indexer for proto entities
	eventTracker        *analytics.EventTracker    // Analytics event tracker
}

// NewServer creates a new API server
func NewServer(storage Storage, db *sql.DB) *Server {
	s := &Server{
		storage: storage,
		router:  mux.NewRouter(),
		db:      db,
	}

	// Initialize handlers if database is provided
	if db != nil {
		s.authHandlers = NewAuthHandlers(db)
		s.compatHandlers = NewCompatibilityHandlers(storage)
		s.validationHandlers = NewValidationHandlers(storage)

		// Initialize search indexer
		storageAdapter := NewSearchStorageAdapter(storage)
		s.searchIndexer = search.NewIndexer(db, storageAdapter)

		// Initialize analytics event tracker
		s.eventTracker = analytics.NewEventTracker(db)
	}

	// Initialize code generation orchestrator (v2)
	// Note: Errors are non-fatal - falls back to v1 compilation if orchestrator fails
	if orch, err := orchestrator.NewOrchestrator(nil); err == nil {
		s.orchestrator = orch
		// Register package generators
		s.registerPackageGenerators()
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all the API routes
func (s *Server) setupRoutes() {
	// Module routes
	s.router.HandleFunc("/modules", s.createModule).Methods("POST")
	s.router.HandleFunc("/modules", s.listModules).Methods("GET")
	s.router.HandleFunc("/modules/{name}", s.getModule).Methods("GET")

	// Version routes
	s.router.HandleFunc("/modules/{name}/versions", s.createVersion).Methods("POST")
	s.router.HandleFunc("/modules/{name}/versions", s.listVersions).Methods("GET")
	s.router.HandleFunc("/modules/{name}/versions/{version}", s.getVersion).Methods("GET")

	// File routes
	s.router.HandleFunc("/modules/{name}/versions/{version}/files/{path:.*}", s.getFile).Methods("GET")

	// Download compilation results
	s.router.HandleFunc("/modules/{name}/versions/{version}/download/{language}", s.downloadCompiled).Methods("GET")

	// Language routes (v2 API)
	s.router.HandleFunc("/api/v1/languages", s.listLanguages).Methods("GET")
	s.router.HandleFunc("/api/v1/languages/{id}", s.getLanguage).Methods("GET")

	// Compilation routes (v2 API)
	s.router.HandleFunc("/api/v1/modules/{name}/versions/{version}/compile", s.compileVersion).Methods("POST")
	s.router.HandleFunc("/api/v1/modules/{name}/versions/{version}/compile/{jobId}", s.getCompilationJob).Methods("GET")

	// Example generation routes
	s.router.HandleFunc("/api/v1/modules/{name}/versions/{version}/examples/{language}", s.getExamples).Methods("GET")

	// Diff routes
	s.router.HandleFunc("/api/v1/modules/{name}/diff", s.compareDiff).Methods("POST")

	// Register authentication routes (if database is available)
	if s.authHandlers != nil {
		s.authHandlers.RegisterRoutes(s.router)
	}

	// Register compatibility routes
	if s.compatHandlers != nil {
		s.compatHandlers.RegisterRoutes(s.router)
	}

	// Register validation routes
	if s.validationHandlers != nil {
		s.validationHandlers.RegisterRoutes(s.router)
	}

	// Register enhanced search routes (v2 API)
	if s.db != nil {
		enhancedSearchHandlers := NewEnhancedSearchHandlers(s.db)
		enhancedSearchHandlers.RegisterRoutes(s.router)

		// Register user features routes (saved searches, bookmarks)
		userFeaturesHandlers := NewUserFeaturesHandlers(s.db)
		userFeaturesHandlers.RegisterRoutes(s.router)

		// Register analytics routes (v2 API)
		analyticsService := analytics.NewService(s.db)
		analyticsHandlers := NewAnalyticsHandlers(analyticsService)
		analyticsHandlers.RegisterRoutes(s.router)
	}
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// RouteRegistrar is an interface for types that can register routes
type RouteRegistrar interface {
	RegisterRoutes(router *mux.Router)
}

// RegisterRoutes registers routes from a RouteRegistrar
func (s *Server) RegisterRoutes(registrar RouteRegistrar) {
	registrar.RegisterRoutes(s.router)
}

// createModule handles POST /modules
func (s *Server) createModule(w http.ResponseWriter, r *http.Request) {
	var module Module
	if !httputil.ParseJSONOrError(w, r, &module) {
		return
	}

	module.CreatedAt = time.Now()
	module.UpdatedAt = time.Now()

	if err := s.storage.CreateModule(&module); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w, module)
}

// listModules handles GET /modules
func (s *Server) listModules(w http.ResponseWriter, r *http.Request) {
	modules, err := s.storage.ListModules()
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	// Get versions for each module
	modulesWithVersions := make([]struct {
		*Module
		Versions []*Version `json:"versions"`
	}, len(modules))

	for i, module := range modules {
		versions, err := s.storage.ListVersions(module.Name)
		if err != nil {
			httputil.WriteInternalError(w, err)
			return
		}

		// Sort versions by newest first
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].CreatedAt.After(versions[j].CreatedAt)
		})

		modulesWithVersions[i] = struct {
			*Module
			Versions []*Version `json:"versions"`
		}{
			Module:   module,
			Versions: versions,
		}
	}

	httputil.WriteSuccess(w, modulesWithVersions)
}

// getModule handles GET /modules/{name}
func (s *Server) getModule(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	module, err := s.storage.GetModule(vars["name"])
	if err != nil {
		httputil.WriteNotFoundError(w, err.Error())
		return
	}

	// Get versions for this module
	versions, err := s.storage.ListVersions(vars["name"])
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	// Sort versions by newest first
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})

	// Add versions to the module response
	moduleWithVersions := struct {
		*Module
		Versions []*Version `json:"versions"`
	}{
		Module:   module,
		Versions: versions,
	}

	// Track module view event asynchronously
	if s.eventTracker != nil {
		async.SafeGo(r.Context(), 5*time.Second, "track module view", func(ctx context.Context) error {
			source := "api"
			if r.Header.Get("User-Agent") != "" && strings.Contains(r.Header.Get("User-Agent"), "Mozilla") {
				source = "web"
			}

			event := analytics.ModuleViewEvent{
				UserID:         analytics.ExtractUserID(r),
				OrganizationID: analytics.ExtractOrganizationID(r),
				ModuleName:     vars["name"],
				Version:        "", // Viewing module list
				Source:         source,
				PageType:       "detail",
				Referrer:       analytics.GetReferrer(r),
				IPAddress:      analytics.GetClientIP(r),
				UserAgent:      analytics.GetUserAgent(r),
			}

			return s.eventTracker.TrackModuleView(ctx, event)
		})
	}

	httputil.WriteSuccess(w, moduleWithVersions)
}

// createVersion handles POST /modules/{name}/versions
func (s *Server) createVersion(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	var version Version
	if !httputil.ParseJSONOrError(w, r, &version) {
		return
	}

	version.ModuleName = vars["name"]
	version.CreatedAt = time.Now()

	if err := s.storage.CreateVersion(&version); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	// Trigger search indexing asynchronously (don't block the response)
	if s.searchIndexer != nil {
		async.SafeGo(r.Context(), 10*time.Second, "index version", func(ctx context.Context) error {
			return s.searchIndexer.IndexVersion(ctx, version.ModuleName, version.Version)
		})
	}

	httputil.WriteCreated(w, version)
}

// listVersions handles GET /modules/{name}/versions
func (s *Server) listVersions(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	versions, err := s.storage.ListVersions(vars["name"])
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, versions)
}

// getVersion handles GET /modules/{name}/versions/{version}
func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	version, err := s.storage.GetVersion(vars["name"], vars["version"])
	if err != nil {
		httputil.WriteNotFoundError(w, err.Error())
		return
	}

	// Track module version view event asynchronously
	if s.eventTracker != nil {
		async.SafeGo(r.Context(), 5*time.Second, "track version view", func(ctx context.Context) error {
			source := "api"
			if r.Header.Get("User-Agent") != "" && strings.Contains(r.Header.Get("User-Agent"), "Mozilla") {
				source = "web"
			}

			event := analytics.ModuleViewEvent{
				UserID:         analytics.ExtractUserID(r),
				OrganizationID: analytics.ExtractOrganizationID(r),
				ModuleName:     vars["name"],
				Version:        vars["version"],
				Source:         source,
				PageType:       "detail",
				Referrer:       analytics.GetReferrer(r),
				IPAddress:      analytics.GetClientIP(r),
				UserAgent:      analytics.GetUserAgent(r),
			}

			return s.eventTracker.TrackModuleView(ctx, event)
		})
	}

	httputil.WriteSuccess(w, version)
}

// getFile handles GET /modules/{name}/versions/{version}/files/{path}
func (s *Server) getFile(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	file, err := s.storage.GetFile(vars["name"], vars["version"], vars["path"])
	if err != nil {
		httputil.WriteNotFoundError(w, err.Error())
		return
	}

	httputil.WriteSuccess(w, file)
}
