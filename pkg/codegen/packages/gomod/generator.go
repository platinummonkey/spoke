package gomod

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/packages"
)

//go:embed go.mod.tmpl
var goModTemplate string

//go:embed go.README.md.tmpl
var goReadmeTemplate string

// Generator generates Go module files
type Generator struct{}

// NewGenerator creates a new Go module generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate creates package manager files for Go
func (g *Generator) Generate(req *packages.GenerateRequest) ([]codegen.GeneratedFile, error) {
	var files []codegen.GeneratedFile

	// Generate go.mod
	goMod, err := g.generateGoMod(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate go.mod: %v", err)
	}
	files = append(files, goMod)

	// Generate README.md
	readme, err := g.generateReadme(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate README: %v", err)
	}
	files = append(files, readme)

	return files, nil
}

// GetName returns the name of the package manager
func (g *Generator) GetName() string {
	return "go-modules"
}

// GetConfigFiles returns the list of config files this generator creates
func (g *Generator) GetConfigFiles() []string {
	return []string{"go.mod", "README.md"}
}

func (g *Generator) generateGoMod(req *packages.GenerateRequest) (codegen.GeneratedFile, error) {
	tmpl, err := template.New("go.mod").Funcs(template.FuncMap{
		"default": func(def, val string) string {
			if val == "" {
				return def
			}
			return val
		},
	}).Parse(goModTemplate)
	if err != nil {
		return codegen.GeneratedFile{}, err
	}

	// Convert module name to Go import path format
	modulePath := convertToGoModulePath(req.ModuleName, req.Version)

	data := map[string]interface{}{
		"ModuleName":      modulePath,
		"GoVersion":       "1.21",
		"ProtobufVersion": "v1.31.0",
		"GRPCVersion":     "v1.60.0",
		"IncludeGRPC":     req.IncludeGRPC,
		"Dependencies":    convertDependencies(req.Dependencies),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return codegen.GeneratedFile{}, err
	}

	return codegen.GeneratedFile{
		Path:    "go.mod",
		Content: buf.Bytes(),
		Size:    int64(buf.Len()),
	}, nil
}

func (g *Generator) generateReadme(req *packages.GenerateRequest) (codegen.GeneratedFile, error) {
	tmpl, err := template.New("README.md").Funcs(template.FuncMap{
		"default": func(def, val string) string {
			if val == "" {
				return def
			}
			return val
		},
	}).Parse(goReadmeTemplate)
	if err != nil {
		return codegen.GeneratedFile{}, err
	}

	modulePath := convertToGoModulePath(req.ModuleName, req.Version)

	data := map[string]interface{}{
		"ModuleName":      modulePath,
		"Version":         req.Version,
		"GoVersion":       "1.21",
		"ProtobufVersion": "v1.31.0",
		"GRPCVersion":     "v1.60.0",
		"IncludeGRPC":     req.IncludeGRPC,
		"OriginalModule":  req.ModuleName,
		"OriginalVersion": req.Version,
		"GeneratedAt":     time.Now().Format(time.RFC3339),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return codegen.GeneratedFile{}, err
	}

	return codegen.GeneratedFile{
		Path:    "README.md",
		Content: buf.Bytes(),
		Size:    int64(buf.Len()),
	}, nil
}

// convertToGoModulePath converts a module name to a valid Go module path
func convertToGoModulePath(moduleName, version string) string {
	// Example: "user-service" -> "github.com/your-org/user-service"
	// This is a simple conversion - in real usage, the organization would be configurable
	path := strings.ToLower(moduleName)
	path = strings.ReplaceAll(path, "_", "-")
	return filepath.Join("github.com", "spoke-generated", path)
}

// convertDependencies converts package dependencies to Go module format
func convertDependencies(deps []packages.Dependency) []map[string]string {
	result := make([]map[string]string, 0, len(deps))
	for _, dep := range deps {
		result = append(result, map[string]string{
			"ImportPath": dep.ImportPath,
			"Version":    dep.Version,
		})
	}
	return result
}
