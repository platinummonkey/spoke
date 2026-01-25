package search

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/stretchr/testify/assert"
)

func TestExtractComments(t *testing.T) {
	idx := &Indexer{}

	tests := []struct {
		name     string
		comments []*protobuf.CommentNode
		expected string
	}{
		{
			name:     "no comments",
			comments: nil,
			expected: "",
		},
		{
			name: "single comment",
			comments: []*protobuf.CommentNode{
				{Text: "This is a comment"},
			},
			expected: "This is a comment",
		},
		{
			name: "multiple comments",
			comments: []*protobuf.CommentNode{
				{Text: "First comment"},
				{Text: "Second comment"},
			},
			expected: "First comment Second comment",
		},
		{
			name: "comments with whitespace",
			comments: []*protobuf.CommentNode{
				{Text: "  Trimmed  "},
				{Text: "  Also trimmed  "},
			},
			expected: "Trimmed Also trimmed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := idx.extractComments(tt.comments)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDescription(t *testing.T) {
	idx := &Indexer{}

	tests := []struct {
		name     string
		comments []*protobuf.CommentNode
		expected string
	}{
		{
			name:     "no comments",
			comments: nil,
			expected: "",
		},
		{
			name: "single line comment",
			comments: []*protobuf.CommentNode{
				{Text: "Description line"},
			},
			expected: "Description line",
		},
		{
			name: "multiline comment",
			comments: []*protobuf.CommentNode{
				{Text: "First line\nSecond line\nThird line"},
			},
			expected: "First line",
		},
		{
			name: "comment with leading whitespace",
			comments: []*protobuf.CommentNode{
				{Text: "  \n  Description with leading whitespace"},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := idx.extractDescription(tt.comments)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNullString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "non-empty string",
			input:    "hello",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNullInt(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected interface{}
	}{
		{
			name:     "zero",
			input:    0,
			expected: nil,
		},
		{
			name:     "non-zero",
			input:    42,
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractMessageEntities(t *testing.T) {
	idx := &Indexer{}

	msg := &protobuf.MessageNode{
		Name: "User",
		Fields: []*protobuf.FieldNode{
			{
				Name:   "id",
				Type:   "string",
				Number: 1,
				Pos:    protobuf.Position{Line: 10},
			},
			{
				Name:     "emails",
				Type:     "string",
				Number:   2,
				Repeated: true,
				Pos:      protobuf.Position{Line: 11},
			},
		},
		Pos: protobuf.Position{Line: 5},
	}

	entities := idx.extractMessageEntities(1, msg, "example.v1", "user.proto")

	assert.Len(t, entities, 3) // 1 message + 2 fields

	// Check message entity
	assert.Equal(t, "message", entities[0].EntityType)
	assert.Equal(t, "User", entities[0].EntityName)
	assert.Equal(t, "example.v1.User", entities[0].FullPath)
	assert.Equal(t, "example.v1", entities[0].ParentPath)
	assert.Equal(t, 5, entities[0].LineNumber)

	// Check first field entity
	assert.Equal(t, "field", entities[1].EntityType)
	assert.Equal(t, "id", entities[1].EntityName)
	assert.Equal(t, "example.v1.User.id", entities[1].FullPath)
	assert.Equal(t, "example.v1.User", entities[1].ParentPath)
	assert.Equal(t, "string", entities[1].FieldType)
	assert.Equal(t, 1, entities[1].FieldNumber)
	assert.False(t, entities[1].IsRepeated)

	// Check second field entity (repeated)
	assert.Equal(t, "field", entities[2].EntityType)
	assert.Equal(t, "emails", entities[2].EntityName)
	assert.True(t, entities[2].IsRepeated)
}

func TestExtractEnumEntities(t *testing.T) {
	idx := &Indexer{}

	enum := &protobuf.EnumNode{
		Name: "Status",
		Values: []*protobuf.EnumValueNode{
			{
				Name:   "UNKNOWN",
				Number: 0,
				Pos:    protobuf.Position{Line: 15},
			},
			{
				Name:   "ACTIVE",
				Number: 1,
				Pos:    protobuf.Position{Line: 16},
			},
		},
		Pos: protobuf.Position{Line: 14},
	}

	entities := idx.extractEnumEntities(1, enum, "example.v1", "status.proto")

	assert.Len(t, entities, 3) // 1 enum + 2 values

	// Check enum entity
	assert.Equal(t, "enum", entities[0].EntityType)
	assert.Equal(t, "Status", entities[0].EntityName)
	assert.Equal(t, "example.v1.Status", entities[0].FullPath)
	assert.Equal(t, 14, entities[0].LineNumber)

	// Check enum values
	assert.Equal(t, "enum_value", entities[1].EntityType)
	assert.Equal(t, "UNKNOWN", entities[1].EntityName)
	assert.Equal(t, 0, entities[1].FieldNumber)

	assert.Equal(t, "enum_value", entities[2].EntityType)
	assert.Equal(t, "ACTIVE", entities[2].EntityName)
	assert.Equal(t, 1, entities[2].FieldNumber)
}

func TestExtractServiceEntities(t *testing.T) {
	idx := &Indexer{}

	service := &protobuf.ServiceNode{
		Name: "UserService",
		RPCs: []*protobuf.RPCNode{
			{
				Name:       "GetUser",
				InputType:  "GetUserRequest",
				OutputType: "User",
				Pos:        protobuf.Position{Line: 25},
			},
			{
				Name:            "StreamUsers",
				InputType:       "StreamUsersRequest",
				OutputType:      "User",
				ServerStreaming: true,
				Pos:             protobuf.Position{Line: 26},
			},
		},
		Pos: protobuf.Position{Line: 24},
	}

	entities := idx.extractServiceEntities(1, service, "example.v1", "service.proto")

	assert.Len(t, entities, 3) // 1 service + 2 methods

	// Check service entity
	assert.Equal(t, "service", entities[0].EntityType)
	assert.Equal(t, "UserService", entities[0].EntityName)
	assert.Equal(t, "example.v1.UserService", entities[0].FullPath)
	assert.Equal(t, 24, entities[0].LineNumber)

	// Check first method
	assert.Equal(t, "method", entities[1].EntityType)
	assert.Equal(t, "GetUser", entities[1].EntityName)
	assert.Equal(t, "GetUserRequest", entities[1].MethodInputType)
	assert.Equal(t, "User", entities[1].MethodOutputType)
	assert.Empty(t, entities[1].Metadata) // No streaming

	// Check second method (streaming)
	assert.Equal(t, "method", entities[2].EntityType)
	assert.Equal(t, "StreamUsers", entities[2].EntityName)
	assert.Contains(t, entities[2].Metadata, "server_streaming")
	assert.True(t, entities[2].Metadata["server_streaming"].(bool))
}
