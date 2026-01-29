package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/httputil"
)


// compileWithGenerator compiles a version using the simplified code generator
func (s *Server) compileWithGenerator(version *Version, language Language) (CompilationInfo, error) {
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

	// Create generation request
	req := &codegen.GenerateRequest{
		ModuleName:   version.ModuleName,
		Version:      version.Version,
		ProtoFiles:   protoFiles,
		Dependencies: dependencies,
		Language:     string(language),
		IncludeGRPC:  false, // TODO: Make this configurable
		Options:      make(map[string]string),
	}

	// Generate code
	ctx := context.Background()
	result, err := codegen.GenerateCode(ctx, req, nil)
	if err != nil {
		return CompilationInfo{}, fmt.Errorf("code generation failed: %w", err)
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
	vars := httputil.GetPathVars(r)
	moduleName := vars["name"]
	versionStr := vars["version"]

	// Parse request body
	var req CompileRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	// Validate languages
	if len(req.Languages) == 0 {
		httputil.WriteBadRequest(w, "At least one language must be specified")
		return
	}

	// Get version from storage
	version, err := s.storage.GetVersion(moduleName, versionStr)
	if err != nil {
		httputil.WriteNotFoundError(w, fmt.Sprintf("Version not found: %v", err))
		return
	}

	// Create generation request
	genReq := &codegen.GenerateRequest{
		ModuleName:  moduleName,
		Version:     versionStr,
		ProtoFiles:  s.convertFilesToProtoFiles(version.Files),
		IncludeGRPC: req.IncludeGRPC,
		Options:     req.Options,
	}

	// Compile all requested languages
	ctx := r.Context()
	results, err := codegen.GenerateCodeParallel(ctx, genReq, req.Languages, nil)
	if err != nil {
		// Partial success is OK - return what we have
		httputil.WriteInternalError(w, fmt.Errorf("compilation partially failed: %w", err))
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
		}
	}

	response := CompileResponse{
		JobID:   fmt.Sprintf("%s-%s", moduleName, versionStr),
		Results: jobInfos,
	}

	httputil.WriteSuccess(w, response)
}

// getCompilationJob returns the status of a compilation job
func (s *Server) getCompilationJob(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	jobID := vars["jobId"]

	// Since we no longer track jobs asynchronously, return a simple completed status
	// In the new simplified model, compilations happen synchronously
	httputil.WriteNotFoundError(w, fmt.Sprintf("Job tracking not supported in simplified model. Job ID: %s", jobID))
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
