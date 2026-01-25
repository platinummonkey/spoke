package rules

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/linter"
)

// EnumNamingRule checks that enum names follow PascalCase
type EnumNamingRule struct {
	BaseRule
}

// NewEnumNamingRule creates a new enum naming rule
func NewEnumNamingRule() *EnumNamingRule {
	return &EnumNamingRule{
		BaseRule: BaseRule{
			RuleName:        "enum-naming",
			RuleCategory:    linter.CategoryNaming,
			RuleSeverity:    linter.SeverityError,
			RuleDescription: "Enum names must use PascalCase",
			AutoFixable:     true,
		},
	}
}

// Check validates enum names
func (r *EnumNamingRule) Check(node *protobuf.RootNode, ctx *linter.LintContext) []linter.Violation {
	violations := make([]linter.Violation, 0)

	// Check top-level enums
	for _, enum := range node.Enums {
		violations = append(violations, r.checkEnum(enum)...)
	}

	// Check enums in messages
	for _, msg := range node.Messages {
		violations = append(violations, r.checkMessage(msg)...)
	}

	return violations
}

func (r *EnumNamingRule) checkEnum(enum *protobuf.EnumNode) []linter.Violation {
	violations := make([]linter.Violation, 0)

	if !isPascalCase(enum.Name) {
		pos := enum.Position()
		violations = append(violations, linter.Violation{
			Rule:     r.Name(),
			Severity: r.Severity(),
			Category: r.Category(),
			Message:  "Enum name '" + enum.Name + "' should be PascalCase",
			Position: pos,
			SuggestedFix: &linter.Fix{
				Description: "Convert to PascalCase",
				Changes: []linter.Change{
					{
						FilePath: "",
						StartPos: pos,
						EndPos:   pos,
						OldText:  enum.Name,
						NewText:  toPascalCase(enum.Name),
					},
				},
			},
		})
	}

	return violations
}

func (r *EnumNamingRule) checkMessage(msg *protobuf.MessageNode) []linter.Violation {
	violations := make([]linter.Violation, 0)

	// Check enums in message
	for _, enum := range msg.Enums {
		violations = append(violations, r.checkEnum(enum)...)
	}

	// Check nested messages
	for _, nested := range msg.Nested {
		violations = append(violations, r.checkMessage(nested)...)
	}

	return violations
}

// AutoFix converts enum names to PascalCase
func (r *EnumNamingRule) AutoFix(node *protobuf.RootNode, violation linter.Violation) (*linter.Fix, error) {
	return violation.SuggestedFix, nil
}

// EnumValueNamingRule checks that enum values follow UPPER_SNAKE_CASE
type EnumValueNamingRule struct {
	BaseRule
}

// NewEnumValueNamingRule creates a new enum value naming rule
func NewEnumValueNamingRule() *EnumValueNamingRule {
	return &EnumValueNamingRule{
		BaseRule: BaseRule{
			RuleName:        "enum-value-naming",
			RuleCategory:    linter.CategoryNaming,
			RuleSeverity:    linter.SeverityError,
			RuleDescription: "Enum values must use UPPER_SNAKE_CASE",
			AutoFixable:     true,
		},
	}
}

// Check validates enum value names
func (r *EnumValueNamingRule) Check(node *protobuf.RootNode, ctx *linter.LintContext) []linter.Violation {
	violations := make([]linter.Violation, 0)

	// Check top-level enums
	for _, enum := range node.Enums {
		violations = append(violations, r.checkEnumValues(enum)...)
	}

	// Check enums in messages
	for _, msg := range node.Messages {
		violations = append(violations, r.checkMessageEnumValues(msg)...)
	}

	return violations
}

func (r *EnumValueNamingRule) checkEnumValues(enum *protobuf.EnumNode) []linter.Violation {
	violations := make([]linter.Violation, 0)

	for _, value := range enum.Values {
		if !isUpperSnakeCase(value.Name) {
			pos := value.Position()
			violations = append(violations, linter.Violation{
				Rule:     r.Name(),
				Severity: r.Severity(),
				Category: r.Category(),
				Message:  "Enum value '" + value.Name + "' should be UPPER_SNAKE_CASE",
				Position: pos,
				SuggestedFix: &linter.Fix{
					Description: "Convert to UPPER_SNAKE_CASE",
					Changes: []linter.Change{
						{
							FilePath: "",
							StartPos: pos,
							EndPos:   pos,
							OldText:  value.Name,
							NewText:  toUpperSnakeCase(value.Name),
						},
					},
				},
			})
		}
	}

	return violations
}

func (r *EnumValueNamingRule) checkMessageEnumValues(msg *protobuf.MessageNode) []linter.Violation {
	violations := make([]linter.Violation, 0)

	// Check enums in message
	for _, enum := range msg.Enums {
		violations = append(violations, r.checkEnumValues(enum)...)
	}

	// Check nested messages
	for _, nested := range msg.Nested {
		violations = append(violations, r.checkMessageEnumValues(nested)...)
	}

	return violations
}

// AutoFix converts enum values to UPPER_SNAKE_CASE
func (r *EnumValueNamingRule) AutoFix(node *protobuf.RootNode, violation linter.Violation) (*linter.Fix, error) {
	return violation.SuggestedFix, nil
}

// isUpperSnakeCase checks if a string is in UPPER_SNAKE_CASE
func isUpperSnakeCase(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Must be all uppercase with underscores
	if !regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`).MatchString(s) {
		return false
	}

	// Should not have consecutive underscores
	if strings.Contains(s, "__") {
		return false
	}

	// Should not end with underscore
	if strings.HasSuffix(s, "_") {
		return false
	}

	return true
}

// toUpperSnakeCase converts a string to UPPER_SNAKE_CASE
func toUpperSnakeCase(s string) string {
	// If already in correct format, return as-is
	if isUpperSnakeCase(s) {
		return s
	}

	// First convert to snake_case if needed
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && !unicode.IsUpper(rune(s[i-1])) && s[i-1] != '_' {
				result.WriteRune('_')
			}
			result.WriteRune(r)
		} else if unicode.IsLower(r) {
			result.WriteRune(unicode.ToUpper(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
