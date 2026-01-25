package packages

import (
	"github.com/platinummonkey/spoke/pkg/codegen"
)

// Generator generates package manager configuration files
type Generator interface {
	// Generate creates package manager files for the compiled code
	Generate(req *GenerateRequest) ([]codegen.GeneratedFile, error)

	// GetName returns the name of the package manager
	GetName() string

	// GetConfigFiles returns the list of config files this generator creates
	GetConfigFiles() []string
}

// GenerateRequest represents a package generation request
type GenerateRequest struct {
	// Module information
	ModuleName    string
	Version       string

	// Language
	Language      string

	// Generated proto files (for detecting what was generated)
	GeneratedFiles []codegen.GeneratedFile

	// Dependencies
	Dependencies  []Dependency

	// Options
	IncludeGRPC   bool
	Options       map[string]string
}

// Dependency represents a module dependency for package managers
type Dependency struct {
	Name     string
	Version  string
	ImportPath string // Language-specific import path
}

// Registry manages package generators
type Registry struct {
	generators map[string]Generator
}

// NewRegistry creates a new package generator registry
func NewRegistry() *Registry {
	return &Registry{
		generators: make(map[string]Generator),
	}
}

// Register adds a generator to the registry
func (r *Registry) Register(name string, gen Generator) {
	r.generators[name] = gen
}

// Get retrieves a generator by name
func (r *Registry) Get(name string) (Generator, bool) {
	gen, ok := r.generators[name]
	return gen, ok
}

// List returns all registered generators
func (r *Registry) List() map[string]Generator {
	return r.generators
}
