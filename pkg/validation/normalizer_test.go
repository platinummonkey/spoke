package validation

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// TestDefaultNormalizationConfig tests the default configuration
func TestDefaultNormalizationConfig(t *testing.T) {
	config := DefaultNormalizationConfig()

	if config == nil {
		t.Fatal("DefaultNormalizationConfig returned nil")
	}

	if !config.SortFields {
		t.Error("SortFields should be enabled by default")
	}
	if !config.SortEnumValues {
		t.Error("SortEnumValues should be enabled by default")
	}
	if !config.SortImports {
		t.Error("SortImports should be enabled by default")
	}
	if !config.CanonicalizeImports {
		t.Error("CanonicalizeImports should be enabled by default")
	}
	if !config.PreserveComments {
		t.Error("PreserveComments should be enabled by default")
	}
	if !config.StandardizeWhitespace {
		t.Error("StandardizeWhitespace should be enabled by default")
	}
	if !config.RemoveTrailingWhitespace {
		t.Error("RemoveTrailingWhitespace should be enabled by default")
	}
}

// TestNewNormalizer tests normalizer creation
func TestNewNormalizer(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		normalizer := NewNormalizer(nil)
		if normalizer == nil {
			t.Fatal("NewNormalizer returned nil")
		}
		if normalizer.config == nil {
			t.Error("Config should be initialized to default")
		}
		if !normalizer.config.SortFields {
			t.Error("Default config should have SortFields enabled")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &NormalizationConfig{
			SortFields:     false,
			SortEnumValues: false,
		}
		normalizer := NewNormalizer(config)
		if normalizer == nil {
			t.Fatal("NewNormalizer returned nil")
		}
		if normalizer.config.SortFields {
			t.Error("Custom config should preserve SortFields=false")
		}
	})
}

// TestNormalize_EmptyAST tests normalizing an empty AST
func TestNormalize_EmptyAST(t *testing.T) {
	normalizer := NewNormalizer(DefaultNormalizationConfig())
	ast := &protobuf.RootNode{}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}
	if result == nil {
		t.Fatal("Normalize returned nil result")
	}
}

// TestNormalize_BasicAST tests normalizing a basic AST
func TestNormalize_BasicAST(t *testing.T) {
	normalizer := NewNormalizer(DefaultNormalizationConfig())
	ast := &protobuf.RootNode{
		Syntax: &protobuf.SyntaxNode{Value: "proto3"},
		Package: &protobuf.PackageNode{Name: "test.package"},
		Messages: []*protobuf.MessageNode{
			{Name: "TestMessage"},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}
	if result == nil {
		t.Fatal("Normalize returned nil result")
	}
	if result.Syntax.Value != "proto3" {
		t.Errorf("Syntax = %q, want %q", result.Syntax.Value, "proto3")
	}
	if result.Package.Name != "test.package" {
		t.Errorf("Package = %q, want %q", result.Package.Name, "test.package")
	}
	if len(result.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(result.Messages))
	}
}

// TestCanonicalizeImportPath tests import path canonicalization
func TestCanonicalizeImportPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		config   *NormalizationConfig
		expected string
	}{
		{
			name:     "with quotes",
			input:    "\"google/protobuf/timestamp.proto\"",
			config:   &NormalizationConfig{CanonicalizeImports: true},
			expected: "google/protobuf/timestamp.proto",
		},
		{
			name:     "with backslashes",
			input:    "google\\protobuf\\timestamp.proto",
			config:   &NormalizationConfig{CanonicalizeImports: true},
			expected: "google/protobuf/timestamp.proto",
		},
		{
			name:     "with double slashes",
			input:    "google//protobuf//timestamp.proto",
			config:   &NormalizationConfig{CanonicalizeImports: true},
			expected: "google/protobuf/timestamp.proto",
		},
		{
			name:     "with multiple issues",
			input:    "\"google\\\\protobuf//timestamp.proto\"",
			config:   &NormalizationConfig{CanonicalizeImports: true},
			expected: "google/protobuf/timestamp.proto",
		},
		{
			name:     "disabled canonicalization",
			input:    "\"google//protobuf\"",
			config:   &NormalizationConfig{CanonicalizeImports: false},
			expected: "\"google//protobuf\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizer := NewNormalizer(tt.config)
			result := normalizer.canonicalizeImportPath(tt.input)
			if result != tt.expected {
				t.Errorf("canonicalizeImportPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNormalizeImports tests import normalization
func TestNormalizeImports(t *testing.T) {
	t.Run("sort imports", func(t *testing.T) {
		normalizer := NewNormalizer(&NormalizationConfig{
			SortImports:         true,
			CanonicalizeImports: false,
		})

		imports := []*protobuf.ImportNode{
			{Path: "c.proto"},
			{Path: "a.proto"},
			{Path: "b.proto"},
		}

		result := normalizer.normalizeImports(imports)

		if len(result) != 3 {
			t.Fatalf("Expected 3 imports, got %d", len(result))
		}
		if result[0].Path != "a.proto" {
			t.Errorf("First import = %q, want a.proto", result[0].Path)
		}
		if result[1].Path != "b.proto" {
			t.Errorf("Second import = %q, want b.proto", result[1].Path)
		}
		if result[2].Path != "c.proto" {
			t.Errorf("Third import = %q, want c.proto", result[2].Path)
		}
	})

	t.Run("no sorting", func(t *testing.T) {
		normalizer := NewNormalizer(&NormalizationConfig{
			SortImports:         false,
			CanonicalizeImports: false,
		})

		imports := []*protobuf.ImportNode{
			{Path: "c.proto"},
			{Path: "a.proto"},
		}

		result := normalizer.normalizeImports(imports)

		if result[0].Path != "c.proto" {
			t.Errorf("First import = %q, want c.proto", result[0].Path)
		}
	})

	t.Run("preserve import modifiers", func(t *testing.T) {
		normalizer := NewNormalizer(&NormalizationConfig{
			SortImports:         true,
			CanonicalizeImports: false,
		})

		imports := []*protobuf.ImportNode{
			{Path: "b.proto", Public: true},
			{Path: "a.proto", Weak: true},
		}

		result := normalizer.normalizeImports(imports)

		if !result[0].Weak {
			t.Error("First import should be weak")
		}
		if !result[1].Public {
			t.Error("Second import should be public")
		}
	})
}

// TestNormalizeFields tests field normalization
func TestNormalizeFields(t *testing.T) {
	t.Run("sort by field number", func(t *testing.T) {
		normalizer := NewNormalizer(&NormalizationConfig{SortFields: true})

		fields := []*protobuf.FieldNode{
			{Name: "field3", Number: 3, Type: "string"},
			{Name: "field1", Number: 1, Type: "int32"},
			{Name: "field2", Number: 2, Type: "bool"},
		}

		result := normalizer.normalizeFields(fields)

		if len(result) != 3 {
			t.Fatalf("Expected 3 fields, got %d", len(result))
		}
		if result[0].Number != 1 {
			t.Errorf("First field number = %d, want 1", result[0].Number)
		}
		if result[1].Number != 2 {
			t.Errorf("Second field number = %d, want 2", result[1].Number)
		}
		if result[2].Number != 3 {
			t.Errorf("Third field number = %d, want 3", result[2].Number)
		}
	})

	t.Run("no sorting", func(t *testing.T) {
		normalizer := NewNormalizer(&NormalizationConfig{SortFields: false})

		fields := []*protobuf.FieldNode{
			{Name: "field3", Number: 3, Type: "string"},
			{Name: "field1", Number: 1, Type: "int32"},
		}

		result := normalizer.normalizeFields(fields)

		if result[0].Number != 3 {
			t.Errorf("First field number = %d, want 3", result[0].Number)
		}
	})
}

// TestNormalizeEnumValues tests enum value normalization
func TestNormalizeEnumValues(t *testing.T) {
	t.Run("sort by enum value number", func(t *testing.T) {
		normalizer := NewNormalizer(&NormalizationConfig{SortEnumValues: true})

		values := []*protobuf.EnumValueNode{
			{Name: "VALUE_THREE", Number: 3},
			{Name: "VALUE_ZERO", Number: 0},
			{Name: "VALUE_ONE", Number: 1},
		}

		result := normalizer.normalizeEnumValues(values)

		if len(result) != 3 {
			t.Fatalf("Expected 3 values, got %d", len(result))
		}
		if result[0].Number != 0 {
			t.Errorf("First value number = %d, want 0", result[0].Number)
		}
		if result[1].Number != 1 {
			t.Errorf("Second value number = %d, want 1", result[1].Number)
		}
		if result[2].Number != 3 {
			t.Errorf("Third value number = %d, want 3", result[2].Number)
		}
	})

	t.Run("no sorting", func(t *testing.T) {
		normalizer := NewNormalizer(&NormalizationConfig{SortEnumValues: false})

		values := []*protobuf.EnumValueNode{
			{Name: "VALUE_THREE", Number: 3},
			{Name: "VALUE_ZERO", Number: 0},
		}

		result := normalizer.normalizeEnumValues(values)

		if result[0].Number != 3 {
			t.Errorf("First value number = %d, want 3", result[0].Number)
		}
	})
}

// TestNormalizeMessages tests message normalization
func TestNormalizeMessages(t *testing.T) {
	t.Run("normalize nested messages", func(t *testing.T) {
		normalizer := NewNormalizer(&NormalizationConfig{
			SortFields: true,
		})

		messages := []*protobuf.MessageNode{
			{
				Name: "Parent",
				Fields: []*protobuf.FieldNode{
					{Name: "field2", Number: 2, Type: "string"},
					{Name: "field1", Number: 1, Type: "int32"},
				},
				Nested: []*protobuf.MessageNode{
					{
						Name: "Child",
						Fields: []*protobuf.FieldNode{
							{Name: "child_field", Number: 1, Type: "bool"},
						},
					},
				},
			},
		}

		result := normalizer.normalizeMessages(messages)

		if len(result) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(result))
		}

		// Check field sorting in parent
		if result[0].Fields[0].Number != 1 {
			t.Errorf("First field number = %d, want 1", result[0].Fields[0].Number)
		}

		// Check nested message exists
		if len(result[0].Nested) != 1 {
			t.Errorf("Expected 1 nested message, got %d", len(result[0].Nested))
		}
		if result[0].Nested[0].Name != "Child" {
			t.Errorf("Nested message name = %q, want Child", result[0].Nested[0].Name)
		}
	})

	t.Run("normalize nested enums", func(t *testing.T) {
		normalizer := NewNormalizer(&NormalizationConfig{
			SortEnumValues: true,
		})

		messages := []*protobuf.MessageNode{
			{
				Name: "Parent",
				Enums: []*protobuf.EnumNode{
					{
						Name: "Status",
						Values: []*protobuf.EnumValueNode{
							{Name: "ACTIVE", Number: 1},
							{Name: "UNKNOWN", Number: 0},
						},
					},
				},
			},
		}

		result := normalizer.normalizeMessages(messages)

		if len(result[0].Enums) != 1 {
			t.Fatalf("Expected 1 enum, got %d", len(result[0].Enums))
		}

		// Check enum value sorting
		if result[0].Enums[0].Values[0].Number != 0 {
			t.Errorf("First enum value number = %d, want 0", result[0].Enums[0].Values[0].Number)
		}
	})
}

// TestNormalizeOneOfs tests oneof normalization
func TestNormalizeOneOfs(t *testing.T) {
	normalizer := NewNormalizer(&NormalizationConfig{SortFields: true})

	oneofs := []*protobuf.OneOfNode{
		{
			Name: "choice",
			Fields: []*protobuf.FieldNode{
				{Name: "option2", Number: 2, Type: "string"},
				{Name: "option1", Number: 1, Type: "int32"},
			},
		},
	}

	result := normalizer.normalizeOneOfs(oneofs)

	if len(result) != 1 {
		t.Fatalf("Expected 1 oneof, got %d", len(result))
	}
	if result[0].Name != "choice" {
		t.Errorf("Oneof name = %q, want choice", result[0].Name)
	}

	// Check field sorting
	if result[0].Fields[0].Number != 1 {
		t.Errorf("First field number = %d, want 1", result[0].Fields[0].Number)
	}
	if result[0].Fields[1].Number != 2 {
		t.Errorf("Second field number = %d, want 2", result[0].Fields[1].Number)
	}
}

// TestNormalizeEnums tests enum normalization
func TestNormalizeEnums(t *testing.T) {
	normalizer := NewNormalizer(&NormalizationConfig{SortEnumValues: true})

	enums := []*protobuf.EnumNode{
		{
			Name: "Status",
			Values: []*protobuf.EnumValueNode{
				{Name: "ACTIVE", Number: 2},
				{Name: "UNKNOWN", Number: 0},
				{Name: "INACTIVE", Number: 1},
			},
		},
	}

	result := normalizer.normalizeEnums(enums)

	if len(result) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(result))
	}
	if result[0].Name != "Status" {
		t.Errorf("Enum name = %q, want Status", result[0].Name)
	}

	// Check value sorting
	if result[0].Values[0].Number != 0 {
		t.Errorf("First value number = %d, want 0", result[0].Values[0].Number)
	}
	if result[0].Values[1].Number != 1 {
		t.Errorf("Second value number = %d, want 1", result[0].Values[1].Number)
	}
	if result[0].Values[2].Number != 2 {
		t.Errorf("Third value number = %d, want 2", result[0].Values[2].Number)
	}
}

// TestNormalizeServices tests service normalization
func TestNormalizeServices(t *testing.T) {
	normalizer := NewNormalizer(DefaultNormalizationConfig())

	services := []*protobuf.ServiceNode{
		{Name: "UserService"},
		{Name: "AccountService"},
	}

	result := normalizer.normalizeServices(services)

	// Services should be kept in declaration order
	if len(result) != 2 {
		t.Fatalf("Expected 2 services, got %d", len(result))
	}
	if result[0].Name != "UserService" {
		t.Errorf("First service name = %q, want UserService", result[0].Name)
	}
	if result[1].Name != "AccountService" {
		t.Errorf("Second service name = %q, want AccountService", result[1].Name)
	}
}

// TestNormalize_ComplexAST tests normalizing a complex AST with all features
func TestNormalize_ComplexAST(t *testing.T) {
	normalizer := NewNormalizer(DefaultNormalizationConfig())

	ast := &protobuf.RootNode{
		Syntax: &protobuf.SyntaxNode{Value: "proto3"},
		Package: &protobuf.PackageNode{Name: "test.complex"},
		Imports: []*protobuf.ImportNode{
			{Path: "z.proto"},
			{Path: "a.proto"},
		},
		Messages: []*protobuf.MessageNode{
			{
				Name: "TestMessage",
				Fields: []*protobuf.FieldNode{
					{Name: "field3", Number: 3, Type: "string"},
					{Name: "field1", Number: 1, Type: "int32"},
				},
				Nested: []*protobuf.MessageNode{
					{Name: "Nested"},
				},
				Enums: []*protobuf.EnumNode{
					{
						Name: "Status",
						Values: []*protobuf.EnumValueNode{
							{Name: "ACTIVE", Number: 1},
							{Name: "UNKNOWN", Number: 0},
						},
					},
				},
				OneOfs: []*protobuf.OneOfNode{
					{
						Name: "choice",
						Fields: []*protobuf.FieldNode{
							{Name: "b", Number: 5, Type: "string"},
							{Name: "a", Number: 4, Type: "int32"},
						},
					},
				},
			},
		},
		Enums: []*protobuf.EnumNode{
			{Name: "GlobalEnum"},
		},
		Services: []*protobuf.ServiceNode{
			{Name: "TestService"},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Check imports are sorted
	if result.Imports[0].Path != "a.proto" {
		t.Errorf("First import = %q, want a.proto", result.Imports[0].Path)
	}

	// Check fields are sorted
	if result.Messages[0].Fields[0].Number != 1 {
		t.Errorf("First field number = %d, want 1", result.Messages[0].Fields[0].Number)
	}

	// Check enum values are sorted
	if result.Messages[0].Enums[0].Values[0].Number != 0 {
		t.Errorf("First enum value = %d, want 0", result.Messages[0].Enums[0].Values[0].Number)
	}

	// Check oneof fields are sorted
	if result.Messages[0].OneOfs[0].Fields[0].Number != 4 {
		t.Errorf("First oneof field = %d, want 4", result.Messages[0].OneOfs[0].Fields[0].Number)
	}

	// Check nested structures preserved
	if len(result.Messages[0].Nested) != 1 {
		t.Errorf("Nested messages count = %d, want 1", len(result.Messages[0].Nested))
	}

	// Check other structures preserved
	if len(result.Enums) != 1 {
		t.Errorf("Global enums count = %d, want 1", len(result.Enums))
	}
	if len(result.Services) != 1 {
		t.Errorf("Services count = %d, want 1", len(result.Services))
	}
}

// TestNormalize_PreservesComments tests that comments are preserved
func TestNormalize_PreservesComments(t *testing.T) {
	normalizer := NewNormalizer(&NormalizationConfig{
		PreserveComments: true,
		SortFields:       false,
	})

	ast := &protobuf.RootNode{
		Comments: []*protobuf.CommentNode{
			{Text: "// Root comment"},
		},
		Messages: []*protobuf.MessageNode{
			{
				Name: "TestMessage",
				Comments: []*protobuf.CommentNode{
					{Text: "// Message comment"},
				},
				Fields: []*protobuf.FieldNode{
					{Name: "field1", Number: 1, Type: "string"},
				},
			},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if len(result.Comments) == 0 {
		t.Error("Root comments should be preserved")
	}
	if len(result.Messages[0].Comments) == 0 {
		t.Error("Message comments should be preserved")
	}
}

// TestNormalize_PreservesOptions tests that options are preserved
func TestNormalize_PreservesOptions(t *testing.T) {
	normalizer := NewNormalizer(DefaultNormalizationConfig())

	ast := &protobuf.RootNode{
		Options: []*protobuf.OptionNode{
			{Name: "java_package", Value: "com.example"},
		},
		Messages: []*protobuf.MessageNode{
			{
				Name: "TestMessage",
				Options: []*protobuf.OptionNode{
					{Name: "deprecated", Value: "true"},
				},
			},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if len(result.Options) != 1 {
		t.Error("Root options should be preserved")
	}
	if len(result.Messages[0].Options) != 1 {
		t.Error("Message options should be preserved")
	}
}

// TestNormalize_PreservesPositions tests that position information is preserved
func TestNormalize_PreservesPositions(t *testing.T) {
	normalizer := NewNormalizer(DefaultNormalizationConfig())

	ast := &protobuf.RootNode{
		Pos:    protobuf.Position{Line: 1, Column: 1, Offset: 0},
		EndPos: protobuf.Position{Line: 10, Column: 1, Offset: 100},
		Messages: []*protobuf.MessageNode{
			{
				Name:   "TestMessage",
				Pos:    protobuf.Position{Line: 5, Column: 1, Offset: 50},
				EndPos: protobuf.Position{Line: 8, Column: 1, Offset: 80},
			},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if result.Pos.Line != 1 {
		t.Errorf("Root Pos.Line = %d, want 1", result.Pos.Line)
	}
	if result.EndPos.Line != 10 {
		t.Errorf("Root EndPos.Line = %d, want 10", result.EndPos.Line)
	}
	if result.Messages[0].Pos.Line != 5 {
		t.Errorf("Message Pos.Line = %d, want 5", result.Messages[0].Pos.Line)
	}
}

// TestNormalize_PreservesSpokeDirectives tests that spoke directives are preserved
func TestNormalize_PreservesSpokeDirectives(t *testing.T) {
	normalizer := NewNormalizer(DefaultNormalizationConfig())

	ast := &protobuf.RootNode{
		SpokeDirectives: []*protobuf.SpokeDirectiveNode{
			{Option: "version", Value: "1"},
		},
		Messages: []*protobuf.MessageNode{
			{
				Name: "TestMessage",
				SpokeDirectives: []*protobuf.SpokeDirectiveNode{
					{Option: "table", Value: "users"},
				},
			},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if len(result.SpokeDirectives) != 1 {
		t.Error("Root spoke directives should be preserved")
	}
	if len(result.Messages[0].SpokeDirectives) != 1 {
		t.Error("Message spoke directives should be preserved")
	}
}

// TestNormalize_EmptySlices tests handling of empty slices
func TestNormalize_EmptySlices(t *testing.T) {
	normalizer := NewNormalizer(DefaultNormalizationConfig())

	ast := &protobuf.RootNode{
		Imports:  []*protobuf.ImportNode{},
		Messages: []*protobuf.MessageNode{},
		Enums:    []*protobuf.EnumNode{},
		Services: []*protobuf.ServiceNode{},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if result.Imports == nil {
		t.Error("Imports should not be nil")
	}
	if result.Messages == nil {
		t.Error("Messages should not be nil")
	}
	if result.Enums == nil {
		t.Error("Enums should not be nil")
	}
	if result.Services == nil {
		t.Error("Services should not be nil")
	}
}

// TestNormalize_DifferentConfigs tests normalization with different config combinations
func TestNormalize_DifferentConfigs(t *testing.T) {
	tests := []struct {
		name   string
		config *NormalizationConfig
	}{
		{
			name: "all disabled",
			config: &NormalizationConfig{
				SortFields:               false,
				SortEnumValues:           false,
				SortImports:              false,
				CanonicalizeImports:      false,
				PreserveComments:         false,
				StandardizeWhitespace:    false,
				RemoveTrailingWhitespace: false,
			},
		},
		{
			name: "only sort fields",
			config: &NormalizationConfig{
				SortFields:     true,
				SortEnumValues: false,
				SortImports:    false,
			},
		},
		{
			name: "only canonicalize imports",
			config: &NormalizationConfig{
				SortFields:          false,
				CanonicalizeImports: true,
			},
		},
	}

	ast := &protobuf.RootNode{
		Imports: []*protobuf.ImportNode{
			{Path: "b.proto"},
			{Path: "a.proto"},
		},
		Messages: []*protobuf.MessageNode{
			{
				Name: "Test",
				Fields: []*protobuf.FieldNode{
					{Name: "f2", Number: 2, Type: "string"},
					{Name: "f1", Number: 1, Type: "int32"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizer := NewNormalizer(tt.config)
			result, err := normalizer.Normalize(ast)
			if err != nil {
				t.Fatalf("Normalize failed: %v", err)
			}
			if result == nil {
				t.Fatal("Normalize returned nil")
			}
		})
	}
}

// TestNormalize_MultipleImportsWithCanonicalization tests import canonicalization
func TestNormalize_MultipleImportsWithCanonicalization(t *testing.T) {
	normalizer := NewNormalizer(&NormalizationConfig{
		SortImports:         true,
		CanonicalizeImports: true,
	})

	ast := &protobuf.RootNode{
		Imports: []*protobuf.ImportNode{
			{Path: "\"google\\protobuf\\timestamp.proto\"", Public: false},
			{Path: "a//b//c.proto", Weak: true},
			{Path: "x.proto", Public: true},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Check imports are canonicalized and sorted
	expectedPaths := []string{
		"a/b/c.proto",
		"google/protobuf/timestamp.proto",
		"x.proto",
	}

	for i, expected := range expectedPaths {
		if result.Imports[i].Path != expected {
			t.Errorf("Import %d: got %q, want %q", i, result.Imports[i].Path, expected)
		}
	}

	// Check modifiers are preserved
	if !result.Imports[0].Weak {
		t.Error("First import should be weak")
	}
	if !result.Imports[2].Public {
		t.Error("Third import should be public")
	}
}

// TestNormalize_DeepNestedMessages tests deeply nested message structures
func TestNormalize_DeepNestedMessages(t *testing.T) {
	normalizer := NewNormalizer(&NormalizationConfig{SortFields: true})

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "Level1",
				Fields: []*protobuf.FieldNode{
					{Name: "field2", Number: 2, Type: "string"},
					{Name: "field1", Number: 1, Type: "int32"},
				},
				Nested: []*protobuf.MessageNode{
					{
						Name: "Level2",
						Fields: []*protobuf.FieldNode{
							{Name: "nested_field2", Number: 2, Type: "bool"},
							{Name: "nested_field1", Number: 1, Type: "string"},
						},
						Nested: []*protobuf.MessageNode{
							{
								Name: "Level3",
								Fields: []*protobuf.FieldNode{
									{Name: "deep_field3", Number: 3, Type: "int32"},
									{Name: "deep_field1", Number: 1, Type: "int32"},
									{Name: "deep_field2", Number: 2, Type: "int32"},
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Check Level1 fields are sorted
	if result.Messages[0].Fields[0].Number != 1 {
		t.Errorf("Level1 first field number = %d, want 1", result.Messages[0].Fields[0].Number)
	}

	// Check Level2 fields are sorted
	level2 := result.Messages[0].Nested[0]
	if level2.Fields[0].Number != 1 {
		t.Errorf("Level2 first field number = %d, want 1", level2.Fields[0].Number)
	}

	// Check Level3 fields are sorted
	level3 := level2.Nested[0]
	if level3.Fields[0].Number != 1 {
		t.Errorf("Level3 first field number = %d, want 1", level3.Fields[0].Number)
	}
	if level3.Fields[1].Number != 2 {
		t.Errorf("Level3 second field number = %d, want 2", level3.Fields[1].Number)
	}
	if level3.Fields[2].Number != 3 {
		t.Errorf("Level3 third field number = %d, want 3", level3.Fields[2].Number)
	}
}

// TestNormalize_MultipleOneOfs tests multiple oneof fields
func TestNormalize_MultipleOneOfs(t *testing.T) {
	normalizer := NewNormalizer(&NormalizationConfig{SortFields: true})

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "TestMessage",
				OneOfs: []*protobuf.OneOfNode{
					{
						Name: "choice1",
						Fields: []*protobuf.FieldNode{
							{Name: "option_b", Number: 3, Type: "string"},
							{Name: "option_a", Number: 2, Type: "int32"},
						},
					},
					{
						Name: "choice2",
						Fields: []*protobuf.FieldNode{
							{Name: "other_b", Number: 5, Type: "bool"},
							{Name: "other_a", Number: 4, Type: "float"},
						},
					},
				},
			},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Check first oneof fields are sorted
	if result.Messages[0].OneOfs[0].Fields[0].Number != 2 {
		t.Errorf("First oneof first field = %d, want 2", result.Messages[0].OneOfs[0].Fields[0].Number)
	}

	// Check second oneof fields are sorted
	if result.Messages[0].OneOfs[1].Fields[0].Number != 4 {
		t.Errorf("Second oneof first field = %d, want 4", result.Messages[0].OneOfs[1].Fields[0].Number)
	}
}

// TestNormalize_EmptyImportPath tests edge case of empty import path
func TestNormalize_EmptyImportPath(t *testing.T) {
	normalizer := NewNormalizer(&NormalizationConfig{
		SortImports:         true,
		CanonicalizeImports: true,
	})

	ast := &protobuf.RootNode{
		Imports: []*protobuf.ImportNode{
			{Path: ""},
			{Path: "a.proto"},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Empty path should sort before non-empty
	if result.Imports[0].Path != "" {
		t.Errorf("First import should be empty string, got %q", result.Imports[0].Path)
	}
}

// TestNormalize_FieldsWithGaps tests fields with non-sequential numbers
func TestNormalize_FieldsWithGaps(t *testing.T) {
	normalizer := NewNormalizer(&NormalizationConfig{SortFields: true})

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "TestMessage",
				Fields: []*protobuf.FieldNode{
					{Name: "field100", Number: 100, Type: "string"},
					{Name: "field1", Number: 1, Type: "int32"},
					{Name: "field50", Number: 50, Type: "bool"},
					{Name: "field25", Number: 25, Type: "float"},
				},
			},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Check fields are sorted correctly
	expectedNumbers := []int{1, 25, 50, 100}
	for i, expected := range expectedNumbers {
		if result.Messages[0].Fields[i].Number != expected {
			t.Errorf("Field %d: number = %d, want %d", i, result.Messages[0].Fields[i].Number, expected)
		}
	}
}

// TestNormalize_EnumValuesWithNegatives tests enum values including negative numbers
func TestNormalize_EnumValuesWithNegatives(t *testing.T) {
	normalizer := NewNormalizer(&NormalizationConfig{SortEnumValues: true})

	ast := &protobuf.RootNode{
		Enums: []*protobuf.EnumNode{
			{
				Name: "Status",
				Values: []*protobuf.EnumValueNode{
					{Name: "POSITIVE", Number: 1},
					{Name: "NEGATIVE", Number: -1},
					{Name: "ZERO", Number: 0},
					{Name: "LARGE", Number: 100},
				},
			},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Check values are sorted correctly (negative first)
	expectedNumbers := []int{-1, 0, 1, 100}
	for i, expected := range expectedNumbers {
		if result.Enums[0].Values[i].Number != expected {
			t.Errorf("Value %d: number = %d, want %d", i, result.Enums[0].Values[i].Number, expected)
		}
	}
}

// TestNormalize_MixedMessageAndEnumOptions tests preservation of various options
func TestNormalize_MixedMessageAndEnumOptions(t *testing.T) {
	normalizer := NewNormalizer(DefaultNormalizationConfig())

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "TestMessage",
				Options: []*protobuf.OptionNode{
					{Name: "deprecated", Value: "true"},
					{Name: "message_set_wire_format", Value: "false"},
				},
			},
		},
		Enums: []*protobuf.EnumNode{
			{
				Name: "Status",
				Values: []*protobuf.EnumValueNode{
					{Name: "UNKNOWN", Number: 0},
				},
				Options: []*protobuf.OptionNode{
					{Name: "allow_alias", Value: "true"},
				},
			},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if len(result.Messages[0].Options) != 2 {
		t.Errorf("Message options count = %d, want 2", len(result.Messages[0].Options))
	}
	if len(result.Enums[0].Options) != 1 {
		t.Errorf("Enum options count = %d, want 1", len(result.Enums[0].Options))
	}
}

// TestNormalize_AllNodeTypes tests normalization with all node types
func TestNormalize_AllNodeTypes(t *testing.T) {
	normalizer := NewNormalizer(DefaultNormalizationConfig())

	ast := &protobuf.RootNode{
		Syntax:  &protobuf.SyntaxNode{Value: "proto3"},
		Package: &protobuf.PackageNode{Name: "test.all"},
		Imports: []*protobuf.ImportNode{
			{Path: "z.proto"},
			{Path: "a.proto"},
		},
		Options: []*protobuf.OptionNode{
			{Name: "go_package", Value: "github.com/example/test"},
		},
		Messages: []*protobuf.MessageNode{
			{
				Name: "Message1",
				Fields: []*protobuf.FieldNode{
					{Name: "field1", Number: 1, Type: "string"},
				},
			},
		},
		Enums: []*protobuf.EnumNode{
			{
				Name: "Enum1",
				Values: []*protobuf.EnumValueNode{
					{Name: "VALUE_0", Number: 0},
				},
			},
		},
		Services: []*protobuf.ServiceNode{
			{
				Name: "Service1",
				RPCs: []*protobuf.RPCNode{
					{
						Name:       "Method1",
						InputType:  "Message1",
						OutputType: "Message1",
					},
				},
			},
		},
		Comments: []*protobuf.CommentNode{
			{Text: "// File comment"},
		},
		SpokeDirectives: []*protobuf.SpokeDirectiveNode{
			{Option: "version", Value: "1"},
		},
	}

	result, err := normalizer.Normalize(ast)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Verify all node types are present
	if result.Syntax == nil {
		t.Error("Syntax should be preserved")
	}
	if result.Package == nil {
		t.Error("Package should be preserved")
	}
	if len(result.Imports) != 2 {
		t.Errorf("Imports count = %d, want 2", len(result.Imports))
	}
	if len(result.Options) != 1 {
		t.Errorf("Options count = %d, want 1", len(result.Options))
	}
	if len(result.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(result.Messages))
	}
	if len(result.Enums) != 1 {
		t.Errorf("Enums count = %d, want 1", len(result.Enums))
	}
	if len(result.Services) != 1 {
		t.Errorf("Services count = %d, want 1", len(result.Services))
	}
	if len(result.Comments) != 1 {
		t.Errorf("Comments count = %d, want 1", len(result.Comments))
	}
	if len(result.SpokeDirectives) != 1 {
		t.Errorf("SpokeDirectives count = %d, want 1", len(result.SpokeDirectives))
	}

	// Check imports are sorted
	if result.Imports[0].Path != "a.proto" {
		t.Errorf("First import = %q, want a.proto", result.Imports[0].Path)
	}
}

// TestCanonicalizeImportPath_EdgeCases tests edge cases for canonicalization
func TestCanonicalizeImportPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "triple slashes",
			input:    "a///b///c.proto",
			expected: "a/b/c.proto",
		},
		{
			name:     "mixed slashes",
			input:    "a\\b/c\\d.proto",
			expected: "a/b/c/d.proto",
		},
		{
			name:     "only quotes",
			input:    "\"\"",
			expected: "",
		},
		{
			name:     "quotes with spaces",
			input:    "\" a/b/c.proto \"",
			expected: " a/b/c.proto ",
		},
		{
			name:     "leading and trailing slashes",
			input:    "/a/b/c.proto/",
			expected: "/a/b/c.proto/",
		},
	}

	normalizer := NewNormalizer(&NormalizationConfig{CanonicalizeImports: true})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.canonicalizeImportPath(tt.input)
			if result != tt.expected {
				t.Errorf("canonicalizeImportPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
