package storage

import (
	"encoding/json"
	"fmt"
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