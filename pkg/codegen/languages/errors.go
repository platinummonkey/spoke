package languages

import "errors"

var (
	// ErrLanguageNotFound is returned when a language is not found in the registry
	ErrLanguageNotFound = errors.New("language not found")

	// ErrLanguageAlreadyExists is returned when trying to register a duplicate language
	ErrLanguageAlreadyExists = errors.New("language already exists")

	// ErrLanguageDisabled is returned when trying to use a disabled language
	ErrLanguageDisabled = errors.New("language is disabled")

	// ErrInvalidLanguageID is returned when a language ID is invalid
	ErrInvalidLanguageID = errors.New("invalid language ID")

	// ErrInvalidLanguageName is returned when a language name is invalid
	ErrInvalidLanguageName = errors.New("invalid language name")

	// ErrInvalidDockerImage is returned when a Docker image is invalid
	ErrInvalidDockerImage = errors.New("invalid Docker image")

	// ErrInvalidProtocPlugin is returned when a protoc plugin is invalid
	ErrInvalidProtocPlugin = errors.New("invalid protoc plugin")
)
