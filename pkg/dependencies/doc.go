// Package dependencies provides proto import dependency resolution and graph analysis.
//
// # Overview
//
// This package extracts dependencies from proto import statements, builds dependency graphs,
// detects circular dependencies, and analyzes impact of schema changes.
//
// # Key Features
//
// Dependency Resolution: Parse imports and resolve to module/version
// Graph Analysis: Build complete dependency graphs with transitive dependencies
// Circular Detection: Find and report circular import chains
// Impact Analysis: Show what modules would be affected by changes
// Lockfiles: Generate dependency lockfiles for reproducibility
//
// # Usage Example
//
// Build dependency graph:
//
//	resolver := dependencies.NewResolver(storage)
//	graph, err := resolver.Resolve(ctx, moduleName, version)
//
//	fmt.Printf("Direct dependencies: %d\n", len(graph.DirectDeps))
//	fmt.Printf("Transitive dependencies: %d\n", len(graph.TransitiveDeps))
//
// Detect circular dependencies:
//
//	cycles := graph.DetectCycles()
//	if len(cycles) > 0 {
//		fmt.Println("Circular dependencies found:")
//		for _, cycle := range cycles {
//			fmt.Printf("  %s\n", strings.Join(cycle, " -> "))
//		}
//	}
//
// Impact analysis:
//
//	impact := resolver.AnalyzeImpact(ctx, "common", "v1.0.0")
//	fmt.Printf("Modules affected: %d\n", len(impact.AffectedModules))
//	for _, module := range impact.AffectedModules {
//		fmt.Printf("  - %s (used by %d versions)\n",
//			module.Name, module.UsageCount)
//	}
//
// Generate lockfile:
//
//	lockfile := graph.GenerateLockfile()
//	// Save to proto.lock
//
// # Related Packages
//
//   - pkg/compatibility: Breaking change detection
//   - pkg/codegen: Uses dependency graph for compilation
package dependencies
