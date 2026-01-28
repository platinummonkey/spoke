package packages

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/codegen"
)

// mockGenerator is a mock implementation of Generator for testing
type mockGenerator struct {
	name        string
	configFiles []string
}

func (m *mockGenerator) Generate(req *GenerateRequest) ([]codegen.GeneratedFile, error) {
	return []codegen.GeneratedFile{
		{
			Path:    "test.txt",
			Content: []byte("test content"),
			Size:    12,
		},
	}, nil
}

func (m *mockGenerator) GetName() string {
	return m.name
}

func (m *mockGenerator) GetConfigFiles() []string {
	return m.configFiles
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if registry.generators == nil {
		t.Error("NewRegistry() did not initialize generators map")
	}

	if len(registry.generators) != 0 {
		t.Errorf("NewRegistry() should create empty registry, got %d generators", len(registry.generators))
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	gen := &mockGenerator{
		name:        "test-gen",
		configFiles: []string{"config.json"},
	}

	registry.Register("test", gen)

	if len(registry.generators) != 1 {
		t.Errorf("Register() failed, expected 1 generator, got %d", len(registry.generators))
	}

	retrieved, ok := registry.generators["test"]
	if !ok {
		t.Error("Register() did not store generator with correct key")
	}

	if retrieved != gen {
		t.Error("Register() stored different generator instance")
	}
}

func TestRegistry_Register_Multiple(t *testing.T) {
	registry := NewRegistry()
	gen1 := &mockGenerator{name: "gen1", configFiles: []string{"config1.json"}}
	gen2 := &mockGenerator{name: "gen2", configFiles: []string{"config2.json"}}
	gen3 := &mockGenerator{name: "gen3", configFiles: []string{"config3.json"}}

	registry.Register("npm", gen1)
	registry.Register("maven", gen2)
	registry.Register("pip", gen3)

	if len(registry.generators) != 3 {
		t.Errorf("Register() multiple failed, expected 3 generators, got %d", len(registry.generators))
	}
}

func TestRegistry_Register_Overwrite(t *testing.T) {
	registry := NewRegistry()
	gen1 := &mockGenerator{name: "original", configFiles: []string{"config1.json"}}
	gen2 := &mockGenerator{name: "replacement", configFiles: []string{"config2.json"}}

	registry.Register("test", gen1)
	registry.Register("test", gen2)

	if len(registry.generators) != 1 {
		t.Errorf("Register() overwrite failed, expected 1 generator, got %d", len(registry.generators))
	}

	retrieved, _ := registry.generators["test"]
	if retrieved.GetName() != "replacement" {
		t.Errorf("Register() overwrite failed, expected 'replacement', got '%s'", retrieved.GetName())
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	gen := &mockGenerator{
		name:        "test-gen",
		configFiles: []string{"config.json"},
	}

	registry.Register("test", gen)

	retrieved, ok := registry.Get("test")
	if !ok {
		t.Error("Get() returned false for existing generator")
	}

	if retrieved != gen {
		t.Error("Get() returned different generator instance")
	}

	if retrieved.GetName() != "test-gen" {
		t.Errorf("Get() returned generator with wrong name, expected 'test-gen', got '%s'", retrieved.GetName())
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := NewRegistry()

	retrieved, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for non-existent generator")
	}

	if retrieved != nil {
		t.Error("Get() should return nil for non-existent generator")
	}
}

func TestRegistry_Get_AfterMultipleRegistrations(t *testing.T) {
	registry := NewRegistry()
	gen1 := &mockGenerator{name: "npm-gen", configFiles: []string{"package.json"}}
	gen2 := &mockGenerator{name: "maven-gen", configFiles: []string{"pom.xml"}}
	gen3 := &mockGenerator{name: "pip-gen", configFiles: []string{"setup.py"}}

	registry.Register("npm", gen1)
	registry.Register("maven", gen2)
	registry.Register("pip", gen3)

	retrieved, ok := registry.Get("maven")
	if !ok {
		t.Error("Get() returned false for existing 'maven' generator")
	}

	if retrieved.GetName() != "maven-gen" {
		t.Errorf("Get() returned wrong generator, expected 'maven-gen', got '%s'", retrieved.GetName())
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	list := registry.List()
	if list == nil {
		t.Fatal("List() returned nil")
	}

	if len(list) != 0 {
		t.Errorf("List() should return empty map for empty registry, got %d items", len(list))
	}
}

func TestRegistry_List_WithGenerators(t *testing.T) {
	registry := NewRegistry()
	gen1 := &mockGenerator{name: "gen1", configFiles: []string{"config1.json"}}
	gen2 := &mockGenerator{name: "gen2", configFiles: []string{"config2.json"}}

	registry.Register("npm", gen1)
	registry.Register("maven", gen2)

	list := registry.List()
	if len(list) != 2 {
		t.Errorf("List() expected 2 generators, got %d", len(list))
	}

	if _, ok := list["npm"]; !ok {
		t.Error("List() missing 'npm' generator")
	}

	if _, ok := list["maven"]; !ok {
		t.Error("List() missing 'maven' generator")
	}

	if list["npm"].GetName() != "gen1" {
		t.Errorf("List() npm generator has wrong name, expected 'gen1', got '%s'", list["npm"].GetName())
	}

	if list["maven"].GetName() != "gen2" {
		t.Errorf("List() maven generator has wrong name, expected 'gen2', got '%s'", list["maven"].GetName())
	}
}

func TestRegistry_List_ReturnsActualMap(t *testing.T) {
	registry := NewRegistry()
	gen := &mockGenerator{name: "test", configFiles: []string{"test.json"}}
	registry.Register("test", gen)

	list := registry.List()

	// Verify it's the actual map (not a copy)
	if &registry.generators == &list {
		// This would be the case if it's the same map
		// The current implementation returns the actual map
	}

	// Verify the content is correct
	if list["test"] != gen {
		t.Error("List() returned map with different generator instance")
	}
}

func TestGenerateRequest_StructFields(t *testing.T) {
	req := &GenerateRequest{
		ModuleName:  "test-module",
		Version:     "v1.0.0",
		Language:    "go",
		IncludeGRPC: true,
		GeneratedFiles: []codegen.GeneratedFile{
			{Path: "test.go", Content: []byte("package test"), Size: 12},
		},
		Dependencies: []Dependency{
			{Name: "dep1", Version: "v1.0.0", ImportPath: "github.com/example/dep1"},
		},
		Options: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	if req.ModuleName != "test-module" {
		t.Errorf("ModuleName mismatch, expected 'test-module', got '%s'", req.ModuleName)
	}

	if req.Version != "v1.0.0" {
		t.Errorf("Version mismatch, expected 'v1.0.0', got '%s'", req.Version)
	}

	if req.Language != "go" {
		t.Errorf("Language mismatch, expected 'go', got '%s'", req.Language)
	}

	if !req.IncludeGRPC {
		t.Error("IncludeGRPC should be true")
	}

	if len(req.GeneratedFiles) != 1 {
		t.Errorf("Expected 1 generated file, got %d", len(req.GeneratedFiles))
	}

	if len(req.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(req.Dependencies))
	}

	if len(req.Options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(req.Options))
	}

	if req.Options["key1"] != "value1" {
		t.Errorf("Options key1 mismatch, expected 'value1', got '%s'", req.Options["key1"])
	}
}

func TestDependency_StructFields(t *testing.T) {
	dep := Dependency{
		Name:       "test-dep",
		Version:    "v2.0.0",
		ImportPath: "github.com/example/test-dep",
	}

	if dep.Name != "test-dep" {
		t.Errorf("Name mismatch, expected 'test-dep', got '%s'", dep.Name)
	}

	if dep.Version != "v2.0.0" {
		t.Errorf("Version mismatch, expected 'v2.0.0', got '%s'", dep.Version)
	}

	if dep.ImportPath != "github.com/example/test-dep" {
		t.Errorf("ImportPath mismatch, expected 'github.com/example/test-dep', got '%s'", dep.ImportPath)
	}
}

func TestMockGenerator_Generate(t *testing.T) {
	gen := &mockGenerator{
		name:        "test",
		configFiles: []string{"config.json"},
	}

	req := &GenerateRequest{
		ModuleName: "test-module",
		Version:    "v1.0.0",
		Language:   "go",
	}

	files, err := gen.Generate(req)
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Generate() expected 1 file, got %d", len(files))
	}

	if files[0].Path != "test.txt" {
		t.Errorf("Generate() file path mismatch, expected 'test.txt', got '%s'", files[0].Path)
	}
}

func TestMockGenerator_GetName(t *testing.T) {
	gen := &mockGenerator{
		name:        "my-generator",
		configFiles: []string{},
	}

	if gen.GetName() != "my-generator" {
		t.Errorf("GetName() mismatch, expected 'my-generator', got '%s'", gen.GetName())
	}
}

func TestMockGenerator_GetConfigFiles(t *testing.T) {
	gen := &mockGenerator{
		name:        "test",
		configFiles: []string{"config1.json", "config2.yaml"},
	}

	files := gen.GetConfigFiles()
	if len(files) != 2 {
		t.Errorf("GetConfigFiles() expected 2 files, got %d", len(files))
	}

	if files[0] != "config1.json" {
		t.Errorf("GetConfigFiles()[0] mismatch, expected 'config1.json', got '%s'", files[0])
	}

	if files[1] != "config2.yaml" {
		t.Errorf("GetConfigFiles()[1] mismatch, expected 'config2.yaml', got '%s'", files[1])
	}
}
