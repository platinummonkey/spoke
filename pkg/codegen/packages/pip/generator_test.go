package pip

import (
	"strings"
	"testing"

	"github.com/platinummonkey/spoke/pkg/codegen/packages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_GetName(t *testing.T) {
	gen := NewGenerator()
	assert.Equal(t, "pip", gen.GetName())
}

func TestGenerator_GetConfigFiles(t *testing.T) {
	gen := NewGenerator()
	files := gen.GetConfigFiles()
	assert.ElementsMatch(t, []string{"setup.py", "pyproject.toml", "README.md"}, files)
}

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()
	assert.NotNil(t, gen)
	assert.IsType(t, &Generator{}, gen)
}

func TestGenerator_Generate(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "test-service",
		Version:     "1.0.0",
		Language:    "python",
		IncludeGRPC: true,
		Dependencies: []packages.Dependency{
			{Name: "spoke-common", Version: "1.0.0"},
		},
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)
	assert.Len(t, files, 3) // setup.py, pyproject.toml, README.md

	// Verify all expected files are present
	fileMap := make(map[string][]byte)
	for _, f := range files {
		fileMap[f.Path] = f.Content
	}

	assert.Contains(t, fileMap, "setup.py")
	assert.Contains(t, fileMap, "pyproject.toml")
	assert.Contains(t, fileMap, "README.md")
}

func TestGenerator_GenerateSetupPy(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "user-service",
		Version:     "1.2.3",
		Language:    "python",
		IncludeGRPC: true,
		Dependencies: []packages.Dependency{
			{Name: "spoke-common", Version: "1.0.0"},
			{Name: "spoke-auth", Version: "2.1.0"},
		},
	}

	file, err := gen.generateSetupPy(req)
	require.NoError(t, err)
	assert.Equal(t, "setup.py", file.Path)
	assert.Greater(t, file.Size, int64(0))

	content := string(file.Content)

	// Verify basic fields
	assert.Contains(t, content, `name="user_service"`)
	assert.Contains(t, content, `version="1.2.3"`)
	assert.Contains(t, content, "Protocol Buffer generated code for user-service")
	assert.Contains(t, content, "from setuptools import setup, find_packages")

	// Verify protobuf dependency
	assert.Contains(t, content, `"protobuf==4.25.1"`)

	// Verify gRPC dependencies are included
	assert.Contains(t, content, `"grpcio==1.60.0"`)
	assert.Contains(t, content, `"grpcio-tools==1.60.0"`)

	// Verify custom dependencies
	assert.Contains(t, content, `"spoke-common==1.0.0"`)
	assert.Contains(t, content, `"spoke-auth==2.1.0"`)

	// Verify Python version requirement
	assert.Contains(t, content, `python_requires=">=3.8"`)

	// Verify classifiers
	assert.Contains(t, content, "Development Status :: 4 - Beta")
	assert.Contains(t, content, "Programming Language :: Python :: 3")
}

func TestGenerator_GenerateSetupPy_NoGRPC(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "test-service",
		Version:     "1.0.0",
		Language:    "python",
		IncludeGRPC: false,
	}

	file, err := gen.generateSetupPy(req)
	require.NoError(t, err)

	content := string(file.Content)

	// Verify gRPC dependencies are NOT included
	assert.NotContains(t, content, `"grpcio==`)
	assert.NotContains(t, content, `"grpcio-tools==`)

	// Verify base dependencies are still included
	assert.Contains(t, content, `"protobuf==4.25.1"`)
}

func TestGenerator_GeneratePyproject(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "user-service",
		Version:     "1.2.3",
		Language:    "python",
		IncludeGRPC: true,
		Dependencies: []packages.Dependency{
			{Name: "spoke-common", Version: "1.0.0"},
			{Name: "spoke-auth", Version: "2.1.0"},
		},
	}

	file, err := gen.generatePyproject(req)
	require.NoError(t, err)
	assert.Equal(t, "pyproject.toml", file.Path)
	assert.Greater(t, file.Size, int64(0))

	content := string(file.Content)

	// Verify build system
	assert.Contains(t, content, "[build-system]")
	assert.Contains(t, content, `requires = ["setuptools>=61.0", "wheel"]`)
	assert.Contains(t, content, `build-backend = "setuptools.build_meta"`)

	// Verify project metadata
	assert.Contains(t, content, "[project]")
	assert.Contains(t, content, `name = "user_service"`)
	assert.Contains(t, content, `version = "1.2.3"`)
	assert.Contains(t, content, "Protocol Buffer generated code for user-service")
	assert.Contains(t, content, `readme = "README.md"`)
	assert.Contains(t, content, `requires-python = ">=3.8"`)

	// Verify dependencies
	assert.Contains(t, content, `"protobuf==4.25.1"`)
	assert.Contains(t, content, `"grpcio==1.60.0"`)
	assert.Contains(t, content, `"grpcio-tools==1.60.0"`)
	assert.Contains(t, content, `"spoke-common==1.0.0"`)
	assert.Contains(t, content, `"spoke-auth==2.1.0"`)

	// Verify setuptools configuration
	assert.Contains(t, content, "[tool.setuptools]")
	assert.Contains(t, content, `packages = ["user_service"]`)
}

func TestGenerator_GeneratePyproject_NoGRPC(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "test-service",
		Version:     "1.0.0",
		Language:    "python",
		IncludeGRPC: false,
	}

	file, err := gen.generatePyproject(req)
	require.NoError(t, err)

	content := string(file.Content)

	// Verify gRPC dependencies are NOT included
	assert.NotContains(t, content, `"grpcio==`)
	assert.NotContains(t, content, `"grpcio-tools==`)

	// Verify base dependencies are still included
	assert.Contains(t, content, `"protobuf==4.25.1"`)
}

func TestGenerator_GenerateREADME(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName: "user-service",
		Version:    "1.0.0",
		Language:   "python",
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
	assert.Contains(t, *readme, "Protocol Buffer generated code for Python")
	assert.Contains(t, *readme, "pip install user_service")
	assert.Contains(t, *readme, "## Installation")
}

func TestConvertToPythonPackageName(t *testing.T) {
	tests := []struct {
		name       string
		moduleName string
		expected   string
	}{
		{
			name:       "simple lowercase",
			moduleName: "userservice",
			expected:   "userservice",
		},
		{
			name:       "with hyphen",
			moduleName: "user-service",
			expected:   "user_service",
		},
		{
			name:       "with uppercase",
			moduleName: "UserService",
			expected:   "userservice",
		},
		{
			name:       "with underscores",
			moduleName: "user_service",
			expected:   "user_service",
		},
		{
			name:       "with spaces",
			moduleName: "user service",
			expected:   "user_service",
		},
		{
			name:       "mixed case and special chars",
			moduleName: "User-Service API",
			expected:   "user_service_api",
		},
		{
			name:       "multiple hyphens",
			moduleName: "my-cool-service",
			expected:   "my_cool_service",
		},
		{
			name:       "multiple spaces",
			moduleName: "my cool service",
			expected:   "my_cool_service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToPythonPackageName(tt.moduleName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertDependencies(t *testing.T) {
	tests := []struct {
		name     string
		deps     []packages.Dependency
		expected []map[string]string
	}{
		{
			name: "single dependency",
			deps: []packages.Dependency{
				{Name: "spoke-common", Version: "1.0.0"},
			},
			expected: []map[string]string{
				{"Name": "spoke-common", "Version": "1.0.0"},
			},
		},
		{
			name: "multiple dependencies",
			deps: []packages.Dependency{
				{Name: "spoke-common", Version: "1.0.0"},
				{Name: "spoke-auth", Version: "2.1.0"},
			},
			expected: []map[string]string{
				{"Name": "spoke-common", "Version": "1.0.0"},
				{"Name": "spoke-auth", "Version": "2.1.0"},
			},
		},
		{
			name:     "empty dependencies",
			deps:     []packages.Dependency{},
			expected: []map[string]string{},
		},
		{
			name: "dependency with import path",
			deps: []packages.Dependency{
				{Name: "spoke-common", Version: "1.0.0", ImportPath: "github.com/spoke/common"},
			},
			expected: []map[string]string{
				{"Name": "spoke-common", "Version": "1.0.0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertDependencies(tt.deps)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerator_Generate_EmptyDependencies(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:   "simple-service",
		Version:      "0.1.0",
		Language:     "python",
		IncludeGRPC:  false,
		Dependencies: []packages.Dependency{},
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)
	assert.Len(t, files, 3)

	// Verify setup.py doesn't break with empty dependencies
	var setupPy string
	for _, f := range files {
		if f.Path == "setup.py" {
			setupPy = string(f.Content)
			break
		}
	}

	assert.Contains(t, setupPy, `name="simple_service"`)
	assert.Contains(t, setupPy, `version="0.1.0"`)
}

func TestGenerator_Generate_FileSize(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "test-service",
		Version:     "1.0.0",
		Language:    "python",
		IncludeGRPC: true,
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)

	// Verify all files have correct size
	for _, f := range files {
		assert.Equal(t, int64(len(f.Content)), f.Size, "File %s has incorrect size", f.Path)
		assert.Greater(t, f.Size, int64(0), "File %s should not be empty", f.Path)
	}
}

func TestGenerator_GenerateSetupPy_ContentStructure(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:   "my-api",
		Version:      "2.0.0",
		Language:     "python",
		IncludeGRPC:  true,
		Dependencies: []packages.Dependency{},
	}

	file, err := gen.generateSetupPy(req)
	require.NoError(t, err)

	content := string(file.Content)

	// Verify structure and order
	setupIndex := strings.Index(content, "setup(")
	nameIndex := strings.Index(content, `name="my_api"`)
	versionIndex := strings.Index(content, `version="2.0.0"`)
	installRequiresIndex := strings.Index(content, "install_requires=[")

	assert.Greater(t, setupIndex, -1, "should contain setup(")
	assert.Greater(t, nameIndex, setupIndex, "name should come after setup(")
	assert.Greater(t, versionIndex, setupIndex, "version should come after setup(")
	assert.Greater(t, installRequiresIndex, setupIndex, "install_requires should come after setup(")
}

func TestGenerator_GeneratePyproject_ContentStructure(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:   "my-api",
		Version:      "2.0.0",
		Language:     "python",
		IncludeGRPC:  false,
		Dependencies: []packages.Dependency{},
	}

	file, err := gen.generatePyproject(req)
	require.NoError(t, err)

	content := string(file.Content)

	// Verify structure sections are present in order
	buildSystemIndex := strings.Index(content, "[build-system]")
	projectIndex := strings.Index(content, "[project]")
	toolIndex := strings.Index(content, "[tool.setuptools]")

	assert.Greater(t, buildSystemIndex, -1, "should contain [build-system]")
	assert.Greater(t, projectIndex, buildSystemIndex, "[project] should come after [build-system]")
	assert.Greater(t, toolIndex, projectIndex, "[tool.setuptools] should come after [project]")
}
