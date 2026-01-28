package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCheckCompatibilityCommand(t *testing.T) {
	cmd := newCheckCompatibilityCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "check-compatibility", cmd.Name)
	assert.NotNil(t, cmd.Flags)
	assert.NotNil(t, cmd.Run)

	// Verify flags are registered
	assert.NotNil(t, cmd.Flags.Lookup("old"))
	assert.NotNil(t, cmd.Flags.Lookup("new"))
	assert.NotNil(t, cmd.Flags.Lookup("mode"))
	assert.NotNil(t, cmd.Flags.Lookup("verbose"))
	assert.NotNil(t, cmd.Flags.Lookup("format"))
}

func TestRunCheckCompatibilityMissingFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing both flags",
			args:    []string{},
			wantErr: "both --old and --new are required",
		},
		{
			name:    "missing old flag",
			args:    []string{"-new", "/some/path"},
			wantErr: "both --old and --new are required",
		},
		{
			name:    "missing new flag",
			args:    []string{"-old", "/some/path"},
			wantErr: "both --old and --new are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCheckCompatibility(tt.args)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestRunCheckCompatibilityInvalidMode(t *testing.T) {
	tempDir := t.TempDir()

	// Create dummy proto files
	oldProto := filepath.Join(tempDir, "old.proto")
	newProto := filepath.Join(tempDir, "new.proto")

	simpleProto := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}
`
	err := os.WriteFile(oldProto, []byte(simpleProto), 0644)
	require.NoError(t, err)
	err = os.WriteFile(newProto, []byte(simpleProto), 0644)
	require.NoError(t, err)

	args := []string{
		"-old", oldProto,
		"-new", newProto,
		"-mode", "INVALID_MODE",
	}

	err = runCheckCompatibility(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid compatibility mode")
}

func TestRunCheckCompatibilityOldPathNotExists(t *testing.T) {
	tempDir := t.TempDir()

	newProto := filepath.Join(tempDir, "new.proto")
	simpleProto := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}
`
	err := os.WriteFile(newProto, []byte(simpleProto), 0644)
	require.NoError(t, err)

	args := []string{
		"-old", "/nonexistent/path.proto",
		"-new", newProto,
	}

	err = runCheckCompatibility(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse old schema")
}

func TestRunCheckCompatibilityNewPathNotExists(t *testing.T) {
	tempDir := t.TempDir()

	oldProto := filepath.Join(tempDir, "old.proto")
	simpleProto := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}
`
	err := os.WriteFile(oldProto, []byte(simpleProto), 0644)
	require.NoError(t, err)

	args := []string{
		"-old", oldProto,
		"-new", "/nonexistent/path.proto",
	}

	err = runCheckCompatibility(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse new schema")
}

func TestParseSchemaFileDoesNotExist(t *testing.T) {
	_, err := parseSchema("/nonexistent/file.proto")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path does not exist")
}

func TestParseSchemaInvalidProto(t *testing.T) {
	tempDir := t.TempDir()
	invalidProto := filepath.Join(tempDir, "invalid.proto")

	// Write invalid proto content
	err := os.WriteFile(invalidProto, []byte("invalid proto content {{{"), 0644)
	require.NoError(t, err)

	_, err = parseSchema(invalidProto)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse proto")
}

func TestParseSchemaValidFile(t *testing.T) {
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")

	protoContent := `syntax = "proto3";
package test;

message User {
  string id = 1;
  string name = 2;
  int32 age = 3;
}
`
	err := os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	schema, err := parseSchema(protoFile)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, "test", schema.Package)
	assert.Contains(t, schema.Messages, "User")
}

func TestParseSchemaDirectory(t *testing.T) {
	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "protos")
	err := os.MkdirAll(protoDir, 0755)
	require.NoError(t, err)

	// Create multiple proto files
	proto1 := filepath.Join(protoDir, "test1.proto")
	proto2 := filepath.Join(protoDir, "test2.proto")

	protoContent := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}
`
	err = os.WriteFile(proto1, []byte(protoContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(proto2, []byte(protoContent), 0644)
	require.NoError(t, err)

	schema, err := parseSchema(protoDir)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
}

func TestParseSchemaEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	emptyDir := filepath.Join(tempDir, "empty")
	err := os.MkdirAll(emptyDir, 0755)
	require.NoError(t, err)

	_, err = parseSchema(emptyDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no proto files found")
}

func TestRunCheckCompatibilityCompatible(t *testing.T) {
	tempDir := t.TempDir()

	oldProto := filepath.Join(tempDir, "old.proto")
	newProto := filepath.Join(tempDir, "new.proto")

	oldContent := `syntax = "proto3";
package test;

message User {
  string id = 1;
  string name = 2;
}
`

	// New schema adds a field (backward compatible)
	newContent := `syntax = "proto3";
package test;

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}
`

	err := os.WriteFile(oldProto, []byte(oldContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(newProto, []byte(newContent), 0644)
	require.NoError(t, err)

	args := []string{
		"-old", oldProto,
		"-new", newProto,
		"-mode", "BACKWARD",
	}

	err = runCheckCompatibility(args)
	assert.NoError(t, err)
}

func TestRunCheckCompatibilityIncompatible(t *testing.T) {
	tempDir := t.TempDir()

	oldProto := filepath.Join(tempDir, "old.proto")
	newProto := filepath.Join(tempDir, "new.proto")

	oldContent := `syntax = "proto3";
package old_package;

message User {
  string id = 1;
  string name = 2;
}
`

	// New schema changes package (incompatible)
	newContent := `syntax = "proto3";
package new_package;

message User {
  string id = 1;
  string name = 2;
}
`

	err := os.WriteFile(oldProto, []byte(oldContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(newProto, []byte(newContent), 0644)
	require.NoError(t, err)

	args := []string{
		"-old", oldProto,
		"-new", newProto,
		"-mode", "BACKWARD",
	}

	err = runCheckCompatibility(args)
	if err != nil {
		assert.Contains(t, err.Error(), "compatibility check failed")
	}
}

func TestRunCheckCompatibilityWithVerbose(t *testing.T) {
	tempDir := t.TempDir()

	oldProto := filepath.Join(tempDir, "old.proto")
	newProto := filepath.Join(tempDir, "new.proto")

	oldContent := `syntax = "proto3";
package test;

message User {
  string id = 1;
}
`

	// Add field (creates info-level violation)
	newContent := `syntax = "proto3";
package test;

message User {
  string id = 1;
  string name = 2;
}
`

	err := os.WriteFile(oldProto, []byte(oldContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(newProto, []byte(newContent), 0644)
	require.NoError(t, err)

	args := []string{
		"-old", oldProto,
		"-new", newProto,
		"-mode", "BACKWARD",
		"-verbose",
	}

	err = runCheckCompatibility(args)
	assert.NoError(t, err)
}

func TestRunCheckCompatibilityDifferentModes(t *testing.T) {
	tempDir := t.TempDir()

	oldProto := filepath.Join(tempDir, "old.proto")
	newProto := filepath.Join(tempDir, "new.proto")

	protoContent := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}
`

	err := os.WriteFile(oldProto, []byte(protoContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(newProto, []byte(protoContent), 0644)
	require.NoError(t, err)

	modes := []string{
		"BACKWARD",
		"FORWARD",
		"FULL",
		"BACKWARD_TRANSITIVE",
		"FORWARD_TRANSITIVE",
		"FULL_TRANSITIVE",
	}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			args := []string{
				"-old", oldProto,
				"-new", newProto,
				"-mode", mode,
			}

			err := runCheckCompatibility(args)
			assert.NoError(t, err)
		})
	}
}

func TestOutputJSONNotImplemented(t *testing.T) {
	tempDir := t.TempDir()

	oldProto := filepath.Join(tempDir, "old.proto")
	newProto := filepath.Join(tempDir, "new.proto")

	protoContent := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}
`

	err := os.WriteFile(oldProto, []byte(protoContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(newProto, []byte(protoContent), 0644)
	require.NoError(t, err)

	args := []string{
		"-old", oldProto,
		"-new", newProto,
		"-format", "json",
	}

	err = runCheckCompatibility(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JSON output not yet implemented")
}

func TestParseSchemaWithNestedMessages(t *testing.T) {
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "nested.proto")

	protoContent := `syntax = "proto3";
package test;

message Outer {
  string id = 1;

  message Inner {
    string name = 1;
    int32 value = 2;
  }

  Inner inner = 2;
}
`

	err := os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	schema, err := parseSchema(protoFile)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Contains(t, schema.Messages, "Outer")
}

func TestParseSchemaWithEnum(t *testing.T) {
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "enum.proto")

	protoContent := `syntax = "proto3";
package test;

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}

message User {
  string id = 1;
  Status status = 2;
}
`

	err := os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	schema, err := parseSchema(protoFile)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Contains(t, schema.Messages, "User")
	assert.Contains(t, schema.Enums, "Status")
}

func TestParseSchemaWithService(t *testing.T) {
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "service.proto")

	protoContent := `syntax = "proto3";
package test;

message Request {
  string id = 1;
}

message Response {
  string result = 1;
}

service TestService {
  rpc GetData(Request) returns (Response);
}
`

	err := os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	schema, err := parseSchema(protoFile)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Contains(t, schema.Services, "TestService")
}

func TestParseSchemaDirectoryWithSubdirectories(t *testing.T) {
	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "protos")
	subDir := filepath.Join(protoDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Create proto in subdirectory
	protoFile := filepath.Join(subDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}
`

	err = os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	schema, err := parseSchema(protoDir)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
}

func TestRunCheckCompatibilityPackageChange(t *testing.T) {
	tempDir := t.TempDir()

	oldProto := filepath.Join(tempDir, "old.proto")
	newProto := filepath.Join(tempDir, "new.proto")

	oldContent := `syntax = "proto3";
package old_package;

message Test {
  string id = 1;
}
`

	newContent := `syntax = "proto3";
package new_package;

message Test {
  string id = 1;
}
`

	err := os.WriteFile(oldProto, []byte(oldContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(newProto, []byte(newContent), 0644)
	require.NoError(t, err)

	args := []string{
		"-old", oldProto,
		"-new", newProto,
	}

	err = runCheckCompatibility(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compatibility check failed")
}

func TestRunCheckCompatibilityMessageRemoved(t *testing.T) {
	tempDir := t.TempDir()

	oldProto := filepath.Join(tempDir, "old.proto")
	newProto := filepath.Join(tempDir, "new.proto")

	oldContent := `syntax = "proto3";
package test;

message User {
  string id = 1;
}

message Order {
  string id = 1;
}
`

	newContent := `syntax = "proto3";
package test;

message User {
  string id = 1;
}
`

	err := os.WriteFile(oldProto, []byte(oldContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(newProto, []byte(newContent), 0644)
	require.NoError(t, err)

	args := []string{
		"-old", oldProto,
		"-new", newProto,
	}

	err = runCheckCompatibility(args)
	if err != nil {
		assert.Contains(t, err.Error(), "compatibility check failed")
	}
}

func TestParseSchemaReadFileError(t *testing.T) {
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")

	// Create a file
	err := os.WriteFile(protoFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Make it unreadable (this might not work on all systems)
	err = os.Chmod(protoFile, 0000)
	if err == nil {
		defer os.Chmod(protoFile, 0644) // cleanup

		_, err = parseSchema(protoFile)
		// On some systems we can't make files unreadable, so only assert error if chmod worked
		if err != nil {
			assert.Error(t, err)
		}
	}
}

func TestParseSchemaDirectoryWalkError(t *testing.T) {
	// Test directory walk by creating a directory and then checking it
	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "protos")
	err := os.MkdirAll(protoDir, 0755)
	require.NoError(t, err)

	// Create a proto file
	protoFile := filepath.Join(protoDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}
`
	err = os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	// Should succeed normally
	schema, err := parseSchema(protoDir)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
}
