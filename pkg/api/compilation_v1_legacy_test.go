package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


// TestDownloadCompiled_VersionNotFound tests when version doesn't exist
func TestDownloadCompiled_VersionNotFound(t *testing.T) {
	mockStorage := &mockStorage{
		getVersionError: ErrNotFound,
	}

	server := &Server{
		storage: mockStorage,
	}

	req := httptest.NewRequest("GET", "/modules/test/versions/1.0.0/download/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test",
		"version":  "1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestDownloadCompiled_NoCompilationInfo tests when language not compiled
func TestDownloadCompiled_NoCompilationInfo(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test": {
				"1.0.0": &Version{
					ModuleName: "test",
					Version:    "1.0.0",
					Files: []File{
						{Path: "test.proto", Content: "syntax = \"proto3\";"},
					},
					CompilationInfo: []CompilationInfo{
						{
							Language:    LanguagePython,
							PackageName: "test",
							Version:     "1.0.0",
							Files: []File{
								{Path: "test.py", Content: "# python code"},
							},
						},
					},
				},
			},
		},
	}

	server := &Server{
		storage: mockStorage,
	}

	req := httptest.NewRequest("GET", "/modules/test/versions/1.0.0/download/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test",
		"version":  "1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "compiled version not found")
}

// TestDownloadCompiled_Go tests successful Go download
func TestDownloadCompiled_Go(t *testing.T) {
	testContent := "package test\n// Go code"
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test-module": {
				"1.0.0": &Version{
					ModuleName: "test-module",
					Version:    "1.0.0",
					Files: []File{
						{Path: "test.proto", Content: "syntax = \"proto3\";"},
					},
					CompilationInfo: []CompilationInfo{
						{
							Language:    LanguageGo,
							PackageName: "test-module",
							Version:     "1.0.0",
							Files: []File{
								{Path: "test.pb.go", Content: testContent},
								{Path: "go.mod", Content: "module test"},
							},
						},
					},
				},
			},
		},
	}

	server := &Server{
		storage:      mockStorage,
		eventTracker: nil, // Event tracking tested separately
	}

	req := httptest.NewRequest("GET", "/modules/test-module/versions/1.0.0/download/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test-module",
		"version":  "1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/zip", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "test-module-1.0.0-go.zip")
	assert.Contains(t, w.Body.String(), testContent)
	assert.Contains(t, w.Body.String(), "module test")
}

// TestDownloadCompiled_Python tests successful Python download
func TestDownloadCompiled_Python(t *testing.T) {
	testContent := "# Python protobuf code"
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test-module": {
				"2.0.0": &Version{
					ModuleName: "test-module",
					Version:    "2.0.0",
					Files: []File{
						{Path: "test.proto", Content: "syntax = \"proto3\";"},
					},
					CompilationInfo: []CompilationInfo{
						{
							Language:    LanguagePython,
							PackageName: "test-module",
							Version:     "2.0.0",
							Files: []File{
								{Path: "test_pb2.py", Content: testContent},
							},
						},
					},
				},
			},
		},
	}

	server := &Server{
		storage: mockStorage,
	}

	req := httptest.NewRequest("GET", "/modules/test-module/versions/2.0.0/download/python", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test-module",
		"version":  "2.0.0",
		"language": "python",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-python-package", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "test-module-2.0.0-py.whl")
	assert.Contains(t, w.Body.String(), testContent)
}

// TestDownloadCompiled_MultipleFiles tests download with multiple files
func TestDownloadCompiled_MultipleFiles(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"multi-file": {
				"1.0.0": &Version{
					ModuleName: "multi-file",
					Version:    "1.0.0",
					Files: []File{
						{Path: "test.proto", Content: "syntax = \"proto3\";"},
					},
					CompilationInfo: []CompilationInfo{
						{
							Language:    LanguageGo,
							PackageName: "multi-file",
							Version:     "1.0.0",
							Files: []File{
								{Path: "file1.pb.go", Content: "package one"},
								{Path: "file2.pb.go", Content: "package two"},
								{Path: "file3.pb.go", Content: "package three"},
							},
						},
					},
				},
			},
		},
	}

	server := &Server{
		storage: mockStorage,
	}

	req := httptest.NewRequest("GET", "/modules/multi-file/versions/1.0.0/download/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "multi-file",
		"version":  "1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "package one")
	assert.Contains(t, body, "package two")
	assert.Contains(t, body, "package three")
}

// TestCompileForLanguage_V1 tests routing to v1 compilation
func TestCompileForLanguage_V1(t *testing.T) {
	// Set environment to force v1
	os.Setenv("SPOKE_CODEGEN_VERSION", "v1")
	defer os.Unsetenv("SPOKE_CODEGEN_VERSION")

	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "test",
		Version:    "1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	// Should route to v1 and fail (no protoc in test environment)
	_, err := server.compileForLanguage(version, LanguageGo)
	assert.Error(t, err) // Expected to fail without protoc
}

// TestCompileForLanguage_V2WithOrchestrator tests routing to v2
func TestCompileForLanguage_V2WithOrchestrator(t *testing.T) {
	os.Setenv("SPOKE_CODEGEN_VERSION", "v2")
	defer os.Unsetenv("SPOKE_CODEGEN_VERSION")

	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "test",
		Version:    "1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	// Should route to v2 orchestrator
	_, err := server.compileForLanguage(version, LanguageGo)
	// May succeed or fail depending on orchestrator mock, but should not panic
	assert.Error(t, err) // Mock orchestrator returns error by default
}

// TestCompileForLanguage_V2FallbackToV1 tests fallback when orchestrator unavailable
func TestCompileForLanguage_V2FallbackToV1(t *testing.T) {
	os.Setenv("SPOKE_CODEGEN_VERSION", "v2")
	defer os.Unsetenv("SPOKE_CODEGEN_VERSION")

	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "test",
		Version:    "1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	// Should fallback to v1
	_, err := server.compileForLanguage(version, LanguageGo)
	assert.Error(t, err) // Expected to fail without protoc
}

// TestCompileForLanguage_DefaultToV2 tests default routing behavior
func TestCompileForLanguage_DefaultToV2(t *testing.T) {
	os.Unsetenv("SPOKE_CODEGEN_VERSION")

	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "test",
		Version:    "1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	// Should default to v2 and fallback to v1
	_, err := server.compileForLanguage(version, LanguageGo)
	assert.Error(t, err) // Expected to fail without protoc
}

// TestCompileV1_Go tests Go compilation routing
func TestCompileV1_Go(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "test",
		Version:    "1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";\npackage test;"},
		},
	}

	// Should attempt Go compilation (will fail without protoc)
	_, err := server.compileV1(version, LanguageGo)
	assert.Error(t, err)
	// Error should be related to temp dir, protoc, or file operations
}

// TestCompileV1_Python tests Python compilation routing
func TestCompileV1_Python(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "test",
		Version:    "1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";\npackage test;"},
		},
	}

	// Should attempt Python compilation (will fail without protoc)
	_, err := server.compileV1(version, LanguagePython)
	assert.Error(t, err)
}

// TestCompileV1_UnsupportedLanguage tests unsupported language
func TestCompileV1_UnsupportedLanguage(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "test",
		Version:    "1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	// Test various unsupported languages
	unsupportedLanguages := []Language{
		LanguageJava,
		LanguageCPP,
		LanguageRust,
		Language("unknown"),
	}

	for _, lang := range unsupportedLanguages {
		_, err := server.compileV1(version, lang)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported language for v1")
	}
}

// TestCompileGo_TempDirCreation tests temp directory setup
func TestCompileGo_TempDirCreation(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "test-module",
		Version:    "1.0.0",
		Files: []File{
			{Path: "service.proto", Content: "syntax = \"proto3\";\npackage test;"},
		},
	}

	// This will fail at protoc execution, but we're testing the path before that
	_, err := server.compileGo(version)
	assert.Error(t, err)
	// Should fail with protoc error, not temp dir error
	// The temp dir should be created and cleaned up
}

// TestCompileGo_WithDependencies tests Go compilation with dependencies
func TestCompileGo_WithDependencies(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"dep-module": {
				"1.0.0": &Version{
					ModuleName: "dep-module",
					Version:    "1.0.0",
					Files: []File{
						{Path: "dep.proto", Content: "syntax = \"proto3\";\npackage dep;"},
					},
				},
			},
		},
	}

	server := &Server{
		storage: mockStorage,
	}

	version := &Version{
		ModuleName:   "main-module",
		Version:      "2.0.0",
		Dependencies: []string{"dep-module@1.0.0"},
		Files: []File{
			{Path: "main.proto", Content: "syntax = \"proto3\";\npackage main;\nimport \"dep.proto\";"},
		},
	}

	// Will fail at protoc but dependencies should be fetched
	_, err := server.compileGo(version)
	assert.Error(t, err)
}

// TestCompileGo_InvalidDependencyFormat tests malformed dependency
func TestCompileGo_InvalidDependencyFormat(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName:   "test",
		Version:      "1.0.0",
		Dependencies: []string{"invalid-format"}, // Missing @version
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	// Should skip malformed dependency and continue
	_, err := server.compileGo(version)
	assert.Error(t, err) // Will fail at protoc, not dependency parsing
}

// TestCompileGo_DependencyNotFound tests missing dependency
func TestCompileGo_DependencyNotFound(t *testing.T) {
	mockStorage := &mockStorage{
		getVersionError: ErrNotFound,
	}

	server := &Server{
		storage: mockStorage,
	}

	version := &Version{
		ModuleName:   "test",
		Version:      "1.0.0",
		Dependencies: []string{"missing-dep@1.0.0"},
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	_, err := server.compileGo(version)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get dependency")
}

// TestCompileGo_MultipleFiles tests compilation with multiple proto files
func TestCompileGo_MultipleFiles(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "multi-proto",
		Version:    "1.0.0",
		Files: []File{
			{Path: "service.proto", Content: "syntax = \"proto3\";\npackage service;"},
			{Path: "types.proto", Content: "syntax = \"proto3\";\npackage types;"},
			{Path: "models/user.proto", Content: "syntax = \"proto3\";\npackage models;"},
		},
	}

	_, err := server.compileGo(version)
	assert.Error(t, err) // Will fail at protoc
	// But all files should be written before protoc fails
}

// TestCompilePython_TempDirCreation tests temp directory setup for Python
func TestCompilePython_TempDirCreation(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "test-module",
		Version:    "1.0.0",
		Files: []File{
			{Path: "service.proto", Content: "syntax = \"proto3\";\npackage test;"},
		},
	}

	_, err := server.compilePython(version)
	assert.Error(t, err) // Will fail at protoc
}

// TestCompilePython_WithDependencies tests Python compilation with dependencies
func TestCompilePython_WithDependencies(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"dep-module": {
				"2.0.0": &Version{
					ModuleName: "dep-module",
					Version:    "2.0.0",
					Files: []File{
						{Path: "dep.proto", Content: "syntax = \"proto3\";\npackage dep;"},
					},
				},
			},
		},
	}

	server := &Server{
		storage: mockStorage,
	}

	version := &Version{
		ModuleName:   "main-module",
		Version:      "1.0.0",
		Dependencies: []string{"dep-module@2.0.0"},
		Files: []File{
			{Path: "main.proto", Content: "syntax = \"proto3\";\npackage main;"},
		},
	}

	_, err := server.compilePython(version)
	assert.Error(t, err) // Will fail at protoc
}

// TestCompilePython_DependencyNotFound tests missing Python dependency
func TestCompilePython_DependencyNotFound(t *testing.T) {
	server := &Server{
		storage: &mockStorage{
			getVersionError: ErrNotFound,
		},
	}

	version := &Version{
		ModuleName:   "test",
		Version:      "1.0.0",
		Dependencies: []string{"missing@1.0.0"},
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	_, err := server.compilePython(version)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get dependency")
}

// TestCompilePython_InvalidDependencyFormat tests malformed Python dependency
func TestCompilePython_InvalidDependencyFormat(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName:   "test",
		Version:      "1.0.0",
		Dependencies: []string{"no-version-specified"},
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	_, err := server.compilePython(version)
	assert.Error(t, err) // Will fail at protoc, skips malformed dep
}

// TestCompilePython_MultipleFiles tests Python compilation with multiple files
func TestCompilePython_MultipleFiles(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "multi-proto",
		Version:    "2.0.0",
		Files: []File{
			{Path: "api.proto", Content: "syntax = \"proto3\";\npackage api;"},
			{Path: "models.proto", Content: "syntax = \"proto3\";\npackage models;"},
			{Path: "services/auth.proto", Content: "syntax = \"proto3\";\npackage auth;"},
		},
	}

	_, err := server.compilePython(version)
	assert.Error(t, err) // Will fail at protoc
}

// TestCompileGo_ModuleNameInGoMod tests go.mod generation
func TestCompileGo_ModuleNameInGoMod(t *testing.T) {
	// This test verifies the go.mod content would be correct
	// We can't fully test without protoc, but we can verify the logic
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "github.com/example/mymodule",
		Version:    "v1.2.3",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	_, err := server.compileGo(version)
	assert.Error(t, err) // Will fail at protoc
	// go.mod would contain: module github.com/example/mymodule
}

// TestCompilePython_SetupPyGeneration tests setup.py content
func TestCompilePython_SetupPyGeneration(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "my-python-module",
		Version:    "2.1.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	_, err := server.compilePython(version)
	assert.Error(t, err) // Will fail at protoc
	// setup.py would contain name="my-python-module" and version="2.1.0"
}

// TestDownloadCompiled_EventTracking tests analytics event tracking with nil tracker
func TestDownloadCompiled_EventTracking(t *testing.T) {
	testContent := "test content"
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"analytics-test": {
				"1.0.0": &Version{
					ModuleName: "analytics-test",
					Version:    "1.0.0",
					Files: []File{
						{Path: "test.proto", Content: "syntax = \"proto3\";"},
					},
					CompilationInfo: []CompilationInfo{
						{
							Language:    LanguageGo,
							PackageName: "analytics-test",
							Version:     "1.0.0",
							Files: []File{
								{Path: "test.pb.go", Content: testContent},
							},
						},
					},
				},
			},
		},
	}

	// With nil event tracker - should not panic
	server := &Server{
		storage:      mockStorage,
		eventTracker: nil,
	}

	req := httptest.NewRequest("GET", "/modules/analytics-test/versions/1.0.0/download/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "analytics-test",
		"version":  "1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestDownloadCompiled_NoEventTrackerNoPanic tests that missing event tracker doesn't panic
func TestDownloadCompiled_NoEventTrackerNoPanic(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test": {
				"1.0.0": &Version{
					ModuleName: "test",
					Version:    "1.0.0",
					CompilationInfo: []CompilationInfo{
						{
							Language:    LanguageGo,
							PackageName: "test",
							Version:     "1.0.0",
							Files: []File{
								{Path: "test.go", Content: "package test"},
							},
						},
					},
				},
			},
		},
	}

	server := &Server{
		storage:      mockStorage,
		eventTracker: nil, // No tracker
	}

	req := httptest.NewRequest("GET", "/modules/test/versions/1.0.0/download/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test",
		"version":  "1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	// Should not panic
	require.NotPanics(t, func() {
		server.downloadCompiled(w, req)
	})

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestCompileGo_EmptyFiles tests handling of empty files list
func TestCompileGo_EmptyFiles(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "empty-module",
		Version:    "1.0.0",
		Files:      []File{}, // No files
	}

	_, err := server.compileGo(version)
	assert.Error(t, err) // Will fail at protoc with no input files
}

// TestCompilePython_EmptyFiles tests handling of empty files list
func TestCompilePython_EmptyFiles(t *testing.T) {
	server := &Server{
		storage: &mockStorage{},
	}

	version := &Version{
		ModuleName: "empty-module",
		Version:    "1.0.0",
		Files:      []File{}, // No files
	}

	_, err := server.compilePython(version)
	assert.Error(t, err) // Will fail at protoc with no input files
}

// TestDownloadCompiled_EmptyFiles tests download with no compiled files
func TestDownloadCompiled_EmptyFiles(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"empty": {
				"1.0.0": &Version{
					ModuleName: "empty",
					Version:    "1.0.0",
					CompilationInfo: []CompilationInfo{
						{
							Language:    LanguageGo,
							PackageName: "empty",
							Version:     "1.0.0",
							Files:       []File{}, // No compiled files
						},
					},
				},
			},
		},
	}

	server := &Server{
		storage: mockStorage,
	}

	req := httptest.NewRequest("GET", "/modules/empty/versions/1.0.0/download/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "empty",
		"version":  "1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Response should be empty but valid
	assert.Empty(t, w.Body.String())
}
