// Package api provides the HTTP REST API server for the Spoke protobuf schema registry.
//
// # Overview
//
// This package implements the core HTTP API layer that exposes Spoke's functionality
// as RESTful endpoints. It handles module registration, version management, proto file
// retrieval, code compilation, schema validation, and various enterprise features like
// authentication, RBAC, analytics, and search.
//
// # Architecture
//
// The API is built on gorilla/mux and organized into domain-specific handler groups:
//
//   - Module Management: Create, list, and retrieve protobuf modules
//   - Version Management: Register and query specific versions of modules
//   - File Operations: Fetch individual proto files from versions
//   - Compilation: Generate code for 15+ programming languages
//   - Compatibility: Check for breaking changes between schema versions
//   - Validation: Lint and validate proto files
//   - Authentication: User login, registration, password management (requires database)
//   - Authorization: Role-based access control (RBAC) for modules and versions
//   - Analytics: Track usage metrics and API calls
//   - Search: Full-text search across proto definitions
//   - User Features: Saved searches, bookmarks, and preferences
//
// # Key Types
//
// Server is the main API server that coordinates all functionality:
//
//	server := api.NewServer(storage, db)
//	http.ListenAndServe(":8080", server)
//
// Module represents a protobuf module (e.g., "user-service", "common-types"):
//
//	module := &api.Module{
//		Name:        "user-service",
//		Description: "User management service definitions",
//	}
//
// Version represents a specific release of a module with its proto files:
//
//	version := &api.Version{
//		ModuleName: "user-service",
//		Version:    "v1.2.0",
//		Files: []api.File{
//			{Path: "user.proto", Content: "syntax = \"proto3\";..."},
//		},
//	}
//
// CompilationInfo contains generated code artifacts for a target language:
//
//	info := api.CompilationInfo{
//		Language:    api.LanguageGo,
//		PackageName: "github.com/example/user",
//		Files: []api.File{
//			{Path: "user.pb.go", Content: "package user..."},
//		},
//	}
//
// # API Endpoints
//
// Core Schema Registry API (v1):
//
//	POST   /modules                                           - Create module
//	GET    /modules                                           - List all modules
//	GET    /modules/{name}                                    - Get module details
//	POST   /modules/{name}/versions                           - Register version
//	GET    /modules/{name}/versions                           - List versions
//	GET    /modules/{name}/versions/{version}                 - Get version details
//	GET    /modules/{name}/versions/{version}/files/{path}    - Download proto file
//	GET    /modules/{name}/versions/{version}/download/{lang} - Download compiled code
//
// Enhanced API (v2):
//
//	GET    /api/v1/languages                                  - List supported languages
//	GET    /api/v1/languages/{id}                             - Get language details
//	POST   /api/v1/modules/{name}/versions/{version}/compile  - Compile to languages
//	GET    /api/v1/modules/{name}/versions/{version}/compile/{jobId} - Check compilation status
//	GET    /api/v1/modules/{name}/versions/{version}/examples/{lang} - Generate usage examples
//	POST   /api/v1/modules/{name}/diff                        - Compare versions for breaking changes
//
// Authentication & Authorization (requires database):
//
//	POST   /auth/register                                     - Register new user
//	POST   /auth/login                                        - Login and get JWT token
//	POST   /auth/refresh                                      - Refresh JWT token
//	GET    /auth/me                                           - Get current user info
//
// Enterprise Features (requires database):
//
//	GET    /api/v1/search                                     - Search proto definitions
//	POST   /api/v1/bookmarks                                  - Save bookmark
//	GET    /api/v1/saved-searches                             - List saved searches
//	GET    /api/v1/analytics/events                           - Query usage analytics
//	POST   /api/v1/compatibility/check                        - Check schema compatibility
//	POST   /api/v1/validation/lint                            - Lint proto files
//
// # Compilation System
//
// The API uses the orchestrator-based compilation system (pkg/codegen/orchestrator)
// which provides Docker-based isolation, dependency management, caching, and parallel builds.
//
// Compilation is triggered automatically when publishing a version, or can be invoked manually:
//
//	info, err := server.CompileGo(version)
//	info, err := server.CompilePython(version)
//
// # Storage Interface Migration
//
// The api.Storage interface is DEPRECATED and will be removed in v2.0.0.
// New code should use the storage.Storage interface from pkg/storage, which provides:
//
//   - Context-aware methods for cancellation and timeouts
//   - Multiple backend support (filesystem, PostgreSQL, Redis, S3)
//   - Improved error handling and consistency
//
// See pkg/storage/DEPRECATION.md for migration details.
//
// # Authentication & Security
//
// Authentication is optional and only available when a database is provided.
// Without a database, the API runs in public mode with no access controls.
//
// With database authentication:
//
//   - JWT-based authentication (login returns access + refresh tokens)
//   - Role-based access control (admin, editor, viewer roles)
//   - Organization-scoped permissions
//   - API key authentication for programmatic access
//
// Protected endpoints require Authorization header:
//
//	Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
//
// # Usage Example
//
// Basic server setup:
//
//	package main
//
//	import (
//		"database/sql"
//		"log"
//		"net/http"
//
//		"github.com/platinummonkey/spoke/pkg/api"
//		"github.com/platinummonkey/spoke/pkg/storage"
//	)
//
//	func main() {
//		// Initialize storage backend
//		store := storage.NewFilesystemStorage("/var/spoke/storage")
//
//		// Optional: Initialize database for enterprise features
//		db, _ := sql.Open("postgres", "postgres://localhost/spoke")
//
//		// Create API server
//		server := api.NewServer(store, db)
//
//		// Start HTTP server
//		log.Fatal(http.ListenAndServe(":8080", server))
//	}
//
// Client usage example:
//
//	// Register a module
//	POST /modules
//	{
//		"name": "user-service",
//		"description": "User management API"
//	}
//
//	// Push a version
//	POST /modules/user-service/versions
//	{
//		"version": "v1.0.0",
//		"files": [
//			{
//				"path": "user.proto",
//				"content": "syntax = \"proto3\"; message User { string id = 1; }"
//			}
//		]
//	}
//
//	// Compile to Go
//	POST /api/v1/modules/user-service/versions/v1.0.0/compile
//	{
//		"languages": ["go"],
//		"include_grpc": true
//	}
//
//	// Download compiled code
//	GET /modules/user-service/versions/v1.0.0/download/go
//
// # Design Decisions
//
// Modular Handler Design: Domain-specific handlers (AuthHandlers, CompatibilityHandlers)
// are registered with the Server. This keeps concerns separated and makes testing easier.
//
// Optional Features: Enterprise features like auth, search, and analytics are only enabled
// when a database is provided. This allows Spoke to run in lightweight mode for development
// or small deployments.
//
// Dual Compilation System: Supporting both v1 (legacy) and v2 (orchestrator) compilation
// provides backward compatibility while enabling gradual migration. The system automatically
// routes to the appropriate backend.
//
// Storage Abstraction: The Storage interface isolates the API from persistence details,
// allowing multiple backends (filesystem, database, cloud storage) without changing API code.
//
// RESTful Design: The API follows REST conventions with resource-based URLs, standard HTTP
// methods (GET/POST/PUT/DELETE), and JSON request/response bodies.
//
// # Performance Considerations
//
// Compilation Caching: Compiled artifacts are cached to avoid redundant compilation.
// The orchestrator (v2) includes sophisticated cache key generation based on proto content
// and dependencies.
//
// Async Compilation: Large compilation jobs can be run asynchronously. The compile endpoint
// returns a job ID that clients poll for status.
//
// Search Indexing: The search indexer runs in the background, parsing proto files and
// indexing messages, enums, services, and fields for fast full-text search.
//
// Database Connection Pooling: When using database features, ensure proper connection pool
// configuration for production deployments.
//
// # Related Packages
//
//   - pkg/storage: Storage backends for modules and versions
//   - pkg/codegen: Code generation system (v1 legacy)
//   - pkg/codegen/orchestrator: Docker-based compilation orchestrator (v2)
//   - pkg/auth: User authentication and session management
//   - pkg/rbac: Role-based access control
//   - pkg/compatibility: Schema compatibility checking
//   - pkg/validation: Proto linting and validation
//   - pkg/search: Full-text search indexing
//   - pkg/analytics: Usage tracking and metrics
package api
