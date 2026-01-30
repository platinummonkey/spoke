package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/artifacts"
	"github.com/platinummonkey/spoke/pkg/codegen/cache"
	pkgconfig "github.com/platinummonkey/spoke/pkg/codegen/config"
	"github.com/platinummonkey/spoke/pkg/codegen/docker"
	"github.com/platinummonkey/spoke/pkg/codegen/languages"
	"github.com/platinummonkey/spoke/pkg/codegen/packages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Docker Runner
type mockDockerRunner struct {
	executeFunc func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error)
	pullFunc    func(ctx context.Context, image string) error
	closeFunc   func() error
}

func (m *mockDockerRunner) Execute(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, req)
	}
	return &docker.ExecutionResult{
		Success:        true,
		ExitCode:       0,
		Duration:       100 * time.Millisecond,
		GeneratedFiles: []codegen.GeneratedFile{},
	}, nil
}

func (m *mockDockerRunner) PullImage(ctx context.Context, image string) error {
	if m.pullFunc != nil {
		return m.pullFunc(ctx, image)
	}
	return nil
}

func (m *mockDockerRunner) Cleanup(ctx context.Context) error {
	return nil
}

func (m *mockDockerRunner) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// Mock Cache
type mockCache struct {
	getFunc   func(ctx context.Context, key *codegen.CacheKey) (*codegen.CompilationResult, error)
	setFunc   func(ctx context.Context, key *codegen.CacheKey, result *codegen.CompilationResult, ttl time.Duration) error
	closeFunc func() error
}

func (m *mockCache) Get(ctx context.Context, key *codegen.CacheKey) (*codegen.CompilationResult, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	return nil, cache.ErrCacheMiss
}

func (m *mockCache) Set(ctx context.Context, key *codegen.CacheKey, result *codegen.CompilationResult, ttl time.Duration) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, result, ttl)
	}
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key *codegen.CacheKey) error {
	return nil
}

func (m *mockCache) Invalidate(ctx context.Context, moduleName, version string) error {
	return nil
}

func (m *mockCache) Stats(ctx context.Context) (*cache.Stats, error) {
	return &cache.Stats{}, nil
}

func (m *mockCache) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// Mock Artifacts Manager
type mockArtifactsManager struct {
	storeFunc    func(ctx context.Context, req *artifacts.StoreRequest) (*artifacts.StoreResult, error)
	retrieveFunc func(ctx context.Context, req *artifacts.RetrieveRequest) (*artifacts.RetrieveResult, error)
	closeFunc    func() error
}

func (m *mockArtifactsManager) Store(ctx context.Context, req *artifacts.StoreRequest) (*artifacts.StoreResult, error) {
	if m.storeFunc != nil {
		return m.storeFunc(ctx, req)
	}
	return &artifacts.StoreResult{
		S3Key:    "test/module/v1.0.0/go.tar.gz",
		S3Bucket: "test-bucket",
		Hash:     "abc123",
	}, nil
}

func (m *mockArtifactsManager) Retrieve(ctx context.Context, req *artifacts.RetrieveRequest) (*artifacts.RetrieveResult, error) {
	if m.retrieveFunc != nil {
		return m.retrieveFunc(ctx, req)
	}
	return nil, errors.New("not found")
}

func (m *mockArtifactsManager) Delete(ctx context.Context, moduleName, version, language string) error {
	return nil
}

func (m *mockArtifactsManager) Exists(ctx context.Context, moduleName, version, language string) (bool, error) {
	return false, nil
}

func (m *mockArtifactsManager) GetURL(ctx context.Context, moduleName, version, language string, ttl int) (string, error) {
	return "", nil
}

func (m *mockArtifactsManager) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// Helper to create orchestrator with mocks
func newMockOrchestrator(t *testing.T, dockerRunner docker.Runner, cacheInstance CacheInterface, artifactsManager artifacts.Manager) *DefaultOrchestrator {
	langRegistry, err := languages.InitializeDefaultRegistry()
	require.NoError(t, err)

	pkgRegistry := packages.NewRegistry()

	return &DefaultOrchestrator{
		config:           DefaultConfig(),
		languageRegistry: langRegistry,
		dockerRunner:     dockerRunner,
		packageRegistry:  pkgRegistry,
		cache:            cacheInstance,
		artifactsManager: artifactsManager,
		jobs:             make(map[string]*codegen.CompilationJob),
	}
}

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
	cfg := DefaultConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, pkgconfig.DefaultMaxParallelWorkers, cfg.MaxParallelWorkers)
	assert.True(t, cfg.EnableCache)
	assert.True(t, cfg.EnableMetrics)
	assert.Equal(t, pkgconfig.DefaultCodeGenVersion, cfg.CodeGenVersion)
	assert.Equal(t, pkgconfig.DefaultCompilationTimeout, cfg.CompilationTimeout)
}

func TestClose(t *testing.T) {
	orch, err := NewOrchestrator(nil)
	require.NoError(t, err)

	err = orch.Close()
	assert.NoError(t, err)
}

func TestClose_WithMocks(t *testing.T) {
	dockerRunner := &mockDockerRunner{
		closeFunc: func() error {
			return nil
		},
	}
	cacheInstance := &mockCache{
		closeFunc: func() error {
			return nil
		},
	}
	artifactsManager := &mockArtifactsManager{
		closeFunc: func() error {
			return nil
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, cacheInstance, artifactsManager)
	err := orch.Close()
	assert.NoError(t, err)
}

func TestClose_WithErrors(t *testing.T) {
	dockerRunner := &mockDockerRunner{
		closeFunc: func() error {
			return errors.New("docker close error")
		},
	}
	cacheInstance := &mockCache{
		closeFunc: func() error {
			return errors.New("cache close error")
		},
	}
	artifactsManager := &mockArtifactsManager{
		closeFunc: func() error {
			return errors.New("artifacts close error")
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, cacheInstance, artifactsManager)
	err := orch.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "close errors")
}

func TestCompileSingle_Success(t *testing.T) {
	dockerRunner := &mockDockerRunner{
		executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
			return &docker.ExecutionResult{
				Success:  true,
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
				GeneratedFiles: []codegen.GeneratedFile{
					{
						Path:    "test.pb.go",
						Content: []byte("package test"),
						Size:    12,
					},
				},
			}, nil
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, nil, nil)
	defer orch.Close()

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
		IncludeGRPC: true,
	}

	result, err := orch.CompileSingle(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "go", result.Language)
	assert.Len(t, result.GeneratedFiles, 1)
	assert.False(t, result.CacheHit)
}

func TestCompileSingle_DisabledLanguage(t *testing.T) {
	dockerRunner := &mockDockerRunner{}
	orch := newMockOrchestrator(t, dockerRunner, nil, nil)
	defer orch.Close()

	// Disable Go language
	langSpec, _ := orch.languageRegistry.Get("go")
	langSpec.Enabled = false

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	_, err := orch.CompileSingle(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")

	// Re-enable for other tests
	langSpec.Enabled = true
}

func TestCompileSingle_CacheHit(t *testing.T) {
	cachedResult := &codegen.CompilationResult{
		Success:  true,
		Language: "go",
		GeneratedFiles: []codegen.GeneratedFile{
			{Path: "cached.pb.go", Content: []byte("cached"), Size: 6},
		},
	}

	cacheInstance := &mockCache{
		getFunc: func(ctx context.Context, key *codegen.CacheKey) (*codegen.CompilationResult, error) {
			return cachedResult, nil
		},
	}

	dockerRunner := &mockDockerRunner{
		executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
			t.Fatal("Execute should not be called on cache hit")
			return nil, nil
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, cacheInstance, nil)
	defer orch.Close()
	orch.config.EnableCache = true

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	result, err := orch.CompileSingle(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.CacheHit)
	assert.True(t, result.Success)
	assert.Len(t, result.GeneratedFiles, 1)
}

func TestCompileSingle_CacheMiss(t *testing.T) {
	cacheInstance := &mockCache{
		getFunc: func(ctx context.Context, key *codegen.CacheKey) (*codegen.CompilationResult, error) {
			return nil, cache.ErrCacheMiss
		},
		setFunc: func(ctx context.Context, key *codegen.CacheKey, result *codegen.CompilationResult, ttl time.Duration) error {
			assert.Equal(t, 24*time.Hour, ttl)
			return nil
		},
	}

	dockerRunner := &mockDockerRunner{
		executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
			return &docker.ExecutionResult{
				Success:        true,
				ExitCode:       0,
				Duration:       50 * time.Millisecond,
				GeneratedFiles: []codegen.GeneratedFile{{Path: "test.pb.go"}},
			}, nil
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, cacheInstance, nil)
	defer orch.Close()
	orch.config.EnableCache = true

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	result, err := orch.CompileSingle(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.CacheHit)
	assert.True(t, result.Success)
}

func TestCompileSingle_WithS3Upload(t *testing.T) {
	dockerRunner := &mockDockerRunner{
		executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
			return &docker.ExecutionResult{
				Success:  true,
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
				GeneratedFiles: []codegen.GeneratedFile{
					{Path: "test.pb.go", Content: []byte("test")},
				},
			}, nil
		},
	}

	artifactsManager := &mockArtifactsManager{
		storeFunc: func(ctx context.Context, req *artifacts.StoreRequest) (*artifacts.StoreResult, error) {
			assert.Equal(t, "test", req.ModuleName)
			assert.Equal(t, "v1.0.0", req.Version)
			assert.Equal(t, "go", req.Language)
			assert.Len(t, req.Files, 1)
			return &artifacts.StoreResult{
				S3Key:    "test/v1.0.0/go.tar.gz",
				S3Bucket: "test-bucket",
				Hash:     "hash123",
			}, nil
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, nil, artifactsManager)
	defer orch.Close()

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	result, err := orch.CompileSingle(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "test/v1.0.0/go.tar.gz", result.S3Key)
	assert.Equal(t, "test-bucket", result.S3Bucket)
	assert.Equal(t, "hash123", result.ArtifactHash)
}

func TestCompileSingle_S3UploadFailure(t *testing.T) {
	dockerRunner := &mockDockerRunner{
		executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
			return &docker.ExecutionResult{
				Success:        true,
				ExitCode:       0,
				Duration:       100 * time.Millisecond,
				GeneratedFiles: []codegen.GeneratedFile{{Path: "test.pb.go"}},
			}, nil
		},
	}

	artifactsManager := &mockArtifactsManager{
		storeFunc: func(ctx context.Context, req *artifacts.StoreRequest) (*artifacts.StoreResult, error) {
			return nil, errors.New("S3 upload failed")
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, nil, artifactsManager)
	defer orch.Close()

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	// Should succeed even if S3 upload fails
	result, err := orch.CompileSingle(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Empty(t, result.S3Key)
}

func TestCompileSingle_DockerExecutionFailure(t *testing.T) {
	dockerRunner := &mockDockerRunner{
		executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
			return nil, errors.New("docker execution failed")
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, nil, nil)
	defer orch.Close()

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	result, err := orch.CompileSingle(ctx, req)
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "compilation failed")
}

func TestCompileAll_Success(t *testing.T) {
	var executionCount int
	var mu sync.Mutex

	dockerRunner := &mockDockerRunner{
		executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
			mu.Lock()
			executionCount++
			currentCount := executionCount
			mu.Unlock()

			return &docker.ExecutionResult{
				Success:  true,
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
				GeneratedFiles: []codegen.GeneratedFile{
					{Path: fmt.Sprintf("test_%d.pb", currentCount), Content: []byte("test")},
				},
			}, nil
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, nil, nil)
	defer orch.Close()

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	languages := []string{"go", "python", "java"}
	results, err := orch.CompileAll(ctx, req, languages)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	for i, result := range results {
		assert.True(t, result.Success, "language %s should succeed", languages[i])
		assert.Equal(t, languages[i], result.Language)
	}

	mu.Lock()
	assert.Equal(t, 3, executionCount)
	mu.Unlock()
}

func TestCompileAll_ParallelExecution(t *testing.T) {
	executionTimes := make(map[string]time.Time)
	var mu sync.Mutex

	dockerRunner := &mockDockerRunner{
		executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
			mu.Lock()
			// Record the approximate execution time
			executionTimes[time.Now().String()] = time.Now()
			mu.Unlock()

			// Simulate some work
			time.Sleep(50 * time.Millisecond)

			return &docker.ExecutionResult{
				Success:        true,
				ExitCode:       0,
				Duration:       50 * time.Millisecond,
				GeneratedFiles: []codegen.GeneratedFile{{Path: "test.pb"}},
			}, nil
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, nil, nil)
	defer orch.Close()
	orch.config.MaxParallelWorkers = 3

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	start := time.Now()
	languages := []string{"go", "python", "java"}
	results, err := orch.CompileAll(ctx, req, languages)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, results, 3)

	// With 3 workers and 3 languages, should complete in roughly 50ms + overhead
	// If sequential, would take 150ms+
	assert.Less(t, duration, 150*time.Millisecond, "Should complete in parallel")
}

func TestCompileAll_PartialFailure(t *testing.T) {
	dockerRunner := &mockDockerRunner{
		executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
			// Fail Python compilation
			if req.ProtocFlags[0] == "--python_out=/output" {
				return nil, errors.New("python compilation failed")
			}
			return &docker.ExecutionResult{
				Success:        true,
				ExitCode:       0,
				Duration:       50 * time.Millisecond,
				GeneratedFiles: []codegen.GeneratedFile{{Path: "test.pb"}},
			}, nil
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, nil, nil)
	defer orch.Close()

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	languages := []string{"go", "python", "java"}
	results, err := orch.CompileAll(ctx, req, languages)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compilation failed")
	assert.Len(t, results, 3)

	// Check that we got results for all languages, even if some failed
	assert.True(t, results[0].Success)  // go
	assert.False(t, results[1].Success) // python
	assert.True(t, results[2].Success)  // java
}

func TestCompileAll_DisabledLanguage(t *testing.T) {
	dockerRunner := &mockDockerRunner{}
	orch := newMockOrchestrator(t, dockerRunner, nil, nil)
	defer orch.Close()

	// Disable Python
	langSpec, _ := orch.languageRegistry.Get("python")
	langSpec.Enabled = false

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	languages := []string{"go", "python"}
	_, err := orch.CompileAll(ctx, req, languages)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")

	// Re-enable
	langSpec.Enabled = true
}

func TestCompileAll_MaxWorkers(t *testing.T) {
	tests := []struct {
		name          string
		maxWorkers    int
		numLanguages  int
		expectedWorkers int
	}{
		{
			name:          "default workers",
			maxWorkers:    0,
			numLanguages:  10,
			expectedWorkers: 5, // default
		},
		{
			name:          "more workers than languages",
			maxWorkers:    10,
			numLanguages:  3,
			expectedWorkers: 3, // capped to number of languages
		},
		{
			name:          "fewer workers than languages",
			maxWorkers:    2,
			numLanguages:  5,
			expectedWorkers: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dockerRunner := &mockDockerRunner{
				executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
					return &docker.ExecutionResult{
						Success:        true,
						GeneratedFiles: []codegen.GeneratedFile{{Path: "test.pb"}},
					}, nil
				},
			}

			orch := newMockOrchestrator(t, dockerRunner, nil, nil)
			defer orch.Close()
			orch.config.MaxParallelWorkers = tt.maxWorkers

			ctx := context.Background()
			req := &CompileRequest{
				ModuleName: "test",
				Version:    "v1.0.0",
				ProtoFiles: []codegen.ProtoFile{
					{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
				},
			}

			languages := make([]string, tt.numLanguages)
			for i := 0; i < tt.numLanguages; i++ {
				// Use available languages cyclically
				availableLangs := []string{"go", "python", "java"}
				languages[i] = availableLangs[i%len(availableLangs)]
			}

			results, err := orch.CompileAll(ctx, req, languages)
			require.NoError(t, err)
			assert.Len(t, results, tt.numLanguages)
		})
	}
}

func TestBuildProtocFlags_CPP(t *testing.T) {
	orch, err := NewOrchestrator(nil)
	require.NoError(t, err)
	defer orch.Close()

	langSpec, err := orch.languageRegistry.Get("cpp")
	require.NoError(t, err)

	req := &CompileRequest{
		IncludeGRPC: true,
	}

	flags := orch.buildProtocFlags(langSpec, req)
	assert.Contains(t, flags, "--cpp_out=/output")
}

func TestBuildProtocFlags_GenericLanguage(t *testing.T) {
	orch, err := NewOrchestrator(nil)
	require.NoError(t, err)
	defer orch.Close()

	// Use a language that doesn't have special handling
	langSpec, err := orch.languageRegistry.Get("rust")
	if err != nil {
		// If rust isn't available, skip this test
		t.Skip("Rust language not available")
	}

	req := &CompileRequest{
		IncludeGRPC: false,
	}

	flags := orch.buildProtocFlags(langSpec, req)
	assert.Contains(t, flags, "--rust_out=/output")
}

func TestCreateJob(t *testing.T) {
	orch := newMockOrchestrator(t, &mockDockerRunner{}, nil, nil)
	defer orch.Close()

	job := orch.createJob(123, "go")
	assert.NotEmpty(t, job.ID)
	assert.Equal(t, int64(123), job.VersionID)
	assert.Equal(t, "go", job.Language)
	assert.Equal(t, codegen.JobStatusPending, job.Status)
	assert.NotNil(t, job.StartedAt)

	// Verify job was added to jobs map
	retrievedJob, err := orch.GetStatus(context.Background(), job.ID)
	require.NoError(t, err)
	assert.Equal(t, job.ID, retrievedJob.ID)
}

func TestUpdateJob(t *testing.T) {
	orch := newMockOrchestrator(t, &mockDockerRunner{}, nil, nil)
	defer orch.Close()

	job := orch.createJob(123, "go")

	result := &codegen.CompilationResult{
		Success:  true,
		Language: "go",
	}

	orch.updateJob(job.ID, codegen.JobStatusCompleted, result, nil)

	retrievedJob, err := orch.GetStatus(context.Background(), job.ID)
	require.NoError(t, err)
	assert.Equal(t, codegen.JobStatusCompleted, retrievedJob.Status)
	assert.NotNil(t, retrievedJob.CompletedAt)
	assert.Equal(t, result, retrievedJob.Result)
	assert.Empty(t, retrievedJob.Error)
}

func TestUpdateJob_WithError(t *testing.T) {
	orch := newMockOrchestrator(t, &mockDockerRunner{}, nil, nil)
	defer orch.Close()

	job := orch.createJob(123, "go")

	testErr := errors.New("compilation error")
	orch.updateJob(job.ID, codegen.JobStatusFailed, nil, testErr)

	retrievedJob, err := orch.GetStatus(context.Background(), job.ID)
	require.NoError(t, err)
	assert.Equal(t, codegen.JobStatusFailed, retrievedJob.Status)
	assert.Equal(t, "compilation error", retrievedJob.Error)
}

func TestUpdateJob_NonExistentJob(t *testing.T) {
	orch := newMockOrchestrator(t, &mockDockerRunner{}, nil, nil)
	defer orch.Close()

	// Should not panic
	orch.updateJob("nonexistent", codegen.JobStatusCompleted, nil, nil)
}

func TestGeneratePackageFiles_NoPackageManager(t *testing.T) {
	orch := newMockOrchestrator(t, &mockDockerRunner{}, nil, nil)
	defer orch.Close()

	langSpec, err := orch.languageRegistry.Get("go")
	require.NoError(t, err)

	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
	}

	// Temporarily set package manager to nil
	originalPM := langSpec.PackageManager
	langSpec.PackageManager = nil
	defer func() { langSpec.PackageManager = originalPM }()

	files, err := orch.generatePackageFiles(langSpec, req)
	assert.NoError(t, err)
	assert.Nil(t, files)
}

func TestGeneratePackageFiles_GeneratorNotFound(t *testing.T) {
	orch := newMockOrchestrator(t, &mockDockerRunner{}, nil, nil)
	defer orch.Close()

	langSpec, err := orch.languageRegistry.Get("go")
	require.NoError(t, err)

	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
	}

	// Set a package manager that doesn't exist
	originalPM := langSpec.PackageManager
	langSpec.PackageManager = &languages.PackageManagerSpec{
		Name: "nonexistent-package-manager",
	}
	defer func() { langSpec.PackageManager = originalPM }()

	files, err := orch.generatePackageFiles(langSpec, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package generator not found")
	assert.Nil(t, files)
}

func TestCompileSingle_CacheSetFailure(t *testing.T) {
	cacheInstance := &mockCache{
		getFunc: func(ctx context.Context, key *codegen.CacheKey) (*codegen.CompilationResult, error) {
			return nil, cache.ErrCacheMiss
		},
		setFunc: func(ctx context.Context, key *codegen.CacheKey, result *codegen.CompilationResult, ttl time.Duration) error {
			return errors.New("cache set failed")
		},
	}

	dockerRunner := &mockDockerRunner{
		executeFunc: func(ctx context.Context, req *docker.ExecutionRequest) (*docker.ExecutionResult, error) {
			return &docker.ExecutionResult{
				Success:        true,
				ExitCode:       0,
				Duration:       50 * time.Millisecond,
				GeneratedFiles: []codegen.GeneratedFile{{Path: "test.pb.go"}},
			}, nil
		},
	}

	orch := newMockOrchestrator(t, dockerRunner, cacheInstance, nil)
	defer orch.Close()
	orch.config.EnableCache = true

	ctx := context.Background()
	req := &CompileRequest{
		ModuleName: "test",
		Version:    "v1.0.0",
		Language:   "go",
		ProtoFiles: []codegen.ProtoFile{
			{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
		},
	}

	// Should succeed even if cache set fails
	result, err := orch.CompileSingle(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Success)
}
