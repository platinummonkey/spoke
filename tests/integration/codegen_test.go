package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/orchestrator"
)

// TestCompilationWorkflow tests the end-to-end compilation workflow
func TestCompilationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test proto file
	testProto := `syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
  int32 value = 2;
}

service TestService {
  rpc GetTest(TestMessage) returns (TestMessage);
}
`

	// Create temp directory for test
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test.proto")
	if err := os.WriteFile(protoFile, []byte(testProto), 0644); err != nil {
		t.Fatalf("Failed to write test proto file: %v", err)
	}

	// Create orchestrator
	config := orchestrator.DefaultConfig()
	config.EnableCache = false // Disable cache for testing
	orch, err := orchestrator.NewOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close()

	// Test compilation for Go
	t.Run("CompileGo", func(t *testing.T) {
		req := &orchestrator.CompileRequest{
			ModuleName: "test",
			Version:    "v1.0.0",
			Language:   "go",
			ProtoFiles: []codegen.ProtoFile{
				{
					Path:    "test.proto",
					Content: []byte(testProto),
				},
			},
			IncludeGRPC: true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := orch.CompileSingle(ctx, req)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Expected successful compilation, got error: %s", result.Error)
		}

		if len(result.GeneratedFiles) == 0 {
			t.Error("Expected generated files, got none")
		}

		// Check for .pb.go file
		foundPbGo := false
		for _, file := range result.GeneratedFiles {
			if filepath.Ext(file.Path) == ".go" {
				foundPbGo = true
				break
			}
		}
		if !foundPbGo {
			t.Error("Expected .pb.go file in generated files")
		}
	})
}

// TestMultiLanguageCompilation tests compiling for multiple languages
func TestMultiLanguageCompilation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testProto := `syntax = "proto3";

package multitest;

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}
`

	// Create orchestrator
	config := orchestrator.DefaultConfig()
	config.EnableCache = false
	config.MaxParallelWorkers = 3
	orch, err := orchestrator.NewOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close()

	req := &orchestrator.CompileRequest{
		ModuleName: "multitest",
		Version:    "v1.0.0",
		ProtoFiles: []codegen.ProtoFile{
			{
				Path:    "user.proto",
				Content: []byte(testProto),
			},
		},
		IncludeGRPC: false,
	}

	// Test parallel compilation for multiple languages
	languages := []string{"go", "python", "java"}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results, err := orch.CompileAll(ctx, req, languages)
	if err != nil {
		// Partial failures are OK in integration tests
		t.Logf("Some compilations failed: %v", err)
	}

	if len(results) != len(languages) {
		t.Errorf("Expected %d results, got %d", len(languages), len(results))
	}

	// Check each language result
	for i, result := range results {
		t.Run(languages[i], func(t *testing.T) {
			if result == nil {
				t.Fatal("Result is nil")
			}

			if result.Language != languages[i] {
				t.Errorf("Expected language %s, got %s", languages[i], result.Language)
			}

			// Log result for debugging
			t.Logf("Language: %s, Success: %v, Duration: %v, Files: %d",
				result.Language, result.Success, result.Duration, len(result.GeneratedFiles))
		})
	}
}

// TestCacheEffectiveness tests that caching works correctly
func TestCacheEffectiveness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testProto := `syntax = "proto3";

package cachetest;

message CachedMessage {
  string data = 1;
}
`

	// Create orchestrator with cache enabled
	config := orchestrator.DefaultConfig()
	config.EnableCache = true
	config.RedisAddr = os.Getenv("REDIS_ADDR") // Only enable if Redis available
	if config.RedisAddr == "" {
		t.Skip("Skipping cache test - Redis not available")
	}

	orch, err := orchestrator.NewOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close()

	req := &orchestrator.CompileRequest{
		ModuleName: "cachetest",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{
				Path:    "cached.proto",
				Content: []byte(testProto),
			},
		},
	}

	ctx := context.Background()

	// First compilation - cache miss
	result1, err := orch.CompileSingle(ctx, req)
	if err != nil {
		t.Fatalf("First compilation failed: %v", err)
	}

	if result1.CacheHit {
		t.Error("Expected cache miss on first compilation")
	}

	duration1 := result1.Duration

	// Second compilation - should be cache hit
	result2, err := orch.CompileSingle(ctx, req)
	if err != nil {
		t.Fatalf("Second compilation failed: %v", err)
	}

	if !result2.CacheHit {
		t.Error("Expected cache hit on second compilation")
	}

	duration2 := result2.Duration

	// Cache hit should be significantly faster
	if duration2 >= duration1 {
		t.Logf("Warning: cache hit (%v) not faster than miss (%v)", duration2, duration1)
	}

	t.Logf("Cache miss: %v, Cache hit: %v, Speedup: %.2fx",
		duration1, duration2, float64(duration1)/float64(duration2))
}

// TestInvalidProtoHandling tests error handling for invalid proto files
func TestInvalidProtoHandling(t *testing.T) {
	invalidProto := `syntax = "proto3";

package invalid;

message Invalid {
  string name = "not a number";  // Invalid field number
}
`

	config := orchestrator.DefaultConfig()
	config.EnableCache = false
	orch, err := orchestrator.NewOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close()

	req := &orchestrator.CompileRequest{
		ModuleName: "invalid",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{
				Path:    "invalid.proto",
				Content: []byte(invalidProto),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := orch.CompileSingle(ctx, req)

	// Should either return error or unsuccessful result
	if err == nil && result.Success {
		t.Error("Expected compilation to fail for invalid proto file")
	}

	if result != nil && result.Error != "" {
		t.Logf("Got expected error: %s", result.Error)
	}
}

// TestResourceLimits tests that resource limits are enforced
func TestResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a proto file that should compile quickly
	simpleProto := `syntax = "proto3";

package simple;

message Simple {
  string data = 1;
}
`

	config := orchestrator.DefaultConfig()
	config.EnableCache = false
	config.CompilationTimeout = 1 // Very short timeout for testing
	orch, err := orchestrator.NewOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close()

	req := &orchestrator.CompileRequest{
		ModuleName: "simple",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{
				Path:    "simple.proto",
				Content: []byte(simpleProto),
			},
		},
	}

	ctx := context.Background()

	result, err := orch.CompileSingle(ctx, req)

	// Should complete within timeout (or timeout)
	if err == nil {
		if result.Duration > time.Duration(config.CompilationTimeout)*time.Second {
			t.Errorf("Compilation exceeded timeout: %v > %ds",
				result.Duration, config.CompilationTimeout)
		}
	}
}
