package docs

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

func TestGenerator_Generate(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{
		Syntax: &protobuf.SyntaxNode{Value: "proto3"},
		Package: &protobuf.PackageNode{Name: "test.package"},
		Messages: []*protobuf.MessageNode{
			{
				Name: "TestMessage",
				Fields: []*protobuf.FieldNode{
					{Name: "id", Number: 1, Type: "string"},
					{Name: "name", Number: 2, Type: "string"},
				},
			},
		},
	}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if doc.PackageName != "test.package" {
		t.Errorf("Expected package name 'test.package', got %s", doc.PackageName)
	}

	if doc.Syntax != "proto3" {
		t.Errorf("Expected syntax 'proto3', got %s", doc.Syntax)
	}

	if len(doc.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(doc.Messages))
	}

	msg := doc.Messages[0]
	if msg.Name != "TestMessage" {
		t.Errorf("Expected message name 'TestMessage', got %s", msg.Name)
	}

	if len(msg.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(msg.Fields))
	}
}

func TestGenerator_GenerateWithEnums(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Enums: []*protobuf.EnumNode{
			{
				Name: "Status",
				Values: []*protobuf.EnumValueNode{
					{Name: "UNKNOWN", Number: 0},
					{Name: "ACTIVE", Number: 1},
				},
			},
		},
	}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(doc.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(doc.Enums))
	}

	enum := doc.Enums[0]
	if enum.Name != "Status" {
		t.Errorf("Expected enum name 'Status', got %s", enum.Name)
	}

	if len(enum.Values) != 2 {
		t.Fatalf("Expected 2 enum values, got %d", len(enum.Values))
	}
}

func TestGenerator_GenerateWithServices(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Services: []*protobuf.ServiceNode{
			{
				Name: "UserService",
				RPCs: []*protobuf.RPCNode{
					{
						Name:       "GetUser",
						InputType:  "GetUserRequest",
						OutputType: "User",
					},
				},
			},
		},
	}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(doc.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(doc.Services))
	}

	svc := doc.Services[0]
	if svc.Name != "UserService" {
		t.Errorf("Expected service name 'UserService', got %s", svc.Name)
	}

	if len(svc.Methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(svc.Methods))
	}

	method := svc.Methods[0]
	if method.Name != "GetUser" {
		t.Errorf("Expected method name 'GetUser', got %s", method.Name)
	}
	if method.RequestType != "GetUserRequest" {
		t.Errorf("Expected request type 'GetUserRequest', got %s", method.RequestType)
	}
}

func TestDocumentation_FindMessage(t *testing.T) {
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name:     "User",
				FullName: "test.package.User",
			},
			{
				Name:     "Post",
				FullName: "test.package.Post",
				NestedTypes: []*MessageDoc{
					{
						Name:     "Comment",
						FullName: "test.package.Post.Comment",
					},
				},
			},
		},
	}

	t.Run("find top-level message", func(t *testing.T) {
		msg := doc.FindMessage("User")
		if msg == nil {
			t.Fatal("Expected to find User message")
		}
		if msg.Name != "User" {
			t.Errorf("Expected name 'User', got %s", msg.Name)
		}
	})

	t.Run("find nested message", func(t *testing.T) {
		msg := doc.FindMessage("Comment")
		if msg == nil {
			t.Fatal("Expected to find Comment message")
		}
		if msg.Name != "Comment" {
			t.Errorf("Expected name 'Comment', got %s", msg.Name)
		}
	})

	t.Run("message not found", func(t *testing.T) {
		msg := doc.FindMessage("NonExistent")
		if msg != nil {
			t.Error("Expected not to find NonExistent message")
		}
	})
}

func TestExtractComments(t *testing.T) {
	comments := []*protobuf.CommentNode{
		{Text: "// This is a comment"},
		{Text: "// Second line"},
	}

	result := extractComments(comments)
	if result == "" {
		t.Error("Expected comments to be extracted")
	}
}

func TestDocumentation_Summary(t *testing.T) {
	doc := &Documentation{
		PackageName: "test.package",
		Messages:    []*MessageDoc{{Name: "User"}},
		Enums:       []*EnumDoc{{Name: "Status"}},
		Services:    []*ServiceDoc{{Name: "UserService"}},
	}

	summary := doc.Summary()
	if summary == "" {
		t.Error("Expected summary to be generated")
	}

	expected := "Package: test.package, Messages: 1, Enums: 1, Services: 1"
	if summary != expected {
		t.Errorf("Expected summary %q, got %q", expected, summary)
	}
}

func TestDocumentation_FindEnum(t *testing.T) {
	doc := &Documentation{
		PackageName: "test.package",
		Enums: []*EnumDoc{
			{
				Name:     "Status",
				FullName: "test.package.Status",
			},
			{
				Name:     "Priority",
				FullName: "test.package.Priority",
			},
		},
		Messages: []*MessageDoc{
			{
				Name:     "User",
				FullName: "test.package.User",
				Enums: []*EnumDoc{
					{
						Name:     "Role",
						FullName: "test.package.User.Role",
					},
				},
			},
		},
	}

	t.Run("find top-level enum by name", func(t *testing.T) {
		enum := doc.FindEnum("Status")
		if enum == nil {
			t.Fatal("Expected to find Status enum")
		}
		if enum.Name != "Status" {
			t.Errorf("Expected name 'Status', got %s", enum.Name)
		}
	})

	t.Run("find top-level enum by full name", func(t *testing.T) {
		enum := doc.FindEnum("test.package.Priority")
		if enum == nil {
			t.Fatal("Expected to find Priority enum")
		}
		if enum.Name != "Priority" {
			t.Errorf("Expected name 'Priority', got %s", enum.Name)
		}
	})

	t.Run("find nested enum", func(t *testing.T) {
		enum := doc.FindEnum("Role")
		if enum == nil {
			t.Fatal("Expected to find Role enum")
		}
		if enum.Name != "Role" {
			t.Errorf("Expected name 'Role', got %s", enum.Name)
		}
		if enum.FullName != "test.package.User.Role" {
			t.Errorf("Expected full name 'test.package.User.Role', got %s", enum.FullName)
		}
	})

	t.Run("enum not found", func(t *testing.T) {
		enum := doc.FindEnum("NonExistent")
		if enum != nil {
			t.Error("Expected not to find NonExistent enum")
		}
	})
}

func TestDocumentation_FindService(t *testing.T) {
	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "UserService",
			},
			{
				Name: "OrderService",
			},
		},
	}

	t.Run("find existing service", func(t *testing.T) {
		svc := doc.FindService("UserService")
		if svc == nil {
			t.Fatal("Expected to find UserService")
		}
		if svc.Name != "UserService" {
			t.Errorf("Expected name 'UserService', got %s", svc.Name)
		}
	})

	t.Run("service not found", func(t *testing.T) {
		svc := doc.FindService("NonExistent")
		if svc != nil {
			t.Error("Expected not to find NonExistent service")
		}
	})
}

func TestGenerator_GenerateWithComments(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Messages: []*protobuf.MessageNode{
			{
				Name: "User",
				Comments: []*protobuf.CommentNode{
					{Text: "// User represents a user in the system"},
					{Text: "// with authentication details"},
				},
				Fields: []*protobuf.FieldNode{
					{
						Name:   "id",
						Number: 1,
						Type:   "string",
						Comments: []*protobuf.CommentNode{
							{Text: "// User ID"},
						},
					},
				},
			},
		},
	}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(doc.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(doc.Messages))
	}

	msg := doc.Messages[0]
	if msg.Description == "" {
		t.Error("Expected message description to be extracted from comments")
	}
	if msg.Description != "User represents a user in the system\nwith authentication details" {
		t.Errorf("Unexpected description: %s", msg.Description)
	}

	if len(msg.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(msg.Fields))
	}

	field := msg.Fields[0]
	if field.Description == "" {
		t.Error("Expected field description to be extracted from comments")
	}
}

func TestGenerator_GenerateWithNestedMessages(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Messages: []*protobuf.MessageNode{
			{
				Name: "Outer",
				Fields: []*protobuf.FieldNode{
					{Name: "id", Number: 1, Type: "string"},
				},
				Nested: []*protobuf.MessageNode{
					{
						Name: "Inner",
						Fields: []*protobuf.FieldNode{
							{Name: "value", Number: 1, Type: "int32"},
						},
						Nested: []*protobuf.MessageNode{
							{
								Name: "DeepNested",
								Fields: []*protobuf.FieldNode{
									{Name: "data", Number: 1, Type: "bytes"},
								},
							},
						},
					},
				},
			},
		},
	}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	outer := doc.FindMessage("Outer")
	if outer == nil {
		t.Fatal("Expected to find Outer message")
	}

	if len(outer.NestedTypes) != 1 {
		t.Fatalf("Expected 1 nested type, got %d", len(outer.NestedTypes))
	}

	inner := doc.FindMessage("Inner")
	if inner == nil {
		t.Fatal("Expected to find Inner message")
	}
	if inner.FullName != "test.package.Outer.Inner" {
		t.Errorf("Expected full name 'test.package.Outer.Inner', got %s", inner.FullName)
	}

	deepNested := doc.FindMessage("DeepNested")
	if deepNested == nil {
		t.Fatal("Expected to find DeepNested message")
	}
	if deepNested.FullName != "test.package.Outer.Inner.DeepNested" {
		t.Errorf("Expected full name 'test.package.Outer.Inner.DeepNested', got %s", deepNested.FullName)
	}
}

func TestGenerator_GenerateWithNestedEnums(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Messages: []*protobuf.MessageNode{
			{
				Name: "User",
				Enums: []*protobuf.EnumNode{
					{
						Name: "Status",
						Values: []*protobuf.EnumValueNode{
							{Name: "UNKNOWN", Number: 0},
							{Name: "ACTIVE", Number: 1},
						},
					},
				},
				Nested: []*protobuf.MessageNode{
					{
						Name: "Profile",
						Enums: []*protobuf.EnumNode{
							{
								Name: "Visibility",
								Values: []*protobuf.EnumValueNode{
									{Name: "PUBLIC", Number: 0},
									{Name: "PRIVATE", Number: 1},
								},
							},
						},
					},
				},
			},
		},
	}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	status := doc.FindEnum("Status")
	if status == nil {
		t.Fatal("Expected to find Status enum")
	}
	if status.FullName != "test.package.User.Status" {
		t.Errorf("Expected full name 'test.package.User.Status', got %s", status.FullName)
	}

	visibility := doc.FindEnum("Visibility")
	if visibility == nil {
		t.Fatal("Expected to find Visibility enum")
	}
	if visibility.FullName != "test.package.User.Profile.Visibility" {
		t.Errorf("Expected full name 'test.package.User.Profile.Visibility', got %s", visibility.FullName)
	}
}

func TestGenerator_GenerateWithFieldLabels(t *testing.T) {
	generator := NewGenerator()

	tests := []struct {
		name     string
		field    *protobuf.FieldNode
		expected struct {
			label    string
			repeated bool
			optional bool
			required bool
		}
	}{
		{
			name: "repeated field",
			field: &protobuf.FieldNode{
				Name:     "tags",
				Number:   1,
				Type:     "string",
				Repeated: true,
			},
			expected: struct {
				label    string
				repeated bool
				optional bool
				required bool
			}{
				label:    "repeated",
				repeated: true,
				optional: false,
				required: false,
			},
		},
		{
			name: "optional field",
			field: &protobuf.FieldNode{
				Name:     "email",
				Number:   2,
				Type:     "string",
				Optional: true,
			},
			expected: struct {
				label    string
				repeated bool
				optional bool
				required bool
			}{
				label:    "optional",
				repeated: false,
				optional: true,
				required: false,
			},
		},
		{
			name: "required field",
			field: &protobuf.FieldNode{
				Name:     "id",
				Number:   3,
				Type:     "string",
				Required: true,
			},
			expected: struct {
				label    string
				repeated bool
				optional bool
				required bool
			}{
				label:    "required",
				repeated: false,
				optional: false,
				required: true,
			},
		},
		{
			name: "no label field",
			field: &protobuf.FieldNode{
				Name:   "name",
				Number: 4,
				Type:   "string",
			},
			expected: struct {
				label    string
				repeated bool
				optional bool
				required bool
			}{
				label:    "",
				repeated: false,
				optional: false,
				required: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Package: &protobuf.PackageNode{Name: "test.package"},
				Messages: []*protobuf.MessageNode{
					{
						Name:   "TestMessage",
						Fields: []*protobuf.FieldNode{tt.field},
					},
				},
			}

			doc, err := generator.Generate(ast)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if len(doc.Messages) != 1 {
				t.Fatalf("Expected 1 message, got %d", len(doc.Messages))
			}

			msg := doc.Messages[0]
			if len(msg.Fields) != 1 {
				t.Fatalf("Expected 1 field, got %d", len(msg.Fields))
			}

			field := msg.Fields[0]
			if field.Label != tt.expected.label {
				t.Errorf("Expected label %q, got %q", tt.expected.label, field.Label)
			}
			if field.Repeated != tt.expected.repeated {
				t.Errorf("Expected repeated %v, got %v", tt.expected.repeated, field.Repeated)
			}
			if field.Optional != tt.expected.optional {
				t.Errorf("Expected optional %v, got %v", tt.expected.optional, field.Optional)
			}
			if field.Required != tt.expected.required {
				t.Errorf("Expected required %v, got %v", tt.expected.required, field.Required)
			}
		})
	}
}

func TestGenerator_GenerateWithStreamingMethods(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Services: []*protobuf.ServiceNode{
			{
				Name: "StreamService",
				RPCs: []*protobuf.RPCNode{
					{
						Name:            "UnaryCall",
						InputType:       "Request",
						OutputType:      "Response",
						ClientStreaming: false,
						ServerStreaming: false,
					},
					{
						Name:            "ServerStream",
						InputType:       "Request",
						OutputType:      "Response",
						ClientStreaming: false,
						ServerStreaming: true,
					},
					{
						Name:            "ClientStream",
						InputType:       "Request",
						OutputType:      "Response",
						ClientStreaming: true,
						ServerStreaming: false,
					},
					{
						Name:            "BidiStream",
						InputType:       "Request",
						OutputType:      "Response",
						ClientStreaming: true,
						ServerStreaming: true,
					},
				},
			},
		},
	}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	svc := doc.FindService("StreamService")
	if svc == nil {
		t.Fatal("Expected to find StreamService")
	}

	if len(svc.Methods) != 4 {
		t.Fatalf("Expected 4 methods, got %d", len(svc.Methods))
	}

	tests := []struct {
		name            string
		clientStreaming bool
		serverStreaming bool
	}{
		{"UnaryCall", false, false},
		{"ServerStream", false, true},
		{"ClientStream", true, false},
		{"BidiStream", true, true},
	}

	for i, tt := range tests {
		method := svc.Methods[i]
		if method.Name != tt.name {
			t.Errorf("Expected method name %q, got %q", tt.name, method.Name)
		}
		if method.ClientStreaming != tt.clientStreaming {
			t.Errorf("Method %s: expected ClientStreaming %v, got %v", tt.name, tt.clientStreaming, method.ClientStreaming)
		}
		if method.ServerStreaming != tt.serverStreaming {
			t.Errorf("Method %s: expected ServerStreaming %v, got %v", tt.name, tt.serverStreaming, method.ServerStreaming)
		}
	}
}

func TestGenerator_GenerateWithImports(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Imports: []*protobuf.ImportNode{
			{Path: "google/protobuf/timestamp.proto"},
			{Path: "google/protobuf/empty.proto"},
			{Path: "custom/types.proto"},
		},
		Messages: []*protobuf.MessageNode{
			{
				Name: "TestMessage",
				Fields: []*protobuf.FieldNode{
					{Name: "id", Number: 1, Type: "string"},
				},
			},
		},
	}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(doc.Imports) != 3 {
		t.Fatalf("Expected 3 imports, got %d", len(doc.Imports))
	}

	expectedImports := []string{
		"google/protobuf/timestamp.proto",
		"google/protobuf/empty.proto",
		"custom/types.proto",
	}

	for i, expected := range expectedImports {
		if doc.Imports[i] != expected {
			t.Errorf("Expected import %q, got %q", expected, doc.Imports[i])
		}
	}
}

func TestExtractComments_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		comments []*protobuf.CommentNode
		expected string
	}{
		{
			name:     "empty comments",
			comments: []*protobuf.CommentNode{},
			expected: "",
		},
		{
			name:     "nil comments",
			comments: nil,
			expected: "",
		},
		{
			name: "single line comment with // prefix",
			comments: []*protobuf.CommentNode{
				{Text: "// This is a comment"},
			},
			expected: "This is a comment",
		},
		{
			name: "multiline comment with /* */ markers",
			comments: []*protobuf.CommentNode{
				{Text: "/* This is a multiline comment */"},
			},
			expected: "This is a multiline comment",
		},
		{
			name: "multiple comment nodes",
			comments: []*protobuf.CommentNode{
				{Text: "// First line"},
				{Text: "// Second line"},
				{Text: "// Third line"},
			},
			expected: "First line\nSecond line\nThird line",
		},
		{
			name: "comments with whitespace",
			comments: []*protobuf.CommentNode{
				{Text: "   //   This has whitespace   "},
			},
			expected: "This has whitespace",
		},
		{
			name: "empty comment text",
			comments: []*protobuf.CommentNode{
				{Text: "//"},
				{Text: ""},
				{Text: "   "},
			},
			expected: "",
		},
		{
			name: "mixed comment styles",
			comments: []*protobuf.CommentNode{
				{Text: "// Line comment"},
				{Text: "/* Block comment */"},
				{Text: "Regular text"},
			},
			expected: "Line comment\nBlock comment\nRegular text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractComments(tt.comments)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerator_GenerateWithEnumComments(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Enums: []*protobuf.EnumNode{
			{
				Name: "Status",
				Comments: []*protobuf.CommentNode{
					{Text: "// Status represents the state of an entity"},
				},
				Values: []*protobuf.EnumValueNode{
					{
						Name:   "UNKNOWN",
						Number: 0,
						Comments: []*protobuf.CommentNode{
							{Text: "// Default unknown state"},
						},
					},
					{
						Name:   "ACTIVE",
						Number: 1,
						Comments: []*protobuf.CommentNode{
							{Text: "// Entity is active"},
						},
					},
				},
			},
		},
	}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(doc.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(doc.Enums))
	}

	enum := doc.Enums[0]
	if enum.Description != "Status represents the state of an entity" {
		t.Errorf("Expected enum description, got %q", enum.Description)
	}

	if len(enum.Values) != 2 {
		t.Fatalf("Expected 2 enum values, got %d", len(enum.Values))
	}

	if enum.Values[0].Description != "Default unknown state" {
		t.Errorf("Expected value description, got %q", enum.Values[0].Description)
	}
}

func TestGenerator_GenerateWithServiceComments(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Services: []*protobuf.ServiceNode{
			{
				Name: "UserService",
				Comments: []*protobuf.CommentNode{
					{Text: "// UserService handles user operations"},
				},
				RPCs: []*protobuf.RPCNode{
					{
						Name:       "GetUser",
						InputType:  "GetUserRequest",
						OutputType: "User",
						Comments: []*protobuf.CommentNode{
							{Text: "// GetUser retrieves a user by ID"},
						},
					},
				},
			},
		},
	}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	svc := doc.FindService("UserService")
	if svc == nil {
		t.Fatal("Expected to find UserService")
	}

	if svc.Description != "UserService handles user operations" {
		t.Errorf("Expected service description, got %q", svc.Description)
	}

	if len(svc.Methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(svc.Methods))
	}

	method := svc.Methods[0]
	if method.Description != "GetUser retrieves a user by ID" {
		t.Errorf("Expected method description, got %q", method.Description)
	}
}

func TestDocumentation_FindMessage_ByFullName(t *testing.T) {
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name:     "User",
				FullName: "test.package.User",
			},
		},
	}

	msg := doc.FindMessage("test.package.User")
	if msg == nil {
		t.Fatal("Expected to find message by full name")
	}
	if msg.Name != "User" {
		t.Errorf("Expected name 'User', got %s", msg.Name)
	}
}

func TestGenerator_GenerateEmptyAST(t *testing.T) {
	generator := NewGenerator()

	ast := &protobuf.RootNode{}

	doc, err := generator.Generate(ast)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Expected documentation to be generated")
	}

	if doc.PackageName != "" {
		t.Errorf("Expected empty package name, got %s", doc.PackageName)
	}

	if doc.Syntax != "" {
		t.Errorf("Expected empty syntax, got %s", doc.Syntax)
	}

	if len(doc.Messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(doc.Messages))
	}

	if len(doc.Enums) != 0 {
		t.Errorf("Expected 0 enums, got %d", len(doc.Enums))
	}

	if len(doc.Services) != 0 {
		t.Errorf("Expected 0 services, got %d", len(doc.Services))
	}
}
