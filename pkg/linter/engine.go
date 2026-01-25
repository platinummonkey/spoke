package linter

import (
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// LintEngine orchestrates the linting process
type LintEngine struct {
	config   *Config
	registry *RuleRegistry
}

// NewLintEngine creates a new lint engine
func NewLintEngine(config *Config) *LintEngine {
	if config == nil {
		config = DefaultConfig()
	}

	return &LintEngine{
		config:   config,
		registry: NewRuleRegistry(),
	}
}

// Lint runs all enabled rules against a proto file
func (e *LintEngine) Lint(filePath string, ast *protobuf.RootNode) LintResult {
	result := LintResult{
		FilePath:   filePath,
		Violations: make([]Violation, 0),
	}

	// Get enabled rules
	rules := e.registry.GetEnabledRules(e.config)

	// Create lint context
	ctx := &LintContext{
		FilePath: filePath,
		AST:      ast,
		Config:   e.config,
	}

	// Run each rule
	for _, rule := range rules {
		violations := rule.Check(ast, ctx)
		result.Violations = append(result.Violations, violations...)
	}

	// Calculate metrics if enabled
	if e.config.Quality.Enabled {
		result.Metrics = e.calculateMetrics(ast)
	}

	return result
}

// LintFiles lints multiple files
func (e *LintEngine) LintFiles(files map[string]*protobuf.RootNode) []LintResult {
	results := make([]LintResult, 0, len(files))
	for path, ast := range files {
		result := e.Lint(path, ast)
		results = append(results, result)
	}
	return results
}

// GenerateSummary creates a summary of lint results
func (e *LintEngine) GenerateSummary(results []LintResult) Summary {
	summary := Summary{
		TotalFiles: len(results),
	}

	for _, result := range results {
		summary.TotalViolations += len(result.Violations)
		for _, v := range result.Violations {
			switch v.Severity {
			case SeverityError:
				summary.Errors++
			case SeverityWarning:
				summary.Warnings++
			case SeverityInfo:
				summary.Infos++
			}
		}
	}

	return summary
}

func (e *LintEngine) calculateMetrics(ast *protobuf.RootNode) FileMetrics {
	// TODO: Implement quality metrics calculation
	return FileMetrics{
		MessageCount:          len(ast.Messages),
		DocumentationCoverage: 0.0,
		ComplexityScore:       0.0,
	}
}

// LintResult contains the result of linting a single file
type LintResult struct {
	FilePath    string
	Violations  []Violation
	Metrics     FileMetrics
	FixedCount  int
}

// Violation represents a linting violation
type Violation struct {
	Rule         string
	Severity     Severity
	Category     Category
	Message      string
	Position     protobuf.Position
	SuggestedFix *Fix
}

// Severity indicates how serious a violation is
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Category groups related rules
type Category string

const (
	CategoryNaming        Category = "naming"
	CategoryStyle         Category = "style"
	CategoryDocumentation Category = "documentation"
	CategoryStructure     Category = "structure"
)

// Fix represents an automatic fix
type Fix struct {
	Description string
	Changes     []Change
}

// Change represents a single text change
type Change struct {
	FilePath string
	StartPos protobuf.Position
	EndPos   protobuf.Position
	OldText  string
	NewText  string
}

// FileMetrics contains quality metrics for a file
type FileMetrics struct {
	FilePath              string
	MessageCount          int
	FieldCount            int
	CommentedMessages     int
	CommentedFields       int
	DocumentationCoverage float64
	ComplexityScore       float64
}

// Summary provides an overview of all lint results
type Summary struct {
	TotalFiles      int
	TotalViolations int
	Errors          int
	Warnings        int
	Infos           int
}

// LintContext provides context during rule checking
type LintContext struct {
	FilePath string
	AST      *protobuf.RootNode
	Config   *Config
}
