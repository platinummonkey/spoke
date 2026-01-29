// Package api provides legacy v1 compilation implementation
//
// DEPRECATED: This file contains legacy v1 compilation logic that directly calls protoc.
// New code should use the v2 orchestrator (compilation_handlers.go) instead.
// This code is maintained for backward compatibility only and will be removed in a future version.
package api

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/analytics"
	"github.com/platinummonkey/spoke/pkg/async"
)

// downloadCompiled handles GET /modules/{name}/versions/{version}/download/{language}
// DEPRECATED: Use v2 compilation API instead
func (s *Server) downloadCompiled(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
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

	// Calculate file size
	var fileSize int64
	for _, file := range compilationInfo.Files {
		fileSize += int64(len(file.Content))
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
	success := true
	var downloadErr error
	for _, file := range compilationInfo.Files {
		if _, err := w.Write([]byte(file.Content)); err != nil {
			success = false
			downloadErr = err
			http.Error(w, err.Error(), http.StatusInternalServerError)
			break
		}
	}

	// Track download event asynchronously (non-blocking)
	if s.eventTracker != nil {
		async.SafeGo(r.Context(), 5*time.Second, "track download", func(ctx context.Context) error {
			event := analytics.DownloadEvent{
				UserID:         analytics.ExtractUserID(r),
				OrganizationID: analytics.ExtractOrganizationID(r),
				ModuleName:     vars["name"],
				Version:        vars["version"],
				Language:       string(language),
				FileSize:       fileSize,
				Duration:       time.Since(startTime),
				Success:        success,
				IPAddress:      analytics.GetClientIP(r),
				UserAgent:      analytics.GetUserAgent(r),
				ClientSDK:      analytics.GetClientSDK(r),
				ClientVersion:  analytics.GetClientVersion(r),
				CacheHit:       false, // TODO: detect cache hit from response headers
			}
			if downloadErr != nil {
				event.ErrorMessage = downloadErr.Error()
			}

			return s.eventTracker.TrackDownload(ctx, event)
		})
	}
}

// compileForLanguage routes compilation to v1 or v2 based on feature flag
// DEPRECATED: Use compileVersion with simplified code generator instead
func (s *Server) compileForLanguage(version *Version, language Language) (CompilationInfo, error) {
	// Try new generator first, fallback to v1 if it fails
	result, err := s.compileWithGenerator(version, language)
	if err == nil {
		return result, nil
	}

	// Fallback to v1 legacy compilation
	return s.compileV1(version, language)
}

// compileV1 routes to legacy compilation methods
// DEPRECATED: Use v2 orchestrator instead
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
// DEPRECATED: Use v2 orchestrator instead
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
// DEPRECATED: Use v2 orchestrator instead
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
