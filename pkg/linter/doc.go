// Package linter provides protobuf linting and style guide enforcement.
//
// # Overview
//
// This package applies configurable style guide rules (Google, Uber, custom) to proto files,
// performs quality metrics analysis, and generates automatic fixes for violations.
//
// # Style Guides
//
// Google: Standard protobuf style guide
// Uber: Uber's proto conventions
// Custom: Organization-specific rules
//
// # Rule Categories
//
// Naming: Message, field, enum, service naming conventions
// Style: Indentation, line length, import ordering
// Documentation: Comment coverage requirements
// Structure: Message complexity, field count limits
//
// # Usage Example
//
// Lint with auto-fix:
//
//	config := &linter.Config{
//		StyleGuide: linter.StyleGuideGoogle,
//		AutoFix:    true,
//		Rules: map[string]bool{
//			"message_naming":    true,
//			"field_naming":      true,
//			"require_comments":  true,
//		},
//	}
//
//	engine := linter.NewEngine(config)
//	result := engine.Lint(protoContent)
//
//	fmt.Printf("Violations: %d errors, %d warnings\n",
//		result.Errors, result.Warnings)
//
//	if result.AutoFixed {
//		fmt.Println("Applied fixes:")
//		fmt.Println(result.FixedContent)
//	}
//
// Quality metrics:
//
//	fmt.Printf("Documentation coverage: %.1f%%\n",
//		result.Metrics.DocCoverage * 100)
//	fmt.Printf("Average message complexity: %.1f\n",
//		result.Metrics.AvgComplexity)
//
// # Related Packages
//
//   - pkg/validation: Semantic validation
//   - pkg/linter/rules: Individual lint rules
//   - pkg/linter/formatters: Output formatters (JSON, text, sarif)
package linter
