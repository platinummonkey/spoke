// Package search provides full-text search capabilities for protobuf schema discovery.
//
// # Overview
//
// This package implements PostgreSQL full-text search over all proto entities (messages,
// fields, enums, services, methods) with advanced query filtering and relevance ranking.
//
// # Search Features
//
// Full-Text Search: Natural language queries across all proto elements
// Entity Filtering: Search specific entity types (message, field, enum, service, method)
// Module/Version Filtering: Scope searches to specific modules or versions
// Relevance Ranking: Results sorted by search score
// Query Suggestions: Auto-complete based on search history
//
// # Query Syntax
//
// Simple search:
//
//	search?q=user
//
// Entity type filter:
//
//	search?q=user&type=message
//
// Module filter:
//
//	search?q=GetUser&module=user-service
//
// Multiple filters:
//
//	search?q=id&type=field&module=user-service&version=v1.0.0
//
// # Usage Example
//
// Search messages:
//
//	results, err := searchService.Search(ctx, &search.Request{
//		Query:      "user",
//		EntityType: search.EntityTypeMessage,
//		Limit:      20,
//	})
//
//	for _, result := range results {
//		fmt.Printf("%s.%s (score: %.2f)\n",
//			result.ModuleName, result.EntityName, result.Score)
//	}
//
// Index new version:
//
//	indexer := search.NewIndexer(db, storage)
//	err := indexer.IndexVersion(ctx, moduleName, version)
//
// # Related Packages
//
//   - pkg/storage: Loads proto files for indexing
//   - pkg/docs: Documentation search
package search
