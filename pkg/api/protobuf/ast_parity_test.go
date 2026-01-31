package protobuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParserParity tests that the new descriptor-based parser produces
// identical AST output to the legacy manual parser.
func TestParserParity_Basic(t *testing.T) {
	content := `syntax = "proto3";

package test;

message User {
  string name = 1;
  int32 age = 2;
}
`

	// Parse with legacy parser
	legacyParser := NewStringParser(content)
	legacyAST, err := legacyParser.Parse()
	require.NoError(t, err)
	require.NotNil(t, legacyAST)

	// Parse with new descriptor parser
	newAST, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, newAST)

	// Debug output
	t.Logf("Legacy parser - Messages: %d", len(legacyAST.Messages))
	if len(legacyAST.Messages) > 0 {
		t.Logf("Legacy parser - First message: %s, Fields: %d", legacyAST.Messages[0].Name, len(legacyAST.Messages[0].Fields))
	}
	t.Logf("New parser - Messages: %d", len(newAST.Messages))
	if len(newAST.Messages) > 0 {
		t.Logf("New parser - First message: %s, Fields: %d", newAST.Messages[0].Name, len(newAST.Messages[0].Fields))
	}

	// Compare syntax
	assert.Equal(t, legacyAST.Syntax.Value, newAST.Syntax.Value)

	// Compare package
	assert.Equal(t, legacyAST.Package.Name, newAST.Package.Name)

	// Compare messages
	require.Len(t, newAST.Messages, len(legacyAST.Messages))
	for i := range legacyAST.Messages {
		legacyMsg := legacyAST.Messages[i]
		newMsg := newAST.Messages[i]

		assert.Equal(t, legacyMsg.Name, newMsg.Name, "Message name mismatch")
		assert.Len(t, newMsg.Fields, len(legacyMsg.Fields), "Field count mismatch")

		// Compare fields
		for j := range legacyMsg.Fields {
			legacyField := legacyMsg.Fields[j]
			newField := newMsg.Fields[j]

			assert.Equal(t, legacyField.Name, newField.Name, "Field name mismatch")
			assert.Equal(t, legacyField.Type, newField.Type, "Field type mismatch")
			assert.Equal(t, legacyField.Number, newField.Number, "Field number mismatch")
			assert.Equal(t, legacyField.Repeated, newField.Repeated, "Field repeated mismatch")
		}
	}
}

func TestParserParity_WithSpokeDirectives(t *testing.T) {
	content := `syntax = "proto3";

// @spoke:domain:github.com/example/test
package test;

// @spoke:option:entity
message User {
  // @spoke:option:required
  string name = 1;
  int32 age = 2;
}
`

	// Parse with legacy parser
	legacyParser := NewStringParser(content)
	legacyAST, err := legacyParser.Parse()
	require.NoError(t, err)
	require.NotNil(t, legacyAST)

	// Parse with new descriptor parser
	newAST, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, newAST)

	// Compare package directives
	require.Len(t, newAST.Package.SpokeDirectives, len(legacyAST.Package.SpokeDirectives))
	if len(legacyAST.Package.SpokeDirectives) > 0 {
		assert.Equal(t, legacyAST.Package.SpokeDirectives[0].Option, newAST.Package.SpokeDirectives[0].Option)
		assert.Equal(t, legacyAST.Package.SpokeDirectives[0].Value, newAST.Package.SpokeDirectives[0].Value)
	}

	// Compare message directives
	require.Len(t, newAST.Messages, len(legacyAST.Messages))
	for i := range legacyAST.Messages {
		legacyMsg := legacyAST.Messages[i]
		newMsg := newAST.Messages[i]

		assert.Len(t, newMsg.SpokeDirectives, len(legacyMsg.SpokeDirectives), "Message directive count mismatch")
		for j := range legacyMsg.SpokeDirectives {
			assert.Equal(t, legacyMsg.SpokeDirectives[j].Option, newMsg.SpokeDirectives[j].Option)
			assert.Equal(t, legacyMsg.SpokeDirectives[j].Value, newMsg.SpokeDirectives[j].Value)
		}

		// Compare field directives
		for j := range legacyMsg.Fields {
			legacyField := legacyMsg.Fields[j]
			newField := newMsg.Fields[j]

			assert.Len(t, newField.SpokeDirectives, len(legacyField.SpokeDirectives), "Field directive count mismatch")
			for k := range legacyField.SpokeDirectives {
				assert.Equal(t, legacyField.SpokeDirectives[k].Option, newField.SpokeDirectives[k].Option)
				assert.Equal(t, legacyField.SpokeDirectives[k].Value, newField.SpokeDirectives[k].Value)
			}
		}
	}
}

func TestParserParity_Enums(t *testing.T) {
	content := `syntax = "proto3";

package test;

enum Status {
  STATUS_UNKNOWN = 0;
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
}
`

	// Parse with legacy parser
	legacyParser := NewStringParser(content)
	legacyAST, err := legacyParser.Parse()
	require.NoError(t, err)
	require.NotNil(t, legacyAST)

	// Parse with new descriptor parser
	newAST, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, newAST)

	// Compare enums
	require.Len(t, newAST.Enums, len(legacyAST.Enums))
	for i := range legacyAST.Enums {
		legacyEnum := legacyAST.Enums[i]
		newEnum := newAST.Enums[i]

		assert.Equal(t, legacyEnum.Name, newEnum.Name)
		assert.Len(t, newEnum.Values, len(legacyEnum.Values))

		for j := range legacyEnum.Values {
			assert.Equal(t, legacyEnum.Values[j].Name, newEnum.Values[j].Name)
			assert.Equal(t, legacyEnum.Values[j].Number, newEnum.Values[j].Number)
		}
	}
}

func TestParserParity_Services(t *testing.T) {
	content := `syntax = "proto3";

package test;

message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  string name = 1;
}

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}
`

	// Parse with legacy parser
	legacyParser := NewStringParser(content)
	legacyAST, err := legacyParser.Parse()
	require.NoError(t, err)
	require.NotNil(t, legacyAST)

	// Parse with new descriptor parser
	newAST, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, newAST)

	// Compare services
	require.Len(t, newAST.Services, len(legacyAST.Services))
	for i := range legacyAST.Services {
		legacySvc := legacyAST.Services[i]
		newSvc := newAST.Services[i]

		assert.Equal(t, legacySvc.Name, newSvc.Name)
		assert.Len(t, newSvc.RPCs, len(legacySvc.RPCs))

		for j := range legacySvc.RPCs {
			legacyRPC := legacySvc.RPCs[j]
			newRPC := newSvc.RPCs[j]

			assert.Equal(t, legacyRPC.Name, newRPC.Name)
			assert.Equal(t, legacyRPC.InputType, newRPC.InputType)
			assert.Equal(t, legacyRPC.OutputType, newRPC.OutputType)
			assert.Equal(t, legacyRPC.ClientStreaming, newRPC.ClientStreaming)
			assert.Equal(t, legacyRPC.ServerStreaming, newRPC.ServerStreaming)
		}
	}
}

func TestParserParity_NestedMessages(t *testing.T) {
	content := `syntax = "proto3";

package test;

message Outer {
  string name = 1;

  message Inner {
    int32 value = 1;
  }

  Inner inner = 2;
}
`

	// Parse with legacy parser
	legacyParser := NewStringParser(content)
	legacyAST, err := legacyParser.Parse()
	require.NoError(t, err)
	require.NotNil(t, legacyAST)

	// Parse with new descriptor parser
	newAST, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, newAST)

	// Compare messages
	require.Len(t, newAST.Messages, len(legacyAST.Messages))
	legacyOuter := legacyAST.Messages[0]
	newOuter := newAST.Messages[0]

	assert.Equal(t, legacyOuter.Name, newOuter.Name)
	assert.Len(t, newOuter.Nested, len(legacyOuter.Nested))

	if len(legacyOuter.Nested) > 0 {
		legacyInner := legacyOuter.Nested[0]
		newInner := newOuter.Nested[0]

		assert.Equal(t, legacyInner.Name, newInner.Name)
		assert.Len(t, newInner.Fields, len(legacyInner.Fields))
	}
}

func TestParserParity_AllFieldTypes(t *testing.T) {
	content := `syntax = "proto3";

package test;

message AllTypes {
  double double_field = 1;
  float float_field = 2;
  int32 int32_field = 3;
  int64 int64_field = 4;
  uint32 uint32_field = 5;
  uint64 uint64_field = 6;
  sint32 sint32_field = 7;
  sint64 sint64_field = 8;
  fixed32 fixed32_field = 9;
  fixed64 fixed64_field = 10;
  sfixed32 sfixed32_field = 11;
  sfixed64 sfixed64_field = 12;
  bool bool_field = 13;
  string string_field = 14;
  bytes bytes_field = 15;
}
`

	// Parse with legacy parser
	legacyParser := NewStringParser(content)
	legacyAST, err := legacyParser.Parse()
	require.NoError(t, err)
	require.NotNil(t, legacyAST)

	// Parse with new descriptor parser
	newAST, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, newAST)

	// Compare all field types
	require.Len(t, newAST.Messages, 1)
	require.Len(t, legacyAST.Messages, 1)

	legacyMsg := legacyAST.Messages[0]
	newMsg := newAST.Messages[0]

	require.Len(t, newMsg.Fields, len(legacyMsg.Fields))
	for i := range legacyMsg.Fields {
		assert.Equal(t, legacyMsg.Fields[i].Type, newMsg.Fields[i].Type, "Field type mismatch at index %d", i)
	}
}

func TestParserParity_RepeatedFields(t *testing.T) {
	content := `syntax = "proto3";

package test;

message User {
  repeated string emails = 1;
  string name = 2;
}
`

	// Parse with legacy parser
	legacyParser := NewStringParser(content)
	legacyAST, err := legacyParser.Parse()
	require.NoError(t, err)
	require.NotNil(t, legacyAST)

	// Parse with new descriptor parser
	newAST, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, newAST)

	// Compare repeated fields
	require.Len(t, newAST.Messages, 1)
	require.Len(t, legacyAST.Messages, 1)

	legacyMsg := legacyAST.Messages[0]
	newMsg := newAST.Messages[0]

	require.Len(t, newMsg.Fields, len(legacyMsg.Fields))
	for i := range legacyMsg.Fields {
		assert.Equal(t, legacyMsg.Fields[i].Repeated, newMsg.Fields[i].Repeated, "Field repeated mismatch at index %d", i)
	}
}
