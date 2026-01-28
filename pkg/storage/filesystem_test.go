package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/api"
)

func TestNewFileSystemStorage(t *testing.T) {
	t.Run("creates storage with new directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		rootDir := filepath.Join(tmpDir, "test-storage")

		storage, err := NewFileSystemStorage(rootDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		if storage == nil {
			t.Fatal("Storage should not be nil")
		}

		if storage.rootDir != rootDir {
			t.Errorf("Expected rootDir %s, got %s", rootDir, storage.rootDir)
		}

		// Verify directory was created
		if _, err := os.Stat(rootDir); os.IsNotExist(err) {
			t.Error("Root directory should have been created")
		}
	})

	t.Run("creates storage with existing directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		if storage == nil {
			t.Fatal("Storage should not be nil")
		}
	})
}

func TestFileSystemStorage_CreateModule(t *testing.T) {
	t.Run("creates module successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		module := &api.Module{
			Name:        "test.module",
			Description: "Test module",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		// Verify module directory exists
		moduleDir := filepath.Join(tmpDir, "test.module")
		if _, err := os.Stat(moduleDir); os.IsNotExist(err) {
			t.Error("Module directory should have been created")
		}

		// Verify module.json file exists
		moduleFile := filepath.Join(moduleDir, "module.json")
		if _, err := os.Stat(moduleFile); os.IsNotExist(err) {
			t.Error("Module file should have been created")
		}
	})
}

func TestFileSystemStorage_GetModule(t *testing.T) {
	t.Run("gets existing module", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create a module
		originalModule := &api.Module{
			Name:        "test.module",
			Description: "Test module",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err = storage.CreateModule(originalModule)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		// Get the module
		module, err := storage.GetModule("test.module")
		if err != nil {
			t.Fatalf("Failed to get module: %v", err)
		}

		if module.Name != originalModule.Name {
			t.Errorf("Expected name %s, got %s", originalModule.Name, module.Name)
		}

		if module.Description != originalModule.Description {
			t.Errorf("Expected description %s, got %s", originalModule.Description, module.Description)
		}
	})

	t.Run("returns error for non-existent module", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		_, err = storage.GetModule("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent module")
		}
	})
}

func TestFileSystemStorage_ListModules(t *testing.T) {
	t.Run("lists all modules", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create multiple modules
		modules := []*api.Module{
			{Name: "module1", Description: "First module", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{Name: "module2", Description: "Second module", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		for _, module := range modules {
			err = storage.CreateModule(module)
			if err != nil {
				t.Fatalf("Failed to create module: %v", err)
			}
		}

		// List modules
		listedModules, err := storage.ListModules()
		if err != nil {
			t.Fatalf("Failed to list modules: %v", err)
		}

		if len(listedModules) != 2 {
			t.Errorf("Expected 2 modules, got %d", len(listedModules))
		}
	})

	t.Run("returns empty list for no modules", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		modules, err := storage.ListModules()
		if err != nil {
			t.Fatalf("Failed to list modules: %v", err)
		}

		if len(modules) != 0 {
			t.Errorf("Expected 0 modules, got %d", len(modules))
		}
	})
}

func TestFileSystemStorage_CreateVersion(t *testing.T) {
	t.Run("creates version successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module first
		module := &api.Module{
			Name:        "test.module",
			Description: "Test module",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		// Create version
		version := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files: []api.File{
				{Path: "user.proto", Content: "syntax = \"proto3\";"},
			},
			CreatedAt: time.Now(),
			SourceInfo: api.SourceInfo{
				Repository: "github.com/test/repo",
				CommitSHA:  "abc123",
				Branch:     "main",
			},
		}

		err = storage.CreateVersion(version)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}

		// Verify version directory exists
		versionDir := filepath.Join(tmpDir, "test.module", "versions", "v1.0.0")
		if _, err := os.Stat(versionDir); os.IsNotExist(err) {
			t.Error("Version directory should have been created")
		}
	})

	t.Run("initializes default source info", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create version with empty source info
		version := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files: []api.File{
				{Path: "test.proto", Content: "syntax = \"proto3\";"},
			},
			CreatedAt:  time.Now(),
			SourceInfo: api.SourceInfo{}, // Empty source info
		}

		err = storage.CreateVersion(version)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}

		// Verify defaults were set
		if version.SourceInfo.Repository != "unknown" {
			t.Errorf("Expected default repository 'unknown', got %s", version.SourceInfo.Repository)
		}
		if version.SourceInfo.CommitSHA != "unknown" {
			t.Errorf("Expected default commit SHA 'unknown', got %s", version.SourceInfo.CommitSHA)
		}
		if version.SourceInfo.Branch != "unknown" {
			t.Errorf("Expected default branch 'unknown', got %s", version.SourceInfo.Branch)
		}
	})
}

func TestFileSystemStorage_GetVersion(t *testing.T) {
	t.Run("gets existing version", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module and version
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		originalVersion := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files:      []api.File{{Path: "test.proto", Content: "syntax = \"proto3\";"}},
			CreatedAt:  time.Now(),
			SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
		}
		err = storage.CreateVersion(originalVersion)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}

		// Get version
		version, err := storage.GetVersion("test.module", "v1.0.0")
		if err != nil {
			t.Fatalf("Failed to get version: %v", err)
		}

		if version.Version != originalVersion.Version {
			t.Errorf("Expected version %s, got %s", originalVersion.Version, version.Version)
		}
	})

	t.Run("gets latest version", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		// Create multiple versions
		versions := []string{"v1.0.0", "v1.1.0", "v2.0.0"}
		for _, v := range versions {
			version := &api.Version{
				ModuleName: "test.module",
				Version:    v,
				Files:      []api.File{{Path: "test.proto", Content: "syntax = \"proto3\";"}},
				CreatedAt:  time.Now(),
				SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
			}
			err = storage.CreateVersion(version)
			if err != nil {
				t.Fatalf("Failed to create version: %v", err)
			}
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		// Get latest version
		latest, err := storage.GetVersion("test.module", "latest")
		if err != nil {
			t.Fatalf("Failed to get latest version: %v", err)
		}

		if latest.Version != "v2.0.0" {
			t.Errorf("Expected latest version v2.0.0, got %s", latest.Version)
		}
	})
}

func TestFileSystemStorage_ListVersions(t *testing.T) {
	t.Run("lists all versions", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		// Create versions
		versionStrings := []string{"v1.0.0", "v1.1.0"}
		for _, v := range versionStrings {
			version := &api.Version{
				ModuleName: "test.module",
				Version:    v,
				Files:      []api.File{{Path: "test.proto", Content: "syntax = \"proto3\";"}},
				CreatedAt:  time.Now(),
				SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
			}
			err = storage.CreateVersion(version)
			if err != nil {
				t.Fatalf("Failed to create version: %v", err)
			}
		}

		// List versions
		versions, err := storage.ListVersions("test.module")
		if err != nil {
			t.Fatalf("Failed to list versions: %v", err)
		}

		if len(versions) != 2 {
			t.Errorf("Expected 2 versions, got %d", len(versions))
		}
	})
}

func TestFileSystemStorage_GetFile(t *testing.T) {
	t.Run("gets file from version", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module and version
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		fileContent := "syntax = \"proto3\";\npackage test;"
		version := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files:      []api.File{{Path: "test.proto", Content: fileContent}},
			CreatedAt:  time.Now(),
			SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
		}
		err = storage.CreateVersion(version)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}

		// Get file
		file, err := storage.GetFile("test.module", "v1.0.0", "test.proto")
		if err != nil {
			t.Fatalf("Failed to get file: %v", err)
		}

		if file.Path != "test.proto" {
			t.Errorf("Expected path test.proto, got %s", file.Path)
		}

		if file.Content != fileContent {
			t.Errorf("Expected content %s, got %s", fileContent, file.Content)
		}
	})
}

func TestFileSystemStorage_UpdateVersion(t *testing.T) {
	t.Run("updates version successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module and version
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		version := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files:      []api.File{{Path: "test.proto", Content: "original"}},
			CreatedAt:  time.Now(),
			SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
		}
		err = storage.CreateVersion(version)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}

		// Update version
		version.Files = []api.File{{Path: "test.proto", Content: "updated"}}
		err = storage.UpdateVersion(version)
		if err != nil {
			t.Fatalf("Failed to update version: %v", err)
		}

		// Verify update
		updated, err := storage.GetVersion("test.module", "v1.0.0")
		if err != nil {
			t.Fatalf("Failed to get updated version: %v", err)
		}

		if len(updated.Files) != 1 || updated.Files[0].Content != "updated" {
			t.Error("Version was not updated correctly")
		}
	})
}

func TestFileSystemStorage_ContextMethods(t *testing.T) {
	t.Run("CreateModuleContext delegates correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		ctx := context.Background()
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}

		err = storage.CreateModuleContext(ctx, module)
		if err != nil {
			t.Fatalf("CreateModuleContext failed: %v", err)
		}

		// Verify it was created
		retrieved, err := storage.GetModuleContext(ctx, "test.module")
		if err != nil {
			t.Fatalf("GetModuleContext failed: %v", err)
		}

		if retrieved.Name != module.Name {
			t.Errorf("Expected name %s, got %s", module.Name, retrieved.Name)
		}
	})

	t.Run("ListModulesContext delegates correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		ctx := context.Background()
		modules, err := storage.ListModulesContext(ctx)
		if err != nil {
			t.Fatalf("ListModulesContext failed: %v", err)
		}

		if len(modules) != 0 {
			t.Errorf("Expected 0 modules, got %d", len(modules))
		}
	})

	t.Run("CreateVersionContext delegates correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		ctx := context.Background()
		version := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files:      []api.File{{Path: "test.proto", Content: "test"}},
			CreatedAt:  time.Now(),
			SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
		}

		err = storage.CreateVersionContext(ctx, version)
		if err != nil {
			t.Fatalf("CreateVersionContext failed: %v", err)
		}
	})

	t.Run("GetVersionContext delegates correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module and version first
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		version := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files:      []api.File{{Path: "test.proto", Content: "test"}},
			CreatedAt:  time.Now(),
			SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
		}
		err = storage.CreateVersion(version)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}

		ctx := context.Background()
		retrieved, err := storage.GetVersionContext(ctx, "test.module", "v1.0.0")
		if err != nil {
			t.Fatalf("GetVersionContext failed: %v", err)
		}

		if retrieved.Version != "v1.0.0" {
			t.Errorf("Expected version v1.0.0, got %s", retrieved.Version)
		}
	})

	t.Run("ListVersionsContext delegates correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module and version
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		version := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files:      []api.File{{Path: "test.proto", Content: "test"}},
			CreatedAt:  time.Now(),
			SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
		}
		err = storage.CreateVersion(version)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}

		ctx := context.Background()
		versions, err := storage.ListVersionsContext(ctx, "test.module")
		if err != nil {
			t.Fatalf("ListVersionsContext failed: %v", err)
		}

		if len(versions) != 1 {
			t.Errorf("Expected 1 version, got %d", len(versions))
		}
	})

	t.Run("GetFileContext delegates correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module and version
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		fileContent := "syntax = \"proto3\";"
		version := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files:      []api.File{{Path: "test.proto", Content: fileContent}},
			CreatedAt:  time.Now(),
			SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
		}
		err = storage.CreateVersion(version)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}

		ctx := context.Background()
		file, err := storage.GetFileContext(ctx, "test.module", "v1.0.0", "test.proto")
		if err != nil {
			t.Fatalf("GetFileContext failed: %v", err)
		}

		if file.Content != fileContent {
			t.Errorf("Expected content %s, got %s", fileContent, file.Content)
		}
	})

	t.Run("UpdateVersionContext delegates correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module and version
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		version := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files:      []api.File{{Path: "test.proto", Content: "original"}},
			CreatedAt:  time.Now(),
			SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
		}
		err = storage.CreateVersion(version)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}

		// Update version
		version.Files = []api.File{{Path: "test.proto", Content: "updated"}}
		ctx := context.Background()
		err = storage.UpdateVersionContext(ctx, version)
		if err != nil {
			t.Fatalf("UpdateVersionContext failed: %v", err)
		}

		// Verify update
		updated, err := storage.GetVersion("test.module", "v1.0.0")
		if err != nil {
			t.Fatalf("Failed to get updated version: %v", err)
		}

		if len(updated.Files) != 1 || updated.Files[0].Content != "updated" {
			t.Error("Version was not updated correctly")
		}
	})
}

func TestFileSystemStorage_ListModulesPaginated(t *testing.T) {
	t.Run("paginates modules correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create 5 modules
		for i := 1; i <= 5; i++ {
			module := &api.Module{
				Name:      "module" + string(rune('0'+i)),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err = storage.CreateModule(module)
			if err != nil {
				t.Fatalf("Failed to create module: %v", err)
			}
		}

		ctx := context.Background()

		// Get first 2 modules
		modules, total, err := storage.ListModulesPaginated(ctx, 2, 0)
		if err != nil {
			t.Fatalf("ListModulesPaginated failed: %v", err)
		}

		if total != 5 {
			t.Errorf("Expected total 5, got %d", total)
		}

		if len(modules) != 2 {
			t.Errorf("Expected 2 modules, got %d", len(modules))
		}

		// Get next 2 modules
		modules, total, err = storage.ListModulesPaginated(ctx, 2, 2)
		if err != nil {
			t.Fatalf("ListModulesPaginated failed: %v", err)
		}

		if len(modules) != 2 {
			t.Errorf("Expected 2 modules, got %d", len(modules))
		}

		// Get last module
		modules, total, err = storage.ListModulesPaginated(ctx, 2, 4)
		if err != nil {
			t.Fatalf("ListModulesPaginated failed: %v", err)
		}

		if len(modules) != 1 {
			t.Errorf("Expected 1 module, got %d", len(modules))
		}
	})
}

func TestFileSystemStorage_ListVersionsPaginated(t *testing.T) {
	t.Run("paginates versions correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		// Create 3 versions
		for i := 1; i <= 3; i++ {
			version := &api.Version{
				ModuleName: "test.module",
				Version:    "v1." + string(rune('0'+i)) + ".0",
				Files:      []api.File{{Path: "test.proto", Content: "test"}},
				CreatedAt:  time.Now(),
				SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
			}
			err = storage.CreateVersion(version)
			if err != nil {
				t.Fatalf("Failed to create version: %v", err)
			}
		}

		ctx := context.Background()

		// Get first 2 versions
		versions, total, err := storage.ListVersionsPaginated(ctx, "test.module", 2, 0)
		if err != nil {
			t.Fatalf("ListVersionsPaginated failed: %v", err)
		}

		if total != 3 {
			t.Errorf("Expected total 3, got %d", total)
		}

		if len(versions) != 2 {
			t.Errorf("Expected 2 versions, got %d", len(versions))
		}
	})
}

func TestFileSystemStorage_GetFileContent(t *testing.T) {
	t.Run("returns not implemented error", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		ctx := context.Background()
		_, err = storage.GetFileContent(ctx, "somehash")
		if err == nil {
			t.Error("Expected error for GetFileContent")
		}

		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("Expected 'not implemented' error, got: %v", err)
		}
	})
}

func TestFileSystemStorage_PutFileContent(t *testing.T) {
	t.Run("returns not implemented error", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		ctx := context.Background()
		_, err = storage.PutFileContent(ctx, strings.NewReader("test"), "text/plain")
		if err == nil {
			t.Error("Expected error for PutFileContent")
		}

		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("Expected 'not implemented' error, got: %v", err)
		}
	})
}

func TestFileSystemStorage_GetCompiledArtifact(t *testing.T) {
	t.Run("gets existing artifact", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module and version
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		version := &api.Version{
			ModuleName: "test.module",
			Version:    "v1.0.0",
			Files:      []api.File{{Path: "test.proto", Content: "test"}},
			CreatedAt:  time.Now(),
			SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
		}
		err = storage.CreateVersion(version)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}

		// Put artifact
		ctx := context.Background()
		artifactContent := "compiled artifact data"
		err = storage.PutCompiledArtifact(ctx, "test.module", "v1.0.0", "go", strings.NewReader(artifactContent))
		if err != nil {
			t.Fatalf("Failed to put artifact: %v", err)
		}

		// Get artifact
		reader, err := storage.GetCompiledArtifact(ctx, "test.module", "v1.0.0", "go")
		if err != nil {
			t.Fatalf("Failed to get artifact: %v", err)
		}
		defer reader.Close()

		content, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("Failed to read artifact: %v", err)
		}

		if string(content) != artifactContent {
			t.Errorf("Expected content %s, got %s", artifactContent, string(content))
		}
	})

	t.Run("returns ErrNotFound for non-existent artifact", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		ctx := context.Background()
		_, err = storage.GetCompiledArtifact(ctx, "test.module", "v1.0.0", "go")
		if err == nil {
			t.Error("Expected error for non-existent artifact")
		}

		if err != api.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got: %v", err)
		}
	})
}

func TestFileSystemStorage_InvalidateCache(t *testing.T) {
	t.Run("no-op returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		ctx := context.Background()
		err = storage.InvalidateCache(ctx, "pattern1", "pattern2")
		if err != nil {
			t.Errorf("InvalidateCache should return nil, got: %v", err)
		}
	})
}

func TestFileSystemStorage_HealthCheck(t *testing.T) {
	t.Run("returns nil for healthy storage", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		ctx := context.Background()
		err = storage.HealthCheck(ctx)
		if err != nil {
			t.Errorf("HealthCheck should return nil, got: %v", err)
		}
	})

	t.Run("returns error for missing root directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Remove the root directory
		err = os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatalf("Failed to remove directory: %v", err)
		}

		ctx := context.Background()
		err = storage.HealthCheck(ctx)
		if err == nil {
			t.Error("HealthCheck should return error for missing directory")
		}
	})
}

func TestFileSystemStorage_VersionSorting(t *testing.T) {
	t.Run("sorts semantic versions correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		// Create versions in non-sequential order
		versions := []string{"v1.0.0", "v2.0.0", "v1.10.0", "v1.2.0"}
		for _, v := range versions {
			version := &api.Version{
				ModuleName: "test.module",
				Version:    v,
				Files:      []api.File{{Path: "test.proto", Content: "test"}},
				CreatedAt:  time.Now(),
				SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
			}
			err = storage.CreateVersion(version)
			if err != nil {
				t.Fatalf("Failed to create version: %v", err)
			}
			time.Sleep(10 * time.Millisecond)
		}

		// Get latest version
		latest, err := storage.GetVersion("test.module", "latest")
		if err != nil {
			t.Fatalf("Failed to get latest version: %v", err)
		}

		// v2.0.0 should be the latest
		if latest.Version != "v2.0.0" {
			t.Errorf("Expected latest version v2.0.0, got %s", latest.Version)
		}
	})

	t.Run("handles versions without v prefix", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileSystemStorage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Create module
		module := &api.Module{Name: "test.module", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		err = storage.CreateModule(module)
		if err != nil {
			t.Fatalf("Failed to create module: %v", err)
		}

		// Create versions without v prefix
		versions := []string{"1.0.0", "2.0.0", "1.5.0"}
		for _, v := range versions {
			version := &api.Version{
				ModuleName: "test.module",
				Version:    v,
				Files:      []api.File{{Path: "test.proto", Content: "test"}},
				CreatedAt:  time.Now(),
				SourceInfo: api.SourceInfo{Repository: "test", CommitSHA: "abc", Branch: "main"},
			}
			err = storage.CreateVersion(version)
			if err != nil {
				t.Fatalf("Failed to create version: %v", err)
			}
			time.Sleep(10 * time.Millisecond)
		}

		// Get latest version
		latest, err := storage.GetVersion("test.module", "latest")
		if err != nil {
			t.Fatalf("Failed to get latest version: %v", err)
		}

		if latest.Version != "2.0.0" {
			t.Errorf("Expected latest version 2.0.0, got %s", latest.Version)
		}
	})
}
