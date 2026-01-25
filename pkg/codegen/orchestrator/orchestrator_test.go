package orchestrator

import (
	"context"
	"testing"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrchestrator(t *testing.T) {
	config := DefaultConfig()
	orch, err := NewOrchestrator(config)
	require.NoError(t, err)
	require.NotNil(t, orch)
	defer orch.Close()

	assert.NotNil(t, orch.languageRegistry)
	assert.NotNil(t, orch.dockerRunner)
	assert.NotNil(t, orch.packageRegistry)
}

func TestNewOrchestrator_NilConfig(t *testing.T) {
	orch, err := NewOrchestrator(nil)
	require.NoError(t, err)
	require.NotNil(t, orch)
	defer orch.Close()

	assert.NotNil(t, orch.config)
	assert.Equal(t, DefaultConfig().MaxParallelWorkers, orch.config.MaxParallelWorkers)
}

func TestValidateRequest(t *testing.T) {
	orch, err := NewOrchestrator(nil)
	require.NoError(t, err)
	defer orch.Close()

	tests := []struct {
		name    string
		req     *CompileRequest
		wantErr bool
		errType error
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "missing module name",
			req: &CompileRequest{
				Version:    "v1.0.0",
				ProtoFiles: []codegen.ProtoFile{{Path: "test.proto"}},
			},
			wantErr: true,
		},
		{
			name: "missing version",
			req: &CompileRequest{
				ModuleName: "test",
				ProtoFiles: []codegen.ProtoFile{{Path: "test.proto"}},
			},
			wantErr: true,
		},
		{
			name: "missing proto files",
			req: &CompileRequest{
				ModuleName: "test",
				Version:    "v1.0.0",
			},
			wantErr: true,
			errType: ErrNoProtoFiles,
		},
		{
			name: "invalid language",
			req: &CompileRequest{
				ModuleName: "test",
				Version:    "v1.0.0",
				Language:   "invalid",
				ProtoFiles: []codegen.ProtoFile{{Path: "test.proto"}},
			},
			wantErr: true,
			errType: ErrLanguageNotSupported,
		},
		{
			name: "valid request",
			req: &CompileRequest{
				ModuleName: "test",
				Version:    "v1.0.0",
				Language:   "go",
				ProtoFiles: []codegen.ProtoFile{{Path: "test.proto"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := orch.validateRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildProtocFlags(t *testing.T) {
	orch, err := NewOrchestrator(nil)
	require.NoError(t, err)
	defer orch.Close()

	tests := []struct {
		name        string
		language    string
		includeGRPC bool
		wantContain []string
	}{
		{
			name:        "go without gRPC",
			language:    "go",
			includeGRPC: false,
			wantContain: []string{"--go_out=/output"},
		},
		{
			name:        "go with gRPC",
			language:    "go",
			includeGRPC: true,
			wantContain: []string{"--go_out=/output", "--go-grpc_out=/output"},
		},
		{
			name:        "python without gRPC",
			language:    "python",
			includeGRPC: false,
			wantContain: []string{"--python_out=/output"},
		},
		{
			name:        "python with gRPC",
			language:    "python",
			includeGRPC: true,
			wantContain: []string{"--python_out=/output", "--grpc_python_out=/output"},
		},
		{
			name:        "java with gRPC",
			language:    "java",
			includeGRPC: true,
			wantContain: []string{"--java_out=/output", "--grpc-java_out=/output"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			langSpec, err := orch.languageRegistry.Get(tt.language)
			require.NoError(t, err)

			req := &CompileRequest{
				IncludeGRPC: tt.includeGRPC,
			}

			flags := orch.buildProtocFlags(langSpec, req)

			for _, want := range tt.wantContain {
				assert.Contains(t, flags, want)
			}
		})
	}
}

func TestCompileSingle_Validation(t *testing.T) {
	orch, err := NewOrchestrator(nil)
	require.NoError(t, err)
	defer orch.Close()

	ctx := context.Background()

	// Test invalid request
	_, err = orch.CompileSingle(ctx, nil)
	assert.Error(t, err)

	// Test missing proto files
	_, err = orch.CompileSingle(ctx, &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		Language:   "go",
	})
	assert.ErrorIs(t, err, ErrNoProtoFiles)

	// Test unsupported language
	_, err = orch.CompileSingle(ctx, &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		Language:   "unsupported",
		ProtoFiles: []codegen.ProtoFile{{Path: "test.proto"}},
	})
	assert.ErrorIs(t, err, ErrLanguageNotSupported)
}

func TestCompileAll_Validation(t *testing.T) {
	orch, err := NewOrchestrator(nil)
	require.NoError(t, err)
	defer orch.Close()

	ctx := context.Background()

	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		ProtoFiles: []codegen.ProtoFile{{Path: "test.proto"}},
	}

	// Test empty languages list
	_, err = orch.CompileAll(ctx, req, []string{})
	assert.Error(t, err)

	// Test unsupported language
	_, err = orch.CompileAll(ctx, req, []string{"unsupported"})
	assert.ErrorIs(t, err, ErrLanguageNotSupported)

	// Test mixed valid and invalid languages
	_, err = orch.CompileAll(ctx, req, []string{"go", "unsupported"})
	assert.ErrorIs(t, err, ErrLanguageNotSupported)
}

func TestGetStatus_NotFound(t *testing.T) {
	orch, err := NewOrchestrator(nil)
	require.NoError(t, err)
	defer orch.Close()

	ctx := context.Background()

	_, err = orch.GetStatus(ctx, "nonexistent-job-id")
	assert.ErrorIs(t, err, ErrJobNotFound)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 5, config.MaxParallelWorkers)
	assert.True(t, config.EnableCache)
	assert.True(t, config.EnableMetrics)
	assert.Equal(t, "v2", config.CodeGenVersion)
	assert.Equal(t, 300, config.CompilationTimeout)
}

func TestClose(t *testing.T) {
	orch, err := NewOrchestrator(nil)
	require.NoError(t, err)

	err = orch.Close()
	assert.NoError(t, err)
}
