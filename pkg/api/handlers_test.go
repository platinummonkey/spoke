package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStorage is a mock implementation of the Storage interface for testing
type mockStorage struct {
	modules  map[string]*Module
	versions map[string]map[string]*Version // moduleName -> version -> Version
	files    map[string]map[string]map[string]*File // moduleName -> version -> path -> File

	createModuleError  error
	getModuleError     error
	listModulesError   error
	createVersionError error
	getVersionError    error
	listVersionsError  error
	getFileError       error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		modules:  make(map[string]*Module),
		versions: make(map[string]map[string]*Version),
		files:    make(map[string]map[string]map[string]*File),
	}
}

func (m *mockStorage) CreateModule(module *Module) error {
	if m.createModuleError != nil {
		return m.createModuleError
	}
	m.modules[module.Name] = module
	return nil
}

func (m *mockStorage) GetModule(name string) (*Module, error) {
	if m.getModuleError != nil {
		return nil, m.getModuleError
	}
	module, ok := m.modules[name]
	if !ok {
		return nil, ErrNotFound
	}
	return module, nil
}

func (m *mockStorage) ListModules() ([]*Module, error) {
	if m.listModulesError != nil {
		return nil, m.listModulesError
	}
	modules := make([]*Module, 0, len(m.modules))
	for _, module := range m.modules {
		modules = append(modules, module)
	}
	return modules, nil
}

func (m *mockStorage) CreateVersion(version *Version) error {
	if m.createVersionError != nil {
		return m.createVersionError
	}
	if m.versions[version.ModuleName] == nil {
		m.versions[version.ModuleName] = make(map[string]*Version)
	}
	m.versions[version.ModuleName][version.Version] = version
	return nil
}

func (m *mockStorage) GetVersion(moduleName, version string) (*Version, error) {
	if m.getVersionError != nil {
		return nil, m.getVersionError
	}
	versions, ok := m.versions[moduleName]
	if !ok {
		return nil, ErrNotFound
	}
	v, ok := versions[version]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

func (m *mockStorage) ListVersions(moduleName string) ([]*Version, error) {
	if m.listVersionsError != nil {
		return nil, m.listVersionsError
	}
	versions := m.versions[moduleName]
	result := make([]*Version, 0, len(versions))
	for _, v := range versions {
		result = append(result, v)
	}
	return result, nil
}

func (m *mockStorage) UpdateVersion(version *Version) error {
	if m.versions[version.ModuleName] == nil {
		return ErrNotFound
	}
	if m.versions[version.ModuleName][version.Version] == nil {
		return ErrNotFound
	}
	m.versions[version.ModuleName][version.Version] = version
	return nil
}

func (m *mockStorage) GetFile(moduleName, version, path string) (*File, error) {
	if m.getFileError != nil {
		return nil, m.getFileError
	}
	moduleFiles, ok := m.files[moduleName]
	if !ok {
		return nil, ErrNotFound
	}
	versionFiles, ok := moduleFiles[version]
	if !ok {
		return nil, ErrNotFound
	}
	file, ok := versionFiles[path]
	if !ok {
		return nil, ErrNotFound
	}
	return file, nil
}

// TestNewServer verifies server initialization
func TestNewServer(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	assert.NotNil(t, server)
	assert.NotNil(t, server.storage)
	assert.NotNil(t, server.router)
	assert.Nil(t, server.db)
	assert.Nil(t, server.authHandlers)
	assert.Nil(t, server.eventTracker)
}

// TestCreateModule_Success tests successful module creation
func TestCreateModule_Success(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	module := Module{
		Name:        "test-module",
		Description: "Test module description",
	}
	body, err := json.Marshal(module)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/modules", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.createModule(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response Module
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-module", response.Name)
	assert.Equal(t, "Test module description", response.Description)
	assert.False(t, response.CreatedAt.IsZero())
	assert.False(t, response.UpdatedAt.IsZero())
}

// TestCreateModule_InvalidJSON tests module creation with invalid JSON
func TestCreateModule_InvalidJSON(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	req := httptest.NewRequest("POST", "/modules", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	server.createModule(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreateModule_StorageError tests module creation with storage error
func TestCreateModule_StorageError(t *testing.T) {
	storage := newMockStorage()
	storage.createModuleError = errors.New("storage error")
	server := NewServer(storage, nil)

	module := Module{
		Name:        "test-module",
		Description: "Test module",
	}
	body, err := json.Marshal(module)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/modules", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.createModule(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestListModules_Empty tests listing when no modules exist
func TestListModules_Empty(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	req := httptest.NewRequest("GET", "/modules", nil)
	w := httptest.NewRecorder()

	server.listModules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response, 0)
}

// TestListModules_WithData tests listing with multiple modules
func TestListModules_WithData(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	// Add test modules
	module1 := &Module{
		Name:        "module1",
		Description: "First module",
		CreatedAt:   time.Now(),
	}
	module2 := &Module{
		Name:        "module2",
		Description: "Second module",
		CreatedAt:   time.Now().Add(-1 * time.Hour),
	}
	storage.modules["module1"] = module1
	storage.modules["module2"] = module2

	// Add versions
	storage.versions["module1"] = map[string]*Version{
		"v1.0.0": {
			ModuleName: "module1",
			Version:    "v1.0.0",
			CreatedAt:  time.Now(),
		},
	}

	req := httptest.NewRequest("GET", "/modules", nil)
	w := httptest.NewRecorder()

	server.listModules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response, 2)
}

// TestListModules_StorageError tests listing with storage error
func TestListModules_StorageError(t *testing.T) {
	storage := newMockStorage()
	storage.listModulesError = errors.New("storage error")
	server := NewServer(storage, nil)

	req := httptest.NewRequest("GET", "/modules", nil)
	w := httptest.NewRecorder()

	server.listModules(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestGetModule_Success tests successful module retrieval
func TestGetModule_Success(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	module := &Module{
		Name:        "test-module",
		Description: "Test module",
		CreatedAt:   time.Now(),
	}
	storage.modules["test-module"] = module

	req := httptest.NewRequest("GET", "/modules/test-module", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.getModule(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test-module", response["name"])
}

// TestGetModule_NotFound tests module not found error
func TestGetModule_NotFound(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	req := httptest.NewRequest("GET", "/modules/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	w := httptest.NewRecorder()

	server.getModule(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetModule_WithVersions tests module retrieval with versions
func TestGetModule_WithVersions(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	module := &Module{
		Name:        "test-module",
		Description: "Test module",
		CreatedAt:   time.Now(),
	}
	storage.modules["test-module"] = module

	// Add versions
	now := time.Now()
	storage.versions["test-module"] = map[string]*Version{
		"v1.0.0": {
			ModuleName: "test-module",
			Version:    "v1.0.0",
			CreatedAt:  now.Add(-2 * time.Hour),
		},
		"v1.1.0": {
			ModuleName: "test-module",
			Version:    "v1.1.0",
			CreatedAt:  now.Add(-1 * time.Hour),
		},
		"v2.0.0": {
			ModuleName: "test-module",
			Version:    "v2.0.0",
			CreatedAt:  now,
		},
	}

	req := httptest.NewRequest("GET", "/modules/test-module", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.getModule(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	versions := response["versions"].([]interface{})
	assert.Len(t, versions, 3)

	// Verify versions are sorted by newest first
	firstVersion := versions[0].(map[string]interface{})
	assert.Equal(t, "v2.0.0", firstVersion["version"])
}

// TestCreateVersion_Success tests successful version creation
func TestCreateVersion_Success(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	version := Version{
		Version: "v1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}
	body, err := json.Marshal(version)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/modules/test-module/versions", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.createVersion(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response Version
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", response.Version)
	assert.Equal(t, "test-module", response.ModuleName)
	assert.False(t, response.CreatedAt.IsZero())
}

// TestCreateVersion_InvalidJSON tests version creation with invalid JSON
func TestCreateVersion_InvalidJSON(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	req := httptest.NewRequest("POST", "/modules/test-module/versions", bytes.NewReader([]byte("{")))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.createVersion(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreateVersion_StorageError tests version creation with storage error
func TestCreateVersion_StorageError(t *testing.T) {
	storage := newMockStorage()
	storage.createVersionError = errors.New("storage error")
	server := NewServer(storage, nil)

	version := Version{
		Version: "v1.0.0",
		Files:   []File{{Path: "test.proto", Content: ""}},
	}
	body, err := json.Marshal(version)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/modules/test-module/versions", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.createVersion(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestListVersions_Empty tests listing versions when none exist
func TestListVersions_Empty(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	req := httptest.NewRequest("GET", "/modules/test-module/versions", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.listVersions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []Version
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response, 0)
}

// TestListVersions_WithData tests listing multiple versions
func TestListVersions_WithData(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	storage.versions["test-module"] = map[string]*Version{
		"v1.0.0": {
			ModuleName: "test-module",
			Version:    "v1.0.0",
			CreatedAt:  time.Now(),
		},
		"v1.1.0": {
			ModuleName: "test-module",
			Version:    "v1.1.0",
			CreatedAt:  time.Now().Add(time.Hour),
		},
	}

	req := httptest.NewRequest("GET", "/modules/test-module/versions", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.listVersions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []Version
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response, 2)
}

// TestListVersions_StorageError tests listing versions with storage error
func TestListVersions_StorageError(t *testing.T) {
	storage := newMockStorage()
	storage.listVersionsError = errors.New("storage error")
	server := NewServer(storage, nil)

	req := httptest.NewRequest("GET", "/modules/test-module/versions", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.listVersions(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestGetVersion_Success tests successful version retrieval
func TestGetVersion_Success(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	version := &Version{
		ModuleName: "test-module",
		Version:    "v1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
		CreatedAt: time.Now(),
	}
	storage.versions["test-module"] = map[string]*Version{
		"v1.0.0": version,
	}

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":    "test-module",
		"version": "v1.0.0",
	})
	w := httptest.NewRecorder()

	server.getVersion(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response Version
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test-module", response.ModuleName)
	assert.Equal(t, "v1.0.0", response.Version)
	assert.Len(t, response.Files, 1)
}

// TestGetVersion_NotFound tests version not found error
func TestGetVersion_NotFound(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":    "test-module",
		"version": "v1.0.0",
	})
	w := httptest.NewRecorder()

	server.getVersion(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetFile_Success tests successful file retrieval
func TestGetFile_Success(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	file := &File{
		Path:    "test.proto",
		Content: "syntax = \"proto3\";",
	}
	storage.files["test-module"] = map[string]map[string]*File{
		"v1.0.0": {
			"test.proto": file,
		},
	}

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/files/test.proto", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":    "test-module",
		"version": "v1.0.0",
		"path":    "test.proto",
	})
	w := httptest.NewRecorder()

	server.getFile(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response File
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test.proto", response.Path)
	assert.Equal(t, "syntax = \"proto3\";", response.Content)
}

// TestGetFile_NotFound tests file not found error
func TestGetFile_NotFound(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/files/test.proto", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":    "test-module",
		"version": "v1.0.0",
		"path":    "test.proto",
	})
	w := httptest.NewRecorder()

	server.getFile(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetFile_StorageError tests file retrieval with storage error
func TestGetFile_StorageError(t *testing.T) {
	storage := newMockStorage()
	storage.getFileError = errors.New("storage error")
	server := NewServer(storage, nil)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/files/test.proto", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":    "test-module",
		"version": "v1.0.0",
		"path":    "test.proto",
	})
	w := httptest.NewRecorder()

	server.getFile(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestServeHTTP verifies that the server implements http.Handler
func TestServeHTTP(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	// Create a module first
	storage.modules["test-module"] = &Module{
		Name:        "test-module",
		Description: "Test",
		CreatedAt:   time.Now(),
	}

	req := httptest.NewRequest("GET", "/modules/test-module", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestServer_RegisterRoutes verifies route registration interface
func TestServer_RegisterRoutes(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	// Create a mock route registrar
	mockRegistrar := &mockRouteRegistrar{}

	// This should not panic
	assert.NotPanics(t, func() {
		server.RegisterRoutes(mockRegistrar)
	})

	assert.True(t, mockRegistrar.called)
}

type mockRouteRegistrar struct {
	called bool
}

func (m *mockRouteRegistrar) RegisterRoutes(router *mux.Router) {
	m.called = true
}

// TestCreateModule_EmptyName tests creating module with empty name
func TestCreateModule_EmptyName(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	module := Module{
		Name:        "",
		Description: "Test",
	}
	body, err := json.Marshal(module)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/modules", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.createModule(w, req)

	// Note: Currently no validation for empty name - this would create it
	// This test documents current behavior; consider adding validation
	assert.Equal(t, http.StatusCreated, w.Code)
}

// TestGetModule_VersionsError tests module retrieval when listing versions fails
func TestGetModule_VersionsError(t *testing.T) {
	storage := newMockStorage()
	storage.listVersionsError = errors.New("versions error")
	server := NewServer(storage, nil)

	storage.modules["test-module"] = &Module{
		Name:        "test-module",
		Description: "Test",
		CreatedAt:   time.Now(),
	}

	req := httptest.NewRequest("GET", "/modules/test-module", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.getModule(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestListModules_VersionsError tests listing modules when listing versions fails
func TestListModules_VersionsError(t *testing.T) {
	storage := newMockStorage()
	storage.listVersionsError = errors.New("versions error")
	server := NewServer(storage, nil)

	storage.modules["test-module"] = &Module{
		Name:        "test-module",
		Description: "Test",
		CreatedAt:   time.Now(),
	}

	req := httptest.NewRequest("GET", "/modules", nil)
	w := httptest.NewRecorder()

	server.listModules(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestCreateVersion_WithDependencies tests version creation with dependencies
func TestCreateVersion_WithDependencies(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	version := Version{
		Version:      "v1.0.0",
		Dependencies: []string{"common/v1.0.0", "types/v1.0.0"},
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}
	body, err := json.Marshal(version)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/modules/test-module/versions", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.createVersion(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response Version
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response.Dependencies, 2)
	assert.Contains(t, response.Dependencies, "common/v1.0.0")
}

// TestGetFile_NestedPath tests retrieving file with nested path
func TestGetFile_NestedPath(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	file := &File{
		Path:    "proto/common/types.proto",
		Content: "syntax = \"proto3\";",
	}
	storage.files["test-module"] = map[string]map[string]*File{
		"v1.0.0": {
			"proto/common/types.proto": file,
		},
	}

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/files/proto/common/types.proto", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":    "test-module",
		"version": "v1.0.0",
		"path":    "proto/common/types.proto",
	})
	w := httptest.NewRecorder()

	server.getFile(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response File
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "proto/common/types.proto", response.Path)
}

// Benchmark tests

func BenchmarkCreateModule(b *testing.B) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	module := Module{
		Name:        "bench-module",
		Description: "Benchmark test",
	}
	body, _ := json.Marshal(module)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/modules", bytes.NewReader(body))
		w := httptest.NewRecorder()
		server.createModule(w, req)
	}
}

func BenchmarkGetModule(b *testing.B) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	storage.modules["test-module"] = &Module{
		Name:        "test-module",
		Description: "Test",
		CreatedAt:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/modules/test-module", nil)
		req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
		w := httptest.NewRecorder()
		server.getModule(w, req)
	}
}

func BenchmarkListModules(b *testing.B) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	// Add some test data
	for i := 0; i < 10; i++ {
		name := "module-" + string(rune(i))
		storage.modules[name] = &Module{
			Name:        name,
			Description: "Test module",
			CreatedAt:   time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/modules", nil)
		w := httptest.NewRecorder()
		server.listModules(w, req)
	}
}
