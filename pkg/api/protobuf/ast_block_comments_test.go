package protobuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWithDescriptor_BlockComments(t *testing.T) {
	t.Run("single line block comment with directive", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

/* @spoke:option:validated */
message TestMessage {
  string name = 1;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify message has directive from block comment
		require.Len(t, ast.Messages, 1)
		msg := ast.Messages[0]
		require.Len(t, msg.SpokeDirectives, 1)
		assert.Equal(t, "option", msg.SpokeDirectives[0].Option)
		assert.Equal(t, "validated", msg.SpokeDirectives[0].Value)
	})

	t.Run("multi-line block comment with directive", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

/*
 * This is a test message
 * @spoke:option:entity
 * More documentation here
 */
message TestMessage {
  string name = 1;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify message has directive from multi-line block comment
		require.Len(t, ast.Messages, 1)
		msg := ast.Messages[0]
		require.Len(t, msg.SpokeDirectives, 1)
		assert.Equal(t, "option", msg.SpokeDirectives[0].Option)
		assert.Equal(t, "entity", msg.SpokeDirectives[0].Value)
	})

	t.Run("block comment with multiple directives", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

/*
 * @spoke:option:validated
 * @spoke:option:entity
 */
message TestMessage {
  string name = 1;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify message has both directives from block comment
		require.Len(t, ast.Messages, 1)
		msg := ast.Messages[0]
		require.Len(t, msg.SpokeDirectives, 2)
		assert.Equal(t, "option", msg.SpokeDirectives[0].Option)
		assert.Equal(t, "validated", msg.SpokeDirectives[0].Value)
		assert.Equal(t, "option", msg.SpokeDirectives[1].Option)
		assert.Equal(t, "entity", msg.SpokeDirectives[1].Value)
	})

	t.Run("field with block comment directive", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

message TestMessage {
  /* @spoke:option:required */
  string name = 1;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify field has directive from block comment
		require.Len(t, ast.Messages, 1)
		msg := ast.Messages[0]
		require.Len(t, msg.Fields, 1)
		field := msg.Fields[0]
		require.Len(t, field.SpokeDirectives, 1)
		assert.Equal(t, "option", field.SpokeDirectives[0].Option)
		assert.Equal(t, "required", field.SpokeDirectives[0].Value)
	})

	t.Run("mixed line and block comments", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

/* @spoke:option:validated */
// @spoke:option:entity
message TestMessage {
  string name = 1;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify message has directives from both comment types
		require.Len(t, ast.Messages, 1)
		msg := ast.Messages[0]
		require.Len(t, msg.SpokeDirectives, 2)

		// Collect directive values
		values := make([]string, len(msg.SpokeDirectives))
		for i, d := range msg.SpokeDirectives {
			values[i] = d.Value
		}

		// Both directives should be present (order may vary)
		assert.Contains(t, values, "validated")
		assert.Contains(t, values, "entity")
	})
}
