package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/platinummonkey/spoke/pkg/api"
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

func TestParseProtoImports(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "no imports",
			content:  `syntax = "proto3"; package test;`,
			expected: []string{},
		},
		{
			name:     "single import",
			content:  `syntax = "proto3"; package test; import "common/common.proto";`,
			expected: []string{"common/common.proto"},
		},
		{
			name:     "multiple imports",
			content:  `syntax = "proto3"; package test; import "common/common.proto"; import "user/user.proto";`,
			expected: []string{"common/common.proto", "user/user.proto"},
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