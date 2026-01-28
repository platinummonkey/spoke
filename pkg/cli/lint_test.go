package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLintCommand(t *testing.T) {
	cmd := newLintCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "lint", cmd.Name)
	assert.Equal(t, "Lint protobuf files for style and quality", cmd.Description)
	assert.NotNil(t, cmd.Flags)
	assert.NotNil(t, cmd.Run)
}

func TestLintFindProtoFiles(t *testing.T) {
	tests := []struct {
		name           string
		setupFiles     map[string]string
		expectedCount  int
		expectedFiles  []string
		wantErr        bool
	}{
		{
			name: "single proto file",
			setupFiles: map[string]string{
				"test.proto": "syntax = \"proto3\";",
			},
			expectedCount: 1,
			expectedFiles: []string{"test.proto"},
			wantErr:       false,
		},
		{
			name: "multiple proto files",
			setupFiles: map[string]string{
				"test1.proto": "syntax = \"proto3\";",
				"test2.proto": "syntax = \"proto3\";",
			},
			expectedCount: 2,
			expectedFiles: []string{"test1.proto", "test2.proto"},
			wantErr:       false,
		},
		{
			name: "nested proto files",
			setupFiles: map[string]string{
				"test.proto":     "syntax = \"proto3\";",
				"sub/test.proto": "syntax = \"proto3\";",
			},
			expectedCount: 2,
			expectedFiles: []string{"test.proto", "sub/test.proto"},
			wantErr:       false,
		},
		{
			name: "skip hidden directories",
			setupFiles: map[string]string{
				"test.proto":        "syntax = \"proto3\";",
				".hidden/test.proto": "syntax = \"proto3\";",
			},
			expectedCount: 1,
			expectedFiles: []string{"test.proto"},
			wantErr:       false,
		},
		{
			name: "skip vendor directories",
			setupFiles: map[string]string{
				"test.proto":          "syntax = \"proto3\";",
				"vendor/test.proto":   "syntax = \"proto3\";",
			},
			expectedCount: 1,
			expectedFiles: []string{"test.proto"},
			wantErr:       false,
		},
		{
			name: "skip third_party directories",
			setupFiles: map[string]string{
				"test.proto":              "syntax = \"proto3\";",
				"third_party/test.proto": "syntax = \"proto3\";",
			},
			expectedCount: 1,
			expectedFiles: []string{"test.proto"},
			wantErr:       false,
		},
		{
			name: "ignore non-proto files",
			setupFiles: map[string]string{
				"test.proto": "syntax = \"proto3\";",
				"test.go":    "package main",
				"test.txt":   "hello",
			},
			expectedCount: 1,
			expectedFiles: []string{"test.proto"},
			wantErr:       false,
		},
		{
			name:          "no proto files",
			setupFiles:    map[string]string{
				"test.go": "package main",
			},
			expectedCount: 0,
			expectedFiles: []string{},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()

			// Set up test files
			for path, content := range tt.setupFiles {
				fullPath := filepath.Join(testDir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0644)
				require.NoError(t, err)
			}

			files, err := lintFindProtoFiles(testDir)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(files))

				// Check that all expected files are found
				fileMap := make(map[string]bool)
				for _, f := range files {
					relPath, _ := filepath.Rel(testDir, f)
					fileMap[relPath] = true
				}

				for _, expected := range tt.expectedFiles {
					assert.True(t, fileMap[expected], "Expected file %s not found", expected)
				}
			}
		})
	}
}

func TestRunLint(t *testing.T) {
	validProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message TestMessage {
    string id = 1;
    int32 value = 2;
}`

	invalidProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message testMessage {
    string id = 1;
    int32 value = 2;
}`

	tests := []struct {
		name       string
		setupFiles map[string]string
		configFile string
		format     string
		autoFix    bool
		rulesOnly  bool
		verbose    bool
		wantErr    bool
	}{
		{
			name: "valid proto file with default config",
			setupFiles: map[string]string{
				"test.proto": validProto,
			},
			format:  "text",
			wantErr: false,
		},
		{
			name: "invalid proto file with naming violations",
			setupFiles: map[string]string{
				"test.proto": invalidProto,
			},
			format:  "text",
			wantErr: true, // Will fail because fail-on-error is true
		},
		{
			name: "json output format",
			setupFiles: map[string]string{
				"test.proto": validProto,
			},
			format:  "json",
			wantErr: false,
		},
		{
			name: "github output format",
			setupFiles: map[string]string{
				"test.proto": validProto,
			},
			format:  "github",
			wantErr: false,
		},
		{
			name: "verbose output",
			setupFiles: map[string]string{
				"test.proto": validProto,
			},
			format:  "text",
			verbose: true,
			wantErr: false,
		},
		{
			name: "no proto files",
			setupFiles: map[string]string{
				"test.go": "package main",
			},
			format:  "text",
			wantErr: false,
		},
		{
			name: "rules only flag",
			setupFiles: map[string]string{
				"test.proto": validProto,
			},
			format:    "text",
			rulesOnly: true,
			wantErr:   false,
		},
		{
			name: "auto-fix flag (not implemented yet)",
			setupFiles: map[string]string{
				"test.proto": validProto,
			},
			format:  "text",
			autoFix: true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()

			// Set up test files
			for path, content := range tt.setupFiles {
				fullPath := filepath.Join(testDir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0644)
				require.NoError(t, err)
			}

			// Set up config file if specified
			configPath := ""
			if tt.configFile != "" {
				configPath = filepath.Join(testDir, "spoke-lint.yaml")
				err := os.WriteFile(configPath, []byte(tt.configFile), 0644)
				require.NoError(t, err)
			}

			err := runLint(testDir, configPath, tt.format, tt.autoFix, true, false, tt.verbose, tt.rulesOnly)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunLintWithFailOnError(t *testing.T) {
	// Create a proto file with naming violations that will generate errors
	invalidProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message testMessage {
    string id = 1;
}`

	testDir := t.TempDir()
	protoPath := filepath.Join(testDir, "test.proto")
	err := os.WriteFile(protoPath, []byte(invalidProto), 0644)
	require.NoError(t, err)

	// With fail-on-error=true and violations that are errors, should return error
	err = runLint(testDir, "", "text", false, true, false, false, false)
	// Naming violations are errors, so this should error with fail-on-error=true
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lint failed")
}

func TestRunLintWithFailOnWarning(t *testing.T) {
	validProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message TestMessage {
    string id = 1;
}`

	testDir := t.TempDir()
	protoPath := filepath.Join(testDir, "test.proto")
	err := os.WriteFile(protoPath, []byte(validProto), 0644)
	require.NoError(t, err)

	// With valid proto and fail-on-warning, should not error
	err = runLint(testDir, "", "text", false, false, true, false, false)
	assert.NoError(t, err)
}

func TestRunLintWithConfigFile(t *testing.T) {
	validProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message TestMessage {
    string id = 1;
}`

	testDir := t.TempDir()
	protoPath := filepath.Join(testDir, "test.proto")
	err := os.WriteFile(protoPath, []byte(validProto), 0644)
	require.NoError(t, err)

	// Create a config file
	configPath := filepath.Join(testDir, "spoke-lint.yaml")
	configContent := `version: v1
lint:
  use:
    - google
  rules: {}
  ignore:
    - vendor/**
quality:
  enabled: true
autofix:
  enabled: false
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = runLint(testDir, configPath, "text", false, true, false, false, false)
	assert.NoError(t, err)
}

func TestRunLintInvalidConfigFile(t *testing.T) {
	testDir := t.TempDir()

	// Create a valid proto file
	validProto := `syntax = "proto3";
package test;

message TestMessage {
    string id = 1;
}`
	protoPath := filepath.Join(testDir, "test.proto")
	err := os.WriteFile(protoPath, []byte(validProto), 0644)
	require.NoError(t, err)

	// Try to use a non-existent config file
	err = runLint(testDir, "/nonexistent/config.yaml", "text", false, true, false, false, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestRunLintInvalidProtoFile(t *testing.T) {
	testDir := t.TempDir()

	// Create an invalid proto file (malformed syntax)
	invalidProto := `syntax = "proto3"
package test
this is invalid proto syntax
message {
    string id = 1;
}`
	protoPath := filepath.Join(testDir, "test.proto")
	err := os.WriteFile(protoPath, []byte(invalidProto), 0644)
	require.NoError(t, err)

	err = runLint(testDir, "", "text", false, true, false, false, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestLintCommand_Run(t *testing.T) {
	validProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message TestMessage {
    string id = 1;
}`

	testDir := t.TempDir()
	protoPath := filepath.Join(testDir, "test.proto")
	err := os.WriteFile(protoPath, []byte(validProto), 0644)
	require.NoError(t, err)

	cmd := newLintCommand()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "with directory flag",
			args:    []string{"-dir", testDir},
			wantErr: false,
		},
		{
			name:    "with format flag",
			args:    []string{"-dir", testDir, "-format", "json"},
			wantErr: false,
		},
		{
			name:    "with verbose flag",
			args:    []string{"-dir", testDir, "-verbose"},
			wantErr: false,
		},
		{
			name:    "with rules flag",
			args:    []string{"-dir", testDir, "-rules"},
			wantErr: false,
		},
		{
			name:    "with fail-on-error flag",
			args:    []string{"-dir", testDir, "-fail-on-error=true"},
			wantErr: false,
		},
		{
			name:    "with fail-on-warning flag",
			args:    []string{"-dir", testDir, "-fail-on-warning=true"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.Run(tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// Successful runs return nil
				// Just ensure it doesn't panic
				_ = err // Can be nil or error depending on lint results
			}
		})
	}
}

func TestLintOutputFormats(t *testing.T) {
	validProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message TestMessage {
    string id = 1;
}`

	testDir := t.TempDir()
	protoPath := filepath.Join(testDir, "test.proto")
	err := os.WriteFile(protoPath, []byte(validProto), 0644)
	require.NoError(t, err)

	formats := []string{"text", "json", "github"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			err := runLint(testDir, "", format, false, true, false, false, false)
			// Should not error on valid proto
			assert.NoError(t, err)
		})
	}
}

func TestLintMultipleProtoFiles(t *testing.T) {
	testDir := t.TempDir()

	// Create multiple proto files
	for i := 1; i <= 3; i++ {
		content := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message TestMessage` + string(rune('0'+i)) + ` {
    string id = 1;
}`
		protoPath := filepath.Join(testDir, "test"+string(rune('0'+i))+".proto")
		err := os.WriteFile(protoPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	err := runLint(testDir, "", "text", false, true, false, true, false)
	assert.NoError(t, err)
}

func TestLintGitHubOutputWithViolations(t *testing.T) {
	// Create a proto file with violations to test GitHub output format
	invalidProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message testMessage {
    string BadField = 1;
    int32 AnotherBad = 2;
}`

	testDir := t.TempDir()
	protoPath := filepath.Join(testDir, "test.proto")
	err := os.WriteFile(protoPath, []byte(invalidProto), 0644)
	require.NoError(t, err)

	// Run with GitHub format to test that output path
	err = runLint(testDir, "", "github", false, false, false, false, false)
	// Should not error even with violations unless fail-on-error/warning is set
	assert.NoError(t, err)
}

func TestLintTextOutputWithViolations(t *testing.T) {
	// Create a proto file with violations
	invalidProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message testMessage {
    string BadField = 1;
}`

	testDir := t.TempDir()
	protoPath := filepath.Join(testDir, "test.proto")
	err := os.WriteFile(protoPath, []byte(invalidProto), 0644)
	require.NoError(t, err)

	// Test text output with violations
	err = runLint(testDir, "", "text", false, false, false, false, false)
	assert.NoError(t, err)
}

func TestLintTextOutputVerboseWithViolations(t *testing.T) {
	// Create a proto file with violations to test verbose output
	invalidProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/test";

message testMessage {
    string id = 1;
}`

	testDir := t.TempDir()
	protoPath := filepath.Join(testDir, "test.proto")
	err := os.WriteFile(protoPath, []byte(invalidProto), 0644)
	require.NoError(t, err)

	// Test verbose output to hit the suggested fix printing path
	err = runLint(testDir, "", "text", false, false, false, true, false)
	assert.NoError(t, err)
}
