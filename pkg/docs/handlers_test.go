package docs

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/api"
)

// mockStorage is a mock implementation of api.Storage
type mockStorage struct {
	getVersionFunc func(moduleName, version string) (*api.Version, error)
}

func (m *mockStorage) GetVersion(moduleName, version string) (*api.Version, error) {
	if m.getVersionFunc != nil {
		return m.getVersionFunc(moduleName, version)
	}
	return nil, errors.New("not implemented")
}

// Implement other Storage interface methods (not used in handlers)
func (m *mockStorage) CreateModule(module *api.Module) error                     { return nil }
func (m *mockStorage) GetModule(name string) (*api.Module, error)               { return nil, nil }
func (m *mockStorage) ListModules() ([]*api.Module, error)                      { return nil, nil }
func (m *mockStorage) CreateVersion(version *api.Version) error                 { return nil }
func (m *mockStorage) ListVersions(moduleName string) ([]*api.Version, error)   { return nil, nil }
func (m *mockStorage) UpdateVersion(version *api.Version) error                 { return nil }
func (m *mockStorage) GetFile(moduleName, version, path string) (*api.File, error) { return nil, nil }

// sampleProtoContent returns a valid proto3 content for testing
func sampleProtoContent() string {
	return `
syntax = "proto3";

package test.package;

// User message represents a user
message User {
  // User ID
  string id = 1;
  // User name
  string name = 2;
  // User email
  string email = 3;
}

// UserService provides user operations
service UserService {
  // GetUser retrieves a user by ID
  rpc GetUser(GetUserRequest) returns (User);
}

// GetUserRequest is the request for GetUser
message GetUserRequest {
  string id = 1;
}
`
}

// createMockVersion creates a mock version with proto content
func createMockVersion(moduleName, version, content string) *api.Version {
	return &api.Version{
		ModuleName: moduleName,
		Version:    version,
		Files: []api.File{
			{
				Path:    "test.proto",
				Content: content,
			},
		},
		CreatedAt: time.Now(),
	}
}

func TestNewDocsHandlers(t *testing.T) {
	storage := &mockStorage{}
	handlers := NewDocsHandlers(storage)

	if handlers == nil {
		t.Fatal("Expected handlers to be created")
	}

	if handlers.storage == nil {
		t.Error("Expected storage to be set")
	}

	if handlers.generator == nil {
		t.Error("Expected generator to be initialized")
	}

	if handlers.htmlExporter == nil {
		t.Error("Expected htmlExporter to be initialized")
	}

	if handlers.markdownExporter == nil {
		t.Error("Expected markdownExporter to be initialized")
	}
}

func TestRegisterRoutes(t *testing.T) {
	storage := &mockStorage{}
	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()

	handlers.RegisterRoutes(router)

	// Test that routes are registered by attempting to match them
	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/docs/mymodule/v1.0.0"},
		{"GET", "/docs/mymodule/v1.0.0/markdown"},
		{"GET", "/docs/mymodule/v1.0.0/json"},
		{"GET", "/docs/mymodule/compare"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		match := &mux.RouteMatch{}
		if !router.Match(req, match) {
			t.Errorf("Route %s %s not registered", tt.method, tt.path)
		}
	}
}

func TestGetVersionDocs_Success(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			if moduleName == "testmodule" && version == "v1.0.0" {
				return createMockVersion(moduleName, version, sampleProtoContent()), nil
			}
			return nil, errors.New("not found")
		},
	}

	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected Content-Type to contain text/html, got %s", contentType)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected non-empty response body")
	}

	// Check for some expected HTML content
	if !strings.Contains(body, "<!DOCTYPE html>") && !strings.Contains(body, "<html") {
		t.Error("Expected HTML content in response")
	}
}

func TestGetVersionDocs_VersionNotFound(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return nil, errors.New("version not found")
		},
	}

	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "version not found") {
		t.Errorf("Expected error message about version not found, got: %s", body)
	}
}

func TestGetVersionDocs_NoProtoFiles(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return &api.Version{
				ModuleName: moduleName,
				Version:    version,
				Files:      []api.File{}, // Empty files
			}, nil
		},
	}

	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "no proto files") {
		t.Errorf("Expected error message about no proto files, got: %s", body)
	}
}

func TestGetVersionDocs_InvalidProto(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return createMockVersion(moduleName, version, "invalid proto content {{{"), nil
		},
	}

	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "failed to parse proto") {
		t.Errorf("Expected error message about parsing proto, got: %s", body)
	}
}

func TestGetVersionDocsMarkdown_Success(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			if moduleName == "testmodule" && version == "v1.0.0" {
				return createMockVersion(moduleName, version, sampleProtoContent()), nil
			}
			return nil, errors.New("not found")
		},
	}

	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0/markdown", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/markdown") {
		t.Errorf("Expected Content-Type to contain text/markdown, got %s", contentType)
	}

	contentDisposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "attachment") {
		t.Error("Expected Content-Disposition to contain attachment")
	}
	if !strings.Contains(contentDisposition, "testmodule-v1.0.0.md") {
		t.Errorf("Expected filename in Content-Disposition, got %s", contentDisposition)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected non-empty response body")
	}

	// Check for markdown formatting
	if !strings.Contains(body, "#") {
		t.Error("Expected markdown headers in response")
	}
}

func TestGetVersionDocsMarkdown_VersionNotFound(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return nil, errors.New("version not found")
		},
	}

	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0/markdown", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetVersionDocsMarkdown_NoProtoFiles(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return &api.Version{
				ModuleName: moduleName,
				Version:    version,
				Files:      []api.File{},
			}, nil
		},
	}

	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0/markdown", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetVersionDocsJSON_Success(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			if moduleName == "testmodule" && version == "v1.0.0" {
				return createMockVersion(moduleName, version, sampleProtoContent()), nil
			}
			return nil, errors.New("not found")
		},
	}

	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0/json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain application/json, got %s", contentType)
	}

	body := w.Body.Bytes()
	var doc Documentation
	err := json.Unmarshal(body, &doc)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if doc.PackageName != "test.package" {
		t.Errorf("Expected package name 'test.package', got %s", doc.PackageName)
	}

	if len(doc.Messages) == 0 {
		t.Error("Expected messages in documentation")
	}

	if len(doc.Services) == 0 {
		t.Error("Expected services in documentation")
	}
}

func TestGetVersionDocsJSON_VersionNotFound(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return nil, errors.New("version not found")
		},
	}

	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0/json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetVersionDocsJSON_InvalidProto(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return createMockVersion(moduleName, version, "invalid proto"), nil
		},
	}

	handlers := NewDocsHandlers(storage)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0/json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestCompareVersions_Success(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			if moduleName == "testmodule" {
				if version == "v1.0.0" {
					return createMockVersion(moduleName, version, sampleProtoContent()), nil
				}
				if version == "v2.0.0" {
					// Modified proto with additional field
					modifiedProto := `
syntax = "proto3";
package test.package;

message User {
  string id = 1;
  string name = 2;
  string email = 3;
  string phone = 4;
}

message NewMessage {
  string id = 1;
}

service UserService {
  rpc GetUser(GetUserRequest) returns (User);
}

message GetUserRequest {
  string id = 1;
}
`
					return createMockVersion(moduleName, version, modifiedProto), nil
				}
			}
			return nil, errors.New("not found")
		},
	}

	handlers := NewDocsHandlers(storage)

	// Test directly calling the handler to avoid routing issues
	req := httptest.NewRequest("GET", "/docs/testmodule/compare?old=v1.0.0&new=v2.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{
		"module": "testmodule",
	})
	w := httptest.NewRecorder()

	handlers.compareVersions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain application/json, got %s", contentType)
	}

	body := w.Body.Bytes()
	var response map[string]interface{}
	err := json.Unmarshal(body, &response)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response["old_version"] != "v1.0.0" {
		t.Errorf("Expected old_version 'v1.0.0', got %v", response["old_version"])
	}

	if response["new_version"] != "v2.0.0" {
		t.Errorf("Expected new_version 'v2.0.0', got %v", response["new_version"])
	}

	changes, ok := response["changes"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected changes to be a map")
	}

	messages, ok := changes["messages"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected messages in changes")
	}

	// Verify message changes are detected
	if messages["added"] == nil || messages["removed"] == nil || messages["modified"] == nil {
		t.Error("Expected added, removed, and modified message lists")
	}
}

func TestCompareVersions_MissingParameters(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return createMockVersion(moduleName, version, sampleProtoContent()), nil
		},
	}
	handlers := NewDocsHandlers(storage)

	tests := []struct {
		name  string
		query string
	}{
		{"missing both", "/docs/testmodule/compare"},
		{"missing new", "/docs/testmodule/compare?old=v1.0.0"},
		{"missing old", "/docs/testmodule/compare?new=v2.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.query, nil)
			req = mux.SetURLVars(req, map[string]string{
				"module": "testmodule",
			})
			w := httptest.NewRecorder()

			handlers.compareVersions(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}

			body := w.Body.String()
			if !strings.Contains(body, "old and new version parameters required") {
				t.Errorf("Expected error message about missing parameters, got: %s", body)
			}
		})
	}
}

func TestCompareVersions_OldVersionNotFound(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return nil, errors.New("version not found")
		},
	}

	handlers := NewDocsHandlers(storage)

	req := httptest.NewRequest("GET", "/docs/testmodule/compare?old=v1.0.0&new=v2.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{
		"module": "testmodule",
	})
	w := httptest.NewRecorder()

	handlers.compareVersions(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "old version not found") {
		t.Errorf("Expected error message about old version, got: %s", body)
	}
}

func TestCompareVersions_NewVersionNotFound(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			if version == "v1.0.0" {
				return createMockVersion(moduleName, version, sampleProtoContent()), nil
			}
			return nil, errors.New("version not found")
		},
	}

	handlers := NewDocsHandlers(storage)

	req := httptest.NewRequest("GET", "/docs/testmodule/compare?old=v1.0.0&new=v2.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{
		"module": "testmodule",
	})
	w := httptest.NewRecorder()

	handlers.compareVersions(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "new version not found") {
		t.Errorf("Expected error message about new version, got: %s", body)
	}
}

func TestCompareVersions_InvalidOldProto(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			if version == "v1.0.0" {
				return createMockVersion(moduleName, version, "invalid proto"), nil
			}
			if version == "v2.0.0" {
				return createMockVersion(moduleName, version, sampleProtoContent()), nil
			}
			return nil, errors.New("not found")
		},
	}

	handlers := NewDocsHandlers(storage)

	req := httptest.NewRequest("GET", "/docs/testmodule/compare?old=v1.0.0&new=v2.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{
		"module": "testmodule",
	})
	w := httptest.NewRecorder()

	handlers.compareVersions(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "failed to parse old proto") {
		t.Errorf("Expected error message about parsing old proto, got: %s", body)
	}
}

func TestCompareVersions_InvalidNewProto(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			if version == "v1.0.0" {
				return createMockVersion(moduleName, version, sampleProtoContent()), nil
			}
			if version == "v2.0.0" {
				return createMockVersion(moduleName, version, "invalid proto"), nil
			}
			return nil, errors.New("not found")
		},
	}

	handlers := NewDocsHandlers(storage)

	req := httptest.NewRequest("GET", "/docs/testmodule/compare?old=v1.0.0&new=v2.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{
		"module": "testmodule",
	})
	w := httptest.NewRecorder()

	handlers.compareVersions(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "failed to parse new proto") {
		t.Errorf("Expected error message about parsing new proto, got: %s", body)
	}
}

func TestCompareDocs_MessagesAdded(t *testing.T) {
	handlers := NewDocsHandlers(&mockStorage{})

	oldDoc := &Documentation{
		Messages: []*MessageDoc{
			{Name: "User"},
		},
	}

	newDoc := &Documentation{
		Messages: []*MessageDoc{
			{Name: "User"},
			{Name: "Post"},
		},
	}

	diff := handlers.compareDocs(oldDoc, newDoc)

	messages := diff["messages"].(map[string]interface{})
	added := messages["added"].([]string)

	if len(added) != 1 {
		t.Errorf("Expected 1 added message, got %d", len(added))
	}

	if added[0] != "Post" {
		t.Errorf("Expected 'Post' to be added, got %s", added[0])
	}
}

func TestCompareDocs_MessagesRemoved(t *testing.T) {
	handlers := NewDocsHandlers(&mockStorage{})

	oldDoc := &Documentation{
		Messages: []*MessageDoc{
			{Name: "User"},
			{Name: "Post"},
		},
	}

	newDoc := &Documentation{
		Messages: []*MessageDoc{
			{Name: "User"},
		},
	}

	diff := handlers.compareDocs(oldDoc, newDoc)

	messages := diff["messages"].(map[string]interface{})
	removed := messages["removed"].([]string)

	if len(removed) != 1 {
		t.Errorf("Expected 1 removed message, got %d", len(removed))
	}

	if removed[0] != "Post" {
		t.Errorf("Expected 'Post' to be removed, got %s", removed[0])
	}
}

func TestCompareDocs_MessagesModified(t *testing.T) {
	handlers := NewDocsHandlers(&mockStorage{})

	oldDoc := &Documentation{
		Messages: []*MessageDoc{
			{
				Name: "User",
				Fields: []*FieldDoc{
					{Name: "id", Number: 1, Type: "string"},
					{Name: "name", Number: 2, Type: "string"},
				},
			},
		},
	}

	newDoc := &Documentation{
		Messages: []*MessageDoc{
			{
				Name: "User",
				Fields: []*FieldDoc{
					{Name: "id", Number: 1, Type: "string"},
					{Name: "name", Number: 2, Type: "string"},
					{Name: "email", Number: 3, Type: "string"},
				},
			},
		},
	}

	diff := handlers.compareDocs(oldDoc, newDoc)

	messages := diff["messages"].(map[string]interface{})
	modified := messages["modified"].([]string)

	if len(modified) != 1 {
		t.Errorf("Expected 1 modified message, got %d", len(modified))
	}

	if modified[0] != "User" {
		t.Errorf("Expected 'User' to be modified, got %s", modified[0])
	}
}

func TestCompareDocs_ServicesAdded(t *testing.T) {
	handlers := NewDocsHandlers(&mockStorage{})

	oldDoc := &Documentation{
		Services: []*ServiceDoc{
			{Name: "UserService"},
		},
	}

	newDoc := &Documentation{
		Services: []*ServiceDoc{
			{Name: "UserService"},
			{Name: "PostService"},
		},
	}

	diff := handlers.compareDocs(oldDoc, newDoc)

	services := diff["services"].(map[string]interface{})
	added := services["added"].([]string)

	if len(added) != 1 {
		t.Errorf("Expected 1 added service, got %d", len(added))
	}

	if added[0] != "PostService" {
		t.Errorf("Expected 'PostService' to be added, got %s", added[0])
	}
}

func TestCompareDocs_ServicesRemoved(t *testing.T) {
	handlers := NewDocsHandlers(&mockStorage{})

	oldDoc := &Documentation{
		Services: []*ServiceDoc{
			{Name: "UserService"},
			{Name: "PostService"},
		},
	}

	newDoc := &Documentation{
		Services: []*ServiceDoc{
			{Name: "UserService"},
		},
	}

	diff := handlers.compareDocs(oldDoc, newDoc)

	services := diff["services"].(map[string]interface{})
	removed := services["removed"].([]string)

	if len(removed) != 1 {
		t.Errorf("Expected 1 removed service, got %d", len(removed))
	}

	if removed[0] != "PostService" {
		t.Errorf("Expected 'PostService' to be removed, got %s", removed[0])
	}
}

func TestCompareDocs_EmptyDocuments(t *testing.T) {
	handlers := NewDocsHandlers(&mockStorage{})

	oldDoc := &Documentation{}
	newDoc := &Documentation{}

	diff := handlers.compareDocs(oldDoc, newDoc)

	if diff == nil {
		t.Fatal("Expected diff to be non-nil")
	}

	messages := diff["messages"].(map[string]interface{})
	services := diff["services"].(map[string]interface{})

	if len(messages["added"].([]string)) != 0 {
		t.Error("Expected no added messages")
	}

	if len(services["added"].([]string)) != 0 {
		t.Error("Expected no added services")
	}
}

func TestGetVersionDocs_DirectHandlerCall(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return createMockVersion(moduleName, version, sampleProtoContent()), nil
		},
	}

	handlers := NewDocsHandlers(storage)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{
		"module":  "testmodule",
		"version": "v1.0.0",
	})
	w := httptest.NewRecorder()

	handlers.getVersionDocs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetVersionDocsMarkdown_DirectHandlerCall(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return createMockVersion(moduleName, version, sampleProtoContent()), nil
		},
	}

	handlers := NewDocsHandlers(storage)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0/markdown", nil)
	req = mux.SetURLVars(req, map[string]string{
		"module":  "testmodule",
		"version": "v1.0.0",
	})
	w := httptest.NewRecorder()

	handlers.getVersionDocsMarkdown(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetVersionDocsJSON_DirectHandlerCall(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return createMockVersion(moduleName, version, sampleProtoContent()), nil
		},
	}

	handlers := NewDocsHandlers(storage)

	req := httptest.NewRequest("GET", "/docs/testmodule/v1.0.0/json", nil)
	req = mux.SetURLVars(req, map[string]string{
		"module":  "testmodule",
		"version": "v1.0.0",
	})
	w := httptest.NewRecorder()

	handlers.getVersionDocsJSON(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify response is valid JSON
	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var doc Documentation
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Errorf("Failed to unmarshal JSON response: %v", err)
	}
}

func TestCompareVersions_DirectHandlerCall(t *testing.T) {
	storage := &mockStorage{
		getVersionFunc: func(moduleName, version string) (*api.Version, error) {
			return createMockVersion(moduleName, version, sampleProtoContent()), nil
		},
	}

	handlers := NewDocsHandlers(storage)

	req := httptest.NewRequest("GET", "/docs/testmodule/compare?old=v1.0.0&new=v2.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{
		"module": "testmodule",
	})
	w := httptest.NewRecorder()

	handlers.compareVersions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
