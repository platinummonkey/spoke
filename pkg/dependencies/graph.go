package dependencies

import (
	"fmt"
	"sort"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// Dependency represents a module dependency
type Dependency struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Type    string `json:"type"` // "direct" or "transitive"
}

// DependencyGraph represents the dependency graph
type DependencyGraph struct {
	nodes map[string]*Node
	edges map[string][]string // key -> list of dependencies
}

// Node represents a node in the dependency graph
type Node struct {
	Module       string
	Version      string
	Dependencies []Dependency
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*Node),
		edges: make(map[string][]string),
	}
}

// AddNode adds a node to the graph
func (g *DependencyGraph) AddNode(module, version string, deps []Dependency) {
	key := moduleVersionKey(module, version)
	g.nodes[key] = &Node{
		Module:       module,
		Version:      version,
		Dependencies: deps,
	}

	// Add edges
	edges := make([]string, 0, len(deps))
	for _, dep := range deps {
		edges = append(edges, moduleVersionKey(dep.Module, dep.Version))
	}
	g.edges[key] = edges
}

// GetNode retrieves a node from the graph
func (g *DependencyGraph) GetNode(module, version string) *Node {
	return g.nodes[moduleVersionKey(module, version)]
}

// GetDependencies returns all dependencies for a module version
func (g *DependencyGraph) GetDependencies(module, version string) []Dependency {
	node := g.GetNode(module, version)
	if node == nil {
		return nil
	}
	return node.Dependencies
}

// GetTransitiveDependencies returns all transitive dependencies
func (g *DependencyGraph) GetTransitiveDependencies(module, version string) []Dependency {
	visited := make(map[string]bool)
	result := make([]Dependency, 0)

	var traverse func(string, string)
	traverse = func(mod, ver string) {
		key := moduleVersionKey(mod, ver)
		if visited[key] {
			return
		}
		visited[key] = true

		node := g.GetNode(mod, ver)
		if node == nil {
			return
		}

		for _, dep := range node.Dependencies {
			result = append(result, Dependency{
				Module:  dep.Module,
				Version: dep.Version,
				Type:    "transitive",
			})
			traverse(dep.Module, dep.Version)
		}
	}

	traverse(module, version)
	return result
}

// GetDependents returns all modules that depend on this module version
func (g *DependencyGraph) GetDependents(module, version string) []Dependency {
	target := moduleVersionKey(module, version)
	dependents := make([]Dependency, 0)

	for nodeKey, edges := range g.edges {
		for _, edge := range edges {
			if edge == target {
				// This node depends on our target
				if node, ok := g.nodes[nodeKey]; ok {
					dependents = append(dependents, Dependency{
						Module:  node.Module,
						Version: node.Version,
						Type:    "direct",
					})
				}
				break
			}
		}
	}

	return dependents
}

// DetectCircularDependencies detects circular dependencies
func (g *DependencyGraph) DetectCircularDependencies(module, version string) ([]string, error) {
	path := make([]string, 0)
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(key string) bool {
		visited[key] = true
		recStack[key] = true
		path = append(path, key)

		for _, dep := range g.edges[key] {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				// Found cycle
				return true
			}
		}

		recStack[key] = false
		path = path[:len(path)-1]
		return false
	}

	key := moduleVersionKey(module, version)
	if hasCycle(key) {
		return path, fmt.Errorf("circular dependency detected")
	}

	return nil, nil
}

// TopologicalSort performs a topological sort of dependencies
func (g *DependencyGraph) TopologicalSort(module, version string) ([]Dependency, error) {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	result := make([]Dependency, 0)

	var visit func(string) error
	visit = func(key string) error {
		if recStack[key] {
			return fmt.Errorf("circular dependency detected at %s", key)
		}
		if visited[key] {
			return nil
		}

		visited[key] = true
		recStack[key] = true

		// Visit dependencies first
		for _, dep := range g.edges[key] {
			if err := visit(dep); err != nil {
				return err
			}
		}

		recStack[key] = false

		// Add to result
		if node, ok := g.nodes[key]; ok {
			result = append(result, Dependency{
				Module:  node.Module,
				Version: node.Version,
				Type:    "direct",
			})
		}

		return nil
	}

	key := moduleVersionKey(module, version)
	if err := visit(key); err != nil {
		return nil, err
	}

	// Result is already in correct order (dependencies before dependents)
	return result, nil
}

// GetImpactAnalysis returns what would be affected by changes to this module
func (g *DependencyGraph) GetImpactAnalysis(module, version string) *ImpactAnalysis {
	directDependents := g.GetDependents(module, version)

	// Get all transitive dependents
	visited := make(map[string]bool)
	allDependents := make([]Dependency, 0)

	var traverse func(string, string)
	traverse = func(mod, ver string) {
		key := moduleVersionKey(mod, ver)
		if visited[key] {
			return
		}
		visited[key] = true

		deps := g.GetDependents(mod, ver)
		for _, dep := range deps {
			allDependents = append(allDependents, Dependency{
				Module:  dep.Module,
				Version: dep.Version,
				Type:    "transitive",
			})
			traverse(dep.Module, dep.Version)
		}
	}

	for _, dep := range directDependents {
		traverse(dep.Module, dep.Version)
	}

	return &ImpactAnalysis{
		Module:             module,
		Version:            version,
		DirectDependents:   directDependents,
		TransitiveDependents: allDependents,
		TotalImpact:        len(directDependents) + len(allDependents),
	}
}

// ImpactAnalysis represents the impact of changes
type ImpactAnalysis struct {
	Module               string       `json:"module"`
	Version              string       `json:"version"`
	DirectDependents     []Dependency `json:"direct_dependents"`
	TransitiveDependents []Dependency `json:"transitive_dependents"`
	TotalImpact          int          `json:"total_impact"`
}

func moduleVersionKey(module, version string) string {
	return module + "@" + version
}

// DependencyResolver resolves dependencies from proto imports
type DependencyResolver struct {
	storage api.Storage
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(storage api.Storage) *DependencyResolver {
	return &DependencyResolver{
		storage: storage,
	}
}

// ResolveDependencies extracts dependencies from a proto file
func (r *DependencyResolver) ResolveDependencies(moduleName, version string) ([]Dependency, error) {
	// Get the version
	ver, err := r.storage.GetVersion(moduleName, version)
	if err != nil {
		return nil, err
	}

	if len(ver.Files) == 0 {
		return []Dependency{}, nil
	}

	// Parse the proto file
	ast, err := protobuf.ParseString(ver.Files[0].Content)
	if err != nil {
		return nil, err
	}

	// Extract imports
	deps := make([]Dependency, 0)
	seen := make(map[string]bool)

	for _, imp := range ast.Imports {
		// Parse import path to extract module and version
		// Format: "common/v1.0.0/types.proto" or "common@v1.0.0"
		module, ver := parseImportPath(imp.Path)
		if module == "" {
			continue // Skip non-versioned imports
		}

		key := moduleVersionKey(module, ver)
		if !seen[key] {
			seen[key] = true
			deps = append(deps, Dependency{
				Module:  module,
				Version: ver,
				Type:    "direct",
			})
		}
	}

	return deps, nil
}

// BuildDependencyGraph builds a dependency graph for a module
func (r *DependencyResolver) BuildDependencyGraph(moduleName, version string) (*DependencyGraph, error) {
	graph := NewDependencyGraph()
	visited := make(map[string]bool)

	var build func(string, string) error
	build = func(mod, ver string) error {
		key := moduleVersionKey(mod, ver)
		if visited[key] {
			return nil
		}
		visited[key] = true

		// Resolve dependencies
		deps, err := r.ResolveDependencies(mod, ver)
		if err != nil {
			return err
		}

		// Add node to graph
		graph.AddNode(mod, ver, deps)

		// Recursively build for dependencies
		for _, dep := range deps {
			if err := build(dep.Module, dep.Version); err != nil {
				return err
			}
		}

		return nil
	}

	if err := build(moduleName, version); err != nil {
		return nil, err
	}

	return graph, nil
}

// parseImportPath parses an import path to extract module and version
// Supports formats:
// - "common@v1.0.0" (explicit version)
// - "common/v1.0.0/types.proto" (path-based version)
func parseImportPath(path string) (module, version string) {
	// Check for @ syntax (or - which is how @ gets preprocessed)
	// Try @ first, then fall back to -
	idx := indexByte(path, '@')
	if idx == -1 {
		idx = indexByte(path, '-')
	}

	if idx != -1 {
		module = path[:idx]
		versionPart := path[idx+1:]

		// Strip any path components after version
		if slashIdx := indexByte(versionPart, '/'); slashIdx != -1 {
			version = versionPart[:slashIdx]
		} else {
			version = versionPart
			// Strip .proto extension if present
			if len(version) > 6 && version[len(version)-6:] == ".proto" {
				version = version[:len(version)-6]
			}
		}
		return
	}

	// Check for path-based version
	// Format: module/version/file.proto
	parts := splitPath(path)
	if len(parts) >= 2 {
		module = parts[0]
		// Check if second part looks like a version
		if len(parts[1]) > 0 && parts[1][0] == 'v' {
			version = parts[1]
			return
		}
	}

	return "", ""
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func splitPath(path string) []string {
	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

// GenerateLockfile generates a dependency lockfile
func (r *DependencyResolver) GenerateLockfile(moduleName, version string) (*Lockfile, error) {
	graph, err := r.BuildDependencyGraph(moduleName, version)
	if err != nil {
		return nil, err
	}

	// Get all dependencies in topological order
	sorted, err := graph.TopologicalSort(moduleName, version)
	if err != nil {
		return nil, err
	}

	lockfile := &Lockfile{
		Module:       moduleName,
		Version:      version,
		Dependencies: sorted,
	}

	return lockfile, nil
}

// Lockfile represents a dependency lockfile
type Lockfile struct {
	Module       string       `json:"module"`
	Version      string       `json:"version"`
	Dependencies []Dependency `json:"dependencies"`
}

// ValidateLockfile validates a lockfile against the current dependencies
func (r *DependencyResolver) ValidateLockfile(lockfile *Lockfile) (bool, []string, error) {
	// Resolve current dependencies
	current, err := r.ResolveDependencies(lockfile.Module, lockfile.Version)
	if err != nil {
		return false, nil, err
	}

	// Build a map of current dependencies
	currentMap := make(map[string]string)
	for _, dep := range current {
		currentMap[dep.Module] = dep.Version
	}

	// Check for differences
	differences := make([]string, 0)

	for _, dep := range lockfile.Dependencies {
		if currentVer, ok := currentMap[dep.Module]; ok {
			if currentVer != dep.Version {
				differences = append(differences, fmt.Sprintf("%s: lockfile has %s, current has %s",
					dep.Module, dep.Version, currentVer))
			}
			delete(currentMap, dep.Module)
		} else {
			differences = append(differences, fmt.Sprintf("%s@%s: in lockfile but not in current dependencies",
				dep.Module, dep.Version))
		}
	}

	// Check for new dependencies
	for mod, ver := range currentMap {
		differences = append(differences, fmt.Sprintf("%s@%s: in current dependencies but not in lockfile",
			mod, ver))
	}

	sort.Strings(differences)
	return len(differences) == 0, differences, nil
}
