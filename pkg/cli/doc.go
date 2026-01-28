// Package cli provides the Spoke command-line interface for schema management.
//
// # Overview
//
// This package implements the `spoke` CLI tool for developers to push/pull proto files,
// compile schemas, validate syntax, and manage the schema registry from the terminal.
//
// # Commands
//
// push: Upload proto files to registry
//
//	spoke push \
//		--module user-service \
//		--version v1.0.0 \
//		--dir ./proto \
//		--description "User management API"
//
// pull: Download proto files from registry
//
//	spoke pull \
//		--module user-service \
//		--version v1.0.0 \
//		--dir ./proto \
//		--recursive  # Include dependencies
//
// compile: Compile proto files locally
//
//	spoke compile \
//		--dir ./proto \
//		--out ./generated \
//		--lang go \
//		--grpc
//
// Multi-language compilation:
//
//	spoke compile \
//		--dir ./proto \
//		--out ./generated \
//		--languages go,python,typescript \
//		--parallel
//
// validate: Validate proto file syntax
//
//	spoke validate --dir ./proto --recursive
//
// batch-push: Push multiple modules
//
//	spoke batch-push --dir ./schemas --recursive
//
// check-compatibility: Check compatibility
//
//	spoke check-compatibility \
//		--module user-service \
//		--old-version v1.0.0 \
//		--new-version v1.1.0 \
//		--mode BACKWARD
//
// lint: Lint proto files
//
//	spoke lint --dir ./proto --style-guide google
//
// languages: List supported languages
//
//	spoke languages
//
// # Configuration
//
// Registry URL:
//
//	export SPOKE_REGISTRY_URL="https://registry.example.com"
//	# Or use --registry flag
//
// # Proto Import Resolution
//
// The CLI automatically:
//   - Extracts import statements from proto files
//   - Maps imports to module/version dependencies
//   - Resolves transitive dependencies
//   - Downloads dependencies recursively with --recursive
//
// # Git Integration
//
// Automatically extracts git metadata during push:
//   - Repository URL
//   - Commit SHA
//   - Branch name
//
// # Related Packages
//
//   - pkg/api: Makes HTTP calls to registry
//   - pkg/api/protobuf: Parses proto files
package cli
