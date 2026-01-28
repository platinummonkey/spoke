// Package codegen provides protobuf code generation for 15+ programming languages.
//
// # Overview
//
// This package implements the compilation system that transforms protobuf definitions (.proto files)
// into language-specific code. It supports two compilation backends: v1 (legacy) for simple
// direct protoc invocation, and v2 (orchestrator) for Docker-based compilation with advanced
// features like caching, dependency management, and parallel builds.
//
// # Architecture
//
// The code generation system consists of six major components:
//
//   1. Orchestrator (pkg/codegen/orchestrator): Coordinates compilation workflow
//   2. Languages (pkg/codegen/languages): Language specifications and plugin management
//   3. Docker (pkg/codegen/docker): Container-based protoc execution
//   4. Cache (pkg/codegen/cache): Two-tier caching (in-memory L1 + Redis L2)
//   5. Packages (pkg/codegen/packages): Package manager file generation (go.mod, setup.py, etc.)
//   6. Artifacts (pkg/codegen/artifacts): S3 storage for compiled artifacts
//
// # Compilation Backends
//
// V1 (Legacy): Direct protoc invocation on the host machine.
//   - Simpler implementation, fewer dependencies
//   - Requires protoc and language plugins installed locally
//   - No dependency resolution or caching
//   - Used as fallback when v2 unavailable
//
// Set SPOKE_CODEGEN_VERSION=v1 to use.
//
// V2 (Orchestrator): Docker-based compilation with full feature set.
//   - Docker containers with pre-installed protoc and plugins
//   - Automatic dependency resolution and compilation
//   - Two-tier caching (L1 in-memory, L2 Redis)
//   - S3 artifact storage for compiled code
//   - Parallel compilation for multiple languages
//   - Package manager file generation
//
// Set SPOKE_CODEGEN_VERSION=v2 or leave unset (v2 is default).
//
// # Supported Languages
//
// The system supports 15+ languages with varying maturity levels:
//
//	Language       Status    gRPC  Package Manager
//	--------       ------    ----  ---------------
//	Go             Stable    Yes   go.mod
//	Python         Stable    Yes   setup.py, requirements.txt
//	Java           Stable    Yes   pom.xml, build.gradle
//	C++            Stable    Yes   CMakeLists.txt
//	C#             Stable    Yes   .csproj
//	TypeScript     Beta      Yes   package.json
//	JavaScript     Beta      Yes   package.json
//	Rust           Beta      Yes   Cargo.toml
//	Kotlin         Beta      Yes   build.gradle.kts
//	Swift          Beta      Yes   Package.swift
//	Dart           Beta      Yes   pubspec.yaml
//	Ruby           Alpha     Yes   Gemfile
//	PHP            Alpha     No    composer.json
//	Scala          Alpha     Yes   build.sbt
//	Objective-C    Alpha     Yes   Podfile
//
// Language specifications are registered in pkg/codegen/languages/registry.go.
//
// # Basic Usage
//
// Using the orchestrator (v2):
//
//	import (
//		"context"
//		"github.com/platinummonkey/spoke/pkg/codegen/orchestrator"
//	)
//
//	// Create orchestrator
//	config := orchestrator.DefaultConfig()
//	config.EnableCache = true
//	config.S3Bucket = "spoke-artifacts"
//	orch, err := orchestrator.NewOrchestrator(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer orch.Close()
//
//	// Compile for Go
//	req := &orchestrator.CompileRequest{
//		ModuleName:  "user-service",
//		Version:     "v1.0.0",
//		Language:    "go",
//		ProtoFiles: []codegen.ProtoFile{
//			{Path: "user.proto", Content: protoContent},
//		},
//		IncludeGRPC: true,
//		Options: map[string]string{
//			"go_package": "github.com/example/user",
//		},
//	}
//
//	result, err := orch.CompileSingle(context.Background(), req)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if result.Success {
//		fmt.Printf("Generated %d files\n", len(result.GeneratedFiles))
//		for _, file := range result.GeneratedFiles {
//			fmt.Printf("  %s (%d bytes)\n", file.Path, file.Size)
//		}
//	}
//
// Compile for multiple languages in parallel:
//
//	results, err := orch.CompileAll(ctx, req, []string{"go", "python", "typescript"})
//	for _, result := range results {
//		if result.Success {
//			fmt.Printf("%s: ✓ (%v)\n", result.Language, result.Duration)
//		} else {
//			fmt.Printf("%s: ✗ %s\n", result.Language, result.Error)
//		}
//	}
//
// # Compilation Flow
//
// The v2 orchestrator follows this workflow:
//
// 1. Validate Request: Check that module, version, language are valid
// 2. Check Cache: Look for cached compilation result (L1 → L2)
// 3. Prepare Proto Files: Write proto files to temporary directory
// 4. Resolve Dependencies: Fetch and prepare dependency proto files
// 5. Create Docker Container: Launch container with protoc + language plugin
// 6. Execute protoc: Run protoc with appropriate flags inside container
// 7. Generate Package Files: Create go.mod, setup.py, package.json, etc.
// 8. Collect Artifacts: Extract generated files from container
// 9. Cache Result: Store in L1 and L2 cache
// 10. Upload to S3: Optionally upload tarball to S3
// 11. Return Result: Return generated files to caller
//
// # Caching Strategy
//
// The orchestrator uses two-tier caching for performance:
//
// L1 (In-Memory): Fast access, limited size (default 10MB), short TTL (5 minutes).
// Best for repeated compilations within the same server instance.
//
//	// L1 cache hit: ~1ms
//	result, _ := orch.CompileSingle(ctx, req) // First call compiles
//	result, _ := orch.CompileSingle(ctx, req) // Second call uses L1
//
// L2 (Redis): Shared across server instances, larger capacity, longer TTL (24 hours).
// Best for popular modules that multiple servers compile frequently.
//
//	// L2 cache hit: ~50ms (Redis network latency)
//	// Server A compiles module
//	resultA, _ := orchA.CompileSingle(ctx, req)
//	// Server B gets cached result from Redis
//	resultB, _ := orchB.CompileSingle(ctx, req) // Uses L2
//
// Cache keys are generated from:
//   - Module name and version
//   - Language and plugin version
//   - Combined hash of all proto files (including dependencies)
//   - Compilation options
//
// This ensures cache hits only when truly identical compilation would occur.
//
// # Dependency Management
//
// Proto files often import other proto files:
//
//	// user.proto
//	import "common/types.proto";
//
//	message User {
//		common.UUID id = 1;
//	}
//
// The orchestrator automatically resolves dependencies:
//
//	req := &orchestrator.CompileRequest{
//		ModuleName: "user-service",
//		Version:    "v1.0.0",
//		Language:   "go",
//		ProtoFiles: []codegen.ProtoFile{
//			{Path: "user.proto", Content: userProtoContent},
//		},
//		Dependencies: []codegen.Dependency{
//			{
//				ModuleName: "common",
//				Version:    "v1.2.0",
//				ProtoFiles: []codegen.ProtoFile{
//					{Path: "common/types.proto", Content: typesProtoContent},
//				},
//			},
//		},
//	}
//
// The orchestrator:
//   1. Creates proto_path with module + all dependencies
//   2. Runs protoc with --proto_path pointing to all directories
//   3. Generates code with proper import paths
//
// # Docker-Based Compilation
//
// V2 uses Docker for isolation and consistency:
//
//	Docker Image: spoke-protoc:<language>-<version>
//	Example: spoke-protoc:go-1.32.0
//
// Each image contains:
//   - protoc (Protocol Buffers compiler)
//   - Language-specific plugin (protoc-gen-go, protoc-gen-python, etc.)
//   - Runtime dependencies (go compiler, python interpreter, etc.)
//
// Container workflow:
//
//	1. Create container from language-specific image
//	2. Mount proto files as volume: /workspace
//	3. Execute: protoc --<lang>_out=/output --proto_path=/workspace ...
//	4. Extract generated files from /output
//	5. Clean up container
//
// Benefits:
//   - Consistent environment (no "works on my machine")
//   - Isolated from host (security and reproducibility)
//   - Pin specific plugin versions (go plugin 1.32.0 always available)
//   - No local installation required (just Docker)
//
// # Package Manager File Generation
//
// The system generates package manager files for each language:
//
// Go (go.mod):
//
//	module github.com/example/user-service
//	go 1.21
//
//	require (
//		google.golang.org/protobuf v1.32.0
//		google.golang.org/grpc v1.60.0
//	)
//
// Python (setup.py):
//
//	from setuptools import setup, find_packages
//
//	setup(
//		name="user-service",
//		version="1.0.0",
//		packages=find_packages(),
//		install_requires=[
//			"protobuf>=4.25.0",
//			"grpcio>=1.60.0",
//		],
//	)
//
// TypeScript (package.json):
//
//	{
//		"name": "@example/user-service",
//		"version": "1.0.0",
//		"dependencies": {
//			"google-protobuf": "^3.21.0",
//			"@grpc/grpc-js": "^1.9.0"
//		}
//	}
//
// Package generators are registered in pkg/codegen/packages/registry.go.
//
// # Artifact Storage
//
// Compiled artifacts can be stored in S3 for long-term retention:
//
//	config := orchestrator.DefaultConfig()
//	config.S3Bucket = "spoke-artifacts"
//	config.S3Prefix = "compiled/"
//	config.S3Region = "us-east-1"
//	orch, _ := orchestrator.NewOrchestrator(config)
//
//	result, _ := orch.CompileSingle(ctx, req)
//	// Automatically uploaded to: s3://spoke-artifacts/compiled/user-service/v1.0.0/go.tar.gz
//
// Artifacts are stored as compressed tarballs with checksums:
//
//	user-service-v1.0.0-go.tar.gz
//	user-service-v1.0.0-go.tar.gz.sha256
//
// Clients can download pre-compiled artifacts instead of compiling locally.
//
// # Error Handling
//
// Compilation can fail for various reasons:
//
//	result, err := orch.CompileSingle(ctx, req)
//	if err != nil {
//		// Orchestrator-level error (Docker unavailable, etc.)
//		log.Fatal(err)
//	}
//
//	if !result.Success {
//		// Compilation error (proto syntax error, missing import, etc.)
//		fmt.Printf("Compilation failed: %s\n", result.Error)
//	}
//
// Common error types:
//
//	ErrLanguageNotSupported  - Language not registered
//	ErrDockerNotAvailable    - Docker daemon not running
//	ErrProtoSyntaxError      - Invalid proto syntax
//	ErrMissingImport         - Import not found in dependencies
//	ErrPluginCrash           - protoc plugin crashed
//	ErrTimeout               - Compilation exceeded timeout
//
// # Configuration
//
// Orchestrator configuration options:
//
//	config := &orchestrator.Config{
//		// Parallelism
//		MaxParallelWorkers: 10,  // Compile 10 languages simultaneously
//
//		// Caching
//		EnableCache:    true,
//		RedisAddr:      "redis:6379",
//		RedisPassword:  "secret",
//
//		// Storage
//		S3Bucket:       "spoke-artifacts",
//		S3Prefix:       "compiled/",
//		S3Region:       "us-east-1",
//
//		// Timeouts
//		CompilationTimeout: 300,  // 5 minutes max per compilation
//	}
//
// # Performance Considerations
//
// Compilation can be expensive. Optimize by:
//
// 1. Enable caching to avoid redundant compilation:
//
//	config.EnableCache = true
//	config.RedisAddr = "redis:6379"  // Share cache across servers
//
// 2. Pre-compile during version push rather than on-demand:
//
//	// In version creation handler
//	languages := []string{"go", "python", "java"}
//	results, _ := orch.CompileAll(ctx, req, languages)
//	// Store results in database
//
// 3. Use parallel compilation for multiple languages:
//
//	// Compiles all 3 languages simultaneously (if workers available)
//	results, _ := orch.CompileAll(ctx, req, []string{"go", "python", "java"})
//
// 4. Store artifacts in S3 and serve pre-compiled downloads:
//
//	// Client downloads tarball instead of compiling
//	GET /modules/user-service/versions/v1.0.0/download/go
//	→ Returns: s3://spoke-artifacts/compiled/user-service/v1.0.0/go.tar.gz
//
// 5. Monitor cache hit rates and adjust TTL:
//
//	metrics := orch.GetMetrics()
//	fmt.Printf("Cache hit rate: %.1f%%\n", metrics.CacheHitRate*100)
//
// # Testing
//
// Test compilation without Docker:
//
//	// Mock the Docker runner
//	type mockRunner struct {
//		docker.Runner
//		executeFunc func(*docker.ExecutionRequest) (*docker.ExecutionResult, error)
//	}
//
//	// In test
//	mock := &mockRunner{
//		executeFunc: func(req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
//			return &docker.ExecutionResult{
//				Success: true,
//				Files: []docker.OutputFile{
//					{Path: "user.pb.go", Content: []byte("package user")},
//				},
//			}, nil
//		},
//	}
//
// For integration tests, use real Docker:
//
//	func TestCompilation(t *testing.T) {
//		if testing.Short() {
//			t.Skip("Skipping Docker-based test")
//		}
//
//		orch, _ := orchestrator.NewOrchestrator(nil)
//		result, err := orch.CompileSingle(ctx, req)
//		assert.NoError(t, err)
//		assert.True(t, result.Success)
//	}
//
// # V1 vs V2 Migration
//
// The v1 backend is simpler but less capable. The v2 backend (orchestrator) adds:
//
//	Feature                  V1      V2
//	--------                 --      --
//	Basic compilation        Yes     Yes
//	Dependency resolution    No      Yes
//	Caching                  No      Yes
//	S3 artifact storage      No      Yes
//	Package file generation  No      Yes
//	Parallel compilation     No      Yes
//	Plugin version pinning   No      Yes
//	Docker isolation         No      Yes
//
// Migration path:
//
//	1. Deploy v2 alongside v1 (both available)
//	2. Route new compilations to v2 via SPOKE_CODEGEN_VERSION=v2
//	3. Monitor for errors, fall back to v1 if needed
//	4. Once stable, make v2 default
//	5. Eventually deprecate v1
//
// The API layer automatically routes to the appropriate backend based on
// SPOKE_CODEGEN_VERSION environment variable.
//
// # Subpackages
//
//   - orchestrator: Compilation workflow coordination
//   - languages: Language specs and plugin registry
//   - docker: Container-based protoc execution
//   - cache: Two-tier caching (L1 in-memory, L2 Redis)
//   - packages: Package manager file generation
//   - artifacts: S3 storage for compiled artifacts
//
// # Related Packages
//
//   - pkg/api: HTTP API that invokes compilation
//   - pkg/storage: Stores proto files and compiled artifacts
//   - pkg/dependencies: Resolves proto import relationships
//   - pkg/validation: Validates proto files before compilation
//   - pkg/compatibility: Checks compatibility between versions
//
// # Design Decisions
//
// Docker-Based Execution: Using Docker ensures consistent, reproducible builds across
// environments. No more "it compiles on my laptop but not in CI" issues.
//
// Two-Tier Caching: L1 for speed (in-memory), L2 for sharing (Redis). This balances
// performance with resource efficiency - hot modules use L1, warm modules use L2.
//
// Async Compilation: Large compilations can be queued and processed asynchronously,
// preventing API timeouts. Clients poll for status and retrieve results when ready.
//
// Language Registry: Centralizing language specifications makes it easy to add new
// languages or update plugin versions without touching orchestrator code.
//
// Package Generation: Generating go.mod, setup.py, etc. makes compiled artifacts
// immediately usable - users can directly import without manual setup.
//
// Content-Addressed Caching: Cache keys include proto file hashes, ensuring cache hits
// only occur when truly identical compilation would happen. Prevents serving stale artifacts.
package codegen
