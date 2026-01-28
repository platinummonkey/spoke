package linter

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/stretchr/testify/assert"
)

// Mock rule for testing
type mockRule struct {
	name        string
	category    Category
	severity    Severity
	description string
	canAutoFix  bool
}

func (m *mockRule) Name() string {
	return m.name
}

func (m *mockRule) Category() Category {
	return m.category
}

func (m *mockRule) Severity() Severity {
	return m.severity
}

func (m *mockRule) Description() string {
	return m.description
}

func (m *mockRule) Check(node *protobuf.RootNode, ctx *LintContext) []Violation {
	return nil
}

func (m *mockRule) CanAutoFix() bool {
	return m.canAutoFix
}

func (m *mockRule) AutoFix(node *protobuf.RootNode, violation Violation) (*Fix, error) {
	return nil, nil
}

func TestNewRuleRegistry(t *testing.T) {
	registry := NewRuleRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.rules)
	assert.Equal(t, 0, len(registry.rules))
}

func TestRuleRegistry_Register(t *testing.T) {
	registry := NewRuleRegistry()

	rule1 := &mockRule{
		name:        "test-rule-1",
		category:    CategoryStyle,
		severity:    SeverityError,
		description: "Test rule 1",
		canAutoFix:  true,
	}

	rule2 := &mockRule{
		name:        "test-rule-2",
		category:    CategoryNaming,
		severity:    SeverityWarning,
		description: "Test rule 2",
		canAutoFix:  false,
	}

	registry.Register(rule1)
	registry.Register(rule2)

	assert.Equal(t, 2, len(registry.rules))

	// Verify rules are registered correctly
	retrievedRule1, ok := registry.GetRule("test-rule-1")
	assert.True(t, ok)
	assert.Equal(t, rule1, retrievedRule1)

	retrievedRule2, ok := registry.GetRule("test-rule-2")
	assert.True(t, ok)
	assert.Equal(t, rule2, retrievedRule2)
}

func TestRuleRegistry_GetRule(t *testing.T) {
	registry := NewRuleRegistry()

	rule := &mockRule{
		name:        "test-rule",
		category:    CategoryStyle,
		severity:    SeverityError,
		description: "Test rule",
	}

	registry.Register(rule)

	tests := []struct {
		name      string
		ruleName  string
		wantFound bool
	}{
		{
			name:      "existing rule",
			ruleName:  "test-rule",
			wantFound: true,
		},
		{
			name:      "non-existent rule",
			ruleName:  "non-existent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrievedRule, found := registry.GetRule(tt.ruleName)
			assert.Equal(t, tt.wantFound, found)

			if tt.wantFound {
				assert.NotNil(t, retrievedRule)
				assert.Equal(t, tt.ruleName, retrievedRule.Name())
			} else {
				assert.Nil(t, retrievedRule)
			}
		})
	}
}

func TestRuleRegistry_GetAllRules(t *testing.T) {
	registry := NewRuleRegistry()

	// Test with empty registry
	rules := registry.GetAllRules()
	assert.NotNil(t, rules)
	assert.Equal(t, 0, len(rules))

	// Add multiple rules
	rule1 := &mockRule{name: "rule-1", category: CategoryStyle, severity: SeverityError}
	rule2 := &mockRule{name: "rule-2", category: CategoryNaming, severity: SeverityWarning}
	rule3 := &mockRule{name: "rule-3", category: CategoryDocumentation, severity: SeverityInfo}

	registry.Register(rule1)
	registry.Register(rule2)
	registry.Register(rule3)

	rules = registry.GetAllRules()
	assert.Equal(t, 3, len(rules))

	// Verify all rules are present (order doesn't matter for map iteration)
	ruleNames := make(map[string]bool)
	for _, rule := range rules {
		ruleNames[rule.Name()] = true
	}

	assert.True(t, ruleNames["rule-1"])
	assert.True(t, ruleNames["rule-2"])
	assert.True(t, ruleNames["rule-3"])
}

func TestRuleRegistry_GetEnabledRules(t *testing.T) {
	registry := NewRuleRegistry()

	rule1 := &mockRule{name: "rule-1", category: CategoryStyle, severity: SeverityError}
	rule2 := &mockRule{name: "rule-2", category: CategoryNaming, severity: SeverityWarning}
	rule3 := &mockRule{name: "rule-3", category: CategoryDocumentation, severity: SeverityInfo}

	registry.Register(rule1)
	registry.Register(rule2)
	registry.Register(rule3)

	config := DefaultConfig()

	// Currently GetEnabledRules returns all rules (TODO implementation)
	enabledRules := registry.GetEnabledRules(config)
	assert.Equal(t, 3, len(enabledRules))

	// Verify all rules are returned
	ruleNames := make(map[string]bool)
	for _, rule := range enabledRules {
		ruleNames[rule.Name()] = true
	}

	assert.True(t, ruleNames["rule-1"])
	assert.True(t, ruleNames["rule-2"])
	assert.True(t, ruleNames["rule-3"])
}

func TestRuleRegistry_GetRulesByCategory(t *testing.T) {
	registry := NewRuleRegistry()

	styleRule1 := &mockRule{name: "style-1", category: CategoryStyle, severity: SeverityError}
	styleRule2 := &mockRule{name: "style-2", category: CategoryStyle, severity: SeverityWarning}
	securityRule := &mockRule{name: "security-1", category: CategoryNaming, severity: SeverityError}
	perfRule := &mockRule{name: "perf-1", category: CategoryDocumentation, severity: SeverityInfo}

	registry.Register(styleRule1)
	registry.Register(styleRule2)
	registry.Register(securityRule)
	registry.Register(perfRule)

	tests := []struct {
		name          string
		category      Category
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "style rules",
			category:      CategoryStyle,
			expectedCount: 2,
			expectedNames: []string{"style-1", "style-2"},
		},
		{
			name:          "security rules",
			category:      CategoryNaming,
			expectedCount: 1,
			expectedNames: []string{"security-1"},
		},
		{
			name:          "performance rules",
			category:      CategoryDocumentation,
			expectedCount: 1,
			expectedNames: []string{"perf-1"},
		},
		{
			name:          "no rules in category",
			category:      CategoryStructure,
			expectedCount: 0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := registry.GetRulesByCategory(tt.category)
			assert.Equal(t, tt.expectedCount, len(rules))

			if tt.expectedCount > 0 {
				ruleNames := make(map[string]bool)
				for _, rule := range rules {
					ruleNames[rule.Name()] = true
					assert.Equal(t, tt.category, rule.Category())
				}

				for _, expectedName := range tt.expectedNames {
					assert.True(t, ruleNames[expectedName], "Expected rule %s not found", expectedName)
				}
			}
		})
	}
}

func TestRuleRegistry_RegisterMultipleSameName(t *testing.T) {
	registry := NewRuleRegistry()

	rule1 := &mockRule{
		name:        "duplicate-rule",
		category:    CategoryStyle,
		severity:    SeverityError,
		description: "First version",
	}

	rule2 := &mockRule{
		name:        "duplicate-rule",
		category:    CategoryNaming,
		severity:    SeverityWarning,
		description: "Second version",
	}

	registry.Register(rule1)
	registry.Register(rule2)

	// Should overwrite first registration
	retrievedRule, ok := registry.GetRule("duplicate-rule")
	assert.True(t, ok)
	assert.Equal(t, "Second version", retrievedRule.Description())
	assert.Equal(t, CategoryNaming, retrievedRule.Category())
}

func TestRuleRegistry_GetAllRulesEmptyCategories(t *testing.T) {
	registry := NewRuleRegistry()

	// Test getting rules by all possible categories when registry is empty
	categories := []Category{
		CategoryStyle,
		CategoryNaming,
		CategoryDocumentation,
		CategoryStructure,
	}

	for _, category := range categories {
		rules := registry.GetRulesByCategory(category)
		assert.NotNil(t, rules)
		assert.Equal(t, 0, len(rules))
	}
}

func TestRuleRegistry_MixedCategories(t *testing.T) {
	registry := NewRuleRegistry()

	// Create rules in all categories
	categories := []Category{
		CategoryStyle,
		CategoryNaming,
		CategoryDocumentation,
		CategoryStructure,
	}

	for _, category := range categories {
		rule := &mockRule{
			name:     string(category) + "-rule",
			category: category,
			severity: SeverityError,
		}
		registry.Register(rule)
	}

	// Verify each category has exactly one rule
	for _, category := range categories {
		rules := registry.GetRulesByCategory(category)
		assert.Equal(t, 1, len(rules), "Category %s should have 1 rule", category)
		assert.Equal(t, category, rules[0].Category())
	}

	// Verify total count
	allRules := registry.GetAllRules()
	assert.Equal(t, len(categories), len(allRules))
}
