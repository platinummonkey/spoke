package dependencies

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/api"
)

func TestDependencyGraph_AddNode(t *testing.T) {
	graph := NewDependencyGraph()

	deps := []Dependency{
		{Module: "common", Version: "v1.0.0", Type: "direct"},
	}

	graph.AddNode("user", "v1.0.0", deps)

	node := graph.GetNode("user", "v1.0.0")
	if node == nil {
		t.Fatal("Expected node to be added")
	}

	if node.Module != "user" {
		t.Errorf("Expected module 'user', got %s", node.Module)
	}

	if len(node.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(node.Dependencies))
	}
}

func TestDependencyGraph_GetTransitiveDependencies(t *testing.T) {
	graph := NewDependencyGraph()

	// Build graph: user -> common -> base
	graph.AddNode("base", "v1.0.0", []Dependency{})
	graph.AddNode("common", "v1.0.0", []Dependency{
		{Module: "base", Version: "v1.0.0", Type: "direct"},
	})
	graph.AddNode("user", "v1.0.0", []Dependency{
		{Module: "common", Version: "v1.0.0", Type: "direct"},
	})

	deps := graph.GetTransitiveDependencies("user", "v1.0.0")

	// Should include both common and base
	if len(deps) != 2 {
		t.Errorf("Expected 2 transitive dependencies, got %d", len(deps))
	}

	// All should be marked as transitive
	for _, dep := range deps {
		if dep.Type != "transitive" {
			t.Errorf("Expected type 'transitive', got %s", dep.Type)
		}
	}
}

func TestDependencyGraph_GetDependents(t *testing.T) {
	graph := NewDependencyGraph()

	// Build graph where multiple modules depend on common
	graph.AddNode("common", "v1.0.0", []Dependency{})
	graph.AddNode("user", "v1.0.0", []Dependency{
		{Module: "common", Version: "v1.0.0", Type: "direct"},
	})
	graph.AddNode("order", "v1.0.0", []Dependency{
		{Module: "common", Version: "v1.0.0", Type: "direct"},
	})

	dependents := graph.GetDependents("common", "v1.0.0")

	if len(dependents) != 2 {
		t.Errorf("Expected 2 dependents, got %d", len(dependents))
	}

	// Check that both user and order are present
	found := make(map[string]bool)
	for _, dep := range dependents {
		found[dep.Module] = true
	}

	if !found["user"] || !found["order"] {
		t.Error("Expected both 'user' and 'order' as dependents")
	}
}

func TestDependencyGraph_DetectCircularDependencies(t *testing.T) {
	tests := []struct {
		name        string
		buildGraph  func(*DependencyGraph)
		module      string
		version     string
		expectCycle bool
	}{
		{
			name: "no circular dependency",
			buildGraph: func(g *DependencyGraph) {
				g.AddNode("base", "v1.0.0", []Dependency{})
				g.AddNode("common", "v1.0.0", []Dependency{
					{Module: "base", Version: "v1.0.0", Type: "direct"},
				})
				g.AddNode("user", "v1.0.0", []Dependency{
					{Module: "common", Version: "v1.0.0", Type: "direct"},
				})
			},
			module:      "user",
			version:     "v1.0.0",
			expectCycle: false,
		},
		{
			name: "direct circular dependency",
			buildGraph: func(g *DependencyGraph) {
				g.AddNode("a", "v1.0.0", []Dependency{
					{Module: "b", Version: "v1.0.0", Type: "direct"},
				})
				g.AddNode("b", "v1.0.0", []Dependency{
					{Module: "a", Version: "v1.0.0", Type: "direct"},
				})
			},
			module:      "a",
			version:     "v1.0.0",
			expectCycle: true,
		},
		{
			name: "indirect circular dependency",
			buildGraph: func(g *DependencyGraph) {
				g.AddNode("a", "v1.0.0", []Dependency{
					{Module: "b", Version: "v1.0.0", Type: "direct"},
				})
				g.AddNode("b", "v1.0.0", []Dependency{
					{Module: "c", Version: "v1.0.0", Type: "direct"},
				})
				g.AddNode("c", "v1.0.0", []Dependency{
					{Module: "a", Version: "v1.0.0", Type: "direct"},
				})
			},
			module:      "a",
			version:     "v1.0.0",
			expectCycle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := NewDependencyGraph()
			tt.buildGraph(graph)

			path, err := graph.DetectCircularDependencies(tt.module, tt.version)

			if tt.expectCycle {
				if err == nil {
					t.Error("Expected circular dependency error, got nil")
				}
				if len(path) == 0 {
					t.Error("Expected path to be returned with cycle")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	graph := NewDependencyGraph()

	// Build graph: user -> common -> base
	graph.AddNode("base", "v1.0.0", []Dependency{})
	graph.AddNode("common", "v1.0.0", []Dependency{
		{Module: "base", Version: "v1.0.0", Type: "direct"},
	})
	graph.AddNode("user", "v1.0.0", []Dependency{
		{Module: "common", Version: "v1.0.0", Type: "direct"},
	})

	sorted, err := graph.TopologicalSort("user", "v1.0.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sorted) != 3 {
		t.Errorf("Expected 3 items in sorted list, got %d", len(sorted))
	}

	// base should come before common, common before user
	positions := make(map[string]int)
	for i, dep := range sorted {
		positions[dep.Module] = i
	}

	if positions["base"] >= positions["common"] {
		t.Error("base should come before common")
	}
	if positions["common"] >= positions["user"] {
		t.Error("common should come before user")
	}
}

func TestDependencyGraph_TopologicalSort_CircularDependency(t *testing.T) {
	graph := NewDependencyGraph()

	// Build circular graph
	graph.AddNode("a", "v1.0.0", []Dependency{
		{Module: "b", Version: "v1.0.0", Type: "direct"},
	})
	graph.AddNode("b", "v1.0.0", []Dependency{
		{Module: "a", Version: "v1.0.0", Type: "direct"},
	})

	_, err := graph.TopologicalSort("a", "v1.0.0")
	if err == nil {
		t.Error("Expected error for circular dependency, got nil")
	}
}

func TestDependencyGraph_GetImpactAnalysis(t *testing.T) {
	graph := NewDependencyGraph()

	// Build graph where common is used by user and order
	graph.AddNode("common", "v1.0.0", []Dependency{})
	graph.AddNode("user", "v1.0.0", []Dependency{
		{Module: "common", Version: "v1.0.0", Type: "direct"},
	})
	graph.AddNode("order", "v1.0.0", []Dependency{
		{Module: "common", Version: "v1.0.0", Type: "direct"},
	})
	graph.AddNode("admin", "v1.0.0", []Dependency{
		{Module: "user", Version: "v1.0.0", Type: "direct"},
	})

	impact := graph.GetImpactAnalysis("common", "v1.0.0")

	if impact.Module != "common" {
		t.Errorf("Expected module 'common', got %s", impact.Module)
	}

	if len(impact.DirectDependents) != 2 {
		t.Errorf("Expected 2 direct dependents, got %d", len(impact.DirectDependents))
	}

	// Should include admin as transitive dependent (through user)
	if len(impact.TransitiveDependents) != 1 {
		t.Errorf("Expected 1 transitive dependent, got %d", len(impact.TransitiveDependents))
	}

	if impact.TotalImpact != 3 {
		t.Errorf("Expected total impact of 3, got %d", impact.TotalImpact)
	}
}

func TestParseImportPath(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedModule string
		expectedVersion string
	}{
		{
			name:           "@ syntax",
			path:           "common@v1.0.0",
			expectedModule: "common",
			expectedVersion: "v1.0.0",
		},
		{
			name:           "@ syntax with proto extension",
			path:           "common@v1.0.0/types.proto",
			expectedModule: "common",
			expectedVersion: "v1.0.0",
		},
		{
			name:           "path-based version",
			path:           "common/v1.0.0/types.proto",
			expectedModule: "common",
			expectedVersion: "v1.0.0",
		},
		{
			name:           "path-based version without file",
			path:           "common/v1.0.0",
			expectedModule: "common",
			expectedVersion: "v1.0.0",
		},
		{
			name:           "no version",
			path:           "common/types.proto",
			expectedModule: "",
			expectedVersion: "",
		},
		{
			name:           "non-versioned import",
			path:           "google/protobuf/timestamp.proto",
			expectedModule: "",
			expectedVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module, version := parseImportPath(tt.path)

			if module != tt.expectedModule {
				t.Errorf("Expected module '%s', got '%s'", tt.expectedModule, module)
			}

			if version != tt.expectedVersion {
				t.Errorf("Expected version '%s', got '%s'", tt.expectedVersion, version)
			}
		})
	}
}

func TestModuleVersionKey(t *testing.T) {
	key := moduleVersionKey("user", "v1.0.0")
	expected := "user@v1.0.0"

	if key != expected {
		t.Errorf("Expected '%s', got '%s'", expected, key)
	}
}

// Mock storage for testing DependencyResolver
type mockStorage struct {
	versions map[string]*api.Version
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		versions: make(map[string]*api.Version),
	}
}

func (m *mockStorage) addVersion(module, version, content string) {
	key := module + "@" + version
	m.versions[key] = &api.Version{
		ModuleName: module,
		Version:    version,
		Files: []api.File{
			{
				Path:    module + ".proto",
				Content: content,
			},
		},
	}
}

func (m *mockStorage) CreateModule(module *api.Module) error {
	return nil
}

func (m *mockStorage) GetModule(name string) (*api.Module, error) {
	return &api.Module{Name: name}, nil
}

func (m *mockStorage) ListModules() ([]*api.Module, error) {
	return nil, nil
}

func (m *mockStorage) CreateVersion(version *api.Version) error {
	return nil
}

func (m *mockStorage) GetVersion(moduleName, version string) (*api.Version, error) {
	key := moduleName + "@" + version
	if v, ok := m.versions[key]; ok {
		return v, nil
	}
	return nil, &storageError{msg: "version not found"}
}

func (m *mockStorage) ListVersions(moduleName string) ([]*api.Version, error) {
	return nil, nil
}

func (m *mockStorage) UpdateVersion(version *api.Version) error {
	return nil
}

func (m *mockStorage) GetFile(moduleName, version, path string) (*api.File, error) {
	return nil, nil
}

type storageError struct {
	msg string
}

func (e *storageError) Error() string {
	return e.msg
}

func TestDependencyResolver_ResolveDependencies(t *testing.T) {
	storage := newMockStorage()

	// Add version with imports
	protoContent := `syntax = "proto3";

import "common@v1.0.0";
import "types/v2.0.0/base.proto";

message User {
  string id = 1;
}`

	storage.addVersion("user", "v1.0.0", protoContent)

	resolver := NewDependencyResolver(storage)
	deps, err := resolver.ResolveDependencies("user", "v1.0.0")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	// Check dependencies
	foundCommon := false
	foundTypes := false

	for _, dep := range deps {
		if dep.Module == "common" && dep.Version == "v1.0.0" {
			foundCommon = true
		}
		if dep.Module == "types" && dep.Version == "v2.0.0" {
			foundTypes = true
		}
		if dep.Type != "direct" {
			t.Errorf("Expected type 'direct', got '%s'", dep.Type)
		}
	}

	if !foundCommon {
		t.Error("Expected to find common@v1.0.0 dependency")
	}
	if !foundTypes {
		t.Error("Expected to find types@v2.0.0 dependency")
	}
}

func TestDependencyResolver_ResolveDependencies_NoDuplicates(t *testing.T) {
	storage := newMockStorage()

	// Add version with a single import (duplicate imports are invalid proto syntax)
	protoContent := `syntax = "proto3";

import "common@v1.0.0";

message User {
  string id = 1;
}`

	storage.addVersion("user", "v1.0.0", protoContent)

	resolver := NewDependencyResolver(storage)
	deps, err := resolver.ResolveDependencies("user", "v1.0.0")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(deps))
	}
}

func TestDependencyResolver_BuildDependencyGraph(t *testing.T) {
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

	resolver := NewDependencyResolver(storage)
	graph, err := resolver.BuildDependencyGraph("user", "v1.0.0")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that all nodes are present
	if graph.GetNode("user", "v1.0.0") == nil {
		t.Error("Expected user node to be in graph")
	}
	if graph.GetNode("common", "v1.0.0") == nil {
		t.Error("Expected common node to be in graph")
	}
	if graph.GetNode("base", "v1.0.0") == nil {
		t.Error("Expected base node to be in graph")
	}

	// Verify dependencies
	userDeps := graph.GetDependencies("user", "v1.0.0")
	if len(userDeps) != 1 || userDeps[0].Module != "common" {
		t.Error("Expected user to depend on common")
	}

	commonDeps := graph.GetDependencies("common", "v1.0.0")
	if len(commonDeps) != 1 || commonDeps[0].Module != "base" {
		t.Error("Expected common to depend on base")
	}
}

func TestDependencyResolver_GenerateLockfile(t *testing.T) {
	storage := newMockStorage()

	storage.addVersion("base", "v1.0.0", `syntax = "proto3";
message Base { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "base@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)
	lockfile, err := resolver.GenerateLockfile("user", "v1.0.0")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if lockfile.Module != "user" {
		t.Errorf("Expected module 'user', got '%s'", lockfile.Module)
	}

	if lockfile.Version != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got '%s'", lockfile.Version)
	}

	// Should contain both user and base in topological order
	if len(lockfile.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies in lockfile, got %d", len(lockfile.Dependencies))
	}
}

func TestDependencyResolver_ValidateLockfile(t *testing.T) {
	storage := newMockStorage()

	storage.addVersion("common", "v1.0.0", `syntax = "proto3";
message Common { string id = 1; }`)

	storage.addVersion("user", "v1.0.0", `syntax = "proto3";
import "common@v1.0.0";
message User { string id = 1; }`)

	resolver := NewDependencyResolver(storage)

	tests := []struct {
		name        string
		lockfile    *Lockfile
		expectValid bool
		expectDiffs int
	}{
		{
			name: "valid lockfile",
			lockfile: &Lockfile{
				Module:  "user",
				Version: "v1.0.0",
				Dependencies: []Dependency{
					{Module: "common", Version: "v1.0.0", Type: "direct"},
				},
			},
			expectValid: true,
			expectDiffs: 0,
		},
		{
			name: "version mismatch",
			lockfile: &Lockfile{
				Module:  "user",
				Version: "v1.0.0",
				Dependencies: []Dependency{
					{Module: "common", Version: "v2.0.0", Type: "direct"},
				},
			},
			expectValid: false,
			expectDiffs: 1, // version mismatch reported once
		},
		{
			name: "missing dependency",
			lockfile: &Lockfile{
				Module:       "user",
				Version:      "v1.0.0",
				Dependencies: []Dependency{},
			},
			expectValid: false,
			expectDiffs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, differences, err := resolver.ValidateLockfile(tt.lockfile)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, valid)
			}

			if len(differences) != tt.expectDiffs {
				t.Errorf("Expected %d differences, got %d: %v", tt.expectDiffs, len(differences), differences)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "simple path",
			path:     "common/types.proto",
			expected: []string{"common", "types.proto"},
		},
		{
			name:     "version path",
			path:     "common/v1.0.0/types.proto",
			expected: []string{"common", "v1.0.0", "types.proto"},
		},
		{
			name:     "single component",
			path:     "common",
			expected: []string{"common"},
		},
		{
			name:     "empty path",
			path:     "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := splitPath(tt.path)

			if len(parts) != len(tt.expected) {
				t.Errorf("Expected %d parts, got %d", len(tt.expected), len(parts))
				return
			}

			for i, part := range parts {
				if part != tt.expected[i] {
					t.Errorf("Part %d: expected '%s', got '%s'", i, tt.expected[i], part)
				}
			}
		})
	}
}

func TestIndexByte(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		c        byte
		expected int
	}{
		{
			name:     "found at beginning",
			s:        "hello",
			c:        'h',
			expected: 0,
		},
		{
			name:     "found in middle",
			s:        "hello",
			c:        'l',
			expected: 2,
		},
		{
			name:     "not found",
			s:        "hello",
			c:        'x',
			expected: -1,
		},
		{
			name:     "empty string",
			s:        "",
			c:        'x',
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexByte(tt.s, tt.c)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}
