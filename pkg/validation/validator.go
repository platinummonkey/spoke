package validation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// Validator performs semantic validation on protobuf schemas
type Validator struct {
	config *ValidationConfig
}

// ValidationConfig defines validation rules
type ValidationConfig struct {
	// EnforceFieldNumberRanges checks field numbers are in valid ranges
	EnforceFieldNumberRanges bool
	// RequireEnumZeroValue requires enums to have a zero value
	RequireEnumZeroValue bool
	// CheckNamingConventions validates naming follows proto style guide
	CheckNamingConventions bool
	// DetectCircularDependencies checks for import cycles
	DetectCircularDependencies bool
	// DetectUnusedImports checks for unused imports
	DetectUnusedImports bool
	// CheckReservedFields validates reserved field usage
	CheckReservedFields bool
	// MaxFieldNumber is the maximum allowed field number
	MaxFieldNumber int
}

// DefaultValidationConfig returns default validation settings
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		EnforceFieldNumberRanges:   true,
		RequireEnumZeroValue:       true,
		CheckNamingConventions:     true,
		DetectCircularDependencies: true,
		DetectUnusedImports:        true,
		CheckReservedFields:        true,
		MaxFieldNumber:             536870911, // 2^29 - 1
	}
}

// NewValidator creates a new validator
func NewValidator(config *ValidationConfig) *Validator {
	if config == nil {
		config = DefaultValidationConfig()
	}
	return &Validator{config: config}
}

// ValidationError represents a validation error
type ValidationError struct {
	Location string
	Rule     string
	Message  string
	Severity Severity
}

// Severity indicates the severity of a validation error
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInfo
)

func (s Severity) String() string {
	return []string{"ERROR", "WARNING", "INFO"}[s]
}

// ValidationResult contains validation errors
type ValidationResult struct {
	Errors   []*ValidationError
	Warnings []*ValidationError
	Valid    bool
}

// Validate performs validation on a protobuf AST
func (v *Validator) Validate(ast *protobuf.RootNode) *ValidationResult {
	result := &ValidationResult{
		Errors:   make([]*ValidationError, 0),
		Warnings: make([]*ValidationError, 0),
		Valid:    true,
	}

	// Check package naming
	if ast.Package != nil {
		v.validatePackageName(ast.Package, result)
	}

	// Validate messages
	for _, msg := range ast.Messages {
		v.validateMessage(msg, "", result)
	}

	// Validate enums
	for _, enum := range ast.Enums {
		v.validateEnum(enum, "", result)
	}

	// Validate services
	for _, svc := range ast.Services {
		v.validateService(svc, result)
	}

	// Check for unused imports
	if v.config.DetectUnusedImports {
		v.detectUnusedImports(ast, result)
	}

	// Set valid flag
	result.Valid = len(result.Errors) == 0

	return result
}

func (v *Validator) validatePackageName(pkg *protobuf.PackageNode, result *ValidationResult) {
	if !v.config.CheckNamingConventions {
		return
	}

	// Package names should be lowercase with dots
	if !isValidPackageName(pkg.Name) {
		result.addError("package", "INVALID_PACKAGE_NAME",
			fmt.Sprintf("Package name %q should be lowercase with dots (e.g., com.example.api)", pkg.Name))
	}
}

func (v *Validator) validateMessage(msg *protobuf.MessageNode, parentName string, result *ValidationResult) {
	fullName := msg.Name
	if parentName != "" {
		fullName = parentName + "." + msg.Name
	}

	// Check message naming convention
	if v.config.CheckNamingConventions && !isPascalCase(msg.Name) {
		result.addWarning(fullName, "MESSAGE_NAME_CONVENTION",
			fmt.Sprintf("Message name %q should be PascalCase", msg.Name))
	}

	// Track field numbers
	fieldNumbers := make(map[int]string)

	// Validate fields
	for _, field := range msg.Fields {
		v.validateField(field, fullName, fieldNumbers, result)
	}

	// Validate oneofs
	for _, oneof := range msg.OneOfs {
		for _, field := range oneof.Fields {
			v.validateField(field, fullName+"."+oneof.Name, fieldNumbers, result)
		}
	}

	// Validate nested messages
	for _, nested := range msg.Nested {
		v.validateMessage(nested, fullName, result)
	}

	// Validate nested enums
	for _, enum := range msg.Enums {
		v.validateEnum(enum, fullName, result)
	}
}

func (v *Validator) validateField(field *protobuf.FieldNode, parentName string, fieldNumbers map[int]string, result *ValidationResult) {
	location := fmt.Sprintf("%s.%s", parentName, field.Name)

	// Check field naming convention
	if v.config.CheckNamingConventions && !isSnakeCase(field.Name) {
		result.addWarning(location, "FIELD_NAME_CONVENTION",
			fmt.Sprintf("Field name %q should be snake_case", field.Name))
	}

	// Check field number ranges
	if v.config.EnforceFieldNumberRanges {
		if field.Number < 1 {
			result.addError(location, "INVALID_FIELD_NUMBER",
				fmt.Sprintf("Field number %d is invalid (must be >= 1)", field.Number))
		}

		if field.Number > v.config.MaxFieldNumber {
			result.addError(location, "INVALID_FIELD_NUMBER",
				fmt.Sprintf("Field number %d exceeds maximum %d", field.Number, v.config.MaxFieldNumber))
		}

		// Check reserved range (19000-19999 reserved by protobuf)
		if field.Number >= 19000 && field.Number <= 19999 {
			result.addError(location, "RESERVED_FIELD_NUMBER",
				fmt.Sprintf("Field number %d is in reserved range (19000-19999)", field.Number))
		}
	}

	// Check for duplicate field numbers
	if existingField, exists := fieldNumbers[field.Number]; exists {
		result.addError(location, "DUPLICATE_FIELD_NUMBER",
			fmt.Sprintf("Field number %d is already used by field %q", field.Number, existingField))
	} else {
		fieldNumbers[field.Number] = field.Name
	}
}

func (v *Validator) validateEnum(enum *protobuf.EnumNode, parentName string, result *ValidationResult) {
	fullName := enum.Name
	if parentName != "" {
		fullName = parentName + "." + enum.Name
	}

	// Check enum naming convention
	if v.config.CheckNamingConventions && !isPascalCase(enum.Name) {
		result.addWarning(fullName, "ENUM_NAME_CONVENTION",
			fmt.Sprintf("Enum name %q should be PascalCase", enum.Name))
	}

	// Check for zero value
	if v.config.RequireEnumZeroValue {
		hasZero := false
		for _, value := range enum.Values {
			if value.Number == 0 {
				hasZero = true
				break
			}
		}
		if !hasZero {
			result.addError(fullName, "MISSING_ENUM_ZERO_VALUE",
				"Enum must have a zero value (required in proto3)")
		}
	}

	// Track enum value numbers
	valueNumbers := make(map[int]string)

	// Validate enum values
	for _, value := range enum.Values {
		location := fmt.Sprintf("%s.%s", fullName, value.Name)

		// Check enum value naming convention (UPPER_SNAKE_CASE)
		if v.config.CheckNamingConventions && !isUpperSnakeCase(value.Name) {
			result.addWarning(location, "ENUM_VALUE_NAME_CONVENTION",
				fmt.Sprintf("Enum value %q should be UPPER_SNAKE_CASE", value.Name))
		}

		// Check for duplicate numbers
		if existingValue, exists := valueNumbers[value.Number]; exists {
			result.addError(location, "DUPLICATE_ENUM_VALUE_NUMBER",
				fmt.Sprintf("Enum value number %d is already used by %q", value.Number, existingValue))
		} else {
			valueNumbers[value.Number] = value.Name
		}
	}
}

func (v *Validator) validateService(svc *protobuf.ServiceNode, result *ValidationResult) {
	// Check service naming convention
	if v.config.CheckNamingConventions && !isPascalCase(svc.Name) {
		result.addWarning(svc.Name, "SERVICE_NAME_CONVENTION",
			fmt.Sprintf("Service name %q should be PascalCase", svc.Name))
	}

	// Validate RPC methods
	for _, rpc := range svc.RPCs {
		location := fmt.Sprintf("%s.%s", svc.Name, rpc.Name)

		if v.config.CheckNamingConventions && !isPascalCase(rpc.Name) {
			result.addWarning(location, "RPC_NAME_CONVENTION",
				fmt.Sprintf("RPC method name %q should be PascalCase", rpc.Name))
		}
	}
}

func (v *Validator) detectUnusedImports(ast *protobuf.RootNode, result *ValidationResult) {
	// TODO: Implement unused import detection
	// This requires tracking type usage throughout the file
	_ = ast
	_ = result
}

func (r *ValidationResult) addError(location, rule, message string) {
	r.Errors = append(r.Errors, &ValidationError{
		Location: location,
		Rule:     rule,
		Message:  message,
		Severity: SeverityError,
	})
}

func (r *ValidationResult) addWarning(location, rule, message string) {
	r.Warnings = append(r.Warnings, &ValidationError{
		Location: location,
		Rule:     rule,
		Message:  message,
		Severity: SeverityWarning,
	})
}

// Naming convention helpers

func isValidPackageName(name string) bool {
	// Package names should be lowercase with dots
	// e.g., com.example.api
	parts := strings.Split(name, ".")
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if !unicode.IsLower(ch) && !unicode.IsDigit(ch) && ch != '_' {
				return false
			}
		}
	}
	return true
}

func isPascalCase(name string) bool {
	if name == "" {
		return false
	}
	// Must start with uppercase letter
	if !unicode.IsUpper(rune(name[0])) {
		return false
	}
	// Should not contain underscores (use camelCase/PascalCase)
	return !strings.Contains(name, "_")
}

func isSnakeCase(name string) bool {
	if name == "" {
		return false
	}
	// Must be lowercase with underscores
	snakeCaseRegex := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	return snakeCaseRegex.MatchString(name)
}

func isUpperSnakeCase(name string) bool {
	if name == "" {
		return false
	}
	// Must be uppercase with underscores
	upperSnakeCaseRegex := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	return upperSnakeCaseRegex.MatchString(name)
}
