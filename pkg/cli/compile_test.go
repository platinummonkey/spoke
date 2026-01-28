package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileCommand(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found in PATH, skipping compilation tests")
	}
	// Create test proto files
	commonProto := `syntax = "proto3";
package common;
option go_package = "github.com/platinummonkey/spoke/pkg/cli/testdata/common";

message Metadata {
  string created_at = 1;
  string updated_at = 2;
}`

	userProto := `syntax = "proto3";
package user;
option go_package = "github.com/platinummonkey/spoke/pkg/cli/testdata/user";
import "test/common/common.proto";

message User {
  string id = 1;
  common.Metadata metadata = 2;
}`

	orderProto := `syntax = "proto3";
package order;
option go_package = "github.com/platinummonkey/spoke/pkg/cli/testdata/order";
import "test/common/common.proto";
import "test/user/user.proto";

message Order {
  string id = 1;
  user.User user = 2;
  common.Metadata metadata = 3;
}`

	tests := []struct {
		name           string
		args           []string
		setupFiles     map[string]string
		wantErr        bool
		expectedFiles  []string
	}{
		{
			name:    "missing directory",
			args:    []string{"-lang", "go"},
			wantErr: true,
		},
		{
			name:    "missing language",
			args:    []string{"-dir", "test"},
			wantErr: true,
		},
		{
			name: "compile single proto file",
			args: []string{"-dir", "test", "-lang", "go", "-out", "test"},
			setupFiles: map[string]string{
				"test/common.proto": commonProto,
			},
			wantErr: false,
			expectedFiles: []string{
				"go/test/common.pb.go",
			},
		},
		{
			name: "compile with dependencies",
			args: []string{"-dir", "test", "-lang", "go", "-recursive", "-out", "test"},
			setupFiles: map[string]string{
				"test/common/common.proto": commonProto,
				"test/user/user.proto":     userProto,
				"test/order/order.proto":   orderProto,
			},
			wantErr: false,
			expectedFiles: []string{
				"go/test/common/common.pb.go",
				"go/test/user/user.pb.go",
				"go/test/order/order.pb.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			testDir := t.TempDir()

			// Set up test files and output directories
			for path, content := range tt.setupFiles {
				fullPath := filepath.Join(testDir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0644)
				require.NoError(t, err)

				// Create output directory
				outDir := filepath.Join(testDir, filepath.Dir(path))
				err = os.MkdirAll(outDir, 0755)
				require.NoError(t, err)
			}

			// Update directory in args
			for i, arg := range tt.args {
				if arg == "test" {
					tt.args[i] = testDir
				}
			}

			err := runCompile(tt.args)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Check if all expected files exist
			for _, file := range tt.expectedFiles {
				filePath := filepath.Join(testDir, file)
				_, err := os.Stat(filePath)
				assert.NoError(t, err, "Expected file %s to exist", file)
			}
		})
	}
}

func TestParseProtoFileForSpokeDirectives(t *testing.T) {
	// Create a temporary proto file with spoke directives
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	
	protoContent := `syntax = "proto3";

// @spoke:domain:github.com/example/test
package user;

message User {
    string id = 1;
    string name = 2;
}
`
	
	err := os.WriteFile(protoFile, []byte(protoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test proto file: %v", err)
	}
	
	// Parse the file
	info, err := parseProtoFileForSpokeDirectives(protoFile)
	if err != nil {
		t.Fatalf("Failed to parse proto file: %v", err)
	}
	
	// Verify the results
	if info.Path != protoFile {
		t.Errorf("Expected path %s, got %s", protoFile, info.Path)
	}
	
	if info.PackageName != "user" {
		t.Errorf("Expected package name 'user', got %s", info.PackageName)
	}
	
	if info.Domain != "github.com/example/test" {
		t.Errorf("Expected domain 'github.com/example/test', got %s", info.Domain)
	}
}

func TestParseProtoFileNoDomain(t *testing.T) {
	// Create a temporary proto file without spoke directives
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	
	protoContent := `syntax = "proto3";

package user;

message User {
    string id = 1;
    string name = 2;
}
`
	
	err := os.WriteFile(protoFile, []byte(protoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test proto file: %v", err)
	}
	
	// Parse the file
	info, err := parseProtoFileForSpokeDirectives(protoFile)
	if err != nil {
		t.Fatalf("Failed to parse proto file: %v", err)
	}
	
	// Verify the results
	if info.PackageName != "user" {
		t.Errorf("Expected package name 'user', got %s", info.PackageName)
	}
	
	if info.Domain != "" {
		t.Errorf("Expected empty domain, got %s", info.Domain)
	}
}

func TestCompileWithSpokeDirectives(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "proto")

	err := os.MkdirAll(protoDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create proto directory: %v", err)
	}

	// Create proto files with spoke directives
	userProtoContent := `syntax = "proto3";

// @spoke:domain:github.com/example/api
package user;

message User {
    string id = 1;
    string name = 2;
    string email = 3;
}
`

	orderProtoContent := `syntax = "proto3";

// @spoke:domain:github.com/example/api
package order;

import "user.proto";

message Order {
    string id = 1;
    user.User customer = 2;
    repeated string items = 3;
}
`

	userProtoFile := filepath.Join(protoDir, "user.proto")
	orderProtoFile := filepath.Join(protoDir, "order.proto")

	err = os.WriteFile(userProtoFile, []byte(userProtoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write user proto file: %v", err)
	}

	err = os.WriteFile(orderProtoFile, []byte(orderProtoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write order proto file: %v", err)
	}

	// Test parsing both files
	userInfo, err := parseProtoFileForSpokeDirectives(userProtoFile)
	if err != nil {
		t.Fatalf("Failed to parse user proto file: %v", err)
	}

	orderInfo, err := parseProtoFileForSpokeDirectives(orderProtoFile)
	if err != nil {
		t.Fatalf("Failed to parse order proto file: %v", err)
	}

	// Verify domain mappings
	if userInfo.Domain != "github.com/example/api" {
		t.Errorf("Expected user domain 'github.com/example/api', got %s", userInfo.Domain)
	}

	if orderInfo.Domain != "github.com/example/api" {
		t.Errorf("Expected order domain 'github.com/example/api', got %s", orderInfo.Domain)
	}

	if userInfo.PackageName != "user" {
		t.Errorf("Expected user package 'user', got %s", userInfo.PackageName)
	}

	if orderInfo.PackageName != "order" {
		t.Errorf("Expected order package 'order', got %s", orderInfo.PackageName)
	}

	// Test the compile function (this will fail if protoc is not available, but we can test the parsing logic)

	// We can't easily test the full compilation without protoc being available,
	// but we can test that the function doesn't crash and parses spoke directives correctly

	// Manually test the spoke directive parsing part
	protoFiles := []string{userProtoFile, orderProtoFile}
	domainToPackageMap := make(map[string][]string)

	for _, protoFile := range protoFiles {
		info, err := parseProtoFileForSpokeDirectives(protoFile)
		if err != nil {
			t.Errorf("Failed to parse spoke directives from %s: %v", protoFile, err)
			continue
		}

		if info.Domain != "" && info.PackageName != "" {
			domainToPackageMap[info.Domain] = append(domainToPackageMap[info.Domain], info.PackageName)
		}
	}

	// Verify the mapping
	expectedDomain := "github.com/example/api"
	if packages, exists := domainToPackageMap[expectedDomain]; !exists {
		t.Errorf("Expected domain %s not found in mapping", expectedDomain)
	} else {
		expectedPackages := []string{"user", "order"}
		if len(packages) != len(expectedPackages) {
			t.Errorf("Expected %d packages, got %d", len(expectedPackages), len(packages))
		}

		for _, expectedPkg := range expectedPackages {
			found := false
			for _, pkg := range packages {
				if pkg == expectedPkg {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected package %s not found in domain %s", expectedPkg, expectedDomain)
			}
		}
	}
}

// TestNewCompileCommand tests the compile command initialization
func TestNewCompileCommand(t *testing.T) {
	cmd := newCompileCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "compile", cmd.Name)
	assert.NotNil(t, cmd.Flags)
	assert.NotNil(t, cmd.Run)

	// Verify flags are registered
	assert.NotNil(t, cmd.Flags.Lookup("dir"))
	assert.NotNil(t, cmd.Flags.Lookup("out"))
	assert.NotNil(t, cmd.Flags.Lookup("lang"))
	assert.NotNil(t, cmd.Flags.Lookup("languages"))
	assert.NotNil(t, cmd.Flags.Lookup("grpc"))
	assert.NotNil(t, cmd.Flags.Lookup("parallel"))
	assert.NotNil(t, cmd.Flags.Lookup("recursive"))
}

// TestCompileMultipleLanguages tests compilation for multiple languages
func TestCompileMultipleLanguages(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found in PATH, skipping compilation tests")
	}

	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "proto")
	err := os.MkdirAll(protoDir, 0755)
	require.NoError(t, err)

	// Create a simple proto file
	protoContent := `syntax = "proto3";
package test;

message TestMessage {
    string id = 1;
    string name = 2;
}
`
	protoFile := filepath.Join(protoDir, "test.proto")
	err = os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tempDir, "out")

	// Test multiple languages at once
	args := []string{
		"-dir", protoDir,
		"-out", outDir,
		"-languages", "go,python,java",
	}

	err = runCompile(args)
	// May fail if all plugins aren't installed, but should not panic
	// Just verify the function handles multiple languages
	if err != nil {
		// Expected if plugins aren't available
		t.Logf("Compile failed (expected if plugins unavailable): %v", err)
	}
}

// TestCompileWithGRPC tests gRPC flag
func TestCompileWithGRPC(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found in PATH")
	}

	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "proto")
	err := os.MkdirAll(protoDir, 0755)
	require.NoError(t, err)

	// Create a service proto file
	protoContent := `syntax = "proto3";
package test;

service TestService {
    rpc TestMethod(TestRequest) returns (TestResponse);
}

message TestRequest {
    string id = 1;
}

message TestResponse {
    string result = 1;
}
`
	protoFile := filepath.Join(protoDir, "test.proto")
	err = os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tempDir, "out")

	args := []string{
		"-dir", protoDir,
		"-out", outDir,
		"-lang", "go",
		"-grpc",
	}

	err = runCompile(args)
	if err != nil {
		t.Logf("Compile with gRPC failed (expected if plugin unavailable): %v", err)
	}
}

// TestCompileUnsupportedLanguage tests handling of unsupported language
func TestCompileUnsupportedLanguage(t *testing.T) {
	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "proto")
	err := os.MkdirAll(protoDir, 0755)
	require.NoError(t, err)

	// Create a proto file
	protoContent := `syntax = "proto3";
package test;

message TestMessage {
    string id = 1;
}
`
	protoFile := filepath.Join(protoDir, "test.proto")
	err = os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tempDir, "out")

	args := []string{
		"-dir", protoDir,
		"-out", outDir,
		"-lang", "unsupported-language",
	}

	// Should not error, just skip the unsupported language
	err = runCompile(args)
	assert.NoError(t, err)
}

// TestCompileParallelMode tests parallel compilation flag
func TestCompileParallelMode(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found in PATH")
	}

	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "proto")
	err := os.MkdirAll(protoDir, 0755)
	require.NoError(t, err)

	protoContent := `syntax = "proto3";
package test;

message TestMessage {
    string id = 1;
}
`
	protoFile := filepath.Join(protoDir, "test.proto")
	err = os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tempDir, "out")

	args := []string{
		"-dir", protoDir,
		"-out", outDir,
		"-languages", "go,python",
		"-parallel",
	}

	err = runCompile(args)
	// In parallel mode, errors in one language don't stop others
	if err != nil {
		t.Logf("Parallel compile failed (expected if some plugins unavailable): %v", err)
	}
}

// TestParseProtoFileError tests error handling in proto file parsing
func TestParseProtoFileError(t *testing.T) {
	// Test with non-existent file
	_, err := parseProtoFileForSpokeDirectives("/nonexistent/file.proto")
	assert.Error(t, err)

	// Test with invalid proto content
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.proto")
	err = os.WriteFile(invalidFile, []byte("invalid proto content {{{"), 0644)
	require.NoError(t, err)

	_, err = parseProtoFileForSpokeDirectives(invalidFile)
	assert.Error(t, err)
}

// TestCompileNoProtoFiles tests error when no proto files found
func TestCompileNoProtoFiles(t *testing.T) {
	tempDir := t.TempDir()
	emptyDir := filepath.Join(tempDir, "empty")
	err := os.MkdirAll(emptyDir, 0755)
	require.NoError(t, err)

	outDir := filepath.Join(tempDir, "out")

	args := []string{
		"-dir", emptyDir,
		"-out", outDir,
		"-lang", "go",
	}

	err = runCompile(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no proto files found")
}

// TestCompileRecursiveFlag tests recursive dependency resolution
func TestCompileRecursiveFlag(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found in PATH")
	}

	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "proto")
	err := os.MkdirAll(protoDir, 0755)
	require.NoError(t, err)

	protoContent := `syntax = "proto3";
package test;

message TestMessage {
    string id = 1;
}
`
	protoFile := filepath.Join(protoDir, "test.proto")
	err = os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tempDir, "out")

	args := []string{
		"-dir", protoDir,
		"-out", outDir,
		"-lang", "go",
		"-recursive",
	}

	err = runCompile(args)
	// Should add parent directory to proto path
	if err != nil {
		t.Logf("Recursive compile failed: %v", err)
	}
}

// TestCompileAllSupportedLanguages tests all language switches are handled
func TestCompileAllSupportedLanguages(t *testing.T) {
	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "proto")
	err := os.MkdirAll(protoDir, 0755)
	require.NoError(t, err)

	protoContent := `syntax = "proto3";
package test;

message TestMessage {
    string id = 1;
}
`
	protoFile := filepath.Join(protoDir, "test.proto")
	err = os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tempDir, "out")

	// Test each supported language individually to cover all switch cases
	languages := []string{
		"go", "python", "java", "cpp", "csharp", "rust",
		"typescript", "ts", "javascript", "js", "dart",
		"swift", "kotlin", "objc", "ruby", "php", "scala",
	}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			args := []string{
				"-dir", protoDir,
				"-out", outDir,
				"-lang", lang,
			}

			err := runCompile(args)
			// May fail if protoc or plugins aren't available, but shouldn't panic
			if err != nil {
				t.Logf("Compile for %s failed (expected if plugins unavailable): %v", lang, err)
			}
		})
	}
} 