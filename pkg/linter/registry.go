package linter

import (
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// Rule interface that all lint rules must implement
type Rule interface {
	Name() string
	Category() Category
	Severity() Severity
	Description() string
	Check(node *protobuf.RootNode, ctx *LintContext) []Violation
	CanAutoFix() bool
	AutoFix(node *protobuf.RootNode, violation Violation) (*Fix, error)
}

// RuleRegistry manages available lint rules
type RuleRegistry struct {
	rules map[string]Rule
}

// NewRuleRegistry creates a new rule registry
func NewRuleRegistry() *RuleRegistry {
	registry := &RuleRegistry{
		rules: make(map[string]Rule),
	}

	// TODO: Register built-in rules
	// registry.Register(rules.NewMessageNamingRule())
	// registry.Register(rules.NewFieldNamingRule())
	// etc.

	return registry
}

// Register adds a rule to the registry
func (r *RuleRegistry) Register(rule Rule) {
	r.rules[rule.Name()] = rule
}

// GetRule retrieves a rule by name
func (r *RuleRegistry) GetRule(name string) (Rule, bool) {
	rule, ok := r.rules[name]
	return rule, ok
}

// GetAllRules returns all registered rules
func (r *RuleRegistry) GetAllRules() []Rule {
	rules := make([]Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		rules = append(rules, rule)
	}
	return rules
}

// GetEnabledRules returns rules enabled by config
func (r *RuleRegistry) GetEnabledRules(config *Config) []Rule {
	// TODO: Filter rules based on config
	// - Check config.Lint.Use for style guides
	// - Check config.Lint.Rules for enabled/disabled rules
	// - Apply severity overrides

	// For now, return all rules
	return r.GetAllRules()
}

// GetRulesByCategory returns rules in a specific category
func (r *RuleRegistry) GetRulesByCategory(category Category) []Rule {
	rules := make([]Rule, 0)
	for _, rule := range r.rules {
		if rule.Category() == category {
			rules = append(rules, rule)
		}
	}
	return rules
}
