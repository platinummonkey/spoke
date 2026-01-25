package api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServer_GetCodeGenVersion(t *testing.T) {
	s := &Server{}

	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "default to v2",
			envValue: "",
			expected: "v2",
		},
		{
			name:     "explicit v1",
			envValue: "v1",
			expected: "v1",
		},
		{
			name:     "explicit v2",
			envValue: "v2",
			expected: "v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv("SPOKE_CODEGEN_VERSION", tt.envValue)
			} else {
				os.Unsetenv("SPOKE_CODEGEN_VERSION")
			}
			defer os.Unsetenv("SPOKE_CODEGEN_VERSION")

			result := s.getCodeGenVersion()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServer_CompileForLanguage_FeatureFlag(t *testing.T) {
	// Skip this test - requires actual storage implementation
	// This is an integration test that would need a real storage backend
	t.Skip("Requires storage implementation for full integration testing")
}

func TestServer_CompileV1_Routing(t *testing.T) {
	s := &Server{}

	version := &Version{
		ModuleName: "test",
		Version:    "v1.0.0",
		Files: []File{
			{
				Path:    "test.proto",
				Content: "syntax = \"proto3\";",
			},
		},
	}

	// Test unsupported language
	_, err := s.compileV1(version, Language("unsupported"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported language")
}

func TestServer_OrchestratorInitialization(t *testing.T) {
	// Skip - requires actual storage implementation
	t.Skip("Requires storage implementation for full integration testing")
}

func TestServer_RegisterPackageGenerators(t *testing.T) {
	s := &Server{}

	// This is currently a placeholder method
	// Should not panic
	s.registerPackageGenerators()

	// Test passes if no panic occurs
	assert.NotNil(t, s)
}

func TestServer_CompileWithOrchestrator_NoOrchestrator(t *testing.T) {
	s := &Server{}

	// Orchestrator is nil (not initialized)
	assert.Nil(t, s.orchestrator)

	version := &Version{
		ModuleName: "test",
		Version:    "v1.0.0",
		Files: []File{
			{
				Path:    "test.proto",
				Content: "syntax = \"proto3\";",
			},
		},
	}

	// Should return error when orchestrator not available
	_, err := s.compileWithOrchestrator(version, LanguageGo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "orchestrator not available")
}
