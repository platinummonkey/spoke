package gomod

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/codegen/packages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_GetName(t *testing.T) {
	gen := NewGenerator()
	assert.Equal(t, "go-modules", gen.GetName())
}

func TestGenerator_GetConfigFiles(t *testing.T) {
	gen := NewGenerator()
	files := gen.GetConfigFiles()
	assert.ElementsMatch(t, []string{"go.mod", "README.md"}, files)
}

func TestGenerator_Generate(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName: "test-service",
		Version:    "v1.0.0",
		Language:   "go",
		IncludeGRPC: true,
		Dependencies: []packages.Dependency{
			{Name: "common", Version: "v1.0.0", ImportPath: "github.com/spoke/common"},
		},
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)
	assert.Len(t, files, 2)

	// Find go.mod file
	var goMod *string
	for _, f := range files {
		if f.Path == "go.mod" {
			content := string(f.Content)
			goMod = &content
			break
		}
	}

	require.NotNil(t, goMod)
	assert.Contains(t, *goMod, "module github.com/spoke-generated/test-service")
	assert.Contains(t, *goMod, "google.golang.org/protobuf")
	assert.Contains(t, *goMod, "google.golang.org/grpc")
	assert.Contains(t, *goMod, "github.com/spoke/common v1.0.0")
}

func TestGenerator_Generate_NoGRPC(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "test-service",
		Version:     "v1.0.0",
		Language:    "go",
		IncludeGRPC: false,
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)

	// Find go.mod file
	var goMod *string
	for _, f := range files {
		if f.Path == "go.mod" {
			content := string(f.Content)
			goMod = &content
			break
		}
	}

	require.NotNil(t, goMod)
	assert.NotContains(t, *goMod, "google.golang.org/grpc")
}

func TestConvertToGoModulePath(t *testing.T) {
	tests := []struct {
		name        string
		moduleName  string
		version     string
		expected    string
	}{
		{
			name:       "simple name",
			moduleName: "user-service",
			version:    "v1.0.0",
			expected:   "github.com/spoke-generated/user-service",
		},
		{
			name:       "with underscores",
			moduleName: "user_service",
			version:    "v1.0.0",
			expected:   "github.com/spoke-generated/user-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToGoModulePath(tt.moduleName, tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}
