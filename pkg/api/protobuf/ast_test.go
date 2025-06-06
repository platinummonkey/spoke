package protobuf

import (
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
				"@spoke:domain:github.com/example/test",
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
