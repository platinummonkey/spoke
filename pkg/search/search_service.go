package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var searchTracer = otel.Tracer("spoke/search/service")

// SearchService provides advanced search capabilities using PostgreSQL FTS
type SearchService struct {
	db     *sql.DB
	parser *QueryParser
}

// NewSearchService creates a new search service
func NewSearchService(db *sql.DB) *SearchService {
	return &SearchService{
		db:     db,
		parser: NewQueryParser(),
	}
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query  string // Raw query string with filters
	Limit  int    // Max results (default: 50)
	Offset int    // Pagination offset (default: 0)
}

// SearchResponse represents search results
type SearchResponse struct {
	Results    []EntitySearchResult `json:"results"`
	TotalCount int            `json:"total_count"`
	Query      string         `json:"query"`
	ParsedQuery *ParsedQuery  `json:"parsed_query,omitempty"`
}

// EntitySearchResult represents a single search result
type EntitySearchResult struct {
	// Entity identification
	ID          int64  `json:"id"`
	EntityType  string `json:"entity_type"`
	EntityName  string `json:"entity_name"`
	FullPath    string `json:"full_path"`
	ParentPath  string `json:"parent_path,omitempty"`

	// Module/version info
	ModuleName string `json:"module_name"`
	Version    string `json:"version"`

	// Proto file context
	ProtoFilePath string `json:"proto_file_path,omitempty"`
	LineNumber    int    `json:"line_number,omitempty"`

	// Content
	Description string `json:"description,omitempty"`
	Comments    string `json:"comments,omitempty"`

	// Field-specific
	FieldType   string `json:"field_type,omitempty"`
	FieldNumber int    `json:"field_number,omitempty"`
	IsRepeated  bool   `json:"is_repeated,omitempty"`
	IsOptional  bool   `json:"is_optional,omitempty"`

	// Method-specific
	MethodInputType  string `json:"method_input_type,omitempty"`
	MethodOutputType string `json:"method_output_type,omitempty"`

	// Search relevance
	Rank float64 `json:"rank"` // PostgreSQL ts_rank score

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Search performs advanced search with filters
func (s *SearchService) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	ctx, span := searchTracer.Start(ctx, "Search",
		trace.WithAttributes(
			attribute.String("query", req.Query),
			attribute.Int("limit", req.Limit),
			attribute.Int("offset", req.Offset),
		),
	)
	defer span.End()

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 1000 {
		req.Limit = 1000 // Cap at 1000 results
	}

	// Parse query
	parsedQuery, err := s.parser.Parse(req.Query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse query")
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	span.SetAttributes(
		attribute.Bool("has_filters", parsedQuery.HasFilters()),
		attribute.Int("term_count", len(parsedQuery.Terms)),
	)

	// Build SQL query
	query, args := s.buildSearchQuery(parsedQuery, req.Limit, req.Offset)

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to execute search")
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer rows.Close()

	// Parse results
	results := make([]EntitySearchResult, 0, req.Limit)
	for rows.Next() {
		var result EntitySearchResult
		var metadataJSON []byte

		err := rows.Scan(
			&result.ID,
			&result.EntityType,
			&result.EntityName,
			&result.FullPath,
			&result.ParentPath,
			&result.ModuleName,
			&result.Version,
			&result.ProtoFilePath,
			&result.LineNumber,
			&result.Description,
			&result.Comments,
			&result.FieldType,
			&result.FieldNumber,
			&result.IsRepeated,
			&result.IsOptional,
			&result.MethodInputType,
			&result.MethodOutputType,
			&result.Rank,
			&metadataJSON,
		)
		if err != nil {
			span.RecordError(err)
			continue // Skip invalid rows
		}

		// TODO: Parse metadata JSON if needed
		// For now, leave it nil

		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error iterating results")
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	// Get total count (without pagination)
	totalCount, err := s.getTotalCount(ctx, parsedQuery)
	if err != nil {
		// Log error but don't fail the request
		span.AddEvent("failed to get total count",
			trace.WithAttributes(attribute.String("error", err.Error())),
		)
		totalCount = len(results) // Fallback to result count
	}

	span.SetAttributes(
		attribute.Int("result_count", len(results)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "search completed")

	return &SearchResponse{
		Results:     results,
		TotalCount:  totalCount,
		Query:       req.Query,
		ParsedQuery: parsedQuery,
	}, nil
}

// buildSearchQuery builds a PostgreSQL query from a parsed query
func (s *SearchService) buildSearchQuery(q *ParsedQuery, limit, offset int) (string, []interface{}) {
	// Base query with joins
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT
			psi.id,
			psi.entity_type,
			psi.entity_name,
			psi.full_path,
			psi.parent_path,
			m.name as module_name,
			v.version,
			psi.proto_file_path,
			psi.line_number,
			psi.description,
			psi.comments,
			psi.field_type,
			psi.field_number,
			psi.is_repeated,
			psi.is_optional,
			psi.method_input_type,
			psi.method_output_type,
	`)

	// Add ranking if we have search terms
	if len(q.Terms) > 0 {
		queryBuilder.WriteString(`
			ts_rank(psi.search_vector, to_tsquery('english', $1)) as rank,
		`)
	} else {
		queryBuilder.WriteString(`
			0.0 as rank,
		`)
	}

	queryBuilder.WriteString(`
			psi.metadata
		FROM proto_search_index psi
		JOIN versions v ON psi.version_id = v.id
		JOIN modules m ON v.module_id = m.id
		WHERE 1=1
	`)

	args := make([]interface{}, 0)
	argIndex := 1

	// Add full-text search condition
	if len(q.Terms) > 0 {
		tsquery := q.ToTsQuery()
		args = append(args, tsquery)
		queryBuilder.WriteString(fmt.Sprintf(`
			AND psi.search_vector @@ to_tsquery('english', $%d)
		`, argIndex))
		argIndex++
	}

	// Add entity type filter
	if len(q.EntityTypes) > 0 {
		placeholders := make([]string, len(q.EntityTypes))
		for i, entityType := range q.EntityTypes {
			args = append(args, entityType)
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			argIndex++
		}
		queryBuilder.WriteString(fmt.Sprintf(`
			AND psi.entity_type IN (%s)
		`, strings.Join(placeholders, ", ")))
	}

	// Add field type filter
	if len(q.FieldTypes) > 0 {
		placeholders := make([]string, len(q.FieldTypes))
		for i, fieldType := range q.FieldTypes {
			args = append(args, fieldType)
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			argIndex++
		}
		queryBuilder.WriteString(fmt.Sprintf(`
			AND psi.field_type IN (%s)
		`, strings.Join(placeholders, ", ")))
	}

	// Add module pattern filter
	if q.ModulePattern != "" {
		args = append(args, q.ModulePattern)
		if strings.Contains(q.ModulePattern, "*") || strings.Contains(q.ModulePattern, "%") {
			// Wildcard pattern
			pattern := strings.ReplaceAll(q.ModulePattern, "*", "%")
			queryBuilder.WriteString(fmt.Sprintf(`
				AND m.name LIKE $%d
			`, argIndex))
			args[len(args)-1] = pattern
		} else {
			// Exact match
			queryBuilder.WriteString(fmt.Sprintf(`
				AND m.name = $%d
			`, argIndex))
		}
		argIndex++
	}

	// Add version constraint filter
	// TODO: Implement semantic version constraint parsing (>=1.0.0, ~1.2.0)
	// For now, use exact match
	if q.VersionConstraint != "" {
		args = append(args, q.VersionConstraint)
		queryBuilder.WriteString(fmt.Sprintf(`
			AND v.version = $%d
		`, argIndex))
	}

	// Add has-comment filter
	if q.HasComment {
		queryBuilder.WriteString(`
			AND (psi.comments IS NOT NULL AND psi.comments != '')
		`)
	}

	// TODO: Add import and depends-on filters
	// These require joins to versions.dependencies JSONB array

	// Add ordering
	if len(q.Terms) > 0 {
		queryBuilder.WriteString(`
			ORDER BY rank DESC, psi.entity_name ASC
		`)
	} else {
		queryBuilder.WriteString(`
			ORDER BY psi.entity_name ASC, m.name ASC
		`)
	}

	// Add pagination
	args = append(args, limit, offset)
	queryBuilder.WriteString(fmt.Sprintf(`
		LIMIT $%d OFFSET $%d
	`, argIndex, argIndex+1))

	return queryBuilder.String(), args
}

// getTotalCount gets the total count of results without pagination
func (s *SearchService) getTotalCount(ctx context.Context, q *ParsedQuery) (int, error) {
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT COUNT(*)
		FROM proto_search_index psi
		JOIN versions v ON psi.version_id = v.id
		JOIN modules m ON v.module_id = m.id
		WHERE 1=1
	`)

	args := make([]interface{}, 0)
	argIndex := 1

	// Add full-text search condition
	if len(q.Terms) > 0 {
		tsquery := q.ToTsQuery()
		args = append(args, tsquery)
		queryBuilder.WriteString(fmt.Sprintf(`
			AND psi.search_vector @@ to_tsquery('english', $%d)
		`, argIndex))
		argIndex++
	}

	// Add entity type filter
	if len(q.EntityTypes) > 0 {
		placeholders := make([]string, len(q.EntityTypes))
		for i, entityType := range q.EntityTypes {
			args = append(args, entityType)
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			argIndex++
		}
		queryBuilder.WriteString(fmt.Sprintf(`
			AND psi.entity_type IN (%s)
		`, strings.Join(placeholders, ", ")))
	}

	// Add field type filter
	if len(q.FieldTypes) > 0 {
		placeholders := make([]string, len(q.FieldTypes))
		for i, fieldType := range q.FieldTypes {
			args = append(args, fieldType)
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			argIndex++
		}
		queryBuilder.WriteString(fmt.Sprintf(`
			AND psi.field_type IN (%s)
		`, strings.Join(placeholders, ", ")))
	}

	// Add module pattern filter
	if q.ModulePattern != "" {
		args = append(args, q.ModulePattern)
		if strings.Contains(q.ModulePattern, "*") || strings.Contains(q.ModulePattern, "%") {
			// Wildcard pattern
			pattern := strings.ReplaceAll(q.ModulePattern, "*", "%")
			queryBuilder.WriteString(fmt.Sprintf(`
				AND m.name LIKE $%d
			`, argIndex))
			args[len(args)-1] = pattern
		} else {
			// Exact match
			queryBuilder.WriteString(fmt.Sprintf(`
				AND m.name = $%d
			`, argIndex))
		}
		argIndex++
	}

	// Add version constraint filter
	if q.VersionConstraint != "" {
		args = append(args, q.VersionConstraint)
		queryBuilder.WriteString(fmt.Sprintf(`
			AND v.version = $%d
		`, argIndex))
	}

	// Add has-comment filter
	if q.HasComment {
		queryBuilder.WriteString(`
			AND (psi.comments IS NOT NULL AND psi.comments != '')
		`)
	}

	var count int
	err := s.db.QueryRowContext(ctx, queryBuilder.String(), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total count: %w", err)
	}

	return count, nil
}

// GetSuggestions returns search query suggestions based on history
func (s *SearchService) GetSuggestions(ctx context.Context, prefix string, limit int) ([]string, error) {
	ctx, span := searchTracer.Start(ctx, "GetSuggestions",
		trace.WithAttributes(
			attribute.String("prefix", prefix),
			attribute.Int("limit", limit),
		),
	)
	defer span.End()

	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	// Query search_suggestions materialized view
	query := `
		SELECT query
		FROM search_suggestions
		WHERE query LIKE $1
		ORDER BY search_count DESC, last_searched_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, prefix+"%", limit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get suggestions")
		return nil, fmt.Errorf("failed to get suggestions: %w", err)
	}
	defer rows.Close()

	suggestions := make([]string, 0, limit)
	for rows.Next() {
		var suggestion string
		if err := rows.Scan(&suggestion); err != nil {
			continue
		}
		suggestions = append(suggestions, suggestion)
	}

	span.SetAttributes(attribute.Int("suggestion_count", len(suggestions)))
	span.SetStatus(codes.Ok, "suggestions retrieved")

	return suggestions, nil
}

// RecordSearch records a search query in history for suggestions
func (s *SearchService) RecordSearch(ctx context.Context, query string, resultCount int, durationMs int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO search_history (query, result_count, search_duration_ms, created_at)
		VALUES ($1, $2, $3, NOW())
	`, query, resultCount, durationMs)

	return err
}
