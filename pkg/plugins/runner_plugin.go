package plugins

import (
	"context"
	"time"
)

// RunnerPlugin for custom execution environments
type RunnerPlugin interface {
	Plugin
	Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error)
}

// ExecutionRequest contains command and environment to execute
type ExecutionRequest struct {
	Command     []string          `json:"command"`
	WorkingDir  string            `json:"working_dir"`
	Environment map[string]string `json:"environment"`
	Timeout     time.Duration     `json:"timeout"`
	Stdin       []byte            `json:"stdin,omitempty"`
}

// ExecutionResult contains execution output and status
type ExecutionResult struct {
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
}
