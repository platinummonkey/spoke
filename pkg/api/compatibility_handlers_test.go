package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// TestNewCompatibilityHandlers verifies handler initialization
func TestNewCompatibilityHandlers(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewCompatibilityHandlers(mockStorage)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.storage)
}

// TestCompatibilityHandlers_RegisterRoutes verifies all routes are registered
func TestCompatibilityHandlers_RegisterRoutes(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewCompatibilityHandlers(mockStorage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{"POST", "/modules/test-module/compatibility"},
		{"GET", "/modules/test-module/versions/1.0.0/compatibility"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			var match mux.RouteMatch
			matched := router.Match(req, &match)
			assert.True(t, matched, "Route %s %s should be registered", tt.method, tt.path)
		})
	}
}

// TestCheckCompatibility_InvalidJSON tests with invalid JSON body
func TestCheckCompatibility_InvalidJSON(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewCompatibilityHandlers(mockStorage)

	req := httptest.NewRequest("POST", "/modules/test-module/compatibility",
		bytes.NewBufferString("invalid json"))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	handlers.checkCompatibility(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCheckCompatibility_MissingOldVersion tests with missing old_version field
func TestCheckCompatibility_MissingOldVersion(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewCompatibilityHandlers(mockStorage)

	reqBody, _ := json.Marshal(map[string]string{
		"old_version": "",
		"new_version": "2.0.0",
	})
	req := httptest.NewRequest("POST", "/modules/test-module/compatibility",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	handlers.checkCompatibility(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCheckCompatibility_MissingNewVersion tests with missing new_version field
func TestCheckCompatibility_MissingNewVersion(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewCompatibilityHandlers(mockStorage)

	reqBody, _ := json.Marshal(map[string]string{
		"old_version": "1.0.0",
		"new_version": "",
	})
	req := httptest.NewRequest("POST", "/modules/test-module/compatibility",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	handlers.checkCompatibility(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCheckCompatibility_InvalidMode tests with invalid compatibility mode
func TestCheckCompatibility_InvalidMode(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewCompatibilityHandlers(mockStorage)

	reqBody, _ := json.Marshal(map[string]string{
		"old_version": "1.0.0",
		"new_version": "2.0.0",
		"mode":        "INVALID_MODE",
	})
	req := httptest.NewRequest("POST", "/modules/test-module/compatibility",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	handlers.checkCompatibility(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid compatibility mode")
}

// TestCheckCompatibility_OldVersionNotFound tests when old version doesn't exist
func TestCheckCompatibility_OldVersionNotFound(t *testing.T) {
	mockStorage := &mockStorage{
		getVersionError: errors.New("version not found"),
	}
	handlers := NewCompatibilityHandlers(mockStorage)

	reqBody, _ := json.Marshal(map[string]string{
		"old_version": "1.0.0",
		"new_version": "2.0.0",
	})
	req := httptest.NewRequest("POST", "/modules/test-module/compatibility",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	handlers.checkCompatibility(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "old version not found")
}

// TestCheckCompatibility_NewVersionNotFound tests when new version doesn't exist
func TestCheckCompatibility_NewVersionNotFound(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test-module": {
				"1.0.0": &Version{
					ModuleName: "test-module",
					Version:    "1.0.0",
					Files: []File{
						{Path: "test.proto", Content: "syntax = \"proto3\";"},
					},
				},
				// Intentionally not adding 2.0.0
			},
		},
	}
	handlers := NewCompatibilityHandlers(mockStorage)

	reqBody, _ := json.Marshal(map[string]string{
		"old_version": "1.0.0",
		"new_version": "2.0.0",
	})
	req := httptest.NewRequest("POST", "/modules/test-module/compatibility",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	handlers.checkCompatibility(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "new version not found")
}

// TestCheckCompatibility_InvalidOldProto tests when old proto file is invalid
func TestCheckCompatibility_InvalidOldProto(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test-module": {
				"1.0.0": &Version{
					ModuleName: "test-module",
					Version:    "1.0.0",
					Files: []File{
						{Path: "test.proto", Content: "invalid proto syntax"},
					},
				},
				"2.0.0": &Version{
					ModuleName: "test-module",
					Version:    "2.0.0",
					Files: []File{
						{Path: "test.proto", Content: "syntax = \"proto3\";"},
					},
				},
			},
		},
	}
	handlers := NewCompatibilityHandlers(mockStorage)

	reqBody, _ := json.Marshal(map[string]string{
		"old_version": "1.0.0",
		"new_version": "2.0.0",
	})
	req := httptest.NewRequest("POST", "/modules/test-module/compatibility",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
	w := httptest.NewRecorder()

	handlers.checkCompatibility(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestCheckVersionCompatibility_VersionNotFound tests when version doesn't exist
func TestCheckVersionCompatibility_VersionNotFound(t *testing.T) {
	mockStorage := &mockStorage{
		getVersionError: errors.New("version not found"),
	}
	handlers := NewCompatibilityHandlers(mockStorage)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/1.0.0/compatibility", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module", "version": "1.0.0"})
	w := httptest.NewRecorder()

	handlers.checkVersionCompatibility(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "version not found")
}

// TestCheckVersionCompatibility_InvalidMode tests with invalid compatibility mode
func TestCheckVersionCompatibility_InvalidMode(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test-module": {
				"1.0.0": &Version{
					ModuleName: "test-module",
					Version:    "1.0.0",
					Files: []File{
						{Path: "test.proto", Content: "syntax = \"proto3\";"},
					},
				},
			},
		},
	}
	handlers := NewCompatibilityHandlers(mockStorage)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/1.0.0/compatibility?mode=INVALID", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module", "version": "1.0.0"})
	w := httptest.NewRecorder()

	handlers.checkVersionCompatibility(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid compatibility mode")
}

// TestCheckVersionCompatibility_ListVersionsError tests error listing versions
func TestCheckVersionCompatibility_ListVersionsError(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test-module": {
				"1.0.0": &Version{
					ModuleName: "test-module",
					Version:    "1.0.0",
					Files: []File{
						{Path: "test.proto", Content: "syntax = \"proto3\";"},
					},
				},
			},
		},
		listVersionsError: errors.New("database error"),
	}
	handlers := NewCompatibilityHandlers(mockStorage)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/1.0.0/compatibility", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module", "version": "1.0.0"})
	w := httptest.NewRecorder()

	handlers.checkVersionCompatibility(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestCheckVersionCompatibility_SingleVersion tests with only one version
func TestCheckVersionCompatibility_SingleVersion(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test-module": {
				"1.0.0": &Version{
					ModuleName: "test-module",
					Version:    "1.0.0",
					Files: []File{
						{Path: "test.proto", Content: "syntax = \"proto3\";"},
					},
				},
			},
		},
	}
	handlers := NewCompatibilityHandlers(mockStorage)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/1.0.0/compatibility", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module", "version": "1.0.0"})
	w := httptest.NewRecorder()

	handlers.checkVersionCompatibility(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "No previous version")
}

// TestCheckCompatibility_ValidModes tests all valid compatibility modes
func TestCheckCompatibility_ValidModes(t *testing.T) {
	validModes := []string{
		"BACKWARD",
		"FORWARD",
		"FULL",
		"BACKWARD_TRANSITIVE",
		"FORWARD_TRANSITIVE",
		"FULL_TRANSITIVE",
		"NONE",
	}

	for _, mode := range validModes {
		t.Run("Mode_"+mode, func(t *testing.T) {
			mockStorage := &mockStorage{
				getVersionError: errors.New("stop early for test"),
			}
			handlers := NewCompatibilityHandlers(mockStorage)

			reqBody, _ := json.Marshal(map[string]string{
				"old_version": "1.0.0",
				"new_version": "2.0.0",
				"mode":        mode,
			})
			req := httptest.NewRequest("POST", "/modules/test-module/compatibility",
				bytes.NewBuffer(reqBody))
			req = mux.SetURLVars(req, map[string]string{"name": "test-module"})
			w := httptest.NewRecorder()

			handlers.checkCompatibility(w, req)

			// Should fail at version lookup, not mode parsing
			assert.Equal(t, http.StatusNotFound, w.Code)
			assert.Contains(t, w.Body.String(), "old version not found")
		})
	}
}
