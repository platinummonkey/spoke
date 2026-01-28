package packages

import (
	"errors"
	"fmt"
)

var (
	// ErrGeneratorNotFound is returned when a package generator is not found
	ErrGeneratorNotFound = errors.New("package generator not found")

	// ErrInvalidTemplate is returned when a template is invalid
	ErrInvalidTemplate = errors.New("invalid template")

	// ErrTemplateExecutionFailed is returned when template execution fails
	ErrTemplateExecutionFailed = errors.New("template execution failed")

	// ErrMissingRequiredField is returned when a required field is missing
	ErrMissingRequiredField = errors.New("missing required field")
)

// IsGeneratorNotFoundError checks if the error is or wraps ErrGeneratorNotFound
func IsGeneratorNotFoundError(err error) bool {
	return errors.Is(err, ErrGeneratorNotFound)
}

// IsInvalidTemplateError checks if the error is or wraps ErrInvalidTemplate
func IsInvalidTemplateError(err error) bool {
	return errors.Is(err, ErrInvalidTemplate)
}

// IsTemplateExecutionFailedError checks if the error is or wraps ErrTemplateExecutionFailed
func IsTemplateExecutionFailedError(err error) bool {
	return errors.Is(err, ErrTemplateExecutionFailed)
}

// IsMissingRequiredFieldError checks if the error is or wraps ErrMissingRequiredField
func IsMissingRequiredFieldError(err error) bool {
	return errors.Is(err, ErrMissingRequiredField)
}

// NewGeneratorNotFoundError creates a new generator not found error with context
func NewGeneratorNotFoundError(generatorName string) error {
	return fmt.Errorf("%w: %s", ErrGeneratorNotFound, generatorName)
}

// NewInvalidTemplateError creates a new invalid template error with context
func NewInvalidTemplateError(templateName string) error {
	return fmt.Errorf("%w: %s", ErrInvalidTemplate, templateName)
}

// NewTemplateExecutionFailedError creates a new template execution failed error with context
func NewTemplateExecutionFailedError(templateName string, cause error) error {
	return fmt.Errorf("%w for template %s: %v", ErrTemplateExecutionFailed, templateName, cause)
}

// NewMissingRequiredFieldError creates a new missing required field error with field name
func NewMissingRequiredFieldError(fieldName string) error {
	return fmt.Errorf("%w: %s", ErrMissingRequiredField, fieldName)
}
