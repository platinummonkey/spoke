package protobuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWithDescriptor_FileOptions(t *testing.T) {
	t.Run("go_package option", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

option go_package = "github.com/example/test";

message TestMessage {
  string name = 1;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify file-level option exists
		require.Len(t, ast.Options, 1)
		assert.Equal(t, "go_package", ast.Options[0].Name)
		assert.Equal(t, "github.com/example/test", ast.Options[0].Value)
	})

	t.Run("multiple file options", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

option go_package = "github.com/example/test";
option java_package = "com.example.test";
option java_outer_classname = "TestProtos";

message TestMessage {
  string name = 1;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify all file-level options exist
		require.Len(t, ast.Options, 3)

		optionMap := make(map[string]string)
		for _, opt := range ast.Options {
			optionMap[opt.Name] = opt.Value
		}

		assert.Equal(t, "github.com/example/test", optionMap["go_package"])
		assert.Equal(t, "com.example.test", optionMap["java_package"])
		assert.Equal(t, "TestProtos", optionMap["java_outer_classname"])
	})

	t.Run("deprecated option", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

option deprecated = true;

message TestMessage {
  string name = 1;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify deprecated option
		require.Len(t, ast.Options, 1)
		assert.Equal(t, "deprecated", ast.Options[0].Name)
		assert.Equal(t, "true", ast.Options[0].Value)
	})

	t.Run("optimize_for option", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

option optimize_for = SPEED;

message TestMessage {
  string name = 1;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify optimize_for option
		require.Len(t, ast.Options, 1)
		assert.Equal(t, "optimize_for", ast.Options[0].Name)
		// The value might be "SPEED" or "1" (enum value)
		assert.Contains(t, []string{"SPEED", "1"}, ast.Options[0].Value)
	})

	t.Run("no file options", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify no file-level options
		assert.Empty(t, ast.Options)
	})
}
