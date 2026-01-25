package rules

import "github.com/platinummonkey/spoke/pkg/linter"

// DefaultRules returns all built-in lint rules
// Caller should register these with their registry
func DefaultRules() []linter.Rule {
	return []linter.Rule{
		// Naming rules
		NewMessageNamingRule(),
		NewFieldNamingRule(),
		NewServiceNamingRule(),
		NewEnumNamingRule(),
		NewEnumValueNamingRule(),

		// TODO: Add more built-in rules:
		// - Package naming
		// - Comment requirements
		// - Documentation coverage
		// - Deprecation tracking
		// - Structure rules
	}
}
