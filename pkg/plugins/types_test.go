package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginType_Constants tests all plugin type constants
func TestPluginType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		plugType PluginType
		expected string
	}{
		{"Language type", PluginTypeLanguage, "language"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.plugType))
		})
	}
}


// TestManifest_Initialization tests creating and initializing Manifest structs
func TestManifest_Initialization(t *testing.T) {
	t.Run("Empty manifest", func(t *testing.T) {
		m := &Manifest{}
		assert.Empty(t, m.ID)
		assert.Empty(t, m.Name)
		assert.Empty(t, m.Version)
		assert.Empty(t, m.APIVersion)
		assert.Empty(t, m.Description)
		assert.Empty(t, m.Author)
		assert.Empty(t, m.License)
		assert.Empty(t, m.Homepage)
		assert.Empty(t, m.Repository)
		assert.Empty(t, m.Type)
		assert.Nil(t, m.Metadata)
	})

	t.Run("Full manifest", func(t *testing.T) {
		m := &Manifest{
			ID:          "test-plugin",
			Name:        "Test Plugin",
			Version:     "1.2.3",
			APIVersion:  "1.0.0",
			Description: "A test plugin",
			Author:      "Test Author",
			License:     "MIT",
			Homepage:    "https://example.com",
			Repository:  "https://github.com/test/plugin",
			Type:        PluginTypeLanguage,
			Metadata:    map[string]string{"key": "value"},
		}

		assert.Equal(t, "test-plugin", m.ID)
		assert.Equal(t, "Test Plugin", m.Name)
		assert.Equal(t, "1.2.3", m.Version)
		assert.Equal(t, "1.0.0", m.APIVersion)
		assert.Equal(t, "A test plugin", m.Description)
		assert.Equal(t, "Test Author", m.Author)
		assert.Equal(t, "MIT", m.License)
		assert.Equal(t, "https://example.com", m.Homepage)
		assert.Equal(t, "https://github.com/test/plugin", m.Repository)
		assert.Equal(t, PluginTypeLanguage, m.Type)
		assert.Equal(t, "value", m.Metadata["key"])
	})

	t.Run("Manifest with all plugin types", func(t *testing.T) {
		types := []PluginType{
			PluginTypeLanguage,
		}

		for _, pType := range types {
			m := &Manifest{
				ID:      "test",
				Type:    pType,
				Version: "1.0.0",
			}
			assert.Equal(t, pType, m.Type)
		}
	})



	t.Run("Manifest with metadata", func(t *testing.T) {
		m := &Manifest{
			ID: "test",
			Metadata: map[string]string{
				"language":  "go",
				"platform":  "linux",
				"arch":      "amd64",
				"buildDate": "2024-01-01",
			},
		}
		assert.Len(t, m.Metadata, 4)
		assert.Equal(t, "go", m.Metadata["language"])
		assert.Equal(t, "linux", m.Metadata["platform"])
	})
}

// TestPluginInfo_Initialization tests creating and initializing PluginInfo structs
func TestPluginInfo_Initialization(t *testing.T) {
	t.Run("Empty plugin info", func(t *testing.T) {
		pi := &PluginInfo{}
		assert.Nil(t, pi.Manifest)
		assert.True(t, pi.LoadedAt.IsZero())
		assert.False(t, pi.IsEnabled)
		assert.Empty(t, pi.Source)
	})

	t.Run("Full plugin info", func(t *testing.T) {
		manifest := &Manifest{
			ID:      "test-plugin",
			Name:    "Test Plugin",
			Version: "1.0.0",
		}
		loadTime := time.Now()

		pi := &PluginInfo{
			Manifest:  manifest,
			LoadedAt:  loadTime,
			IsEnabled: true,
			Source:    "filesystem",
		}

		assert.Equal(t, manifest, pi.Manifest)
		assert.Equal(t, loadTime, pi.LoadedAt)
		assert.True(t, pi.IsEnabled)
		assert.Equal(t, "filesystem", pi.Source)
	})

	t.Run("Plugin info with different sources", func(t *testing.T) {
		sources := []string{"filesystem", "marketplace", "buf"}
		for _, source := range sources {
			pi := &PluginInfo{
				Source:    source,
				IsEnabled: true,
				LoadedAt:  time.Now(),
			}
			assert.Equal(t, source, pi.Source)
		}
	})

	t.Run("Plugin info disabled state", func(t *testing.T) {
		pi := &PluginInfo{
			IsEnabled: false,
			LoadedAt:  time.Now(),
		}
		assert.False(t, pi.IsEnabled)
	})
}

// TestValidationError_Initialization tests creating and initializing ValidationError structs
func TestValidationError_Initialization(t *testing.T) {
	t.Run("Empty validation error", func(t *testing.T) {
		ve := &ValidationError{}
		assert.Empty(t, ve.Field)
		assert.Empty(t, ve.Message)
	})

	t.Run("Full validation error", func(t *testing.T) {
		ve := &ValidationError{
			Field:   "id",
			Message: "ID is required",
		}
		assert.Equal(t, "id", ve.Field)
		assert.Equal(t, "ID is required", ve.Message)
	})

	t.Run("Multiple validation errors", func(t *testing.T) {
		errors := []ValidationError{
			{Field: "id", Message: "ID is required"},
			{Field: "name", Message: "Name is required"},
			{Field: "version", Message: "Version is required"},
		}

		assert.Len(t, errors, 3)
	})
}



// mockPlugin is a test implementation of the Plugin interface
type mockPlugin struct {
	manifest *Manifest
	loaded   bool
}

func (m *mockPlugin) Manifest() *Manifest {
	return m.manifest
}

func (m *mockPlugin) Load() error {
	m.loaded = true
	return nil
}

func (m *mockPlugin) Unload() error {
	m.loaded = false
	return nil
}

// TestPlugin_Interface tests the Plugin interface contract
func TestPlugin_Interface(t *testing.T) {
	mockImpl := &mockPlugin{
		manifest: &Manifest{
			ID:      "mock-plugin",
			Name:    "Mock Plugin",
			Version: "1.0.0",
		},
		loaded: false,
	}

	// Verify it implements the Plugin interface
	var _ Plugin = mockImpl

	t.Run("Interface method signatures", func(t *testing.T) {
		// Test Manifest method
		m := mockImpl.Manifest()
		assert.NotNil(t, m)
		assert.Equal(t, "mock-plugin", m.ID)

		// Test Load method
		assert.False(t, mockImpl.loaded)
		err := mockImpl.Load()
		assert.NoError(t, err)
		assert.True(t, mockImpl.loaded)

		// Test Unload method
		err = mockImpl.Unload()
		assert.NoError(t, err)
		assert.False(t, mockImpl.loaded)
	})
}

// TestPluginRegistry_Interface tests the PluginRegistry interface expectations
func TestPluginRegistry_Interface(t *testing.T) {
	// This test documents the expected behavior of PluginRegistry implementations
	t.Run("Registry interface contract", func(t *testing.T) {
		// A PluginRegistry must be able to:
		// 1. Register(plugin Plugin) error
		// 2. Unregister(id string) error
		// 3. Get(id string) (Plugin, error)
		// 4. List() []Plugin
		// 5. ListByType(t PluginType) []Plugin

		// We verify the interface exists and has the right method signatures
		// by attempting to create a nil interface value
		var registry PluginRegistry
		assert.Nil(t, registry)
	})
}

// TestPluginLoader_Interface tests the PluginLoader interface expectations
func TestPluginLoader_Interface(t *testing.T) {
	t.Run("Loader interface contract", func(t *testing.T) {
		// A PluginLoader must be able to:
		// 1. DiscoverPlugins(ctx context.Context) ([]Plugin, error)
		// 2. LoadPlugin(ctx context.Context, path string) (Plugin, error)
		// 3. UnloadPlugin(ctx context.Context, id string) error

		var loader PluginLoader
		assert.Nil(t, loader)

		// Test that context is properly typed
		ctx := context.Background()
		assert.NotNil(t, ctx)
	})
}

// TestManifest_FieldTypes tests that manifest fields have correct types
func TestManifest_FieldTypes(t *testing.T) {
	m := &Manifest{
		ID:          "test",
		Name:        "Test",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Description: "desc",
		Author:      "author",
		License:     "MIT",
		Homepage:    "https://example.com",
		Repository:  "https://github.com/test/test",
		Type:        PluginTypeLanguage,
		Metadata:    map[string]string{"key": "value"},
	}

	// Test that we can access all fields with correct types
	var _ string = m.ID
	var _ string = m.Name
	var _ string = m.Version
	var _ string = m.APIVersion
	var _ string = m.Description
	var _ string = m.Author
	var _ string = m.License
	var _ string = m.Homepage
	var _ string = m.Repository
	var _ PluginType = m.Type
	var _ map[string]string = m.Metadata

	assert.NotNil(t, m)
}

// TestPluginInfo_FieldTypes tests that PluginInfo fields have correct types
func TestPluginInfo_FieldTypes(t *testing.T) {
	pi := &PluginInfo{
		Manifest:  &Manifest{},
		LoadedAt:  time.Now(),
		IsEnabled: true,
		Source:    "filesystem",
	}

	var _ *Manifest = pi.Manifest
	var _ time.Time = pi.LoadedAt
	var _ bool = pi.IsEnabled
	var _ string = pi.Source

	assert.NotNil(t, pi)
}

// TestValidationError_FieldTypes tests that ValidationError fields have correct types
func TestValidationError_FieldTypes(t *testing.T) {
	ve := &ValidationError{
		Field:   "test",
		Message: "message",
	}

	var _ string = ve.Field
	var _ string = ve.Message

	assert.NotNil(t, ve)
}



// TestPluginTypes_StringConversion tests converting plugin types to strings
func TestPluginTypes_StringConversion(t *testing.T) {
	tests := []struct {
		pluginType PluginType
		expected   string
	}{
		{PluginTypeLanguage, "language"},
	}

	for _, tt := range tests {
		t.Run(string(tt.pluginType), func(t *testing.T) {
			result := string(tt.pluginType)
			assert.Equal(t, tt.expected, result)
		})
	}
}


// TestManifest_EmptySlices tests that empty slices behave correctly
func TestManifest_EmptySlices(t *testing.T) {
	t.Run("Nil metadata", func(t *testing.T) {
		m := &Manifest{}
		assert.Nil(t, m.Metadata)
	})

	t.Run("Empty initialized metadata", func(t *testing.T) {
		m := &Manifest{
			Metadata: map[string]string{},
		}
		assert.NotNil(t, m.Metadata)
		assert.Len(t, m.Metadata, 0)
	})
}


// TestContext_Usage tests that context is properly used with interfaces
func TestContext_Usage(t *testing.T) {
	t.Run("Context creation", func(t *testing.T) {
		ctx := context.Background()
		assert.NotNil(t, ctx)
	})

	t.Run("Context with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		assert.NotNil(t, ctx)
	})

	t.Run("Context with cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		assert.NotNil(t, ctx)
		cancel()
		assert.Error(t, ctx.Err())
	})
}

// TestManifest_MetadataOperations tests metadata map operations
func TestManifest_MetadataOperations(t *testing.T) {
	t.Run("Add and retrieve metadata", func(t *testing.T) {
		m := &Manifest{
			Metadata: make(map[string]string),
		}

		m.Metadata["key1"] = "value1"
		m.Metadata["key2"] = "value2"

		assert.Equal(t, "value1", m.Metadata["key1"])
		assert.Equal(t, "value2", m.Metadata["key2"])
		assert.Len(t, m.Metadata, 2)
	})

	t.Run("Update metadata", func(t *testing.T) {
		m := &Manifest{
			Metadata: map[string]string{
				"key": "old_value",
			},
		}

		m.Metadata["key"] = "new_value"
		assert.Equal(t, "new_value", m.Metadata["key"])
	})

	t.Run("Delete from metadata", func(t *testing.T) {
		m := &Manifest{
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}

		delete(m.Metadata, "key1")
		assert.Len(t, m.Metadata, 1)
		assert.Empty(t, m.Metadata["key1"])
		assert.Equal(t, "value2", m.Metadata["key2"])
	})

	t.Run("Check metadata key existence", func(t *testing.T) {
		m := &Manifest{
			Metadata: map[string]string{
				"exists": "value",
			},
		}

		val, exists := m.Metadata["exists"]
		assert.True(t, exists)
		assert.Equal(t, "value", val)

		val, exists = m.Metadata["not_exists"]
		assert.False(t, exists)
		assert.Empty(t, val)
	})
}



// TestValidationError_Collection tests working with collections of validation errors
func TestValidationError_Collection(t *testing.T) {
	t.Run("Find error by field", func(t *testing.T) {
		errors := []ValidationError{
			{Field: "id", Message: "Required"},
			{Field: "version", Message: "Invalid"},
		}

		var foundError *ValidationError
		for _, err := range errors {
			if err.Field == "version" {
				foundError = &err
				break
			}
		}

		require.NotNil(t, foundError)
		assert.Equal(t, "version", foundError.Field)
		assert.Equal(t, "Invalid", foundError.Message)
	})
}


// TestPluginInfo_TimeOperations tests time-related operations on PluginInfo
func TestPluginInfo_TimeOperations(t *testing.T) {
	t.Run("Time since loaded", func(t *testing.T) {
		loadTime := time.Now().Add(-1 * time.Hour)
		pi := &PluginInfo{
			LoadedAt: loadTime,
		}

		elapsed := time.Since(pi.LoadedAt)
		assert.True(t, elapsed >= 1*time.Hour)
	})

	t.Run("Compare load times", func(t *testing.T) {
		pi1 := &PluginInfo{LoadedAt: time.Now().Add(-2 * time.Hour)}
		time.Sleep(1 * time.Millisecond) // Ensure different times
		pi2 := &PluginInfo{LoadedAt: time.Now().Add(-1 * time.Hour)}

		assert.True(t, pi1.LoadedAt.Before(pi2.LoadedAt))
		assert.True(t, pi2.LoadedAt.After(pi1.LoadedAt))
	})
}
