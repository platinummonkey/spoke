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

func TestPullCommand(t *testing.T) {
	// Create test proto files
	commonProto := `syntax = "proto3";
package common;

message Metadata {
  string created_at = 1;
  string updated_at = 2;
}`

	userProto := `syntax = "proto3";
package user;
import "common/common.proto";

message User {
  string id = 1;
  common.Metadata metadata = 2;
}`

	orderProto := `syntax = "proto3";
package order;
import "common/common.proto";
import "user/user.proto";

message Order {
  string id = 1;
  user.User user = 2;
  common.Metadata metadata = 3;
}`

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/modules/common/versions/v1.0.0":
			version := api.Version{
				ModuleName: "common",
				Version:    "v1.0.0",
				Files: []api.File{
					{
						Path:    "common.proto",
						Content: commonProto,
					},
				},
			}
			json.NewEncoder(w).Encode(version)
		case "/modules/user/versions/v1.0.0":
			version := api.Version{
				ModuleName: "user",
				Version:    "v1.0.0",
				Files: []api.File{
					{
						Path:    "user.proto",
						Content: userProto,
					},
				},
				Dependencies: []string{"common@v1.0.0"},
			}
			json.NewEncoder(w).Encode(version)
		case "/modules/order/versions/v1.0.0":
			version := api.Version{
				ModuleName: "order",
				Version:    "v1.0.0",
				Files: []api.File{
					{
						Path:    "order.proto",
						Content: orderProto,
					},
				},
				Dependencies: []string{"common@v1.0.0", "user@v1.0.0"},
			}
			json.NewEncoder(w).Encode(version)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tests := []struct {
		name           string
		args           []string
		wantErr        bool
		expectedFiles  []string
		checkContents  bool
	}{
		{
			name:    "missing module",
			args:    []string{"-version", "v1.0.0"},
			wantErr: true,
		},
		{
			name:    "missing version",
			args:    []string{"-module", "test"},
			wantErr: true,
		},
		{
			name:          "pull common module",
			args:          []string{"-module", "common", "-version", "v1.0.0", "-registry", server.URL},
			wantErr:       false,
			expectedFiles: []string{"common/common.proto"},
			checkContents: true,
		},
		{
			name:          "pull order module with dependencies",
			args:          []string{"-module", "order", "-version", "v1.0.0", "-registry", server.URL, "-recursive"},
			wantErr:       false,
			expectedFiles: []string{
				"order/order.proto",
				"user/user.proto",
				"common/common.proto",
			},
			checkContents: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			testDir := t.TempDir()
			tt.args = append(tt.args, "-dir", testDir)

			err := runPull(tt.args)
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

				if tt.checkContents {
					content, err := os.ReadFile(filePath)
					require.NoError(t, err)
					assert.NotEmpty(t, content, "File %s should not be empty", file)
				}
			}
		})
	}
} 