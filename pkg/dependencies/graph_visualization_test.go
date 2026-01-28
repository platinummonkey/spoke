package dependencies

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestNewGraphVisualizationHandlers(t *testing.T) {
	storage := newMockStorage()
	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)

	if handlers == nil {
		t.Fatal("Expected handlers to be created")
	}

	if handlers.resolver != resolver {
		t.Error("Expected resolver to be set correctly")
	}
}

func TestGraphVisualizationHandlers_RegisterRoutes(t *testing.T) {
	storage := newMockStorage()
	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)
	router := mux.NewRouter()

	// Should not panic
	handlers.RegisterRoutes(router)

	// Verify routes are registered
	var routeCount int
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		routeCount++
		return nil
	})

	// Should have at least 2 routes registered
	if routeCount < 2 {
		t.Errorf("Expected at least 2 routes registered, got %d", routeCount)
	}
}

func TestGraphVisualizationHandlers_GetCytoscapeGraph(t *testing.T) {
	storage := newMockStorage()

	// Build test graph: user -> common -> base
	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/v2/modules/user/versions/v1.0.0/graph", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var cytoGraph CytoscapeGraph
	if err := json.NewDecoder(w.Body).Decode(&cytoGraph); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have 3 nodes: user, common, base
	if len(cytoGraph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(cytoGraph.Nodes))
	}

	// Should have 2 edges: user->common, common->base
	if len(cytoGraph.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(cytoGraph.Edges))
	}

	// Verify current node type
	foundCurrentNode := false
	for _, node := range cytoGraph.Nodes {
		if node.Data.Name == "user" && node.Data.Type == "current" {
			foundCurrentNode = true
			break
		}
	}
	if !foundCurrentNode {
		t.Error("Expected to find current node with type 'current'")
	}
}

func TestGraphVisualizationHandlers_GetCytoscapeGraph_QueryParams(t *testing.T) {
	storage := newMockStorage()

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/v2/dependencies/graph?module=user&version=v1.0.0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var cytoGraph CytoscapeGraph
	if err := json.NewDecoder(w.Body).Decode(&cytoGraph); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have 1 node: user (no dependencies)
	if len(cytoGraph.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(cytoGraph.Nodes))
	}
}

func TestGraphVisualizationHandlers_GetCytoscapeGraph_MissingParams(t *testing.T) {
	storage := newMockStorage()
	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "missing module",
			url:  "/api/v2/dependencies/graph?version=v1.0.0",
		},
		{
			name: "missing version",
			url:  "/api/v2/dependencies/graph?module=user",
		},
		{
			name: "missing both",
			url:  "/api/v2/dependencies/graph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestGraphVisualizationHandlers_GetCytoscapeGraph_TransitiveParam(t *testing.T) {
	storage := newMockStorage()

	// Build test graph: user -> common -> base
	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	// Test with transitive=false
	req := httptest.NewRequest("GET", "/api/v2/modules/user/versions/v1.0.0/graph?transitive=false", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var cytoGraph CytoscapeGraph
	if err := json.NewDecoder(w.Body).Decode(&cytoGraph); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have 2 nodes: user and common only (not base)
	if len(cytoGraph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes with transitive=false, got %d", len(cytoGraph.Nodes))
	}

	// Should have 1 edge: user->common only
	if len(cytoGraph.Edges) != 1 {
		t.Errorf("Expected 1 edge with transitive=false, got %d", len(cytoGraph.Edges))
	}
}

func TestGraphVisualizationHandlers_GetCytoscapeGraph_DepthParam(t *testing.T) {
	storage := newMockStorage()

	// Build test graph: user -> common -> base
	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	// Test with depth=1 (only direct dependencies)
	req := httptest.NewRequest("GET", "/api/v2/modules/user/versions/v1.0.0/graph?depth=1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var cytoGraph CytoscapeGraph
	if err := json.NewDecoder(w.Body).Decode(&cytoGraph); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have 2 nodes: user and common (depth=1 stops at common)
	if len(cytoGraph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes with depth=1, got %d", len(cytoGraph.Nodes))
	}
}

func TestGraphVisualizationHandlers_GetCytoscapeGraph_DirectionDependents(t *testing.T) {
	storage := newMockStorage()

	// Build graph where common is used by user
	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	// Request dependents of user (should be empty)
	req := httptest.NewRequest("GET", "/api/v2/modules/user/versions/v1.0.0/graph?direction=dependents", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var cytoGraph CytoscapeGraph
	if err := json.NewDecoder(w.Body).Decode(&cytoGraph); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have 1 node: user only (no dependents)
	if len(cytoGraph.Nodes) != 1 {
		t.Errorf("Expected 1 node with direction=dependents, got %d", len(cytoGraph.Nodes))
	}

	// Should have 0 edges (no dependents)
	if len(cytoGraph.Edges) != 0 {
		t.Errorf("Expected 0 edges with direction=dependents, got %d", len(cytoGraph.Edges))
	}
}

func TestGraphVisualizationHandlers_GetCytoscapeGraph_DirectionBoth(t *testing.T) {
	storage := newMockStorage()

	// Build graph
	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	// Request both dependencies and dependents
	req := httptest.NewRequest("GET", "/api/v2/modules/common/versions/v1.0.0/graph?direction=both", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var cytoGraph CytoscapeGraph
	if err := json.NewDecoder(w.Body).Decode(&cytoGraph); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have at least 2 nodes (common and base as dependency)
	if len(cytoGraph.Nodes) < 2 {
		t.Errorf("Expected at least 2 nodes with direction=both, got %d", len(cytoGraph.Nodes))
	}
}

func TestGraphVisualizationHandlers_GetCytoscapeGraph_BuildError(t *testing.T) {
	storage := newMockStorage()

	// Don't add the version to storage to trigger an error
	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/v2/modules/nonexistent/versions/v1.0.0/graph", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestBuildCytoscapeGraph_DirectDependencies(t *testing.T) {
	storage := newMockStorage()

	// Build simple graph: user -> common
	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)

	graph, _ := resolver.BuildDependencyGraph("user", "v1.0.0")
	cytoGraph := handlers.buildCytoscapeGraph(graph, "user", "v1.0.0", false, -1, "dependencies")

	// Should have 2 nodes
	if len(cytoGraph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(cytoGraph.Nodes))
	}

	// Should have 1 edge
	if len(cytoGraph.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(cytoGraph.Edges))
	}

	// Check edge type is "direct"
	if cytoGraph.Edges[0].Data.Type != "direct" {
		t.Errorf("Expected edge type 'direct', got '%s'", cytoGraph.Edges[0].Data.Type)
	}
}

func TestBuildCytoscapeGraph_TransitiveDependencies(t *testing.T) {
	storage := newMockStorage()

	// Build graph: user -> common -> base
	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)

	graph, _ := resolver.BuildDependencyGraph("user", "v1.0.0")
	cytoGraph := handlers.buildCytoscapeGraph(graph, "user", "v1.0.0", true, -1, "dependencies")

	// Should have 3 nodes
	if len(cytoGraph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(cytoGraph.Nodes))
	}

	// Should have 2 edges
	if len(cytoGraph.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(cytoGraph.Edges))
	}

	// Check that we have direct edges
	hasDirectEdge := false
	for _, edge := range cytoGraph.Edges {
		if edge.Data.Type == "direct" {
			hasDirectEdge = true
			break
		}
	}

	if !hasDirectEdge {
		t.Error("Expected to find at least one direct edge")
	}
}

func TestBuildCytoscapeGraph_WithDependents(t *testing.T) {
	storage := newMockStorage()

	// Build graph where common is used by user and order
	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	storage.addVersion("order", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message Order { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)

	// Build graph from user to include common
	graph, _ := resolver.BuildDependencyGraph("user", "v1.0.0")

	// Manually add order to the graph (simulating a full graph)
	orderDeps := []Dependency{{Module: "common", Version: "v1.0.0", Type: "direct"}}
	graph.AddNode("order", "v1.0.0", orderDeps)

	cytoGraph := handlers.buildCytoscapeGraph(graph, "common", "v1.0.0", false, -1, "dependents")

	// Should have nodes for common and its dependents
	if len(cytoGraph.Nodes) < 2 {
		t.Errorf("Expected at least 2 nodes, got %d", len(cytoGraph.Nodes))
	}

	// Check that dependent nodes have correct type
	foundDependent := false
	for _, node := range cytoGraph.Nodes {
		if node.Data.Type == "dependent" {
			foundDependent = true
			break
		}
	}
	if !foundDependent {
		t.Error("Expected to find node with type 'dependent'")
	}

	// Check edge direction (dependent -> current)
	if len(cytoGraph.Edges) > 0 {
		if cytoGraph.Edges[0].Data.Type != "depends-on" {
			t.Errorf("Expected edge type 'depends-on', got '%s'", cytoGraph.Edges[0].Data.Type)
		}
	}
}

func TestBuildCytoscapeGraph_MaxDepth(t *testing.T) {
	storage := newMockStorage()

	// Build deep graph: user -> common -> base -> foundation
	storage.addVersion("foundation", "v1.0.0", `syntax = "proto3";
message Foundation { string id = 1; }`)

	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
import "foundation@v1.0.0";
message Base { string id = 1; }`)

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)

	graph, _ := resolver.BuildDependencyGraph("user", "v1.0.0")

	// Test with depth=2 (should stop at base)
	cytoGraph := handlers.buildCytoscapeGraph(graph, "user", "v1.0.0", true, 2, "dependencies")

	// Should have 3 nodes: user, common, base (not foundation)
	if len(cytoGraph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes with depth=2, got %d", len(cytoGraph.Nodes))
	}

	// Verify foundation is not included
	for _, node := range cytoGraph.Nodes {
		if node.Data.Name == "foundation" {
			t.Error("Expected foundation not to be included with depth=2")
		}
	}
}

func TestAddDirectDependencies(t *testing.T) {
	storage := newMockStorage()

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)

	graph, _ := resolver.BuildDependencyGraph("user", "v1.0.0")

	cytoGraph := CytoscapeGraph{
		Nodes: make([]CytoscapeNode, 0),
		Edges: make([]CytoscapeEdge, 0),
	}

	visited := make(map[string]bool)
	currentKey := moduleVersionKey("user", "v1.0.0")

	// Add current node
	cytoGraph.Nodes = append(cytoGraph.Nodes, CytoscapeNode{
		Data: CytoscapeNodeData{
			ID:      currentKey,
			Name:    "user",
			Version: "v1.0.0",
			Type:    "current",
		},
	})
	visited[currentKey] = true

	handlers.addDirectDependencies(&cytoGraph, graph, "user", "v1.0.0", visited)

	// Should add 1 node and 1 edge
	if len(cytoGraph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes after adding direct dependencies, got %d", len(cytoGraph.Nodes))
	}

	if len(cytoGraph.Edges) != 1 {
		t.Errorf("Expected 1 edge after adding direct dependencies, got %d", len(cytoGraph.Edges))
	}

	// Verify the dependency node type
	if cytoGraph.Nodes[1].Data.Type != "dependency" {
		t.Errorf("Expected dependency node type to be 'dependency', got '%s'", cytoGraph.Nodes[1].Data.Type)
	}
}

func TestAddTransitiveDependencies_WithDepthLimit(t *testing.T) {
	storage := newMockStorage()

	// Build graph: user -> common -> base
	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)

	graph, _ := resolver.BuildDependencyGraph("user", "v1.0.0")

	cytoGraph := CytoscapeGraph{
		Nodes: make([]CytoscapeNode, 0),
		Edges: make([]CytoscapeEdge, 0),
	}

	visited := make(map[string]bool)
	currentKey := moduleVersionKey("user", "v1.0.0")

	// Add current node
	cytoGraph.Nodes = append(cytoGraph.Nodes, CytoscapeNode{
		Data: CytoscapeNodeData{
			ID:      currentKey,
			Name:    "user",
			Version: "v1.0.0",
			Type:    "current",
		},
	})
	visited[currentKey] = true

	// Add transitive dependencies with depth=1
	handlers.addTransitiveDependencies(&cytoGraph, graph, "user", "v1.0.0", visited, 1, 0)

	// Should only add common, not base (depth=1)
	if len(cytoGraph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes with depth=1, got %d", len(cytoGraph.Nodes))
	}
}

func TestAddDependents(t *testing.T) {
	storage := newMockStorage()

	// Build graph where common is used by user
	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	handlers := NewGraphVisualizationHandlers(resolver)

	graph, _ := resolver.BuildDependencyGraph("user", "v1.0.0")

	cytoGraph := CytoscapeGraph{
		Nodes: make([]CytoscapeNode, 0),
		Edges: make([]CytoscapeEdge, 0),
	}

	visited := make(map[string]bool)
	currentKey := moduleVersionKey("common", "v1.0.0")

	// Add current node
	cytoGraph.Nodes = append(cytoGraph.Nodes, CytoscapeNode{
		Data: CytoscapeNodeData{
			ID:      currentKey,
			Name:    "common",
			Version: "v1.0.0",
			Type:    "current",
		},
	})
	visited[currentKey] = true

	handlers.addDependents(&cytoGraph, graph, "common", "v1.0.0", visited)

	// Should add user as a dependent
	if len(cytoGraph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes after adding dependents, got %d", len(cytoGraph.Nodes))
	}

	if len(cytoGraph.Edges) != 1 {
		t.Errorf("Expected 1 edge after adding dependents, got %d", len(cytoGraph.Edges))
	}

	// Verify the dependent node type
	if cytoGraph.Nodes[1].Data.Type != "dependent" {
		t.Errorf("Expected dependent node type to be 'dependent', got '%s'", cytoGraph.Nodes[1].Data.Type)
	}

	// Verify edge direction (dependent -> current)
	if cytoGraph.Edges[0].Data.Source != "user@v1.0.0" {
		t.Errorf("Expected edge source to be 'user@v1.0.0', got '%s'", cytoGraph.Edges[0].Data.Source)
	}

	if cytoGraph.Edges[0].Data.Target != "common@v1.0.0" {
		t.Errorf("Expected edge target to be 'common@v1.0.0', got '%s'", cytoGraph.Edges[0].Data.Target)
	}
}
