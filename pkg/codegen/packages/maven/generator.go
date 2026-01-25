package maven

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/packages"
)

//go:embed pom.xml.tmpl
var pomXMLTemplate string

// Generator generates Maven pom.xml files
type Generator struct{}

// NewGenerator creates a new Maven generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate creates package manager files for Maven
func (g *Generator) Generate(req *packages.GenerateRequest) ([]codegen.GeneratedFile, error) {
	var files []codegen.GeneratedFile

	// Generate pom.xml
	pomXML, err := g.generatePomXML(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pom.xml: %v", err)
	}
	files = append(files, pomXML)

	// Generate README.md
	readme := codegen.GeneratedFile{
		Path:    "README.md",
		Content: []byte(fmt.Sprintf("# %s\n\nProtocol Buffer generated code for Java.\n\n## Installation\n\n```xml\n<dependency>\n    <groupId>com.spoke.generated</groupId>\n    <artifactId>%s</artifactId>\n    <version>%s</version>\n</dependency>\n```\n", req.ModuleName, convertToArtifactId(req.ModuleName), req.Version)),
		Size:    0,
	}
	readme.Size = int64(len(readme.Content))
	files = append(files, readme)

	return files, nil
}

// GetName returns the name of the package manager
func (g *Generator) GetName() string {
	return "maven"
}

// GetConfigFiles returns the list of config files this generator creates
func (g *Generator) GetConfigFiles() []string {
	return []string{"pom.xml", "README.md"}
}

func (g *Generator) generatePomXML(req *packages.GenerateRequest) (codegen.GeneratedFile, error) {
	tmpl, err := template.New("pom.xml").Funcs(template.FuncMap{
		"default": func(def, val string) string {
			if val == "" {
				return def
			}
			return val
		},
	}).Parse(pomXMLTemplate)
	if err != nil {
		return codegen.GeneratedFile{}, err
	}

	data := map[string]interface{}{
		"GroupId":         "com.spoke.generated",
		"ArtifactId":      convertToArtifactId(req.ModuleName),
		"Version":         req.Version,
		"ModuleName":      req.ModuleName,
		"ProtobufVersion": "3.25.1",
		"GRPCVersion":     "1.60.0",
		"IncludeGRPC":     req.IncludeGRPC,
		"Dependencies":    convertDependencies(req.Dependencies),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return codegen.GeneratedFile{}, err
	}

	return codegen.GeneratedFile{
		Path:    "pom.xml",
		Content: buf.Bytes(),
		Size:    int64(buf.Len()),
	}, nil
}

// convertToArtifactId converts a module name to a valid Maven artifact ID
func convertToArtifactId(moduleName string) string {
	// Example: "user-service" -> "user-service"
	name := strings.ToLower(moduleName)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

// convertDependencies converts package dependencies to Maven format
func convertDependencies(deps []packages.Dependency) []map[string]string {
	result := make([]map[string]string, 0, len(deps))
	for _, dep := range deps {
		// Parse groupId:artifactId from dependency name
		parts := strings.Split(dep.Name, ":")
		groupId := "com.spoke.generated"
		artifactId := dep.Name

		if len(parts) == 2 {
			groupId = parts[0]
			artifactId = parts[1]
		}

		result = append(result, map[string]string{
			"GroupId":    groupId,
			"ArtifactId": artifactId,
			"Version":    dep.Version,
		})
	}
	return result
}
