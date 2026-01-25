package docker

import "errors"

var (
	// ErrDockerNotAvailable is returned when Docker is not available
	ErrDockerNotAvailable = errors.New("docker is not available")

	// ErrImagePullFailed is returned when image pull fails
	ErrImagePullFailed = errors.New("failed to pull docker image")

	// ErrContainerFailed is returned when container execution fails
	ErrContainerFailed = errors.New("container execution failed")

	// ErrTimeout is returned when execution times out
	ErrTimeout = errors.New("execution timeout")

	// ErrResourceLimit is returned when resource limit is exceeded
	ErrResourceLimit = errors.New("resource limit exceeded")

	// ErrNoGeneratedFiles is returned when no files were generated
	ErrNoGeneratedFiles = errors.New("no files were generated")
)
