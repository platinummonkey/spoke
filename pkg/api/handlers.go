package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Server represents our API server
type Server struct {
	storage Storage
	router  *mux.Router
}

// NewServer creates a new API server
func NewServer(storage Storage) *Server {
	s := &Server{
		storage: storage,
		router:  mux.NewRouter(),
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
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// createModule handles POST /modules
func (s *Server) createModule(w http.ResponseWriter, r *http.Request) {
	var module Module
	if err := json.NewDecoder(r.Body).Decode(&module); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	module.CreatedAt = time.Now()
	module.UpdatedAt = time.Now()

	if err := s.storage.CreateModule(&module); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(module)
}

// listModules handles GET /modules
func (s *Server) listModules(w http.ResponseWriter, r *http.Request) {
	modules, err := s.storage.ListModules()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		modulesWithVersions[i] = struct {
			*Module
			Versions []*Version `json:"versions"`
		}{
			Module:   module,
			Versions: versions,
		}
	}

	json.NewEncoder(w).Encode(modulesWithVersions)
}

// getModule handles GET /modules/{name}
func (s *Server) getModule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	module, err := s.storage.GetModule(vars["name"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get versions for this module
	versions, err := s.storage.ListVersions(vars["name"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add versions to the module response
	moduleWithVersions := struct {
		*Module
		Versions []*Version `json:"versions"`
	}{
		Module:   module,
		Versions: versions,
	}

	json.NewEncoder(w).Encode(moduleWithVersions)
}

// createVersion handles POST /modules/{name}/versions
func (s *Server) createVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var version Version
	if err := json.NewDecoder(r.Body).Decode(&version); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	version.ModuleName = vars["name"]
	version.CreatedAt = time.Now()

	if err := s.storage.CreateVersion(&version); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(version)
}

// listVersions handles GET /modules/{name}/versions
func (s *Server) listVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	versions, err := s.storage.ListVersions(vars["name"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(versions)
}

// getVersion handles GET /modules/{name}/versions/{version}
func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	version, err := s.storage.GetVersion(vars["name"], vars["version"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(version)
}

// getFile handles GET /modules/{name}/versions/{version}/files/{path}
func (s *Server) getFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	file, err := s.storage.GetFile(vars["name"], vars["version"], vars["path"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(file)
} 