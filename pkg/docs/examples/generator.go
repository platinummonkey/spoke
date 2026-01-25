package examples

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/platinummonkey/spoke/pkg/api"
)

//go:embed templates/*/*.tmpl
var templateFS embed.FS

// Generator generates code examples from protobuf definitions
type Generator struct {
	templates map[string]*template.Template
}

// NewGenerator creates a new example generator
func NewGenerator() (*Generator, error) {
	g := &Generator{
		templates: make(map[string]*template.Template),
	}

	// Load templates for each language
	languages := []string{"go", "python", "java", "rust", "typescript"}
	for _, lang := range languages {
		tmplPath := fmt.Sprintf("templates/%s/grpc-client.tmpl", lang)
		tmplContent, err := templateFS.ReadFile(tmplPath)
		if err != nil {
			// Template may not exist yet, skip
			continue
		}

		tmpl, err := template.New(lang).Parse(string(tmplContent))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template for %s: %w", lang, err)
		}

		g.templates[lang] = tmpl
	}

	return g, nil
}

// Generate generates a code example for the given language
func (g *Generator) Generate(language, moduleName, version string, files []api.File) (string, error) {
	tmpl, ok := g.templates[language]
	if !ok {
		return "", fmt.Errorf("template not found for language: %s", language)
	}

	// Extract services and messages from files
	data := g.extractData(language, moduleName, version, files)

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// extractData extracts example data from proto files
func (g *Generator) extractData(language, moduleName, version string, files []api.File) ExampleData {
	data := ExampleData{
		Language:    language,
		ModuleName:  moduleName,
		Version:     version,
		PackagePath: g.getPackagePath(language, moduleName),
		Imports:     g.getImports(language),
		PackageManager: PackageManagerInfo{
			Command:        g.getPackageCommand(language),
			PackageName:    moduleName,
			InstallExample: g.getInstallExample(language, moduleName),
		},
	}

	// Parse files and extract services/methods
	// For now, create a simple example
	// TODO: Implement actual parsing when needed

	return data
}

func (g *Generator) getPackagePath(language, moduleName string) string {
	switch language {
	case "go":
		return fmt.Sprintf("github.com/example/%s", moduleName)
	case "python":
		return strings.ReplaceAll(moduleName, "-", "_")
	case "java":
		return fmt.Sprintf("com.example.%s", moduleName)
	case "rust":
		return strings.ReplaceAll(moduleName, "-", "_")
	case "typescript":
		return moduleName
	default:
		return moduleName
	}
}

func (g *Generator) getImports(language string) []string {
	switch language {
	case "go":
		return []string{
			"context",
			"log",
			"google.golang.org/grpc",
			"google.golang.org/grpc/credentials/insecure",
		}
	case "python":
		return []string{
			"grpc",
		}
	case "java":
		return []string{
			"io.grpc.ManagedChannel",
			"io.grpc.ManagedChannelBuilder",
		}
	case "rust":
		return []string{
			"tonic::transport::Channel",
		}
	case "typescript":
		return []string{
			"@grpc/grpc-js",
		}
	default:
		return []string{}
	}
}

func (g *Generator) getPackageCommand(language string) string {
	switch language {
	case "go":
		return "go get"
	case "python":
		return "pip install"
	case "java":
		return "maven"
	case "rust":
		return "cargo add"
	case "typescript":
		return "npm install"
	default:
		return "install"
	}
}

func (g *Generator) getInstallExample(language, moduleName string) string {
	switch language {
	case "go":
		return fmt.Sprintf("go get github.com/example/%s", moduleName)
	case "python":
		return fmt.Sprintf("pip install %s", strings.ReplaceAll(moduleName, "-", "_"))
	case "java":
		return fmt.Sprintf("<dependency>%s</dependency>", moduleName)
	case "rust":
		return fmt.Sprintf("cargo add %s", moduleName)
	case "typescript":
		return fmt.Sprintf("npm install %s", moduleName)
	default:
		return moduleName
	}
}

// getSampleValue generates a sample value for a field based on its name and type
func getSampleValue(fieldName, fieldType string) string {
	// Based on field name heuristics
	fieldNameLower := strings.ToLower(fieldName)

	if strings.Contains(fieldNameLower, "email") {
		return `"user@example.com"`
	}
	if strings.Contains(fieldNameLower, "name") {
		return `"Example Name"`
	}
	if strings.Contains(fieldNameLower, "id") {
		return `"example-id-123"`
	}

	// Based on type
	switch fieldType {
	case "string":
		return `"example value"`
	case "int32", "int64", "uint32", "uint64":
		return "42"
	case "float", "double":
		return "3.14"
	case "bool":
		return "true"
	case "bytes":
		return `"base64-encoded-data"`
	default:
		// Message type
		return "{}"
	}
}
