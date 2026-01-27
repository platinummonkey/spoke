package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/orchestrator"
	"github.com/stretchr/testify/assert"
)

// mockOrchestrator implements orchestrator.Orchestrator for testing
type mockOrchestrator struct {
	compileSingleFunc func(ctx context.Context, req *orchestrator.CompileRequest) (*codegen.CompilationResult, error)
	compileAllFunc    func(ctx context.Context, req *orchestrator.CompileRequest, languages []string) ([]*codegen.CompilationResult, error)
	getStatusFunc     func(ctx context.Context, jobID string) (*codegen.CompilationJob, error)
	closeFunc         func() error
}

func (m *mockOrchestrator) CompileSingle(ctx context.Context, req *orchestrator.CompileRequest) (*codegen.CompilationResult, error) {
	if m.compileSingleFunc != nil {
		return m.compileSingleFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockOrchestrator) CompileAll(ctx context.Context, req *orchestrator.CompileRequest, languages []string) ([]*codegen.CompilationResult, error) {
	if m.compileAllFunc != nil {
		return m.compileAllFunc(ctx, req, languages)
	}
	return nil, errors.New("not implemented")
}

func (m *mockOrchestrator) GetStatus(ctx context.Context, jobID string) (*codegen.CompilationJob, error) {
	if m.getStatusFunc != nil {
		return m.getStatusFunc(ctx, jobID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockOrchestrator) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// TestRegisterPackageGenerators tests the placeholder method
func TestRegisterPackageGenerators(t *testing.T) {
	mockStorage := &mockStorage{}
	server := &Server{
		storage:      mockStorage,
		orchestrator: nil,
	}

	// Should not panic when orchestrator is nil
	server.registerPackageGenerators()

	server.orchestrator = &mockOrchestrator{}
	server.registerPackageGenerators()
}

// TestGetCodeGenVersion tests version detection from environment
func TestGetCodeGenVersion(t *testing.T) {
	server := &Server{}

	// Test default version
	os.Unsetenv("SPOKE_CODEGEN_VERSION")
	version := server.getCodeGenVersion()
	assert.Equal(t, "v2", version)

	// Test custom version
	os.Setenv("SPOKE_CODEGEN_VERSION", "v3")
	version = server.getCodeGenVersion()
	assert.Equal(t, "v3", version)

	// Clean up
	os.Unsetenv("SPOKE_CODEGEN_VERSION")
}

// TestCompileWithOrchestrator_NoOrchestrator tests error when orchestrator unavailable
func TestCompileWithOrchestrator_NoOrchestrator(t *testing.T) {
	server := &Server{
		orchestrator: nil,
	}

	version := &Version{
		ModuleName: "test-module",
		Version:    "1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	_, err := server.compileWithOrchestrator(version, LanguageGo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "orchestrator not available")
}

// TestCompileWithOrchestrator_Success tests successful compilation
func TestCompileWithOrchestrator_Success(t *testing.T) {
	mockOrch := &mockOrchestrator{
		compileSingleFunc: func(ctx context.Context, req *orchestrator.CompileRequest) (*codegen.CompilationResult, error) {
			return &codegen.CompilationResult{
				Success:  true,
				Language: "go",
				GeneratedFiles: []codegen.GeneratedFile{
					{Path: "test.pb.go", Content: []byte("package test")},
				},
				PackageFiles: []codegen.GeneratedFile{
					{Path: "go.mod", Content: []byte("module test")},
				},
			}, nil
		},
	}

	server := &Server{
		storage:      &mockStorage{},
		orchestrator: mockOrch,
	}

	version := &Version{
		ModuleName: "test-module",
		Version:    "1.0.0",
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	info, err := server.compileWithOrchestrator(version, LanguageGo)
	assert.NoError(t, err)
	assert.Equal(t, LanguageGo, info.Language)
	assert.Equal(t, "test-module", info.PackageName)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Len(t, info.Files, 2)
}

// TestCompileWithOrchestrator_WithDependencies tests compilation with dependencies
func TestCompileWithOrchestrator_WithDependencies(t *testing.T) {
	mockStorage := &mockStorage{
		versions: map[string]map[string]*Version{
			"dep-module": {
				"1.0.0": &Version{
					ModuleName: "dep-module",
					Version:    "1.0.0",
					Files: []File{
						{Path: "dep.proto", Content: "syntax = \"proto3\";"},
					},
				},
			},
		},
	}

	mockOrch := &mockOrchestrator{
		compileSingleFunc: func(ctx context.Context, req *orchestrator.CompileRequest) (*codegen.CompilationResult, error) {
			// Verify dependencies were included
			assert.Len(t, req.Dependencies, 1)
			assert.Equal(t, "dep-module", req.Dependencies[0].ModuleName)
			return &codegen.CompilationResult{
				Success:  true,
				Language: "go",
			}, nil
		},
	}

	server := &Server{
		storage:      mockStorage,
		orchestrator: mockOrch,
	}

	version := &Version{
		ModuleName:   "test-module",
		Version:      "1.0.0",
		Dependencies: []string{"dep-module@1.0.0"},
		Files: []File{
			{Path: "test.proto", Content: "syntax = \"proto3\";"},
		},
	}

	_, err := server.compileWithOrchestrator(version, LanguageGo)
	assert.NoError(t, err)
}

// TestCompileVersion_NoOrchestrator tests when orchestrator unavailable
func TestCompileVersion_NoOrchestrator(t *testing.T) {
	server := &Server{
		storage:      &mockStorage{},
		orchestrator: nil,
	}

	req := httptest.NewRequest("POST", "/modules/test/versions/1.0.0/compile", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test", "version": "1.0.0"})
	w := httptest.NewRecorder()

	server.compileVersion(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "not available")
}

// TestCompileVersion_InvalidJSON tests with invalid JSON body
func TestCompileVersion_InvalidJSON(t *testing.T) {
	server := &Server{
		storage:      &mockStorage{},
		orchestrator: &mockOrchestrator{},
	}

	req := httptest.NewRequest("POST", "/modules/test/versions/1.0.0/compile",
		bytes.NewBufferString("invalid json"))
	req = mux.SetURLVars(req, map[string]string{"name": "test", "version": "1.0.0"})
	w := httptest.NewRecorder()

	server.compileVersion(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCompileVersion_NoLanguages tests with empty languages list
func TestCompileVersion_NoLanguages(t *testing.T) {
	server := &Server{
		storage:      &mockStorage{},
		orchestrator: &mockOrchestrator{},
	}

	reqBody, _ := json.Marshal(CompileRequest{
		Languages: []string{},
	})
	req := httptest.NewRequest("POST", "/modules/test/versions/1.0.0/compile",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test", "version": "1.0.0"})
	w := httptest.NewRecorder()

	server.compileVersion(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "At least one language")
}

// TestCompileVersion_VersionNotFound tests when version doesn't exist
func TestCompileVersion_VersionNotFound(t *testing.T) {
	mockStorage := &mockStorage{
		getVersionError: errors.New("version not found"),
	}

	server := &Server{
		storage:      mockStorage,
		orchestrator: &mockOrchestrator{},
	}

	reqBody, _ := json.Marshal(CompileRequest{
		Languages: []string{"go"},
	})
	req := httptest.NewRequest("POST", "/modules/test/versions/1.0.0/compile",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test", "version": "1.0.0"})
	w := httptest.NewRecorder()

	server.compileVersion(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

// TestCompileVersion_Success tests successful compilation
func TestCompileVersion_Success(t *testing.T) {
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

	mockOrch := &mockOrchestrator{
		compileAllFunc: func(ctx context.Context, req *orchestrator.CompileRequest, languages []string) ([]*codegen.CompilationResult, error) {
			results := make([]*codegen.CompilationResult, len(languages))
			for i, lang := range languages {
				results[i] = &codegen.CompilationResult{
					Success:  true,
					Language: lang,
					Duration: 1000 * time.Millisecond,
					CacheHit: false,
				}
			}
			return results, nil
		},
	}

	server := &Server{
		storage:      mockStorage,
		orchestrator: mockOrch,
	}

	reqBody, _ := json.Marshal(CompileRequest{
		Languages:   []string{"go", "python"},
		IncludeGRPC: true,
		Options:     map[string]string{"optimize": "true"},
	})
	req := httptest.NewRequest("POST", "/modules/test-module/versions/1.0.0/compile",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test-module", "version": "1.0.0"})
	w := httptest.NewRecorder()

	server.compileVersion(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response CompileResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-module-1.0.0", response.JobID)
	assert.Len(t, response.Results, 2)
	assert.Equal(t, "go", response.Results[0].Language)
	assert.Equal(t, "python", response.Results[1].Language)
}

// TestGetCompilationJob_NoOrchestrator tests when orchestrator unavailable
func TestGetCompilationJob_NoOrchestrator(t *testing.T) {
	server := &Server{
		orchestrator: nil,
	}

	req := httptest.NewRequest("GET", "/api/v1/jobs/test-job-123", nil)
	req = mux.SetURLVars(req, map[string]string{"jobId": "test-job-123"})
	w := httptest.NewRecorder()

	server.getCompilationJob(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "not available")
}

// TestGetCompilationJob_NotFound tests when job doesn't exist
func TestGetCompilationJob_NotFound(t *testing.T) {
	mockOrch := &mockOrchestrator{
		getStatusFunc: func(ctx context.Context, jobID string) (*codegen.CompilationJob, error) {
			return nil, errors.New("job not found")
		},
	}

	server := &Server{
		orchestrator: mockOrch,
	}

	req := httptest.NewRequest("GET", "/api/v1/jobs/test-job-123", nil)
	req = mux.SetURLVars(req, map[string]string{"jobId": "test-job-123"})
	w := httptest.NewRecorder()

	server.getCompilationJob(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

// TestGetCompilationJob_Success tests successful job retrieval
func TestGetCompilationJob_Success(t *testing.T) {
	now := time.Now()
	completed := now.Add(5 * time.Second)
	mockOrch := &mockOrchestrator{
		getStatusFunc: func(ctx context.Context, jobID string) (*codegen.CompilationJob, error) {
			return &codegen.CompilationJob{
				ID:          jobID,
				Language:    "go",
				Status:      codegen.JobStatusCompleted,
				StartedAt:   &now,
				CompletedAt: &completed,
				CacheHit:    true,
				Result: &codegen.CompilationResult{
					Success:  true,
					Language: "go",
					Duration: 5 * time.Second,
					S3Key:    "s3://bucket/key",
					S3Bucket: "spoke-artifacts",
				},
			}, nil
		},
	}

	server := &Server{
		orchestrator: mockOrch,
	}

	req := httptest.NewRequest("GET", "/api/v1/jobs/test-job-123", nil)
	req = mux.SetURLVars(req, map[string]string{"jobId": "test-job-123"})
	w := httptest.NewRecorder()

	server.getCompilationJob(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response CompilationJobInfo
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-job-123", response.ID)
	assert.Equal(t, "go", response.Language)
	assert.Equal(t, string(codegen.JobStatusCompleted), response.Status)
	assert.True(t, response.CacheHit)
	assert.Equal(t, "s3://bucket/key", response.S3Key)
	assert.Equal(t, "spoke-artifacts", response.S3Bucket)
}

// TestConvertFilesToProtoFiles tests the helper function
func TestConvertFilesToProtoFiles(t *testing.T) {
	server := &Server{}

	files := []File{
		{Path: "test1.proto", Content: "syntax = \"proto3\";"},
		{Path: "test2.proto", Content: "package test;"},
	}

	protoFiles := server.convertFilesToProtoFiles(files)

	assert.Len(t, protoFiles, 2)
	assert.Equal(t, "test1.proto", protoFiles[0].Path)
	assert.Equal(t, []byte("syntax = \"proto3\";"), protoFiles[0].Content)
	assert.Equal(t, "test2.proto", protoFiles[1].Path)
	assert.Equal(t, []byte("package test;"), protoFiles[1].Content)
}

// TestGetStatusFromResult tests the status determination helper
func TestGetStatusFromResult(t *testing.T) {
	tests := []struct {
		name     string
		result   *codegen.CompilationResult
		expected string
	}{
		{
			name:     "successful compilation",
			result:   &codegen.CompilationResult{Success: true, Error: ""},
			expected: "completed",
		},
		{
			name:     "failed compilation",
			result:   &codegen.CompilationResult{Success: false, Error: "syntax error"},
			expected: "failed",
		},
		{
			name:     "running compilation",
			result:   &codegen.CompilationResult{Success: false, Error: ""},
			expected: "running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := getStatusFromResult(tt.result)
			assert.Equal(t, tt.expected, status)
		})
	}
}
