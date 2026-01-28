package api

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSearchStorageAdapter(t *testing.T) {
	storage := newMockStorage()

	adapter := NewSearchStorageAdapter(storage)

	require.NotNil(t, adapter)
	assert.NotNil(t, adapter.storage)
}

func TestSearchStorageAdapter_GetVersion(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		// Add test version
		version := &Version{
			ModuleName: "test-module",
			Version:    "v1.0.0",
			Files: []File{
				{Path: "test.proto", Content: "syntax = \"proto3\";"},
				{Path: "other.proto", Content: "syntax = \"proto3\";\npackage test;"},
			},
			Dependencies: []string{"dep1", "dep2"},
		}
		err := storage.CreateVersion(version)
		require.NoError(t, err)

		// Get version through adapter
		searchVer, err := adapter.GetVersion("test-module", "v1.0.0")

		require.NoError(t, err)
		assert.Equal(t, "v1.0.0", searchVer.Version)
		assert.Equal(t, "test-module", searchVer.ModuleName)
		assert.Len(t, searchVer.Files, 2)
		assert.Equal(t, "test.proto", searchVer.Files[0].Path)
		assert.Equal(t, "syntax = \"proto3\";", searchVer.Files[0].Content)
		assert.Equal(t, "other.proto", searchVer.Files[1].Path)
		assert.Len(t, searchVer.Dependencies, 2)
		assert.Contains(t, searchVer.Dependencies, "dep1")
		assert.Contains(t, searchVer.Dependencies, "dep2")
	})

	t.Run("not found error", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		// Try to get non-existent version
		searchVer, err := adapter.GetVersion("nonexistent", "v1.0.0")

		assert.Error(t, err)
		assert.Nil(t, searchVer)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("storage error", func(t *testing.T) {
		storage := newMockStorage()
		storage.getVersionError = errors.New("database error")
		adapter := NewSearchStorageAdapter(storage)

		searchVer, err := adapter.GetVersion("test-module", "v1.0.0")

		assert.Error(t, err)
		assert.Nil(t, searchVer)
		assert.Contains(t, err.Error(), "database error")
	})

	t.Run("empty files", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		// Version with no files
		version := &Version{
			ModuleName:   "empty-module",
			Version:      "v1.0.0",
			Files:        []File{},
			Dependencies: []string{},
		}
		err := storage.CreateVersion(version)
		require.NoError(t, err)

		searchVer, err := adapter.GetVersion("empty-module", "v1.0.0")

		require.NoError(t, err)
		assert.Empty(t, searchVer.Files)
		assert.Empty(t, searchVer.Dependencies)
	})
}

func TestSearchStorageAdapter_GetFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		// Add test version and file
		version := &Version{
			ModuleName: "test-module",
			Version:    "v1.0.0",
			Files: []File{
				{Path: "test.proto", Content: "syntax = \"proto3\";\npackage test;"},
			},
		}
		err := storage.CreateVersion(version)
		require.NoError(t, err)

		// Need to set up file in mock storage
		if storage.files["test-module"] == nil {
			storage.files["test-module"] = make(map[string]map[string]*File)
		}
		if storage.files["test-module"]["v1.0.0"] == nil {
			storage.files["test-module"]["v1.0.0"] = make(map[string]*File)
		}
		storage.files["test-module"]["v1.0.0"]["test.proto"] = &File{
			Path:    "test.proto",
			Content: "syntax = \"proto3\";\npackage test;",
		}

		// Get file through adapter
		searchFile, err := adapter.GetFile("test-module", "v1.0.0", "test.proto")

		require.NoError(t, err)
		assert.Equal(t, "test.proto", searchFile.Path)
		assert.Equal(t, []byte("syntax = \"proto3\";\npackage test;"), searchFile.Content)
	})

	t.Run("not found error", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		searchFile, err := adapter.GetFile("nonexistent", "v1.0.0", "test.proto")

		assert.Error(t, err)
		assert.Nil(t, searchFile)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("storage error", func(t *testing.T) {
		storage := newMockStorage()
		storage.getFileError = errors.New("file read error")
		adapter := NewSearchStorageAdapter(storage)

		searchFile, err := adapter.GetFile("test-module", "v1.0.0", "test.proto")

		assert.Error(t, err)
		assert.Nil(t, searchFile)
		assert.Contains(t, err.Error(), "file read error")
	})

	t.Run("empty content", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		// Set up empty file
		if storage.files["test-module"] == nil {
			storage.files["test-module"] = make(map[string]map[string]*File)
		}
		if storage.files["test-module"]["v1.0.0"] == nil {
			storage.files["test-module"]["v1.0.0"] = make(map[string]*File)
		}
		storage.files["test-module"]["v1.0.0"]["empty.proto"] = &File{
			Path:    "empty.proto",
			Content: "",
		}

		searchFile, err := adapter.GetFile("test-module", "v1.0.0", "empty.proto")

		require.NoError(t, err)
		assert.Equal(t, "empty.proto", searchFile.Path)
		assert.Equal(t, []byte(""), searchFile.Content)
	})
}

func TestSearchStorageAdapter_ListModules(t *testing.T) {
	t.Run("success with multiple modules", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		// Add test modules
		module1 := &Module{
			Name:        "module1",
			Description: "First module",
			CreatedAt:   time.Now(),
		}
		module2 := &Module{
			Name:        "module2",
			Description: "Second module",
			CreatedAt:   time.Now(),
		}
		err := storage.CreateModule(module1)
		require.NoError(t, err)
		err = storage.CreateModule(module2)
		require.NoError(t, err)

		// List modules through adapter
		searchModules, err := adapter.ListModules()

		require.NoError(t, err)
		assert.Len(t, searchModules, 2)

		// Check that both modules are present (order may vary)
		moduleNames := make(map[string]bool)
		for _, mod := range searchModules {
			moduleNames[mod.Name] = true
			// Verify fields are converted
			assert.NotEmpty(t, mod.Name)
		}
		assert.True(t, moduleNames["module1"])
		assert.True(t, moduleNames["module2"])
	})

	t.Run("empty list", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		searchModules, err := adapter.ListModules()

		require.NoError(t, err)
		assert.Empty(t, searchModules)
	})

	t.Run("storage error", func(t *testing.T) {
		storage := newMockStorage()
		storage.listModulesError = errors.New("database connection failed")
		adapter := NewSearchStorageAdapter(storage)

		searchModules, err := adapter.ListModules()

		assert.Error(t, err)
		assert.Nil(t, searchModules)
		assert.Contains(t, err.Error(), "database connection failed")
	})
}

func TestSearchStorageAdapter_ListVersions(t *testing.T) {
	t.Run("success with multiple versions", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		// Add test versions
		version1 := &Version{
			ModuleName: "test-module",
			Version:    "v1.0.0",
			Files: []File{
				{Path: "test.proto", Content: "syntax = \"proto3\";"},
			},
			Dependencies: []string{"dep1"},
		}
		version2 := &Version{
			ModuleName: "test-module",
			Version:    "v2.0.0",
			Files: []File{
				{Path: "test.proto", Content: "syntax = \"proto3\";\npackage test;"},
				{Path: "other.proto", Content: "syntax = \"proto3\";"},
			},
			Dependencies: []string{"dep1", "dep2"},
		}
		err := storage.CreateVersion(version1)
		require.NoError(t, err)
		err = storage.CreateVersion(version2)
		require.NoError(t, err)

		// List versions through adapter
		searchVersions, err := adapter.ListVersions("test-module")

		require.NoError(t, err)
		assert.Len(t, searchVersions, 2)

		// Verify versions are converted properly
		for _, ver := range searchVersions {
			assert.Equal(t, "test-module", ver.ModuleName)
			assert.NotEmpty(t, ver.Version)
			assert.NotEmpty(t, ver.Files)
			assert.NotEmpty(t, ver.Dependencies)

			// Check files are converted
			for _, file := range ver.Files {
				assert.NotEmpty(t, file.Path)
				assert.NotEmpty(t, file.Content)
			}
		}
	})

	t.Run("empty list for module with no versions", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		searchVersions, err := adapter.ListVersions("empty-module")

		require.NoError(t, err)
		assert.Empty(t, searchVersions)
	})

	t.Run("storage error", func(t *testing.T) {
		storage := newMockStorage()
		storage.listVersionsError = errors.New("query failed")
		adapter := NewSearchStorageAdapter(storage)

		searchVersions, err := adapter.ListVersions("test-module")

		assert.Error(t, err)
		assert.Nil(t, searchVersions)
		assert.Contains(t, err.Error(), "query failed")
	})

	t.Run("versions with empty files", func(t *testing.T) {
		storage := newMockStorage()
		adapter := NewSearchStorageAdapter(storage)

		// Version with no files or dependencies
		version := &Version{
			ModuleName:   "test-module",
			Version:      "v1.0.0",
			Files:        []File{},
			Dependencies: []string{},
		}
		err := storage.CreateVersion(version)
		require.NoError(t, err)

		searchVersions, err := adapter.ListVersions("test-module")

		require.NoError(t, err)
		assert.Len(t, searchVersions, 1)
		assert.Empty(t, searchVersions[0].Files)
		assert.Empty(t, searchVersions[0].Dependencies)
	})
}
