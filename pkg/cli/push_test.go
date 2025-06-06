package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushCommand(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/modules":
			// Handle module creation
			if r.Method == http.MethodPost {
				var module api.Module
				err := json.NewDecoder(r.Body).Decode(&module)
				require.NoError(t, err)
				w.WriteHeader(http.StatusOK)
				return
			}
		case "/modules/test/versions":
			// Handle version creation
			if r.Method == http.MethodPost {
				var version api.Version
				err := json.NewDecoder(r.Body).Decode(&version)
				require.NoError(t, err)
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create test proto file
	testDir := t.TempDir()
	protoContent := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/pkg/cli/testdata/test";

message TestMessage {
  string id = 1;
  common.Metadata metadata = 2;
}`
	err := os.WriteFile(filepath.Join(testDir, "test.proto"), []byte(protoContent), 0644)
	require.NoError(t, err)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "missing module",
			args:    []string{"-version", "v1.0.0", "-dir", testDir},
			wantErr: true,
		},
		{
			name:    "missing version",
			args:    []string{"-module", "test", "-dir", testDir},
			wantErr: true,
		},
		{
			name:    "successful push",
			args:    []string{"-module", "test", "-version", "v1.0.0", "-dir", testDir, "-registry", server.URL},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runPush(tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetGitInfo(t *testing.T) {
	tests := []struct {
		name           string
		remoteURL      string
		expectedRepo   string
		setupGit       bool
	}{
		{
			name:           "ssh url",
			remoteURL:      "git@github.com:test/repo.git",
			expectedRepo:   "https://github.com/test/repo",
			setupGit:       true,
		},
		{
			name:           "https url",
			remoteURL:      "https://github.com/test/repo.git",
			expectedRepo:   "https://github.com/test/repo",
			setupGit:       true,
		},
		{
			name:           "non-git directory",
			remoteURL:      "",
			expectedRepo:   "unknown",
			setupGit:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			testDir := t.TempDir()

			if tt.setupGit {
				// Initialize git repository
				cmd := exec.Command("git", "init")
				cmd.Dir = testDir
				err := cmd.Run()
				require.NoError(t, err)

				// Set up git remote
				cmd = exec.Command("git", "remote", "add", "origin", tt.remoteURL)
				cmd.Dir = testDir
				err = cmd.Run()
				require.NoError(t, err)

				// Create a test file and commit it
				testFile := filepath.Join(testDir, "test.txt")
				err = os.WriteFile(testFile, []byte("test"), 0644)
				require.NoError(t, err)

				cmd = exec.Command("git", "add", "test.txt")
				cmd.Dir = testDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "commit", "-m", "test commit")
				cmd.Dir = testDir
				err = cmd.Run()
				require.NoError(t, err)
			}

			// Test git info collection
			info, err := getGitInfo(testDir)
			require.NoError(t, err)

			// Verify repository URL
			assert.Equal(t, tt.expectedRepo, info.Repository)

			if tt.setupGit {
				// Verify commit SHA is not empty
				assert.NotEmpty(t, info.CommitSHA)

				// Verify branch is not empty
				assert.NotEmpty(t, info.Branch)
			} else {
				// Verify default values for non-git directory
				assert.Equal(t, "unknown", info.CommitSHA)
				assert.Equal(t, "unknown", info.Branch)
			}
		})
	}
}

func TestParseProtoImports(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []protobuf.ProtoImport
	}{
		{
			name:     "no imports",
			content:  `syntax = "proto3"; package test;`,
			expected: []protobuf.ProtoImport{},
		},
		{
			name: "single import",
			content: `syntax = "proto3";
import "google/protobuf/timestamp.proto";
package test;`,
			expected: []protobuf.ProtoImport{
				{Path: "google/protobuf/timestamp.proto"},
			},
		},
		{
			name: "multiple imports",
			content: `syntax = "proto3";
import "google/protobuf/timestamp.proto";
import public "common/types.proto";
package test;`,
			expected: []protobuf.ProtoImport{
				{Path: "google/protobuf/timestamp.proto"},
				{Path: "common/types.proto", Public: true},
			},
		},
		{
			name: "import with weak",
			content: `syntax = "proto3";
import weak "deprecated.proto";
package test;`,
			expected: []protobuf.ProtoImport{
				{Path: "deprecated.proto", Weak: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports := parseProtoImports(tt.content)
			assert.Equal(t, tt.expected, imports)
		})
	}
}

func TestExtractModuleFromImport(t *testing.T) {
	tests := []struct {
		name     string
		import_  string
		expected string
	}{
		{
			name:     "simple import",
			import_:  "common/common.proto",
			expected: "common",
		},
		{
			name:     "nested import",
			import_:  "user/v1/user.proto",
			expected: "user",
		},
		{
			name:     "empty import",
			import_:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module := extractModuleFromImport(tt.import_)
			assert.Equal(t, tt.expected, module)
		})
	}
} 