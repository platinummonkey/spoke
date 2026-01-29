package plugins

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestLanguageSpec_JSONMarshaling tests JSON marshaling and unmarshaling of LanguageSpec
func TestLanguageSpec_JSONMarshaling(t *testing.T) {
	spec := &LanguageSpec{
		ID:               "go",
		Name:             "Go",
		DisplayName:      "Go Language",
		SupportsGRPC:     true,
		FileExtensions:   []string{".go", ".pb.go"},
		Enabled:          true,
		Stable:           true,
		Description:      "Go language plugin for protobuf",
		DocumentationURL: "https://golang.org/protobuf",
		PluginVersion:    "1.28.0",
		ProtocPlugin:     "go",
		DockerImage:      "golang:1.21",
		PackageManager: &PackageManager{
			Name:            "go-modules",
			ConfigFiles:     []string{"go.mod", "go.sum"},
			DependencyMap:   map[string]string{"protobuf": "google.golang.org/protobuf"},
			DefaultVersions: map[string]string{"protobuf": "v1.28.0"},
		},
		CustomOptions: map[string]string{
			"paths":        "source_relative",
			"require_unimplemented_servers": "false",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(spec)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var decoded LanguageSpec
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, spec.ID, decoded.ID)
	assert.Equal(t, spec.Name, decoded.Name)
	assert.Equal(t, spec.DisplayName, decoded.DisplayName)
	assert.Equal(t, spec.SupportsGRPC, decoded.SupportsGRPC)
	assert.Equal(t, spec.FileExtensions, decoded.FileExtensions)
	assert.Equal(t, spec.Enabled, decoded.Enabled)
	assert.Equal(t, spec.Stable, decoded.Stable)
	assert.Equal(t, spec.Description, decoded.Description)
	assert.Equal(t, spec.DocumentationURL, decoded.DocumentationURL)
	assert.Equal(t, spec.PluginVersion, decoded.PluginVersion)
	assert.Equal(t, spec.ProtocPlugin, decoded.ProtocPlugin)
	assert.Equal(t, spec.DockerImage, decoded.DockerImage)
	assert.NotNil(t, decoded.PackageManager)
	assert.Equal(t, spec.PackageManager.Name, decoded.PackageManager.Name)
	assert.Equal(t, spec.CustomOptions, decoded.CustomOptions)
}

// TestLanguageSpec_YAMLMarshaling tests YAML marshaling and unmarshaling of LanguageSpec
func TestLanguageSpec_YAMLMarshaling(t *testing.T) {
	spec := &LanguageSpec{
		ID:               "rust",
		Name:             "Rust",
		DisplayName:      "Rust Language",
		SupportsGRPC:     true,
		FileExtensions:   []string{".rs"},
		Enabled:          true,
		Stable:           false,
		Description:      "Rust language plugin",
		DocumentationURL: "https://rust-lang.org",
		PluginVersion:    "0.1.0",
		ProtocPlugin:     "rust",
		DockerImage:      "rust:1.70",
		CustomOptions: map[string]string{
			"edition": "2021",
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(spec)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal from YAML
	var decoded LanguageSpec
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, spec.ID, decoded.ID)
	assert.Equal(t, spec.Name, decoded.Name)
	assert.Equal(t, spec.DisplayName, decoded.DisplayName)
	assert.Equal(t, spec.SupportsGRPC, decoded.SupportsGRPC)
	assert.Equal(t, spec.FileExtensions, decoded.FileExtensions)
	assert.Equal(t, spec.Enabled, decoded.Enabled)
	assert.Equal(t, spec.Stable, decoded.Stable)
	assert.Equal(t, spec.CustomOptions, decoded.CustomOptions)
}

// TestLanguageSpec_WithNilPackageManager tests LanguageSpec with nil PackageManager
func TestLanguageSpec_WithNilPackageManager(t *testing.T) {
	spec := &LanguageSpec{
		ID:             "python",
		Name:           "Python",
		DisplayName:    "Python 3",
		SupportsGRPC:   true,
		FileExtensions: []string{".py"},
		Enabled:        true,
		Stable:         true,
		PackageManager: nil, // Explicitly nil
	}

	// Marshal to JSON
	data, err := json.Marshal(spec)
	require.NoError(t, err)

	// Unmarshal from JSON
	var decoded LanguageSpec
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.PackageManager)
	assert.Equal(t, spec.ID, decoded.ID)
}

// TestLanguageSpec_WithNilCustomOptions tests LanguageSpec with nil CustomOptions
func TestLanguageSpec_WithNilCustomOptions(t *testing.T) {
	spec := &LanguageSpec{
		ID:            "cpp",
		Name:          "C++",
		DisplayName:   "C++17",
		CustomOptions: nil, // Explicitly nil
	}

	// Marshal to JSON
	data, err := json.Marshal(spec)
	require.NoError(t, err)

	// Unmarshal from JSON
	var decoded LanguageSpec
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.CustomOptions)
	assert.Equal(t, spec.Name, decoded.Name)
}

// TestLanguageSpec_EmptyFields tests LanguageSpec with empty fields
func TestLanguageSpec_EmptyFields(t *testing.T) {
	spec := &LanguageSpec{
		ID:             "",
		Name:           "",
		FileExtensions: []string{},
		CustomOptions:  map[string]string{},
	}

	// Marshal to JSON
	data, err := json.Marshal(spec)
	require.NoError(t, err)

	// Unmarshal from JSON
	var decoded LanguageSpec
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "", decoded.ID)
	assert.Equal(t, "", decoded.Name)
	// Note: JSON unmarshaling creates empty slices/maps as non-nil
	assert.Equal(t, 0, len(decoded.FileExtensions))
	assert.Equal(t, 0, len(decoded.CustomOptions))
}

// TestPackageManager_JSONMarshaling tests JSON marshaling of PackageManager
func TestPackageManager_JSONMarshaling(t *testing.T) {
	pm := &PackageManager{
		Name:        "npm",
		ConfigFiles: []string{"package.json", "package-lock.json"},
		DependencyMap: map[string]string{
			"protobuf": "@grpc/grpc-js",
			"types":    "@types/google-protobuf",
		},
		DefaultVersions: map[string]string{
			"protobuf": "^1.9.0",
			"types":    "^3.15.0",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(pm)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var decoded PackageManager
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, pm.Name, decoded.Name)
	assert.Equal(t, pm.ConfigFiles, decoded.ConfigFiles)
	assert.Equal(t, pm.DependencyMap, decoded.DependencyMap)
	assert.Equal(t, pm.DefaultVersions, decoded.DefaultVersions)
}

// TestPackageManager_YAMLMarshaling tests YAML marshaling of PackageManager
func TestPackageManager_YAMLMarshaling(t *testing.T) {
	pm := &PackageManager{
		Name:        "cargo",
		ConfigFiles: []string{"Cargo.toml", "Cargo.lock"},
		DependencyMap: map[string]string{
			"protobuf": "prost",
		},
		DefaultVersions: map[string]string{
			"protobuf": "0.11.0",
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(pm)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal from YAML
	var decoded PackageManager
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, pm.Name, decoded.Name)
	assert.Equal(t, pm.ConfigFiles, decoded.ConfigFiles)
	assert.Equal(t, pm.DependencyMap, decoded.DependencyMap)
	assert.Equal(t, pm.DefaultVersions, decoded.DefaultVersions)
}

// TestPackageManager_EmptyMaps tests PackageManager with empty maps
func TestPackageManager_EmptyMaps(t *testing.T) {
	pm := &PackageManager{
		Name:            "test",
		ConfigFiles:     []string{"config.json"},
		DependencyMap:   map[string]string{},
		DefaultVersions: map[string]string{},
	}

	data, err := json.Marshal(pm)
	require.NoError(t, err)

	var decoded PackageManager
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, pm.Name, decoded.Name)
	assert.NotNil(t, decoded.DependencyMap)
	assert.Equal(t, 0, len(decoded.DependencyMap))
	assert.NotNil(t, decoded.DefaultVersions)
	assert.Equal(t, 0, len(decoded.DefaultVersions))
}

// TestCommandRequest_JSONMarshaling tests JSON marshaling of CommandRequest
func TestCommandRequest_JSONMarshaling(t *testing.T) {
	req := &CommandRequest{
		ProtoFiles:  []string{"service.proto", "types.proto"},
		ImportPaths: []string{"/proto", "/include"},
		OutputDir:   "/output",
		Options: map[string]string{
			"paths":        "source_relative",
			"go_package":   "./gen/go",
		},
		PluginPath: "/usr/local/bin/protoc-gen-go",
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var decoded CommandRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.ProtoFiles, decoded.ProtoFiles)
	assert.Equal(t, req.ImportPaths, decoded.ImportPaths)
	assert.Equal(t, req.OutputDir, decoded.OutputDir)
	assert.Equal(t, req.Options, decoded.Options)
	assert.Equal(t, req.PluginPath, decoded.PluginPath)
}

// TestCommandRequest_EmptyPluginPath tests CommandRequest with empty PluginPath
func TestCommandRequest_EmptyPluginPath(t *testing.T) {
	req := &CommandRequest{
		ProtoFiles:  []string{"service.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		Options:     map[string]string{},
		PluginPath:  "", // Empty plugin path
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded CommandRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "", decoded.PluginPath)
	assert.Equal(t, req.ProtoFiles, decoded.ProtoFiles)
}

// TestCommandRequest_NilOptions tests CommandRequest with nil Options
func TestCommandRequest_NilOptions(t *testing.T) {
	req := &CommandRequest{
		ProtoFiles:  []string{"service.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		Options:     nil,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded CommandRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.Options)
}

// TestCommandRequest_EmptyArrays tests CommandRequest with empty arrays
func TestCommandRequest_EmptyArrays(t *testing.T) {
	req := &CommandRequest{
		ProtoFiles:  []string{},
		ImportPaths: []string{},
		OutputDir:   "/output",
		Options:     map[string]string{},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded CommandRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.NotNil(t, decoded.ProtoFiles)
	assert.Equal(t, 0, len(decoded.ProtoFiles))
	assert.NotNil(t, decoded.ImportPaths)
	assert.Equal(t, 0, len(decoded.ImportPaths))
}

// TestCommandResult_JSONMarshaling tests JSON marshaling of CommandResult
func TestCommandResult_JSONMarshaling(t *testing.T) {
	result := &CommandResult{
		ExitCode:       0,
		Stdout:         "Compilation successful",
		Stderr:         "",
		GeneratedFiles: []string{"service.pb.go", "types.pb.go"},
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var decoded CommandResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.ExitCode, decoded.ExitCode)
	assert.Equal(t, result.Stdout, decoded.Stdout)
	assert.Equal(t, result.Stderr, decoded.Stderr)
	assert.Equal(t, result.GeneratedFiles, decoded.GeneratedFiles)
}

// TestCommandResult_WithError tests CommandResult with error state
func TestCommandResult_WithError(t *testing.T) {
	result := &CommandResult{
		ExitCode:       1,
		Stdout:         "",
		Stderr:         "protoc: error: file not found",
		GeneratedFiles: []string{},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded CommandResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 1, decoded.ExitCode)
	assert.NotEmpty(t, decoded.Stderr)
	assert.Empty(t, decoded.Stdout)
	assert.NotNil(t, decoded.GeneratedFiles)
	assert.Equal(t, 0, len(decoded.GeneratedFiles))
}

// TestCommandResult_EmptyFields tests CommandResult with empty fields
func TestCommandResult_EmptyFields(t *testing.T) {
	result := &CommandResult{
		ExitCode:       0,
		Stdout:         "",
		Stderr:         "",
		GeneratedFiles: nil,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded CommandResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 0, decoded.ExitCode)
	assert.Equal(t, "", decoded.Stdout)
	assert.Equal(t, "", decoded.Stderr)
	assert.Nil(t, decoded.GeneratedFiles)
}

// TestCommandResult_LargeOutput tests CommandResult with large output
func TestCommandResult_LargeOutput(t *testing.T) {
	largeStdout := string(make([]byte, 10000))
	result := &CommandResult{
		ExitCode:       0,
		Stdout:         largeStdout,
		Stderr:         "",
		GeneratedFiles: make([]string, 100),
	}

	for i := 0; i < 100; i++ {
		result.GeneratedFiles[i] = "file" + string(rune(i)) + ".pb.go"
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded CommandResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, len(result.Stdout), len(decoded.Stdout))
	assert.Equal(t, len(result.GeneratedFiles), len(decoded.GeneratedFiles))
}

// TestLanguageSpec_MultipleFileExtensions tests LanguageSpec with multiple file extensions
func TestLanguageSpec_MultipleFileExtensions(t *testing.T) {
	spec := &LanguageSpec{
		ID:             "cpp",
		Name:           "C++",
		FileExtensions: []string{".h", ".cc", ".cpp", ".hpp"},
	}

	data, err := json.Marshal(spec)
	require.NoError(t, err)

	var decoded LanguageSpec
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 4, len(decoded.FileExtensions))
	assert.Contains(t, decoded.FileExtensions, ".h")
	assert.Contains(t, decoded.FileExtensions, ".cc")
	assert.Contains(t, decoded.FileExtensions, ".cpp")
	assert.Contains(t, decoded.FileExtensions, ".hpp")
}

// TestLanguageSpec_BooleanFields tests LanguageSpec boolean fields
func TestLanguageSpec_BooleanFields(t *testing.T) {
	testCases := []struct {
		name         string
		supportsGRPC bool
		enabled      bool
		stable       bool
	}{
		{"all true", true, true, true},
		{"all false", false, false, false},
		{"mixed", true, false, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spec := &LanguageSpec{
				ID:           "test",
				Name:         "Test",
				SupportsGRPC: tc.supportsGRPC,
				Enabled:      tc.enabled,
				Stable:       tc.stable,
			}

			data, err := json.Marshal(spec)
			require.NoError(t, err)

			var decoded LanguageSpec
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tc.supportsGRPC, decoded.SupportsGRPC)
			assert.Equal(t, tc.enabled, decoded.Enabled)
			assert.Equal(t, tc.stable, decoded.Stable)
		})
	}
}

// TestCommandRequest_WithComplexOptions tests CommandRequest with complex options
func TestCommandRequest_WithComplexOptions(t *testing.T) {
	req := &CommandRequest{
		ProtoFiles:  []string{"api/v1/service.proto", "api/v2/service.proto"},
		ImportPaths: []string{"/proto", "/include", "/third_party"},
		OutputDir:   "/output/gen",
		Options: map[string]string{
			"paths":                         "source_relative",
			"go_package":                    "github.com/example/gen/go",
			"require_unimplemented_servers": "false",
			"annotate_code":                 "true",
			"module":                        "github.com/example/api",
		},
		PluginPath: "/usr/local/bin/protoc-gen-go-grpc",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded CommandRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 5, len(decoded.Options))
	assert.Equal(t, "source_relative", decoded.Options["paths"])
	assert.Equal(t, "github.com/example/gen/go", decoded.Options["go_package"])
}

// TestPackageManager_NilMaps tests PackageManager with nil maps
func TestPackageManager_NilMaps(t *testing.T) {
	pm := &PackageManager{
		Name:            "test",
		ConfigFiles:     []string{"test.json"},
		DependencyMap:   nil,
		DefaultVersions: nil,
	}

	data, err := json.Marshal(pm)
	require.NoError(t, err)

	var decoded PackageManager
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.DependencyMap)
	assert.Nil(t, decoded.DefaultVersions)
}

// TestLanguageSpec_CompleteExample tests a complete, realistic LanguageSpec
func TestLanguageSpec_CompleteExample(t *testing.T) {
	spec := &LanguageSpec{
		ID:               "go-grpc",
		Name:             "Go gRPC",
		DisplayName:      "Go with gRPC Support",
		SupportsGRPC:     true,
		FileExtensions:   []string{".go", ".pb.go", "_grpc.pb.go"},
		Enabled:          true,
		Stable:           true,
		Description:      "Go language plugin with full gRPC support",
		DocumentationURL: "https://grpc.io/docs/languages/go/",
		PluginVersion:    "1.3.0",
		ProtocPlugin:     "go-grpc",
		DockerImage:      "golang:1.21-alpine",
		PackageManager: &PackageManager{
			Name:        "go-modules",
			ConfigFiles: []string{"go.mod", "go.sum"},
			DependencyMap: map[string]string{
				"protobuf":      "google.golang.org/protobuf",
				"grpc":          "google.golang.org/grpc",
				"grpc-gateway":  "github.com/grpc-ecosystem/grpc-gateway/v2",
			},
			DefaultVersions: map[string]string{
				"protobuf":     "v1.31.0",
				"grpc":         "v1.58.0",
				"grpc-gateway": "v2.18.0",
			},
		},
		CustomOptions: map[string]string{
			"paths":                         "source_relative",
			"require_unimplemented_servers": "false",
			"module":                        "github.com/example/api",
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(spec)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var jsonDecoded LanguageSpec
	err = json.Unmarshal(jsonData, &jsonDecoded)
	require.NoError(t, err)
	assert.Equal(t, spec.ID, jsonDecoded.ID)
	assert.Equal(t, spec.SupportsGRPC, jsonDecoded.SupportsGRPC)
	assert.NotNil(t, jsonDecoded.PackageManager)
	assert.Equal(t, 3, len(jsonDecoded.PackageManager.DependencyMap))

	// Test YAML marshaling
	yamlData, err := yaml.Marshal(spec)
	require.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	var yamlDecoded LanguageSpec
	err = yaml.Unmarshal(yamlData, &yamlDecoded)
	require.NoError(t, err)
	assert.Equal(t, spec.ID, yamlDecoded.ID)
	assert.Equal(t, spec.SupportsGRPC, yamlDecoded.SupportsGRPC)
	assert.NotNil(t, yamlDecoded.PackageManager)
}

// testLanguagePlugin is a mock implementation of LanguagePlugin for testing
type testLanguagePlugin struct {
	manifest      *Manifest
	languageSpec  *LanguageSpec
	loadError     error
	buildError    error
	validateError error
}

func (m *testLanguagePlugin) Manifest() *Manifest {
	return m.manifest
}

func (m *testLanguagePlugin) Load() error {
	return m.loadError
}

func (m *testLanguagePlugin) Unload() error {
	return nil
}

func (m *testLanguagePlugin) GetLanguageSpec() *LanguageSpec {
	return m.languageSpec
}

func (m *testLanguagePlugin) BuildProtocCommand(ctx context.Context, req *CommandRequest) ([]string, error) {
	if m.buildError != nil {
		return nil, m.buildError
	}
	return []string{"protoc", "--go_out=" + req.OutputDir}, nil
}

func (m *testLanguagePlugin) ValidateOutput(ctx context.Context, files []string) error {
	return m.validateError
}

// TestLanguagePlugin_InterfaceCompliance tests that implementations satisfy the interface
func TestLanguagePlugin_InterfaceCompliance(t *testing.T) {
	manifest := &Manifest{
		ID:      "mock-plugin",
		Name:    "Mock Plugin",
		Version: "1.0.0",
		Type:    PluginTypeLanguage,
	}

	languageSpec := &LanguageSpec{
		ID:             "mock",
		Name:           "Mock Language",
		FileExtensions: []string{".mock"},
	}

	mock := &testLanguagePlugin{
		manifest:     manifest,
		languageSpec: languageSpec,
	}

	// Verify it implements Plugin interface
	var _ Plugin = mock

	// Verify it implements LanguagePlugin interface
	var _ LanguagePlugin = mock

	// Test interface methods
	assert.Equal(t, manifest, mock.Manifest())
	assert.NoError(t, mock.Load())
	assert.NoError(t, mock.Unload())
	assert.Equal(t, languageSpec, mock.GetLanguageSpec())

	ctx := context.Background()
	req := &CommandRequest{
		ProtoFiles:  []string{"test.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
	}

	cmd, err := mock.BuildProtocCommand(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, cmd)

	err = mock.ValidateOutput(ctx, []string{"test.go"})
	assert.NoError(t, err)
}

// TestLanguagePlugin_InterfaceMethods tests interface method signatures
func TestLanguagePlugin_InterfaceMethods(t *testing.T) {
	mock := &testLanguagePlugin{
		manifest: &Manifest{
			ID:      "test",
			Name:    "Test",
			Version: "1.0.0",
		},
		languageSpec: &LanguageSpec{
			ID:   "test",
			Name: "Test",
		},
	}

	// Test GetLanguageSpec returns *LanguageSpec
	spec := mock.GetLanguageSpec()
	assert.IsType(t, &LanguageSpec{}, spec)

	// Test BuildProtocCommand takes context and CommandRequest
	ctx := context.Background()
	req := &CommandRequest{
		ProtoFiles:  []string{"test.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
	}
	cmd, err := mock.BuildProtocCommand(ctx, req)
	assert.NoError(t, err)
	assert.IsType(t, []string{}, cmd)

	// Test ValidateOutput takes context and string slice
	err = mock.ValidateOutput(ctx, []string{"file.go"})
	assert.NoError(t, err)
}

// TestLanguagePlugin_ContextHandling tests context handling in interface methods
func TestLanguagePlugin_ContextHandling(t *testing.T) {
	mock := &testLanguagePlugin{
		manifest: &Manifest{ID: "test", Name: "Test", Version: "1.0.0"},
		languageSpec: &LanguageSpec{ID: "test", Name: "Test"},
	}

	// Test with background context
	ctx := context.Background()
	req := &CommandRequest{
		ProtoFiles:  []string{"test.proto"},
		OutputDir:   "/output",
	}

	_, err := mock.BuildProtocCommand(ctx, req)
	assert.NoError(t, err)

	err = mock.ValidateOutput(ctx, []string{"test.go"})
	assert.NoError(t, err)

	// Test with canceled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = mock.BuildProtocCommand(canceledCtx, req)
	assert.NoError(t, err) // Mock doesn't check context, but interface supports it

	err = mock.ValidateOutput(canceledCtx, []string{"test.go"})
	assert.NoError(t, err) // Mock doesn't check context, but interface supports it
}

// TestLanguageSpec_JSONFieldTags tests that all JSON field tags are correct
func TestLanguageSpec_JSONFieldTags(t *testing.T) {
	spec := &LanguageSpec{
		ID:               "test",
		Name:             "Test",
		DisplayName:      "Test Display",
		SupportsGRPC:     true,
		FileExtensions:   []string{".test"},
		Enabled:          true,
		Stable:           true,
		Description:      "Test description",
		DocumentationURL: "https://test.com",
		PluginVersion:    "1.0.0",
		ProtocPlugin:     "test",
		DockerImage:      "test:latest",
	}

	data, err := json.Marshal(spec)
	require.NoError(t, err)

	// Parse JSON to verify field names
	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// Verify JSON field names match tags
	assert.Contains(t, raw, "id")
	assert.Contains(t, raw, "name")
	assert.Contains(t, raw, "display_name")
	assert.Contains(t, raw, "supports_grpc")
	assert.Contains(t, raw, "file_extensions")
	assert.Contains(t, raw, "enabled")
	assert.Contains(t, raw, "stable")
	assert.Contains(t, raw, "description")
	assert.Contains(t, raw, "documentation_url")
	assert.Contains(t, raw, "plugin_version")
	assert.Contains(t, raw, "protoc_plugin")
	assert.Contains(t, raw, "docker_image")
}

// TestPackageManager_JSONFieldTags tests that all JSON field tags are correct
func TestPackageManager_JSONFieldTags(t *testing.T) {
	pm := &PackageManager{
		Name:            "test",
		ConfigFiles:     []string{"test.json"},
		DependencyMap:   map[string]string{"dep": "value"},
		DefaultVersions: map[string]string{"dep": "1.0.0"},
	}

	data, err := json.Marshal(pm)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "name")
	assert.Contains(t, raw, "config_files")
	assert.Contains(t, raw, "dependency_map")
	assert.Contains(t, raw, "default_versions")
}

// TestCommandRequest_JSONFieldTags tests that all JSON field tags are correct
func TestCommandRequest_JSONFieldTags(t *testing.T) {
	req := &CommandRequest{
		ProtoFiles:  []string{"test.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		Options:     map[string]string{"opt": "value"},
		PluginPath:  "/path/to/plugin",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "proto_files")
	assert.Contains(t, raw, "import_paths")
	assert.Contains(t, raw, "output_dir")
	assert.Contains(t, raw, "options")
	assert.Contains(t, raw, "plugin_path")
}

// TestCommandResult_JSONFieldTags tests that all JSON field tags are correct
func TestCommandResult_JSONFieldTags(t *testing.T) {
	result := &CommandResult{
		ExitCode:       0,
		Stdout:         "output",
		Stderr:         "error",
		GeneratedFiles: []string{"file.go"},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "exit_code")
	assert.Contains(t, raw, "stdout")
	assert.Contains(t, raw, "stderr")
	assert.Contains(t, raw, "generated_files")
}
