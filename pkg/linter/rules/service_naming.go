package rules

import (
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/linter"
)

// ServiceNamingRule checks that service names follow PascalCase
type ServiceNamingRule struct {
	BaseRule
}

// NewServiceNamingRule creates a new service naming rule
func NewServiceNamingRule() *ServiceNamingRule {
	return &ServiceNamingRule{
		BaseRule: BaseRule{
			RuleName:        "service-naming",
			RuleCategory:    linter.CategoryNaming,
			RuleSeverity:    linter.SeverityError,
			RuleDescription: "Service names must use PascalCase",
			AutoFixable:     true,
		},
	}
}

// Check validates service names
func (r *ServiceNamingRule) Check(node *protobuf.RootNode, ctx *linter.LintContext) []linter.Violation {
	violations := make([]linter.Violation, 0)

	for _, svc := range node.Services {
		if !isPascalCase(svc.Name) {
			pos := svc.Position()
			violations = append(violations, linter.Violation{
				Rule:     r.Name(),
				Severity: r.Severity(),
				Category: r.Category(),
				Message:  "Service name '" + svc.Name + "' should be PascalCase",
				Position: pos,
				SuggestedFix: &linter.Fix{
					Description: "Convert to PascalCase",
					Changes: []linter.Change{
						{
							FilePath: "",
							StartPos: pos,
							EndPos:   pos,
							OldText:  svc.Name,
							NewText:  toPascalCase(svc.Name),
						},
					},
				},
			})
		}
	}

	return violations
}

// AutoFix converts service names to PascalCase
func (r *ServiceNamingRule) AutoFix(node *protobuf.RootNode, violation linter.Violation) (*linter.Fix, error) {
	return violation.SuggestedFix, nil
}
