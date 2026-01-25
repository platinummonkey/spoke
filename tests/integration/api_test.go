package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platinummonkey/spoke/pkg/api"
)

// TestAPILanguagesList tests the GET /api/v1/languages endpoint
func TestAPILanguagesList(t *testing.T) {
	// Create test server
	storage := &mockStorage{}
	server := api.NewServer(storage, nil)

	req := httptest.NewRequest("GET", "/api/v1/languages", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var languages []api.LanguageInfo
	if err := json.NewDecoder(w.Body).Decode(&languages); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should return 15 languages
	if len(languages) != 15 {
		t.Errorf("Expected 15 languages, got %d", len(languages))
	}

	// Verify specific languages exist
	languageIDs := make(map[string]bool)
	for _, lang := range languages {
		languageIDs[lang.ID] = true

		// Verify required fields
		if lang.ID == "" {
			t.Error("Language ID is empty")
		}
		if lang.Name == "" {
			t.Error("Language name is empty")
		}
		if lang.PluginVersion == "" {
			t.Error("Plugin version is empty")
		}
	}

	// Check for key languages
	expectedLanguages := []string{"go", "python", "java", "rust", "typescript"}
	for _, langID := range expectedLanguages {
		if !languageIDs[langID] {
			t.Errorf("Expected language %s not found", langID)
		}
	}
}

// TestAPILanguageDetails tests the GET /api/v1/languages/{id} endpoint
func TestAPILanguageDetails(t *testing.T) {
	storage := &mockStorage{}
	server := api.NewServer(storage, nil)

	testCases := []struct {
		languageID string
		wantStatus int
	}{
		{"go", http.StatusOK},
		{"python", http.StatusOK},
		{"rust", http.StatusOK},
		{"nonexistent", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.languageID, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/languages/"+tc.languageID, nil)
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Expected status %d, got %d", tc.wantStatus, w.Code)
			}

			if tc.wantStatus == http.StatusOK {
				var lang api.LanguageInfo
				if err := json.NewDecoder(w.Body).Decode(&lang); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if lang.ID != tc.languageID {
					t.Errorf("Expected ID %s, got %s", tc.languageID, lang.ID)
				}
			}
		})
	}
}

// TestAPICompileVersion tests the POST /api/v1/modules/{name}/versions/{ver}/compile endpoint
func TestAPICompileVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API integration test in short mode")
	}

	storage := &mockStorage{
		versions: map[string]*api.Version{
			"test-module:v1.0.0": {
				ModuleName: "test-module",
				Version:    "v1.0.0",
				Files: []api.File{
					{
						Path: "test.proto",
						Content: `syntax = "proto3";
package test;
message Test { string data = 1; }`,
					},
				},
			},
		},
	}
	server := api.NewServer(storage, nil)

	// Test compile request
	compileReq := api.CompileRequest{
		Languages:   []string{"go", "python"},
		IncludeGRPC: true,
	}

	body, _ := json.Marshal(compileReq)
	req := httptest.NewRequest("POST", "/api/v1/modules/test-module/versions/v1.0.0/compile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	// May return 503 if orchestrator not available in test env
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d: %s", w.Code, w.Body.String())
	}

	if w.Code == http.StatusOK {
		var resp api.CompileResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if resp.JobID == "" {
			t.Error("Expected job ID")
		}

		if len(resp.Results) != len(compileReq.Languages) {
			t.Errorf("Expected %d results, got %d", len(compileReq.Languages), len(resp.Results))
		}
	}
}

// TestAPICompileVersionValidation tests request validation
func TestAPICompileVersionValidation(t *testing.T) {
	storage := &mockStorage{}
	server := api.NewServer(storage, nil)

	testCases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "Empty languages",
			body:       `{"languages":[],"include_grpc":true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid JSON",
			body:       `{invalid json}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Valid request",
			body:       `{"languages":["go"],"include_grpc":false}`,
			wantStatus: http.StatusServiceUnavailable, // Orchestrator not available
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/modules/test/versions/v1.0.0/compile",
				bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Expected status %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// mockStorage implements the Storage interface for testing
type mockStorage struct {
	versions map[string]*api.Version
}

func (m *mockStorage) CreateModule(module *api.Module) error {
	return nil
}

func (m *mockStorage) GetModule(name string) (*api.Module, error) {
	return &api.Module{Name: name}, nil
}

func (m *mockStorage) ListModules() ([]*api.Module, error) {
	return nil, nil
}

func (m *mockStorage) CreateVersion(version *api.Version) error {
	if m.versions == nil {
		m.versions = make(map[string]*api.Version)
	}
	key := version.ModuleName + ":" + version.Version
	m.versions[key] = version
	return nil
}

func (m *mockStorage) GetVersion(moduleName, version string) (*api.Version, error) {
	if m.versions == nil {
		return nil, api.ErrNotFound
	}
	key := moduleName + ":" + version
	v, ok := m.versions[key]
	if !ok {
		return nil, api.ErrNotFound
	}
	return v, nil
}

func (m *mockStorage) ListVersions(moduleName string) ([]*api.Version, error) {
	return nil, nil
}

func (m *mockStorage) UpdateVersion(version *api.Version) error {
	return nil
}

func (m *mockStorage) GetFile(moduleName, version, path string) (*api.File, error) {
	return nil, nil
}
