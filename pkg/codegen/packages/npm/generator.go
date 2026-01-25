package npm

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/packages"
)

//go:embed package.json.tmpl
var packageJSONTemplate string

// Generator generates npm package files
type Generator struct{}

// NewGenerator creates a new npm generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate creates package manager files for npm
func (g *Generator) Generate(req *packages.GenerateRequest) ([]codegen.GeneratedFile, error) {
	var files []codegen.GeneratedFile

	// Generate package.json
	packageJSON, err := g.generatePackageJSON(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate package.json: %v", err)
	}
	files = append(files, packageJSON)

	// Generate tsconfig.json
	tsconfig, err := g.generateTSConfig(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tsconfig.json: %v", err)
	}
	files = append(files, tsconfig)

	// Generate README.md
	readme := codegen.GeneratedFile{
		Path:    "README.md",
		Content: []byte(fmt.Sprintf("# %s\n\nProtocol Buffer generated code for JavaScript/TypeScript.\n\n## Installation\n\n```bash\nnpm install %s\n```\n", req.ModuleName, convertToNPMPackageName(req.ModuleName))),
		Size:    0,
	}
	readme.Size = int64(len(readme.Content))
	files = append(files, readme)

	return files, nil
}

// GetName returns the name of the package manager
func (g *Generator) GetName() string {
	return "npm"
}

// GetConfigFiles returns the list of config files this generator creates
func (g *Generator) GetConfigFiles() []string {
	return []string{"package.json", "tsconfig.json", "README.md"}
}

func (g *Generator) generatePackageJSON(req *packages.GenerateRequest) (codegen.GeneratedFile, error) {
	packageName := convertToNPMPackageName(req.ModuleName)

	pkg := map[string]interface{}{
		"name":        packageName,
		"version":     req.Version,
		"description": fmt.Sprintf("Protocol Buffer generated code for %s", req.ModuleName),
		"main":        "index.js",
		"types":       "index.d.ts",
		"scripts": map[string]string{
			"build": "tsc",
			"test":  "echo \"No tests specified\"",
		},
		"dependencies": map[string]string{
			"google-protobuf": "^3.21.0",
		},
		"devDependencies": map[string]string{
			"typescript": "^5.0.0",
		},
		"engines": map[string]string{
			"node": ">=16.0.0",
		},
	}

	if req.IncludeGRPC {
		deps := pkg["dependencies"].(map[string]string)
		deps["@grpc/grpc-js"] = "^1.9.0"
		deps["@grpc/proto-loader"] = "^0.7.10"
	}

	// Add dependencies
	for _, dep := range req.Dependencies {
		deps := pkg["dependencies"].(map[string]string)
		deps[dep.Name] = dep.Version
	}

	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return codegen.GeneratedFile{}, err
	}

	return codegen.GeneratedFile{
		Path:    "package.json",
		Content: jsonData,
		Size:    int64(len(jsonData)),
	}, nil
}

func (g *Generator) generateTSConfig(req *packages.GenerateRequest) (codegen.GeneratedFile, error) {
	tsconfig := map[string]interface{}{
		"compilerOptions": map[string]interface{}{
			"target":           "ES2020",
			"module":           "commonjs",
			"declaration":      true,
			"outDir":           "./dist",
			"rootDir":          "./src",
			"strict":           true,
			"esModuleInterop":  true,
			"skipLibCheck":     true,
			"forceConsistentCasingInFileNames": true,
		},
		"include": []string{"src/**/*"},
		"exclude": []string{"node_modules", "dist"},
	}

	jsonData, err := json.MarshalIndent(tsconfig, "", "  ")
	if err != nil {
		return codegen.GeneratedFile{}, err
	}

	return codegen.GeneratedFile{
		Path:    "tsconfig.json",
		Content: jsonData,
		Size:    int64(len(jsonData)),
	}, nil
}

// convertToNPMPackageName converts a module name to a valid npm package name
func convertToNPMPackageName(moduleName string) string {
	// Example: "user-service" -> "@spoke/user-service"
	name := strings.ToLower(moduleName)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return "@spoke/" + name
}
