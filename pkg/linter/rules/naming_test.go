package rules

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/linter"
)

func TestMessageNamingRule(t *testing.T) {
	rule := NewMessageNamingRule()

	tests := []struct {
		name           string
		messageName    string
		expectViolation bool
	}{
		{"valid PascalCase", "UserProfile", false},
		{"valid single word", "User", false},
		{"invalid snake_case", "user_profile", true},
		{"invalid camelCase", "userProfile", true},
		{"invalid lowercase", "user", true},
		{"invalid with underscore", "User_Profile", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Messages: []*protobuf.MessageNode{
					{Name: tt.messageName},
				},
			}

			ctx := &linter.LintContext{
				FilePath: "test.proto",
				AST:      ast,
			}

			violations := rule.Check(ast, ctx)

			if tt.expectViolation && len(violations) == 0 {
				t.Errorf("Expected violation for message name '%s'", tt.messageName)
			}
			if !tt.expectViolation && len(violations) > 0 {
				t.Errorf("Unexpected violation for message name '%s': %s", tt.messageName, violations[0].Message)
			}
		})
	}
}

func TestFieldNamingRule(t *testing.T) {
	rule := NewFieldNamingRule()

	tests := []struct {
		name           string
		fieldName      string
		expectViolation bool
	}{
		{"valid snake_case", "user_id", false},
		{"valid single word", "name", false},
		{"valid with numbers", "user_id_123", false},
		{"invalid PascalCase", "UserId", true},
		{"invalid camelCase", "userId", true},
		{"invalid UPPER_CASE", "USER_ID", true},
		{"invalid consecutive underscores", "user__id", true},
		{"invalid leading underscore", "_user_id", true},
		{"invalid trailing underscore", "user_id_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Messages: []*protobuf.MessageNode{
					{
						Name: "TestMessage",
						Fields: []*protobuf.FieldNode{
							{Name: tt.fieldName, Type: "string", Number: 1},
						},
					},
				},
			}

			ctx := &linter.LintContext{
				FilePath: "test.proto",
				AST:      ast,
			}

			violations := rule.Check(ast, ctx)

			if tt.expectViolation && len(violations) == 0 {
				t.Errorf("Expected violation for field name '%s'", tt.fieldName)
			}
			if !tt.expectViolation && len(violations) > 0 {
				t.Errorf("Unexpected violation for field name '%s': %s", tt.fieldName, violations[0].Message)
			}
		})
	}
}

func TestServiceNamingRule(t *testing.T) {
	rule := NewServiceNamingRule()

	tests := []struct {
		name           string
		serviceName    string
		expectViolation bool
	}{
		{"valid PascalCase", "UserService", false},
		{"valid single word", "Users", false},
		{"invalid snake_case", "user_service", true},
		{"invalid camelCase", "userService", true},
		{"invalid lowercase", "users", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Services: []*protobuf.ServiceNode{
					{Name: tt.serviceName},
				},
			}

			ctx := &linter.LintContext{
				FilePath: "test.proto",
				AST:      ast,
			}

			violations := rule.Check(ast, ctx)

			if tt.expectViolation && len(violations) == 0 {
				t.Errorf("Expected violation for service name '%s'", tt.serviceName)
			}
			if !tt.expectViolation && len(violations) > 0 {
				t.Errorf("Unexpected violation for service name '%s': %s", tt.serviceName, violations[0].Message)
			}
		})
	}
}

func TestEnumNamingRule(t *testing.T) {
	rule := NewEnumNamingRule()

	tests := []struct {
		name           string
		enumName       string
		expectViolation bool
	}{
		{"valid PascalCase", "UserStatus", false},
		{"valid single word", "Status", false},
		{"invalid snake_case", "user_status", true},
		{"invalid UPPER_CASE", "USER_STATUS", true},
		{"invalid lowercase", "status", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Enums: []*protobuf.EnumNode{
					{Name: tt.enumName},
				},
			}

			ctx := &linter.LintContext{
				FilePath: "test.proto",
				AST:      ast,
			}

			violations := rule.Check(ast, ctx)

			if tt.expectViolation && len(violations) == 0 {
				t.Errorf("Expected violation for enum name '%s'", tt.enumName)
			}
			if !tt.expectViolation && len(violations) > 0 {
				t.Errorf("Unexpected violation for enum name '%s': %s", tt.enumName, violations[0].Message)
			}
		})
	}
}

func TestEnumValueNamingRule(t *testing.T) {
	rule := NewEnumValueNamingRule()

	tests := []struct {
		name           string
		valueName      string
		expectViolation bool
	}{
		{"valid UPPER_SNAKE_CASE", "USER_STATUS_ACTIVE", false},
		{"valid single word", "ACTIVE", false},
		{"valid with numbers", "STATUS_1", false},
		{"invalid PascalCase", "StatusActive", true},
		{"invalid snake_case", "status_active", true},
		{"invalid camelCase", "statusActive", true},
		{"invalid consecutive underscores", "STATUS__ACTIVE", true},
		{"invalid trailing underscore", "STATUS_ACTIVE_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Enums: []*protobuf.EnumNode{
					{
						Name: "Status",
						Values: []*protobuf.EnumValueNode{
							{Name: tt.valueName, Number: 0},
						},
					},
				},
			}

			ctx := &linter.LintContext{
				FilePath: "test.proto",
				AST:      ast,
			}

			violations := rule.Check(ast, ctx)

			if tt.expectViolation && len(violations) == 0 {
				t.Errorf("Expected violation for enum value '%s'", tt.valueName)
			}
			if !tt.expectViolation && len(violations) > 0 {
				t.Errorf("Unexpected violation for enum value '%s': %s", tt.valueName, violations[0].Message)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user_profile", "UserProfile"},
		{"userProfile", "UserProfile"},
		{"user", "User"},
		{"User", "User"},
		{"user_profile_data", "UserProfileData"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("toPascalCase(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UserProfile", "user_profile"},
		{"userProfile", "user_profile"},
		{"user_profile", "user_profile"},
		{"User", "user"},
		{"userId", "user_id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToUpperSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"StatusActive", "STATUS_ACTIVE"},
		{"statusActive", "STATUS_ACTIVE"},
		{"STATUS_ACTIVE", "STATUS_ACTIVE"},
		{"ACTIVE", "ACTIVE"},
		{"user_status_active", "USER_STATUS_ACTIVE"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toUpperSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toUpperSnakeCase(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}
