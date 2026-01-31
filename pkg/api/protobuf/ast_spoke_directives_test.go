package protobuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSpokeDirective(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid domain directive",
			input:    "@spoke:domain:github.com/example/test",
			expected: true,
		},
		{
			name:     "valid option directive",
			input:    "@spoke:option:some-value",
			expected: true,
		},
		{
			name:     "regular comment",
			input:    "This is a regular comment",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "@ but not spoke",
			input:    "@other:directive",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSpokeDirective(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSpokeDirective(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedOpt   string
		expectedVal   string
		expectError   bool
	}{
		{
			name:        "domain directive",
			input:       "@spoke:domain:github.com/example/test",
			expectedOpt: "domain",
			expectedVal: "github.com/example/test",
			expectError: false,
		},
		{
			name:        "option directive",
			input:       "@spoke:option:required",
			expectedOpt: "option",
			expectedVal: "required",
			expectError: false,
		},
		{
			name:        "directive with spaces",
			input:       "@spoke: domain : github.com/example/test ",
			expectedOpt: "domain",
			expectedVal: "github.com/example/test",
			expectError: false,
		},
		{
			name:        "missing value",
			input:       "@spoke:domain",
			expectedOpt: "",
			expectedVal: "",
			expectError: true,
		},
		{
			name:        "not a spoke directive",
			input:       "regular comment",
			expectedOpt: "",
			expectedVal: "",
			expectError: true,
		},
		{
			name:        "value with colons",
			input:       "@spoke:option:key:value:extra",
			expectedOpt: "option",
			expectedVal: "key:value:extra",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directive, err := ExtractSpokeDirective(tt.input, 1, 0)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOpt, directive.Option)
				assert.Equal(t, tt.expectedVal, directive.Value)
				assert.Equal(t, 1, directive.Pos.Line)
			}
		})
	}
}

func TestParseSpokeDirectivesFromContent(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		expectedDirectives int
		expectedComments   int
		checkDirective    func(t *testing.T, directives map[int]*SpokeDirectiveNode)
	}{
		{
			name: "single directive",
			content: `// @spoke:domain:github.com/example/test
package test;`,
			expectedDirectives: 1,
			expectedComments:   0,
			checkDirective: func(t *testing.T, directives map[int]*SpokeDirectiveNode) {
				directive, ok := directives[1]
				require.True(t, ok, "Should have directive at line 1")
				assert.Equal(t, "domain", directive.Option)
				assert.Equal(t, "github.com/example/test", directive.Value)
			},
		},
		{
			name: "multiple directives",
			content: `// @spoke:domain:github.com/example/test
// @spoke:option:required
package test;`,
			expectedDirectives: 2,
			expectedComments:   0,
			checkDirective: func(t *testing.T, directives map[int]*SpokeDirectiveNode) {
				directive1, ok := directives[1]
				require.True(t, ok)
				assert.Equal(t, "domain", directive1.Option)

				directive2, ok := directives[2]
				require.True(t, ok)
				assert.Equal(t, "option", directive2.Option)
			},
		},
		{
			name: "mixed directives and comments",
			content: `// This is a regular comment
// @spoke:domain:github.com/example/test
// Another comment
package test;`,
			expectedDirectives: 1,
			expectedComments:   2,
			checkDirective: func(t *testing.T, directives map[int]*SpokeDirectiveNode) {
				directive, ok := directives[2]
				require.True(t, ok)
				assert.Equal(t, "domain", directive.Option)
			},
		},
		{
			name: "directive on message field",
			content: `package test;

message User {
  // @spoke:option:required
  string name = 1;
}`,
			expectedDirectives: 1,
			expectedComments:   0,
			checkDirective: func(t *testing.T, directives map[int]*SpokeDirectiveNode) {
				directive, ok := directives[4]
				require.True(t, ok)
				assert.Equal(t, "option", directive.Option)
				assert.Equal(t, "required", directive.Value)
			},
		},
		{
			name: "no directives",
			content: `// Regular comment
package test;

// Another comment
message User {
  string name = 1;
}`,
			expectedDirectives: 0,
			expectedComments:   2,
			checkDirective: func(t *testing.T, directives map[int]*SpokeDirectiveNode) {
				assert.Len(t, directives, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directives, comments, err := ParseSpokeDirectivesFromContent(tt.content)
			require.NoError(t, err)

			assert.Len(t, directives, tt.expectedDirectives, "Expected %d directives", tt.expectedDirectives)

			totalComments := 0
			for _, commentList := range comments {
				totalComments += len(commentList)
			}
			assert.Equal(t, tt.expectedComments, totalComments, "Expected %d comments", tt.expectedComments)

			if tt.checkDirective != nil {
				tt.checkDirective(t, directives)
			}
		})
	}
}

func TestAssociateSpokeDirectivesWithNode(t *testing.T) {
	t.Run("associate with package node", func(t *testing.T) {
		directives := map[int]*SpokeDirectiveNode{
			1: {Option: "domain", Value: "github.com/example/test"},
		}
		comments := map[int][]*CommentNode{
			2: {{Text: "// Package comment"}},
		}

		pkg := &PackageNode{
			Name:            "test",
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
			Comments:        make([]*CommentNode, 0),
		}

		AssociateSpokeDirectivesWithNode(pkg, directives, comments, 3)

		assert.Len(t, pkg.SpokeDirectives, 1)
		assert.Equal(t, "domain", pkg.SpokeDirectives[0].Option)
		assert.Len(t, pkg.Comments, 1)
	})

	t.Run("associate with message node", func(t *testing.T) {
		directives := map[int]*SpokeDirectiveNode{
			5: {Option: "option", Value: "serializable"},
		}
		comments := map[int][]*CommentNode{}

		msg := &MessageNode{
			Name:            "User",
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
			Comments:        make([]*CommentNode, 0),
		}

		AssociateSpokeDirectivesWithNode(msg, directives, comments, 6)

		assert.Len(t, msg.SpokeDirectives, 1)
		assert.Equal(t, "option", msg.SpokeDirectives[0].Option)
		assert.Equal(t, "serializable", msg.SpokeDirectives[0].Value)
	})

	t.Run("associate with field node", func(t *testing.T) {
		directives := map[int]*SpokeDirectiveNode{
			10: {Option: "option", Value: "required"},
		}
		comments := map[int][]*CommentNode{
			10: {{Text: "// Field comment"}},
		}

		field := &FieldNode{
			Name:            "name",
			Type:            "string",
			Number:          1,
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
			Comments:        make([]*CommentNode, 0),
		}

		AssociateSpokeDirectivesWithNode(field, directives, comments, 11)

		assert.Len(t, field.SpokeDirectives, 1)
		assert.Equal(t, "option", field.SpokeDirectives[0].Option)
		assert.Len(t, field.Comments, 1)
	})

	t.Run("no association when lines don't match", func(t *testing.T) {
		directives := map[int]*SpokeDirectiveNode{
			1: {Option: "domain", Value: "test"},
		}
		comments := map[int][]*CommentNode{}

		msg := &MessageNode{
			Name:            "User",
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
			Comments:        make([]*CommentNode, 0),
		}

		// Line 20 is too far from line 1
		AssociateSpokeDirectivesWithNode(msg, directives, comments, 20)

		assert.Len(t, msg.SpokeDirectives, 0)
	})
}
