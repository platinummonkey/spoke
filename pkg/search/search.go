package search

import (
	"strings"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// SearchEngine provides search capabilities across schemas
type SearchEngine struct {
	storage StorageReader
}

// NewSearchEngine creates a new search engine
func NewSearchEngine(storage StorageReader) *SearchEngine {
	return &SearchEngine{
		storage: storage,
	}
}

// SearchQuery represents a search query
type SearchQuery struct {
	Query      string   `json:"query"`       // Free-text search
	Type       string   `json:"type"`        // message, field, service, enum, package
	Module     string   `json:"module"`      // Filter by module name
	Version    string   `json:"version"`     // Filter by version
	Tags       []string `json:"tags"`        // Filter by tags
	Deprecated bool     `json:"deprecated"`  // Include deprecated items
	Limit      int      `json:"limit"`       // Max results
}

// SearchResult represents a search result
type SearchResult struct {
	Type        string            `json:"type"`         // message, field, service, etc.
	Module      string            `json:"module"`       // Module name
	Version     string            `json:"version"`      // Version
	Name        string            `json:"name"`         // Item name
	FullName    string            `json:"full_name"`    // Fully qualified name
	Description string            `json:"description"`  // Description/comments
	Deprecated  bool              `json:"deprecated"`   // Is deprecated
	Score       float64           `json:"score"`        // Relevance score
	Metadata    map[string]string `json:"metadata"`     // Additional metadata
}

// SearchResults contains search results and metadata
type SearchResults struct {
	Results    []SearchResult `json:"results"`
	TotalCount int            `json:"total_count"`
	Query      string         `json:"query"`
}

// Search performs a search across all schemas
func (s *SearchEngine) Search(query SearchQuery) (*SearchResults, error) {
	results := &SearchResults{
		Results: make([]SearchResult, 0),
		Query:   query.Query,
	}

	// Get all modules
	modules, err := s.storage.ListModules()
	if err != nil {
		return nil, err
	}

	// Filter modules if specified
	if query.Module != "" {
		filtered := make([]*Module, 0)
		for _, mod := range modules {
			if matchesPattern(mod.Name, query.Module) {
				filtered = append(filtered, mod)
			}
		}
		modules = filtered
	}

	// Search each module
	for _, module := range modules {
		// Get versions for module
		versions, err := s.storage.ListVersions(module.Name)
		if err != nil {
			continue // Skip modules with errors
		}

		if query.Version != "" {
			filtered := make([]*Version, 0)
			for _, ver := range versions {
				if matchesPattern(ver.Version, query.Version) {
					filtered = append(filtered, ver)
				}
			}
			versions = filtered
		}

		// Search each version
		for _, version := range versions {
			if len(version.Files) == 0 {
				continue
			}

			// Parse proto file
			parser := protobuf.NewStringParser(version.Files[0].Content)
			ast, err := parser.Parse()
			if err != nil {
				continue // Skip files that don't parse
			}

			// Search within AST
			s.searchAST(ast, module.Name, version.Version, query, results)
		}
	}

	// Limit results
	if query.Limit > 0 && len(results.Results) > query.Limit {
		results.Results = results.Results[:query.Limit]
	}

	results.TotalCount = len(results.Results)
	return results, nil
}

func (s *SearchEngine) searchAST(ast *protobuf.RootNode, moduleName, version string, query SearchQuery, results *SearchResults) {
	queryLower := strings.ToLower(query.Query)

	// Search package
	if ast.Package != nil && (query.Type == "" || query.Type == "package") {
		if matchesSearch(ast.Package.Name, queryLower) {
			results.Results = append(results.Results, SearchResult{
				Type:     "package",
				Module:   moduleName,
				Version:  version,
				Name:     ast.Package.Name,
				FullName: ast.Package.Name,
				Score:    calculateScore(ast.Package.Name, queryLower),
			})
		}
	}

	// Search messages
	if query.Type == "" || query.Type == "message" {
		for _, msg := range ast.Messages {
			s.searchMessage(msg, moduleName, version, queryLower, results)
		}
	}

	// Search services
	if query.Type == "" || query.Type == "service" {
		for _, svc := range ast.Services {
			if matchesSearch(svc.Name, queryLower) {
				results.Results = append(results.Results, SearchResult{
					Type:     "service",
					Module:   moduleName,
					Version:  version,
					Name:     svc.Name,
					FullName: svc.Name,
					Score:    calculateScore(svc.Name, queryLower),
				})
			}

			// Search RPC methods
			if query.Type == "" || query.Type == "method" {
				for _, rpc := range svc.RPCs {
					if matchesSearch(rpc.Name, queryLower) {
						results.Results = append(results.Results, SearchResult{
							Type:     "method",
							Module:   moduleName,
							Version:  version,
							Name:     rpc.Name,
							FullName: svc.Name + "." + rpc.Name,
							Score:    calculateScore(rpc.Name, queryLower),
							Metadata: map[string]string{
								"service": svc.Name,
								"input":   rpc.InputType,
								"output":  rpc.OutputType,
							},
						})
					}
				}
			}
		}
	}

	// Search enums
	if query.Type == "" || query.Type == "enum" {
		for _, enum := range ast.Enums {
			if matchesSearch(enum.Name, queryLower) {
				results.Results = append(results.Results, SearchResult{
					Type:     "enum",
					Module:   moduleName,
					Version:  version,
					Name:     enum.Name,
					FullName: enum.Name,
					Score:    calculateScore(enum.Name, queryLower),
				})
			}
		}
	}
}

func (s *SearchEngine) searchMessage(msg *protobuf.MessageNode, moduleName, version, query string, results *SearchResults) {
	// Search message name
	if matchesSearch(msg.Name, query) {
		results.Results = append(results.Results, SearchResult{
			Type:     "message",
			Module:   moduleName,
			Version:  version,
			Name:     msg.Name,
			FullName: msg.Name,
			Score:    calculateScore(msg.Name, query),
		})
	}

	// Search fields
	for _, field := range msg.Fields {
		if matchesSearch(field.Name, query) || matchesSearch(field.Type, query) {
			results.Results = append(results.Results, SearchResult{
				Type:     "field",
				Module:   moduleName,
				Version:  version,
				Name:     field.Name,
				FullName: msg.Name + "." + field.Name,
				Score:    calculateScore(field.Name, query),
				Metadata: map[string]string{
					"message": msg.Name,
					"type":    field.Type,
				},
			})
		}
	}

	// Search nested messages
	for _, nested := range msg.Nested {
		s.searchMessage(nested, moduleName, version, query, results)
	}

	// Search nested enums
	for _, enum := range msg.Enums {
		if matchesSearch(enum.Name, query) {
			results.Results = append(results.Results, SearchResult{
				Type:     "enum",
				Module:   moduleName,
				Version:  version,
				Name:     enum.Name,
				FullName: msg.Name + "." + enum.Name,
				Score:    calculateScore(enum.Name, query),
				Metadata: map[string]string{
					"message": msg.Name,
				},
			})
		}
	}
}

// matchesSearch checks if a name matches the search query
func matchesSearch(name, query string) bool {
	if query == "" {
		return false
	}
	return strings.Contains(strings.ToLower(name), query)
}

// matchesPattern checks if a string matches a pattern (supports * wildcard)
func matchesPattern(s, pattern string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	// Simple wildcard matching
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		s = strings.ToLower(s)

		// Check prefix
		if !strings.HasPrefix(s, strings.ToLower(parts[0])) {
			return false
		}

		// Check suffix if exists
		if len(parts) > 1 && parts[1] != "" {
			if !strings.HasSuffix(s, strings.ToLower(parts[1])) {
				return false
			}
		}

		return true
	}

	return strings.EqualFold(s, pattern)
}

// calculateScore calculates relevance score (0.0 - 1.0)
func calculateScore(name, query string) float64 {
	nameLower := strings.ToLower(name)
	queryLower := strings.ToLower(query)

	// Exact match = highest score
	if nameLower == queryLower {
		return 1.0
	}

	// Starts with query = high score
	if strings.HasPrefix(nameLower, queryLower) {
		return 0.8
	}

	// Contains query = medium score
	if strings.Contains(nameLower, queryLower) {
		// Score based on position and length
		index := strings.Index(nameLower, queryLower)
		return 0.5 - (float64(index) / float64(len(nameLower)) * 0.3)
	}

	return 0.0
}
