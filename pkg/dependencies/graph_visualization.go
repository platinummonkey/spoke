package dependencies

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// CytoscapeNode represents a node in Cytoscape.js format
type CytoscapeNode struct {
	Data CytoscapeNodeData `json:"data"`
}

// CytoscapeNodeData contains node data for Cytoscape.js
type CytoscapeNodeData struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // "current", "dependency", "dependent"
}

// CytoscapeEdge represents an edge in Cytoscape.js format
type CytoscapeEdge struct {
	Data CytoscapeEdgeData `json:"data"`
}

// CytoscapeEdgeData contains edge data for Cytoscape.js
type CytoscapeEdgeData struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type,omitempty"` // "direct", "transitive"
}

// CytoscapeGraph represents the complete graph in Cytoscape.js format
type CytoscapeGraph struct {
	Nodes []CytoscapeNode `json:"nodes"`
	Edges []CytoscapeEdge `json:"edges"`
}

// GraphVisualizationHandlers provides HTTP handlers for graph visualization
type GraphVisualizationHandlers struct {
	resolver *DependencyResolver
}

// NewGraphVisualizationHandlers creates new graph visualization handlers
func NewGraphVisualizationHandlers(resolver *DependencyResolver) *GraphVisualizationHandlers {
	return &GraphVisualizationHandlers{
		resolver: resolver,
	}
}

// RegisterRoutes registers graph visualization routes
func (h *GraphVisualizationHandlers) RegisterRoutes(router *mux.Router) {
	// Cytoscape.js format endpoint
	router.HandleFunc("/api/v2/modules/{name}/versions/{version}/graph", h.getCytoscapeGraph).Methods("GET")
	router.HandleFunc("/api/v2/dependencies/graph", h.getCytoscapeGraph).Methods("GET")
}

// getCytoscapeGraph handles GET /api/v2/modules/{name}/versions/{version}/graph
// Query parameters:
//   - transitive: include transitive dependencies (default: true)
//   - depth: max depth for transitive dependencies (default: unlimited)
//   - direction: "dependencies", "dependents", or "both" (default: "dependencies")
func (h *GraphVisualizationHandlers) getCytoscapeGraph(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]

	if moduleName == "" {
		moduleName = r.URL.Query().Get("module")
	}
	if version == "" {
		version = r.URL.Query().Get("version")
	}

	if moduleName == "" || version == "" {
		http.Error(w, "module and version are required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	transitive := true
	if t := r.URL.Query().Get("transitive"); t != "" {
		transitive = t == "true" || t == "1"
	}

	maxDepth := -1 // unlimited
	if d := r.URL.Query().Get("depth"); d != "" {
		if depth, err := strconv.Atoi(d); err == nil && depth > 0 {
			maxDepth = depth
		}
	}

	direction := r.URL.Query().Get("direction")
	if direction == "" {
		direction = "dependencies"
	}

	// Build dependency graph
	graph, err := h.resolver.BuildDependencyGraph(moduleName, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to Cytoscape format
	cytoGraph := h.buildCytoscapeGraph(graph, moduleName, version, transitive, maxDepth, direction)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cytoGraph)
}

// buildCytoscapeGraph builds a Cytoscape.js compatible graph
func (h *GraphVisualizationHandlers) buildCytoscapeGraph(
	graph *DependencyGraph,
	moduleName, version string,
	transitive bool,
	maxDepth int,
	direction string,
) CytoscapeGraph {
	cytoGraph := CytoscapeGraph{
		Nodes: make([]CytoscapeNode, 0),
		Edges: make([]CytoscapeEdge, 0),
	}

	visited := make(map[string]bool)
	currentKey := moduleVersionKey(moduleName, version)

	// Add current module as root node
	cytoGraph.Nodes = append(cytoGraph.Nodes, CytoscapeNode{
		Data: CytoscapeNodeData{
			ID:      currentKey,
			Name:    moduleName,
			Version: version,
			Type:    "current",
		},
	})
	visited[currentKey] = true

	// Add dependencies if requested
	if direction == "dependencies" || direction == "both" {
		if transitive {
			h.addTransitiveDependencies(&cytoGraph, graph, moduleName, version, visited, maxDepth, 0)
		} else {
			h.addDirectDependencies(&cytoGraph, graph, moduleName, version, visited)
		}
	}

	// Add dependents if requested
	if direction == "dependents" || direction == "both" {
		h.addDependents(&cytoGraph, graph, moduleName, version, visited)
	}

	return cytoGraph
}

// addDirectDependencies adds direct dependencies to the graph
func (h *GraphVisualizationHandlers) addDirectDependencies(
	cytoGraph *CytoscapeGraph,
	graph *DependencyGraph,
	moduleName, version string,
	visited map[string]bool,
) {
	deps := graph.GetDependencies(moduleName, version)
	currentKey := moduleVersionKey(moduleName, version)

	for _, dep := range deps {
		depKey := moduleVersionKey(dep.Module, dep.Version)

		// Add node if not visited
		if !visited[depKey] {
			cytoGraph.Nodes = append(cytoGraph.Nodes, CytoscapeNode{
				Data: CytoscapeNodeData{
					ID:      depKey,
					Name:    dep.Module,
					Version: dep.Version,
					Type:    "dependency",
				},
			})
			visited[depKey] = true
		}

		// Add edge
		edgeID := currentKey + "->" + depKey
		cytoGraph.Edges = append(cytoGraph.Edges, CytoscapeEdge{
			Data: CytoscapeEdgeData{
				ID:     edgeID,
				Source: currentKey,
				Target: depKey,
				Type:   "direct",
			},
		})
	}
}

// addTransitiveDependencies adds transitive dependencies recursively
func (h *GraphVisualizationHandlers) addTransitiveDependencies(
	cytoGraph *CytoscapeGraph,
	graph *DependencyGraph,
	moduleName, version string,
	visited map[string]bool,
	maxDepth, currentDepth int,
) {
	// Check depth limit
	if maxDepth >= 0 && currentDepth >= maxDepth {
		return
	}

	deps := graph.GetDependencies(moduleName, version)
	currentKey := moduleVersionKey(moduleName, version)

	for _, dep := range deps {
		depKey := moduleVersionKey(dep.Module, dep.Version)

		// Add node if not visited
		if !visited[depKey] {
			cytoGraph.Nodes = append(cytoGraph.Nodes, CytoscapeNode{
				Data: CytoscapeNodeData{
					ID:      depKey,
					Name:    dep.Module,
					Version: dep.Version,
					Type:    "dependency",
				},
			})
			visited[depKey] = true

			// Recursively add transitive dependencies
			h.addTransitiveDependencies(cytoGraph, graph, dep.Module, dep.Version, visited, maxDepth, currentDepth+1)
		}

		// Add edge
		edgeID := currentKey + "->" + depKey
		edgeType := "direct"
		if currentDepth > 0 {
			edgeType = "transitive"
		}

		cytoGraph.Edges = append(cytoGraph.Edges, CytoscapeEdge{
			Data: CytoscapeEdgeData{
				ID:     edgeID,
				Source: currentKey,
				Target: depKey,
				Type:   edgeType,
			},
		})
	}
}

// addDependents adds modules that depend on the current module
func (h *GraphVisualizationHandlers) addDependents(
	cytoGraph *CytoscapeGraph,
	graph *DependencyGraph,
	moduleName, version string,
	visited map[string]bool,
) {
	dependents := graph.GetDependents(moduleName, version)
	currentKey := moduleVersionKey(moduleName, version)

	for _, dependent := range dependents {
		depKey := moduleVersionKey(dependent.Module, dependent.Version)

		// Add node if not visited
		if !visited[depKey] {
			cytoGraph.Nodes = append(cytoGraph.Nodes, CytoscapeNode{
				Data: CytoscapeNodeData{
					ID:      depKey,
					Name:    dependent.Module,
					Version: dependent.Version,
					Type:    "dependent",
				},
			})
			visited[depKey] = true
		}

		// Add edge (reversed: dependent -> current)
		edgeID := depKey + "->" + currentKey
		cytoGraph.Edges = append(cytoGraph.Edges, CytoscapeEdge{
			Data: CytoscapeEdgeData{
				ID:     edgeID,
				Source: depKey,
				Target: currentKey,
				Type:   "depends-on",
			},
		})
	}
}
