package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompareDiff_Success tests successful diff comparison
func TestCompareDiff_Success(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	// Create two versions
	storage.versions["test-module"] = map[string]*Version{
		"v1.0.0": &Version{
			ModuleName: "test-module",
			Version:    "v1.0.0",
			Files: []File{
				{Path: "test.proto", Content: "syntax = \"proto3\";"},
			},
			CreatedAt: time.Now(),
		},
		"v1.1.0": &Version{
			ModuleName: "test-module",
			Version:    "v1.1.0",
			Files: []File{
				{Path: "test.proto", Content: "syntax = \"proto3\";\nmessage Test {}"},
				{Path: "new.proto", Content: "syntax = \"proto3\";"},
			},
			CreatedAt: time.Now(),
		},
	}

	reqBody, _ := json.Marshal(DiffRequest{
		FromVersion: "v1.0.0",
		ToVersion:   "v1.1.0",
	})
	req := httptest.NewRequest("POST", "/modules/test-module/diff", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.compareDiff(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", response["from_version"])
	assert.Equal(t, "v1.1.0", response["to_version"])
	assert.NotNil(t, response["changes"])
}

// TestCompareDiff_InvalidJSON tests with invalid JSON
func TestCompareDiff_InvalidJSON(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	req := httptest.NewRequest("POST", "/modules/test-module/diff", bytes.NewBufferString("invalid json"))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.compareDiff(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCompareDiff_FromVersionNotFound tests when from version doesn't exist
func TestCompareDiff_FromVersionNotFound(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	reqBody, _ := json.Marshal(DiffRequest{
		FromVersion: "v1.0.0",
		ToVersion:   "v1.1.0",
	})
	req := httptest.NewRequest("POST", "/modules/test-module/diff", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.compareDiff(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "From version not found")
}

// TestCompareDiff_ToVersionNotFound tests when to version doesn't exist
func TestCompareDiff_ToVersionNotFound(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	// Create only the from version
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

	reqBody, _ := json.Marshal(DiffRequest{
		FromVersion: "v1.0.0",
		ToVersion:   "v1.1.0",
	})
	req := httptest.NewRequest("POST", "/modules/test-module/diff", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.compareDiff(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "To version not found")
}

// TestCompareDiff_MissingVersions tests with empty version strings
func TestCompareDiff_MissingVersions(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	reqBody, _ := json.Marshal(DiffRequest{
		FromVersion: "",
		ToVersion:   "",
	})
	req := httptest.NewRequest("POST", "/modules/test-module/diff", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	server.compareDiff(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
