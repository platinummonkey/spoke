package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCommand(t *testing.T) {
	// Create test proto files
	validProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/pkg/cli/testdata/test";

message TestMessage {
    string id = 1;
    int32 value = 2;
}`

	invalidProto := `syntax = "proto3";
package test;
option go_package = "github.com/platinummonkey/spoke/pkg/cli/testdata/test";

message TestMessage {
    string id = 1;
    int32 value = 1; // Duplicate field number
}`

	missingSyntaxProto := `package test;
option go_package = "github.com/platinummonkey/spoke/pkg/cli/testdata/test";

message TestMessage {
    string id = 1;
}`

	tests := []struct {
		name       string
		args       []string
		setupFiles map[string]string
		wantErr    bool
	}{
		{
			name:    "missing directory",
			args:    []string{},
			wantErr: true,
		},
		{
			name: "valid proto file",
			args: []string{"-dir", "test"},
			setupFiles: map[string]string{
				"test/test.proto": validProto,
			},
			wantErr: false,
		},
		{
			name: "invalid proto file - duplicate field number",
			args: []string{"-dir", "test"},
			setupFiles: map[string]string{
				"test/test.proto": invalidProto,
			},
			wantErr: true,
		},
		{
			name: "invalid proto file - missing syntax",
			args: []string{"-dir", "test"},
			setupFiles: map[string]string{
				"test/test.proto": missingSyntaxProto,
			},
			wantErr: true,
		},
		{
			name: "multiple valid proto files",
			args: []string{"-dir", "test", "-recursive"},
			setupFiles: map[string]string{
				"test/valid1.proto": `syntax = "proto3";
package test1;
option go_package = "github.com/platinummonkey/spoke/pkg/cli/testdata/test1";

message TestMessage {
    string id = 1;
    int32 value = 2;
}`,
				"test/valid2.proto": `syntax = "proto3";
package test2;
option go_package = "github.com/platinummonkey/spoke/pkg/cli/testdata/test2";

message TestMessage {
    string id = 1;
    int32 value = 2;
}`,
			},
			wantErr: false,
		},
		{
			name: "mixed valid and invalid proto files",
			args: []string{"-dir", "test", "-recursive"},
			setupFiles: map[string]string{
				"test/valid.proto":   validProto,
				"test/invalid.proto": invalidProto,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			testDir := t.TempDir()

			// Set up test files
			for path, content := range tt.setupFiles {
				fullPath := filepath.Join(testDir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0644)
				require.NoError(t, err)
			}

			// Update directory in args
			for i, arg := range tt.args {
				if arg == "test" {
					tt.args[i] = testDir
				}
			}

			err := runValidate(tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
} 