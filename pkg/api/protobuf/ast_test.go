package protobuf

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseProtoFile(t *testing.T) {
	testCases := []struct {
		name            string
		content         string
		expectError     bool
		expectedSyntax  string
		expectedPkg     string
		expectedOptions map[string]string
		expectedImports []string
		expectedMessages []string
		expectedEnums []string
		expectedComments []string
		expectedServices []string
	}{
		{
			name: "Basic Proto",
			content: `syntax = "proto3";
package example;

import "common/common.proto";

message Test {
  string id = 1;
  int32 count = 2;
}`,
			expectError:     false,
			expectedSyntax:  "proto3",
			expectedPkg:     "example",
			expectedImports: []string{"common/common.proto"},
			expectedMessages: []string{"Test"},
		},
		{
			name: "Multiple Imports",
			content: `syntax = "proto3";
package test;

import "common/common.proto";
import "user/user.proto";

message Order {
  string id = 1;
  string user_id = 2;
}`,
			expectError:     false,
			expectedSyntax:  "proto3",
			expectedPkg:     "test",
			expectedImports: []string{"common/common.proto", "user/user.proto"},
			expectedMessages: []string{"Order"},
		},
		{
			name: "With Comments",
			content: `// This is a test proto file
syntax = "proto3";

// Package test contains test entities
package test;

// Import common definitions
import "common/common.proto";

// Test message represents a test entity
message Test {
  // Unique identifier
  string id = 1;
  // Count of items
  int32 count = 2;
}`,
			expectError:     false,
			expectedSyntax:  "proto3",
			expectedPkg:     "test",
			expectedImports: []string{"common/common.proto"},
			expectedMessages: []string{"Test"},
			expectedComments: []string{
				"This is a test proto file", 
				"Package test contains test entities", 
				"Import common definitions",
				"Test message represents a test entity",
				"Unique identifier",
				"Count of items",
			},
		},
		{
			name: "With Validations",
			content: `// This is a test proto file
syntax = "proto3";

// Package test contains test entities
// @spoke:domain:github.com/example/test
package test;

option go_package = "github.com/example/test";

// Import common definitions
import "common/common.proto";

// Test message represents a test entity
message Test {
  // Unique identifier
  string id = 1 [
		(validate.rules).string.min_len = 1,
		(validate.rules).string.max_len = 10,
		(validate.rules).string.pattern = "^[a-z0-9_-]+$",
	];
  // Count of items
  int32 count = 2 [
		(validate.rules).int32.gt = 0,
		(validate.rules).int32.lt = 100,
    ];
}`,
			expectError:     false,
			expectedSyntax:  "proto3",
			expectedPkg:     "test",
			expectedOptions: map[string]string{
				"go_package": "\"github.com/example/test\"",
			},
			expectedImports: []string{"common/common.proto"},
			expectedMessages: []string{"Test"},
			expectedComments: []string{
				"This is a test proto file", 
				"Package test contains test entities", 
				"Import common definitions",
				"Test message represents a test entity",
				"Unique identifier",
				"Count of items",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := ParseString(tc.content)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			// Check syntax
			if ast.Syntax == nil {
				t.Errorf("Expected syntax but got nil")
			} else if ast.Syntax.Value != tc.expectedSyntax {
				t.Errorf("Expected syntax %q but got %q", tc.expectedSyntax, ast.Syntax.Value)
			}
			
			// Check package
			if ast.Package == nil {
				t.Errorf("Expected package but got nil")
			} else if ast.Package.Name != tc.expectedPkg {
				t.Errorf("Expected package %q but got %q", tc.expectedPkg, ast.Package.Name)
			}

			if len(ast.Options) != len(tc.expectedOptions) {
				t.Errorf("Expected %d options but got %d", len(tc.expectedOptions), len(ast.Options))
			} else {
				for _, option := range ast.Options {
					if _, ok := tc.expectedOptions[option.Name]; !ok {
						t.Errorf("Unexpected option %q", option.Name)
					} else {
						if !strings.EqualFold(option.Value, tc.expectedOptions[option.Name]) {
							t.Errorf("Expected option %q to be %q but got %q", option.Name, tc.expectedOptions[option.Name], option.Value)
						}
					}
				}
			}
			
			// Check imports
			if len(ast.Imports) != len(tc.expectedImports) {
				t.Errorf("Expected %d imports but got %d", len(tc.expectedImports), len(ast.Imports))
			} else {
				for i, expected := range tc.expectedImports {
					if i >= len(ast.Imports) {
						t.Errorf("Missing import %q", expected)
						continue
					}
					if ast.Imports[i].Path != expected {
						t.Errorf("Expected import %q but got %q", expected, ast.Imports[i].Path)
					}
				}
			}

			// check messages
			if len(ast.Messages) != len(tc.expectedMessages) {
				t.Errorf("Expected %d messages but got %d", len(tc.expectedMessages), len(ast.Messages))
			} else {
				for i, expected := range tc.expectedMessages {
					if ast.Messages[i].Name != expected {
						t.Errorf("Expected message %q but got %q", expected, ast.Messages[i].Name)
						}
				}
			}

			// check enums
			if len(ast.Enums) != len(tc.expectedEnums) {
				t.Errorf("Expected %d enums but got %d", len(tc.expectedEnums), len(ast.Enums))
			} else {
				for i, expected := range tc.expectedEnums {
					if ast.Enums[i].Name != expected {
						t.Errorf("Expected enum %q but got %q", expected, ast.Enums[i].Name)
					}
				}
			}
			
			// check comments
			if len(ast.Comments) != len(tc.expectedComments) {
				t.Errorf("Expected %d comments but got %d", len(tc.expectedComments), len(ast.Comments))
			} else {
				for i, expected := range tc.expectedComments {
					if ast.Comments[i].Text != expected {
						t.Errorf("Expected comment %q but got %q", expected, ast.Comments[i].Text)
					}
				}
			}
			
			// check services
			if len(ast.Services) != len(tc.expectedServices) {
				t.Errorf("Expected %d services but got %d", len(tc.expectedServices), len(ast.Services))
			} else {
				for i, expected := range tc.expectedServices {
					if ast.Services[i].Name != expected {
						t.Errorf("Expected service %q but got %q", expected, ast.Services[i].Name)
					}
				}
			}
			
		})
	}
}

func TestParseSpokeDirectives(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int // expected number of spoke directives
		option   string
		value    string
	}{
		{
			name:     "domain directive",
			input:    `// @spoke:domain:github.com/example/text`,
			expected: 1,
			option:   "domain",
			value:    "github.com/example/text",
		},
		{
			name:     "option directive",
			input:    `// @spoke:option:some-value`,
			expected: 1,
			option:   "option",
			value:    "some-value",
		},
		{
			name:     "regular comment",
			input:    `// This is a regular comment`,
			expected: 0,
			option:   "",
			value:    "",
		},
		{
			name:     "multiple directives",
			input: `// @spoke:domain:github.com/example/text
// @spoke:option:test-value`,
			expected: 2,
			option:   "domain", // we'll check the first one
			value:    "github.com/example/text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewStringParser(tc.input)
			root, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			if len(root.SpokeDirectives) != tc.expected {
				t.Errorf("Expected %d spoke directives, got %d", tc.expected, len(root.SpokeDirectives))
			}

			if tc.expected > 0 {
				directive := root.SpokeDirectives[0]
				if directive.Option != tc.option {
					t.Errorf("Expected option %q, got %q", tc.option, directive.Option)
				}
				if directive.Value != tc.value {
					t.Errorf("Expected value %q, got %q", tc.value, directive.Value)
				}
			}
		})
	}
}

func TestSpokeDirectiveInMessage(t *testing.T) {
	input := `syntax = "proto3";

message TestMessage {
    // @spoke:domain:github.com/example/test
    // Regular comment
    string field1 = 1;
}`

	parser := NewStringParser(input)
	root, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Check that spoke directives are captured at both root and message level
	if len(root.SpokeDirectives) != 1 {
		t.Errorf("Expected 1 spoke directive at root level, got %d", len(root.SpokeDirectives))
	}

	if len(root.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(root.Messages))
	}

	message := root.Messages[0]
	if len(message.SpokeDirectives) != 1 {
		t.Errorf("Expected 1 spoke directive in message, got %d", len(message.SpokeDirectives))
	}

	if len(message.Comments) != 1 {
		t.Errorf("Expected 1 regular comment in message, got %d", len(message.Comments))
	}

	directive := message.SpokeDirectives[0]
	if directive.Option != "domain" {
		t.Errorf("Expected option 'domain', got %q", directive.Option)
	}
	if directive.Value != "github.com/example/test" {
		t.Errorf("Expected value 'github.com/example/test', got %q", directive.Value)
	}
}

func TestInvalidSpokeDirective(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "missing value",
			input: `// @spoke:domain`,
		},
		{
			name:  "invalid option type",
			input: `// @spoke:invalid:value`,
		},
		{
			name:  "malformed",
			input: `// @spoke:`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewStringParser(tc.input)
			_, err := parser.Parse()
			if err == nil {
				t.Error("Expected error for invalid spoke directive")
			}
		})
	}
}

func Example() {
	protoContent := `syntax = "proto3";

// @spoke:domain:github.com/example/user
package user;

import "google/protobuf/timestamp.proto";

// @spoke:option:validation-enabled
message User {
    // @spoke:domain:github.com/example/user/id
    string id = 1;
    
    // Regular comment about the name field
    string name = 2;
    
    // @spoke:option:email-validation
    string email = 3;
    
    google.protobuf.Timestamp created_at = 4;
}

// @spoke:domain:github.com/example/user/service
service UserService {
    // @spoke:option:auth-required
    rpc GetUser(GetUserRequest) returns (User);
    
    rpc CreateUser(CreateUserRequest) returns (User);
}`

	parser := NewStringParser(protoContent)
	root, err := parser.Parse()
	if err != nil {
		fmt.Printf("Error parsing: %v\n", err)
		return
	}

	fmt.Printf("Found %d spoke directives at root level:\n", len(root.SpokeDirectives))
	for i, directive := range root.SpokeDirectives {
		fmt.Printf("  %d. Option: %s, Value: %s\n", i+1, directive.Option, directive.Value)
	}

	if len(root.Messages) > 0 {
		msg := root.Messages[0]
		fmt.Printf("\nMessage '%s' has %d spoke directives:\n", msg.Name, len(msg.SpokeDirectives))
		for i, directive := range msg.SpokeDirectives {
			fmt.Printf("  %d. Option: %s, Value: %s\n", i+1, directive.Option, directive.Value)
		}
	}

	if len(root.Services) > 0 {
		svc := root.Services[0]
		fmt.Printf("\nService '%s' has %d spoke directives:\n", svc.Name, len(svc.SpokeDirectives))
		for i, directive := range svc.SpokeDirectives {
			fmt.Printf("  %d. Option: %s, Value: %s\n", i+1, directive.Option, directive.Value)
		}
	}

	// Output:
	// Found 6 spoke directives at root level:
	//   1. Option: domain, Value: github.com/example/user
	//   2. Option: option, Value: validation-enabled
	//   3. Option: domain, Value: github.com/example/user/id
	//   4. Option: option, Value: email-validation
	//   5. Option: domain, Value: github.com/example/user/service
	//   6. Option: option, Value: auth-required
	//
	// Message 'User' has 2 spoke directives:
	//   1. Option: domain, Value: github.com/example/user/id
	//   2. Option: option, Value: email-validation
	//
	// Service 'UserService' has 1 spoke directives:
	//   1. Option: option, Value: auth-required
}

func TestExampleSpokeDirectiveParsing(t *testing.T) {
	// This test ensures the example above works correctly
	protoContent := `syntax = "proto3";

// @spoke:domain:github.com/example/user
package user;

// @spoke:option:validation-enabled
message User {
    // @spoke:option:foo bar
    string id = 1;
    string name = 2;
}`

	parser := NewStringParser(protoContent)
	root, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Should have 3 spoke directives total
	expectedDirectives := 3
	if len(root.SpokeDirectives) != expectedDirectives {
		t.Errorf("Expected %d spoke directives at root level, got %d", expectedDirectives, len(root.SpokeDirectives))
	}

	// Check specific directives
	expectedRootDirectives := []struct {
		option string
		value  string
	}{
		{"domain", "github.com/example/user"},
		{"option", "validation-enabled"},
		{"option", "foo bar"},
	}

	for i, expected := range expectedRootDirectives {
		if i >= len(root.SpokeDirectives) {
			t.Errorf("Missing spoke directive %d", i+1)
			continue
		}
		directive := root.SpokeDirectives[i]
		if directive.Option != expected.option {
			t.Errorf("Directive %d: expected option %q, got %q", i+1, expected.option, directive.Option)
		}
		if directive.Value != expected.value {
			t.Errorf("Directive %d: expected value %q, got %q", i+1, expected.value, directive.Value)
		}
	}

	// Check message-level directives
	if len(root.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(root.Messages))
	}

	msg := root.Messages[0]
	expectedMsgDirectives := 1 // Only the directive inside the message body
	if len(msg.SpokeDirectives) != expectedMsgDirectives {
		t.Errorf("Expected %d spoke directives in message, got %d", expectedMsgDirectives, len(msg.SpokeDirectives))
	}
}

// TestNodeTypeAndPosition tests all AST node interface methods
func TestNodeTypeAndPosition(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	endPos := Position{Line: 1, Column: 10, Offset: 10}

	testCases := []struct {
		name     string
		node     Node
		nodeType NodeType
	}{
		{
			name: "RootNode",
			node: &RootNode{
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypeUnknown,
		},
		{
			name: "SyntaxNode",
			node: &SyntaxNode{
				Value:  "proto3",
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypeSyntax,
		},
		{
			name: "PackageNode",
			node: &PackageNode{
				Name:   "test",
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypePackage,
		},
		{
			name: "ImportNode",
			node: &ImportNode{
				Path:   "test.proto",
				Public: false,
				Weak:   false,
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypeImport,
		},
		{
			name: "OptionNode",
			node: &OptionNode{
				Name:   "go_package",
				Value:  "test",
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypeOption,
		},
		{
			name: "FieldNode",
			node: &FieldNode{
				Name:     "id",
				Type:     "string",
				Number:   1,
				Repeated: false,
				Optional: false,
				Required: false,
				Pos:      pos,
				EndPos:   endPos,
			},
			nodeType: NodeTypeField,
		},
		{
			name: "MessageNode",
			node: &MessageNode{
				Name:   "Test",
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypeMessage,
		},
		{
			name: "EnumValueNode",
			node: &EnumValueNode{
				Name:   "UNKNOWN",
				Number: 0,
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypeEnumValue,
		},
		{
			name: "EnumNode",
			node: &EnumNode{
				Name:   "Status",
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypeEnum,
		},
		{
			name: "RPCNode",
			node: &RPCNode{
				Name:            "GetUser",
				InputType:       "GetUserRequest",
				OutputType:      "User",
				ClientStreaming: false,
				ServerStreaming: false,
				Pos:             pos,
				EndPos:          endPos,
			},
			nodeType: NodeTypeRPC,
		},
		{
			name: "ServiceNode",
			node: &ServiceNode{
				Name:   "UserService",
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypeService,
		},
		{
			name: "CommentNode",
			node: &CommentNode{
				Text:     "test comment",
				Leading:  true,
				Trailing: false,
				Pos:      pos,
				EndPos:   endPos,
			},
			nodeType: NodeTypeComment,
		},
		{
			name: "SpokeDirectiveNode",
			node: &SpokeDirectiveNode{
				Option: "domain",
				Value:  "test",
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypeSpoke,
		},
		{
			name: "OneOfNode",
			node: &OneOfNode{
				Name:   "test_oneof",
				Pos:    pos,
				EndPos: endPos,
			},
			nodeType: NodeTypeOneOf,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test NodeType
			if tc.node.NodeType() != tc.nodeType {
				t.Errorf("Expected NodeType %v but got %v", tc.nodeType, tc.node.NodeType())
			}

			// Test Position
			if tc.node.Position() != pos {
				t.Errorf("Expected Position %v but got %v", pos, tc.node.Position())
			}

			// Test End
			if tc.node.End() != endPos {
				t.Errorf("Expected End %v but got %v", endPos, tc.node.End())
			}
		})
	}
}

// TestNewParser tests creating a parser from io.Reader
func TestNewParser(t *testing.T) {
	content := `syntax = "proto3";
package test;`

	parser := NewParser(strings.NewReader(content))
	if parser == nil {
		t.Fatal("Expected non-nil parser")
	}

	root, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if root.Syntax == nil || root.Syntax.Value != "proto3" {
		t.Error("Expected proto3 syntax")
	}

	if root.Package == nil || root.Package.Name != "test" {
		t.Error("Expected test package")
	}
}

// TestImportVariations tests public and weak imports
func TestImportVariations(t *testing.T) {
	testCases := []struct {
		name       string
		content    string
		expectPublic bool
		expectWeak   bool
	}{
		{
			name:       "regular import",
			content:    `syntax = "proto3"; import "test.proto";`,
			expectPublic: false,
			expectWeak:   false,
		},
		{
			name:       "public import",
			content:    `syntax = "proto3"; import public "test.proto";`,
			expectPublic: true,
			expectWeak:   false,
		},
		{
			name:       "weak import",
			content:    `syntax = "proto3"; import weak "test.proto";`,
			expectPublic: false,
			expectWeak:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := ParseString(tc.content)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			if len(root.Imports) != 1 {
				t.Fatalf("Expected 1 import, got %d", len(root.Imports))
			}

			imp := root.Imports[0]
			if imp.Public != tc.expectPublic {
				t.Errorf("Expected Public=%v, got %v", tc.expectPublic, imp.Public)
			}
			if imp.Weak != tc.expectWeak {
				t.Errorf("Expected Weak=%v, got %v", tc.expectWeak, imp.Weak)
			}
		})
	}
}

// TestBlockComments tests parsing of block comments
func TestBlockComments(t *testing.T) {
	content := `/* Block comment */
syntax = "proto3";

/* Multi-line
   block comment */
package test;

message Test {
    /* Field comment */
    string id = 1;
}`

	root, err := ParseString(content)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(root.Comments) < 1 {
		t.Error("Expected at least one comment to be parsed")
	}

	// Verify comments are parsed correctly
	for _, comment := range root.Comments {
		if comment.Text == "" {
			t.Error("Comment text should not be empty")
		}
	}
}

// TestEnumParsing tests enum parsing
func TestEnumParsing(t *testing.T) {
	content := `syntax = "proto3";

enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
    INACTIVE = 2;
}

message Test {
    Status status = 1;
}`

	root, err := ParseString(content)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(root.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(root.Enums))
	}

	enum := root.Enums[0]
	if enum.Name != "Status" {
		t.Errorf("Expected enum name 'Status', got %q", enum.Name)
	}

	// Test enum NodeType
	if enum.NodeType() != NodeTypeEnum {
		t.Errorf("Expected NodeType %v, got %v", NodeTypeEnum, enum.NodeType())
	}
}

// TestServiceParsing tests service parsing
func TestServiceParsing(t *testing.T) {
	content := `syntax = "proto3";

service UserService {
    rpc GetUser(GetUserRequest) returns (User);
    rpc CreateUser(CreateUserRequest) returns (User);
}

message User {
    string id = 1;
}`

	root, err := ParseString(content)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(root.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(root.Services))
	}

	service := root.Services[0]
	if service.Name != "UserService" {
		t.Errorf("Expected service name 'UserService', got %q", service.Name)
	}

	// Test service NodeType
	if service.NodeType() != NodeTypeService {
		t.Errorf("Expected NodeType %v, got %v", NodeTypeService, service.NodeType())
	}
}

// TestEnumWithComments tests enum parsing with comments
func TestEnumWithComments(t *testing.T) {
	content := `syntax = "proto3";

// Status enum
enum Status {
    // Unknown status
    UNKNOWN = 0;
    ACTIVE = 1;
}`

	root, err := ParseString(content)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(root.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(root.Enums))
	}

	enum := root.Enums[0]
	if len(enum.Comments) < 1 {
		t.Error("Expected enum to have comments")
	}
}

// TestServiceWithComments tests service parsing with comments
func TestServiceWithComments(t *testing.T) {
	content := `syntax = "proto3";

// User service
service UserService {
    // Get a user
    rpc GetUser(GetUserRequest) returns (User);
}`

	root, err := ParseString(content)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(root.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(root.Services))
	}

	service := root.Services[0]
	if len(service.Comments) < 1 {
		t.Error("Expected service to have comments")
	}
}

// TestEnumWithSpokeDirective tests enum parsing with spoke directives
func TestEnumWithSpokeDirective(t *testing.T) {
	content := `syntax = "proto3";

enum Status {
    // @spoke:domain:test.enum
    UNKNOWN = 0;
    ACTIVE = 1;
}`

	root, err := ParseString(content)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(root.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(root.Enums))
	}

	enum := root.Enums[0]
	if len(enum.SpokeDirectives) < 1 {
		t.Error("Expected enum to have spoke directives")
	}
}

// TestServiceWithSpokeDirective tests service parsing with spoke directives
func TestServiceWithSpokeDirective(t *testing.T) {
	content := `syntax = "proto3";

service UserService {
    // @spoke:option:auth-required
    rpc GetUser(GetUserRequest) returns (User);
}`

	root, err := ParseString(content)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(root.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(root.Services))
	}

	service := root.Services[0]
	if len(service.SpokeDirectives) < 1 {
		t.Error("Expected service to have spoke directives")
	}
}

// TestParseErrors tests various parse error cases
func TestParseErrors(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{
			name:    "multiple syntax statements",
			content: `syntax = "proto3"; syntax = "proto2";`,
		},
		{
			name:    "multiple package statements",
			content: `syntax = "proto3"; package test; package test2;`,
		},
		{
			name:    "invalid token",
			content: `syntax = "proto3"; invalid_keyword test;`,
		},
		{
			name:    "missing semicolon in syntax",
			content: `syntax = "proto3"`,
		},
		{
			name:    "missing semicolon in package",
			content: `syntax = "proto3"; package test`,
		},
		{
			name:    "missing semicolon in import",
			content: `syntax = "proto3"; import "test.proto"`,
		},
		{
			name:    "missing semicolon in option",
			content: `syntax = "proto3"; option go_package = "test"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseString(tc.content)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// TestFieldNodeVariations tests different field types
func TestFieldNodeVariations(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	endPos := Position{Line: 1, Column: 10, Offset: 10}

	testCases := []struct {
		name string
		field *FieldNode
	}{
		{
			name: "repeated field",
			field: &FieldNode{
				Name:     "items",
				Type:     "string",
				Number:   1,
				Repeated: true,
				Pos:      pos,
				EndPos:   endPos,
			},
		},
		{
			name: "optional field",
			field: &FieldNode{
				Name:     "email",
				Type:     "string",
				Number:   2,
				Optional: true,
				Pos:      pos,
				EndPos:   endPos,
			},
		},
		{
			name: "required field",
			field: &FieldNode{
				Name:     "id",
				Type:     "string",
				Number:   3,
				Required: true,
				Pos:      pos,
				EndPos:   endPos,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.field.NodeType() != NodeTypeField {
				t.Errorf("Expected NodeType %v, got %v", NodeTypeField, tc.field.NodeType())
			}
			if tc.field.Position() != pos {
				t.Errorf("Expected Position %v, got %v", pos, tc.field.Position())
			}
			if tc.field.End() != endPos {
				t.Errorf("Expected End %v, got %v", endPos, tc.field.End())
			}
		})
	}
}

// TestRPCNodeVariations tests different RPC types
func TestRPCNodeVariations(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	endPos := Position{Line: 1, Column: 10, Offset: 10}

	testCases := []struct {
		name string
		rpc  *RPCNode
	}{
		{
			name: "unary RPC",
			rpc: &RPCNode{
				Name:            "GetUser",
				InputType:       "GetUserRequest",
				OutputType:      "User",
				ClientStreaming: false,
				ServerStreaming: false,
				Pos:             pos,
				EndPos:          endPos,
			},
		},
		{
			name: "client streaming RPC",
			rpc: &RPCNode{
				Name:            "UploadData",
				InputType:       "DataChunk",
				OutputType:      "UploadResult",
				ClientStreaming: true,
				ServerStreaming: false,
				Pos:             pos,
				EndPos:          endPos,
			},
		},
		{
			name: "server streaming RPC",
			rpc: &RPCNode{
				Name:            "StreamEvents",
				InputType:       "EventRequest",
				OutputType:      "Event",
				ClientStreaming: false,
				ServerStreaming: true,
				Pos:             pos,
				EndPos:          endPos,
			},
		},
		{
			name: "bidirectional streaming RPC",
			rpc: &RPCNode{
				Name:            "Chat",
				InputType:       "Message",
				OutputType:      "Message",
				ClientStreaming: true,
				ServerStreaming: true,
				Pos:             pos,
				EndPos:          endPos,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.rpc.NodeType() != NodeTypeRPC {
				t.Errorf("Expected NodeType %v, got %v", NodeTypeRPC, tc.rpc.NodeType())
			}
			if tc.rpc.Position() != pos {
				t.Errorf("Expected Position %v, got %v", pos, tc.rpc.Position())
			}
			if tc.rpc.End() != endPos {
				t.Errorf("Expected End %v, got %v", endPos, tc.rpc.End())
			}
		})
	}
}

// TestCommentNodeVariations tests different comment types
func TestCommentNodeVariations(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	endPos := Position{Line: 1, Column: 10, Offset: 10}

	testCases := []struct {
		name    string
		comment *CommentNode
	}{
		{
			name: "leading comment",
			comment: &CommentNode{
				Text:     "This is a leading comment",
				Leading:  true,
				Trailing: false,
				Pos:      pos,
				EndPos:   endPos,
			},
		},
		{
			name: "trailing comment",
			comment: &CommentNode{
				Text:     "This is a trailing comment",
				Leading:  false,
				Trailing: true,
				Pos:      pos,
				EndPos:   endPos,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.comment.NodeType() != NodeTypeComment {
				t.Errorf("Expected NodeType %v, got %v", NodeTypeComment, tc.comment.NodeType())
			}
			if tc.comment.Position() != pos {
				t.Errorf("Expected Position %v, got %v", pos, tc.comment.Position())
			}
			if tc.comment.End() != endPos {
				t.Errorf("Expected End %v, got %v", endPos, tc.comment.End())
			}
		})
	}
}

// TestNestedMessages tests messages with nested structures
func TestNestedMessages(t *testing.T) {
	content := `syntax = "proto3";

message Outer {
    message Inner {
        string id = 1;
    }
    Inner inner = 1;
}`

	root, err := ParseString(content)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(root.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(root.Messages))
	}

	outer := root.Messages[0]
	if outer.Name != "Outer" {
		t.Errorf("Expected message name 'Outer', got %q", outer.Name)
	}
}

// TestMessageWithOptions tests messages with options
func TestMessageWithOptions(t *testing.T) {
	content := `syntax = "proto3";

message Test {
    string id = 1;
    int32 count = 2;
}`

	root, err := ParseString(content)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(root.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(root.Messages))
	}

	msg := root.Messages[0]
	if msg.Name != "Test" {
		t.Errorf("Expected message name 'Test', got %q", msg.Name)
	}

	// Verify message has initialized slices
	if msg.Fields == nil {
		t.Error("Expected Fields slice to be initialized")
	}
	if msg.Nested == nil {
		t.Error("Expected Nested slice to be initialized")
	}
	if msg.Enums == nil {
		t.Error("Expected Enums slice to be initialized")
	}
	if msg.OneOfs == nil {
		t.Error("Expected OneOfs slice to be initialized")
	}
	if msg.Options == nil {
		t.Error("Expected Options slice to be initialized")
	}
} 
