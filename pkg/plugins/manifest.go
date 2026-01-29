package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

var semverRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`)

// LoadManifest loads and parses a plugin manifest from a file
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// LoadManifestFromDir loads a plugin manifest from a directory (looks for plugin.yaml)
func LoadManifestFromDir(dir string) (*Manifest, error) {
	manifestPath := filepath.Join(dir, "plugin.yaml")
	return LoadManifest(manifestPath)
}

// SaveManifest saves a plugin manifest to a file
func SaveManifest(manifest *Manifest, path string) error {
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// ValidateManifest performs basic validation on a plugin manifest
func ValidateManifest(manifest *Manifest) []ValidationError {
	var errors []ValidationError

	// Required fields
	if manifest.ID == "" {
		errors = append(errors, ValidationError{
			Field:   "id",
			Message: "Plugin ID is required",
		})
	}

	if manifest.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Plugin name is required",
		})
	}

	if manifest.Version == "" {
		errors = append(errors, ValidationError{
			Field:   "version",
			Message: "Version is required",
		})
	}

	if manifest.APIVersion == "" {
		errors = append(errors, ValidationError{
			Field:   "api_version",
			Message: "API version is required",
		})
	}

	if manifest.Type == "" {
		errors = append(errors, ValidationError{
			Field:   "type",
			Message: "Plugin type is required",
		})
	}

	// Validate semver format
	if manifest.Version != "" && !isValidSemver(manifest.Version) {
		errors = append(errors, ValidationError{
			Field:   "version",
			Message: fmt.Sprintf("Invalid semver format: %s", manifest.Version),
		})
	}

	if manifest.APIVersion != "" && !isValidSemver(manifest.APIVersion) {
		errors = append(errors, ValidationError{
			Field:   "api_version",
			Message: fmt.Sprintf("Invalid semver format: %s", manifest.APIVersion),
		})
	}

	// Validate plugin type
	if manifest.Type != PluginTypeLanguage {
		errors = append(errors, ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("Invalid plugin type: %s (only 'language' is supported)", manifest.Type),
		})
	}

	return errors
}

// isValidSemver checks if a version string follows semantic versioning
func isValidSemver(version string) bool {
	return semverRegex.MatchString(version)
}

// IsCompatibleAPIVersion checks if a plugin's API version is compatible with the current SDK
func IsCompatibleAPIVersion(pluginAPIVersion, sdkAPIVersion string) bool {
	// For now, we only check major version compatibility
	// v1.x.x is compatible with v1.y.z
	pluginMajor := extractMajorVersion(pluginAPIVersion)
	sdkMajor := extractMajorVersion(sdkAPIVersion)

	return pluginMajor == sdkMajor
}

func extractMajorVersion(version string) string {
	matches := semverRegex.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1]
	}
	return "0"
}
