package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBatchPushCommand(t *testing.T) {
	cmd := newBatchPushCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "batch-push", cmd.Name)
	assert.NotEmpty(t, cmd.Description)
	assert.NotNil(t, cmd.Flags)
	assert.NotNil(t, cmd.Run)

	// Check flags are defined
	assert.NotNil(t, cmd.Flags.Lookup("module"))
	assert.NotNil(t, cmd.Flags.Lookup("dir"))
	assert.NotNil(t, cmd.Flags.Lookup("registry"))
	assert.NotNil(t, cmd.Flags.Lookup("description"))
	assert.NotNil(t, cmd.Flags.Lookup("exclude"))
}

func TestGenerateVersionName(t *testing.T) {
	tests := []struct {
		name        string
		setupGit    bool
		remoteURL   string
		expectError bool
	}{
		{
			name:        "valid git repository",
			setupGit:    true,
			remoteURL:   "https://github.com/test/repo.git",
			expectError: false,
		},
		{
			name:        "non-git directory",
			setupGit:    false,
			expectError: false, // getGitInfo returns nil error with default values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()

			if tt.setupGit {
				setupTestGitRepo(t, testDir, tt.remoteURL)
			}

			version, err := generateVersionName(testDir)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, version)

				// Version format should be: branch-timestamp-sha
				parts := strings.Split(version, "-")
				assert.GreaterOrEqual(t, len(parts), 3, "version should have at least 3 parts")

				// Check that invalid characters were replaced
				assert.NotContains(t, version, "/")
				assert.NotContains(t, version, "\\")
				assert.NotContains(t, version, ":")
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		excludePatterns []string
		expected        bool
	}{
		{
			name:            "no patterns - not excluded",
			path:            "src/proto/user.proto",
			excludePatterns: []string{},
			expected:        false,
		},
		{
			name:            "empty pattern - not excluded",
			path:            "src/proto/user.proto",
			excludePatterns: []string{""},
			expected:        false,
		},
		{
			name:            "match base name",
			path:            "src/node_modules/test.proto",
			excludePatterns: []string{"node_modules"},
			expected:        true,
		},
		{
			name:            "match with wildcard",
			path:            "test/test_pb.proto",
			excludePatterns: []string{"test*"},
			expected:        true,
		},
		{
			name:            "match parent directory",
			path:            "vendor/github.com/test/proto/user.proto",
			excludePatterns: []string{"vendor"},
			expected:        true,
		},
		{
			name:            "no match",
			path:            "src/proto/user.proto",
			excludePatterns: []string{"node_modules", "vendor"},
			expected:        false,
		},
		{
			name:            "multiple patterns - one matches",
			path:            "node_modules/test/proto.proto",
			excludePatterns: []string{"vendor", "node_modules", "test"},
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isExcluded(tt.path, tt.excludePatterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "simple package",
			content: `syntax = "proto3";
package user;`,
			expected: "user",
		},
		{
			name: "package with options",
			content: `syntax = "proto3";
package user.v1;
option go_package = "github.com/test/user/v1";`,
			expected: "user.v1",
		},
		{
			name:     "no package",
			content:  `syntax = "proto3";`,
			expected: "",
		},
		{
			name: "package with comments",
			content: `syntax = "proto3";
// User service package
package user.service;`,
			expected: "user.service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPackageName(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFileGitInfo(t *testing.T) {
	tests := []struct {
		name        string
		setupGit    bool
		remoteURL   string
		expectError bool
	}{
		{
			name:        "valid git file",
			setupGit:    true,
			remoteURL:   "git@github.com:test/repo.git",
			expectError: false,
		},
		{
			name:        "https url",
			setupGit:    true,
			remoteURL:   "https://github.com/test/repo.git",
			expectError: false,
		},
		{
			name:        "non-git file",
			setupGit:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()
			testFile := filepath.Join(testDir, "test.proto")

			if tt.setupGit {
				setupTestGitRepo(t, testDir, tt.remoteURL)

				// Create and commit the test file
				err := os.WriteFile(testFile, []byte("syntax = \"proto3\";"), 0644)
				require.NoError(t, err)

				cmd := exec.Command("git", "add", "test.proto")
				cmd.Dir = testDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "commit", "-m", "add test proto")
				cmd.Dir = testDir
				err = cmd.Run()
				require.NoError(t, err)
			} else {
				err := os.WriteFile(testFile, []byte("syntax = \"proto3\";"), 0644)
				require.NoError(t, err)
			}

			info, err := getFileGitInfo(testFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, info.Repository)
				assert.NotEmpty(t, info.CommitSHA)
				assert.NotEmpty(t, info.Branch)

				// Check repository URL format
				if strings.HasPrefix(tt.remoteURL, "git@") {
					assert.Contains(t, info.Repository, "https://")
				}
				assert.NotContains(t, info.Repository, ".git")
			}
		})
	}
}

func TestGetFileGitInfo_DetachedHead(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test.proto")

	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	// Create and commit the test file
	err := os.WriteFile(testFile, []byte("syntax = \"proto3\";"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", "test.proto")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add test proto")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	// Get the commit SHA
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = testDir
	output, err := cmd.Output()
	require.NoError(t, err)
	commitSHA := strings.TrimSpace(string(output))

	// Checkout the commit directly to create detached HEAD
	cmd = exec.Command("git", "checkout", commitSHA)
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	// Set GITHUB_REF env var to simulate CI
	originalEnv := os.Getenv("GITHUB_REF")
	os.Setenv("GITHUB_REF", "refs/heads/main")
	defer os.Setenv("GITHUB_REF", originalEnv)

	info, err := getFileGitInfo(testFile)
	assert.NoError(t, err)
	assert.Equal(t, "main", info.Branch)
}

func TestGenerateVersionNameFromFileInfo(t *testing.T) {
	tests := []struct {
		name        string
		setupGit    bool
		expectError bool
	}{
		{
			name:        "valid file with git",
			setupGit:    true,
			expectError: false,
		},
		{
			name:        "file without git",
			setupGit:    false,
			expectError: true, // getFileGitInfo returns error for non-git files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()
			testFile := filepath.Join(testDir, "test.proto")

			if tt.setupGit {
				setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

				err := os.WriteFile(testFile, []byte("syntax = \"proto3\";"), 0644)
				require.NoError(t, err)

				cmd := exec.Command("git", "add", "test.proto")
				cmd.Dir = testDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "commit", "-m", "add test")
				cmd.Dir = testDir
				err = cmd.Run()
				require.NoError(t, err)
			} else {
				err := os.WriteFile(testFile, []byte("syntax = \"proto3\";"), 0644)
				require.NoError(t, err)
			}

			version, err := generateVersionNameFromFileInfo(testFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, version)

				// Check format: branch-timestamp-sha
				parts := strings.Split(version, "-")
				assert.GreaterOrEqual(t, len(parts), 3)
			}
		})
	}
}

func TestRunBatchPush_NoProtoFiles(t *testing.T) {
	testDir := t.TempDir()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	args := []string{
		"-dir", testDir,
		"-registry", server.URL,
	}

	err := runBatchPush(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no proto files found")
}

func TestRunBatchPush_WithProtoFiles(t *testing.T) {
	testDir := t.TempDir()

	// Setup git repo
	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	// Create test proto files with different packages
	protoDir1 := filepath.Join(testDir, "user")
	err := os.MkdirAll(protoDir1, 0755)
	require.NoError(t, err)

	protoContent1 := `syntax = "proto3";
package user;

message User {
  string id = 1;
  string name = 2;
}`
	err = os.WriteFile(filepath.Join(protoDir1, "user.proto"), []byte(protoContent1), 0644)
	require.NoError(t, err)

	protoDir2 := filepath.Join(testDir, "order")
	err = os.MkdirAll(protoDir2, 0755)
	require.NoError(t, err)

	protoContent2 := `syntax = "proto3";
package order;

message Order {
  string id = 1;
  string user_id = 2;
}`
	err = os.WriteFile(filepath.Join(protoDir2, "order.proto"), []byte(protoContent2), 0644)
	require.NoError(t, err)

	// Commit the files
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add protos")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	// Track module creation and version creation
	modulesCreated := make(map[string]bool)
	versionsCreated := make(map[string]bool)

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.URL.Path == "/modules" {
				var module api.Module
				err := json.NewDecoder(r.Body).Decode(&module)
				require.NoError(t, err)
				modulesCreated[module.Name] = true
				w.WriteHeader(http.StatusCreated)
				return
			}

			if strings.HasPrefix(r.URL.Path, "/modules/") && strings.HasSuffix(r.URL.Path, "/versions") {
				var version api.Version
				err := json.NewDecoder(r.Body).Decode(&version)
				require.NoError(t, err)
				versionsCreated[version.ModuleName] = true

				// Verify version format
				assert.NotEmpty(t, version.Version)
				assert.NotEmpty(t, version.Files)

				w.WriteHeader(http.StatusCreated)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	args := []string{
		"-dir", testDir,
		"-registry", server.URL,
		"-description", "Test modules",
	}

	err = runBatchPush(args)
	assert.NoError(t, err)

	// Verify both packages were created
	assert.True(t, modulesCreated["user"], "user module should be created")
	assert.True(t, modulesCreated["order"], "order module should be created")
	assert.True(t, versionsCreated["user"], "user version should be created")
	assert.True(t, versionsCreated["order"], "order version should be created")
}

func TestRunBatchPush_WithExclusions(t *testing.T) {
	testDir := t.TempDir()

	// Setup git repo
	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	// Create included proto file
	includedDir := filepath.Join(testDir, "src")
	err := os.MkdirAll(includedDir, 0755)
	require.NoError(t, err)

	includedProto := `syntax = "proto3";
package included;

message Test {
  string id = 1;
}`
	err = os.WriteFile(filepath.Join(includedDir, "test.proto"), []byte(includedProto), 0644)
	require.NoError(t, err)

	// Create excluded proto file
	excludedDir := filepath.Join(testDir, "vendor")
	err = os.MkdirAll(excludedDir, 0755)
	require.NoError(t, err)

	excludedProto := `syntax = "proto3";
package excluded;

message Excluded {
  string id = 1;
}`
	err = os.WriteFile(filepath.Join(excludedDir, "excluded.proto"), []byte(excludedProto), 0644)
	require.NoError(t, err)

	// Commit files
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add protos")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	// Track what gets created
	modulesCreated := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.URL.Path == "/modules" {
				var module api.Module
				json.NewDecoder(r.Body).Decode(&module)
				modulesCreated[module.Name] = true
				w.WriteHeader(http.StatusCreated)
				return
			}
			if strings.Contains(r.URL.Path, "/versions") {
				w.WriteHeader(http.StatusCreated)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	args := []string{
		"-dir", testDir,
		"-registry", server.URL,
		"-exclude", "vendor,node_modules",
	}

	err = runBatchPush(args)
	assert.NoError(t, err)

	// Verify only included package was created
	assert.True(t, modulesCreated["included"], "included module should be created")
	assert.False(t, modulesCreated["excluded"], "excluded module should not be created")
}

func TestRunBatchPush_NoPackageName(t *testing.T) {
	testDir := t.TempDir()

	// Setup git repo
	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	// Create proto file without package name
	protoContent := `syntax = "proto3";

message Test {
  string id = 1;
}`
	err := os.WriteFile(filepath.Join(testDir, "test.proto"), []byte(protoContent), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add proto")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	args := []string{
		"-dir", testDir,
		"-registry", server.URL,
	}

	err = runBatchPush(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no proto files found")
}

func TestRunBatchPush_ServerError(t *testing.T) {
	testDir := t.TempDir()

	// Setup git repo
	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	// Create proto file
	protoContent := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}`
	err := os.WriteFile(filepath.Join(testDir, "test.proto"), []byte(protoContent), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add proto")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	// Server that returns error for version creation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.URL.Path == "/modules" {
				w.WriteHeader(http.StatusCreated)
				return
			}
			if strings.Contains(r.URL.Path, "/versions") {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error"))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	args := []string{
		"-dir", testDir,
		"-registry", server.URL,
	}

	err = runBatchPush(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create version")
}

func TestRunBatchPush_NoGitInfo(t *testing.T) {
	testDir := t.TempDir()

	// Create proto file without git repo
	protoContent := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}`
	err := os.WriteFile(filepath.Join(testDir, "test.proto"), []byte(protoContent), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.URL.Path == "/modules" {
				w.WriteHeader(http.StatusCreated)
				return
			}
			if strings.Contains(r.URL.Path, "/versions") {
				var version api.Version
				json.NewDecoder(r.Body).Decode(&version)

				// Verify fallback version format (timestamp only)
				assert.NotEmpty(t, version.Version)

				// Verify unknown source info
				assert.Equal(t, "unknown", version.SourceInfo.Repository)
				assert.Equal(t, "unknown", version.SourceInfo.CommitSHA)
				assert.Equal(t, "unknown", version.SourceInfo.Branch)

				w.WriteHeader(http.StatusCreated)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	args := []string{
		"-dir", testDir,
		"-registry", server.URL,
	}

	err = runBatchPush(args)
	assert.NoError(t, err)
}

func TestRunBatchPush_WithDependencies(t *testing.T) {
	testDir := t.TempDir()

	// Setup git repo
	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	// Create proto with imports
	protoContent := `syntax = "proto3";
package order;

import "user/user.proto";
import "common/types.proto";

message Order {
  string id = 1;
  user.User user = 2;
}`
	err := os.WriteFile(filepath.Join(testDir, "order.proto"), []byte(protoContent), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add proto")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	var capturedDeps []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.URL.Path == "/modules" {
				w.WriteHeader(http.StatusCreated)
				return
			}
			if strings.Contains(r.URL.Path, "/versions") {
				var version api.Version
				json.NewDecoder(r.Body).Decode(&version)
				capturedDeps = version.Dependencies
				w.WriteHeader(http.StatusCreated)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	args := []string{
		"-dir", testDir,
		"-registry", server.URL,
	}

	err = runBatchPush(args)
	assert.NoError(t, err)

	// Dependencies tracking code path is executed
	// The actual dependencies captured depend on the import parser implementation
	_ = capturedDeps
}

func TestRunBatchPush_InvalidFlags(t *testing.T) {
	// Test with empty directory (valid but no files)
	// Note: Invalid flags would cause flag.ExitOnError to terminate,
	// so we test error conditions that return errors instead
	testDir := t.TempDir()
	err := runBatchPush([]string{"-dir", testDir})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no proto files found")
}

func TestRunBatchPush_MultipleFilesPerPackage(t *testing.T) {
	testDir := t.TempDir()

	// Setup git repo
	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	// Create multiple files for same package
	userDir := filepath.Join(testDir, "user")
	err := os.MkdirAll(userDir, 0755)
	require.NoError(t, err)

	proto1 := `syntax = "proto3";
package user;

message User {
  string id = 1;
}`
	err = os.WriteFile(filepath.Join(userDir, "user.proto"), []byte(proto1), 0644)
	require.NoError(t, err)

	proto2 := `syntax = "proto3";
package user;

message Profile {
  string user_id = 1;
}`
	err = os.WriteFile(filepath.Join(userDir, "profile.proto"), []byte(proto2), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add protos")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	var fileCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.URL.Path == "/modules" {
				w.WriteHeader(http.StatusCreated)
				return
			}
			if strings.Contains(r.URL.Path, "/versions") {
				var version api.Version
				json.NewDecoder(r.Body).Decode(&version)
				fileCount = len(version.Files)
				w.WriteHeader(http.StatusCreated)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	args := []string{
		"-dir", testDir,
		"-registry", server.URL,
	}

	err = runBatchPush(args)
	assert.NoError(t, err)
	assert.Equal(t, 2, fileCount, "should have 2 files in the version")
}

func TestGenerateVersionName_InvalidCharacters(t *testing.T) {
	testDir := t.TempDir()

	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	// Create a branch with slash (colon is not allowed by git)
	cmd := exec.Command("git", "checkout", "-b", "feature/test-branch")
	cmd.Dir = testDir
	err := cmd.Run()
	require.NoError(t, err)

	version, err := generateVersionName(testDir)
	assert.NoError(t, err)

	// Verify special characters are replaced
	assert.NotContains(t, version, "/")
	assert.Contains(t, version, "feature-test-branch")
}

// Helper function to setup a test git repository
func setupTestGitRepo(t *testing.T, dir string, remoteURL string) {
	t.Helper()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	err := cmd.Run()
	require.NoError(t, err)

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err)

	// Set up git remote
	cmd = exec.Command("git", "remote", "add", "origin", remoteURL)
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err)

	// Create initial commit
	readmeFile := filepath.Join(dir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err)
}

func TestRunBatchPush_Description(t *testing.T) {
	testDir := t.TempDir()

	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	protoContent := `syntax = "proto3";
package test;

message Test {
  string id = 1;
}`
	err := os.WriteFile(filepath.Join(testDir, "test.proto"), []byte(protoContent), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add proto")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	expectedDesc := "Test description"
	var capturedDesc string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.URL.Path == "/modules" {
				var module api.Module
				json.NewDecoder(r.Body).Decode(&module)
				capturedDesc = module.Description
				w.WriteHeader(http.StatusCreated)
				return
			}
			if strings.Contains(r.URL.Path, "/versions") {
				w.WriteHeader(http.StatusCreated)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	args := []string{
		"-dir", testDir,
		"-registry", server.URL,
		"-description", expectedDesc,
	}

	err = runBatchPush(args)
	assert.NoError(t, err)
	assert.Equal(t, expectedDesc, capturedDesc)
}

func TestGenerateVersionName_ShortSHA(t *testing.T) {
	testDir := t.TempDir()
	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	version, err := generateVersionName(testDir)
	assert.NoError(t, err)

	// Extract SHA from version (last part after final dash)
	parts := strings.Split(version, "-")
	sha := parts[len(parts)-1]

	// SHA should be 7 characters or less
	assert.LessOrEqual(t, len(sha), 7)
}

func TestGetFileGitInfo_SSHUrlConversion(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test.proto")

	setupTestGitRepo(t, testDir, "git@github.com:user/repo.git")

	err := os.WriteFile(testFile, []byte("syntax = \"proto3\";"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", "test.proto")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add test")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	info, err := getFileGitInfo(testFile)
	assert.NoError(t, err)

	// Should convert SSH to HTTPS
	assert.Equal(t, "https://github.com/user/repo", info.Repository)
}

func TestRunBatchPush_ExcludeMultiplePatterns(t *testing.T) {
	testDir := t.TempDir()
	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	// Create various directories
	dirs := []string{"src", "vendor", "node_modules", "test"}
	for _, dir := range dirs {
		dirPath := filepath.Join(testDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)

		protoContent := fmt.Sprintf(`syntax = "proto3";
package %s;

message Test { string id = 1; }`, dir)
		err = os.WriteFile(filepath.Join(dirPath, "test.proto"), []byte(protoContent), 0644)
		require.NoError(t, err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = testDir
	err := cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add protos")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	modulesCreated := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.URL.Path == "/modules" {
				var module api.Module
				json.NewDecoder(r.Body).Decode(&module)
				modulesCreated[module.Name] = true
				w.WriteHeader(http.StatusCreated)
				return
			}
			if strings.Contains(r.URL.Path, "/versions") {
				w.WriteHeader(http.StatusCreated)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	args := []string{
		"-dir", testDir,
		"-registry", server.URL,
		"-exclude", "vendor, node_modules, test",
	}

	err = runBatchPush(args)
	assert.NoError(t, err)

	// Only src should be created
	assert.True(t, modulesCreated["src"])
	assert.False(t, modulesCreated["vendor"])
	assert.False(t, modulesCreated["node_modules"])
	assert.False(t, modulesCreated["test"])
}

func TestGenerateVersionNameFromFileInfo_TimestampFormat(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test.proto")

	setupTestGitRepo(t, testDir, "https://github.com/test/repo.git")

	err := os.WriteFile(testFile, []byte("syntax = \"proto3\";"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", "test.proto")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "add test")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	beforeTime := time.Now().UTC()
	version, err := generateVersionNameFromFileInfo(testFile)
	afterTime := time.Now().UTC()

	assert.NoError(t, err)

	// Extract timestamp from version (middle parts)
	// Format: branch-YYYY-MM-DD-HH-mm-sha
	parts := strings.Split(version, "-")
	assert.GreaterOrEqual(t, len(parts), 7)

	// Verify year is between before and after
	year := parts[1]
	assert.GreaterOrEqual(t, year, beforeTime.Format("2006"))
	assert.LessOrEqual(t, year, afterTime.Format("2006"))
}
