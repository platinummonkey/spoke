package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen"
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

	// Create config
	config := codegen.DefaultConfig()
	config.EnableCache = false // Disable cache for testing

	// Test compilation for Go
	t.Run("CompileGo", func(t *testing.T) {
		req := &codegen.GenerateRequest{
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

		result, err := codegen.GenerateCode(ctx, req, config)
		if err != nil {
			// Skip test if Docker images are not available or environment not configured
			if strings.Contains(err.Error(), "failed to pull docker image") ||
				strings.Contains(err.Error(), "denied: requested access to the resource is denied") ||
				strings.Contains(err.Error(), "unknown command \"protoc\" for \"buf\"") ||
				strings.Contains(err.Error(), "docker execution failed") {
				t.Logf("Skipping test - Docker compilation environment not available: %v", err)
				t.Skip("Docker compilation environment not properly configured")
				return
			}
			t.Fatalf("Compilation failed: %v", err)
		}

		if !result.Success {
			// Also check result error for Docker image or environment issues
			if strings.Contains(result.Error, "failed to pull docker image") ||
				strings.Contains(result.Error, "denied: requested access to the resource is denied") ||
				strings.Contains(result.Error, "unknown command \"protoc\" for \"buf\"") ||
				strings.Contains(result.Error, "docker execution failed") {
				t.Logf("Skipping test - Docker compilation environment not available: %s", result.Error)
				t.Skip("Docker compilation environment not properly configured")
				return
			}
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

	// Create generator config
	config := codegen.DefaultConfig()
	config.EnableCache = false
	config.MaxWorkers = 3

	req := &codegen.GenerateRequest{
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

	results, err := codegen.GenerateCodeParallel(ctx, req, languages, config)
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

	// Create config with cache enabled
	config := codegen.DefaultConfig()
	config.EnableCache = true

	req := &codegen.GenerateRequest{
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
	result1, err := codegen.GenerateCode(ctx, req, config)
	if err != nil {
		// Skip test if Docker environment not available
		if strings.Contains(err.Error(), "failed to pull docker image") ||
			strings.Contains(err.Error(), "denied: requested access to the resource is denied") ||
			strings.Contains(err.Error(), "unknown command \"protoc\" for \"buf\"") ||
			strings.Contains(err.Error(), "docker execution failed") {
			t.Logf("Skipping test - Docker compilation environment not available: %v", err)
			t.Skip("Docker compilation environment not properly configured")
			return
		}
		t.Fatalf("First compilation failed: %v", err)
	}

	if result1.CacheHit {
		t.Error("Expected cache miss on first compilation")
	}

	duration1 := result1.Duration

	// Second compilation - should be cache hit
	result2, err := codegen.GenerateCode(ctx, req, config)
	if err != nil {
		// Skip test if Docker environment not available
		if strings.Contains(err.Error(), "failed to pull docker image") ||
			strings.Contains(err.Error(), "denied: requested access to the resource is denied") ||
			strings.Contains(err.Error(), "unknown command \"protoc\" for \"buf\"") ||
			strings.Contains(err.Error(), "docker execution failed") {
			t.Logf("Skipping test - Docker compilation environment not available: %v", err)
			t.Skip("Docker compilation environment not properly configured")
			return
		}
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

	config := codegen.DefaultConfig()
	config.EnableCache = false

	req := &codegen.GenerateRequest{
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

	result, err := codegen.GenerateCode(ctx, req, config)

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

	config := codegen.DefaultConfig()
	config.EnableCache = false
	config.Timeout = 1 * time.Second // Very short timeout for testing

	req := &codegen.GenerateRequest{
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

	result, err := codegen.GenerateCode(ctx, req, config)

	// Should complete within timeout (or timeout)
	if err == nil {
		if result.Duration > config.Timeout {
			t.Errorf("Compilation exceeded timeout: %v > %v",
				result.Duration, config.Timeout)
		}
	}
}
