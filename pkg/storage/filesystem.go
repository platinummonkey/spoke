package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api"
)

// FileSystemStorage implements the Storage interface using the local filesystem
type FileSystemStorage struct {
	rootDir string
}

// NewFileSystemStorage creates a new filesystem-based storage
func NewFileSystemStorage(rootDir string) (*FileSystemStorage, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}
	return &FileSystemStorage{rootDir: rootDir}, nil
}

// CreateModule implements Storage.CreateModule
func (s *FileSystemStorage) CreateModule(module *api.Module) error {
	moduleDir := filepath.Join(s.rootDir, module.Name)
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return fmt.Errorf("failed to create module directory: %w", err)
	}

	moduleFile := filepath.Join(moduleDir, "module.json")
	data, err := json.Marshal(module)
	if err != nil {
		return fmt.Errorf("failed to marshal module: %w", err)
	}

	if err := os.WriteFile(moduleFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write module file: %w", err)
	}

	return nil
}

// GetModule implements Storage.GetModule
func (s *FileSystemStorage) GetModule(name string) (*api.Module, error) {
	moduleFile := filepath.Join(s.rootDir, name, "module.json")
	data, err := os.ReadFile(moduleFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read module file: %w", err)
	}

	var module api.Module
	if err := json.Unmarshal(data, &module); err != nil {
		return nil, fmt.Errorf("failed to unmarshal module: %w", err)
	}

	return &module, nil
}

// ListModules implements Storage.ListModules
func (s *FileSystemStorage) ListModules() ([]*api.Module, error) {
	entries, err := os.ReadDir(s.rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read root directory: %w", err)
	}

	var modules []*api.Module
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		module, err := s.GetModule(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to get module %s: %w", entry.Name(), err)
		}
		modules = append(modules, module)
	}

	return modules, nil
}

// CreateVersion implements Storage.CreateVersion
func (s *FileSystemStorage) CreateVersion(version *api.Version) error {
	versionDir := filepath.Join(s.rootDir, version.ModuleName, "versions", version.Version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	// Initialize SourceInfo with default values if not set
	if version.SourceInfo.Repository == "" {
		version.SourceInfo.Repository = "unknown"
	}
	if version.SourceInfo.CommitSHA == "" {
		version.SourceInfo.CommitSHA = "unknown"
	}
	if version.SourceInfo.Branch == "" {
		version.SourceInfo.Branch = "unknown"
	}

	// Write version metadata
	versionFile := filepath.Join(versionDir, "version.json")
	data, err := json.Marshal(version)
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	if err := os.WriteFile(versionFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	// Write proto files
	for _, file := range version.Files {
		filePath := filepath.Join(versionDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create file directory: %w", err)
		}

		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return fmt.Errorf("failed to write proto file: %w", err)
		}
	}

	return nil
}

// GetVersion implements Storage.GetVersion
func (s *FileSystemStorage) GetVersion(moduleName, version string) (*api.Version, error) {
	// Special handling for "latest" version
	if version == "latest" {
		return s.getLatestVersion(moduleName)
	}

	versionFile := filepath.Join(s.rootDir, moduleName, "versions", version, "version.json")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read version file: %w", err)
	}

	var ver api.Version
	if err := json.Unmarshal(data, &ver); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version: %w", err)
	}

	return &ver, nil
}

// getLatestVersion finds the latest semantic version for a module
func (s *FileSystemStorage) getLatestVersion(moduleName string) (*api.Version, error) {
	versions, err := s.ListVersions(moduleName)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions for module %s: %w", moduleName, err)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for module %s", moduleName)
	}

	// Sort versions in reverse order (newest first)
	// First try to sort by semantic version
	sort.Slice(versions, func(i, j int) bool {
		vi := versions[i].Version
		vj := versions[j].Version

		// Skip the "v" prefix if present
		if len(vi) > 0 && vi[0] == 'v' {
			vi = vi[1:]
		}
		if len(vj) > 0 && vj[0] == 'v' {
			vj = vj[1:]
		}

		// Split versions by dots
		partsI := strings.Split(vi, ".")
		partsJ := strings.Split(vj, ".")

		// Compare major, minor, patch versions
		for k := 0; k < len(partsI) && k < len(partsJ); k++ {
			numI, errI := strconv.Atoi(partsI[k])
			numJ, errJ := strconv.Atoi(partsJ[k])
			
			// If we can't parse as integers, fall back to string comparison
			if errI != nil || errJ != nil {
				// If string comparison determines they're different
				if partsI[k] != partsJ[k] {
					return partsI[k] > partsJ[k] // lexicographic comparison
				}
				continue // They're the same, continue to next part
			}
			
			// If numeric comparison determines they're different
			if numI != numJ {
				return numI > numJ // numeric comparison
			}
		}
		
		// If one version has more parts than the other
		if len(partsI) != len(partsJ) {
			return len(partsI) > len(partsJ)
		}
		
		// If semantic versions appear identical, fall back to creation time
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})

	// Return the first (newest) version
	return versions[0], nil
}

// ListVersions implements Storage.ListVersions
func (s *FileSystemStorage) ListVersions(moduleName string) ([]*api.Version, error) {
	versionsDir := filepath.Join(s.rootDir, moduleName, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read versions directory: %w", err)
	}

	var versions []*api.Version
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version, err := s.GetVersion(moduleName, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to get version %s: %w", entry.Name(), err)
		}
		versions = append(versions, version)
	}

	return versions, nil
}

// GetFile implements Storage.GetFile
func (s *FileSystemStorage) GetFile(moduleName, version, path string) (*api.File, error) {
	filePath := filepath.Join(s.rootDir, moduleName, "versions", version, path)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return &api.File{
		Path:    path,
		Content: string(content),
	}, nil
}

// UpdateVersion implements Storage.UpdateVersion
func (s *FileSystemStorage) UpdateVersion(version *api.Version) error {
	versionDir := filepath.Join(s.rootDir, version.ModuleName, "versions", version.Version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	// Write version metadata
	versionFile := filepath.Join(versionDir, "version.json")
	data, err := json.Marshal(version)
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	if err := os.WriteFile(versionFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}

// Context-aware methods that implement the storage.Storage interface
// These delegate to the existing non-context methods for backward compatibility

// CreateModuleContext implements storage.ModuleWriter
func (s *FileSystemStorage) CreateModuleContext(ctx context.Context, module *api.Module) error {
	return s.CreateModule(module)
}

// GetModuleContext implements storage.ModuleReader
func (s *FileSystemStorage) GetModuleContext(ctx context.Context, name string) (*api.Module, error) {
	return s.GetModule(name)
}

// ListModulesContext implements storage.ModuleReader
func (s *FileSystemStorage) ListModulesContext(ctx context.Context) ([]*api.Module, error) {
	return s.ListModules()
}

// ListModulesPaginated implements storage.ModuleReader
func (s *FileSystemStorage) ListModulesPaginated(ctx context.Context, limit, offset int) ([]*api.Module, int64, error) {
	modules, err := s.ListModules()
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(modules))

	// Apply pagination
	start := offset
	if start > len(modules) {
		start = len(modules)
	}

	end := start + limit
	if end > len(modules) {
		end = len(modules)
	}

	return modules[start:end], total, nil
}

// CreateVersionContext implements storage.VersionWriter
func (s *FileSystemStorage) CreateVersionContext(ctx context.Context, version *api.Version) error {
	return s.CreateVersion(version)
}

// GetVersionContext implements storage.VersionReader
func (s *FileSystemStorage) GetVersionContext(ctx context.Context, moduleName, version string) (*api.Version, error) {
	return s.GetVersion(moduleName, version)
}

// ListVersionsContext implements storage.VersionReader
func (s *FileSystemStorage) ListVersionsContext(ctx context.Context, moduleName string) ([]*api.Version, error) {
	return s.ListVersions(moduleName)
}

// ListVersionsPaginated implements storage.VersionReader
func (s *FileSystemStorage) ListVersionsPaginated(ctx context.Context, moduleName string, limit, offset int) ([]*api.Version, int64, error) {
	versions, err := s.ListVersions(moduleName)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(versions))

	// Apply pagination
	start := offset
	if start > len(versions) {
		start = len(versions)
	}

	end := start + limit
	if end > len(versions) {
		end = len(versions)
	}

	return versions[start:end], total, nil
}

// GetFileContext implements storage.VersionReader
func (s *FileSystemStorage) GetFileContext(ctx context.Context, moduleName, version, path string) (*api.File, error) {
	return s.GetFile(moduleName, version, path)
}

// UpdateVersionContext implements storage.VersionWriter
func (s *FileSystemStorage) UpdateVersionContext(ctx context.Context, version *api.Version) error {
	return s.UpdateVersion(version)
}

// GetFileContent implements storage.FileStorage
// Filesystem storage doesn't support content-addressed storage
func (s *FileSystemStorage) GetFileContent(ctx context.Context, hash string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("GetFileContent not implemented: filesystem storage doesn't support content-addressed storage")
}

// PutFileContent implements storage.FileStorage
// Filesystem storage doesn't support content-addressed storage
func (s *FileSystemStorage) PutFileContent(ctx context.Context, content io.Reader, contentType string) (hash string, err error) {
	return "", fmt.Errorf("PutFileContent not implemented: filesystem storage doesn't support content-addressed storage")
}

// GetCompiledArtifact implements storage.ArtifactStorage
func (s *FileSystemStorage) GetCompiledArtifact(ctx context.Context, moduleName, version, language string) (io.ReadCloser, error) {
	artifactPath := filepath.Join(s.rootDir, moduleName, "versions", version, "compiled", language+".tar.gz")
	file, err := os.Open(artifactPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, api.ErrNotFound
		}
		return nil, fmt.Errorf("failed to open compiled artifact: %w", err)
	}
	return file, nil
}

// PutCompiledArtifact implements storage.ArtifactStorage
func (s *FileSystemStorage) PutCompiledArtifact(ctx context.Context, moduleName, version, language string, artifact io.Reader) error {
	compiledDir := filepath.Join(s.rootDir, moduleName, "versions", version, "compiled")
	if err := os.MkdirAll(compiledDir, 0755); err != nil {
		return fmt.Errorf("failed to create compiled directory: %w", err)
	}

	artifactPath := filepath.Join(compiledDir, language+".tar.gz")
	file, err := os.Create(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to create artifact file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, artifact); err != nil {
		return fmt.Errorf("failed to write artifact: %w", err)
	}

	return nil
}

// InvalidateCache implements storage.CacheManager
// Filesystem storage has no cache to invalidate
func (s *FileSystemStorage) InvalidateCache(ctx context.Context, patterns ...string) error {
	return nil // No-op for filesystem storage
}

// HealthCheck implements storage.HealthChecker
func (s *FileSystemStorage) HealthCheck(ctx context.Context) error {
	_, err := os.Stat(s.rootDir)
	if err != nil {
		return fmt.Errorf("filesystem storage health check failed: %w", err)
	}
	return nil
}

// Verify that FileSystemStorage implements storage.Storage at compile time
var _ Storage = (*FileSystemStorage)(nil) 