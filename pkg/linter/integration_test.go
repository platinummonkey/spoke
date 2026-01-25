package linter_test

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/linter"
	"github.com/platinummonkey/spoke/pkg/linter/rules"
)

func TestLintEngine_NamingRules(t *testing.T) {
	config := linter.DefaultConfig()
	engine := linter.NewLintEngine(config)

	// Register naming rules
	for _, rule := range rules.DefaultRules() {
		engine.Registry().Register(rule)
	}

	// Create test proto with various violations
	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Messages: []*protobuf.MessageNode{
			{
				Name: "user_profile", // Violation: should be PascalCase
				Fields: []*protobuf.FieldNode{
					{Name: "UserId", Type: "string", Number: 1},    // Violation: should be snake_case
					{Name: "user_name", Type: "string", Number: 2}, // Valid
				},
			},
			{
				Name: "Address", // Valid
				Fields: []*protobuf.FieldNode{
					{Name: "street", Type: "string", Number: 1}, // Valid
					{Name: "City", Type: "string", Number: 2},   // Violation: should be snake_case
				},
			},
		},
		Services: []*protobuf.ServiceNode{
			{Name: "user_service"}, // Violation: should be PascalCase
		},
		Enums: []*protobuf.EnumNode{
			{
				Name: "status", // Violation: should be PascalCase
				Values: []*protobuf.EnumValueNode{
					{Name: "Active", Number: 0},   // Violation: should be UPPER_SNAKE_CASE
					{Name: "INACTIVE", Number: 1}, // Valid
				},
			},
		},
	}

	result := engine.Lint("test.proto", ast)

	// Expected violations:
	// 1. Message "user_profile" -> "UserProfile"
	// 2. Field "UserId" -> "user_id"
	// 3. Field "City" -> "city"
	// 4. Service "user_service" -> "UserService"
	// 5. Enum "status" -> "Status"
	// 6. Enum value "Active" -> "ACTIVE"
	// Total: 6 violations

	if len(result.Violations) != 6 {
		t.Errorf("Expected 6 violations, got %d", len(result.Violations))
		for i, v := range result.Violations {
			t.Logf("Violation %d: [%s] %s", i+1, v.Rule, v.Message)
		}
	}

	// Verify each violation has a suggested fix
	for _, v := range result.Violations {
		if v.SuggestedFix == nil {
			t.Errorf("Violation [%s] '%s' has no suggested fix", v.Rule, v.Message)
		}
	}

	// Verify violation details
	violationsByRule := make(map[string]int)
	for _, v := range result.Violations {
		violationsByRule[v.Rule]++
	}

	expectedCounts := map[string]int{
		"message-naming":    1, // user_profile
		"field-naming":      2, // UserId, City
		"service-naming":    1, // user_service
		"enum-naming":       1, // status
		"enum-value-naming": 1, // Active
	}

	for rule, expectedCount := range expectedCounts {
		if count := violationsByRule[rule]; count != expectedCount {
			t.Errorf("Expected %d violations for rule '%s', got %d", expectedCount, rule, count)
		}
	}
}

func TestLintEngine_NoViolations(t *testing.T) {
	config := linter.DefaultConfig()
	engine := linter.NewLintEngine(config)
	for _, rule := range rules.DefaultRules() {
		engine.Registry().Register(rule)
	}

	// Create perfectly valid proto
	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.package"},
		Messages: []*protobuf.MessageNode{
			{
				Name: "UserProfile",
				Fields: []*protobuf.FieldNode{
					{Name: "user_id", Type: "string", Number: 1},
					{Name: "user_name", Type: "string", Number: 2},
					{Name: "email_address", Type: "string", Number: 3},
				},
			},
			{
				Name: "Address",
				Fields: []*protobuf.FieldNode{
					{Name: "street", Type: "string", Number: 1},
					{Name: "city", Type: "string", Number: 2},
					{Name: "zip_code", Type: "string", Number: 3},
				},
			},
		},
		Services: []*protobuf.ServiceNode{
			{Name: "UserService"},
			{Name: "AddressService"},
		},
		Enums: []*protobuf.EnumNode{
			{
				Name: "Status",
				Values: []*protobuf.EnumValueNode{
					{Name: "STATUS_UNKNOWN", Number: 0},
					{Name: "STATUS_ACTIVE", Number: 1},
					{Name: "STATUS_INACTIVE", Number: 2},
				},
			},
		},
	}

	result := engine.Lint("test.proto", ast)

	if len(result.Violations) != 0 {
		t.Errorf("Expected no violations, got %d", len(result.Violations))
		for i, v := range result.Violations {
			t.Logf("Unexpected violation %d: [%s] %s", i+1, v.Rule, v.Message)
		}
	}
}

func TestLintEngine_MultipleFiles(t *testing.T) {
	config := linter.DefaultConfig()
	engine := linter.NewLintEngine(config)
	for _, rule := range rules.DefaultRules() {
		engine.Registry().Register(rule)
	}

	files := map[string]*protobuf.RootNode{
		"user.proto": {
			Messages: []*protobuf.MessageNode{
				{Name: "user_profile"}, // 1 violation
			},
		},
		"address.proto": {
			Messages: []*protobuf.MessageNode{
				{Name: "Address"}, // Valid
			},
		},
		"status.proto": {
			Enums: []*protobuf.EnumNode{
				{
					Name: "status", // 1 violation
					Values: []*protobuf.EnumValueNode{
						{Name: "Active", Number: 0}, // 1 violation
					},
				},
			},
		},
	}

	results := engine.LintFiles(files)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	totalViolations := 0
	for _, result := range results {
		totalViolations += len(result.Violations)
	}

	if totalViolations != 3 {
		t.Errorf("Expected 3 total violations, got %d", totalViolations)
	}
}

func TestLintEngine_Summary(t *testing.T) {
	config := linter.DefaultConfig()
	engine := linter.NewLintEngine(config)
	for _, rule := range rules.DefaultRules() {
		engine.Registry().Register(rule)
	}

	files := map[string]*protobuf.RootNode{
		"file1.proto": {
			Messages: []*protobuf.MessageNode{
				{Name: "invalid_name"}, // Error severity
			},
		},
		"file2.proto": {
			Messages: []*protobuf.MessageNode{
				{Name: "ValidName"}, // No violations
			},
		},
		"file3.proto": {
			Services: []*protobuf.ServiceNode{
				{Name: "invalid_service"}, // Error severity
			},
		},
	}

	results := engine.LintFiles(files)
	summary := engine.GenerateSummary(results)

	if summary.TotalFiles != 3 {
		t.Errorf("Expected 3 files, got %d", summary.TotalFiles)
	}

	if summary.TotalViolations != 2 {
		t.Errorf("Expected 2 violations, got %d", summary.TotalViolations)
	}

	if summary.Errors != 2 {
		t.Errorf("Expected 2 errors (all naming violations are errors), got %d", summary.Errors)
	}

	if summary.Warnings != 0 {
		t.Errorf("Expected 0 warnings, got %d", summary.Warnings)
	}
}

func TestLintEngine_NestedMessages(t *testing.T) {
	config := linter.DefaultConfig()
	engine := linter.NewLintEngine(config)
	for _, rule := range rules.DefaultRules() {
		engine.Registry().Register(rule)
	}

	// Test nested messages and enums
	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "OuterMessage",
				Nested: []*protobuf.MessageNode{
					{
						Name: "inner_message", // Violation
						Fields: []*protobuf.FieldNode{
							{Name: "NestedField", Type: "string", Number: 1}, // Violation
						},
					},
				},
				Enums: []*protobuf.EnumNode{
					{
						Name: "nested_enum", // Violation
						Values: []*protobuf.EnumValueNode{
							{Name: "value_one", Number: 0}, // Violation
						},
					},
				},
			},
		},
	}

	result := engine.Lint("test.proto", ast)

	// Expected: 4 violations (nested message name, nested field, nested enum, nested enum value)
	if len(result.Violations) != 4 {
		t.Errorf("Expected 4 violations for nested structures, got %d", len(result.Violations))
		for i, v := range result.Violations {
			t.Logf("Violation %d: [%s] %s", i+1, v.Rule, v.Message)
		}
	}
}

func TestLintEngine_QualityMetrics(t *testing.T) {
	config := linter.DefaultConfig()
	config.Quality.Enabled = true
	engine := linter.NewLintEngine(config)

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{Name: "Message1"},
			{Name: "Message2"},
			{Name: "Message3"},
		},
	}

	result := engine.Lint("test.proto", ast)

	// Check that metrics are calculated
	if result.Metrics.MessageCount != 3 {
		t.Errorf("Expected message count 3, got %d", result.Metrics.MessageCount)
	}
}
