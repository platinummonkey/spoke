package linter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if config.Version != "v1" {
		t.Errorf("Expected version v1, got %s", config.Version)
	}

	if len(config.Lint.Use) != 1 || config.Lint.Use[0] != "google" {
		t.Errorf("Expected use to contain 'google', got %v", config.Lint.Use)
	}

	if len(config.Lint.Ignore) != 2 {
		t.Errorf("Expected 2 ignore patterns, got %d", len(config.Lint.Ignore))
	}

	if !config.Quality.Enabled {
		t.Error("Expected quality to be enabled")
	}

	if config.Quality.DocumentationCoverage.MinCoverage != 80.0 {
		t.Errorf("Expected min coverage 80.0, got %f", config.Quality.DocumentationCoverage.MinCoverage)
	}

	if config.Quality.Complexity.MaxMessageDepth != 5 {
		t.Errorf("Expected max message depth 5, got %d", config.Quality.Complexity.MaxMessageDepth)
	}

	if config.Quality.Maintainability.MaxFileLines != 500 {
		t.Errorf("Expected max file lines 500, got %d", config.Quality.Maintainability.MaxFileLines)
	}

	if config.AutoFix.Enabled {
		t.Error("Expected autofix to be disabled")
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `version: v1
lint:
  use:
    - google
    - uber
  rules:
    naming: true
  ignore:
    - vendor/**
  categories:
    style: warning
quality:
  enabled: true
  documentation_coverage:
    min_coverage: 90.0
    weight: 0.4
  complexity:
    max_message_depth: 10
    max_field_count: 100
    weight: 0.3
  maintainability:
    max_file_lines: 1000
    weight: 0.3
autofix:
  enabled: true
  rules:
    formatting: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.Version != "v1" {
		t.Errorf("Expected version v1, got %s", config.Version)
	}

	if len(config.Lint.Use) != 2 {
		t.Errorf("Expected 2 style guides, got %d", len(config.Lint.Use))
	}

	if config.Quality.DocumentationCoverage.MinCoverage != 90.0 {
		t.Errorf("Expected min coverage 90.0, got %f", config.Quality.DocumentationCoverage.MinCoverage)
	}

	if !config.AutoFix.Enabled {
		t.Error("Expected autofix to be enabled")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `version: v1
lint:
  use: [invalid yaml content
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestLoadConfigFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with spoke-lint.yaml
	configPath := filepath.Join(tmpDir, "spoke-lint.yaml")
	configContent := `version: v2
lint:
  use:
    - uber
quality:
  enabled: false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir() failed: %v", err)
	}

	if config.Version != "v2" {
		t.Errorf("Expected version v2, got %s", config.Version)
	}

	if config.Quality.Enabled {
		t.Error("Expected quality to be disabled")
	}
}

func TestLoadConfigFromDir_AlternativeNames(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"spoke-lint.yaml", "spoke-lint.yaml"},
		{"spoke-lint.yml", "spoke-lint.yml"},
		{".spoke-lint.yaml", ".spoke-lint.yaml"},
		{".spoke-lint.yml", ".spoke-lint.yml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, tt.filename)

			configContent := `version: test-version`
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			config, err := LoadConfigFromDir(tmpDir)
			if err != nil {
				t.Fatalf("LoadConfigFromDir() failed: %v", err)
			}

			if config.Version != "test-version" {
				t.Errorf("Expected version 'test-version', got %s", config.Version)
			}
		})
	}
}

func TestLoadConfigFromDir_NoConfigReturnsDefault(t *testing.T) {
	tmpDir := t.TempDir()

	config, err := LoadConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfigFromDir() failed: %v", err)
	}

	// Should return default config
	if config.Version != "v1" {
		t.Errorf("Expected default version v1, got %s", config.Version)
	}

	if !config.Quality.Enabled {
		t.Error("Expected default quality to be enabled")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "saved-config.yaml")

	config := DefaultConfig()
	config.Version = "v2"
	config.Quality.DocumentationCoverage.MinCoverage = 95.0

	err := SaveConfig(config, configPath)
	if err != nil {
		t.Fatalf("SaveConfig() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load it back and verify
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Version != "v2" {
		t.Errorf("Expected version v2, got %s", loadedConfig.Version)
	}

	if loadedConfig.Quality.DocumentationCoverage.MinCoverage != 95.0 {
		t.Errorf("Expected min coverage 95.0, got %f", loadedConfig.Quality.DocumentationCoverage.MinCoverage)
	}
}

func TestSaveConfig_InvalidPath(t *testing.T) {
	config := DefaultConfig()

	// Try to save to an invalid path
	err := SaveConfig(config, "/nonexistent/directory/config.yaml")
	if err == nil {
		t.Error("Expected error when saving to invalid path, got nil")
	}
}
