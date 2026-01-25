package linter

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the linting configuration
type Config struct {
	Version string       `yaml:"version"`
	Lint    LintRules    `yaml:"lint"`
	Quality QualityConfig `yaml:"quality"`
	AutoFix AutoFixConfig `yaml:"autofix"`
}

// LintRules contains rule configuration
type LintRules struct {
	Use        []string               `yaml:"use"` // Style guides: "google", "uber"
	Rules      map[string]interface{} `yaml:"rules"`
	Ignore     []string               `yaml:"ignore"`
	Files      map[string]FileRules   `yaml:"files"`
	Categories map[string]string      `yaml:"categories"` // category -> severity
}

// FileRules contains per-file rule overrides
type FileRules struct {
	Rules map[string]bool `yaml:"rules"`
}

// QualityConfig configures quality metrics
type QualityConfig struct {
	Enabled               bool                      `yaml:"enabled"`
	DocumentationCoverage DocumentationCoverageConfig `yaml:"documentation_coverage"`
	Complexity            ComplexityConfig          `yaml:"complexity"`
	Maintainability       MaintainabilityConfig     `yaml:"maintainability"`
}

// DocumentationCoverageConfig for documentation metrics
type DocumentationCoverageConfig struct {
	MinCoverage float64 `yaml:"min_coverage"`
	Weight      float64 `yaml:"weight"`
}

// ComplexityConfig for complexity metrics
type ComplexityConfig struct {
	MaxMessageDepth int     `yaml:"max_message_depth"`
	MaxFieldCount   int     `yaml:"max_field_count"`
	Weight          float64 `yaml:"weight"`
}

// MaintainabilityConfig for maintainability metrics
type MaintainabilityConfig struct {
	MaxFileLines int     `yaml:"max_file_lines"`
	Weight       float64 `yaml:"weight"`
}

// AutoFixConfig configures automatic fixing
type AutoFixConfig struct {
	Enabled bool            `yaml:"enabled"`
	Rules   map[string]bool `yaml:"rules"`
}

// DefaultConfig returns default linting configuration
func DefaultConfig() *Config {
	return &Config{
		Version: "v1",
		Lint: LintRules{
			Use:        []string{"google"},
			Rules:      make(map[string]interface{}),
			Ignore:     []string{"vendor/**", "third_party/**"},
			Files:      make(map[string]FileRules),
			Categories: make(map[string]string),
		},
		Quality: QualityConfig{
			Enabled: true,
			DocumentationCoverage: DocumentationCoverageConfig{
				MinCoverage: 80.0,
				Weight:      0.3,
			},
			Complexity: ComplexityConfig{
				MaxMessageDepth: 5,
				MaxFieldCount:   50,
				Weight:          0.3,
			},
			Maintainability: MaintainabilityConfig{
				MaxFileLines: 500,
				Weight:       0.4,
			},
		},
		AutoFix: AutoFixConfig{
			Enabled: false,
			Rules:   make(map[string]bool),
		},
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadConfigFromDir searches for config file in directory
func LoadConfigFromDir(dir string) (*Config, error) {
	configNames := []string{"spoke-lint.yaml", "spoke-lint.yml", ".spoke-lint.yaml", ".spoke-lint.yml"}

	for _, name := range configNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return LoadConfig(path)
		}
	}

	// Return default if no config found
	return DefaultConfig(), nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
