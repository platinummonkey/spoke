package rules

import (
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/linter"
)

// Rule interface that all lint rules must implement
type Rule interface {
	Name() string
	Category() linter.Category
	Severity() linter.Severity
	Description() string
	Check(node *protobuf.RootNode, ctx *linter.LintContext) []linter.Violation
	CanAutoFix() bool
	AutoFix(node *protobuf.RootNode, violation linter.Violation) (*linter.Fix, error)
}

// BaseRule provides common functionality for rules
type BaseRule struct {
	RuleName        string
	RuleCategory    linter.Category
	RuleSeverity    linter.Severity
	RuleDescription string
	AutoFixable     bool
}

func (r *BaseRule) Name() string                    { return r.RuleName }
func (r *BaseRule) Category() linter.Category       { return r.RuleCategory }
func (r *BaseRule) Severity() linter.Severity       { return r.RuleSeverity }
func (r *BaseRule) Description() string             { return r.RuleDescription }
func (r *BaseRule) CanAutoFix() bool                { return r.AutoFixable }

// Default AutoFix returns not implemented
func (r *BaseRule) AutoFix(node *protobuf.RootNode, violation linter.Violation) (*linter.Fix, error) {
	return nil, nil
}
