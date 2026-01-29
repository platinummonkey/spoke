package codegen

import (
	"fmt"
	"sync"
)

// LanguageSpec defines a language configuration
type LanguageSpec struct {
	ID             string
	Name           string
	Enabled        bool
	DockerImage    string
	DockerTag      string
	ProtocFlags    []string
	SupportsGRPC   bool
	GRPCFlags      []string
	PluginVersion  string
	PackageManager *PackageManagerSpec
}

// PackageManagerSpec defines package manager configuration
type PackageManagerSpec struct {
	Name    string
	Enabled bool
}

var (
	languageRegistry = make(map[string]*LanguageSpec)
	registryMu       sync.RWMutex
)

// RegisterLanguage registers a language specification
func RegisterLanguage(spec *LanguageSpec) {
	registryMu.Lock()
	defer registryMu.Unlock()
	languageRegistry[spec.ID] = spec
}

// GetLanguageSpec retrieves a language specification
func GetLanguageSpec(langID string) (*LanguageSpec, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	spec, exists := languageRegistry[langID]
	if !exists {
		return nil, fmt.Errorf("language not found: %s", langID)
	}
	return spec, nil
}

// Initialize default languages
func init() {
	RegisterLanguage(&LanguageSpec{
		ID:            "go",
		Name:          "Go",
		Enabled:       true,
		DockerImage:   "bufbuild/buf",
		DockerTag:     "latest",
		ProtocFlags:   []string{"--go_opt=paths=source_relative"},
		SupportsGRPC:  true,
		GRPCFlags:     []string{"--go-grpc_opt=paths=source_relative"},
		PluginVersion: "v1.28.0",
		PackageManager: &PackageManagerSpec{
			Name:    "gomod",
			Enabled: true,
		},
	})

	RegisterLanguage(&LanguageSpec{
		ID:            "python",
		Name:          "Python",
		Enabled:       true,
		DockerImage:   "bufbuild/buf",
		DockerTag:     "latest",
		ProtocFlags:   []string{},
		SupportsGRPC:  true,
		GRPCFlags:     []string{},
		PluginVersion: "v4.21.0",
		PackageManager: &PackageManagerSpec{
			Name:    "pip",
			Enabled: true,
		},
	})

	RegisterLanguage(&LanguageSpec{
		ID:            "java",
		Name:          "Java",
		Enabled:       true,
		DockerImage:   "bufbuild/buf",
		DockerTag:     "latest",
		ProtocFlags:   []string{},
		SupportsGRPC:  true,
		GRPCFlags:     []string{},
		PluginVersion: "v3.21.0",
		PackageManager: &PackageManagerSpec{
			Name:    "maven",
			Enabled: true,
		},
	})

	RegisterLanguage(&LanguageSpec{
		ID:            "typescript",
		Name:          "TypeScript",
		Enabled:       true,
		DockerImage:   "bufbuild/buf",
		DockerTag:     "latest",
		ProtocFlags:   []string{},
		SupportsGRPC:  true,
		GRPCFlags:     []string{},
		PluginVersion: "v0.8.0",
		PackageManager: &PackageManagerSpec{
			Name:    "npm",
			Enabled: true,
		},
	})
}
