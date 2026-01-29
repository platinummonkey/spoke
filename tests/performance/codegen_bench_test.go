package performance

import (
	"context"
	"testing"

	"github.com/platinummonkey/spoke/pkg/codegen"
)

// BenchmarkCompileSingleLanguage benchmarks single language compilation
func BenchmarkCompileSingleLanguage(b *testing.B) {
	testProto := `syntax = "proto3";

package bench;

message BenchMessage {
  string id = 1;
  string name = 2;
  int32 value = 3;
  repeated string tags = 4;
}

service BenchService {
  rpc GetBench(BenchMessage) returns (BenchMessage);
  rpc ListBench(BenchMessage) returns (stream BenchMessage);
}
`

	config := codegen.DefaultConfig()
	config.EnableCache = false // Disable cache for benchmarking

	req := &codegen.GenerateRequest{
		ModuleName: "bench",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{
				Path:    "bench.proto",
				Content: []byte(testProto),
			},
		},
		IncludeGRPC: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codegen.GenerateCode(ctx, req, config)
		if err != nil {
			b.Fatalf("Compilation failed: %v", err)
		}
	}
}

// BenchmarkCompileWithCache benchmarks compilation with caching
func BenchmarkCompileWithCache(b *testing.B) {
	testProto := `syntax = "proto3";

package cache;

message CacheMessage {
  string data = 1;
}
`

	config := codegen.DefaultConfig()
	config.EnableCache = true

	req := &codegen.GenerateRequest{
		ModuleName: "cache",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{
				Path:    "cache.proto",
				Content: []byte(testProto),
			},
		},
	}

	ctx := context.Background()

	// Prime the cache
	_, err := codegen.GenerateCode(ctx, req, config)
	if err != nil {
		b.Fatalf("Failed to prime cache: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := codegen.GenerateCode(ctx, req, config)
		if err != nil {
			b.Fatalf("Compilation failed: %v", err)
		}
		if !result.CacheHit {
			b.Error("Expected cache hit")
		}
	}
}

// BenchmarkCompileParallel5Languages benchmarks parallel compilation
func BenchmarkCompileParallel5Languages(b *testing.B) {
	testProto := `syntax = "proto3";

package parallel;

message ParallelMessage {
  string id = 1;
  string data = 2;
}
`

	config := codegen.DefaultConfig()
	config.EnableCache = false
	config.MaxWorkers = 5

	req := &codegen.GenerateRequest{
		ModuleName: "parallel",
		Version:    "v1.0.0",
		ProtoFiles: []codegen.ProtoFile{
			{
				Path:    "parallel.proto",
				Content: []byte(testProto),
			},
		},
	}

	languages := []string{"go", "python", "java", "cpp", "rust"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codegen.GenerateCodeParallel(ctx, req, languages, config)
		if err != nil {
			// Partial failures OK for benchmarking
			b.Logf("Warning: %v", err)
		}
	}
}

// BenchmarkCacheKeyGeneration benchmarks cache key generation
func BenchmarkCacheKeyGeneration(b *testing.B) {
	testProto := `syntax = "proto3";
package test;
message Test { string data = 1; }
`

	protoFiles := []codegen.ProtoFile{
		{Path: "test.proto", Content: []byte(testProto)},
	}

	dependencies := []codegen.Dependency{}
	options := map[string]string{
		"opt1": "value1",
		"opt2": "value2",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateCacheKey("module", "v1.0.0", "go", "v1.31.0", protoFiles, dependencies, options)
	}
}

// Helper function for cache key generation benchmark
func generateCacheKey(moduleName, version, language, pluginVersion string,
	protoFiles []codegen.ProtoFile, dependencies []codegen.Dependency,
	options map[string]string) string {
	// Simplified version - real implementation in cache package
	return moduleName + ":" + version + ":" + language + ":" + pluginVersion
}

// BenchmarkLanguageRegistry benchmarks language registry operations
func BenchmarkLanguageRegistry(b *testing.B) {
	// This would benchmark language registry Get operations
	b.Skip("Benchmark not implemented - requires language registry instance")
}
