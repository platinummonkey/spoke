package storage

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultConfig tests the DefaultConfig function
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test default values
	assert.Equal(t, "filesystem", cfg.Type)
	assert.Equal(t, "/tmp/spoke", cfg.FilesystemRoot)
	assert.Equal(t, 20, cfg.PostgresMaxConns)
	assert.Equal(t, 2, cfg.PostgresMinConns)
	assert.Equal(t, 10*time.Second, cfg.PostgresTimeout)
	assert.Equal(t, 0, cfg.RedisDB)
	assert.Equal(t, 3, cfg.RedisMaxRetries)
	assert.Equal(t, 10, cfg.RedisPoolSize)
	assert.True(t, cfg.CacheEnabled)
	assert.Equal(t, int64(10*1024*1024), cfg.L1CacheSize)

	// Test cache TTL defaults
	require.NotNil(t, cfg.CacheTTL)
	assert.Equal(t, 1*time.Hour, cfg.CacheTTL["module"])
	assert.Equal(t, 1*time.Hour, cfg.CacheTTL["version"])
	assert.Equal(t, 30*time.Minute, cfg.CacheTTL["version_full"])
	assert.Equal(t, 5*time.Minute, cfg.CacheTTL["version_list"])
	assert.Equal(t, 1*time.Minute, cfg.CacheTTL["latest"])
	assert.Equal(t, 24*time.Hour, cfg.CacheTTL["compiled"])
	assert.Equal(t, 24*time.Hour, cfg.CacheTTL["proto_content"])
	assert.Equal(t, 1*time.Hour, cfg.CacheTTL["dependency_tree"])
}

// TestConfig_Fields tests that Config struct fields can be set
func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		Type:           "postgres",
		FilesystemRoot: "/custom/path",

		PostgresURL:         "postgres://localhost:5432/spoke",
		PostgresReplicaURLs: "postgres://replica1:5432/spoke,postgres://replica2:5432/spoke",
		PostgresMaxConns:    50,
		PostgresMinConns:    5,
		PostgresTimeout:     30 * time.Second,

		S3Endpoint:       "https://s3.amazonaws.com",
		S3Region:         "us-west-2",
		S3Bucket:         "spoke-artifacts",
		S3AccessKey:      "access-key",
		S3SecretKey:      "secret-key",
		S3UsePathStyle:   true,
		S3ForcePathStyle: false,

		RedisURL:        "redis://localhost:6379",
		RedisPassword:   "password",
		RedisDB:         1,
		RedisMaxRetries: 5,
		RedisPoolSize:   20,

		CacheEnabled: false,
		CacheTTL: map[string]time.Duration{
			"custom": 2 * time.Hour,
		},
		L1CacheSize: 20 * 1024 * 1024,
	}

	assert.Equal(t, "postgres", cfg.Type)
	assert.Equal(t, "/custom/path", cfg.FilesystemRoot)
	assert.Equal(t, "postgres://localhost:5432/spoke", cfg.PostgresURL)
	assert.Equal(t, "postgres://replica1:5432/spoke,postgres://replica2:5432/spoke", cfg.PostgresReplicaURLs)
	assert.Equal(t, 50, cfg.PostgresMaxConns)
	assert.Equal(t, 5, cfg.PostgresMinConns)
	assert.Equal(t, 30*time.Second, cfg.PostgresTimeout)
	assert.Equal(t, "https://s3.amazonaws.com", cfg.S3Endpoint)
	assert.Equal(t, "us-west-2", cfg.S3Region)
	assert.Equal(t, "spoke-artifacts", cfg.S3Bucket)
	assert.Equal(t, "access-key", cfg.S3AccessKey)
	assert.Equal(t, "secret-key", cfg.S3SecretKey)
	assert.True(t, cfg.S3UsePathStyle)
	assert.False(t, cfg.S3ForcePathStyle)
	assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
	assert.Equal(t, "password", cfg.RedisPassword)
	assert.Equal(t, 1, cfg.RedisDB)
	assert.Equal(t, 5, cfg.RedisMaxRetries)
	assert.Equal(t, 20, cfg.RedisPoolSize)
	assert.False(t, cfg.CacheEnabled)
	assert.Equal(t, 2*time.Hour, cfg.CacheTTL["custom"])
	assert.Equal(t, int64(20*1024*1024), cfg.L1CacheSize)
}

// TestConfig_ZeroValues tests that Config can be initialized with zero values
func TestConfig_ZeroValues(t *testing.T) {
	var cfg Config

	assert.Equal(t, "", cfg.Type)
	assert.Equal(t, "", cfg.FilesystemRoot)
	assert.Equal(t, 0, cfg.PostgresMaxConns)
	assert.Equal(t, 0, cfg.PostgresMinConns)
	assert.Equal(t, time.Duration(0), cfg.PostgresTimeout)
	assert.False(t, cfg.CacheEnabled)
	assert.Nil(t, cfg.CacheTTL)
	assert.Equal(t, int64(0), cfg.L1CacheSize)
}

// Mock implementations for interface testing

type mockModuleReader struct {
	getModuleFunc          func(ctx context.Context, name string) (*api.Module, error)
	listModulesFunc        func(ctx context.Context) ([]*api.Module, error)
	listModulesPaginated   func(ctx context.Context, limit, offset int) ([]*api.Module, int64, error)
}

func (m *mockModuleReader) GetModuleContext(ctx context.Context, name string) (*api.Module, error) {
	if m.getModuleFunc != nil {
		return m.getModuleFunc(ctx, name)
	}
	return &api.Module{Name: name}, nil
}

func (m *mockModuleReader) ListModulesContext(ctx context.Context) ([]*api.Module, error) {
	if m.listModulesFunc != nil {
		return m.listModulesFunc(ctx)
	}
	return []*api.Module{}, nil
}

func (m *mockModuleReader) ListModulesPaginated(ctx context.Context, limit, offset int) ([]*api.Module, int64, error) {
	if m.listModulesPaginated != nil {
		return m.listModulesPaginated(ctx, limit, offset)
	}
	return []*api.Module{}, 0, nil
}

// TestModuleReader_Interface tests that ModuleReader interface can be implemented
func TestModuleReader_Interface(t *testing.T) {
	var _ ModuleReader = (*mockModuleReader)(nil)

	mock := &mockModuleReader{}
	ctx := context.Background()

	module, err := mock.GetModuleContext(ctx, "test-module")
	require.NoError(t, err)
	assert.Equal(t, "test-module", module.Name)

	modules, err := mock.ListModulesContext(ctx)
	require.NoError(t, err)
	assert.NotNil(t, modules)

	pagedModules, total, err := mock.ListModulesPaginated(ctx, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, pagedModules)
	assert.Equal(t, int64(0), total)
}

type mockModuleWriter struct {
	createModuleFunc func(ctx context.Context, module *api.Module) error
}

func (m *mockModuleWriter) CreateModuleContext(ctx context.Context, module *api.Module) error {
	if m.createModuleFunc != nil {
		return m.createModuleFunc(ctx, module)
	}
	return nil
}

// TestModuleWriter_Interface tests that ModuleWriter interface can be implemented
func TestModuleWriter_Interface(t *testing.T) {
	var _ ModuleWriter = (*mockModuleWriter)(nil)

	mock := &mockModuleWriter{}
	ctx := context.Background()

	err := mock.CreateModuleContext(ctx, &api.Module{Name: "test"})
	require.NoError(t, err)
}

type mockVersionReader struct {
	getVersionFunc       func(ctx context.Context, moduleName, version string) (*api.Version, error)
	listVersionsFunc     func(ctx context.Context, moduleName string) ([]*api.Version, error)
	listVersionsPagFunc  func(ctx context.Context, moduleName string, limit, offset int) ([]*api.Version, int64, error)
	getFileFunc          func(ctx context.Context, moduleName, version, path string) (*api.File, error)
}

func (m *mockVersionReader) GetVersionContext(ctx context.Context, moduleName, version string) (*api.Version, error) {
	if m.getVersionFunc != nil {
		return m.getVersionFunc(ctx, moduleName, version)
	}
	return &api.Version{ModuleName: moduleName, Version: version}, nil
}

func (m *mockVersionReader) ListVersionsContext(ctx context.Context, moduleName string) ([]*api.Version, error) {
	if m.listVersionsFunc != nil {
		return m.listVersionsFunc(ctx, moduleName)
	}
	return []*api.Version{}, nil
}

func (m *mockVersionReader) ListVersionsPaginated(ctx context.Context, moduleName string, limit, offset int) ([]*api.Version, int64, error) {
	if m.listVersionsPagFunc != nil {
		return m.listVersionsPagFunc(ctx, moduleName, limit, offset)
	}
	return []*api.Version{}, 0, nil
}

func (m *mockVersionReader) GetFileContext(ctx context.Context, moduleName, version, path string) (*api.File, error) {
	if m.getFileFunc != nil {
		return m.getFileFunc(ctx, moduleName, version, path)
	}
	return &api.File{Path: path}, nil
}

// TestVersionReader_Interface tests that VersionReader interface can be implemented
func TestVersionReader_Interface(t *testing.T) {
	var _ VersionReader = (*mockVersionReader)(nil)

	mock := &mockVersionReader{}
	ctx := context.Background()

	version, err := mock.GetVersionContext(ctx, "test-module", "v1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "test-module", version.ModuleName)
	assert.Equal(t, "v1.0.0", version.Version)

	versions, err := mock.ListVersionsContext(ctx, "test-module")
	require.NoError(t, err)
	assert.NotNil(t, versions)

	pagedVersions, total, err := mock.ListVersionsPaginated(ctx, "test-module", 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, pagedVersions)
	assert.Equal(t, int64(0), total)

	file, err := mock.GetFileContext(ctx, "test-module", "v1.0.0", "test.proto")
	require.NoError(t, err)
	assert.Equal(t, "test.proto", file.Path)
}

type mockVersionWriter struct {
	createVersionFunc func(ctx context.Context, version *api.Version) error
	updateVersionFunc func(ctx context.Context, version *api.Version) error
}

func (m *mockVersionWriter) CreateVersionContext(ctx context.Context, version *api.Version) error {
	if m.createVersionFunc != nil {
		return m.createVersionFunc(ctx, version)
	}
	return nil
}

func (m *mockVersionWriter) UpdateVersionContext(ctx context.Context, version *api.Version) error {
	if m.updateVersionFunc != nil {
		return m.updateVersionFunc(ctx, version)
	}
	return nil
}

// TestVersionWriter_Interface tests that VersionWriter interface can be implemented
func TestVersionWriter_Interface(t *testing.T) {
	var _ VersionWriter = (*mockVersionWriter)(nil)

	mock := &mockVersionWriter{}
	ctx := context.Background()

	err := mock.CreateVersionContext(ctx, &api.Version{ModuleName: "test", Version: "v1.0.0"})
	require.NoError(t, err)

	err = mock.UpdateVersionContext(ctx, &api.Version{ModuleName: "test", Version: "v1.0.0"})
	require.NoError(t, err)
}

type mockReadCloser struct {
	reader io.Reader
	closed bool
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}

type mockFileStorage struct {
	getFileFunc func(ctx context.Context, hash string) (io.ReadCloser, error)
	putFileFunc func(ctx context.Context, content io.Reader, contentType string) (string, error)
}

func (m *mockFileStorage) GetFileContent(ctx context.Context, hash string) (io.ReadCloser, error) {
	if m.getFileFunc != nil {
		return m.getFileFunc(ctx, hash)
	}
	return &mockReadCloser{reader: strings.NewReader("test content")}, nil
}

func (m *mockFileStorage) PutFileContent(ctx context.Context, content io.Reader, contentType string) (string, error) {
	if m.putFileFunc != nil {
		return m.putFileFunc(ctx, content, contentType)
	}
	return "test-hash", nil
}

// TestFileStorage_Interface tests that FileStorage interface can be implemented
func TestFileStorage_Interface(t *testing.T) {
	var _ FileStorage = (*mockFileStorage)(nil)

	mock := &mockFileStorage{}
	ctx := context.Background()

	reader, err := mock.GetFileContent(ctx, "test-hash")
	require.NoError(t, err)
	assert.NotNil(t, reader)
	defer reader.Close()

	hash, err := mock.PutFileContent(ctx, strings.NewReader("content"), "text/plain")
	require.NoError(t, err)
	assert.Equal(t, "test-hash", hash)
}

type mockArtifactStorage struct {
	getArtifactFunc func(ctx context.Context, moduleName, version, language string) (io.ReadCloser, error)
	putArtifactFunc func(ctx context.Context, moduleName, version, language string, artifact io.Reader) error
}

func (m *mockArtifactStorage) GetCompiledArtifact(ctx context.Context, moduleName, version, language string) (io.ReadCloser, error) {
	if m.getArtifactFunc != nil {
		return m.getArtifactFunc(ctx, moduleName, version, language)
	}
	return &mockReadCloser{reader: strings.NewReader("artifact content")}, nil
}

func (m *mockArtifactStorage) PutCompiledArtifact(ctx context.Context, moduleName, version, language string, artifact io.Reader) error {
	if m.putArtifactFunc != nil {
		return m.putArtifactFunc(ctx, moduleName, version, language, artifact)
	}
	return nil
}

// TestArtifactStorage_Interface tests that ArtifactStorage interface can be implemented
func TestArtifactStorage_Interface(t *testing.T) {
	var _ ArtifactStorage = (*mockArtifactStorage)(nil)

	mock := &mockArtifactStorage{}
	ctx := context.Background()

	reader, err := mock.GetCompiledArtifact(ctx, "test-module", "v1.0.0", "go")
	require.NoError(t, err)
	assert.NotNil(t, reader)
	defer reader.Close()

	err = mock.PutCompiledArtifact(ctx, "test-module", "v1.0.0", "go", strings.NewReader("artifact"))
	require.NoError(t, err)
}

type mockCacheManager struct {
	invalidateCacheFunc func(ctx context.Context, patterns ...string) error
}

func (m *mockCacheManager) InvalidateCache(ctx context.Context, patterns ...string) error {
	if m.invalidateCacheFunc != nil {
		return m.invalidateCacheFunc(ctx, patterns...)
	}
	return nil
}

// TestCacheManager_Interface tests that CacheManager interface can be implemented
func TestCacheManager_Interface(t *testing.T) {
	var _ CacheManager = (*mockCacheManager)(nil)

	mock := &mockCacheManager{}
	ctx := context.Background()

	err := mock.InvalidateCache(ctx, "pattern1", "pattern2")
	require.NoError(t, err)
}

type mockHealthChecker struct {
	healthCheckFunc func(ctx context.Context) error
}

func (m *mockHealthChecker) HealthCheck(ctx context.Context) error {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return nil
}

// TestHealthChecker_Interface tests that HealthChecker interface can be implemented
func TestHealthChecker_Interface(t *testing.T) {
	var _ HealthChecker = (*mockHealthChecker)(nil)

	mock := &mockHealthChecker{}
	ctx := context.Background()

	err := mock.HealthCheck(ctx)
	require.NoError(t, err)
}

type mockStorage struct {
	*mockModuleReader
	*mockModuleWriter
	*mockVersionReader
	*mockVersionWriter
	*mockFileStorage
	*mockArtifactStorage
	*mockCacheManager
	*mockHealthChecker
}

// TestStorage_Interface tests that Storage interface can be implemented
func TestStorage_Interface(t *testing.T) {
	mock := &mockStorage{
		mockModuleReader:    &mockModuleReader{},
		mockModuleWriter:    &mockModuleWriter{},
		mockVersionReader:   &mockVersionReader{},
		mockVersionWriter:   &mockVersionWriter{},
		mockFileStorage:     &mockFileStorage{},
		mockArtifactStorage: &mockArtifactStorage{},
		mockCacheManager:    &mockCacheManager{},
		mockHealthChecker:   &mockHealthChecker{},
	}

	var _ Storage = mock

	ctx := context.Background()

	// Test ModuleReader methods
	module, err := mock.GetModuleContext(ctx, "test")
	require.NoError(t, err)
	assert.NotNil(t, module)

	modules, err := mock.ListModulesContext(ctx)
	require.NoError(t, err)
	assert.NotNil(t, modules)

	pagedModules, total, err := mock.ListModulesPaginated(ctx, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, pagedModules)
	assert.Equal(t, int64(0), total)

	// Test ModuleWriter methods
	err = mock.CreateModuleContext(ctx, &api.Module{Name: "test"})
	require.NoError(t, err)

	// Test VersionReader methods
	version, err := mock.GetVersionContext(ctx, "test", "v1.0.0")
	require.NoError(t, err)
	assert.NotNil(t, version)

	versions, err := mock.ListVersionsContext(ctx, "test")
	require.NoError(t, err)
	assert.NotNil(t, versions)

	pagedVersions, total, err := mock.ListVersionsPaginated(ctx, "test", 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, pagedVersions)
	assert.Equal(t, int64(0), total)

	file, err := mock.GetFileContext(ctx, "test", "v1.0.0", "test.proto")
	require.NoError(t, err)
	assert.NotNil(t, file)

	// Test VersionWriter methods
	err = mock.CreateVersionContext(ctx, &api.Version{ModuleName: "test", Version: "v1.0.0"})
	require.NoError(t, err)

	err = mock.UpdateVersionContext(ctx, &api.Version{ModuleName: "test", Version: "v1.0.0"})
	require.NoError(t, err)

	// Test FileStorage methods
	reader, err := mock.GetFileContent(ctx, "hash")
	require.NoError(t, err)
	assert.NotNil(t, reader)
	reader.Close()

	hash, err := mock.PutFileContent(ctx, strings.NewReader("content"), "text/plain")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Test ArtifactStorage methods
	artifact, err := mock.GetCompiledArtifact(ctx, "test", "v1.0.0", "go")
	require.NoError(t, err)
	assert.NotNil(t, artifact)
	artifact.Close()

	err = mock.PutCompiledArtifact(ctx, "test", "v1.0.0", "go", strings.NewReader("artifact"))
	require.NoError(t, err)

	// Test CacheManager methods
	err = mock.InvalidateCache(ctx, "pattern")
	require.NoError(t, err)

	// Test HealthChecker methods
	err = mock.HealthCheck(ctx)
	require.NoError(t, err)
}

// TestConfig_CacheTTL_Modification tests that CacheTTL map can be modified
func TestConfig_CacheTTL_Modification(t *testing.T) {
	cfg := DefaultConfig()

	// Test that we can modify cache TTL
	cfg.CacheTTL["module"] = 2 * time.Hour
	assert.Equal(t, 2*time.Hour, cfg.CacheTTL["module"])

	// Test that we can add new entries
	cfg.CacheTTL["custom"] = 5 * time.Minute
	assert.Equal(t, 5*time.Minute, cfg.CacheTTL["custom"])

	// Test that we can delete entries
	delete(cfg.CacheTTL, "module")
	_, exists := cfg.CacheTTL["module"]
	assert.False(t, exists)
}

// TestConfig_StorageTypes tests different storage type configurations
func TestConfig_StorageTypes(t *testing.T) {
	tests := []struct {
		name        string
		storageType string
	}{
		{"filesystem", "filesystem"},
		{"postgres", "postgres"},
		{"hybrid", "hybrid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Type: tt.storageType}
			assert.Equal(t, tt.storageType, cfg.Type)
		})
	}
}
