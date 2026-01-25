package orchestrator

import "errors"

var (
	// ErrNoProtoFiles is returned when no proto files are provided
	ErrNoProtoFiles = errors.New("no proto files provided")

	// ErrLanguageNotSupported is returned when a language is not supported
	ErrLanguageNotSupported = errors.New("language not supported")

	// ErrCompilationFailed is returned when compilation fails
	ErrCompilationFailed = errors.New("compilation failed")

	// ErrJobNotFound is returned when a compilation job is not found
	ErrJobNotFound = errors.New("compilation job not found")

	// ErrDependencyNotFound is returned when a dependency is not found
	ErrDependencyNotFound = errors.New("dependency not found")
)
