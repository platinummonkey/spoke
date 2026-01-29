package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewBasicLanguagePlugin(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-lang",
		Name:    "Test Language",
		Version: "1.0.0",
	}
	pluginDir := "/tmp/test-plugin"

	plugin := NewBasicLanguagePlugin(manifest, pluginDir)

	assert.NotNil(t, plugin)
	assert.Equal(t, manifest, plugin.manifest)
	assert.Equal(t, pluginDir, plugin.pluginDir)
	assert.Nil(t, plugin.languageSpec) // Not loaded yet
}

func TestBasicLanguagePlugin_Manifest(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-lang",
		Name:    "Test Language",
		Version: "1.0.0",
	}
	pluginDir := "/tmp/test-plugin"

	plugin := NewBasicLanguagePlugin(manifest, pluginDir)
	result := plugin.Manifest()

	assert.Equal(t, manifest, result)
}

func TestBasicLanguagePlugin_Load_WithLanguageSpecFile(t *testing.T) {
	// Create temporary plugin directory
	tmpDir := t.TempDir()

	// Create language spec file
	langSpec := &LanguageSpec{
		ID:             "go",
		Name:           "Go",
		DisplayName:    "Go",
		SupportsGRPC:   true,
		FileExtensions: []string{".go", ".pb.go"},
		Enabled:        true,
		Stable:         true,
		Description:    "Go language support",
		ProtocPlugin:   "go",
	}

	langSpecData, err := yaml.Marshal(langSpec)
	require.NoError(t, err)

	langSpecPath := filepath.Join(tmpDir, "language_spec.yaml")
	err = os.WriteFile(langSpecPath, langSpecData, 0644)
	require.NoError(t, err)

	// Create plugin
	manifest := &Manifest{
		ID:      "go-plugin",
		Name:    "Go Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	err = plugin.Load()

	assert.NoError(t, err)
	assert.NotNil(t, plugin.languageSpec)
	assert.Equal(t, "go", plugin.languageSpec.ID)
	assert.Equal(t, "Go", plugin.languageSpec.Name)
	assert.True(t, plugin.languageSpec.SupportsGRPC)
	assert.Equal(t, []string{".go", ".pb.go"}, plugin.languageSpec.FileExtensions)
}

func TestBasicLanguagePlugin_Load_WithManifestMetadata(t *testing.T) {
	// Create temporary plugin directory (no language_spec.yaml)
	tmpDir := t.TempDir()

	// Create manifest with embedded language spec
	langSpec := &LanguageSpec{
		ID:             "rust",
		Name:           "Rust",
		DisplayName:    "Rust",
		SupportsGRPC:   true,
		FileExtensions: []string{".rs"},
		Enabled:        true,
		Stable:         false,
		Description:    "Rust language support",
	}

	langSpecData, err := yaml.Marshal(langSpec)
	require.NoError(t, err)

	manifest := &Manifest{
		ID:       "rust-plugin",
		Name:     "Rust Plugin",
		Version:  "1.0.0",
		Metadata: map[string]string{"language_spec": string(langSpecData)},
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	err = plugin.Load()

	assert.NoError(t, err)
	assert.NotNil(t, plugin.languageSpec)
	assert.Equal(t, "rust", plugin.languageSpec.ID)
	assert.Equal(t, "Rust", plugin.languageSpec.Name)
	assert.Equal(t, []string{".rs"}, plugin.languageSpec.FileExtensions)
}

func TestBasicLanguagePlugin_Load_WithDefaultSpec(t *testing.T) {
	// Create temporary plugin directory (no language_spec.yaml)
	tmpDir := t.TempDir()

	// Create manifest without language spec metadata
	manifest := &Manifest{
		ID:          "default-plugin",
		Name:        "Default Plugin",
		Version:     "1.0.0",
		Description: "Default plugin description",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	err := plugin.Load()

	assert.NoError(t, err)
	assert.NotNil(t, plugin.languageSpec)
	assert.Equal(t, "default-plugin", plugin.languageSpec.ID)
	assert.Equal(t, "Default Plugin", plugin.languageSpec.Name)
	assert.Equal(t, "Default Plugin", plugin.languageSpec.DisplayName)
	assert.True(t, plugin.languageSpec.Enabled)
	assert.True(t, plugin.languageSpec.Stable)
	assert.Equal(t, "Default plugin description", plugin.languageSpec.Description)
}

func TestBasicLanguagePlugin_Load_InvalidYAMLFile(t *testing.T) {
	// Create temporary plugin directory
	tmpDir := t.TempDir()

	// Create invalid YAML file
	langSpecPath := filepath.Join(tmpDir, "language_spec.yaml")
	err := os.WriteFile(langSpecPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	manifest := &Manifest{
		ID:      "invalid-plugin",
		Name:    "Invalid Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	err = plugin.Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse language spec")
}

func TestBasicLanguagePlugin_Load_InvalidMetadataYAML(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:       "invalid-metadata",
		Name:     "Invalid Metadata",
		Version:  "1.0.0",
		Metadata: map[string]string{"language_spec": "invalid: yaml: ["},
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	err := plugin.Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse language spec from manifest")
}

func TestBasicLanguagePlugin_Load_UnreadableSpecFile(t *testing.T) {
	// Create temporary plugin directory
	tmpDir := t.TempDir()

	// Create a language spec file with no read permissions
	langSpecPath := filepath.Join(tmpDir, "language_spec.yaml")
	err := os.WriteFile(langSpecPath, []byte("id: test"), 0000)
	require.NoError(t, err)

	manifest := &Manifest{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	err = plugin.Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read language spec file")

	// Clean up - restore permissions so temp dir can be deleted
	os.Chmod(langSpecPath, 0644)
}

func TestBasicLanguagePlugin_Unload(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	err := plugin.Unload()

	// Basic plugin has no resources to clean up
	assert.NoError(t, err)
}

func TestBasicLanguagePlugin_GetLanguageSpec(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:            "test-plugin",
		Name:          "Test Plugin",
		Version:       "1.0.0",
		SecurityLevel: SecurityLevelOfficial,
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	err := plugin.Load()
	require.NoError(t, err)

	spec := plugin.GetLanguageSpec()

	assert.NotNil(t, spec)
	assert.Equal(t, "test-plugin", spec.ID)
}

func TestBasicLanguagePlugin_GetLanguageSpec_NotLoaded(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	spec := plugin.GetLanguageSpec()

	assert.Nil(t, spec)
}

func TestBasicLanguagePlugin_BuildProtocCommand_Basic(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "go-plugin",
		Name:    "Go Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	plugin.languageSpec = &LanguageSpec{
		ID:           "go",
		Name:         "Go",
		ProtocPlugin: "go",
	}

	ctx := context.Background()
	req := &CommandRequest{
		ProtoFiles:  []string{"service.proto", "types.proto"},
		ImportPaths: []string{"/proto", "/include"},
		OutputDir:   "/output",
		Options:     map[string]string{},
	}

	cmd, err := plugin.BuildProtocCommand(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, "protoc", cmd[0])
	assert.Contains(t, cmd, "--proto_path=/proto")
	assert.Contains(t, cmd, "--proto_path=/include")
	assert.Contains(t, cmd, "--go_out=/output")
	assert.Contains(t, cmd, "service.proto")
	assert.Contains(t, cmd, "types.proto")
}

func TestBasicLanguagePlugin_BuildProtocCommand_WithPluginPath(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "cpp-plugin",
		Name:    "C++ Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	plugin.languageSpec = &LanguageSpec{
		ID:           "cpp",
		Name:         "C++",
		ProtocPlugin: "cpp",
	}

	ctx := context.Background()
	req := &CommandRequest{
		ProtoFiles:  []string{"service.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		PluginPath:  "/usr/local/bin/protoc-gen-cpp",
		Options:     map[string]string{},
	}

	cmd, err := plugin.BuildProtocCommand(ctx, req)

	assert.NoError(t, err)
	assert.Contains(t, cmd, "--plugin=protoc-gen-cpp=/usr/local/bin/protoc-gen-cpp")
}

func TestBasicLanguagePlugin_BuildProtocCommand_WithOptions(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "go-plugin",
		Name:    "Go Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	plugin.languageSpec = &LanguageSpec{
		ID:   "go",
		Name: "Go",
	}

	ctx := context.Background()
	req := &CommandRequest{
		ProtoFiles:  []string{"service.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		Options: map[string]string{
			"paths":        "source_relative",
			"require_unimplemented_servers": "false",
		},
	}

	cmd, err := plugin.BuildProtocCommand(ctx, req)

	assert.NoError(t, err)
	// Check that options are included (order may vary)
	optFlag := ""
	for _, arg := range cmd {
		if len(arg) >= 10 && arg[:9] == "--go_opt=" {
			optFlag = arg
			break
		}
	}
	assert.NotEmpty(t, optFlag, "Expected to find --go_opt flag in command")
	assert.Contains(t, optFlag, "paths=source_relative")
	assert.Contains(t, optFlag, "require_unimplemented_servers=false")
}

func TestBasicLanguagePlugin_BuildProtocCommand_WithBooleanOption(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "go-plugin",
		Name:    "Go Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	plugin.languageSpec = &LanguageSpec{
		ID:   "go",
		Name: "Go",
	}

	ctx := context.Background()
	req := &CommandRequest{
		ProtoFiles:  []string{"service.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		Options: map[string]string{
			"annotate_code": "",
		},
	}

	cmd, err := plugin.BuildProtocCommand(ctx, req)

	assert.NoError(t, err)
	// Check that boolean option is included without value
	found := false
	for _, arg := range cmd {
		if arg == "--go_opt=annotate_code" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find --go_opt=annotate_code")
}

func TestBasicLanguagePlugin_BuildProtocCommand_NoLanguageSpec(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "no-spec-plugin",
		Name:    "No Spec Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	// Don't load or set language spec

	ctx := context.Background()
	req := &CommandRequest{
		ProtoFiles:  []string{"service.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		Options:     map[string]string{},
	}

	cmd, err := plugin.BuildProtocCommand(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "language spec not loaded")
}

func TestBasicLanguagePlugin_BuildProtocCommand_PluginNameFallback(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "custom-plugin",
		Name:    "Custom Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	plugin.languageSpec = &LanguageSpec{
		ID:   "custom",
		Name: "Custom",
		// ProtocPlugin not set - should use ID
	}

	ctx := context.Background()
	req := &CommandRequest{
		ProtoFiles:  []string{"service.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		PluginPath:  "/usr/local/bin/protoc-gen-custom",
		Options:     map[string]string{},
	}

	cmd, err := plugin.BuildProtocCommand(ctx, req)

	assert.NoError(t, err)
	// Should use ID when ProtocPlugin is empty
	assert.Contains(t, cmd, "--plugin=protoc-gen-custom=/usr/local/bin/protoc-gen-custom")
}

func TestBasicLanguagePlugin_ValidateOutput_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "go-plugin",
		Name:    "Go Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	plugin.languageSpec = &LanguageSpec{
		ID:             "go",
		Name:           "Go",
		FileExtensions: []string{".go", ".pb.go"},
	}

	ctx := context.Background()
	files := []string{
		"service.pb.go",
		"types.pb.go",
		"handler.go",
	}

	err := plugin.ValidateOutput(ctx, files)

	assert.NoError(t, err)
}

func TestBasicLanguagePlugin_ValidateOutput_InvalidExtension(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "go-plugin",
		Name:    "Go Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	plugin.languageSpec = &LanguageSpec{
		ID:             "go",
		Name:           "Go",
		FileExtensions: []string{".go", ".pb.go"},
	}

	ctx := context.Background()
	files := []string{
		"service.pb.go",
		"types.py", // Invalid extension
	}

	err := plugin.ValidateOutput(ctx, files)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected file extension")
	assert.Contains(t, err.Error(), "types.py")
}

func TestBasicLanguagePlugin_ValidateOutput_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "go-plugin",
		Name:    "Go Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	plugin.languageSpec = &LanguageSpec{
		ID:             "go",
		Name:           "Go",
		FileExtensions: []string{".go"},
	}

	ctx := context.Background()
	files := []string{}

	err := plugin.ValidateOutput(ctx, files)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no files generated")
}

func TestBasicLanguagePlugin_ValidateOutput_NoLanguageSpec(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "no-spec-plugin",
		Name:    "No Spec Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	// Don't load or set language spec

	ctx := context.Background()
	files := []string{"output.go"}

	err := plugin.ValidateOutput(ctx, files)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "language spec not loaded")
}

func TestBasicLanguagePlugin_ValidateOutput_MultipleExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &Manifest{
		ID:      "multi-plugin",
		Name:    "Multi Plugin",
		Version: "1.0.0",
	}

	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	plugin.languageSpec = &LanguageSpec{
		ID:             "multi",
		Name:           "Multi",
		FileExtensions: []string{".h", ".cc", ".cpp"},
	}

	ctx := context.Background()
	files := []string{
		"service.h",
		"service.cc",
		"types.cpp",
	}

	err := plugin.ValidateOutput(ctx, files)

	assert.NoError(t, err)
}

func TestBasicLanguagePlugin_EndToEnd(t *testing.T) {
	// Create temporary plugin directory
	tmpDir := t.TempDir()

	// Create language spec file
	langSpec := &LanguageSpec{
		ID:             "python",
		Name:           "Python",
		DisplayName:    "Python",
		SupportsGRPC:   true,
		FileExtensions: []string{".py", "_pb2.py"},
		Enabled:        true,
		Stable:         true,
		Description:    "Python language support",
		ProtocPlugin:   "python",
	}

	langSpecData, err := yaml.Marshal(langSpec)
	require.NoError(t, err)

	langSpecPath := filepath.Join(tmpDir, "language_spec.yaml")
	err = os.WriteFile(langSpecPath, langSpecData, 0644)
	require.NoError(t, err)

	// Create manifest
	manifest := &Manifest{
		ID:          "python-plugin",
		Name:        "Python Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Description: "Python protobuf plugin",
	}

	// Create and load plugin
	plugin := NewBasicLanguagePlugin(manifest, tmpDir)
	err = plugin.Load()
	require.NoError(t, err)

	// Test language spec
	spec := plugin.GetLanguageSpec()
	assert.Equal(t, "python", spec.ID)
	assert.True(t, spec.SupportsGRPC)

	// Test building protoc command
	ctx := context.Background()
	req := &CommandRequest{
		ProtoFiles:  []string{"service.proto"},
		ImportPaths: []string{"/proto"},
		OutputDir:   "/output",
		Options: map[string]string{
			"pyi_out": "/output",
		},
	}

	cmd, err := plugin.BuildProtocCommand(ctx, req)
	require.NoError(t, err)
	assert.Contains(t, cmd, "protoc")
	assert.Contains(t, cmd, "--python_out=/output")

	// Test validating output
	files := []string{
		"service_pb2.py",
		"types_pb2.py",
	}

	err = plugin.ValidateOutput(ctx, files)
	assert.NoError(t, err)

	// Test unload
	err = plugin.Unload()
	assert.NoError(t, err)
}
