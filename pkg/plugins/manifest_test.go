package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadManifest tests loading a valid manifest from a file
func TestLoadManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "plugin.yaml")

	// Create a valid manifest
	manifest := &Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Description: "A test plugin",
		Author:      "Test Author",
		License:     "MIT",
		Homepage:    "https://example.com",
		Repository:  "https://github.com/example/test-plugin",
		Metadata:    map[string]string{"key": "value"},
	}

	err := SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	// Load the manifest
	loaded, err := LoadManifest(manifestPath)
	assert.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, "test-plugin", loaded.ID)
	assert.Equal(t, "Test Plugin", loaded.Name)
	assert.Equal(t, "1.0.0", loaded.Version)
	assert.Equal(t, "1.0.0", loaded.APIVersion)
	assert.Equal(t, PluginTypeLanguage, loaded.Type)
	assert.Equal(t, "A test plugin", loaded.Description)
	assert.Equal(t, "Test Author", loaded.Author)
	assert.Equal(t, "MIT", loaded.License)
	assert.Equal(t, "https://example.com", loaded.Homepage)
	assert.Equal(t, "https://github.com/example/test-plugin", loaded.Repository)
	assert.Equal(t, "value", loaded.Metadata["key"])
}

// TestLoadManifest_NonexistentFile tests loading from a non-existent file
func TestLoadManifest_NonexistentFile(t *testing.T) {
	loaded, err := LoadManifest("/nonexistent/path/plugin.yaml")
	assert.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "failed to read manifest")
}

// TestLoadManifest_InvalidYAML tests loading invalid YAML content
func TestLoadManifest_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	err := os.WriteFile(manifestPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	loaded, err := LoadManifest(manifestPath)
	assert.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

// TestLoadManifestFromDir tests loading a manifest from a directory
func TestLoadManifestFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid manifest
	manifest := &Manifest{
		ID:         "test-plugin",
		Name:       "Test Plugin",
		Version:    "1.0.0",
		APIVersion: "1.0.0",
		Type:       PluginTypeLanguage,
	}

	manifestPath := filepath.Join(tmpDir, "plugin.yaml")
	err := SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	// Load from directory
	loaded, err := LoadManifestFromDir(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, "test-plugin", loaded.ID)
}

// TestLoadManifestFromDir_NoManifest tests loading from a directory without plugin.yaml
func TestLoadManifestFromDir_NoManifest(t *testing.T) {
	tmpDir := t.TempDir()

	loaded, err := LoadManifestFromDir(tmpDir)
	assert.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "failed to read manifest")
}

// TestSaveManifest tests saving a manifest to a file
func TestSaveManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "plugin.yaml")

	manifest := &Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "2.1.3",
		APIVersion:  "1.5.0",
		Type:        PluginTypeValidator,
		Description: "Test description",
		Author:      "Test Author",
	}

	err := SaveManifest(manifest, manifestPath)
	assert.NoError(t, err)

	// Verify file exists and can be read
	data, err := os.ReadFile(manifestPath)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "test-plugin")
	assert.Contains(t, string(data), "Test Plugin")
	assert.Contains(t, string(data), "2.1.3")
}

// TestSaveManifest_InvalidPath tests saving to an invalid path
func TestSaveManifest_InvalidPath(t *testing.T) {
	manifest := &Manifest{
		ID:   "test-plugin",
		Name: "Test Plugin",
	}

	err := SaveManifest(manifest, "/nonexistent/deeply/nested/path/plugin.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write manifest")
}

// TestManifestValidation_Valid tests validation of a valid manifest
func TestManifestValidation_Valid(t *testing.T) {
	manifest := &Manifest{
		ID:         "valid-plugin",
		Name:       "Valid Plugin",
		Version:    "1.0.0",
		APIVersion: "1.0.0",
		Type:       PluginTypeLanguage,
	}

	errors := ValidateManifest(manifest)
	assert.Empty(t, errors)
}

// TestManifestValidation_MissingRequiredFields tests validation with missing required fields
func TestManifestValidation_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name          string
		manifest      *Manifest
		expectedField string
		expectedMsg   string
	}{
		{
			name:          "missing ID",
			manifest:      &Manifest{Name: "Test", Version: "1.0.0", APIVersion: "1.0.0", Type: PluginTypeLanguage},
			expectedField: "id",
			expectedMsg:   "Plugin ID is required",
		},
		{
			name:          "missing Name",
			manifest:      &Manifest{ID: "test", Version: "1.0.0", APIVersion: "1.0.0", Type: PluginTypeLanguage},
			expectedField: "name",
			expectedMsg:   "Plugin name is required",
		},
		{
			name:          "missing Version",
			manifest:      &Manifest{ID: "test", Name: "Test", APIVersion: "1.0.0", Type: PluginTypeLanguage},
			expectedField: "version",
			expectedMsg:   "Version is required",
		},
		{
			name:          "missing APIVersion",
			manifest:      &Manifest{ID: "test", Name: "Test", Version: "1.0.0", Type: PluginTypeLanguage},
			expectedField: "api_version",
			expectedMsg:   "API version is required",
		},
		{
			name:          "missing Type",
			manifest:      &Manifest{ID: "test", Name: "Test", Version: "1.0.0", APIVersion: "1.0.0"},
			expectedField: "type",
			expectedMsg:   "Plugin type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateManifest(tt.manifest)
			assert.NotEmpty(t, errors)
			found := false
			for _, err := range errors {
				if err.Field == tt.expectedField {
					assert.Contains(t, err.Message, tt.expectedMsg)
					found = true
					break
				}
			}
			assert.True(t, found, "Expected error for field %s not found", tt.expectedField)
		})
	}
}

// TestManifestValidation_InvalidSemver tests validation of invalid semantic versions
func TestManifestValidation_InvalidSemver(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		apiVersion string
	}{
		{"invalid version format", "1.0", "1.0.0"},
		{"invalid api version format", "1.0.0", "invalid"},
		{"non-numeric version", "abc.def.ghi", "1.0.0"},
		{"missing patch", "1.0.", "1.0.0"},
		{"both invalid", "bad", "also-bad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := &Manifest{
				ID:         "test-plugin",
				Name:       "Test Plugin",
				Version:    tt.version,
				APIVersion: tt.apiVersion,
				Type:       PluginTypeLanguage,
			}

			errors := ValidateManifest(manifest)
			assert.NotEmpty(t, errors)

			// Check for version-related errors
			hasVersionError := false
			for _, err := range errors {
				if err.Field == "version" || err.Field == "api_version" {
					assert.Contains(t, err.Message, "Invalid semver format")
					hasVersionError = true
				}
			}
			assert.True(t, hasVersionError, "Expected semver validation error")
		})
	}
}

// TestManifestValidation_ValidSemverFormats tests various valid semver formats
func TestManifestValidation_ValidSemverFormats(t *testing.T) {
	validVersions := []string{
		"1.0.0",
		"v1.0.0",
		"2.3.4",
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-0.3.7",
		"1.0.0-x.7.z.92",
		"1.0.0+20130313144700",
		"1.0.0-beta+exp.sha.5114f85",
		"v10.20.30",
	}

	for _, version := range validVersions {
		t.Run("version_"+version, func(t *testing.T) {
			manifest := &Manifest{
				ID:         "test-plugin",
				Name:       "Test Plugin",
				Version:    version,
				APIVersion: "1.0.0",
				Type:       PluginTypeLanguage,
			}

			errors := ValidateManifest(manifest)
			// Should have no version-related errors
			for _, err := range errors {
				assert.NotEqual(t, "version", err.Field)
			}
		})
	}
}

// TestManifestValidation_InvalidPluginType tests validation of invalid plugin types
func TestManifestValidation_InvalidPluginType(t *testing.T) {
	manifest := &Manifest{
		ID:         "test-plugin",
		Name:       "Test Plugin",
		Version:    "1.0.0",
		APIVersion: "1.0.0",
		Type:       PluginType("invalid-type"),
	}

	errors := ValidateManifest(manifest)
	assert.NotEmpty(t, errors)

	found := false
	for _, err := range errors {
		if err.Field == "type" {
			assert.Contains(t, err.Message, "Invalid plugin type")
			found = true
			break
		}
	}
	assert.True(t, found, "Expected type validation error")
}

// TestManifestValidation_ValidPluginTypes tests all valid plugin types
func TestManifestValidation_ValidPluginTypes(t *testing.T) {
	validTypes := []PluginType{
		PluginTypeLanguage,
		PluginTypeValidator,
		PluginTypeGenerator,
		PluginTypeRunner,
		PluginTypeTransform,
	}

	for _, pluginType := range validTypes {
		t.Run(string(pluginType), func(t *testing.T) {
			manifest := &Manifest{
				ID:         "test-plugin",
				Name:       "Test Plugin",
				Version:    "1.0.0",
				APIVersion: "1.0.0",
				Type:       pluginType,
			}

			errors := ValidateManifest(manifest)
			// Should have no type-related errors
			for _, err := range errors {
				assert.NotEqual(t, "type", err.Field)
			}
		})
	}
}

// TestManifestValidation_InvalidPermissions tests validation of invalid permissions
func TestManifestValidation_InvalidPermissions(t *testing.T) {
	manifest := &Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Permissions: []string{"filesystem:read", "invalid:permission", "another:bad"},
	}

	errors := ValidateManifest(manifest)
	assert.NotEmpty(t, errors)

	// Should have 2 permission errors
	permissionErrors := 0
	for _, err := range errors {
		if err.Field == "permissions" {
			assert.Contains(t, err.Message, "Unknown permission")
			permissionErrors++
		}
	}
	assert.Equal(t, 2, permissionErrors, "Expected 2 permission validation errors")
}

// TestManifestValidation_ValidPermissions tests all valid permissions
func TestManifestValidation_ValidPermissions(t *testing.T) {
	manifest := &Manifest{
		ID:         "test-plugin",
		Name:       "Test Plugin",
		Version:    "1.0.0",
		APIVersion: "1.0.0",
		Type:       PluginTypeLanguage,
		Permissions: []string{
			"filesystem:read",
			"filesystem:write",
			"network:read",
			"network:write",
			"process:exec",
			"env:read",
		},
	}

	errors := ValidateManifest(manifest)
	// Should have no permission-related errors
	for _, err := range errors {
		assert.NotEqual(t, "permissions", err.Field)
	}
}

// TestManifestValidation_MultipleErrors tests that multiple validation errors are returned
func TestManifestValidation_MultipleErrors(t *testing.T) {
	manifest := &Manifest{
		// Missing ID, Name, Version, APIVersion, Type
		SecurityLevel: SecurityLevel("invalid"),
		Permissions:   []string{"bad:permission"},
	}

	errors := ValidateManifest(manifest)
	assert.GreaterOrEqual(t, len(errors), 5, "Should have at least 5 validation errors")

	// Check that we have errors for all expected fields
	errorFields := make(map[string]bool)
	for _, err := range errors {
		errorFields[err.Field] = true
	}

	assert.True(t, errorFields["id"], "Should have ID error")
	assert.True(t, errorFields["name"], "Should have name error")
	assert.True(t, errorFields["version"], "Should have version error")
	assert.True(t, errorFields["api_version"], "Should have api_version error")
	assert.True(t, errorFields["type"], "Should have type error")
}

// TestIsValidSemver tests the semver validation function
func TestIsValidSemver(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		// Valid versions
		{"1.0.0", true},
		{"v1.0.0", true},
		{"2.3.4", true},
		{"10.20.30", true},
		{"1.0.0-alpha", true},
		{"1.0.0-alpha.1", true},
		{"1.0.0-0.3.7", true},
		{"1.0.0+20130313144700", true},
		{"1.0.0-beta+exp.sha.5114f85", true},
		{"v1.2.3-rc.1+build.123", true},

		// Invalid versions
		{"", false},
		{"1", false},
		{"1.0", false},
		{"1.0.", false},
		{"1.0.0.", false},
		{"a.b.c", false},
		{"1.0.0.0", false},
		{"1.0.0-", false},
		{"1.0.0+", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := isValidSemver(tt.version)
			assert.Equal(t, tt.valid, result, "Version %s should be valid=%v", tt.version, tt.valid)
		})
	}
}

// TestIsCompatibleAPIVersion tests API version compatibility checking
func TestIsCompatibleAPIVersion(t *testing.T) {
	tests := []struct {
		name             string
		pluginAPIVersion string
		sdkAPIVersion    string
		compatible       bool
	}{
		// Compatible versions (same major version)
		{"same version", "1.0.0", "1.0.0", true},
		{"different minor", "1.2.0", "1.3.0", true},
		{"different patch", "1.2.3", "1.2.9", true},
		{"with v prefix", "v1.0.0", "v1.9.9", true},
		{"mixed prefix", "v1.0.0", "1.0.0", true},
		{"major v2", "2.0.0", "2.5.1", true},

		// Incompatible versions (different major version)
		{"different major v1 v2", "1.0.0", "2.0.0", false},
		{"different major v2 v1", "2.0.0", "1.0.0", false},
		{"different major v1 v3", "1.0.0", "3.0.0", false},
		{"major v0 v1", "0.9.0", "1.0.0", false},

		// Edge cases
		{"both invalid", "invalid", "invalid", true}, // Both return "0", so compatible
		{"plugin invalid", "invalid", "1.0.0", false},
		{"sdk invalid", "1.0.0", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCompatibleAPIVersion(tt.pluginAPIVersion, tt.sdkAPIVersion)
			assert.Equal(t, tt.compatible, result,
				"API versions %s and %s compatibility should be %v",
				tt.pluginAPIVersion, tt.sdkAPIVersion, tt.compatible)
		})
	}
}

// TestExtractMajorVersion tests major version extraction
func TestExtractMajorVersion(t *testing.T) {
	tests := []struct {
		version       string
		expectedMajor string
	}{
		{"1.0.0", "1"},
		{"v1.0.0", "1"},
		{"2.3.4", "2"},
		{"10.20.30", "10"},
		{"0.1.0", "0"},
		{"1.0.0-alpha", "1"},
		{"v2.0.0+build", "2"},
		{"invalid", "0"},
		{"", "0"},
		{"1", "0"},
		{"1.0", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := extractMajorVersion(tt.version)
			assert.Equal(t, tt.expectedMajor, result,
				"Major version of %s should be %s", tt.version, tt.expectedMajor)
		})
	}
}

// TestLoadManifest_ComplexMetadata tests loading manifest with complex metadata
func TestLoadManifest_ComplexMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "plugin.yaml")

	manifest := &Manifest{
		ID:         "complex-plugin",
		Name:       "Complex Plugin",
		Version:    "1.0.0",
		APIVersion: "1.0.0",
		Type:       PluginTypeGenerator,
		Metadata: map[string]string{
			"language":      "go",
			"platform":      "darwin",
			"arch":          "arm64",
			"binary_name":   "plugin-binary",
			"config_schema": "schema.json",
		},
	}

	err := SaveManifest(manifest, manifestPath)
	require.NoError(t, err)

	loaded, err := LoadManifest(manifestPath)
	assert.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, 5, len(loaded.Metadata))
	assert.Equal(t, "go", loaded.Metadata["language"])
	assert.Equal(t, "darwin", loaded.Metadata["platform"])
	assert.Equal(t, "arm64", loaded.Metadata["arch"])
}

// TestLoadManifest_AllPluginTypes tests loading manifests with all plugin types
func TestLoadManifest_AllPluginTypes(t *testing.T) {
	types := []PluginType{
		PluginTypeLanguage,
		PluginTypeValidator,
		PluginTypeGenerator,
		PluginTypeRunner,
		PluginTypeTransform,
	}

	for _, pluginType := range types {
		t.Run(string(pluginType), func(t *testing.T) {
			tmpDir := t.TempDir()
			manifestPath := filepath.Join(tmpDir, "plugin.yaml")

			manifest := &Manifest{
				ID:         "test-" + string(pluginType),
				Name:       "Test " + string(pluginType),
				Version:    "1.0.0",
				APIVersion: "1.0.0",
				Type:       pluginType,
			}

			err := SaveManifest(manifest, manifestPath)
			require.NoError(t, err)

			loaded, err := LoadManifest(manifestPath)
			assert.NoError(t, err)
			assert.Equal(t, pluginType, loaded.Type)
		})
	}
}

// TestManifestValidation_EmptyPermissions tests that empty permissions list is valid
func TestManifestValidation_EmptyPermissions(t *testing.T) {
	manifest := &Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Permissions: []string{}, // Empty is valid
	}

	errors := ValidateManifest(manifest)
	// Should have no permission-related errors
	for _, err := range errors {
		assert.NotEqual(t, "permissions", err.Field)
	}
}

// TestManifestValidation_NilPermissions tests that nil permissions list is valid
func TestManifestValidation_NilPermissions(t *testing.T) {
	manifest := &Manifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0.0",
		Type:        PluginTypeLanguage,
		Permissions: nil, // Nil is valid
	}

	errors := ValidateManifest(manifest)
	// Should have no permission-related errors
	for _, err := range errors {
		assert.NotEqual(t, "permissions", err.Field)
	}
}
