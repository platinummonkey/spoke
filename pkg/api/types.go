package api

import "time"

// Module represents a protobuf module with its metadata
type Module struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Version represents a specific version of a protobuf module
type Version struct {
	ModuleName  string    `json:"module_name"`
	Version     string    `json:"version"` // Can be semantic version or commit hash
	Files       []File    `json:"files"`
	CreatedAt   time.Time `json:"created_at"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// File represents a single protobuf file
type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Storage interface defines the methods required for storing and retrieving protobuf modules
type Storage interface {
	// Module operations
	CreateModule(module *Module) error
	GetModule(name string) (*Module, error)
	ListModules() ([]*Module, error)
	
	// Version operations
	CreateVersion(version *Version) error
	GetVersion(moduleName, version string) (*Version, error)
	ListVersions(moduleName string) ([]*Version, error)
	
	// File operations
	GetFile(moduleName, version, path string) (*File, error)
} 