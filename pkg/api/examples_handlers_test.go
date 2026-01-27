package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// TestGetExamples_Success tests successful example generation
func TestGetExamples_Success(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	storage.versions["test-module"] = map[string]*Version{
		"v1.0.0": &Version{
			ModuleName: "test-module",
			Version:    "v1.0.0",
			Files: []File{
				{Path: "test.proto", Content: "syntax = \"proto3\";"},
				{Path: "example.proto", Content: "syntax = \"proto3\";"},
			},
			CreatedAt: time.Now(),
		},
	}

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/examples/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test-module",
		"version":  "v1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.getExamples(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
	body := w.Body.String()
	assert.Contains(t, body, "test-module")
	assert.Contains(t, body, "v1.0.0")
	assert.Contains(t, body, "go")
	assert.Contains(t, body, "Proto files: 2")
}

// TestGetExamples_VersionNotFound tests when version doesn't exist
func TestGetExamples_VersionNotFound(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/examples/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test-module",
		"version":  "v1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.getExamples(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Version not found")
}

// TestGetExamples_DifferentLanguages tests examples for different languages
func TestGetExamples_DifferentLanguages(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	storage.versions["test-module"] = map[string]*Version{
		"v1.0.0": &Version{
			ModuleName: "test-module",
			Version:    "v1.0.0",
			Files: []File{
				{Path: "test.proto", Content: "syntax = \"proto3\";"},
			},
			CreatedAt: time.Now(),
		},
	}

	languages := []string{"go", "python", "java", "cpp"}
	for _, lang := range languages {
		req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/examples/"+lang, nil)
		req = mux.SetURLVars(req, map[string]string{
			"name":     "test-module",
			"version":  "v1.0.0",
			"language": lang,
		})
		w := httptest.NewRecorder()

		server.getExamples(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Failed for language: "+lang)
		assert.Contains(t, w.Body.String(), lang, "Body should contain language: "+lang)
	}
}

// TestGetExamples_EmptyFiles tests with version having no files
func TestGetExamples_EmptyFiles(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	storage.versions["test-module"] = map[string]*Version{
		"v1.0.0": &Version{
			ModuleName: "test-module",
			Version:    "v1.0.0",
			Files:      []File{},
			CreatedAt:  time.Now(),
		},
	}

	req := httptest.NewRequest("GET", "/modules/test-module/versions/v1.0.0/examples/go", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name":     "test-module",
		"version":  "v1.0.0",
		"language": "go",
	})
	w := httptest.NewRecorder()

	server.getExamples(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Proto files: 0")
}
