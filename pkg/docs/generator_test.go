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
}
