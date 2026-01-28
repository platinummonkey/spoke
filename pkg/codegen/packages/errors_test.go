package packages

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrGeneratorNotFound(t *testing.T) {
	assert.NotNil(t, ErrGeneratorNotFound)
	assert.Equal(t, "package generator not found", ErrGeneratorNotFound.Error())
}

func TestErrInvalidTemplate(t *testing.T) {
	assert.NotNil(t, ErrInvalidTemplate)
	assert.Equal(t, "invalid template", ErrInvalidTemplate.Error())
}

func TestErrTemplateExecutionFailed(t *testing.T) {
	assert.NotNil(t, ErrTemplateExecutionFailed)
	assert.Equal(t, "template execution failed", ErrTemplateExecutionFailed.Error())
}

func TestErrMissingRequiredField(t *testing.T) {
	assert.NotNil(t, ErrMissingRequiredField)
	assert.Equal(t, "missing required field", ErrMissingRequiredField.Error())
}

func TestErrorComparison(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected error
		match    bool
	}{
		{
			name:     "match generator not found",
			err:      ErrGeneratorNotFound,
			expected: ErrGeneratorNotFound,
			match:    true,
		},
		{
			name:     "match invalid template",
			err:      ErrInvalidTemplate,
			expected: ErrInvalidTemplate,
			match:    true,
		},
		{
			name:     "match template execution failed",
			err:      ErrTemplateExecutionFailed,
			expected: ErrTemplateExecutionFailed,
			match:    true,
		},
		{
			name:     "match missing required field",
			err:      ErrMissingRequiredField,
			expected: ErrMissingRequiredField,
			match:    true,
		},
		{
			name:     "no match different errors",
			err:      ErrGeneratorNotFound,
			expected: ErrInvalidTemplate,
			match:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.match {
				assert.True(t, errors.Is(tt.err, tt.expected))
			} else {
				assert.False(t, errors.Is(tt.err, tt.expected))
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	tests := []struct {
		name        string
		baseErr     error
		wrappedErr  error
		shouldMatch bool
	}{
		{
			name:        "wrapped generator not found",
			baseErr:     ErrGeneratorNotFound,
			wrappedErr:  errors.Join(ErrGeneratorNotFound, errors.New("additional context")),
			shouldMatch: true,
		},
		{
			name:        "wrapped invalid template",
			baseErr:     ErrInvalidTemplate,
			wrappedErr:  errors.Join(ErrInvalidTemplate, errors.New("template parse error")),
			shouldMatch: true,
		},
		{
			name:        "wrapped template execution failed",
			baseErr:     ErrTemplateExecutionFailed,
			wrappedErr:  errors.Join(ErrTemplateExecutionFailed, errors.New("execution context")),
			shouldMatch: true,
		},
		{
			name:        "wrapped missing required field",
			baseErr:     ErrMissingRequiredField,
			wrappedErr:  errors.Join(ErrMissingRequiredField, errors.New("field: name")),
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, errors.Is(tt.wrappedErr, tt.baseErr))
		})
	}
}

func TestAllErrorsAreUnique(t *testing.T) {
	allErrors := []error{
		ErrGeneratorNotFound,
		ErrInvalidTemplate,
		ErrTemplateExecutionFailed,
		ErrMissingRequiredField,
	}

	// Verify all errors are unique
	for i, err1 := range allErrors {
		for j, err2 := range allErrors {
			if i != j {
				assert.NotEqual(t, err1, err2, "errors at index %d and %d should be different", i, j)
			}
		}
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedMessage string
	}{
		{
			name:            "generator not found message",
			err:             ErrGeneratorNotFound,
			expectedMessage: "package generator not found",
		},
		{
			name:            "invalid template message",
			err:             ErrInvalidTemplate,
			expectedMessage: "invalid template",
		},
		{
			name:            "template execution failed message",
			err:             ErrTemplateExecutionFailed,
			expectedMessage: "template execution failed",
		},
		{
			name:            "missing required field message",
			err:             ErrMissingRequiredField,
			expectedMessage: "missing required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedMessage, tt.err.Error())
		})
	}
}

func TestIsGeneratorNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "is generator not found error",
			err:      ErrGeneratorNotFound,
			expected: true,
		},
		{
			name:     "is wrapped generator not found error",
			err:      NewGeneratorNotFoundError("test-generator"),
			expected: true,
		},
		{
			name:     "is not generator not found error",
			err:      ErrInvalidTemplate,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsGeneratorNotFoundError(tt.err))
		})
	}
}

func TestIsInvalidTemplateError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "is invalid template error",
			err:      ErrInvalidTemplate,
			expected: true,
		},
		{
			name:     "is wrapped invalid template error",
			err:      NewInvalidTemplateError("test.tmpl"),
			expected: true,
		},
		{
			name:     "is not invalid template error",
			err:      ErrGeneratorNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsInvalidTemplateError(tt.err))
		})
	}
}

func TestIsTemplateExecutionFailedError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "is template execution failed error",
			err:      ErrTemplateExecutionFailed,
			expected: true,
		},
		{
			name:     "is wrapped template execution failed error",
			err:      NewTemplateExecutionFailedError("test.tmpl", errors.New("parse error")),
			expected: true,
		},
		{
			name:     "is not template execution failed error",
			err:      ErrGeneratorNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsTemplateExecutionFailedError(tt.err))
		})
	}
}

func TestIsMissingRequiredFieldError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "is missing required field error",
			err:      ErrMissingRequiredField,
			expected: true,
		},
		{
			name:     "is wrapped missing required field error",
			err:      NewMissingRequiredFieldError("moduleName"),
			expected: true,
		},
		{
			name:     "is not missing required field error",
			err:      ErrGeneratorNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsMissingRequiredFieldError(tt.err))
		})
	}
}

func TestNewGeneratorNotFoundError(t *testing.T) {
	err := NewGeneratorNotFoundError("test-generator")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "package generator not found")
	assert.Contains(t, err.Error(), "test-generator")
	assert.True(t, errors.Is(err, ErrGeneratorNotFound))
}

func TestNewInvalidTemplateError(t *testing.T) {
	err := NewInvalidTemplateError("test.tmpl")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid template")
	assert.Contains(t, err.Error(), "test.tmpl")
	assert.True(t, errors.Is(err, ErrInvalidTemplate))
}

func TestNewTemplateExecutionFailedError(t *testing.T) {
	cause := errors.New("parse error")
	err := NewTemplateExecutionFailedError("test.tmpl", cause)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "template execution failed")
	assert.Contains(t, err.Error(), "test.tmpl")
	assert.Contains(t, err.Error(), "parse error")
	assert.True(t, errors.Is(err, ErrTemplateExecutionFailed))
}

func TestNewMissingRequiredFieldError(t *testing.T) {
	err := NewMissingRequiredFieldError("moduleName")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "missing required field")
	assert.Contains(t, err.Error(), "moduleName")
	assert.True(t, errors.Is(err, ErrMissingRequiredField))
}
