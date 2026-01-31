package protobuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWithDescriptor_OneOf(t *testing.T) {
	t.Run("basic oneof", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

message TestMessage {
  oneof test_oneof {
    string name = 1;
    int32 id = 2;
  }
  string other_field = 3;
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify message exists
		require.Len(t, ast.Messages, 1)
		msg := ast.Messages[0]
		assert.Equal(t, "TestMessage", msg.Name)

		// Verify oneof exists
		require.Len(t, msg.OneOfs, 1)
		oneof := msg.OneOfs[0]
		assert.Equal(t, "test_oneof", oneof.Name)

		// Verify oneof has the correct fields
		require.Len(t, oneof.Fields, 2)
		assert.Equal(t, "name", oneof.Fields[0].Name)
		assert.Equal(t, "string", oneof.Fields[0].Type)
		assert.Equal(t, 1, oneof.Fields[0].Number)

		assert.Equal(t, "id", oneof.Fields[1].Name)
		assert.Equal(t, "int32", oneof.Fields[1].Type)
		assert.Equal(t, 2, oneof.Fields[1].Number)

		// Verify regular field is still present in message fields
		require.Len(t, msg.Fields, 3)
		assert.Equal(t, "other_field", msg.Fields[2].Name)
	})

	t.Run("multiple oneofs", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

message TestMessage {
  oneof first {
    string name = 1;
    int32 id = 2;
  }

  string regular = 3;

  oneof second {
    bool active = 4;
    double value = 5;
  }
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify message exists
		require.Len(t, ast.Messages, 1)
		msg := ast.Messages[0]

		// Verify both oneofs exist
		require.Len(t, msg.OneOfs, 2)

		// First oneof
		assert.Equal(t, "first", msg.OneOfs[0].Name)
		require.Len(t, msg.OneOfs[0].Fields, 2)
		assert.Equal(t, "name", msg.OneOfs[0].Fields[0].Name)
		assert.Equal(t, "id", msg.OneOfs[0].Fields[1].Name)

		// Second oneof
		assert.Equal(t, "second", msg.OneOfs[1].Name)
		require.Len(t, msg.OneOfs[1].Fields, 2)
		assert.Equal(t, "active", msg.OneOfs[1].Fields[0].Name)
		assert.Equal(t, "value", msg.OneOfs[1].Fields[1].Name)

		// Verify all fields are in message fields list
		require.Len(t, msg.Fields, 5)
	})

	t.Run("oneof with spoke directives", func(t *testing.T) {
		content := `syntax = "proto3";

package test;

message TestMessage {
  // @spoke:option:validated
  oneof test_oneof {
    // @spoke:option:required
    string name = 1;
    int32 id = 2;
  }
}`

		ast, err := ParseWithDescriptor("test.proto", content)
		require.NoError(t, err)
		require.NotNil(t, ast)

		// Verify oneof directive
		require.Len(t, ast.Messages, 1)
		msg := ast.Messages[0]
		require.Len(t, msg.OneOfs, 1)
		oneof := msg.OneOfs[0]

		require.Len(t, oneof.SpokeDirectives, 1)
		assert.Equal(t, "option", oneof.SpokeDirectives[0].Option)
		assert.Equal(t, "validated", oneof.SpokeDirectives[0].Value)

		// Verify field directive within oneof
		require.Len(t, oneof.Fields, 2)
		require.Len(t, oneof.Fields[0].SpokeDirectives, 1)
		assert.Equal(t, "required", oneof.Fields[0].SpokeDirectives[0].Value)
	})
}
