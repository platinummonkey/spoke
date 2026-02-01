package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestDownloadCompiled_VersionNotFound(t *testing.T) {
	mockStore := newMockStorage()
	server := NewServer(mockStore, nil)

	req := httptest.NewRequest("GET", "/modules/nonexistent/versions/v1.0.0/download/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "nonexistent",
		"version":  "v1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDownloadCompiled_NoCompilationInfo(t *testing.T) {
	mockStore := newMockStorage()
	server := NewServer(mockStore, nil)

	// Create a module and version without compilation info
	module := &Module{
		Name:        "test-module",
		Description: "Test module",
	}
	err := mockStore.CreateModule(module)
	assert.NoError(t, err)

	version := &Version{
		ModuleName: "test-module",
		Version:    "v1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
		// No CompilationInfo
	}
	err = mockStore.CreateVersion(version)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/download/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test-module",
		"version":  "v1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "compiled version not found")
}

func TestDownloadCompiled_Success_Go(t *testing.T) {
	mockStore := newMockStorage()
	server := NewServer(mockStore, nil)

	// Create a module and version with compilation info
	module := &Module{
		Name:        "test-module",
		Description: "Test module",
	}
	err := mockStore.CreateModule(module)
	assert.NoError(t, err)

	version := &Version{
		ModuleName: "test-module",
		Version:    "v1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
		CompilationInfo: []CompilationInfo{
			{
				Language: LanguageGo,
				Files: []File{
					{Path: "test.pb.go", Content: "package test"},
				},
			},
		},
	}
	err = mockStore.CreateVersion(version)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/download/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test-module",
		"version":  "v1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/zip", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "test-module-v1.0.0-go.zip")
	assert.Equal(t, "package test", w.Body.String())
}

func TestDownloadCompiled_Success_Python(t *testing.T) {
	mockStore := newMockStorage()
	server := NewServer(mockStore, nil)

	// Create a module and version with Python compilation info
	module := &Module{
		Name:        "test-module",
		Description: "Test module",
	}
	err := mockStore.CreateModule(module)
	assert.NoError(t, err)

	version := &Version{
		ModuleName: "test-module",
		Version:    "v1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
		CompilationInfo: []CompilationInfo{
			{
				Language: LanguagePython,
				Files: []File{
					{Path: "test_pb2.py", Content: "# Python code"},
				},
			},
		},
	}
	err = mockStore.CreateVersion(version)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/download/python", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test-module",
		"version":  "v1.0.0",
		"language": "python",
	})
	w := httptest.NewRecorder()

	server.downloadCompiled(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-python-package", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "test-module-v1.0.0-py.whl")
	assert.Equal(t, "# Python code", w.Body.String())
}

func TestCompileForLanguage(t *testing.T) {
	mockStore := newMockStorage()
	server := NewServer(mockStore, nil)

	version := &Version{
		ModuleName: "test-module",
		Version:    "v1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	// This will fail because we don't have a real compiler set up,
	// but it tests that the function exists and can be called
	_, err := server.compileForLanguage(version, LanguageGo)
	// We expect an error since we don't have a real compiler
	assert.Error(t, err)
}
