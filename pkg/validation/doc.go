// Package validation provides semantic validation for protobuf schema files.
//
// # Overview
//
// This package performs deep semantic validation beyond syntax checking, enforcing
// proto conventions, detecting structural issues, and normalizing schemas for storage.
//
// # Validation Checks
//
// Naming Conventions:
//   - Messages: PascalCase (User, OrderRequest)
//   - Fields: snake_case (user_id, created_at)
//   - Enums: UPPER_SNAKE_CASE (STATUS_ACTIVE, ERROR_NOT_FOUND)
//   - Services: PascalCase (UserService, OrderManagement)
//   - RPC methods: PascalCase (GetUser, ListOrders)
//
// Structural Rules:
//   - Field number conflicts
//   - Reserved range violations
//   - Circular dependencies
//   - Unused imports
//   - Missing required fields
//
// # Usage Example
//
// Validate proto file:
//
//	validator := validation.NewValidator(&validation.Config{
//		EnforceNaming:    true,
//		CheckReserved:    true,
//		DetectCircular:   true,
//	})
//
//	errors := validator.Validate(protoContent)
//	for _, err := range errors {
//		fmt.Printf("[%s] %s: %s\n", err.Severity, err.Location, err.Message)
//	}
//
// Normalize schema:
//
//	normalizer := validation.NewNormalizer()
//	normalized := normalizer.Normalize(protoContent)
//	// Sorted imports, sorted fields, consistent whitespace
//
// # Related Packages
//
//   - pkg/linter: Style guide enforcement
//   - pkg/compatibility: Breaking change detection
package validation
