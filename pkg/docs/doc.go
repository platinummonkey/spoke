// Package docs provides documentation generation from protobuf schemas.
//
// # Overview
//
// This package generates structured documentation from proto files, preserving comments
// and metadata, with support for multiple export formats (Markdown, HTML).
//
// # Documentation Structure
//
// Extracted Elements:
//   - Messages with fields and nested types
//   - Enums with values
//   - Services with RPC methods
//   - Field types and tags
//   - Deprecation markers
//   - Comments and descriptions
//
// # Usage Example
//
// Generate documentation:
//
//	generator := docs.NewGenerator()
//	documentation := generator.Generate(protoAST)
//
// Export to Markdown:
//
//	exporter := docs.NewMarkdownExporter()
//	markdown := exporter.Export(documentation)
//	// Save to README.md
//
// Export to HTML:
//
//	exporter := docs.NewHTMLExporter(&docs.HTMLOptions{
//		Theme:         "github",
//		IncludeTOC:    true,
//		SyntaxHighlight: true,
//	})
//	html := exporter.Export(documentation)
//
// # Diff Support
//
// The pkg/docs/diff subpackage provides change documentation:
//
//	diff := diff.Compare(oldDoc, newDoc)
//	fmt.Printf("Added: %d, Removed: %d, Modified: %d\n",
//		diff.Added, diff.Removed, diff.Modified)
//
// # Related Packages
//
//   - pkg/docs/diff: Documentation diffing
//   - pkg/docs/examples: Usage example generation
package docs
