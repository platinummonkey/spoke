// Package marketplace provides plugin discovery and distribution for the Spoke registry.
//
// # Overview
//
// This package manages a marketplace for schema compilation and validation plugins with
// versioning, reviews, installation tracking, and security levels.
//
// # Plugin Types
//
// Language: Code generators (Go, Python, Java, etc.)
// Validator: Schema linters and validators
// Generator: Documentation generators, mock generators
// Runner: Custom execution environments
// Transform: Schema transformation utilities
//
// # Security Levels
//
// Official: Maintained by Spoke team, fully trusted
// Verified: Reviewed and approved by Spoke, trusted
// Community: User-submitted, use with caution
//
// # Usage Example
//
// Search plugins:
//
//	results, err := service.Search(ctx, &marketplace.SearchRequest{
//		Query:  "rust",
//		Type:   marketplace.PluginTypeLanguage,
//		Level:  marketplace.SecurityLevelOfficial,
//	})
//
// Install plugin:
//
//	plugin, err := service.Download(ctx, "rust-generator", "v1.0.0")
//	// Verify checksum
//	if !plugin.VerifyChecksum() {
//		return errors.New("checksum mismatch")
//	}
//
// # Related Packages
//
//   - pkg/plugins: Plugin system
//   - pkg/codegen: Uses language plugins
package marketplace
