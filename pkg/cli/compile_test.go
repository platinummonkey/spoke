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
				"test/common.pb.go",
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
				"test/common/common.pb.go",
				"test/user/user.pb.go",
				"test/order/order.pb.go",
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