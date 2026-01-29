package validation

import (
	"strings"
	"testing"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

func TestNewSerializer(t *testing.T) {
	config := DefaultNormalizationConfig()
	serializer := NewSerializer(config)

	if serializer == nil {
		t.Error("NewSerializer returned nil")
	}

	if serializer.config != config {
		t.Error("Serializer config not set correctly")
	}

	if serializer.indent != "  " {
		t.Errorf("Serializer indent = %q, want %q", serializer.indent, "  ")
	}
}

func TestSerializer_SerializeBasic(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	tests := []struct {
		name     string
		ast      *protobuf.RootNode
		contains []string
	}{
		{
			name: "syntax only",
			ast: &protobuf.RootNode{
				Syntax: &protobuf.SyntaxNode{Value: "proto3"},
			},
			contains: []string{"syntax = \"proto3\";"},
		},
		{
			name: "package only",
			ast: &protobuf.RootNode{
				Package: &protobuf.PackageNode{Name: "com.example.api"},
			},
			contains: []string{"package com.example.api;"},
		},
		{
			name: "syntax and package",
			ast: &protobuf.RootNode{
				Syntax:  &protobuf.SyntaxNode{Value: "proto3"},
				Package: &protobuf.PackageNode{Name: "com.example.api"},
			},
			contains: []string{
				"syntax = \"proto3\";",
				"package com.example.api;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := serializer.Serialize(tt.ast)
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Serialize() result missing %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestSerializer_SerializeImports(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	tests := []struct {
		name     string
		imports  []*protobuf.ImportNode
		contains []string
	}{
		{
			name: "regular import",
			imports: []*protobuf.ImportNode{
				{Path: "google/protobuf/timestamp.proto"},
			},
			contains: []string{`import "google/protobuf/timestamp.proto";`},
		},
		{
			name: "public import",
			imports: []*protobuf.ImportNode{
				{Path: "google/protobuf/any.proto", Public: true},
			},
			contains: []string{`import public "google/protobuf/any.proto";`},
		},
		{
			name: "weak import",
			imports: []*protobuf.ImportNode{
				{Path: "google/protobuf/descriptor.proto", Weak: true},
			},
			contains: []string{`import weak "google/protobuf/descriptor.proto";`},
		},
		{
			name: "multiple imports",
			imports: []*protobuf.ImportNode{
				{Path: "a.proto"},
				{Path: "b.proto", Public: true},
				{Path: "c.proto", Weak: true},
			},
			contains: []string{
				`import "a.proto";`,
				`import public "b.proto";`,
				`import weak "c.proto";`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Imports: tt.imports,
			}

			result, err := serializer.Serialize(ast)
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Serialize() result missing %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestSerializer_SerializeOptions(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	ast := &protobuf.RootNode{
		Options: []*protobuf.OptionNode{
			{Name: "go_package", Value: "\"github.com/example/api\""},
			{Name: "java_multiple_files", Value: "true"},
		},
	}

	result, err := serializer.Serialize(ast)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	expected := []string{
		"option go_package = \"github.com/example/api\";",
		"option java_multiple_files = true;",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Serialize() result missing %q\nGot: %s", exp, result)
		}
	}
}

func TestSerializer_SerializeMessage(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	tests := []struct {
		name     string
		message  *protobuf.MessageNode
		contains []string
	}{
		{
			name: "simple message",
			message: &protobuf.MessageNode{
				Name: "User",
				Fields: []*protobuf.FieldNode{
					{Name: "id", Type: "int32", Number: 1},
					{Name: "name", Type: "string", Number: 2},
				},
			},
			contains: []string{
				"message User {",
				"int32 id = 1;",
				"string name = 2;",
				"}",
			},
		},
		{
			name: "message with repeated field",
			message: &protobuf.MessageNode{
				Name: "Users",
				Fields: []*protobuf.FieldNode{
					{Name: "items", Type: "User", Number: 1, Repeated: true},
				},
			},
			contains: []string{
				"message Users {",
				"repeated User items = 1;",
			},
		},
		{
			name: "message with optional field",
			message: &protobuf.MessageNode{
				Name: "Profile",
				Fields: []*protobuf.FieldNode{
					{Name: "bio", Type: "string", Number: 1, Optional: true},
				},
			},
			contains: []string{
				"message Profile {",
				"optional string bio = 1;",
			},
		},
		{
			name: "message with required field",
			message: &protobuf.MessageNode{
				Name: "Required",
				Fields: []*protobuf.FieldNode{
					{Name: "id", Type: "int32", Number: 1, Required: true},
				},
			},
			contains: []string{
				"message Required {",
				"required int32 id = 1;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Messages: []*protobuf.MessageNode{tt.message},
			}

			result, err := serializer.Serialize(ast)
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Serialize() result missing %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestSerializer_SerializeNestedMessage(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "Outer",
				Fields: []*protobuf.FieldNode{
					{Name: "id", Type: "int32", Number: 1},
				},
				Nested: []*protobuf.MessageNode{
					{
						Name: "Inner",
						Fields: []*protobuf.FieldNode{
							{Name: "value", Type: "string", Number: 1},
						},
					},
				},
			},
		},
	}

	result, err := serializer.Serialize(ast)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	expected := []string{
		"message Outer {",
		"message Inner {",
		"string value = 1;",
		"int32 id = 1;",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Serialize() result missing %q\nGot: %s", exp, result)
		}
	}
}

func TestSerializer_SerializeEnum(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	tests := []struct {
		name     string
		enum     *protobuf.EnumNode
		contains []string
	}{
		{
			name: "simple enum",
			enum: &protobuf.EnumNode{
				Name: "Status",
				Values: []*protobuf.EnumValueNode{
					{Name: "UNKNOWN", Number: 0},
					{Name: "ACTIVE", Number: 1},
					{Name: "INACTIVE", Number: 2},
				},
			},
			contains: []string{
				"enum Status {",
				"UNKNOWN = 0;",
				"ACTIVE = 1;",
				"INACTIVE = 2;",
				"}",
			},
		},
		{
			name: "enum with options",
			enum: &protobuf.EnumNode{
				Name: "Priority",
				Values: []*protobuf.EnumValueNode{
					{Name: "LOW", Number: 0},
					{Name: "HIGH", Number: 1},
				},
				Options: []*protobuf.OptionNode{
					{Name: "allow_alias", Value: "true"},
				},
			},
			contains: []string{
				"enum Priority {",
				"LOW = 0;",
				"HIGH = 1;",
				"option allow_alias = true;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Enums: []*protobuf.EnumNode{tt.enum},
			}

			result, err := serializer.Serialize(ast)
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Serialize() result missing %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestSerializer_SerializeNestedEnum(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "User",
				Fields: []*protobuf.FieldNode{
					{Name: "status", Type: "Status", Number: 1},
				},
				Enums: []*protobuf.EnumNode{
					{
						Name: "Status",
						Values: []*protobuf.EnumValueNode{
							{Name: "ACTIVE", Number: 0},
							{Name: "INACTIVE", Number: 1},
						},
					},
				},
			},
		},
	}

	result, err := serializer.Serialize(ast)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	expected := []string{
		"message User {",
		"enum Status {",
		"ACTIVE = 0;",
		"INACTIVE = 1;",
		"Status status = 1;",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Serialize() result missing %q\nGot: %s", exp, result)
		}
	}
}

func TestSerializer_SerializeOneOf(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "Payment",
				OneOfs: []*protobuf.OneOfNode{
					{
						Name: "payment_method",
						Fields: []*protobuf.FieldNode{
							{Name: "cash", Type: "CashPayment", Number: 1},
							{Name: "credit_card", Type: "CreditCardPayment", Number: 2},
						},
					},
				},
			},
		},
	}

	result, err := serializer.Serialize(ast)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	expected := []string{
		"message Payment {",
		"oneof payment_method {",
		"CashPayment cash = 1;",
		"CreditCardPayment credit_card = 2;",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Serialize() result missing %q\nGot: %s", exp, result)
		}
	}
}

func TestSerializer_SerializeService(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	tests := []struct {
		name     string
		service  *protobuf.ServiceNode
		contains []string
	}{
		{
			name: "simple service",
			service: &protobuf.ServiceNode{
				Name: "UserService",
				RPCs: []*protobuf.RPCNode{
					{
						Name:       "GetUser",
						InputType:  "GetUserRequest",
						OutputType: "GetUserResponse",
					},
				},
			},
			contains: []string{
				"service UserService {",
				"rpc GetUser(GetUserRequest) returns (GetUserResponse);",
				"}",
			},
		},
		{
			name: "service with client streaming",
			service: &protobuf.ServiceNode{
				Name: "DataService",
				RPCs: []*protobuf.RPCNode{
					{
						Name:            "Upload",
						InputType:       "DataChunk",
						OutputType:      "UploadResponse",
						ClientStreaming: true,
					},
				},
			},
			contains: []string{
				"service DataService {",
				"rpc Upload(stream DataChunk) returns (UploadResponse);",
			},
		},
		{
			name: "service with server streaming",
			service: &protobuf.ServiceNode{
				Name: "StreamService",
				RPCs: []*protobuf.RPCNode{
					{
						Name:            "Download",
						InputType:       "DownloadRequest",
						OutputType:      "DataChunk",
						ServerStreaming: true,
					},
				},
			},
			contains: []string{
				"service StreamService {",
				"rpc Download(DownloadRequest) returns (stream DataChunk);",
			},
		},
		{
			name: "service with bidirectional streaming",
			service: &protobuf.ServiceNode{
				Name: "ChatService",
				RPCs: []*protobuf.RPCNode{
					{
						Name:            "Chat",
						InputType:       "ChatMessage",
						OutputType:      "ChatMessage",
						ClientStreaming: true,
						ServerStreaming: true,
					},
				},
			},
			contains: []string{
				"service ChatService {",
				"rpc Chat(stream ChatMessage) returns (stream ChatMessage);",
			},
		},
		{
			name: "service with options",
			service: &protobuf.ServiceNode{
				Name: "ApiService",
				RPCs: []*protobuf.RPCNode{
					{
						Name:       "GetData",
						InputType:  "Request",
						OutputType: "Response",
					},
				},
				Options: []*protobuf.OptionNode{
					{Name: "deprecated", Value: "true"},
				},
			},
			contains: []string{
				"service ApiService {",
				"rpc GetData(Request) returns (Response);",
				"option deprecated = true;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Services: []*protobuf.ServiceNode{tt.service},
			}

			result, err := serializer.Serialize(ast)
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Serialize() result missing %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestSerializer_RemoveTrailingWhitespace(t *testing.T) {
	tests := []struct {
		name                     string
		removeTrailingWhitespace bool
		wantNoTrailingSpaces     bool
	}{
		{
			name:                     "remove trailing whitespace enabled",
			removeTrailingWhitespace: true,
			wantNoTrailingSpaces:     true,
		},
		{
			name:                     "remove trailing whitespace disabled",
			removeTrailingWhitespace: false,
			wantNoTrailingSpaces:     false, // May have trailing spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &NormalizationConfig{
				RemoveTrailingWhitespace: tt.removeTrailingWhitespace,
			}
			serializer := NewSerializer(config)

			ast := &protobuf.RootNode{
				Syntax:  &protobuf.SyntaxNode{Value: "proto3"},
				Package: &protobuf.PackageNode{Name: "test"},
				Messages: []*protobuf.MessageNode{
					{
						Name: "Test",
						Fields: []*protobuf.FieldNode{
							{Name: "id", Type: "int32", Number: 1},
						},
					},
				},
			}

			result, err := serializer.Serialize(ast)
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			lines := strings.Split(result, "\n")
			hasTrailingSpaces := false
			for _, line := range lines {
				if len(line) > 0 && (strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t")) {
					hasTrailingSpaces = true
					break
				}
			}

			if tt.wantNoTrailingSpaces && hasTrailingSpaces {
				t.Error("Expected no trailing whitespace but found some")
			}
		})
	}
}

func TestSerializer_Indentation(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	tests := []struct {
		name  string
		depth int
		want  string
	}{
		{"depth 0", 0, ""},
		{"depth 1", 1, "  "},
		{"depth 2", 2, "    "},
		{"depth 3", 3, "      "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializer.indentation(tt.depth)
			if got != tt.want {
				t.Errorf("indentation(%d) = %q, want %q", tt.depth, got, tt.want)
			}
		})
	}
}

func TestSerializer_ComplexProtobuf(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: true,
	}
	serializer := NewSerializer(config)

	ast := &protobuf.RootNode{
		Syntax:  &protobuf.SyntaxNode{Value: "proto3"},
		Package: &protobuf.PackageNode{Name: "com.example.api"},
		Imports: []*protobuf.ImportNode{
			{Path: "google/protobuf/timestamp.proto"},
			{Path: "google/protobuf/empty.proto", Public: true},
		},
		Options: []*protobuf.OptionNode{
			{Name: "go_package", Value: "\"github.com/example/api\""},
		},
		Messages: []*protobuf.MessageNode{
			{
				Name: "User",
				Fields: []*protobuf.FieldNode{
					{Name: "id", Type: "int32", Number: 1},
					{Name: "email", Type: "string", Number: 2},
					{Name: "status", Type: "Status", Number: 3},
				},
				Enums: []*protobuf.EnumNode{
					{
						Name: "Status",
						Values: []*protobuf.EnumValueNode{
							{Name: "ACTIVE", Number: 0},
							{Name: "INACTIVE", Number: 1},
						},
					},
				},
			},
		},
		Enums: []*protobuf.EnumNode{
			{
				Name: "Role",
				Values: []*protobuf.EnumValueNode{
					{Name: "ADMIN", Number: 0},
					{Name: "USER", Number: 1},
				},
			},
		},
		Services: []*protobuf.ServiceNode{
			{
				Name: "UserService",
				RPCs: []*protobuf.RPCNode{
					{
						Name:       "GetUser",
						InputType:  "GetUserRequest",
						OutputType: "User",
					},
					{
						Name:            "ListUsers",
						InputType:       "ListUsersRequest",
						OutputType:      "User",
						ServerStreaming: true,
					},
				},
			},
		},
	}

	result, err := serializer.Serialize(ast)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	expected := []string{
		"syntax = \"proto3\";",
		"package com.example.api;",
		"import \"google/protobuf/timestamp.proto\";",
		"import public \"google/protobuf/empty.proto\";",
		"option go_package = \"github.com/example/api\";",
		"message User {",
		"enum Status {",
		"ACTIVE = 0;",
		"INACTIVE = 1;",
		"int32 id = 1;",
		"string email = 2;",
		"Status status = 3;",
		"enum Role {",
		"ADMIN = 0;",
		"USER = 1;",
		"service UserService {",
		"rpc GetUser(GetUserRequest) returns (User);",
		"rpc ListUsers(ListUsersRequest) returns (stream User);",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Serialize() result missing %q\nGot: %s", exp, result)
		}
	}
}

func TestSerializer_MessageWithOptions(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "DeprecatedMessage",
				Fields: []*protobuf.FieldNode{
					{Name: "id", Type: "int32", Number: 1},
				},
				Options: []*protobuf.OptionNode{
					{Name: "deprecated", Value: "true"},
				},
			},
		},
	}

	result, err := serializer.Serialize(ast)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	expected := []string{
		"message DeprecatedMessage {",
		"int32 id = 1;",
		"option deprecated = true;",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Serialize() result missing %q\nGot: %s", exp, result)
		}
	}
}

func TestSerializer_EmptyAST(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	ast := &protobuf.RootNode{}

	result, err := serializer.Serialize(ast)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Empty AST should produce empty or minimal output
	if result != "" && result != "\n" {
		t.Logf("Empty AST produced: %q", result)
	}
}

func TestSerializer_MultipleMessagesAndEnums(t *testing.T) {
	config := &NormalizationConfig{
		RemoveTrailingWhitespace: false,
	}
	serializer := NewSerializer(config)

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "Message1",
				Fields: []*protobuf.FieldNode{
					{Name: "field1", Type: "string", Number: 1},
				},
			},
			{
				Name: "Message2",
				Fields: []*protobuf.FieldNode{
					{Name: "field2", Type: "int32", Number: 1},
				},
			},
		},
		Enums: []*protobuf.EnumNode{
			{
				Name: "Enum1",
				Values: []*protobuf.EnumValueNode{
					{Name: "VALUE1", Number: 0},
				},
			},
			{
				Name: "Enum2",
				Values: []*protobuf.EnumValueNode{
					{Name: "VALUE2", Number: 0},
				},
			},
		},
	}

	result, err := serializer.Serialize(ast)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	expected := []string{
		"message Message1 {",
		"string field1 = 1;",
		"message Message2 {",
		"int32 field2 = 1;",
		"enum Enum1 {",
		"VALUE1 = 0;",
		"enum Enum2 {",
		"VALUE2 = 0;",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Serialize() result missing %q\nGot: %s", exp, result)
		}
	}
}
