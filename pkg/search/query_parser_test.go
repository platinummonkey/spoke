package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryParser_ParseBasic(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name     string
		input    string
		expected *ParsedQuery
	}{
		{
			name:  "simple term",
			input: "user",
			expected: &ParsedQuery{
				Terms: []string{"user"},
				Raw:   "user",
			},
		},
		{
			name:  "multiple terms",
			input: "user email address",
			expected: &ParsedQuery{
				Terms: []string{"user", "email", "address"},
				Raw:   "user email address",
			},
		},
		{
			name:  "empty query",
			input: "",
			expected: &ParsedQuery{
				Terms: []string{},
				Raw:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Terms, result.Terms)
			assert.Equal(t, tt.expected.Raw, result.Raw)
		})
	}
}

func TestQueryParser_ParseEntityFilter(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name         string
		input        string
		expectedType []string
	}{
		{
			name:         "entity:message",
			input:        "user entity:message",
			expectedType: []string{"message"},
		},
		{
			name:         "entity:field",
			input:        "email entity:field",
			expectedType: []string{"field"},
		},
		{
			name:         "entity:enum",
			input:        "Status entity:enum",
			expectedType: []string{"enum"},
		},
		{
			name:         "entity:service",
			input:        "UserService entity:service",
			expectedType: []string{"service"},
		},
		{
			name:         "entity:method",
			input:        "CreateUser entity:method",
			expectedType: []string{"method"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedType, result.EntityTypes)
		})
	}
}

func TestQueryParser_ParseInvalidEntity(t *testing.T) {
	parser := NewQueryParser()

	result, err := parser.Parse("user entity:invalid")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestQueryParser_ParseFieldTypeFilter(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name         string
		input        string
		expectedType []string
	}{
		{
			name:         "type:string",
			input:        "user type:string",
			expectedType: []string{"string"},
		},
		{
			name:         "type:int32",
			input:        "id type:int32",
			expectedType: []string{"int32"},
		},
		{
			name:         "type:bool",
			input:        "active type:bool",
			expectedType: []string{"bool"},
		},
		{
			name:         "multiple type filters",
			input:        "field type:string type:int32",
			expectedType: []string{"string", "int32"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedType, result.FieldTypes)
		})
	}
}

func TestQueryParser_ParseModuleFilter(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name           string
		input          string
		expectedModule string
	}{
		{
			name:           "module:user",
			input:          "email module:user",
			expectedModule: "user",
		},
		{
			name:           "module with wildcard",
			input:          "Status module:common.*",
			expectedModule: "common.*",
		},
		{
			name:           "quoted module name",
			input:          `module:"user-service"`,
			expectedModule: "user-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedModule, result.ModulePattern)
		})
	}
}

func TestQueryParser_ParseVersionFilter(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name            string
		input           string
		expectedVersion string
	}{
		{
			name:            "version:1.0.0",
			input:           "user version:1.0.0",
			expectedVersion: "1.0.0",
		},
		{
			name:            "version:>=1.0.0",
			input:           "user version:>=1.0.0",
			expectedVersion: ">=1.0.0",
		},
		{
			name:            "version:~1.2.0",
			input:           "user version:~1.2.0",
			expectedVersion: "~1.2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedVersion, result.VersionConstraint)
		})
	}
}

func TestQueryParser_ParseImportsFilter(t *testing.T) {
	parser := NewQueryParser()

	result, err := parser.Parse("User imports:common.proto")
	require.NoError(t, err)
	assert.Equal(t, []string{"common.proto"}, result.Imports)
}

func TestQueryParser_ParseDependsOnFilter(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name        string
		input       string
		expectedDep []string
	}{
		{
			name:        "depends-on:common",
			input:       "User depends-on:common",
			expectedDep: []string{"common"},
		},
		{
			name:        "depends_on:common (underscore)",
			input:       "User depends_on:common",
			expectedDep: []string{"common"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedDep, result.DependsOn)
		})
	}
}

func TestQueryParser_ParseHasCommentFilter(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name        string
		input       string
		hasComment  bool
	}{
		{
			name:       "has-comment:true",
			input:      "deprecated has-comment:true",
			hasComment: true,
		},
		{
			name:       "has-comment:1",
			input:      "deprecated has-comment:1",
			hasComment: true,
		},
		{
			name:       "has-comment:yes",
			input:      "deprecated has-comment:yes",
			hasComment: true,
		},
		{
			name:       "has-comment:false",
			input:      "deprecated has-comment:false",
			hasComment: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.hasComment, result.HasComment)
		})
	}
}

func TestQueryParser_ParseBooleanOperators(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name              string
		input             string
		expectedTerms     []string
		expectedOperators []string
	}{
		{
			name:              "OR operator",
			input:             "user OR email",
			expectedTerms:     []string{"user", "email"},
			expectedOperators: []string{"OR"},
		},
		{
			name:              "AND operator",
			input:             "user AND active",
			expectedTerms:     []string{"user", "active"},
			expectedOperators: []string{"AND"},
		},
		{
			name:              "NOT operator",
			input:             "user NOT deleted",
			expectedTerms:     []string{"user", "deleted"},
			expectedOperators: []string{"NOT"},
		},
		{
			name:              "multiple operators",
			input:             "user AND active OR admin",
			expectedTerms:     []string{"user", "active", "admin"},
			expectedOperators: []string{"AND", "OR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedTerms, result.Terms)
			assert.Equal(t, tt.expectedOperators, result.Operators)
		})
	}
}

func TestQueryParser_ParseCombinedFilters(t *testing.T) {
	parser := NewQueryParser()

	input := "email entity:field type:string module:user version:>=1.0.0"
	result, err := parser.Parse(input)
	require.NoError(t, err)

	assert.Equal(t, []string{"email"}, result.Terms)
	assert.Equal(t, []string{"field"}, result.EntityTypes)
	assert.Equal(t, []string{"string"}, result.FieldTypes)
	assert.Equal(t, "user", result.ModulePattern)
	assert.Equal(t, ">=1.0.0", result.VersionConstraint)
}

func TestQueryParser_ToTsQuery(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single term",
			input:    "user",
			expected: "user:*",
		},
		{
			name:     "multiple terms (AND)",
			input:    "user email",
			expected: "user:* & email:*",
		},
		{
			name:     "OR operator",
			input:    "user OR email",
			expected: "user:* | email:*",
		},
		{
			name:     "NOT operator",
			input:    "user NOT deleted",
			expected: "user:* &! deleted:*",
		},
		{
			name:     "explicit AND operator",
			input:    "user AND active",
			expected: "user:* & active:*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			tsquery := result.ToTsQuery()
			assert.Equal(t, tt.expected, tsquery)
		})
	}
}

func TestQueryParser_HasFilters(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name       string
		input      string
		hasFilters bool
	}{
		{
			name:       "no filters",
			input:      "user email",
			hasFilters: false,
		},
		{
			name:       "entity filter",
			input:      "user entity:message",
			hasFilters: true,
		},
		{
			name:       "type filter",
			input:      "email type:string",
			hasFilters: true,
		},
		{
			name:       "module filter",
			input:      "user module:common",
			hasFilters: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.hasFilters, result.HasFilters())
		})
	}
}

func TestQueryParser_String(t *testing.T) {
	parser := NewQueryParser()

	input := "email entity:field type:string"
	result, err := parser.Parse(input)
	require.NoError(t, err)

	str := result.String()
	assert.Contains(t, str, "terms:[email]")
	assert.Contains(t, str, "entity:[field]")
	assert.Contains(t, str, "type:[string]")
}

func TestSanitizeTsQueryTerm(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple term",
			input:    "user",
			expected: "user:*",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace",
			input:    "  ",
			expected: "",
		},
		{
			name:     "term with quotes",
			input:    "user's",
			expected: "user''s:*",
		},
		{
			name:     "operator-like term",
			input:    "&",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeTsQueryTerm(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
