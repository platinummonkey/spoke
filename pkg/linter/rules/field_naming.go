package rules

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/linter"
)

// FieldNamingRule checks that field names follow snake_case
type FieldNamingRule struct {
	BaseRule
}

// NewFieldNamingRule creates a new field naming rule
func NewFieldNamingRule() *FieldNamingRule {
	return &FieldNamingRule{
		BaseRule: BaseRule{
			RuleName:        "field-naming",
			RuleCategory:    linter.CategoryNaming,
			RuleSeverity:    linter.SeverityError,
			RuleDescription: "Field names must use snake_case",
			AutoFixable:     true,
		},
	}
}

// Check validates field names
func (r *FieldNamingRule) Check(node *protobuf.RootNode, ctx *linter.LintContext) []linter.Violation {
	violations := make([]linter.Violation, 0)

	// Check all messages
	for _, msg := range node.Messages {
		violations = append(violations, r.checkMessage(msg)...)
	}

	return violations
}

func (r *FieldNamingRule) checkMessage(msg *protobuf.MessageNode) []linter.Violation {
	violations := make([]linter.Violation, 0)

	// Check fields
	for _, field := range msg.Fields {
		if !isSnakeCase(field.Name) {
			pos := field.Position()
			violations = append(violations, linter.Violation{
				Rule:     r.Name(),
				Severity: r.Severity(),
				Category: r.Category(),
				Message:  "Field name '" + field.Name + "' should be snake_case",
				Position: pos,
				SuggestedFix: &linter.Fix{
					Description: "Convert to snake_case",
					Changes: []linter.Change{
						{
							FilePath: "",
							StartPos: pos,
							EndPos:   pos,
							OldText:  field.Name,
							NewText:  toSnakeCase(field.Name),
						},
					},
				},
			})
		}
	}

	// Check nested messages
	for _, nested := range msg.Nested {
		violations = append(violations, r.checkMessage(nested)...)
	}

	return violations
}

// AutoFix converts field names to snake_case
func (r *FieldNamingRule) AutoFix(node *protobuf.RootNode, violation linter.Violation) (*linter.Fix, error) {
	return violation.SuggestedFix, nil
}

// isSnakeCase checks if a string is in snake_case
func isSnakeCase(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Must be all lowercase with underscores
	if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(s) {
		return false
	}

	// Should not have consecutive underscores
	if strings.Contains(s, "__") {
		return false
	}

	// Should not start or end with underscore
	if strings.HasPrefix(s, "_") || strings.HasSuffix(s, "_") {
		return false
	}

	return true
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	// Handle PascalCase and camelCase
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
