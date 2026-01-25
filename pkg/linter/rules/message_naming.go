package rules

import (
	"regexp"
	"unicode"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/linter"
)

// MessageNamingRule checks that message names follow PascalCase
type MessageNamingRule struct {
	BaseRule
}

// NewMessageNamingRule creates a new message naming rule
func NewMessageNamingRule() *MessageNamingRule {
	return &MessageNamingRule{
		BaseRule: BaseRule{
			RuleName:        "message-naming",
			RuleCategory:    linter.CategoryNaming,
			RuleSeverity:    linter.SeverityError,
			RuleDescription: "Message names must use PascalCase",
			AutoFixable:     true,
		},
	}
}

// Check validates message names
func (r *MessageNamingRule) Check(node *protobuf.RootNode, ctx *linter.LintContext) []linter.Violation {
	violations := make([]linter.Violation, 0)

	// Check top-level messages
	for _, msg := range node.Messages {
		violations = append(violations, r.checkMessage(msg)...)
	}

	return violations
}

func (r *MessageNamingRule) checkMessage(msg *protobuf.MessageNode) []linter.Violation {
	violations := make([]linter.Violation, 0)

	// Check if name is PascalCase
	if !isPascalCase(msg.Name) {
		pos := msg.Position()
		violations = append(violations, linter.Violation{
			Rule:     r.Name(),
			Severity: r.Severity(),
			Category: r.Category(),
			Message:  "Message name '" + msg.Name + "' should be PascalCase",
			Position: pos,
			SuggestedFix: &linter.Fix{
				Description: "Convert to PascalCase",
				Changes: []linter.Change{
					{
						FilePath: "",
						StartPos: pos,
						EndPos:   pos,
						OldText:  msg.Name,
						NewText:  toPascalCase(msg.Name),
					},
				},
			},
		})
	}

	// Check nested messages
	for _, nested := range msg.Nested {
		violations = append(violations, r.checkMessage(nested)...)
	}

	return violations
}

// AutoFix converts message names to PascalCase
func (r *MessageNamingRule) AutoFix(node *protobuf.RootNode, violation linter.Violation) (*linter.Fix, error) {
	return violation.SuggestedFix, nil
}

// isPascalCase checks if a string is in PascalCase
func isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Must start with uppercase letter
	if !unicode.IsUpper(rune(s[0])) {
		return false
	}

	// Must not contain underscores
	if regexp.MustCompile(`_`).MatchString(s) {
		return false
	}

	// Must not contain non-alphanumeric characters
	if !regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(s) {
		return false
	}

	return true
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	// Handle snake_case
	if regexp.MustCompile(`_`).MatchString(s) {
		parts := regexp.MustCompile(`_`).Split(s, -1)
		result := ""
		for _, part := range parts {
			if len(part) > 0 {
				result += string(unicode.ToUpper(rune(part[0]))) + part[1:]
			}
		}
		return result
	}

	// Handle lowercase
	if len(s) > 0 && unicode.IsLower(rune(s[0])) {
		return string(unicode.ToUpper(rune(s[0]))) + s[1:]
	}

	return s
}
