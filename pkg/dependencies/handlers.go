package dependencies

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/api"
)

// DependencyHandlers provides HTTP handlers for dependencies
type DependencyHandlers struct {
	resolver *DependencyResolver
}

// NewDependencyHandlers creates new dependency handlers
func NewDependencyHandlers(storage api.Storage) *DependencyHandlers {
	return &DependencyHandlers{
		resolver: NewDependencyResolver(storage),
	}
}

// RegisterRoutes registers dependency routes
func (h *DependencyHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/modules/{name}/versions/{version}/dependencies", h.getDependencies).Methods("GET")
	router.HandleFunc("/modules/{name}/versions/{version}/dependencies/transitive", h.getTransitiveDependencies).Methods("GET")
	router.HandleFunc("/modules/{name}/versions/{version}/dependents", h.getDependents).Methods("GET")
	router.HandleFunc("/modules/{name}/versions/{version}/impact", h.getImpact).Methods("GET")
	router.HandleFunc("/modules/{name}/versions/{version}/lockfile", h.getLockfile).Methods("GET")
	router.HandleFunc("/modules/{name}/versions/{version}/lockfile/validate", h.validateLockfile).Methods("POST")
	router.HandleFunc("/modules/{name}/versions/{version}/graph", h.getDependencyGraph).Methods("GET")

	// Register v2 graph visualization endpoints (Cytoscape.js format)
	vizHandlers := NewGraphVisualizationHandlers(h.resolver)
	vizHandlers.RegisterRoutes(router)
}

// getDependencies handles GET /modules/{name}/versions/{version}/dependencies
func (h *DependencyHandlers) getDependencies(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]

	deps, err := h.resolver.ResolveDependencies(moduleName, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"module":       moduleName,
		"version":      version,
		"dependencies": deps,
		"count":        len(deps),
	})
}

// getTransitiveDependencies handles GET /modules/{name}/versions/{version}/dependencies/transitive
func (h *DependencyHandlers) getTransitiveDependencies(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]

	graph, err := h.resolver.BuildDependencyGraph(moduleName, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	deps := graph.GetTransitiveDependencies(moduleName, version)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"module":       moduleName,
		"version":      version,
		"dependencies": deps,
		"count":        len(deps),
	})
}

// getDependents handles GET /modules/{name}/versions/{version}/dependents
func (h *DependencyHandlers) getDependents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]

	graph, err := h.resolver.BuildDependencyGraph(moduleName, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dependents := graph.GetDependents(moduleName, version)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"module":     moduleName,
		"version":    version,
		"dependents": dependents,
		"count":      len(dependents),
	})
}

// getImpact handles GET /modules/{name}/versions/{version}/impact
func (h *DependencyHandlers) getImpact(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]

	graph, err := h.resolver.BuildDependencyGraph(moduleName, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	impact := graph.GetImpactAnalysis(moduleName, version)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(impact)
}

// getLockfile handles GET /modules/{name}/versions/{version}/lockfile
func (h *DependencyHandlers) getLockfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]

	lockfile, err := h.resolver.GenerateLockfile(moduleName, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lockfile)
}

// validateLockfile handles POST /modules/{name}/versions/{version}/lockfile/validate
func (h *DependencyHandlers) validateLockfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]

	var lockfile Lockfile
	if err := json.NewDecoder(r.Body).Decode(&lockfile); err != nil {
		http.Error(w, "invalid lockfile format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Override module/version from URL
	lockfile.Module = moduleName
	lockfile.Version = version

	valid, differences, err := h.resolver.ValidateLockfile(&lockfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":       valid,
		"differences": differences,
	})
}

// getDependencyGraph handles GET /modules/{name}/versions/{version}/graph
func (h *DependencyHandlers) getDependencyGraph(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]

	graph, err := h.resolver.BuildDependencyGraph(moduleName, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check for circular dependencies
	cycles, cycleErr := graph.DetectCircularDependencies(moduleName, version)

	// Convert graph to JSON-friendly format
	nodes := make([]map[string]interface{}, 0)
	edges := make([]map[string]interface{}, 0)

	for key, node := range graph.nodes {
		nodes = append(nodes, map[string]interface{}{
			"id":      key,
			"module":  node.Module,
			"version": node.Version,
		})

		for _, dep := range node.Dependencies {
			edges = append(edges, map[string]interface{}{
				"from": key,
				"to":   moduleVersionKey(dep.Module, dep.Version),
				"type": dep.Type,
			})
		}
	}

	response := map[string]interface{}{
		"module":  moduleName,
		"version": version,
		"nodes":   nodes,
		"edges":   edges,
	}

	if cycleErr != nil {
		response["has_circular_dependency"] = true
		response["circular_path"] = cycles
	} else {
		response["has_circular_dependency"] = false
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
