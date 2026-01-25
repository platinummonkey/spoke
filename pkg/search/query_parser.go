package search

import (
	"fmt"
	"regexp"
	"strings"
)

// ParsedQuery represents a parsed search query with filters
type ParsedQuery struct {
	// Free-text search terms
	Terms []string

	// Entity type filters: message, enum, service, method, field
	EntityTypes []string

	// Field type filters: string, int32, bool, etc.
	FieldTypes []string

	// Module name pattern (supports wildcards)
	ModulePattern string

	// Version constraint (e.g., ">=1.0.0", "~1.2.0")
	VersionConstraint string

	// Import filters (proto files)
	Imports []string

	// Dependency filters (module names)
	DependsOn []string

	// Has comment filter
	HasComment bool

	// Boolean operators between terms (AND, OR, NOT)
	Operators []string

	// Original query string
	Raw string
}

// QueryParser parses advanced search query syntax
type QueryParser struct {
	// Filter patterns
	filterPattern *regexp.Regexp
}

// NewQueryParser creates a new query parser
func NewQueryParser() *QueryParser {
	// Pattern to match filters: key:value or key:"quoted value"
	// Supports hyphens and underscores in key names
	filterPattern := regexp.MustCompile(`([\w-]+):("([^"]+)"|(\S+))`)

	return &QueryParser{
		filterPattern: filterPattern,
	}
}

// Parse parses a search query string into a ParsedQuery
func (p *QueryParser) Parse(queryStr string) (*ParsedQuery, error) {
	query := &ParsedQuery{
		Terms:       make([]string, 0),
		EntityTypes: make([]string, 0),
		FieldTypes:  make([]string, 0),
		Imports:     make([]string, 0),
		DependsOn:   make([]string, 0),
		Operators:   make([]string, 0),
		Raw:         queryStr,
	}

	// Find all filters
	matches := p.filterPattern.FindAllStringSubmatch(queryStr, -1)
	filterPositions := make(map[int]bool)

	for _, match := range matches {
		// Mark filter positions to exclude from free-text terms
		for i := 0; i < len(match[0]); i++ {
			idx := strings.Index(queryStr, match[0])
			if idx != -1 {
				filterPositions[idx+i] = true
			}
		}

		key := match[1]
		value := match[3] // Quoted value
		if value == "" {
			value = match[4] // Unquoted value
		}

		// Parse filter based on key
		if err := p.parseFilter(query, key, value); err != nil {
			return nil, err
		}
	}

	// Remove filters from query string to get free-text terms
	cleanQuery := p.filterPattern.ReplaceAllString(queryStr, "")
	cleanQuery = strings.TrimSpace(cleanQuery)

	// Split by whitespace to get terms
	if cleanQuery != "" {
		terms := strings.Fields(cleanQuery)
		for _, term := range terms {
			// Check for boolean operators
			upper := strings.ToUpper(term)
			if upper == "AND" || upper == "OR" || upper == "NOT" {
				query.Operators = append(query.Operators, upper)
			} else {
				query.Terms = append(query.Terms, term)
			}
		}
	}

	return query, nil
}

// parseFilter parses a single filter key-value pair
func (p *QueryParser) parseFilter(query *ParsedQuery, key, value string) error {
	switch strings.ToLower(key) {
	case "type":
		// Field type filter: type:string
		query.FieldTypes = append(query.FieldTypes, value)

	case "entity":
		// Entity type filter: entity:message
		validTypes := map[string]bool{
			"message":    true,
			"field":      true,
			"enum":       true,
			"enum_value": true,
			"service":    true,
			"method":     true,
		}
		if !validTypes[value] {
			return fmt.Errorf("invalid entity type: %s (must be one of: message, field, enum, enum_value, service, method)", value)
		}
		query.EntityTypes = append(query.EntityTypes, value)

	case "module":
		// Module name filter: module:user
		query.ModulePattern = value

	case "version":
		// Version constraint: version:>=1.0.0
		query.VersionConstraint = value

	case "imports":
		// Import filter: imports:common.proto
		query.Imports = append(query.Imports, value)

	case "depends-on", "depends_on":
		// Dependency filter: depends-on:common
		query.DependsOn = append(query.DependsOn, value)

	case "has-comment", "has_comment":
		// Has comment filter: has-comment:true
		query.HasComment = value == "true" || value == "1" || value == "yes"

	default:
		// Unknown filter - could add warning or ignore
		// For now, treat as a search term
		query.Terms = append(query.Terms, fmt.Sprintf("%s:%s", key, value))
	}

	return nil
}

// ToTsQuery converts parsed query to PostgreSQL tsquery format
// This generates a tsquery string that can be used with @@ operator
func (q *ParsedQuery) ToTsQuery() string {
	if len(q.Terms) == 0 {
		return ""
	}

	// Build tsquery with operators
	parts := make([]string, 0)
	defaultOp := "&" // AND by default

	for i, term := range q.Terms {
		// Sanitize term for tsquery
		sanitized := sanitizeTsQueryTerm(term)
		if sanitized == "" {
			continue
		}

		parts = append(parts, sanitized)

		// Add operator if there's a next term
		if i < len(q.Terms)-1 {
			op := defaultOp
			if i < len(q.Operators) {
				switch q.Operators[i] {
				case "OR":
					op = "|"
				case "NOT":
					op = "&!"
				case "AND":
					op = "&"
				}
			}
			parts = append(parts, op)
		}
	}

	return strings.Join(parts, " ")
}

// sanitizeTsQueryTerm sanitizes a search term for use in tsquery
func sanitizeTsQueryTerm(term string) string {
	// Remove special characters that have meaning in tsquery
	term = strings.TrimSpace(term)
	if term == "" {
		return ""
	}

	// Escape single quotes
	term = strings.ReplaceAll(term, "'", "''")

	// Remove tsquery operators if they appear standalone
	if term == "&" || term == "|" || term == "!" || term == "<->" {
		return ""
	}

	// Add wildcard suffix for prefix matching
	// This allows "user" to match "users", "username", etc.
	if !strings.HasSuffix(term, ":*") {
		term = term + ":*"
	}

	return term
}

// HasFilters returns true if the query has any filters
func (q *ParsedQuery) HasFilters() bool {
	return len(q.EntityTypes) > 0 ||
		len(q.FieldTypes) > 0 ||
		q.ModulePattern != "" ||
		q.VersionConstraint != "" ||
		len(q.Imports) > 0 ||
		len(q.DependsOn) > 0 ||
		q.HasComment
}

// String returns a human-readable representation of the query
func (q *ParsedQuery) String() string {
	parts := make([]string, 0)

	if len(q.Terms) > 0 {
		parts = append(parts, fmt.Sprintf("terms:%v", q.Terms))
	}
	if len(q.EntityTypes) > 0 {
		parts = append(parts, fmt.Sprintf("entity:%v", q.EntityTypes))
	}
	if len(q.FieldTypes) > 0 {
		parts = append(parts, fmt.Sprintf("type:%v", q.FieldTypes))
	}
	if q.ModulePattern != "" {
		parts = append(parts, fmt.Sprintf("module:%s", q.ModulePattern))
	}
	if q.VersionConstraint != "" {
		parts = append(parts, fmt.Sprintf("version:%s", q.VersionConstraint))
	}
	if len(q.Imports) > 0 {
		parts = append(parts, fmt.Sprintf("imports:%v", q.Imports))
	}
	if len(q.DependsOn) > 0 {
		parts = append(parts, fmt.Sprintf("depends-on:%v", q.DependsOn))
	}
	if q.HasComment {
		parts = append(parts, "has-comment:true")
	}

	return strings.Join(parts, ", ")
}

// Examples for documentation:
//
// Basic search:
//   "user" -> searches for "user" in all fields
//
// Entity type filter:
//   "email entity:field" -> searches for fields named "email"
//   "Status entity:enum" -> searches for enums named "Status"
//
// Field type filter:
//   "user type:string" -> searches for "user" in string fields
//   "id type:int32" -> searches for "id" in int32 fields
//
// Module filter:
//   "email module:user" -> searches for "email" in user module
//   "Status module:common.*" -> searches in modules starting with "common"
//
// Version constraint:
//   "CreateUser version:>=1.0.0" -> searches in versions >= 1.0.0
//
// Dependency filter:
//   "User depends-on:common" -> searches in modules that depend on common
//
// Comment filter:
//   "deprecated has-comment:true" -> searches entities with comments containing "deprecated"
//
// Boolean operators:
//   "user OR email" -> searches for "user" OR "email"
//   "user AND active" -> searches for documents containing both terms
//   "user NOT deleted" -> searches for "user" but not "deleted"
//
// Combined filters:
//   "email entity:field type:string module:user" -> very specific search
