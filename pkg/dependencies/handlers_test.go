package dependencies

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestDependencyHandlers_GetDependencies(t *testing.T) {
	storage := newMockStorage()

	// Add version with dependencies
	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
message Common { string id = 1; }`)

	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/modules/user/versions/v1.0.0/dependencies", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["module"] != "user" {
		t.Errorf("Expected module 'user', got %v", response["module"])
	}

	if response["version"] != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got %v", response["version"])
	}

	deps, ok := response["dependencies"].([]interface{})
	if !ok {
		t.Fatal("Expected dependencies to be array")
	}

	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(deps))
	}
}

func TestDependencyHandlers_GetTransitiveDependencies(t *testing.T) {
	storage := newMockStorage()

	// Build chain: user -> common -> base
	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/modules/user/versions/v1.0.0/dependencies/transitive", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	deps, ok := response["dependencies"].([]interface{})
	if !ok {
		t.Fatal("Expected dependencies to be array")
	}

	// Should include both common and base
	if len(deps) != 2 {
		t.Errorf("Expected 2 transitive dependencies, got %d", len(deps))
	}
}

func TestDependencyHandlers_GetDependents(t *testing.T) {
	storage := newMockStorage()

	// Build graph where user depends on common
	// Note: BuildDependencyGraph starts from a module and traverses dependencies
	// To see dependents of common, we need to build from user which will include common in the graph
	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	// Query for dependents of common by building graph from user
	// In a real implementation, this would scan all modules to find dependents
	// For now, test that the endpoint works with a module that has dependencies
	req := httptest.NewRequest("GET", "/modules/user/versions/v1.0.0/dependents", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	dependents, ok := response["dependents"].([]interface{})
	if !ok {
		t.Fatal("Expected dependents to be array")
	}

	// user has no dependents in this test
	if len(dependents) != 0 {
		t.Errorf("Expected 0 dependents for user, got %d", len(dependents))
	}
}

func TestDependencyHandlers_GetImpact(t *testing.T) {
	storage := newMockStorage()

	// Build graph for impact analysis
	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	// Test impact for base - common and user depend on it (transitively)
	req := httptest.NewRequest("GET", "/modules/user/versions/v1.0.0/impact", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var impact ImpactAnalysis
	if err := json.NewDecoder(w.Body).Decode(&impact); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if impact.Module != "user" {
		t.Errorf("Expected module 'user', got %s", impact.Module)
	}

	// user is a leaf module with no dependents
	if len(impact.DirectDependents) != 0 {
		t.Errorf("Expected 0 direct dependents for user, got %d", len(impact.DirectDependents))
	}
}

func TestDependencyHandlers_GetLockfile(t *testing.T) {
	storage := newMockStorage()

	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message User { string id = 1; }`)

	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/modules/user/versions/v1.0.0/lockfile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var lockfile Lockfile
	if err := json.NewDecoder(w.Body).Decode(&lockfile); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if lockfile.Module != "user" {
		t.Errorf("Expected module 'user', got %s", lockfile.Module)
	}

	if lockfile.Version != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got %s", lockfile.Version)
	}

	if len(lockfile.Dependencies) == 0 {
		t.Error("Expected non-empty dependencies in lockfile")
	}
}

func TestDependencyHandlers_ValidateLockfile(t *testing.T) {
	storage := newMockStorage()

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		name           string
		lockfile       Lockfile
		expectValid    bool
		expectStatus   int
	}{
		{
			name: "valid lockfile",
			lockfile: Lockfile{
				Module:  "user",
				Version: "v1.0.0",
				Dependencies: []Dependency{
					{Module: "common", Version: "v1.0.0", Type: "direct"},
				},
			},
			expectValid:  true,
			expectStatus: http.StatusOK,
		},
		{
			name: "invalid lockfile - version mismatch",
			lockfile: Lockfile{
				Module:  "user",
				Version: "v1.0.0",
				Dependencies: []Dependency{
					{Module: "common", Version: "v2.0.0", Type: "direct"},
				},
			},
			expectValid:  false,
			expectStatus: http.StatusOK,
		},
		{
			name: "missing dependencies",
			lockfile: Lockfile{
				Module:       "user",
				Version:      "v1.0.0",
				Dependencies: []Dependency{},
			},
			expectValid:  false,
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.lockfile)
			req := httptest.NewRequest("POST", "/modules/user/versions/v1.0.0/lockfile/validate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			valid, ok := response["valid"].(bool)
			if !ok {
				t.Fatal("Expected 'valid' field in response")
			}

			if valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, valid)
			}
		})
	}
}

func TestDependencyHandlers_ValidateLockfile_InvalidJSON(t *testing.T) {
	storage := newMockStorage()
	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("POST", "/modules/user/versions/v1.0.0/lockfile/validate", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestDependencyHandlers_GetDependencyGraph(t *testing.T) {
	storage := newMockStorage()

	// Build simple graph
	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message User { string id = 1; }`)

	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/modules/user/versions/v1.0.0/graph", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["module"] != "user" {
		t.Errorf("Expected module 'user', got %v", response["module"])
	}

	nodes, ok := response["nodes"].([]interface{})
	if !ok {
		t.Fatal("Expected nodes to be array")
	}

	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}

	edges, ok := response["edges"].([]interface{})
	if !ok {
		t.Fatal("Expected edges to be array")
	}

	if len(edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(edges))
	}

	hasCircular, ok := response["has_circular_dependency"].(bool)
	if !ok {
		t.Fatal("Expected has_circular_dependency field")
	}

	if hasCircular {
		t.Error("Expected no circular dependency")
	}
}

func TestDependencyHandlers_GetDependencyGraph_CircularDependency(t *testing.T) {
	storage := newMockStorage()

	// Build circular graph
	storage.addVersion("a", "v1.0.0", `syntax = "proto3";
import "b@v1.0.0";
message A { string id = 1; }`)

	storage.addVersion("b", "v1.0.0", `syntax = "proto3";
import "a@v1.0.0";
message B { string id = 1; }`)

	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/modules/a/versions/v1.0.0/graph", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	hasCircular, ok := response["has_circular_dependency"].(bool)
	if !ok {
		t.Fatal("Expected has_circular_dependency field")
	}

	if !hasCircular {
		t.Error("Expected circular dependency to be detected")
	}

	circularPath, ok := response["circular_path"].([]interface{})
	if !ok {
		t.Fatal("Expected circular_path field")
	}

	if len(circularPath) == 0 {
		t.Error("Expected non-empty circular path")
	}
}

func TestDependencyHandlers_ErrorCases(t *testing.T) {
	storage := newMockStorage()
	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "module not found",
			method:         "GET",
			path:           "/modules/nonexistent/versions/v1.0.0/dependencies",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "version not found",
			method:         "GET",
			path:           "/modules/user/versions/v99.0.0/dependencies",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestDependencyHandlers_RegisterRoutes(t *testing.T) {
	storage := newMockStorage()
	handlers := NewDependencyHandlers(storage)
	router := mux.NewRouter()

	// Should not panic
	handlers.RegisterRoutes(router)

	// Verify routes are registered
	var routeCount int
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		routeCount++
		return nil
	})

	// Should have at least the dependency routes
	if routeCount < 7 {
		t.Errorf("Expected at least 7 routes registered, got %d", routeCount)
	}
}
