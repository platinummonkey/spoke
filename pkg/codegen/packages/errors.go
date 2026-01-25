package packages

import "errors"

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
