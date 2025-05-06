package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileCommand(t *testing.T) {
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