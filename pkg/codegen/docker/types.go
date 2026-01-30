package docker

import (
	"context"
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/config"
)

// Runner executes protoc compilation in Docker containers
type Runner interface {
	// Execute runs a compilation in a Docker container
	Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error)

	// PullImage ensures the Docker image is available locally
	PullImage(ctx context.Context, image string) error

	// Cleanup removes stopped containers and unused images
	Cleanup(ctx context.Context) error

	// Close releases resources
	Close() error
}

// ExecutionRequest represents a Docker execution request
type ExecutionRequest struct {
	// Docker configuration
	Image         string
	Tag           string

	// Input files
	ProtoFiles    []codegen.ProtoFile
	WorkDir       string // Working directory inside container

	// Protoc command
	ProtocFlags   []string
	OutputDir     string // Output directory inside container

	// Resource limits
	MemoryLimit   int64         // Memory limit in bytes (see config.DefaultDockerMemoryLimit)
	CPULimit      float64       // CPU limit (see config.DefaultDockerCPULimit)
	Timeout       time.Duration // Execution timeout (see config.DefaultDockerTimeout)

	// Environment variables
	Env           map[string]string
}

// ExecutionResult represents the result of a Docker execution
type ExecutionResult struct {
	Success       bool
	ExitCode      int
	Stdout        string
	Stderr        string
	Duration      time.Duration
	GeneratedFiles []codegen.GeneratedFile
	Error         error
}

// ResourceLimits defines default resource limits
var (
	DefaultMemoryLimit = config.DefaultDockerMemoryLimit
	DefaultCPULimit    = config.DefaultDockerCPULimit
	DefaultTimeout     = config.DefaultDockerTimeout
)
