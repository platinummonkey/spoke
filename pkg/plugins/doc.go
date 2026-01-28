// Package plugins provides an extensibility framework for protobuf compilation and validation.
//
// # Overview
//
// This package manages plugin discovery, loading, validation, and execution for extending
// Spoke's capabilities with custom language generators, validators, and runners.
//
// # Plugin System
//
// Plugin Interface: Base interface all plugins implement (Manifest, Load, Unload)
// Registry: In-memory registry for loaded plugins
// Loader: Discovers and loads plugins from filesystem and marketplace
// Validator: Validates plugin manifests and security
//
// # Plugin Types
//
// LanguagePlugin: Generates code for target languages
//
//	type LanguagePlugin interface {
//		Plugin
//		GetLanguageSpec() *LanguageSpec
//		BuildProtocCommand(req *CompileRequest) ([]string, error)
//	}
//
// ValidatorPlugin: Validates proto schemas
//
//	type ValidatorPlugin interface {
//		Plugin
//		Validate(schema *Schema) ([]Violation, error)
//	}
//
// # Security
//
// Manifest validation: Required fields, version formats
// Permission checking: File access, network access, etc.
// Security scanning: CWE issue detection
// Signature verification: Plugin integrity
//
// # Usage Example
//
// Load plugin:
//
//	registry := plugins.NewRegistry()
//	loader := plugins.NewLoader(registry)
//
//	plugin, err := loader.LoadFromFile("/plugins/rust-gen.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Execute language plugin:
//
//	langPlugin := plugin.(plugins.LanguagePlugin)
//	spec := langPlugin.GetLanguageSpec()
//	fmt.Printf("Language: %s, Plugin: %s\n", spec.ID, spec.PluginVersion)
//
// # Related Packages
//
//   - pkg/plugins/buf: Buf registry integration
//   - pkg/marketplace: Plugin distribution
//   - pkg/codegen: Uses language plugins
package plugins
