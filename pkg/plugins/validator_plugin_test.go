package plugins

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidationRequest tests the ValidationRequest struct
func TestValidationRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      *ValidationRequest
		wantNil  bool
		validate func(*testing.T, *ValidationRequest)
	}{
		{
			name: "empty request",
			req:  &ValidationRequest{},
			validate: func(t *testing.T, req *ValidationRequest) {
				assert.Empty(t, req.ProtoFiles)
				assert.Empty(t, req.ModuleName)
				assert.Empty(t, req.Version)
				assert.Empty(t, req.Options)
			},
		},
		{
			name: "request with proto files",
			req: &ValidationRequest{
				ProtoFiles: []string{"foo.proto", "bar.proto"},
				ModuleName: "test-module",
				Version:    "1.0.0",
			},
			validate: func(t *testing.T, req *ValidationRequest) {
				assert.Len(t, req.ProtoFiles, 2)
				assert.Equal(t, "foo.proto", req.ProtoFiles[0])
				assert.Equal(t, "bar.proto", req.ProtoFiles[1])
				assert.Equal(t, "test-module", req.ModuleName)
				assert.Equal(t, "1.0.0", req.Version)
			},
		},
		{
			name: "request with options",
			req: &ValidationRequest{
				ProtoFiles: []string{"test.proto"},
				ModuleName: "module",
				Version:    "2.0.0",
				Options: map[string]string{
					"strict":        "true",
					"lint-level":    "error",
					"allow-warning": "false",
				},
			},
			validate: func(t *testing.T, req *ValidationRequest) {
				assert.Len(t, req.Options, 3)
				assert.Equal(t, "true", req.Options["strict"])
				assert.Equal(t, "error", req.Options["lint-level"])
				assert.Equal(t, "false", req.Options["allow-warning"])
			},
		},
		{
			name: "request with empty module name",
			req: &ValidationRequest{
				ProtoFiles: []string{"test.proto"},
				Version:    "1.0.0",
			},
			validate: func(t *testing.T, req *ValidationRequest) {
				assert.Empty(t, req.ModuleName)
				assert.NotEmpty(t, req.ProtoFiles)
			},
		},
		{
			name: "request with empty version",
			req: &ValidationRequest{
				ProtoFiles: []string{"test.proto"},
				ModuleName: "module",
			},
			validate: func(t *testing.T, req *ValidationRequest) {
				assert.Empty(t, req.Version)
				assert.NotEmpty(t, req.ModuleName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantNil {
				assert.NotNil(t, tt.req)
			}
			if tt.validate != nil {
				tt.validate(t, tt.req)
			}
		})
	}
}

// TestValidationResult tests the ValidationResult struct
func TestValidationResult(t *testing.T) {
	tests := []struct {
		name     string
		result   *ValidationResult
		validate func(*testing.T, *ValidationResult)
	}{
		{
			name: "valid result",
			result: &ValidationResult{
				Valid:    true,
				Errors:   nil,
				Warnings: nil,
			},
			validate: func(t *testing.T, result *ValidationResult) {
				assert.True(t, result.Valid)
				assert.Empty(t, result.Errors)
				assert.Empty(t, result.Warnings)
			},
		},
		{
			name: "invalid result with errors",
			result: &ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Field: "name", Message: "required", Severity: "error"},
					{Field: "version", Message: "invalid format", Severity: "error"},
				},
				Warnings: nil,
			},
			validate: func(t *testing.T, result *ValidationResult) {
				assert.False(t, result.Valid)
				assert.Len(t, result.Errors, 2)
				assert.Empty(t, result.Warnings)
				assert.Equal(t, "name", result.Errors[0].Field)
				assert.Equal(t, "version", result.Errors[1].Field)
			},
		},
		{
			name: "valid result with warnings",
			result: &ValidationResult{
				Valid:  true,
				Errors: nil,
				Warnings: []ValidationWarning{
					{File: "test.proto", Line: 10, Column: 5, Message: "deprecated field", Rule: "deprecation"},
					{File: "test.proto", Line: 20, Column: 1, Message: "style issue", Rule: "naming"},
				},
			},
			validate: func(t *testing.T, result *ValidationResult) {
				assert.True(t, result.Valid)
				assert.Empty(t, result.Errors)
				assert.Len(t, result.Warnings, 2)
				assert.Equal(t, "test.proto", result.Warnings[0].File)
				assert.Equal(t, 10, result.Warnings[0].Line)
				assert.Equal(t, 5, result.Warnings[0].Column)
			},
		},
		{
			name: "invalid result with both errors and warnings",
			result: &ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Field: "syntax", Message: "parse error", Severity: "error"},
				},
				Warnings: []ValidationWarning{
					{File: "foo.proto", Line: 1, Message: "missing comment", Rule: "documentation"},
				},
			},
			validate: func(t *testing.T, result *ValidationResult) {
				assert.False(t, result.Valid)
				assert.Len(t, result.Errors, 1)
				assert.Len(t, result.Warnings, 1)
			},
		},
		{
			name: "empty result",
			result: &ValidationResult{
				Valid:    false,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			},
			validate: func(t *testing.T, result *ValidationResult) {
				assert.False(t, result.Valid)
				assert.Empty(t, result.Errors)
				assert.Empty(t, result.Warnings)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.result)
			if tt.validate != nil {
				tt.validate(t, tt.result)
			}
		})
	}
}

// TestValidationWarning tests the ValidationWarning struct
func TestValidationWarning(t *testing.T) {
	tests := []struct {
		name    string
		warning ValidationWarning
		check   func(*testing.T, ValidationWarning)
	}{
		{
			name: "complete warning",
			warning: ValidationWarning{
				File:    "service.proto",
				Line:    42,
				Column:  15,
				Message: "Field name should be snake_case",
				Rule:    "FIELD_NAMES_LOWER_SNAKE_CASE",
			},
			check: func(t *testing.T, w ValidationWarning) {
				assert.Equal(t, "service.proto", w.File)
				assert.Equal(t, 42, w.Line)
				assert.Equal(t, 15, w.Column)
				assert.NotEmpty(t, w.Message)
				assert.NotEmpty(t, w.Rule)
			},
		},
		{
			name: "warning with zero line",
			warning: ValidationWarning{
				File:    "test.proto",
				Line:    0,
				Column:  0,
				Message: "File-level warning",
				Rule:    "FILE_LEVEL",
			},
			check: func(t *testing.T, w ValidationWarning) {
				assert.Equal(t, 0, w.Line)
				assert.Equal(t, 0, w.Column)
			},
		},
		{
			name: "warning with long message",
			warning: ValidationWarning{
				File:    "api.proto",
				Line:    100,
				Column:  1,
				Message: "This is a very long warning message that explains in detail what the issue is and how to fix it. It should be preserved correctly.",
				Rule:    "VERBOSE_RULE",
			},
			check: func(t *testing.T, w ValidationWarning) {
				assert.Greater(t, len(w.Message), 50)
				assert.Contains(t, w.Message, "very long warning")
			},
		},
		{
			name: "warning with special characters in file",
			warning: ValidationWarning{
				File:    "path/to/my-service.v1.proto",
				Line:    5,
				Column:  10,
				Message: "Warning message",
				Rule:    "RULE_NAME",
			},
			check: func(t *testing.T, w ValidationWarning) {
				assert.Contains(t, w.File, "/")
				assert.Contains(t, w.File, "-")
				assert.Contains(t, w.File, ".")
			},
		},
		{
			name: "empty warning",
			warning: ValidationWarning{
				File:    "",
				Line:    0,
				Column:  0,
				Message: "",
				Rule:    "",
			},
			check: func(t *testing.T, w ValidationWarning) {
				assert.Empty(t, w.File)
				assert.Empty(t, w.Message)
				assert.Empty(t, w.Rule)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.check != nil {
				tt.check(t, tt.warning)
			}
		})
	}
}

// TestValidationRequestOptionsHandling tests option handling
func TestValidationRequestOptionsHandling(t *testing.T) {
	req := &ValidationRequest{
		ProtoFiles: []string{"test.proto"},
		ModuleName: "test",
		Version:    "1.0.0",
		Options:    make(map[string]string),
	}

	// Test adding options
	req.Options["strict"] = "true"
	req.Options["max-warnings"] = "10"
	req.Options["format"] = "json"

	assert.Len(t, req.Options, 3)
	assert.Equal(t, "true", req.Options["strict"])
	assert.Equal(t, "10", req.Options["max-warnings"])
	assert.Equal(t, "json", req.Options["format"])

	// Test overwriting option
	req.Options["strict"] = "false"
	assert.Equal(t, "false", req.Options["strict"])

	// Test deleting option
	delete(req.Options, "max-warnings")
	assert.Len(t, req.Options, 2)
	assert.NotContains(t, req.Options, "max-warnings")
}

// TestValidationResultMultipleErrors tests handling multiple errors
func TestValidationResultMultipleErrors(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{Field: "id", Message: "required", Severity: "error"},
			{Field: "name", Message: "too short", Severity: "error"},
			{Field: "version", Message: "invalid format", Severity: "error"},
			{Field: "type", Message: "unknown type", Severity: "error"},
			{Field: "permissions", Message: "invalid permission", Severity: "error"},
		},
		Warnings: []ValidationWarning{
			{File: "a.proto", Line: 1, Message: "warning 1", Rule: "rule1"},
			{File: "b.proto", Line: 2, Message: "warning 2", Rule: "rule2"},
		},
	}

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 5)
	assert.Len(t, result.Warnings, 2)

	// Check error fields
	errorFields := make([]string, len(result.Errors))
	for i, err := range result.Errors {
		errorFields[i] = err.Field
	}
	assert.Contains(t, errorFields, "id")
	assert.Contains(t, errorFields, "name")
	assert.Contains(t, errorFields, "version")
	assert.Contains(t, errorFields, "type")
	assert.Contains(t, errorFields, "permissions")
}

// TestValidationWarningRuleTypes tests different rule types
func TestValidationWarningRuleTypes(t *testing.T) {
	ruleTypes := []struct {
		rule        string
		description string
	}{
		{"FIELD_NAMES_LOWER_SNAKE_CASE", "Field naming convention"},
		{"MESSAGES_HAVE_COMMENTS", "Documentation requirement"},
		{"SYNTAX_PROTO3", "Syntax version check"},
		{"PACKAGE_IS_DECLARED", "Package declaration"},
		{"SERVICE_NAMES_CAPITALIZED", "Service naming convention"},
		{"RPC_NAMES_CAPITALIZED", "RPC naming convention"},
		{"ENUM_ZERO_VALUES_INVALID", "Enum validation"},
		{"IMPORTS_NOT_WEAK", "Import type check"},
		{"FILE_OPTIONS_REQUIRE_GO_PACKAGE", "Go package option"},
		{"CUSTOM_RULE_1", "Custom validation rule"},
	}

	for _, rt := range ruleTypes {
		t.Run(rt.rule, func(t *testing.T) {
			warning := ValidationWarning{
				File:    "test.proto",
				Line:    1,
				Column:  1,
				Message: rt.description,
				Rule:    rt.rule,
			}

			assert.Equal(t, rt.rule, warning.Rule)
			assert.NotEmpty(t, warning.Message)
		})
	}
}

// TestValidationRequestWithNilOptions tests handling nil options
func TestValidationRequestWithNilOptions(t *testing.T) {
	req := &ValidationRequest{
		ProtoFiles: []string{"test.proto"},
		ModuleName: "module",
		Version:    "1.0.0",
		Options:    nil,
	}

	assert.NotNil(t, req)
	assert.Nil(t, req.Options)

	// Should be able to initialize options
	req.Options = make(map[string]string)
	req.Options["key"] = "value"
	assert.NotNil(t, req.Options)
	assert.Equal(t, "value", req.Options["key"])
}

// TestValidationResultSeverityHandling tests error severity handling
func TestValidationResultSeverityHandling(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{Field: "critical", Message: "critical error", Severity: "critical"},
			{Field: "error", Message: "error message", Severity: "error"},
			{Field: "warning", Message: "warning message", Severity: "warning"},
			{Field: "info", Message: "info message", Severity: "info"},
		},
	}

	// Count by severity
	severityCounts := make(map[string]int)
	for _, err := range result.Errors {
		severityCounts[err.Severity]++
	}

	assert.Equal(t, 1, severityCounts["critical"])
	assert.Equal(t, 1, severityCounts["error"])
	assert.Equal(t, 1, severityCounts["warning"])
	assert.Equal(t, 1, severityCounts["info"])
}

// TestValidationWarningFilePathHandling tests various file path formats
func TestValidationWarningFilePathHandling(t *testing.T) {
	testPaths := []string{
		"simple.proto",
		"path/to/file.proto",
		"deep/nested/path/to/file.proto",
		"./relative/path.proto",
		"../parent/path.proto",
		"windows\\style\\path.proto",
		"proto/v1/service.proto",
		"api/v2/types/message.proto",
	}

	for _, path := range testPaths {
		t.Run(path, func(t *testing.T) {
			warning := ValidationWarning{
				File:    path,
				Line:    1,
				Column:  1,
				Message: "test warning",
				Rule:    "TEST_RULE",
			}

			assert.Equal(t, path, warning.File)
			assert.NotEmpty(t, warning.File)
		})
	}
}

// TestValidationRequestProtoFilesOrder tests that proto file order is preserved
func TestValidationRequestProtoFilesOrder(t *testing.T) {
	files := []string{
		"first.proto",
		"second.proto",
		"third.proto",
		"fourth.proto",
		"fifth.proto",
	}

	req := &ValidationRequest{
		ProtoFiles: files,
		ModuleName: "test",
		Version:    "1.0.0",
	}

	assert.Len(t, req.ProtoFiles, 5)
	for i, expected := range files {
		assert.Equal(t, expected, req.ProtoFiles[i], "File order should be preserved at index %d", i)
	}
}

// MockValidatorPlugin is a mock implementation for testing
type MockValidatorPlugin struct {
	manifest      *Manifest
	validateFunc  func(context.Context, *ValidationRequest) (*ValidationResult, error)
	loadCalled    bool
	unloadCalled  bool
	validateCalls int
}

func (m *MockValidatorPlugin) Manifest() *Manifest {
	return m.manifest
}

func (m *MockValidatorPlugin) Load() error {
	m.loadCalled = true
	return nil
}

func (m *MockValidatorPlugin) Unload() error {
	m.unloadCalled = true
	return nil
}

func (m *MockValidatorPlugin) Validate(ctx context.Context, req *ValidationRequest) (*ValidationResult, error) {
	m.validateCalls++
	if m.validateFunc != nil {
		return m.validateFunc(ctx, req)
	}
	return &ValidationResult{Valid: true}, nil
}

// TestMockValidatorPlugin tests the mock validator plugin
func TestMockValidatorPlugin(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-validator",
		Name:    "Test Validator",
		Version: "1.0.0",
		Type:    PluginTypeValidator,
	}

	mock := &MockValidatorPlugin{
		manifest: manifest,
	}

	// Test Manifest
	assert.Equal(t, manifest, mock.Manifest())

	// Test Load
	err := mock.Load()
	assert.NoError(t, err)
	assert.True(t, mock.loadCalled)

	// Test Validate
	ctx := context.Background()
	req := &ValidationRequest{
		ProtoFiles: []string{"test.proto"},
		ModuleName: "test",
		Version:    "1.0.0",
	}

	result, err := mock.Validate(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Equal(t, 1, mock.validateCalls)

	// Test Unload
	err = mock.Unload()
	assert.NoError(t, err)
	assert.True(t, mock.unloadCalled)
}

// TestMockValidatorPluginCustomValidation tests custom validation logic
func TestMockValidatorPluginCustomValidation(t *testing.T) {
	tests := []struct {
		name         string
		validateFunc func(context.Context, *ValidationRequest) (*ValidationResult, error)
		request      *ValidationRequest
		expectValid  bool
		expectError  bool
	}{
		{
			name: "always valid",
			validateFunc: func(ctx context.Context, req *ValidationRequest) (*ValidationResult, error) {
				return &ValidationResult{Valid: true}, nil
			},
			request: &ValidationRequest{
				ProtoFiles: []string{"test.proto"},
			},
			expectValid: true,
			expectError: false,
		},
		{
			name: "validation with errors",
			validateFunc: func(ctx context.Context, req *ValidationRequest) (*ValidationResult, error) {
				return &ValidationResult{
					Valid: false,
					Errors: []ValidationError{
						{Field: "proto_files", Message: "no files provided", Severity: "error"},
					},
				}, nil
			},
			request:     &ValidationRequest{},
			expectValid: false,
			expectError: false,
		},
		{
			name: "validation with warnings",
			validateFunc: func(ctx context.Context, req *ValidationRequest) (*ValidationResult, error) {
				return &ValidationResult{
					Valid: true,
					Warnings: []ValidationWarning{
						{File: "test.proto", Line: 1, Message: "style issue", Rule: "STYLE"},
					},
				}, nil
			},
			request: &ValidationRequest{
				ProtoFiles: []string{"test.proto"},
			},
			expectValid: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockValidatorPlugin{
				manifest: &Manifest{
					ID:   "test",
					Type: PluginTypeValidator,
				},
				validateFunc: tt.validateFunc,
			}

			result, err := mock.Validate(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectValid, result.Valid)
			}
		})
	}
}

// TestValidatorPluginInterfaceCompliance tests that mock implements the interface
func TestValidatorPluginInterfaceCompliance(t *testing.T) {
	var _ ValidatorPlugin = (*MockValidatorPlugin)(nil)

	// If this compiles, the mock implements the interface correctly
	t.Log("MockValidatorPlugin correctly implements ValidatorPlugin interface")
}

// TestValidationRequestClone tests cloning a validation request
func TestValidationRequestClone(t *testing.T) {
	original := &ValidationRequest{
		ProtoFiles: []string{"a.proto", "b.proto"},
		ModuleName: "module",
		Version:    "1.0.0",
		Options: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Create a clone manually (since there's no built-in clone method)
	clone := &ValidationRequest{
		ProtoFiles: make([]string, len(original.ProtoFiles)),
		ModuleName: original.ModuleName,
		Version:    original.Version,
		Options:    make(map[string]string),
	}
	copy(clone.ProtoFiles, original.ProtoFiles)
	for k, v := range original.Options {
		clone.Options[k] = v
	}

	// Verify clone is equal but separate
	assert.Equal(t, original.ModuleName, clone.ModuleName)
	assert.Equal(t, original.Version, clone.Version)
	assert.Equal(t, len(original.ProtoFiles), len(clone.ProtoFiles))
	assert.Equal(t, len(original.Options), len(clone.Options))

	// Modify clone and verify original is unchanged
	clone.ProtoFiles = append(clone.ProtoFiles, "c.proto")
	clone.Options["key3"] = "value3"

	assert.NotEqual(t, len(original.ProtoFiles), len(clone.ProtoFiles))
	assert.NotEqual(t, len(original.Options), len(clone.Options))
}

// TestValidationResultErrorFiltering tests filtering errors by severity
func TestValidationResultErrorFiltering(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{Field: "f1", Message: "critical", Severity: "critical"},
			{Field: "f2", Message: "error", Severity: "error"},
			{Field: "f3", Message: "error", Severity: "error"},
			{Field: "f4", Message: "warning", Severity: "warning"},
			{Field: "f5", Message: "info", Severity: "info"},
		},
	}

	// Filter by severity
	filterBySeverity := func(errors []ValidationError, severity string) []ValidationError {
		var filtered []ValidationError
		for _, err := range errors {
			if err.Severity == severity {
				filtered = append(filtered, err)
			}
		}
		return filtered
	}

	critical := filterBySeverity(result.Errors, "critical")
	errors := filterBySeverity(result.Errors, "error")
	warnings := filterBySeverity(result.Errors, "warning")
	info := filterBySeverity(result.Errors, "info")

	assert.Len(t, critical, 1)
	assert.Len(t, errors, 2)
	assert.Len(t, warnings, 1)
	assert.Len(t, info, 1)
}

// TestValidationWarningGrouping tests grouping warnings by file
func TestValidationWarningGrouping(t *testing.T) {
	warnings := []ValidationWarning{
		{File: "a.proto", Line: 1, Message: "w1", Rule: "R1"},
		{File: "a.proto", Line: 2, Message: "w2", Rule: "R2"},
		{File: "b.proto", Line: 1, Message: "w3", Rule: "R1"},
		{File: "b.proto", Line: 2, Message: "w4", Rule: "R2"},
		{File: "c.proto", Line: 1, Message: "w5", Rule: "R1"},
	}

	// Group by file
	grouped := make(map[string][]ValidationWarning)
	for _, w := range warnings {
		grouped[w.File] = append(grouped[w.File], w)
	}

	assert.Len(t, grouped, 3)
	assert.Len(t, grouped["a.proto"], 2)
	assert.Len(t, grouped["b.proto"], 2)
	assert.Len(t, grouped["c.proto"], 1)
}

// TestValidationRequestValidation tests validation of the request itself
func TestValidationRequestValidation(t *testing.T) {
	tests := []struct {
		name      string
		req       *ValidationRequest
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid request",
			req: &ValidationRequest{
				ProtoFiles: []string{"test.proto"},
				ModuleName: "module",
				Version:    "1.0.0",
			},
			expectErr: false,
		},
		{
			name: "missing proto files",
			req: &ValidationRequest{
				ModuleName: "module",
				Version:    "1.0.0",
			},
			expectErr: true,
			errMsg:    "proto files required",
		},
		{
			name: "empty proto files",
			req: &ValidationRequest{
				ProtoFiles: []string{},
				ModuleName: "module",
				Version:    "1.0.0",
			},
			expectErr: true,
			errMsg:    "proto files required",
		},
	}

	// Simple validator function
	validateRequest := func(req *ValidationRequest) error {
		if req.ProtoFiles == nil || len(req.ProtoFiles) == 0 {
			return assert.AnError
		}
		return nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequest(tt.req)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
