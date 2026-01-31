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

// TestNewValidationHandlers verifies handler initialization
func TestNewValidationHandlers(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.storage)
}

// TestValidationHandlers_RegisterRoutes verifies all routes are registered
func TestValidationHandlers_RegisterRoutes(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{"POST", "/validate"},
		{"GET", "/modules/test-module/versions/1.0.0/validate"},
		{"POST", "/normalize"},
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

// TestValidateProto_InvalidJSON tests with invalid JSON body
func TestValidateProto_InvalidJSON(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	req := httptest.NewRequest("POST", "/validate", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handlers.validateProto(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestValidateProto_MissingContent tests with missing content field
func TestValidateProto_MissingContent(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	reqBody, _ := json.Marshal(map[string]string{
		"content": "",
	})
	req := httptest.NewRequest("POST", "/validate", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	handlers.validateProto(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestValidateProto_InvalidProto tests with invalid proto syntax
func TestValidateProto_InvalidProto(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	reqBody, _ := json.Marshal(map[string]string{
		"content": "this is not valid proto syntax",
	})
	req := httptest.NewRequest("POST", "/validate", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	handlers.validateProto(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "failed to parse proto")
}

// TestValidateProto_ValidProto tests with valid proto content
func TestValidateProto_ValidProto(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	validProto := `
syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
  int32 id = 2;
}
`

	reqBody, _ := json.Marshal(map[string]interface{}{
		"content": validProto,
	})
	req := httptest.NewRequest("POST", "/validate", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	handlers.validateProto(w, req)

	// Should return 200 or 422 depending on validation result
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusUnprocessableEntity)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "valid")
	assert.Contains(t, response, "errors")
	assert.Contains(t, response, "warnings")
}

// TestValidateProto_WithConfig tests validation with custom config
func TestValidateProto_WithConfig(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	validProto := `
syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
}
`

	reqBody, _ := json.Marshal(map[string]interface{}{
		"content": validProto,
		"config": map[string]bool{
			"enforce_field_number_ranges":   true,
			"require_enum_zero_value":       true,
			"check_naming_conventions":      true,
			"detect_circular_dependencies":  true,
			"detect_unused_imports":         true,
			"check_reserved_fields":         true,
		},
	})
	req := httptest.NewRequest("POST", "/validate", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	handlers.validateProto(w, req)

	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusUnprocessableEntity)
}

// TestValidateVersion_NotFound tests when version not found
func TestValidateVersion_NotFound(t *testing.T) {
	mockStorage := &mockStorage{
		getVersionError: errors.New("version not found"),
	}
	handlers := NewValidationHandlers(mockStorage)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/1.0.0/validate", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module", "version": "1.0.0"})
	w := httptest.NewRecorder()

	handlers.validateVersion(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "version not found")
}

// TestValidateVersion_NoFiles tests when version has no files
func TestValidateVersion_NoFiles(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test-module": {
				"1.0.0": &Version{
					ModuleName:  "test-module",
					Version: "1.0.0",
					Files:   []File{}, // Empty files
				},
			},
		},
	}
	handlers := NewValidationHandlers(mockStorage)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/1.0.0/validate", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module", "version": "1.0.0"})
	w := httptest.NewRecorder()

	handlers.validateVersion(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "no proto files")
}

// TestValidateVersion_InvalidProto tests when proto file is invalid
func TestValidateVersion_InvalidProto(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test-module": {
				"1.0.0": &Version{
					ModuleName:  "test-module",
					Version: "1.0.0",
					Files: []File{
						{
							Path:    "test.proto",
							Content: "invalid proto syntax",
						},
					},
				},
			},
		},
	}
	handlers := NewValidationHandlers(mockStorage)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/1.0.0/validate", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module", "version": "1.0.0"})
	w := httptest.NewRecorder()

	handlers.validateVersion(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestValidateVersion_Success tests successful validation
func TestValidateVersion_Success(t *testing.T) {
	validProto := `
syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
}
`

	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"test-module": {
				"1.0.0": &Version{
					ModuleName:  "test-module",
					Version: "1.0.0",
					Files: []File{
						{
							Path:    "test.proto",
							Content: validProto,
						},
					},
				},
			},
		},
	}
	handlers := NewValidationHandlers(mockStorage)

	req := httptest.NewRequest("GET", "/modules/test-module/versions/1.0.0/validate", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-module", "version": "1.0.0"})
	w := httptest.NewRecorder()

	handlers.validateVersion(w, req)

	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusUnprocessableEntity)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-module", response["module_name"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.Contains(t, response, "valid")
	assert.Contains(t, response, "errors")
	assert.Contains(t, response, "warnings")
}

// TestNormalizeProto_InvalidJSON tests with invalid JSON body
func TestNormalizeProto_InvalidJSON(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	req := httptest.NewRequest("POST", "/normalize", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handlers.normalizeProto(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestNormalizeProto_MissingContent tests with missing content field
func TestNormalizeProto_MissingContent(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	reqBody, _ := json.Marshal(map[string]string{
		"content": "",
	})
	req := httptest.NewRequest("POST", "/normalize", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	handlers.normalizeProto(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestNormalizeProto_InvalidProto tests with invalid proto syntax
func TestNormalizeProto_InvalidProto(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	reqBody, _ := json.Marshal(map[string]string{
		"content": "this is not valid proto syntax",
	})
	req := httptest.NewRequest("POST", "/normalize", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	handlers.normalizeProto(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "failed to normalize")
}

// TestNormalizeProto_Success tests successful normalization
func TestNormalizeProto_Success(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	// Use simple valid proto content
	validProto := `
syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
  int32 id = 2;
}
`

	reqBody, _ := json.Marshal(map[string]interface{}{
		"content": validProto,
	})
	req := httptest.NewRequest("POST", "/normalize", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	handlers.normalizeProto(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "normalized")
}

// TestNormalizeProto_WithConfig tests normalization with custom config
func TestNormalizeProto_WithConfig(t *testing.T) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	validProto := `
syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
  int32 id = 2;
}
`

	reqBody, _ := json.Marshal(map[string]interface{}{
		"content": validProto,
		"config": map[string]bool{
			"sort_fields":                 true,
			"sort_enum_values":            true,
			"sort_imports":                true,
			"canonicalize_imports":        true,
			"preserve_comments":           true,
			"standardize_whitespace":      true,
			"remove_trailing_whitespace":  true,
		},
	})
	req := httptest.NewRequest("POST", "/normalize", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	handlers.normalizeProto(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "normalized")
	normalized := response["normalized"].(string)
	assert.NotEmpty(t, normalized)
}

// Benchmark tests

func BenchmarkValidateProto(b *testing.B) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	validProto := `
syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
  int32 id = 2;
}
`

	reqBody, _ := json.Marshal(map[string]interface{}{
		"content": validProto,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/validate", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()
		handlers.validateProto(w, req)
	}
}

func BenchmarkNormalizeProto(b *testing.B) {
	mockStorage := &mockStorage{}
	handlers := NewValidationHandlers(mockStorage)

	validProto := `
syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
  int32 id = 2;
}
`

	reqBody, _ := json.Marshal(map[string]interface{}{
		"content": validProto,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/normalize", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()
		handlers.normalizeProto(w, req)
	}
}
