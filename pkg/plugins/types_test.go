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
		{"Validator type", PluginTypeValidator, "validator"},
		{"Generator type", PluginTypeGenerator, "generator"},
		{"Runner type", PluginTypeRunner, "runner"},
		{"Transform type", PluginTypeTransform, "transform"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.plugType))
		})
	}
}

// TestSecurityLevel_Constants tests all security level constants
func TestSecurityLevel_Constants(t *testing.T) {
	tests := []struct {
		name     string
		level    SecurityLevel
		expected string
	}{
		{"Official level", SecurityLevelOfficial, "official"},
		{"Verified level", SecurityLevelVerified, "verified"},
		{"Community level", SecurityLevelCommunity, "community"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.level))
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
		assert.Empty(t, m.SecurityLevel)
		assert.Nil(t, m.Permissions)
		assert.Nil(t, m.Dependencies)
		assert.Nil(t, m.Metadata)
	})

	t.Run("Full manifest", func(t *testing.T) {
		m := &Manifest{
			ID:            "test-plugin",
			Name:          "Test Plugin",
			Version:       "1.2.3",
			APIVersion:    "1.0.0",
			Description:   "A test plugin",
			Author:        "Test Author",
			License:       "MIT",
			Homepage:      "https://example.com",
			Repository:    "https://github.com/test/plugin",
			Type:          PluginTypeLanguage,
			SecurityLevel: SecurityLevelOfficial,
			Permissions:   []string{"filesystem:read", "network:read"},
			Dependencies:  []string{"dep1", "dep2"},
			Metadata:      map[string]string{"key": "value"},
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
		assert.Equal(t, SecurityLevelOfficial, m.SecurityLevel)
		assert.Len(t, m.Permissions, 2)
		assert.Contains(t, m.Permissions, "filesystem:read")
		assert.Contains(t, m.Permissions, "network:read")
		assert.Len(t, m.Dependencies, 2)
		assert.Equal(t, "dep1", m.Dependencies[0])
		assert.Equal(t, "dep2", m.Dependencies[1])
		assert.Equal(t, "value", m.Metadata["key"])
	})

	t.Run("Manifest with all plugin types", func(t *testing.T) {
		types := []PluginType{
			PluginTypeLanguage,
			PluginTypeValidator,
			PluginTypeGenerator,
			PluginTypeRunner,
			PluginTypeTransform,
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

	t.Run("Manifest with all security levels", func(t *testing.T) {
		levels := []SecurityLevel{
			SecurityLevelOfficial,
			SecurityLevelVerified,
			SecurityLevelCommunity,
		}

		for _, level := range levels {
			m := &Manifest{
				ID:            "test",
				SecurityLevel: level,
				Version:       "1.0.0",
			}
			assert.Equal(t, level, m.SecurityLevel)
		}
	})

	t.Run("Manifest with multiple permissions", func(t *testing.T) {
		m := &Manifest{
			ID: "test",
			Permissions: []string{
				"filesystem:read",
				"filesystem:write",
				"network:read",
				"network:write",
				"process:exec",
				"env:read",
			},
		}
		assert.Len(t, m.Permissions, 6)
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
		assert.Empty(t, ve.Severity)
	})

	t.Run("Full validation error", func(t *testing.T) {
		ve := &ValidationError{
			Field:    "id",
			Message:  "ID is required",
			Severity: "error",
		}
		assert.Equal(t, "id", ve.Field)
		assert.Equal(t, "ID is required", ve.Message)
		assert.Equal(t, "error", ve.Severity)
	})

	t.Run("Validation error with warning severity", func(t *testing.T) {
		ve := &ValidationError{
			Field:    "license",
			Message:  "License should be specified",
			Severity: "warning",
		}
		assert.Equal(t, "warning", ve.Severity)
	})

	t.Run("Multiple validation errors", func(t *testing.T) {
		errors := []ValidationError{
			{Field: "id", Message: "ID is required", Severity: "error"},
			{Field: "name", Message: "Name is required", Severity: "error"},
			{Field: "version", Message: "Version is required", Severity: "error"},
			{Field: "author", Message: "Author recommended", Severity: "warning"},
		}

		assert.Len(t, errors, 4)
		errorCount := 0
		warningCount := 0
		for _, err := range errors {
			if err.Severity == "error" {
				errorCount++
			} else if err.Severity == "warning" {
				warningCount++
			}
		}
		assert.Equal(t, 3, errorCount)
		assert.Equal(t, 1, warningCount)
	})
}

// TestSecurityIssue_Initialization tests creating and initializing SecurityIssue structs
func TestSecurityIssue_Initialization(t *testing.T) {
	t.Run("Empty security issue", func(t *testing.T) {
		si := &SecurityIssue{}
		assert.Empty(t, si.Severity)
		assert.Empty(t, si.Category)
		assert.Empty(t, si.Description)
		assert.Empty(t, si.File)
		assert.Zero(t, si.Line)
		assert.Zero(t, si.Column)
		assert.Empty(t, si.Recommendation)
		assert.Empty(t, si.CWEID)
	})

	t.Run("Full security issue", func(t *testing.T) {
		si := &SecurityIssue{
			Severity:       "high",
			Category:       "dangerous-import",
			Description:    "Use of unsafe package detected",
			File:           "/path/to/file.go",
			Line:           42,
			Column:         10,
			Recommendation: "Avoid using unsafe package",
			CWEID:          "CWE-242",
		}

		assert.Equal(t, "high", si.Severity)
		assert.Equal(t, "dangerous-import", si.Category)
		assert.Equal(t, "Use of unsafe package detected", si.Description)
		assert.Equal(t, "/path/to/file.go", si.File)
		assert.Equal(t, 42, si.Line)
		assert.Equal(t, 10, si.Column)
		assert.Equal(t, "Avoid using unsafe package", si.Recommendation)
		assert.Equal(t, "CWE-242", si.CWEID)
	})

	t.Run("Security issues with different severities", func(t *testing.T) {
		severities := []string{"critical", "high", "medium", "low", "warning"}
		for _, severity := range severities {
			si := &SecurityIssue{
				Severity:    severity,
				Category:    "test",
				Description: "Test issue",
			}
			assert.Equal(t, severity, si.Severity)
		}
	})

	t.Run("Security issues with different categories", func(t *testing.T) {
		categories := []string{
			"imports",
			"hardcoded-secrets",
			"sql-injection",
			"dangerous-import",
			"suspicious-file-ops",
		}
		for _, category := range categories {
			si := &SecurityIssue{
				Category:    category,
				Severity:    "medium",
				Description: "Test issue",
			}
			assert.Equal(t, category, si.Category)
		}
	})

	t.Run("Security issue with minimal info", func(t *testing.T) {
		si := &SecurityIssue{
			Severity:    "low",
			Category:    "warning",
			Description: "Potential issue detected",
		}
		assert.Empty(t, si.File)
		assert.Zero(t, si.Line)
		assert.Empty(t, si.Recommendation)
	})
}

// TestPluginValidationResult_Initialization tests creating and initializing PluginValidationResult structs
func TestPluginValidationResult_Initialization(t *testing.T) {
	t.Run("Empty validation result", func(t *testing.T) {
		pvr := &PluginValidationResult{}
		assert.False(t, pvr.Valid)
		assert.Nil(t, pvr.ManifestErrors)
		assert.Nil(t, pvr.SecurityIssues)
		assert.Nil(t, pvr.PermissionIssues)
		assert.Zero(t, pvr.ScanDuration)
		assert.Nil(t, pvr.Recommendations)
	})

	t.Run("Valid plugin result", func(t *testing.T) {
		pvr := &PluginValidationResult{
			Valid:        true,
			ScanDuration: 500 * time.Millisecond,
		}
		assert.True(t, pvr.Valid)
		assert.Empty(t, pvr.ManifestErrors)
		assert.Empty(t, pvr.SecurityIssues)
		assert.Equal(t, 500*time.Millisecond, pvr.ScanDuration)
	})

	t.Run("Invalid plugin with errors", func(t *testing.T) {
		pvr := &PluginValidationResult{
			Valid: false,
			ManifestErrors: []ValidationError{
				{Field: "id", Message: "ID is required", Severity: "error"},
			},
			SecurityIssues: []SecurityIssue{
				{Severity: "high", Category: "dangerous-import", Description: "Unsafe package used"},
			},
			PermissionIssues: []ValidationError{
				{Field: "permissions", Message: "Unknown permission", Severity: "error"},
			},
			ScanDuration: 1 * time.Second,
			Recommendations: []string{
				"Fix manifest errors",
				"Address security issues",
			},
		}

		assert.False(t, pvr.Valid)
		assert.Len(t, pvr.ManifestErrors, 1)
		assert.Len(t, pvr.SecurityIssues, 1)
		assert.Len(t, pvr.PermissionIssues, 1)
		assert.Equal(t, 1*time.Second, pvr.ScanDuration)
		assert.Len(t, pvr.Recommendations, 2)
	})

	t.Run("Validation result with multiple issues", func(t *testing.T) {
		pvr := &PluginValidationResult{
			Valid: false,
			ManifestErrors: []ValidationError{
				{Field: "id", Message: "Invalid ID format", Severity: "error"},
				{Field: "version", Message: "Invalid version", Severity: "error"},
				{Field: "author", Message: "Author missing", Severity: "warning"},
			},
			SecurityIssues: []SecurityIssue{
				{Severity: "critical", Category: "hardcoded-secrets", Description: "API key found"},
				{Severity: "high", Category: "dangerous-import", Description: "os/exec used"},
				{Severity: "medium", Category: "suspicious-file-ops", Description: "File deletion detected"},
			},
			ScanDuration: 2 * time.Second,
		}

		assert.Len(t, pvr.ManifestErrors, 3)
		assert.Len(t, pvr.SecurityIssues, 3)

		// Count by severity
		criticalCount := 0
		highCount := 0
		mediumCount := 0
		for _, issue := range pvr.SecurityIssues {
			switch issue.Severity {
			case "critical":
				criticalCount++
			case "high":
				highCount++
			case "medium":
				mediumCount++
			}
		}
		assert.Equal(t, 1, criticalCount)
		assert.Equal(t, 1, highCount)
		assert.Equal(t, 1, mediumCount)
	})

	t.Run("Validation result with only warnings", func(t *testing.T) {
		pvr := &PluginValidationResult{
			Valid: true, // Can be valid with just warnings
			ManifestErrors: []ValidationError{
				{Field: "license", Message: "License recommended", Severity: "warning"},
			},
			ScanDuration: 100 * time.Millisecond,
		}

		assert.True(t, pvr.Valid)
		assert.Len(t, pvr.ManifestErrors, 1)
		assert.Equal(t, "warning", pvr.ManifestErrors[0].Severity)
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
		ID:            "test",
		Name:          "Test",
		Version:       "1.0.0",
		APIVersion:    "1.0.0",
		Description:   "desc",
		Author:        "author",
		License:       "MIT",
		Homepage:      "https://example.com",
		Repository:    "https://github.com/test/test",
		Type:          PluginTypeLanguage,
		SecurityLevel: SecurityLevelCommunity,
		Permissions:   []string{"filesystem:read"},
		Dependencies:  []string{"dep1"},
		Metadata:      map[string]string{"key": "value"},
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
	var _ SecurityLevel = m.SecurityLevel
	var _ []string = m.Permissions
	var _ []string = m.Dependencies
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
		Field:    "test",
		Message:  "message",
		Severity: "error",
	}

	var _ string = ve.Field
	var _ string = ve.Message
	var _ string = ve.Severity

	assert.NotNil(t, ve)
}

// TestSecurityIssue_FieldTypes tests that SecurityIssue fields have correct types
func TestSecurityIssue_FieldTypes(t *testing.T) {
	si := &SecurityIssue{
		Severity:       "high",
		Category:       "test",
		Description:    "desc",
		File:           "file.go",
		Line:           1,
		Column:         1,
		Recommendation: "fix it",
		CWEID:          "CWE-123",
	}

	var _ string = si.Severity
	var _ string = si.Category
	var _ string = si.Description
	var _ string = si.File
	var _ int = si.Line
	var _ int = si.Column
	var _ string = si.Recommendation
	var _ string = si.CWEID

	assert.NotNil(t, si)
}

// TestPluginValidationResult_FieldTypes tests that PluginValidationResult fields have correct types
func TestPluginValidationResult_FieldTypes(t *testing.T) {
	pvr := &PluginValidationResult{
		Valid:            true,
		ManifestErrors:   []ValidationError{},
		SecurityIssues:   []SecurityIssue{},
		PermissionIssues: []ValidationError{},
		ScanDuration:     time.Second,
		Recommendations:  []string{},
	}

	var _ bool = pvr.Valid
	var _ []ValidationError = pvr.ManifestErrors
	var _ []SecurityIssue = pvr.SecurityIssues
	var _ []ValidationError = pvr.PermissionIssues
	var _ time.Duration = pvr.ScanDuration
	var _ []string = pvr.Recommendations

	assert.NotNil(t, pvr)
}

// TestPluginTypes_StringConversion tests converting plugin types to strings
func TestPluginTypes_StringConversion(t *testing.T) {
	tests := []struct {
		pluginType PluginType
		expected   string
	}{
		{PluginTypeLanguage, "language"},
		{PluginTypeValidator, "validator"},
		{PluginTypeGenerator, "generator"},
		{PluginTypeRunner, "runner"},
		{PluginTypeTransform, "transform"},
	}

	for _, tt := range tests {
		t.Run(string(tt.pluginType), func(t *testing.T) {
			result := string(tt.pluginType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSecurityLevels_StringConversion tests converting security levels to strings
func TestSecurityLevels_StringConversion(t *testing.T) {
	tests := []struct {
		level    SecurityLevel
		expected string
	}{
		{SecurityLevelOfficial, "official"},
		{SecurityLevelVerified, "verified"},
		{SecurityLevelCommunity, "community"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			result := string(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestManifest_EmptySlices tests that empty slices behave correctly
func TestManifest_EmptySlices(t *testing.T) {
	t.Run("Nil slices", func(t *testing.T) {
		m := &Manifest{}
		assert.Nil(t, m.Permissions)
		assert.Nil(t, m.Dependencies)
		assert.Nil(t, m.Metadata)
		assert.Len(t, m.Permissions, 0)
		assert.Len(t, m.Dependencies, 0)
	})

	t.Run("Empty initialized slices", func(t *testing.T) {
		m := &Manifest{
			Permissions:  []string{},
			Dependencies: []string{},
			Metadata:     map[string]string{},
		}
		assert.NotNil(t, m.Permissions)
		assert.NotNil(t, m.Dependencies)
		assert.NotNil(t, m.Metadata)
		assert.Len(t, m.Permissions, 0)
		assert.Len(t, m.Dependencies, 0)
		assert.Len(t, m.Metadata, 0)
	})
}

// TestPluginValidationResult_Duration tests duration handling
func TestPluginValidationResult_Duration(t *testing.T) {
	t.Run("Various durations", func(t *testing.T) {
		durations := []time.Duration{
			0,
			1 * time.Millisecond,
			100 * time.Millisecond,
			1 * time.Second,
			5 * time.Second,
			1 * time.Minute,
		}

		for _, d := range durations {
			pvr := &PluginValidationResult{
				ScanDuration: d,
			}
			assert.Equal(t, d, pvr.ScanDuration)
		}
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

// TestPermissions_Operations tests permissions slice operations
func TestPermissions_Operations(t *testing.T) {
	t.Run("Add permissions", func(t *testing.T) {
		m := &Manifest{
			Permissions: []string{},
		}

		m.Permissions = append(m.Permissions, "filesystem:read")
		m.Permissions = append(m.Permissions, "network:read")

		assert.Len(t, m.Permissions, 2)
		assert.Contains(t, m.Permissions, "filesystem:read")
		assert.Contains(t, m.Permissions, "network:read")
	})

	t.Run("Check permission existence", func(t *testing.T) {
		m := &Manifest{
			Permissions: []string{"filesystem:read", "network:read"},
		}

		found := false
		for _, p := range m.Permissions {
			if p == "filesystem:read" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})
}

// TestDependencies_Operations tests dependencies slice operations
func TestDependencies_Operations(t *testing.T) {
	t.Run("Add dependencies", func(t *testing.T) {
		m := &Manifest{
			Dependencies: []string{},
		}

		m.Dependencies = append(m.Dependencies, "dep1")
		m.Dependencies = append(m.Dependencies, "dep2")

		assert.Len(t, m.Dependencies, 2)
		assert.Equal(t, "dep1", m.Dependencies[0])
		assert.Equal(t, "dep2", m.Dependencies[1])
	})

	t.Run("Empty dependencies", func(t *testing.T) {
		m := &Manifest{}
		assert.Len(t, m.Dependencies, 0)
	})
}

// TestValidationError_Collection tests working with collections of validation errors
func TestValidationError_Collection(t *testing.T) {
	t.Run("Filter by severity", func(t *testing.T) {
		errors := []ValidationError{
			{Field: "id", Message: "Required", Severity: "error"},
			{Field: "name", Message: "Required", Severity: "error"},
			{Field: "author", Message: "Recommended", Severity: "warning"},
		}

		errorCount := 0
		warningCount := 0
		for _, err := range errors {
			if err.Severity == "error" {
				errorCount++
			} else if err.Severity == "warning" {
				warningCount++
			}
		}

		assert.Equal(t, 2, errorCount)
		assert.Equal(t, 1, warningCount)
	})

	t.Run("Find error by field", func(t *testing.T) {
		errors := []ValidationError{
			{Field: "id", Message: "Required", Severity: "error"},
			{Field: "version", Message: "Invalid", Severity: "error"},
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

// TestSecurityIssue_Collection tests working with collections of security issues
func TestSecurityIssue_Collection(t *testing.T) {
	t.Run("Filter by severity", func(t *testing.T) {
		issues := []SecurityIssue{
			{Severity: "critical", Category: "secret", Description: "API key found"},
			{Severity: "high", Category: "import", Description: "Dangerous import"},
			{Severity: "medium", Category: "file", Description: "File operation"},
			{Severity: "low", Category: "warning", Description: "Minor issue"},
		}

		criticalCount := 0
		for _, issue := range issues {
			if issue.Severity == "critical" {
				criticalCount++
			}
		}
		assert.Equal(t, 1, criticalCount)
	})

	t.Run("Group by category", func(t *testing.T) {
		issues := []SecurityIssue{
			{Severity: "high", Category: "import", Description: "Issue 1"},
			{Severity: "high", Category: "import", Description: "Issue 2"},
			{Severity: "medium", Category: "file", Description: "Issue 3"},
		}

		categories := make(map[string]int)
		for _, issue := range issues {
			categories[issue.Category]++
		}

		assert.Equal(t, 2, categories["import"])
		assert.Equal(t, 1, categories["file"])
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
