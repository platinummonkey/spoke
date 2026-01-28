package search

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStorageReader is a mock implementation of StorageReader for testing
type mockStorageReader struct {
	modules  []*Module
	versions map[string][]*Version
	version  *Version
	file     *File
	err      error
}

func (m *mockStorageReader) GetVersion(moduleName, version string) (*Version, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.version, nil
}

func (m *mockStorageReader) GetFile(moduleName, version, path string) (*File, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.file, nil
}

func (m *mockStorageReader) ListModules() ([]*Module, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.modules, nil
}

func (m *mockStorageReader) ListVersions(moduleName string) ([]*Version, error) {
	if m.err != nil {
		return nil, m.err
	}
	if versions, ok := m.versions[moduleName]; ok {
		return versions, nil
	}
	return []*Version{}, nil
}

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
		{
			name: "comments with empty text",
			comments: []*protobuf.CommentNode{
				{Text: ""},
				{Text: "Non-empty comment"},
				{Text: ""},
			},
			expected: "Non-empty comment",
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

func TestNewIndexer(t *testing.T) {
	db := &sql.DB{}
	storage := &mockStorageReader{}

	indexer := NewIndexer(db, storage)

	assert.NotNil(t, indexer)
	assert.Equal(t, db, indexer.db)
	assert.Equal(t, storage, indexer.storage)
}

func TestExtractEntities(t *testing.T) {
	idx := &Indexer{}

	ast := &protobuf.RootNode{
		Package: &protobuf.PackageNode{Name: "test.v1"},
		Messages: []*protobuf.MessageNode{
			{
				Name: "User",
				Pos:  protobuf.Position{Line: 5},
			},
		},
		Enums: []*protobuf.EnumNode{
			{
				Name: "Status",
				Pos:  protobuf.Position{Line: 10},
			},
		},
		Services: []*protobuf.ServiceNode{
			{
				Name: "UserService",
				Pos:  protobuf.Position{Line: 15},
			},
		},
	}

	entities := idx.extractEntities(context.Background(), 1, ast, "test.proto")

	// Should have 1 message + 1 enum + 1 service = 3 entities
	assert.Len(t, entities, 3)

	// Verify message entity
	assert.Equal(t, "message", entities[0].EntityType)
	assert.Equal(t, "User", entities[0].EntityName)
	assert.Equal(t, "test.v1.User", entities[0].FullPath)

	// Verify enum entity
	assert.Equal(t, "enum", entities[1].EntityType)
	assert.Equal(t, "Status", entities[1].EntityName)

	// Verify service entity
	assert.Equal(t, "service", entities[2].EntityType)
	assert.Equal(t, "UserService", entities[2].EntityName)
}

func TestExtractEntitiesNoPackage(t *testing.T) {
	idx := &Indexer{}

	ast := &protobuf.RootNode{
		Messages: []*protobuf.MessageNode{
			{
				Name: "User",
				Pos:  protobuf.Position{Line: 5},
			},
		},
	}

	entities := idx.extractEntities(context.Background(), 1, ast, "test.proto")

	assert.Len(t, entities, 1)
	assert.Equal(t, "User", entities[0].FullPath) // No package prefix
}

func TestExtractMessageEntitiesWithNested(t *testing.T) {
	idx := &Indexer{}

	msg := &protobuf.MessageNode{
		Name: "User",
		Nested: []*protobuf.MessageNode{
			{
				Name: "Address",
				Pos:  protobuf.Position{Line: 15},
			},
		},
		Enums: []*protobuf.EnumNode{
			{
				Name: "Type",
				Pos:  protobuf.Position{Line: 20},
			},
		},
		Pos: protobuf.Position{Line: 5},
	}

	entities := idx.extractMessageEntities(1, msg, "example.v1", "user.proto")

	// 1 message + 1 nested message + 1 nested enum = 3
	assert.Len(t, entities, 3)
	assert.Equal(t, "message", entities[0].EntityType)
	assert.Equal(t, "User", entities[0].EntityName)
	assert.Equal(t, "message", entities[1].EntityType)
	assert.Equal(t, "Address", entities[1].EntityName)
	assert.Equal(t, "example.v1.User.Address", entities[1].FullPath)
	assert.Equal(t, "enum", entities[2].EntityType)
	assert.Equal(t, "Type", entities[2].EntityName)
}

func TestExtractMessageEntitiesWithComments(t *testing.T) {
	idx := &Indexer{}

	msg := &protobuf.MessageNode{
		Name: "User",
		Comments: []*protobuf.CommentNode{
			{Text: "User represents a system user"},
			{Text: "Additional documentation"},
		},
		Pos: protobuf.Position{Line: 5},
	}

	entities := idx.extractMessageEntities(1, msg, "", "user.proto")

	assert.Len(t, entities, 1)
	assert.Equal(t, "User represents a system user", entities[0].Description)
	assert.Contains(t, entities[0].Comments, "User represents a system user")
	assert.Contains(t, entities[0].Comments, "Additional documentation")
}

func TestExtractMessageEntitiesOptionalField(t *testing.T) {
	idx := &Indexer{}

	msg := &protobuf.MessageNode{
		Name: "User",
		Fields: []*protobuf.FieldNode{
			{
				Name:     "email",
				Type:     "string",
				Number:   1,
				Optional: true,
				Pos:      protobuf.Position{Line: 10},
			},
		},
		Pos: protobuf.Position{Line: 5},
	}

	entities := idx.extractMessageEntities(1, msg, "", "user.proto")

	assert.Len(t, entities, 2) // 1 message + 1 field
	assert.Equal(t, "field", entities[1].EntityType)
	assert.True(t, entities[1].IsOptional)
	assert.False(t, entities[1].IsRepeated)
}

func TestExtractServiceEntitiesWithClientStreaming(t *testing.T) {
	idx := &Indexer{}

	service := &protobuf.ServiceNode{
		Name: "UserService",
		RPCs: []*protobuf.RPCNode{
			{
				Name:            "Upload",
				InputType:       "UploadRequest",
				OutputType:      "UploadResponse",
				ClientStreaming: true,
				Pos:             protobuf.Position{Line: 25},
			},
		},
		Pos: protobuf.Position{Line: 24},
	}

	entities := idx.extractServiceEntities(1, service, "example.v1", "service.proto")

	assert.Len(t, entities, 2)
	assert.Equal(t, "method", entities[1].EntityType)
	assert.Contains(t, entities[1].Metadata, "client_streaming")
	assert.True(t, entities[1].Metadata["client_streaming"].(bool))
}

func TestExtractServiceEntitiesNoStreaming(t *testing.T) {
	idx := &Indexer{}

	service := &protobuf.ServiceNode{
		Name: "UserService",
		RPCs: []*protobuf.RPCNode{
			{
				Name:            "GetUser",
				InputType:       "GetUserRequest",
				OutputType:      "User",
				ClientStreaming: false,
				ServerStreaming: false,
				Pos:             protobuf.Position{Line: 25},
			},
		},
		Pos: protobuf.Position{Line: 24},
	}

	entities := idx.extractServiceEntities(1, service, "", "service.proto")

	assert.Len(t, entities, 2)
	// Metadata should not contain streaming keys
	assert.NotContains(t, entities[1].Metadata, "client_streaming")
	assert.NotContains(t, entities[1].Metadata, "server_streaming")
}

func TestBatchInsertEntities(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{}
	idx := NewIndexer(db, storage)

	entities := []SearchEntity{
		{
			VersionID:     1,
			EntityType:    "message",
			EntityName:    "User",
			FullPath:      "test.User",
			ParentPath:    "test",
			ProtoFilePath: "test.proto",
			LineNumber:    5,
			Description:   "User message",
			Comments:      "User comments",
			Metadata:      make(map[string]interface{}),
		},
	}

	// Expect the insert query
	mock.ExpectExec("INSERT INTO proto_search_index").
		WithArgs(
			int64(1), "message", "User", "test.User", "test",
			"test.proto", 5, "User message", "User comments",
			nil, nil, false, false, nil, nil, sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = idx.batchInsertEntities(context.Background(), entities)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchInsertEntitiesEmpty(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{}
	idx := NewIndexer(db, storage)

	err = idx.batchInsertEntities(context.Background(), []SearchEntity{})
	assert.NoError(t, err)
}

func TestBatchInsertEntitiesMultipleBatches(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{}
	idx := NewIndexer(db, storage)

	// Create 150 entities to trigger multiple batches (batch size is 100)
	entities := make([]SearchEntity, 150)
	for i := 0; i < 150; i++ {
		entities[i] = SearchEntity{
			VersionID:  1,
			EntityType: "message",
			EntityName: "Message",
			FullPath:   "test.Message",
			Metadata:   make(map[string]interface{}),
		}
	}

	// Expect 2 batch inserts (100 + 50)
	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnResult(sqlmock.NewResult(1, 100))
	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnResult(sqlmock.NewResult(101, 50))

	err = idx.batchInsertEntities(context.Background(), entities)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchInsertEntitiesBatchError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{}
	idx := NewIndexer(db, storage)

	entities := make([]SearchEntity, 150)
	for i := 0; i < 150; i++ {
		entities[i] = SearchEntity{
			VersionID:  1,
			EntityType: "message",
			EntityName: "Message",
			FullPath:   "test.Message",
			Metadata:   make(map[string]interface{}),
		}
	}

	// First batch succeeds
	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnResult(sqlmock.NewResult(1, 100))

	// Second batch fails
	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnError(assert.AnError)

	err = idx.batchInsertEntities(context.Background(), entities)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert batch")
}

func TestInsertEntityBatchWithAllFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{}
	idx := NewIndexer(db, storage)

	entities := []SearchEntity{
		{
			VersionID:        1,
			EntityType:       "method",
			EntityName:       "GetUser",
			FullPath:         "test.UserService.GetUser",
			ParentPath:       "test.UserService",
			ProtoFilePath:    "service.proto",
			LineNumber:       10,
			Description:      "Get user method",
			Comments:         "Retrieves a user",
			FieldType:        "",
			FieldNumber:      0,
			IsRepeated:       false,
			IsOptional:       false,
			MethodInputType:  "GetUserRequest",
			MethodOutputType: "User",
			Metadata:         map[string]interface{}{"streaming": true},
		},
	}

	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = idx.insertEntityBatch(context.Background(), entities)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertEntityBatchEmpty(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{}
	idx := NewIndexer(db, storage)

	err = idx.insertEntityBatch(context.Background(), []SearchEntity{})
	assert.NoError(t, err)
}

func TestInsertEntityBatchDatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{}
	idx := NewIndexer(db, storage)

	entities := []SearchEntity{
		{
			VersionID:  1,
			EntityType: "message",
			EntityName: "User",
			FullPath:   "test.User",
			Metadata:   make(map[string]interface{}),
		},
	}

	// Expect the insert query to fail
	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnError(assert.AnError)

	err = idx.insertEntityBatch(context.Background(), entities)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert entities")
}

func TestIndexVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	protoContent := `
syntax = "proto3";
package test.v1;

message User {
  string id = 1;
  string name = 2;
}
`

	storage := &mockStorageReader{
		version: &Version{
			Version:    "v1.0.0",
			ModuleName: "test",
			Files: []FileInfo{
				{Path: "user.proto", Content: protoContent},
			},
		},
		file: &File{
			Path:    "user.proto",
			Content: []byte(protoContent),
		},
	}
	idx := NewIndexer(db, storage)

	// Expect version lookup
	rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
	mock.ExpectQuery("SELECT v.id FROM versions v").
		WithArgs("test", "v1.0.0").
		WillReturnRows(rows)

	// Expect delete of existing index entries
	mock.ExpectExec("DELETE FROM proto_search_index").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect insert of new entities
	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnResult(sqlmock.NewResult(1, 3)) // 1 message + 2 fields

	err = idx.IndexVersion(context.Background(), "test", "v1.0.0")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIndexVersionNoFiles(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{
		version: &Version{
			Version:    "v1.0.0",
			ModuleName: "test",
			Files:      []FileInfo{},
		},
	}
	idx := NewIndexer(db, storage)

	// Expect version lookup
	rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
	mock.ExpectQuery("SELECT v.id FROM versions v").
		WithArgs("test", "v1.0.0").
		WillReturnRows(rows)

	// Expect delete of existing index entries
	mock.ExpectExec("DELETE FROM proto_search_index").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// No insert expected since no files

	err = idx.IndexVersion(context.Background(), "test", "v1.0.0")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIndexVersionVersionNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{}
	idx := NewIndexer(db, storage)

	// Expect version lookup to fail
	mock.ExpectQuery("SELECT v.id FROM versions v").
		WithArgs("test", "v1.0.0").
		WillReturnError(sql.ErrNoRows)

	err = idx.IndexVersion(context.Background(), "test", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get version ID")
}

func TestIndexVersionGetVersionError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{
		err: assert.AnError,
	}
	idx := NewIndexer(db, storage)

	// Expect version lookup
	rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
	mock.ExpectQuery("SELECT v.id FROM versions v").
		WithArgs("test", "v1.0.0").
		WillReturnRows(rows)

	// Expect delete of existing index entries
	mock.ExpectExec("DELETE FROM proto_search_index").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = idx.IndexVersion(context.Background(), "test", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get version")
}

func TestIndexVersionWithInvalidProto(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	invalidProto := "this is not valid proto syntax"

	storage := &mockStorageReader{
		version: &Version{
			Version:    "v1.0.0",
			ModuleName: "test",
			Files: []FileInfo{
				{Path: "invalid.proto", Content: invalidProto},
			},
		},
		file: &File{
			Path:    "invalid.proto",
			Content: []byte(invalidProto),
		},
	}
	idx := NewIndexer(db, storage)

	// Expect version lookup
	rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
	mock.ExpectQuery("SELECT v.id FROM versions v").
		WithArgs("test", "v1.0.0").
		WillReturnRows(rows)

	// Expect delete of existing index entries
	mock.ExpectExec("DELETE FROM proto_search_index").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// No insert expected since proto is invalid
	// But error should not be returned - it should continue

	err = idx.IndexVersion(context.Background(), "test", "v1.0.0")
	assert.NoError(t, err) // Should not error, just skip the file
}

func TestIndexVersionDeleteIndexError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{}
	idx := NewIndexer(db, storage)

	// Expect version lookup
	rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
	mock.ExpectQuery("SELECT v.id FROM versions v").
		WithArgs("test", "v1.0.0").
		WillReturnRows(rows)

	// Expect delete to fail
	mock.ExpectExec("DELETE FROM proto_search_index").
		WithArgs(int64(1)).
		WillReturnError(assert.AnError)

	err = idx.IndexVersion(context.Background(), "test", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clear existing index")
}

func TestIndexVersionInsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	protoContent := `
syntax = "proto3";
package test.v1;

message User {
  string id = 1;
}
`

	storage := &mockStorageReader{
		version: &Version{
			Version:    "v1.0.0",
			ModuleName: "test",
			Files: []FileInfo{
				{Path: "user.proto", Content: protoContent},
			},
		},
		file: &File{
			Path:    "user.proto",
			Content: []byte(protoContent),
		},
	}
	idx := NewIndexer(db, storage)

	// Expect version lookup
	rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
	mock.ExpectQuery("SELECT v.id FROM versions v").
		WithArgs("test", "v1.0.0").
		WillReturnRows(rows)

	// Expect delete of existing index entries
	mock.ExpectExec("DELETE FROM proto_search_index").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect insert to fail
	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnError(assert.AnError)

	err = idx.IndexVersion(context.Background(), "test", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert entities")
}

func TestReindexAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	protoContent := `
syntax = "proto3";
package test.v1;

message User {
  string id = 1;
}
`

	storage := &mockStorageReader{
		modules: []*Module{
			{Name: "module1", Description: "Module 1"},
			{Name: "module2", Description: "Module 2"},
		},
		versions: map[string][]*Version{
			"module1": {
				{Version: "v1.0.0", ModuleName: "module1"},
			},
			"module2": {
				{Version: "v1.0.0", ModuleName: "module2"},
			},
		},
		version: &Version{
			Version:    "v1.0.0",
			ModuleName: "test",
			Files: []FileInfo{
				{Path: "user.proto", Content: protoContent},
			},
		},
		file: &File{
			Path:    "user.proto",
			Content: []byte(protoContent),
		},
	}
	idx := NewIndexer(db, storage)

	// Expect 2 version lookups (one for each module)
	rows1 := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
	mock.ExpectQuery("SELECT v.id FROM versions v").
		WithArgs("module1", "v1.0.0").
		WillReturnRows(rows1)
	mock.ExpectExec("DELETE FROM proto_search_index").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnResult(sqlmock.NewResult(1, 2))

	rows2 := sqlmock.NewRows([]string{"id"}).AddRow(int64(2))
	mock.ExpectQuery("SELECT v.id FROM versions v").
		WithArgs("module2", "v1.0.0").
		WillReturnRows(rows2)
	mock.ExpectExec("DELETE FROM proto_search_index").
		WithArgs(int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnResult(sqlmock.NewResult(3, 2))

	err = idx.ReindexAll(context.Background())
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReindexAllListModulesError(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{
		err: assert.AnError,
	}
	idx := NewIndexer(db, storage)

	err = idx.ReindexAll(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list modules")
}

func TestReindexAllEmptyModules(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorageReader{
		modules: []*Module{},
	}
	idx := NewIndexer(db, storage)

	err = idx.ReindexAll(context.Background())
	assert.NoError(t, err)
}

func TestReindexAllContinuesOnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create a storage that returns an error for ListVersions of first module
	// but succeeds for the second
	protoContent := `
syntax = "proto3";
package test.v1;

message User {
  string id = 1;
}
`

	storage := &mockStorageReader{
		modules: []*Module{
			{Name: "module1", Description: "Module 1"},
			{Name: "module2", Description: "Module 2"},
		},
		versions: map[string][]*Version{
			// module1 has no versions (will return empty list)
			"module2": {
				{Version: "v1.0.0", ModuleName: "module2"},
			},
		},
		version: &Version{
			Version:    "v1.0.0",
			ModuleName: "module2",
			Files: []FileInfo{
				{Path: "user.proto", Content: protoContent},
			},
		},
		file: &File{
			Path:    "user.proto",
			Content: []byte(protoContent),
		},
	}
	idx := NewIndexer(db, storage)

	// Only expect one index operation for module2
	rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(2))
	mock.ExpectQuery("SELECT v.id FROM versions v").
		WithArgs("module2", "v1.0.0").
		WillReturnRows(rows)
	mock.ExpectExec("DELETE FROM proto_search_index").
		WithArgs(int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO proto_search_index").
		WillReturnResult(sqlmock.NewResult(1, 2))

	err = idx.ReindexAll(context.Background())
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
