package plugins

import (
	"context"
)

// ValidatorPlugin for schema linting and validation
type ValidatorPlugin interface {
	Plugin
	Validate(ctx context.Context, req *ValidationRequest) (*ValidationResult, error)
}

// ValidationRequest contains files and metadata to validate
type ValidationRequest struct {
	ProtoFiles []string          `json:"proto_files"`
	ModuleName string            `json:"module_name"`
	Version    string            `json:"version"`
	Options    map[string]string `json:"options"`
}

// ValidationResult contains validation errors and warnings
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors"`
	Warnings []ValidationWarning `json:"warnings"`
}

// ValidationWarning represents a non-blocking validation issue
type ValidationWarning struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
	Rule    string `json:"rule"`
}
