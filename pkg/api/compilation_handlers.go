package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/orchestrator"
)

// registerPackageGenerators registers package generators with the orchestrator
func (s *Server) registerPackageGenerators() {
	if s.orchestrator == nil {
		return
	}

	// Access the internal orchestrator to get the package registry
	// Note: This is a bit of a hack - ideally orchestrator would expose a method
	// For now, we'll register generators when creating the orchestrator itself
	// This method is a placeholder for future enhancement

	// TODO: Add method to orchestrator to register generators externally
	// For now, generators are registered internally in the orchestrator
}

// getCodeGenVersion returns the code generation version to use based on environment variable
func (s *Server) getCodeGenVersion() string {
	version := os.Getenv("SPOKE_CODEGEN_VERSION")
	if version == "" {
		return "v2" // Default to v2 (new orchestrator)
	}
	return version
}

// compileWithOrchestrator compiles a version using the v2 orchestrator
func (s *Server) compileWithOrchestrator(version *Version, language Language) (CompilationInfo, error) {
	if s.orchestrator == nil {
		return CompilationInfo{}, fmt.Errorf("orchestrator not available")
	}

	// Convert Version to proto files
	protoFiles := make([]codegen.ProtoFile, 0, len(version.Files))
	for _, file := range version.Files {
		protoFiles = append(protoFiles, codegen.ProtoFile{
			Path:    file.Path,
			Content: []byte(file.Content),
		})
	}

	// Convert dependencies
	dependencies := make([]codegen.Dependency, 0, len(version.Dependencies))
	for _, dep := range version.Dependencies {
		parts := strings.Split(dep, "@")
		if len(parts) != 2 {
			continue
		}
		depModule := parts[0]
		depVersion := parts[1]

		// Fetch dependency proto files
		depVer, err := s.storage.GetVersion(depModule, depVersion)
		if err != nil {
			continue // Skip invalid dependencies
		}

		depProtoFiles := make([]codegen.ProtoFile, 0, len(depVer.Files))
		for _, file := range depVer.Files {
			depProtoFiles = append(depProtoFiles, codegen.ProtoFile{
				Path:    file.Path,
				Content: []byte(file.Content),
			})
		}

		dependencies = append(dependencies, codegen.Dependency{
			ModuleName: depModule,
			Version:    depVersion,
			ProtoFiles: depProtoFiles,
		})
	}

	// Create compilation request
	req := &orchestrator.CompileRequest{
		ModuleName:   version.ModuleName,
		Version:      version.Version,
		VersionID:    0, // Not tracked in current system
		ProtoFiles:   protoFiles,
		Dependencies: dependencies,
		Language:     string(language),
		IncludeGRPC:  false, // TODO: Make this configurable
		Options:      make(map[string]string),
		StorageDir:   "", // Will be set by orchestrator
		S3Bucket:     "", // TODO: Configure from server settings
	}

	// Compile using orchestrator
	ctx := context.Background()
	result, err := s.orchestrator.CompileSingle(ctx, req)
	if err != nil {
		return CompilationInfo{}, fmt.Errorf("orchestrator compilation failed: %w", err)
	}

	// Convert result to CompilationInfo
	files := make([]File, 0, len(result.GeneratedFiles)+len(result.PackageFiles))

	// Add generated files
	for _, gf := range result.GeneratedFiles {
		files = append(files, File{
			Path:    gf.Path,
			Content: string(gf.Content),
		})
	}

	// Add package manager files
	for _, pf := range result.PackageFiles {
		files = append(files, File{
			Path:    pf.Path,
			Content: string(pf.Content),
		})
	}

	return CompilationInfo{
		Language:    language,
		PackageName: version.ModuleName,
		Version:     version.Version,
		Files:       files,
	}, nil
}

// compileVersion triggers compilation for a version
func (s *Server) compileVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	versionStr := vars["version"]

	// Check if orchestrator is available
	if s.orchestrator == nil {
		http.Error(w, "Code generation orchestrator not available", http.StatusServiceUnavailable)
		return
	}

	// Parse request body
	var req CompileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate languages
	if len(req.Languages) == 0 {
		http.Error(w, "At least one language must be specified", http.StatusBadRequest)
		return
	}

	// Get version from storage
	version, err := s.storage.GetVersion(moduleName, versionStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Version not found: %v", err), http.StatusNotFound)
		return
	}

	// Convert to orchestrator request
	orchReq := &orchestrator.CompileRequest{
		ModuleName:  moduleName,
		Version:     versionStr,
		ProtoFiles:  s.convertFilesToProtoFiles(version.Files),
		IncludeGRPC: req.IncludeGRPC,
		Options:     req.Options,
	}

	// Compile all requested languages
	ctx := r.Context()
	results, err := s.orchestrator.CompileAll(ctx, orchReq, req.Languages)
	if err != nil {
		// Partial success is OK - return what we have
		http.Error(w, fmt.Sprintf("Compilation partially failed: %v", err), http.StatusInternalServerError)
	}

	// Convert results to response
	jobInfos := make([]CompilationJobInfo, len(results))
	for i, result := range results {
		jobInfos[i] = CompilationJobInfo{
			ID:       fmt.Sprintf("%s-%s-%s", moduleName, versionStr, result.Language),
			Language: result.Language,
			Status:   getStatusFromResult(result),
			Duration: result.Duration.Milliseconds(),
			CacheHit: result.CacheHit,
			Error:    result.Error,
			S3Key:    result.S3Key,
			S3Bucket: result.S3Bucket,
		}
	}

	response := CompileResponse{
		JobID:   fmt.Sprintf("%s-%s", moduleName, versionStr),
		Results: jobInfos,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getCompilationJob returns the status of a compilation job
func (s *Server) getCompilationJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	// Check if orchestrator is available
	if s.orchestrator == nil {
		http.Error(w, "Code generation orchestrator not available", http.StatusServiceUnavailable)
		return
	}

	// Get job status from orchestrator
	ctx := r.Context()
	job, err := s.orchestrator.GetStatus(ctx, jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Job not found: %v", err), http.StatusNotFound)
		return
	}

	// Convert to API response
	jobInfo := CompilationJobInfo{
		ID:          job.ID,
		Language:    job.Language,
		Status:      string(job.Status),
		StartedAt:   job.StartedAt,
		CompletedAt: job.CompletedAt,
		CacheHit:    job.CacheHit,
		Error:       job.Error,
	}

	if job.Result != nil {
		jobInfo.Duration = job.Result.Duration.Milliseconds()
		jobInfo.S3Key = job.Result.S3Key
		jobInfo.S3Bucket = job.Result.S3Bucket
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobInfo)
}

// Helper functions

func (s *Server) convertFilesToProtoFiles(files []File) []codegen.ProtoFile {
	protoFiles := make([]codegen.ProtoFile, len(files))
	for i, file := range files {
		protoFiles[i] = codegen.ProtoFile{
			Path:    file.Path,
			Content: []byte(file.Content),
		}
	}
	return protoFiles
}

func getStatusFromResult(result *codegen.CompilationResult) string {
	if result.Success {
		return "completed"
	}
	if result.Error != "" {
		return "failed"
	}
	return "running"
}
