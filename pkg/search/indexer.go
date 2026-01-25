package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var indexerTracer = otel.Tracer("spoke/search/indexer")

// StorageReader defines the minimal interface needed for indexing
// This avoids import cycles with pkg/api
type StorageReader interface {
	GetVersion(moduleName, version string) (*Version, error)
	GetFile(moduleName, version, path string) (*File, error)
	ListModules() ([]*Module, error)
	ListVersions(moduleName string) ([]*Version, error)
}

// Module represents a minimal module for indexing
type Module struct {
	Name        string
	Description string
}

// Version represents a minimal version for indexing
type Version struct {
	Version      string
	ModuleName   string
	Files        []FileInfo
	Dependencies []string
}

// FileInfo represents minimal file info for indexing
type FileInfo struct {
	Path    string
	Content string
}

// File represents a minimal file for indexing
type File struct {
	Path    string
	Content []byte
}

// Indexer indexes proto entities for full-text search
type Indexer struct {
	db      *sql.DB
	storage StorageReader
}

// NewIndexer creates a new search indexer
func NewIndexer(db *sql.DB, storage StorageReader) *Indexer {
	return &Indexer{
		db:      db,
		storage: storage,
	}
}

// SearchEntity represents a searchable proto entity
type SearchEntity struct {
	VersionID        int64
	EntityType       string // 'message', 'enum', 'service', 'method', 'field'
	EntityName       string
	FullPath         string
	ParentPath       string
	ProtoFilePath    string
	LineNumber       int
	Description      string
	Comments         string
	FieldType        string // For fields
	FieldNumber      int    // For fields
	IsRepeated       bool
	IsOptional       bool
	MethodInputType  string // For methods
	MethodOutputType string // For methods
	Metadata         map[string]interface{}
}

// IndexVersion indexes all proto entities in a version
func (idx *Indexer) IndexVersion(ctx context.Context, moduleName, version string) error {
	ctx, span := indexerTracer.Start(ctx, "IndexVersion",
		trace.WithAttributes(
			attribute.String("module", moduleName),
			attribute.String("version", version),
		),
	)
	defer span.End()

	// Get version ID from database
	var versionID int64
	err := idx.db.QueryRowContext(ctx, `
		SELECT v.id
		FROM versions v
		JOIN modules m ON v.module_id = m.id
		WHERE m.name = $1 AND v.version = $2
	`, moduleName, version).Scan(&versionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get version ID")
		return fmt.Errorf("failed to get version ID: %w", err)
	}

	// Delete existing index entries for this version
	_, err = idx.db.ExecContext(ctx, `
		DELETE FROM proto_search_index WHERE version_id = $1
	`, versionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to clear existing index")
		return fmt.Errorf("failed to clear existing index: %w", err)
	}

	// Get version with files
	ver, err := idx.storage.GetVersion(moduleName, version)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get version")
		return fmt.Errorf("failed to get version: %w", err)
	}

	if len(ver.Files) == 0 {
		// No files to index
		span.SetStatus(codes.Ok, "no files to index")
		return nil
	}

	// Extract entities from each proto file
	var allEntities []SearchEntity
	for _, file := range ver.Files {
		// Get file content
		fileContent, err := idx.storage.GetFile(moduleName, version, file.Path)
		if err != nil {
			// Log error but continue with other files
			span.AddEvent("failed to read file",
				trace.WithAttributes(attribute.String("file", file.Path)),
			)
			continue
		}

		// Parse proto file
		parser := protobuf.NewStringParser(string(fileContent.Content))
		ast, err := parser.Parse()
		if err != nil {
			// Log error but continue with other files
			span.AddEvent("failed to parse file",
				trace.WithAttributes(
					attribute.String("file", file.Path),
					attribute.String("error", err.Error()),
				),
			)
			continue
		}

		// Extract entities from AST
		entities := idx.extractEntities(ctx, versionID, ast, file.Path)
		allEntities = append(allEntities, entities...)
	}

	// Batch insert entities
	if len(allEntities) > 0 {
		err = idx.batchInsertEntities(ctx, allEntities)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to insert entities")
			return fmt.Errorf("failed to insert entities: %w", err)
		}
	}

	span.SetStatus(codes.Ok, fmt.Sprintf("indexed %d entities", len(allEntities)))
	return nil
}

// extractEntities extracts searchable entities from a proto AST
func (idx *Indexer) extractEntities(ctx context.Context, versionID int64, ast *protobuf.RootNode, filePath string) []SearchEntity {
	var entities []SearchEntity

	// Get package name for building full paths
	packageName := ""
	if ast.Package != nil {
		packageName = ast.Package.Name
	}

	// Extract messages
	for _, msg := range ast.Messages {
		entities = append(entities, idx.extractMessageEntities(versionID, msg, packageName, filePath)...)
	}

	// Extract enums
	for _, enum := range ast.Enums {
		entities = append(entities, idx.extractEnumEntities(versionID, enum, packageName, filePath)...)
	}

	// Extract services
	for _, service := range ast.Services {
		entities = append(entities, idx.extractServiceEntities(versionID, service, packageName, filePath)...)
	}

	return entities
}

// extractMessageEntities extracts entities from a message node
func (idx *Indexer) extractMessageEntities(versionID int64, msg *protobuf.MessageNode, packageName, filePath string) []SearchEntity {
	var entities []SearchEntity

	// Build full path for message
	fullPath := msg.Name
	if packageName != "" {
		fullPath = packageName + "." + msg.Name
	}

	// Extract comments
	comments := idx.extractComments(msg.Comments)
	description := idx.extractDescription(msg.Comments)

	// Create message entity
	entities = append(entities, SearchEntity{
		VersionID:     versionID,
		EntityType:    "message",
		EntityName:    msg.Name,
		FullPath:      fullPath,
		ParentPath:    packageName,
		ProtoFilePath: filePath,
		LineNumber:    msg.Pos.Line,
		Description:   description,
		Comments:      comments,
		Metadata:      make(map[string]interface{}),
	})

	// Extract fields
	for _, field := range msg.Fields {
		fieldEntity := SearchEntity{
			VersionID:     versionID,
			EntityType:    "field",
			EntityName:    field.Name,
			FullPath:      fullPath + "." + field.Name,
			ParentPath:    fullPath,
			ProtoFilePath: filePath,
			LineNumber:    field.Pos.Line,
			FieldType:     field.Type,
			FieldNumber:   field.Number,
			IsRepeated:    field.Repeated,
			IsOptional:    field.Optional,
			Comments:      idx.extractComments(field.Comments),
			Description:   idx.extractDescription(field.Comments),
			Metadata:      make(map[string]interface{}),
		}
		entities = append(entities, fieldEntity)
	}

	// Extract nested messages
	for _, nested := range msg.Nested {
		nestedEntities := idx.extractMessageEntities(versionID, nested, fullPath, filePath)
		entities = append(entities, nestedEntities...)
	}

	// Extract nested enums
	for _, enum := range msg.Enums {
		enumEntities := idx.extractEnumEntities(versionID, enum, fullPath, filePath)
		entities = append(entities, enumEntities...)
	}

	return entities
}

// extractEnumEntities extracts entities from an enum node
func (idx *Indexer) extractEnumEntities(versionID int64, enum *protobuf.EnumNode, packageName, filePath string) []SearchEntity {
	var entities []SearchEntity

	// Build full path for enum
	fullPath := enum.Name
	if packageName != "" {
		fullPath = packageName + "." + enum.Name
	}

	// Extract comments
	comments := idx.extractComments(enum.Comments)
	description := idx.extractDescription(enum.Comments)

	// Create enum entity
	entities = append(entities, SearchEntity{
		VersionID:     versionID,
		EntityType:    "enum",
		EntityName:    enum.Name,
		FullPath:      fullPath,
		ParentPath:    packageName,
		ProtoFilePath: filePath,
		LineNumber:    enum.Pos.Line,
		Description:   description,
		Comments:      comments,
		Metadata:      make(map[string]interface{}),
	})

	// Extract enum values
	for _, value := range enum.Values {
		valueEntity := SearchEntity{
			VersionID:     versionID,
			EntityType:    "enum_value",
			EntityName:    value.Name,
			FullPath:      fullPath + "." + value.Name,
			ParentPath:    fullPath,
			ProtoFilePath: filePath,
			LineNumber:    value.Pos.Line,
			FieldNumber:   value.Number,
			Comments:      idx.extractComments(value.Comments),
			Description:   idx.extractDescription(value.Comments),
			Metadata:      make(map[string]interface{}),
		}
		entities = append(entities, valueEntity)
	}

	return entities
}

// extractServiceEntities extracts entities from a service node
func (idx *Indexer) extractServiceEntities(versionID int64, service *protobuf.ServiceNode, packageName, filePath string) []SearchEntity {
	var entities []SearchEntity

	// Build full path for service
	fullPath := service.Name
	if packageName != "" {
		fullPath = packageName + "." + service.Name
	}

	// Extract comments
	comments := idx.extractComments(service.Comments)
	description := idx.extractDescription(service.Comments)

	// Create service entity
	entities = append(entities, SearchEntity{
		VersionID:     versionID,
		EntityType:    "service",
		EntityName:    service.Name,
		FullPath:      fullPath,
		ParentPath:    packageName,
		ProtoFilePath: filePath,
		LineNumber:    service.Pos.Line,
		Description:   description,
		Comments:      comments,
		Metadata:      make(map[string]interface{}),
	})

	// Extract RPC methods
	for _, rpc := range service.RPCs {
		methodEntity := SearchEntity{
			VersionID:        versionID,
			EntityType:       "method",
			EntityName:       rpc.Name,
			FullPath:         fullPath + "." + rpc.Name,
			ParentPath:       fullPath,
			ProtoFilePath:    filePath,
			LineNumber:       rpc.Pos.Line,
			MethodInputType:  rpc.InputType,
			MethodOutputType: rpc.OutputType,
			Comments:         idx.extractComments(rpc.Comments),
			Description:      idx.extractDescription(rpc.Comments),
			Metadata:         make(map[string]interface{}),
		}

		// Add streaming metadata
		if rpc.ClientStreaming || rpc.ServerStreaming {
			methodEntity.Metadata["client_streaming"] = rpc.ClientStreaming
			methodEntity.Metadata["server_streaming"] = rpc.ServerStreaming
		}

		entities = append(entities, methodEntity)
	}

	return entities
}

// extractComments extracts all comment text from comment nodes
func (idx *Indexer) extractComments(comments []*protobuf.CommentNode) string {
	if len(comments) == 0 {
		return ""
	}

	var parts []string
	for _, comment := range comments {
		if comment.Text != "" {
			parts = append(parts, strings.TrimSpace(comment.Text))
		}
	}

	return strings.Join(parts, " ")
}

// extractDescription extracts the first line of comments as description
func (idx *Indexer) extractDescription(comments []*protobuf.CommentNode) string {
	if len(comments) == 0 {
		return ""
	}

	// Use the first comment as description
	firstComment := comments[0].Text
	lines := strings.Split(firstComment, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}

	return strings.TrimSpace(firstComment)
}

// batchInsertEntities inserts entities in batches
func (idx *Indexer) batchInsertEntities(ctx context.Context, entities []SearchEntity) error {
	// Batch size for insertion
	const batchSize = 100

	for i := 0; i < len(entities); i += batchSize {
		end := i + batchSize
		if end > len(entities) {
			end = len(entities)
		}

		batch := entities[i:end]
		err := idx.insertEntityBatch(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to insert batch at offset %d: %w", i, err)
		}
	}

	return nil
}

// insertEntityBatch inserts a batch of entities
func (idx *Indexer) insertEntityBatch(ctx context.Context, entities []SearchEntity) error {
	if len(entities) == 0 {
		return nil
	}

	// Build multi-row insert query
	query := `
		INSERT INTO proto_search_index (
			version_id, entity_type, entity_name, full_path, parent_path,
			proto_file_path, line_number, description, comments,
			field_type, field_number, is_repeated, is_optional,
			method_input_type, method_output_type, metadata
		) VALUES
	`

	var values []interface{}
	var placeholders []string

	for i, entity := range entities {
		offset := i * 16
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7, offset+8,
			offset+9, offset+10, offset+11, offset+12, offset+13, offset+14, offset+15, offset+16,
		))

		// Serialize metadata to JSON
		metadataJSON, err := json.Marshal(entity.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		values = append(values,
			entity.VersionID,
			entity.EntityType,
			entity.EntityName,
			entity.FullPath,
			nullString(entity.ParentPath),
			nullString(entity.ProtoFilePath),
			nullInt(entity.LineNumber),
			nullString(entity.Description),
			nullString(entity.Comments),
			nullString(entity.FieldType),
			nullInt(entity.FieldNumber),
			entity.IsRepeated,
			entity.IsOptional,
			nullString(entity.MethodInputType),
			nullString(entity.MethodOutputType),
			string(metadataJSON),
		)
	}

	query += strings.Join(placeholders, ", ")

	_, err := idx.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to insert entities: %w", err)
	}

	return nil
}

// ReindexAll re-indexes all versions in the registry
func (idx *Indexer) ReindexAll(ctx context.Context) error {
	ctx, span := indexerTracer.Start(ctx, "ReindexAll")
	defer span.End()

	// Get all modules
	modules, err := idx.storage.ListModules()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to list modules")
		return fmt.Errorf("failed to list modules: %w", err)
	}

	totalVersions := 0
	for _, module := range modules {
		versions, err := idx.storage.ListVersions(module.Name)
		if err != nil {
			span.AddEvent("failed to list versions",
				trace.WithAttributes(attribute.String("module", module.Name)),
			)
			continue
		}

		for _, version := range versions {
			err = idx.IndexVersion(ctx, module.Name, version.Version)
			if err != nil {
				span.AddEvent("failed to index version",
					trace.WithAttributes(
						attribute.String("module", module.Name),
						attribute.String("version", version.Version),
						attribute.String("error", err.Error()),
					),
				)
				continue
			}
			totalVersions++
		}
	}

	span.SetStatus(codes.Ok, fmt.Sprintf("reindexed %d versions", totalVersions))
	return nil
}

// Helper functions

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullInt(i int) interface{} {
	if i == 0 {
		return nil
	}
	return i
}
