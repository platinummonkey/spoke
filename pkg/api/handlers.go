package api

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/platinummonkey/spoke/pkg/codegen/orchestrator"
	"github.com/platinummonkey/spoke/pkg/search"
)

// Server represents our API server
type Server struct {
	storage             Storage
	router              *mux.Router
	db                  *sql.DB
	authHandlers        *AuthHandlers
	compatHandlers      *CompatibilityHandlers
	validationHandlers  *ValidationHandlers
	orchestrator        orchestrator.Orchestrator // Code generation orchestrator (v2)
	searchIndexer       *search.Indexer            // Search indexer for proto entities
}

// NewServer creates a new API server
func NewServer(storage Storage, db *sql.DB) *Server {
	s := &Server{
		storage: storage,
		router:  mux.NewRouter(),
		db:      db,
	}

	// Initialize handlers if database is provided
	if db != nil {
		s.authHandlers = NewAuthHandlers(db)
		s.compatHandlers = NewCompatibilityHandlers(storage)
		s.validationHandlers = NewValidationHandlers(storage)

		// Initialize search indexer
		storageAdapter := NewSearchStorageAdapter(storage)
		s.searchIndexer = search.NewIndexer(db, storageAdapter)
	}

	// Initialize code generation orchestrator (v2)
	// Note: Errors are non-fatal - falls back to v1 compilation if orchestrator fails
	if orch, err := orchestrator.NewOrchestrator(nil); err == nil {
		s.orchestrator = orch
		// Register package generators
		s.registerPackageGenerators()
	}

	s.setupRoutes()
	return s
}

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

// setupRoutes configures all the API routes
func (s *Server) setupRoutes() {
	// Module routes
	s.router.HandleFunc("/modules", s.createModule).Methods("POST")
	s.router.HandleFunc("/modules", s.listModules).Methods("GET")
	s.router.HandleFunc("/modules/{name}", s.getModule).Methods("GET")

	// Version routes
	s.router.HandleFunc("/modules/{name}/versions", s.createVersion).Methods("POST")
	s.router.HandleFunc("/modules/{name}/versions", s.listVersions).Methods("GET")
	s.router.HandleFunc("/modules/{name}/versions/{version}", s.getVersion).Methods("GET")

	// File routes
	s.router.HandleFunc("/modules/{name}/versions/{version}/files/{path:.*}", s.getFile).Methods("GET")

	// Download compilation results
	s.router.HandleFunc("/modules/{name}/versions/{version}/download/{language}", s.downloadCompiled).Methods("GET")

	// Language routes (v2 API)
	s.router.HandleFunc("/api/v1/languages", s.listLanguages).Methods("GET")
	s.router.HandleFunc("/api/v1/languages/{id}", s.getLanguage).Methods("GET")

	// Compilation routes (v2 API)
	s.router.HandleFunc("/api/v1/modules/{name}/versions/{version}/compile", s.compileVersion).Methods("POST")
	s.router.HandleFunc("/api/v1/modules/{name}/versions/{version}/compile/{jobId}", s.getCompilationJob).Methods("GET")

	// Example generation routes
	s.router.HandleFunc("/api/v1/modules/{name}/versions/{version}/examples/{language}", s.getExamples).Methods("GET")

	// Diff routes
	s.router.HandleFunc("/api/v1/modules/{name}/diff", s.compareDiff).Methods("POST")

	// Register authentication routes (if database is available)
	if s.authHandlers != nil {
		s.authHandlers.RegisterRoutes(s.router)
	}

	// Register compatibility routes
	if s.compatHandlers != nil {
		s.compatHandlers.RegisterRoutes(s.router)
	}

	// Register validation routes
	if s.validationHandlers != nil {
		s.validationHandlers.RegisterRoutes(s.router)
	}
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// RouteRegistrar is an interface for types that can register routes
type RouteRegistrar interface {
	RegisterRoutes(router *mux.Router)
}

// RegisterRoutes registers routes from a RouteRegistrar
func (s *Server) RegisterRoutes(registrar RouteRegistrar) {
	registrar.RegisterRoutes(s.router)
}

// createModule handles POST /modules
func (s *Server) createModule(w http.ResponseWriter, r *http.Request) {
	var module Module
	if err := json.NewDecoder(r.Body).Decode(&module); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	module.CreatedAt = time.Now()
	module.UpdatedAt = time.Now()

	if err := s.storage.CreateModule(&module); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(module)
}

// listModules handles GET /modules
func (s *Server) listModules(w http.ResponseWriter, r *http.Request) {
	modules, err := s.storage.ListModules()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get versions for each module
	modulesWithVersions := make([]struct {
		*Module
		Versions []*Version `json:"versions"`
	}, len(modules))

	for i, module := range modules {
		versions, err := s.storage.ListVersions(module.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Sort versions by newest first
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].CreatedAt.After(versions[j].CreatedAt)
		})

		modulesWithVersions[i] = struct {
			*Module
			Versions []*Version `json:"versions"`
		}{
			Module:   module,
			Versions: versions,
		}
	}

	json.NewEncoder(w).Encode(modulesWithVersions)
}

// getModule handles GET /modules/{name}
func (s *Server) getModule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	module, err := s.storage.GetModule(vars["name"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get versions for this module
	versions, err := s.storage.ListVersions(vars["name"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sort versions by newest first
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})

	// Add versions to the module response
	moduleWithVersions := struct {
		*Module
		Versions []*Version `json:"versions"`
	}{
		Module:   module,
		Versions: versions,
	}

	json.NewEncoder(w).Encode(moduleWithVersions)
}

// createVersion handles POST /modules/{name}/versions
func (s *Server) createVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var version Version
	if err := json.NewDecoder(r.Body).Decode(&version); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	version.ModuleName = vars["name"]
	version.CreatedAt = time.Now()

	if err := s.storage.CreateVersion(&version); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Trigger search indexing asynchronously (don't block the response)
	if s.searchIndexer != nil {
		go func() {
			ctx := context.Background()
			if err := s.searchIndexer.IndexVersion(ctx, version.ModuleName, version.Version); err != nil {
				log.Printf("Failed to index version %s/%s: %v", version.ModuleName, version.Version, err)
				// Don't fail the request - indexing is non-critical
			}
		}()
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(version)
}

// listVersions handles GET /modules/{name}/versions
func (s *Server) listVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	versions, err := s.storage.ListVersions(vars["name"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(versions)
}

// getVersion handles GET /modules/{name}/versions/{version}
func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	version, err := s.storage.GetVersion(vars["name"], vars["version"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(version)
}

// getFile handles GET /modules/{name}/versions/{version}/files/{path}
func (s *Server) getFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	file, err := s.storage.GetFile(vars["name"], vars["version"], vars["path"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(file)
}

// downloadCompiled handles GET /modules/{name}/versions/{version}/download/{language}
func (s *Server) downloadCompiled(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	language := Language(vars["language"])

	// Get the version
	version, err := s.storage.GetVersion(vars["name"], vars["version"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Find compilation info for the requested language
	var compilationInfo *CompilationInfo
	for _, info := range version.CompilationInfo {
		if info.Language == language {
			compilationInfo = &info
			break
		}
	}

	if compilationInfo == nil {
		http.Error(w, "compiled version not found", http.StatusNotFound)
		return
	}

	// Set appropriate headers based on language
	switch language {
	case LanguageGo:
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s-go.zip", vars["name"], vars["version"]))
	case LanguagePython:
		w.Header().Set("Content-Type", "application/x-python-package")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s-py.whl", vars["name"], vars["version"]))
	}

	// Stream the compiled files
	for _, file := range compilationInfo.Files {
		if _, err := w.Write([]byte(file.Content)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// compileForLanguage routes compilation to v1 or v2 based on feature flag
func (s *Server) compileForLanguage(version *Version, language Language) (CompilationInfo, error) {
	codeGenVersion := s.getCodeGenVersion()

	// Route to appropriate implementation
	switch codeGenVersion {
	case "v1":
		// Use legacy direct protoc compilation
		return s.compileV1(version, language)
	case "v2":
		// Use new orchestrator (default)
		if s.orchestrator != nil {
			return s.compileWithOrchestrator(version, language)
		}
		// Fallback to v1 if orchestrator unavailable
		return s.compileV1(version, language)
	default:
		// Default to v2
		if s.orchestrator != nil {
			return s.compileWithOrchestrator(version, language)
		}
		return s.compileV1(version, language)
	}
}

// compileV1 routes to legacy compilation methods
func (s *Server) compileV1(version *Version, language Language) (CompilationInfo, error) {
	switch language {
	case LanguageGo:
		return s.compileGo(version)
	case LanguagePython:
		return s.compilePython(version)
	default:
		return CompilationInfo{}, fmt.Errorf("unsupported language for v1: %s", language)
	}
}

// compileGo compiles a version into Go code (legacy v1 implementation)
func (s *Server) compileGo(version *Version) (CompilationInfo, error) {
	// Create a temporary directory for compilation
	tmpDir, err := os.MkdirTemp("", "spoke-go-compile-*")
	if err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the output directory for Go files
	goOutDir := filepath.Join(tmpDir, "go")
	if err := os.MkdirAll(goOutDir, 0755); err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to create go output dir: %w", err)
	}

	// Create a directory for all proto files
	protoDir := filepath.Join(tmpDir, "proto")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to create proto dir: %w", err)
	}

	// Write all proto files to the temp directory
	for _, file := range version.Files {
		filePath := filepath.Join(protoDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return CompilationInfo{}, fmt.Errorf("failed to create proto file dir: %w", err)
		}
		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return CompilationInfo{}, fmt.Errorf("failed to write proto file: %w", err)
		}
	}

	// Handle dependencies
	for _, dep := range version.Dependencies {
		parts := strings.Split(dep, "@")
		if len(parts) != 2 {
			continue
		}
		depModule := parts[0]
		depVersion := parts[1]

		// Get the dependency version
		depVer, err := s.storage.GetVersion(depModule, depVersion)
		if err != nil {
			return CompilationInfo{}, fmt.Errorf("failed to get dependency %s@%s: %w", depModule, depVersion, err)
		}

		// Write dependency proto files
		for _, file := range depVer.Files {
			filePath := filepath.Join(protoDir, depModule, file.Path)
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return CompilationInfo{}, fmt.Errorf("failed to create dependency proto file dir: %w", err)
			}
			if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
				return CompilationInfo{}, fmt.Errorf("failed to write dependency proto file: %w", err)
			}
		}
	}

	// Create go.mod file
	goModContent := fmt.Sprintf(`module %s

go 1.21

require (
	google.golang.org/protobuf v1.31.0
)`, version.ModuleName)
	if err := os.WriteFile(filepath.Join(goOutDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to write go.mod: %w", err)
	}

	// Run protoc to generate Go code
	protoFiles := make([]string, 0, len(version.Files))
	for _, file := range version.Files {
		protoFiles = append(protoFiles, filepath.Join(protoDir, file.Path))
	}

	args := append([]string{
		"--go_out=" + goOutDir,
		"--go_opt=paths=source_relative",
		"-I" + protoDir,
	}, protoFiles...)
	cmd := exec.Command("protoc", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return CompilationInfo{}, fmt.Errorf("protoc failed: %s: %w", output, err)
	}

	// Create a zip file containing the generated Go code
	zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuf)

	// Walk through the generated files and add them to the zip
	err = filepath.Walk(goOutDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Get the relative path for the zip file
		relPath, err := filepath.Rel(goOutDir, path)
		if err != nil {
			return err
		}

		// Create a new file in the zip
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// Read and write the file content
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = zipFile.Write(content)
		return err
	})
	if err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to create zip: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to close zip: %w", err)
	}

	return CompilationInfo{
		Language:    LanguageGo,
		PackageName: version.ModuleName,
		Version:     version.Version,
		Files: []File{
			{
				Path:    "go.zip",
				Content: zipBuf.String(),
			},
		},
	}, nil
}

// compilePython compiles a version into Python code (legacy v1 implementation)
func (s *Server) compilePython(version *Version) (CompilationInfo, error) {
	// Create a temporary directory for compilation
	tmpDir, err := os.MkdirTemp("", "spoke-python-compile-*")
	if err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the output directory for Python files
	pyOutDir := filepath.Join(tmpDir, "python")
	if err := os.MkdirAll(pyOutDir, 0755); err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to create python output dir: %w", err)
	}

	// Create a directory for all proto files
	protoDir := filepath.Join(tmpDir, "proto")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to create proto dir: %w", err)
	}

	// Write all proto files to the temp directory
	for _, file := range version.Files {
		filePath := filepath.Join(protoDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return CompilationInfo{}, fmt.Errorf("failed to create proto file dir: %w", err)
		}
		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return CompilationInfo{}, fmt.Errorf("failed to write proto file: %w", err)
		}
	}

	// Handle dependencies
	for _, dep := range version.Dependencies {
		parts := strings.Split(dep, "@")
		if len(parts) != 2 {
			continue
		}
		depModule := parts[0]
		depVersion := parts[1]

		// Get the dependency version
		depVer, err := s.storage.GetVersion(depModule, depVersion)
		if err != nil {
			return CompilationInfo{}, fmt.Errorf("failed to get dependency %s@%s: %w", depModule, depVersion, err)
		}

		// Write dependency proto files
		for _, file := range depVer.Files {
			filePath := filepath.Join(protoDir, depModule, file.Path)
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return CompilationInfo{}, fmt.Errorf("failed to create dependency proto file dir: %w", err)
			}
			if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
				return CompilationInfo{}, fmt.Errorf("failed to write dependency proto file: %w", err)
			}
		}
	}

	// Create setup.py
	setupPyContent := fmt.Sprintf(`from setuptools import setup, find_packages

setup(
    name="%s",
    version="%s",
    packages=find_packages(),
    install_requires=[
        "protobuf>=4.24.0",
    ],
    python_requires=">=3.7",
)`, version.ModuleName, version.Version)
	if err := os.WriteFile(filepath.Join(pyOutDir, "setup.py"), []byte(setupPyContent), 0644); err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to write setup.py: %w", err)
	}

	// Create pyproject.toml
	pyprojectContent := `[build-system]
requires = ["setuptools>=42", "wheel"]
build-backend = "setuptools.build_meta"`
	if err := os.WriteFile(filepath.Join(pyOutDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	// Run protoc to generate Python code
	protoFiles := make([]string, 0, len(version.Files))
	for _, file := range version.Files {
		protoFiles = append(protoFiles, filepath.Join(protoDir, file.Path))
	}

	args := append([]string{
		"--python_out=" + pyOutDir,
		"-I" + protoDir,
	}, protoFiles...)
	cmd := exec.Command("protoc", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return CompilationInfo{}, fmt.Errorf("protoc failed: %s: %w", output, err)
	}

	// Build the wheel package
	cmd = exec.Command("python", "-m", "build", "--wheel")
	cmd.Dir = pyOutDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to build wheel: %s: %w", output, err)
	}

	// Find the generated wheel file
	wheelFiles, err := filepath.Glob(filepath.Join(pyOutDir, "dist", "*.whl"))
	if err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to find wheel file: %w", err)
	}
	if len(wheelFiles) == 0 {
		return CompilationInfo{}, fmt.Errorf("no wheel file generated")
	}

	// Read the wheel file
	wheelContent, err := os.ReadFile(wheelFiles[0])
	if err != nil {
		return CompilationInfo{}, fmt.Errorf("failed to read wheel file: %w", err)
	}

	return CompilationInfo{
		Language:    LanguagePython,
		PackageName: version.ModuleName,
		Version:     version.Version,
		Files: []File{
			{
				Path:    filepath.Base(wheelFiles[0]),
				Content: string(wheelContent),
			},
		},
	}, nil
}

// listLanguages returns a list of all supported languages
func (s *Server) listLanguages(w http.ResponseWriter, r *http.Request) {
	// Check if orchestrator is available
	if s.orchestrator == nil {
		http.Error(w, "Code generation orchestrator not available", http.StatusServiceUnavailable)
		return
	}

	// Get language registry from orchestrator (we need to add a method for this)
	// For now, return hardcoded list based on our language constants
	languages := []LanguageInfo{
		{
			ID:               string(LanguageGo),
			Name:             "Go",
			DisplayName:      "Go (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".pb.go"},
			Enabled:          true,
			Stable:           true,
			Description:      "Go language support with protoc-gen-go",
			DocumentationURL: "https://protobuf.dev/reference/go/go-generated/",
			PluginVersion:    "v1.31.0",
			PackageManager:   &PackageManagerInfo{Name: "go-modules", ConfigFiles: []string{"go.mod"}},
		},
		{
			ID:               string(LanguagePython),
			Name:             "Python",
			DisplayName:      "Python (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{"_pb2.py", "_pb2_grpc.py"},
			Enabled:          true,
			Stable:           true,
			Description:      "Python language support with protobuf and grpcio",
			DocumentationURL: "https://protobuf.dev/reference/python/python-generated/",
			PluginVersion:    "4.24.0",
			PackageManager:   &PackageManagerInfo{Name: "pip", ConfigFiles: []string{"setup.py", "pyproject.toml"}},
		},
		{
			ID:               string(LanguageJava),
			Name:             "Java",
			DisplayName:      "Java (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".java"},
			Enabled:          true,
			Stable:           true,
			Description:      "Java language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/java/java-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "maven", ConfigFiles: []string{"pom.xml"}},
		},
		{
			ID:               string(LanguageCPP),
			Name:             "C++",
			DisplayName:      "C++ (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".pb.h", ".pb.cc"},
			Enabled:          true,
			Stable:           true,
			Description:      "C++ language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/cpp/cpp-generated/",
			PluginVersion:    "3.21.0",
		},
		{
			ID:               string(LanguageCSharp),
			Name:             "C#",
			DisplayName:      "C# (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".cs"},
			Enabled:          true,
			Stable:           true,
			Description:      "C# language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/csharp/csharp-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "nuget", ConfigFiles: []string{"Package.csproj"}},
		},
		{
			ID:               string(LanguageRust),
			Name:             "Rust",
			DisplayName:      "Rust (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".rs"},
			Enabled:          true,
			Stable:           true,
			Description:      "Rust language support with prost and tonic",
			DocumentationURL: "https://github.com/tokio-rs/prost",
			PluginVersion:    "3.2.0",
			PackageManager:   &PackageManagerInfo{Name: "cargo", ConfigFiles: []string{"Cargo.toml"}},
		},
		{
			ID:               string(LanguageTypeScript),
			Name:             "TypeScript",
			DisplayName:      "TypeScript (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".ts", "_pb.ts"},
			Enabled:          true,
			Stable:           true,
			Description:      "TypeScript language support with ts-proto",
			DocumentationURL: "https://github.com/stephenh/ts-proto",
			PluginVersion:    "5.0.1",
			PackageManager:   &PackageManagerInfo{Name: "npm", ConfigFiles: []string{"package.json", "tsconfig.json"}},
		},
		{
			ID:               string(LanguageJavaScript),
			Name:             "JavaScript",
			DisplayName:      "JavaScript (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{"_pb.js"},
			Enabled:          true,
			Stable:           true,
			Description:      "JavaScript language support with protobufjs",
			DocumentationURL: "https://protobuf.dev/reference/javascript/javascript-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "npm", ConfigFiles: []string{"package.json"}},
		},
		{
			ID:               string(LanguageDart),
			Name:             "Dart",
			DisplayName:      "Dart (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".pb.dart", ".pbgrpc.dart"},
			Enabled:          true,
			Stable:           true,
			Description:      "Dart language support with protobuf and gRPC",
			DocumentationURL: "https://pub.dev/packages/protobuf",
			PluginVersion:    "3.1.0",
			PackageManager:   &PackageManagerInfo{Name: "pub", ConfigFiles: []string{"pubspec.yaml"}},
		},
		{
			ID:               string(LanguageSwift),
			Name:             "Swift",
			DisplayName:      "Swift (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".pb.swift", ".grpc.swift"},
			Enabled:          true,
			Stable:           true,
			Description:      "Swift language support with SwiftProtobuf and gRPC-Swift",
			DocumentationURL: "https://github.com/apple/swift-protobuf",
			PluginVersion:    "1.25.0",
			PackageManager:   &PackageManagerInfo{Name: "swift-package", ConfigFiles: []string{"Package.swift"}},
		},
		{
			ID:               string(LanguageKotlin),
			Name:             "Kotlin",
			DisplayName:      "Kotlin (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".kt"},
			Enabled:          true,
			Stable:           true,
			Description:      "Kotlin language support with protobuf-kotlin and gRPC-Kotlin",
			DocumentationURL: "https://github.com/grpc/grpc-kotlin",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "gradle", ConfigFiles: []string{"build.gradle.kts"}},
		},
		{
			ID:               string(LanguageObjectiveC),
			Name:             "Objective-C",
			DisplayName:      "Objective-C (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".pbobjc.h", ".pbobjc.m"},
			Enabled:          true,
			Stable:           true,
			Description:      "Objective-C language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/objective-c/objective-c-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "cocoapods", ConfigFiles: []string{"Podspec"}},
		},
		{
			ID:               string(LanguageRuby),
			Name:             "Ruby",
			DisplayName:      "Ruby (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{"_pb.rb"},
			Enabled:          true,
			Stable:           true,
			Description:      "Ruby language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/ruby/ruby-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "gem", ConfigFiles: []string{"gemspec"}},
		},
		{
			ID:               string(LanguagePHP),
			Name:             "PHP",
			DisplayName:      "PHP (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".php"},
			Enabled:          true,
			Stable:           true,
			Description:      "PHP language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/php/php-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "composer", ConfigFiles: []string{"composer.json"}},
		},
		{
			ID:               string(LanguageScala),
			Name:             "Scala",
			DisplayName:      "Scala (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".scala"},
			Enabled:          true,
			Stable:           true,
			Description:      "Scala language support with ScalaPB",
			DocumentationURL: "https://scalapb.github.io/",
			PluginVersion:    "0.11.13",
			PackageManager:   &PackageManagerInfo{Name: "sbt", ConfigFiles: []string{"build.sbt"}},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(languages)
}

// getLanguage returns details for a specific language
func (s *Server) getLanguage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	languageID := vars["id"]

	// Check if orchestrator is available
	if s.orchestrator == nil {
		http.Error(w, "Code generation orchestrator not available", http.StatusServiceUnavailable)
		return
	}

	// Get all languages and find the requested one
	// In a real implementation, we would query the language registry directly
	var targetLang *LanguageInfo

	// Call listLanguages to get all languages (reuse logic)
	allLanguages := []LanguageInfo{} // We'd populate this from registry
	// For now, find in hardcoded list
	for _, lang := range allLanguages {
		if lang.ID == languageID {
			targetLang = &lang
			break
		}
	}

	if targetLang == nil {
		http.Error(w, fmt.Sprintf("Language %s not found", languageID), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targetLang)
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
// getExamples generates code examples for a specific language
func (s *Server) getExamples(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]
	language := vars["language"]

	// Get the version from storage
	ver, err := s.storage.GetVersion(moduleName, version)
	if err != nil {
		http.Error(w, fmt.Sprintf("Version not found: %v", err), http.StatusNotFound)
		return
	}

	// TODO: Implement example generation using the examples package
	// For now, return a placeholder
	fileCount := len(ver.Files)
	placeholder := fmt.Sprintf(`// Example code for %s
// Module: %s
// Version: %s
// Language: %s
// Proto files: %d

// Code generation coming soon!
`, moduleName, moduleName, version, language, fileCount)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(placeholder))
}

// DiffRequest represents a request to compare two versions
type DiffRequest struct {
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
}

// compareDiff compares two versions and returns the differences
func (s *Server) compareDiff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]

	// Parse request body
	var req DiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Get both versions
	fromVer, err := s.storage.GetVersion(moduleName, req.FromVersion)
	if err != nil {
		http.Error(w, fmt.Sprintf("From version not found: %v", err), http.StatusNotFound)
		return
	}

	toVer, err := s.storage.GetVersion(moduleName, req.ToVersion)
	if err != nil {
		http.Error(w, fmt.Sprintf("To version not found: %v", err), http.StatusNotFound)
		return
	}

	// TODO: Implement actual diff analysis
	// For now, return a placeholder
	placeholder := map[string]interface{}{
		"from_version": req.FromVersion,
		"to_version":   req.ToVersion,
		"changes": []map[string]interface{}{
			{
				"type":        "placeholder",
				"severity":    "non_breaking",
				"location":    "example.proto",
				"description": "Schema diff analysis coming soon!",
			},
		},
		"_note": fmt.Sprintf("Comparing %d files in v%s with %d files in v%s", 
			len(fromVer.Files), req.FromVersion, len(toVer.Files), req.ToVersion),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(placeholder)
}
