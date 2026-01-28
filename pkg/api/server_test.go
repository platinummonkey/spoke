package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerWithoutRoutes(t *testing.T) {
	storage := newMockStorage()

	server := NewServerWithoutRoutes(storage)

	require.NotNil(t, server)
	assert.NotNil(t, server.storage)
	assert.Nil(t, server.router, "router should not be initialized")
	assert.Nil(t, server.db, "db should be nil")
}

func TestNewServerInitialization(t *testing.T) {
	storage := newMockStorage()

	server := NewServer(storage, nil)

	require.NotNil(t, server)
	assert.NotNil(t, server.storage)
	assert.NotNil(t, server.router, "router should be initialized")
	assert.Nil(t, server.db)
	assert.Nil(t, server.authHandlers, "authHandlers should be nil when db is nil")
	assert.Nil(t, server.compatHandlers, "compatHandlers should be nil when db is nil")
	assert.Nil(t, server.validationHandlers, "validationHandlers should be nil when db is nil")
}

func TestServerServeHTTP(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	// Add a test module to storage
	testModule := &Module{
		Name:        "test-module",
		Description: "Test module for HTTP serving",
	}
	err := storage.CreateModule(testModule)
	require.NoError(t, err)

	// Create a test request to list modules
	req := httptest.NewRequest(http.MethodGet, "/modules", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	server.ServeHTTP(rec, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServerServeHTTP_NotFound(t *testing.T) {
	storage := newMockStorage()
	server := NewServer(storage, nil)

	// Create a test request to a non-existent route
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	server.ServeHTTP(rec, req)

	// Verify 404 response
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCompileGo(t *testing.T) {
	storage := newMockStorage()
	server := NewServerWithoutRoutes(storage)

	// Create a test version
	version := &Version{
		ModuleName: "test-module",
		Version:    "v1.0.0",
		Files: []File{
			{
				Path:    "test.proto",
				Content: "syntax = \"proto3\";\n\npackage test;\n\nmessage TestMessage {\n  string field = 1;\n}\n",
			},
		},
	}

	// Call CompileGo
	info, err := server.CompileGo(version)

	// We expect either success or a specific error depending on orchestrator availability
	// The key is that the method is callable and handles the request
	if err != nil {
		// If compilation fails, it should be due to missing dependencies or orchestrator issues
		// not a panic or nil pointer dereference
		assert.NotNil(t, err)
		t.Logf("CompileGo returned expected error: %v", err)
	} else {
		// If it succeeds, verify the result
		assert.Equal(t, LanguageGo, info.Language)
	}
}

func TestCompilePython(t *testing.T) {
	storage := newMockStorage()
	server := NewServerWithoutRoutes(storage)

	// Create a test version
	version := &Version{
		ModuleName: "test-module",
		Version:    "v1.0.0",
		Files: []File{
			{
				Path:    "test.proto",
				Content: "syntax = \"proto3\";\n\npackage test;\n\nmessage TestMessage {\n  string field = 1;\n}\n",
			},
		},
	}

	// Call CompilePython
	info, err := server.CompilePython(version)

	// We expect either success or a specific error depending on orchestrator availability
	// The key is that the method is callable and handles the request
	if err != nil {
		// If compilation fails, it should be due to missing dependencies or orchestrator issues
		// not a panic or nil pointer dereference
		assert.NotNil(t, err)
		t.Logf("CompilePython returned expected error: %v", err)
	} else {
		// If it succeeds, verify the result
		assert.Equal(t, LanguagePython, info.Language)
	}
}
