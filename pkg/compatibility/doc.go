// Package compatibility provides protobuf schema compatibility checking for safe API evolution.
//
// # Overview
//
// This package analyzes protobuf schema changes between versions to detect breaking changes.
// It implements compatibility rules based on protobuf's wire format and code generation,
// ensuring that API changes won't break existing clients or services.
//
// # Compatibility Modes
//
// The package supports seven compatibility modes with varying strictness:
//
// NONE: No compatibility checking. Any change is allowed.
// Use for internal schemas or when manually managing compatibility.
//
// BACKWARD: New schema can read data written by old schema.
// Consumers can upgrade independently before producers.
// Safe additions: optional fields, new message types, new enum values.
// Breaking changes: removing fields, changing field types, removing enum values.
//
// FORWARD: Old schema can read data written by new schema.
// Producers can upgrade independently before consumers.
// Safe additions: removing optional fields (consumers ignore unknown fields).
// Breaking changes: adding required fields, removing enum values.
//
// FULL: New schema can read old data AND old schema can read new data.
// Most restrictive. Only additions that both schemas understand are allowed.
// Bidirectional compatibility (BACKWARD + FORWARD).
//
// BACKWARD_TRANSITIVE: Backward compatible with all previous versions, not just immediate predecessor.
// Ensures cumulative compatibility across version history.
//
// FORWARD_TRANSITIVE: Forward compatible with all previous versions.
// Rare in practice; used when old systems must handle data from any future version.
//
// FULL_TRANSITIVE: Full compatibility with all versions in history.
// Most restrictive; ensures maximum interoperability across version chains.
//
// # Usage Example
//
// Basic compatibility check:
//
//	import "github.com/platinummonkey/spoke/pkg/compatibility"
//
//	// Parse old and new schema versions
//	oldSchema, _ := compatibility.ParseSchema(oldProtoContent)
//	newSchema, _ := compatibility.ParseSchema(newProtoContent)
//
//	// Create comparator with BACKWARD mode
//	comparator := compatibility.NewComparator(
//		compatibility.CompatibilityModeBackward,
//		oldSchema,
//		newSchema,
//	)
//
//	// Run comparison
//	result, err := comparator.Compare()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if !result.Compatible {
//		fmt.Println("Breaking changes detected:")
//		for _, v := range result.Violations {
//			if v.Level == compatibility.ViolationLevelError {
//				fmt.Printf("  [%s] %s: %s\n", v.Level, v.Location, v.Message)
//			}
//		}
//		os.Exit(1)
//	}
//
//	fmt.Println("Schema change is compatible!")
//
// # Breaking Change Detection
//
// The comparator detects two types of breaking changes:
//
// Wire Breaking: Changes that affect binary wire format compatibility.
// Causes deserialization errors or data corruption.
// Examples:
//   - Changing field number
//   - Changing field type (int32 → string)
//   - Removing required field
//   - Changing message to different type
//
// Source Breaking: Changes that break generated code compilation.
// Client code using old generated code fails to compile or run.
// Examples:
//   - Removing message or field (old code references non-existent type)
//   - Changing enum value name
//   - Removing RPC method
//   - Changing method signature
//
// Violation example:
//
//	violation := Violation{
//		Rule:           "field_number_changed",
//		Level:          ViolationLevelError,
//		Category:       CategoryFieldChange,
//		Message:        "Field number changed from 1 to 2",
//		Location:       "User.id",
//		OldValue:       "1",
//		NewValue:       "2",
//		WireBreaking:   true,   // Binary format incompatible
//		SourceBreaking: false,  // Generated code still compiles
//		Suggestion:     "Field numbers must remain stable. Use a new field with number 2.",
//	}
//
// # Safe Schema Changes
//
// Backward compatible changes (safe for most use cases):
//
//	// Adding optional field (new field number)
//	message User {
//		string id = 1;
//		string name = 2;
//		string email = 3;  // ✓ Added - consumers ignore unknown fields
//	}
//
//	// Adding new message type
//	message Address {     // ✓ Added - doesn't affect existing messages
//		string street = 1;
//	}
//
//	// Adding enum value
//	enum Status {
//		UNKNOWN = 0;
//		ACTIVE = 1;
//		INACTIVE = 2;
//		SUSPENDED = 3;  // ✓ Added - old consumers treat as UNKNOWN
//	}
//
//	// Marking field as deprecated
//	message User {
//		string id = 1;
//		string name = 2 [deprecated = true];  // ✓ Signals removal intent
//	}
//
//	// Adding RPC method
//	service UserService {
//		rpc GetUser(GetUserRequest) returns (User);
//		rpc UpdateUser(UpdateUserRequest) returns (User);  // ✓ Added
//	}
//
//	// Relaxing field label (required → optional)
//	message User {
//		optional string name = 2;  // Was: required (proto2 only)
//	}
//
// # Breaking Schema Changes
//
// Backward incompatible changes (break old clients):
//
//	// Removing field
//	message User {
//		string id = 1;
//		// string name = 2;  // ✗ Removed - old code expects this field
//	}
//
//	// Changing field number
//	message User {
//		string id = 1;
//		string name = 3;  // ✗ Was 2 - wire format incompatible
//	}
//
//	// Changing field type
//	message User {
//		string id = 1;
//		int32 name = 2;  // ✗ Was string - type mismatch
//	}
//
//	// Removing enum value
//	enum Status {
//		UNKNOWN = 0;
//		ACTIVE = 1;
//		// INACTIVE = 2;  // ✗ Removed - old code references this value
//	}
//
//	// Changing package name
//	package user.v2;  // ✗ Was user.v1 - breaks imports
//
//	// Removing RPC method
//	service UserService {
//		rpc GetUser(GetUserRequest) returns (User);
//		// rpc UpdateUser(UpdateUserRequest) returns (User);  // ✗ Removed
//	}
//
//	// Tightening field label (optional → required)
//	message User {
//		required string name = 2;  // ✗ Was optional - old messages missing field
//	}
//
// # Schema Graph
//
// The package parses proto files into a SchemaGraph for analysis:
//
//	type SchemaGraph struct {
//		Package  string              // Package name
//		Syntax   string              // "proto2" or "proto3"
//		Imports  []Import            // Import statements
//		Messages map[string]*Message // All messages by fully qualified name
//		Enums    map[string]*Enum    // All enums
//		Services map[string]*Service // All gRPC services
//		Dependencies map[string]*SchemaGraph // Imported schemas
//	}
//
// Messages contain field metadata:
//
//	type Message struct {
//		Name         string
//		FullName     string              // package.Message
//		Fields       map[int]*Field      // Field number → Field
//		FieldsByName map[string]*Field   // Field name → Field
//		Reserved     *Reserved           // Reserved numbers/names
//		Nested       map[string]*Message // Nested messages
//		NestedEnums  map[string]*Enum    // Nested enums
//		OneOfs       map[string]*OneOf   // OneOf groups
//	}
//
// Fields track type and metadata:
//
//	type Field struct {
//		Name         string
//		Number       int
//		Type         FieldType  // int32, string, message, etc.
//		Label        FieldLabel // optional, required, repeated
//		TypeName     string     // For message/enum types
//		InOneOf      string     // OneOf group name if applicable
//		Deprecated   bool
//	}
//
// # Comparison Rules
//
// The comparator applies comprehensive rules:
//
// Message Comparison:
//   - Field numbers must remain stable
//   - Field types must remain compatible (some widening allowed)
//   - Field labels can relax (required → optional) but not tighten
//   - Removed fields must have numbers reserved
//   - New fields must be optional (proto2) or use default values (proto3)
//
// Enum Comparison:
//   - Enum values must remain stable by number
//   - Removing enum values breaks old code
//   - Adding enum values is safe (unrecognized values → default)
//   - Enum value 0 must always exist (proto3 default value)
//
// Service Comparison:
//   - RPC method signatures must remain stable
//   - Removing RPC methods breaks clients
//   - Adding RPC methods is safe
//   - Changing streaming modes is breaking
//
// Type Compatibility Matrix:
//
//	Old Type → New Type  Compatible?
//	--------   --------  -----------
//	int32   →  int64     Yes (wire compatible, value preserved)
//	int32   →  uint32    Yes (same wire format, semantic change)
//	int32   →  string    No  (wire format incompatible)
//	string  →  bytes     Yes (same wire format)
//	bool    →  int32     Yes (wire compatible: 0/1)
//	enum    →  int32     Yes (enum values are integers)
//
// # Violation Reporting
//
// Violations are categorized by severity:
//
//	ViolationLevelError: Breaking change, blocks schema registration
//	ViolationLevelWarning: Suspicious change, may break semantics
//	ViolationLevelInfo: Informational, documents change
//
// Example violation:
//
//	Violation{
//		Rule:     "field_removed",
//		Level:    ViolationLevelError,
//		Category: CategoryFieldChange,
//		Message:  "Field 'name' was removed",
//		Location: "User.name",
//		OldValue: "string name = 2",
//		NewValue: "(removed)",
//		WireBreaking:   false,  // Old binaries can deserialize
//		SourceBreaking: true,   // Generated code references field
//		Suggestion: "Mark field as deprecated and reserve field number 2",
//	}
//
// Check result summary:
//
//	result := CheckResult{
//		Compatible: false,
//		Mode:       "BACKWARD",
//		Violations: [...],
//		Summary: Summary{
//			TotalViolations: 3,
//			Errors:          1,
//			Warnings:        2,
//			Infos:           0,
//			WireBreaking:    0,
//			SourceBreaking:  1,
//		},
//	}
//
// # Integration with API
//
// The pkg/api package integrates compatibility checking:
//
//	POST /api/v1/modules/{name}/diff
//	{
//		"old_version": "v1.0.0",
//		"new_version": "v1.1.0",
//		"mode": "BACKWARD"
//	}
//
//	Response:
//	{
//		"compatible": false,
//		"mode": "BACKWARD",
//		"violations": [
//			{
//				"rule": "field_removed",
//				"level": "ERROR",
//				"message": "Field 'User.name' was removed",
//				"location": "User.name",
//				"wire_breaking": false,
//				"source_breaking": true
//			}
//		],
//		"summary": {
//			"total_violations": 1,
//			"errors": 1,
//			"warnings": 0
//		}
//	}
//
// # CI/CD Integration
//
// Block incompatible schema changes in CI:
//
//	#!/bin/bash
//	# .github/workflows/schema-check.yml
//
//	spoke pull -module user-service -version main
//	spoke validate -dir ./proto
//
//	# Compare with latest version
//	spoke diff \
//		-module user-service \
//		-old-version v1.0.0 \
//		-new-version main \
//		-mode BACKWARD
//
//	if [ $? -ne 0 ]; then
//		echo "Breaking schema changes detected!"
//		exit 1
//	fi
//
// # Reserved Fields
//
// When removing fields, reserve their numbers to prevent reuse:
//
//	message User {
//		string id = 1;
//		// string name = 2;  // Removed
//		reserved 2;            // Prevent accidental reuse
//		reserved "name";       // Reserve by name too
//
//		string email = 3;
//	}
//
// Reserved ranges:
//
//	message User {
//		string id = 1;
//		reserved 2 to 10;      // Reserve range
//		reserved 15, 16, 20 to 30;
//		string email = 11;
//	}
//
// The comparator validates reserved fields aren't violated.
//
// # Proto2 vs Proto3
//
// Proto2 compatibility considerations:
//
//	// Proto2: required fields are breaking changes
//	message User {
//		required string id = 1;      // Can't remove
//		optional string name = 2;    // Can remove
//		repeated string tags = 3;    // Can remove
//	}
//
//	// Changing required → optional is SAFE (relaxing constraint)
//	// Changing optional → required is UNSAFE (tightening constraint)
//
// Proto3 compatibility:
//
//	// Proto3: all fields are optional by default
//	message User {
//		string id = 1;      // Can't remove (source breaking)
//		string name = 2;    // Can't remove
//	}
//
//	// No required fields exist in proto3
//	// Removing fields always breaks source compatibility
//
// # Performance Considerations
//
// Schema parsing and comparison can be expensive for large proto files:
//
// 1. Cache parsed schemas:
//
//	var schemaCache = make(map[string]*SchemaGraph)
//	schema, ok := schemaCache[version]
//	if !ok {
//		schema, _ = ParseSchema(protoContent)
//		schemaCache[version] = schema
//	}
//
// 2. Compare only changed files in multi-file schemas:
//
//	// Only reparse files that changed
//	changedFiles := detectChangedFiles(oldVersion, newVersion)
//	for _, file := range changedFiles {
//		oldSchema := parseFile(oldVersion, file)
//		newSchema := parseFile(newVersion, file)
//		compare(oldSchema, newSchema)
//	}
//
// 3. Use transitive modes sparingly (check all version pairs is O(n²)):
//
//	// BACKWARD: compare only v1.1.0 vs v1.0.0 (O(1))
//	// BACKWARD_TRANSITIVE: compare v1.1.0 vs v1.0.0, v0.9.0, v0.8.0... (O(n))
//
// # Testing
//
// Test compatibility rules:
//
//	func TestFieldRemovalBreaksBackwardCompatibility(t *testing.T) {
//		oldProto := `
//			syntax = "proto3";
//			message User {
//				string id = 1;
//				string name = 2;
//			}
//		`
//		newProto := `
//			syntax = "proto3";
//			message User {
//				string id = 1;
//			}
//		`
//
//		oldSchema, _ := ParseSchema(oldProto)
//		newSchema, _ := ParseSchema(newProto)
//
//		comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
//		result, _ := comparator.Compare()
//
//		assert.False(t, result.Compatible)
//		assert.Equal(t, 1, result.Summary.Errors)
//
//		violation := result.Violations[0]
//		assert.Equal(t, "field_removed", violation.Rule)
//		assert.Equal(t, "User.name", violation.Location)
//	}
//
// # Related Packages
//
//   - pkg/api: HTTP API that exposes compatibility checking
//   - pkg/validation: Lints proto files for style and correctness
//   - pkg/storage: Stores proto file versions for comparison
//   - pkg/api/protobuf: Proto file parsing and AST
//
// # Design Decisions
//
// Wire vs Source Breaking: Distinguishing these helps users understand impact.
// Wire breaking affects running systems, source breaking affects development.
//
// Violation Categories: Grouping violations (field_change, enum_change, etc.)
// enables filtering and custom handling per category.
//
// Transitive Modes: Supporting full version history comparison catches cumulative
// incompatibilities missed by pairwise checks.
//
// Schema Graph Abstraction: Building an intermediate graph decouples parsing from
// comparison logic, making it easier to support multiple proto parsers.
//
// Actionable Suggestions: Violations include suggestions for fixes, guiding users
// toward compatible changes (e.g., "reserve field number instead of removing").
package compatibility
