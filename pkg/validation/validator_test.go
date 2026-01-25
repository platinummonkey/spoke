package validation

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

func TestValidator_ValidateFieldNumbers(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name        string
		fieldNumber int
		wantError   bool
	}{
		{"valid field 1", 1, false},
		{"valid field 100", 100, false},
		{"invalid field 0", 0, true},
		{"invalid negative", -1, true},
		{"reserved range start", 19000, true},
		{"reserved range end", 19999, true},
		{"valid after reserved", 20000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Messages: []*protobuf.MessageNode{
					{
						Name: "TestMessage",
						Fields: []*protobuf.FieldNode{
							{Name: "field", Number: tt.fieldNumber, Type: "string"},
						},
					},
				},
			}

			result := validator.Validate(ast)
			hasError := len(result.Errors) > 0

			if hasError != tt.wantError {
				t.Errorf("Field number %d: hasError = %v, wantError = %v", tt.fieldNumber, hasError, tt.wantError)
				if hasError {
					t.Logf("Errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestValidator_ValidateDuplicateFieldNumbers(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "TestMessage",
				Fields: []*protobuf.FieldNode{
					{Name: "field1", Number: 1, Type: "string"},
					{Name: "field2", Number: 1, Type: "int32"}, // Duplicate!
				},
			},
		},
	}

	result := validator.Validate(ast)

	if len(result.Errors) == 0 {
		t.Error("Expected error for duplicate field number")
	}

	// Check for specific error
	found := false
	for _, err := range result.Errors {
		if err.Rule == "DUPLICATE_FIELD_NUMBER" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected DUPLICATE_FIELD_NUMBER error")
	}
}

func TestValidator_ValidateEnumZeroValue(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name      string
		values    []*protobuf.EnumValueNode
		wantError bool
	}{
		{
			name: "has zero value",
			values: []*protobuf.EnumValueNode{
				{Name: "UNSPECIFIED", Number: 0},
				{Name: "VALUE_ONE", Number: 1},
			},
			wantError: false,
		},
		{
			name: "missing zero value",
			values: []*protobuf.EnumValueNode{
				{Name: "VALUE_ONE", Number: 1},
				{Name: "VALUE_TWO", Number: 2},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Enums: []*protobuf.EnumNode{
					{Name: "TestEnum", Values: tt.values},
				},
			}

			result := validator.Validate(ast)
			hasError := len(result.Errors) > 0

			if hasError != tt.wantError {
				t.Errorf("hasError = %v, wantError = %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidator_NamingConventions(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name        string
		messageName string
		fieldName   string
		enumName    string
		enumValue   string
		wantWarning bool
	}{
		{
			name:        "all correct",
			messageName: "UserProfile",
			fieldName:   "user_name",
			enumName:    "Status",
			enumValue:   "STATUS_ACTIVE",
			wantWarning: false,
		},
		{
			name:        "message not PascalCase",
			messageName: "user_profile",
			fieldName:   "user_name",
			enumName:    "Status",
			enumValue:   "STATUS_ACTIVE",
			wantWarning: true,
		},
		{
			name:        "field not snake_case",
			messageName: "UserProfile",
			fieldName:   "userName",
			enumName:    "Status",
			enumValue:   "STATUS_ACTIVE",
			wantWarning: true,
		},
		{
			name:        "enum value not UPPER_SNAKE_CASE",
			messageName: "UserProfile",
			fieldName:   "user_name",
			enumName:    "Status",
			enumValue:   "StatusActive",
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Messages: []*protobuf.MessageNode{
					{
						Name: tt.messageName,
						Fields: []*protobuf.FieldNode{
							{Name: tt.fieldName, Number: 1, Type: "string"},
						},
					},
				},
				Enums: []*protobuf.EnumNode{
					{
						Name: tt.enumName,
						Values: []*protobuf.EnumValueNode{
							{Name: tt.enumValue, Number: 0},
						},
					},
				},
			}

			result := validator.Validate(ast)
			hasWarning := len(result.Warnings) > 0

			if hasWarning != tt.wantWarning {
				t.Errorf("hasWarning = %v, wantWarning = %v", hasWarning, tt.wantWarning)
				if hasWarning {
					t.Logf("Warnings: %v", result.Warnings)
				}
			}
		})
	}
}

func TestValidator_PackageName(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name        string
		packageName string
		wantError   bool
	}{
		{"valid lowercase", "com.example.api", false},
		{"valid with numbers", "com.example.v1", false},
		{"invalid uppercase", "Com.Example.Api", true},
		{"invalid mixed case", "com.Example.api", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := &protobuf.RootNode{
				Package: &protobuf.PackageNode{Name: tt.packageName},
			}

			result := validator.Validate(ast)
			hasError := len(result.Errors) > 0

			if hasError != tt.wantError {
				t.Errorf("Package %q: hasError = %v, wantError = %v", tt.packageName, hasError, tt.wantError)
			}
		})
	}
}

func TestIsValidPackageName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"com.example.api", true},
		{"com.example", true},
		{"example", true},
		{"com.example.v1", true},
		{"Com.Example", false},
		{"com.Example", false},
		{"com..example", false},
		{".com.example", false},
		{"com.example.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidPackageName(tt.name)
			if got != tt.want {
				t.Errorf("isValidPackageName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsPascalCase(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"UserProfile", true},
		{"User", true},
		{"HTTPServer", true},
		{"user_profile", false},
		{"userProfile", false},
		{"User_Profile", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPascalCase(tt.name)
			if got != tt.want {
				t.Errorf("isPascalCase(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsSnakeCase(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"user_name", true},
		{"user", true},
		{"user_name_123", true},
		{"userName", false},
		{"UserName", false},
		{"user-name", false},
		{"", false},
		{"123user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSnakeCase(tt.name)
			if got != tt.want {
				t.Errorf("isSnakeCase(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsUpperSnakeCase(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"STATUS_ACTIVE", true},
		{"STATUS", true},
		{"STATUS_123", true},
		{"status_active", false},
		{"StatusActive", false},
		{"Status_Active", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUpperSnakeCase(tt.name)
			if got != tt.want {
				t.Errorf("isUpperSnakeCase(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestValidationResult_Summary(t *testing.T) {
	result := &ValidationResult{
		Errors: []*ValidationError{
			{Rule: "ERROR1", Severity: SeverityError},
			{Rule: "ERROR2", Severity: SeverityError},
		},
		Warnings: []*ValidationError{
			{Rule: "WARN1", Severity: SeverityWarning},
		},
		Valid: false,
	}

	if len(result.Errors) != 2 {
		t.Errorf("Errors count = %d, want 2", len(result.Errors))
	}

	if len(result.Warnings) != 1 {
		t.Errorf("Warnings count = %d, want 1", len(result.Warnings))
	}

	if result.Valid {
		t.Error("Result should not be valid with errors")
	}
}
