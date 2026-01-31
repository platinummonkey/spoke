package protobuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWithDescriptor_Basic(t *testing.T) {
	content := `syntax = "proto3";

package test;

message User {
  string name = 1;
  int32 age = 2;
}
`

	ast, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify syntax
	require.NotNil(t, ast.Syntax)
	assert.Equal(t, "proto3", ast.Syntax.Value)

	// Verify package
	require.NotNil(t, ast.Package)
	assert.Equal(t, "test", ast.Package.Name)

	// Verify message
	require.Len(t, ast.Messages, 1)
	msg := ast.Messages[0]
	assert.Equal(t, "User", msg.Name)

	// Verify fields
	require.Len(t, msg.Fields, 2)
	assert.Equal(t, "name", msg.Fields[0].Name)
	assert.Equal(t, "string", msg.Fields[0].Type)
	assert.Equal(t, 1, msg.Fields[0].Number)

	assert.Equal(t, "age", msg.Fields[1].Name)
	assert.Equal(t, "int32", msg.Fields[1].Type)
	assert.Equal(t, 2, msg.Fields[1].Number)
}

func TestParseWithDescriptor_WithSpokeDirectives(t *testing.T) {
	content := `syntax = "proto3";

// @spoke:domain:github.com/example/test
package test;

message User {
  // @spoke:option:required
  string name = 1;
  int32 age = 2;
}
`

	ast, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify package has spoke directive
	require.NotNil(t, ast.Package)
	require.Len(t, ast.Package.SpokeDirectives, 1)
	assert.Equal(t, "domain", ast.Package.SpokeDirectives[0].Option)
	assert.Equal(t, "github.com/example/test", ast.Package.SpokeDirectives[0].Value)

	// Verify field has spoke directive
	require.Len(t, ast.Messages, 1)
	msg := ast.Messages[0]
	require.Len(t, msg.Fields, 2)

	field := msg.Fields[0]
	require.Len(t, field.SpokeDirectives, 1)
	assert.Equal(t, "option", field.SpokeDirectives[0].Option)
	assert.Equal(t, "required", field.SpokeDirectives[0].Value)
}

func TestParseWithDescriptor_Enums(t *testing.T) {
	content := `syntax = "proto3";

package test;

enum Status {
  STATUS_UNKNOWN = 0;
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
}
`

	ast, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify enum
	require.Len(t, ast.Enums, 1)
	enum := ast.Enums[0]
	assert.Equal(t, "Status", enum.Name)

	// Verify enum values
	require.Len(t, enum.Values, 3)
	assert.Equal(t, "STATUS_UNKNOWN", enum.Values[0].Name)
	assert.Equal(t, 0, enum.Values[0].Number)

	assert.Equal(t, "STATUS_ACTIVE", enum.Values[1].Name)
	assert.Equal(t, 1, enum.Values[1].Number)

	assert.Equal(t, "STATUS_INACTIVE", enum.Values[2].Name)
	assert.Equal(t, 2, enum.Values[2].Number)
}

func TestParseWithDescriptor_Services(t *testing.T) {
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

	ast, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify messages
	require.Len(t, ast.Messages, 2)

	// Verify service
	require.Len(t, ast.Services, 1)
	svc := ast.Services[0]
	assert.Equal(t, "UserService", svc.Name)

	// Verify RPC
	require.Len(t, svc.RPCs, 1)
	rpc := svc.RPCs[0]
	assert.Equal(t, "GetUser", rpc.Name)
	assert.Equal(t, "test.GetUserRequest", rpc.InputType)
	assert.Equal(t, "test.GetUserResponse", rpc.OutputType)
	assert.False(t, rpc.ClientStreaming)
	assert.False(t, rpc.ServerStreaming)
}

func TestParseWithDescriptor_Imports(t *testing.T) {
	content := `syntax = "proto3";

package test;

import "google/protobuf/timestamp.proto";
import public "common/common.proto";

message User {
  string name = 1;
}
`

	ast, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify imports
	require.Len(t, ast.Imports, 2)

	assert.Equal(t, "google/protobuf/timestamp.proto", ast.Imports[0].Path)
	assert.False(t, ast.Imports[0].Public)

	assert.Equal(t, "common/common.proto", ast.Imports[1].Path)
	assert.True(t, ast.Imports[1].Public)
}

func TestParseWithDescriptor_NestedMessages(t *testing.T) {
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

	ast, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify outer message
	require.Len(t, ast.Messages, 1)
	outer := ast.Messages[0]
	assert.Equal(t, "Outer", outer.Name)

	// Verify nested message
	require.Len(t, outer.Nested, 1)
	inner := outer.Nested[0]
	assert.Equal(t, "Inner", inner.Name)

	// Verify inner message has field
	require.Len(t, inner.Fields, 1)
	assert.Equal(t, "value", inner.Fields[0].Name)
	assert.Equal(t, "int32", inner.Fields[0].Type)
}

func TestParseWithDescriptor_FieldTypes(t *testing.T) {
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

	ast, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify message
	require.Len(t, ast.Messages, 1)
	msg := ast.Messages[0]

	// Verify all field types
	require.Len(t, msg.Fields, 15)

	expectedTypes := []string{
		"double", "float", "int32", "int64", "uint32", "uint64",
		"sint32", "sint64", "fixed32", "fixed64", "sfixed32", "sfixed64",
		"bool", "string", "bytes",
	}

	for i, expectedType := range expectedTypes {
		assert.Equal(t, expectedType, msg.Fields[i].Type, "Field %d type mismatch", i)
	}
}

func TestParseWithDescriptor_RepeatedFields(t *testing.T) {
	content := `syntax = "proto3";

package test;

message User {
  repeated string emails = 1;
  string name = 2;
}
`

	ast, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify message
	require.Len(t, ast.Messages, 1)
	msg := ast.Messages[0]

	// Verify fields
	require.Len(t, msg.Fields, 2)

	// First field should be repeated
	assert.True(t, msg.Fields[0].Repeated)
	assert.Equal(t, "emails", msg.Fields[0].Name)

	// Second field should not be repeated
	assert.False(t, msg.Fields[1].Repeated)
	assert.Equal(t, "name", msg.Fields[1].Name)
}

func TestParseWithDescriptor_ComplexWithDirectives(t *testing.T) {
	content := `syntax = "proto3";

// @spoke:domain:github.com/example/userservice
package userservice;

import "google/protobuf/timestamp.proto";

// User represents a user in the system
// @spoke:option:entity
message User {
  // @spoke:option:required
  string id = 1;

  // @spoke:option:required
  string name = 2;

  string email = 3;

  // @spoke:option:immutable
  google.protobuf.Timestamp created_at = 4;
}

// Status enum
enum UserStatus {
  USER_STATUS_UNKNOWN = 0;
  USER_STATUS_ACTIVE = 1;
  USER_STATUS_INACTIVE = 2;
}

// @spoke:option:service
service UserService {
  // @spoke:option:authenticated
  rpc GetUser(GetUserRequest) returns (User);
}

message GetUserRequest {
  string id = 1;
}
`

	ast, err := ParseWithDescriptor("test.proto", content)
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify package directive
	require.NotNil(t, ast.Package)
	require.Len(t, ast.Package.SpokeDirectives, 1)
	assert.Equal(t, "domain", ast.Package.SpokeDirectives[0].Option)
	assert.Equal(t, "github.com/example/userservice", ast.Package.SpokeDirectives[0].Value)

	// Verify message directive
	require.Len(t, ast.Messages, 2)
	userMsg := ast.Messages[0]
	require.Len(t, userMsg.SpokeDirectives, 1)
	assert.Equal(t, "option", userMsg.SpokeDirectives[0].Option)
	assert.Equal(t, "entity", userMsg.SpokeDirectives[0].Value)

	// Verify field directives
	require.Len(t, userMsg.Fields, 4)

	// id field should have required directive
	assert.Equal(t, "id", userMsg.Fields[0].Name)
	require.Len(t, userMsg.Fields[0].SpokeDirectives, 1)
	assert.Equal(t, "required", userMsg.Fields[0].SpokeDirectives[0].Value)

	// name field should have required directive
	assert.Equal(t, "name", userMsg.Fields[1].Name)
	require.Len(t, userMsg.Fields[1].SpokeDirectives, 1)
	assert.Equal(t, "required", userMsg.Fields[1].SpokeDirectives[0].Value)

	// email field should have no directive
	assert.Equal(t, "email", userMsg.Fields[2].Name)
	assert.Len(t, userMsg.Fields[2].SpokeDirectives, 0)

	// created_at field should have immutable directive
	assert.Equal(t, "created_at", userMsg.Fields[3].Name)
	require.Len(t, userMsg.Fields[3].SpokeDirectives, 1)
	assert.Equal(t, "immutable", userMsg.Fields[3].SpokeDirectives[0].Value)

	// Verify enum
	require.Len(t, ast.Enums, 1)
	assert.Equal(t, "UserStatus", ast.Enums[0].Name)

	// Verify service directive
	require.Len(t, ast.Services, 1)
	svc := ast.Services[0]
	require.Len(t, svc.SpokeDirectives, 1)
	assert.Equal(t, "service", svc.SpokeDirectives[0].Value)

	// Verify RPC directive
	require.Len(t, svc.RPCs, 1)
	rpc := svc.RPCs[0]
	require.Len(t, rpc.SpokeDirectives, 1)
	assert.Equal(t, "authenticated", rpc.SpokeDirectives[0].Value)
}

func TestParseWithDescriptor_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "valid proto",
			content: `syntax = "proto3";
package test;
message User { string name = 1; }`,
			expectError: false,
		},
		{
			name: "syntax error",
			content: `syntax = "proto3"
package test
message User { string name = 1 }`, // missing semicolons
			expectError: true,
		},
		{
			name: "invalid field number",
			content: `syntax = "proto3";
package test;
message User { string name = 0; }`, // field number 0 is invalid
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseWithDescriptor("test.proto", tt.content)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestParseWithFallback tests the fallback mechanism
func TestParseWithFallback(t *testing.T) {
	content := `syntax = "proto3";

package test;

message User {
  string name = 1;
}
`

	// Test with descriptor parser disabled (default)
	UseDescriptorParser = false
	ast, err := ParseWithFallback(content)
	require.NoError(t, err)
	require.NotNil(t, ast)
	assert.Equal(t, "test", ast.Package.Name)

	// Test with descriptor parser enabled
	UseDescriptorParser = true
	ast, err = ParseWithFallback(content)
	require.NoError(t, err)
	require.NotNil(t, ast)
	assert.Equal(t, "test", ast.Package.Name)

	// Reset to default
	UseDescriptorParser = false
}
