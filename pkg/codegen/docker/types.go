package docker

import (
	"context"
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen"
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
	MemoryLimit   int64         // Memory limit in bytes (default: 512MB)
	CPULimit      float64       // CPU limit (default: 1.0)
	Timeout       time.Duration // Execution timeout (default: 5 minutes)

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
	DefaultMemoryLimit = int64(512 * 1024 * 1024) // 512MB
	DefaultCPULimit    = 1.0                        // 1 CPU core
	DefaultTimeout     = 5 * time.Minute            // 5 minutes
)
