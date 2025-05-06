package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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