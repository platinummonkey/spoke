package npm

import (
	"encoding/json"
	"testing"

	"github.com/platinummonkey/spoke/pkg/codegen/packages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_GetName(t *testing.T) {
	gen := NewGenerator()
	assert.Equal(t, "npm", gen.GetName())
}

func TestGenerator_GetConfigFiles(t *testing.T) {
	gen := NewGenerator()
	files := gen.GetConfigFiles()
	assert.ElementsMatch(t, []string{"package.json", "tsconfig.json", "README.md"}, files)
}

func TestGenerator_Generate(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "test-service",
		Version:     "1.0.0",
		Language:    "javascript",
		IncludeGRPC: true,
		Dependencies: []packages.Dependency{
			{Name: "@spoke/common", Version: "^1.0.0"},
		},
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)
	assert.Len(t, files, 3) // package.json, tsconfig.json, README.md

	// Verify all expected files are present
	fileMap := make(map[string][]byte)
	for _, f := range files {
		fileMap[f.Path] = f.Content
	}

	assert.Contains(t, fileMap, "package.json")
	assert.Contains(t, fileMap, "tsconfig.json")
	assert.Contains(t, fileMap, "README.md")
}

func TestGenerator_GeneratePackageJSON(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "user-service",
		Version:     "1.2.3",
		Language:    "javascript",
		IncludeGRPC: true,
		Dependencies: []packages.Dependency{
			{Name: "@spoke/common", Version: "^1.0.0"},
			{Name: "@spoke/auth", Version: "^2.1.0"},
		},
	}

	file, err := gen.generatePackageJSON(req)
	require.NoError(t, err)
	assert.Equal(t, "package.json", file.Path)
	assert.Greater(t, file.Size, int64(0))

	// Parse the JSON to verify structure
	var pkg map[string]interface{}
	err = json.Unmarshal(file.Content, &pkg)
	require.NoError(t, err)

	// Verify basic fields
	assert.Equal(t, "@spoke/user-service", pkg["name"])
	assert.Equal(t, "1.2.3", pkg["version"])
	assert.Contains(t, pkg["description"], "user-service")
	assert.Equal(t, "index.js", pkg["main"])
	assert.Equal(t, "index.d.ts", pkg["types"])

	// Verify dependencies
	deps := pkg["dependencies"].(map[string]interface{})
	assert.Equal(t, "^3.21.0", deps["google-protobuf"])
	assert.Equal(t, "^1.9.0", deps["@grpc/grpc-js"])
	assert.Equal(t, "^0.7.10", deps["@grpc/proto-loader"])
	assert.Equal(t, "^1.0.0", deps["@spoke/common"])
	assert.Equal(t, "^2.1.0", deps["@spoke/auth"])

	// Verify devDependencies
	devDeps := pkg["devDependencies"].(map[string]interface{})
	assert.Equal(t, "^5.0.0", devDeps["typescript"])

	// Verify scripts
	scripts := pkg["scripts"].(map[string]interface{})
	assert.Equal(t, "tsc", scripts["build"])

	// Verify engines
	engines := pkg["engines"].(map[string]interface{})
	assert.Equal(t, ">=16.0.0", engines["node"])
}

func TestGenerator_GeneratePackageJSON_NoGRPC(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "test-service",
		Version:     "1.0.0",
		Language:    "javascript",
		IncludeGRPC: false,
	}

	file, err := gen.generatePackageJSON(req)
	require.NoError(t, err)

	var pkg map[string]interface{}
	err = json.Unmarshal(file.Content, &pkg)
	require.NoError(t, err)

	// Verify gRPC dependencies are NOT included
	deps := pkg["dependencies"].(map[string]interface{})
	assert.NotContains(t, deps, "@grpc/grpc-js")
	assert.NotContains(t, deps, "@grpc/proto-loader")

	// Verify base dependencies are still included
	assert.Contains(t, deps, "google-protobuf")
}

func TestGenerator_GenerateTSConfig(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName: "test-service",
		Version:    "1.0.0",
		Language:   "javascript",
	}

	file, err := gen.generateTSConfig(req)
	require.NoError(t, err)
	assert.Equal(t, "tsconfig.json", file.Path)
	assert.Greater(t, file.Size, int64(0))

	// Parse the JSON to verify structure
	var tsconfig map[string]interface{}
	err = json.Unmarshal(file.Content, &tsconfig)
	require.NoError(t, err)

	// Verify compiler options
	compilerOptions := tsconfig["compilerOptions"].(map[string]interface{})
	assert.Equal(t, "ES2020", compilerOptions["target"])
	assert.Equal(t, "commonjs", compilerOptions["module"])
	assert.Equal(t, true, compilerOptions["declaration"])
	assert.Equal(t, "./dist", compilerOptions["outDir"])
	assert.Equal(t, "./src", compilerOptions["rootDir"])
	assert.Equal(t, true, compilerOptions["strict"])
	assert.Equal(t, true, compilerOptions["esModuleInterop"])
	assert.Equal(t, true, compilerOptions["skipLibCheck"])
	assert.Equal(t, true, compilerOptions["forceConsistentCasingInFileNames"])

	// Verify include/exclude
	include := tsconfig["include"].([]interface{})
	assert.Contains(t, include, "src/**/*")

	exclude := tsconfig["exclude"].([]interface{})
	assert.Contains(t, exclude, "node_modules")
	assert.Contains(t, exclude, "dist")
}

func TestGenerator_GenerateREADME(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName: "user-service",
		Version:    "1.0.0",
		Language:   "javascript",
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)

	// Find README.md
	var readme *string
	for _, f := range files {
		if f.Path == "README.md" {
			content := string(f.Content)
			readme = &content
			break
		}
	}

	require.NotNil(t, readme)
	assert.Contains(t, *readme, "# user-service")
	assert.Contains(t, *readme, "Protocol Buffer generated code")
	assert.Contains(t, *readme, "npm install @spoke/user-service")
	assert.Contains(t, *readme, "## Installation")
}

func TestConvertToNPMPackageName(t *testing.T) {
	tests := []struct {
		name       string
		moduleName string
		expected   string
	}{
		{
			name:       "simple lowercase",
			moduleName: "user-service",
			expected:   "@spoke/user-service",
		},
		{
			name:       "with uppercase",
			moduleName: "UserService",
			expected:   "@spoke/userservice",
		},
		{
			name:       "with underscores",
			moduleName: "user_service",
			expected:   "@spoke/user-service",
		},
		{
			name:       "with spaces",
			moduleName: "user service",
			expected:   "@spoke/user-service",
		},
		{
			name:       "mixed case and special chars",
			moduleName: "User_Service API",
			expected:   "@spoke/user-service-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToNPMPackageName(tt.moduleName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()
	assert.NotNil(t, gen)
	assert.IsType(t, &Generator{}, gen)
}
