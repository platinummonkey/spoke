package pip

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/packages"
)

//go:embed setup.py.tmpl
var setupPyTemplate string

//go:embed pyproject.toml.tmpl
var pyprojectTemplate string

// Generator generates Python pip files
type Generator struct{}

// NewGenerator creates a new Python pip generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate creates package manager files for Python
func (g *Generator) Generate(req *packages.GenerateRequest) ([]codegen.GeneratedFile, error) {
	var files []codegen.GeneratedFile

	// Generate setup.py
	setupPy, err := g.generateSetupPy(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate setup.py: %v", err)
	}
	files = append(files, setupPy)

	// Generate pyproject.toml
	pyproject, err := g.generatePyproject(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pyproject.toml: %v", err)
	}
	files = append(files, pyproject)

	// Generate README.md
	readme := codegen.GeneratedFile{
		Path:    "README.md",
		Content: []byte(fmt.Sprintf("# %s\n\nProtocol Buffer generated code for Python.\n\n## Installation\n\n```bash\npip install %s\n```\n", req.ModuleName, convertToPythonPackageName(req.ModuleName))),
		Size:    0,
	}
	readme.Size = int64(len(readme.Content))
	files = append(files, readme)

	return files, nil
}

// GetName returns the name of the package manager
func (g *Generator) GetName() string {
	return "pip"
}

// GetConfigFiles returns the list of config files this generator creates
func (g *Generator) GetConfigFiles() []string {
	return []string{"setup.py", "pyproject.toml", "README.md"}
}

func (g *Generator) generateSetupPy(req *packages.GenerateRequest) (codegen.GeneratedFile, error) {
	tmpl, err := template.New("setup.py").Funcs(template.FuncMap{
		"default": func(def, val string) string {
			if val == "" {
				return def
			}
			return val
		},
	}).Parse(setupPyTemplate)
	if err != nil {
		return codegen.GeneratedFile{}, err
	}

	data := map[string]interface{}{
		"PackageName":     convertToPythonPackageName(req.ModuleName),
		"Version":         req.Version,
		"ModuleName":      req.ModuleName,
		"ProtobufVersion": "4.25.1",
		"GRPCVersion":     "1.60.0",
		"IncludeGRPC":     req.IncludeGRPC,
		"Dependencies":    convertDependencies(req.Dependencies),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return codegen.GeneratedFile{}, err
	}

	return codegen.GeneratedFile{
		Path:    "setup.py",
		Content: buf.Bytes(),
		Size:    int64(buf.Len()),
	}, nil
}

func (g *Generator) generatePyproject(req *packages.GenerateRequest) (codegen.GeneratedFile, error) {
	tmpl, err := template.New("pyproject.toml").Funcs(template.FuncMap{
		"default": func(def, val string) string {
			if val == "" {
				return def
			}
			return val
		},
	}).Parse(pyprojectTemplate)
	if err != nil {
		return codegen.GeneratedFile{}, err
	}

	data := map[string]interface{}{
		"PackageName":     convertToPythonPackageName(req.ModuleName),
		"Version":         req.Version,
		"ModuleName":      req.ModuleName,
		"ProtobufVersion": "4.25.1",
		"GRPCVersion":     "1.60.0",
		"IncludeGRPC":     req.IncludeGRPC,
		"Dependencies":    convertDependencies(req.Dependencies),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return codegen.GeneratedFile{}, err
	}

	return codegen.GeneratedFile{
		Path:    "pyproject.toml",
		Content: buf.Bytes(),
		Size:    int64(buf.Len()),
	}, nil
}

// convertToPythonPackageName converts a module name to a valid Python package name
func convertToPythonPackageName(moduleName string) string {
	// Example: "user-service" -> "user_service"
	name := strings.ToLower(moduleName)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

// convertDependencies converts package dependencies to Python format
func convertDependencies(deps []packages.Dependency) []map[string]string {
	result := make([]map[string]string, 0, len(deps))
	for _, dep := range deps {
		result = append(result, map[string]string{
			"Name":    dep.Name,
			"Version": dep.Version,
		})
	}
	return result
}
