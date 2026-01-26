package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// BasicLanguagePlugin is a simple file-based language plugin implementation
// This is used when native Go plugins are not available or for YAML-only plugins
type BasicLanguagePlugin struct {
	manifest     *Manifest
	pluginDir    string
	languageSpec *LanguageSpec
}

// NewBasicLanguagePlugin creates a new basic language plugin
func NewBasicLanguagePlugin(manifest *Manifest, pluginDir string) *BasicLanguagePlugin {
	return &BasicLanguagePlugin{
		manifest:  manifest,
		pluginDir: pluginDir,
	}
}

// Manifest returns the plugin manifest
func (p *BasicLanguagePlugin) Manifest() *Manifest {
	return p.manifest
}

// Load initializes the plugin
func (p *BasicLanguagePlugin) Load() error {
	// Load language spec from manifest or separate file
	if err := p.loadLanguageSpec(); err != nil {
		return fmt.Errorf("failed to load language spec: %w", err)
	}

	return nil
}

// Unload cleans up plugin resources
func (p *BasicLanguagePlugin) Unload() error {
	// No resources to clean up for basic plugin
	return nil
}

// GetLanguageSpec returns the language specification
func (p *BasicLanguagePlugin) GetLanguageSpec() *LanguageSpec {
	return p.languageSpec
}

// BuildProtocCommand builds a protoc command for this language
func (p *BasicLanguagePlugin) BuildProtocCommand(ctx context.Context, req *CommandRequest) ([]string, error) {
	if p.languageSpec == nil {
		return nil, fmt.Errorf("language spec not loaded")
	}

	cmd := []string{"protoc"}

	// Add import paths
	for _, path := range req.ImportPaths {
		cmd = append(cmd, "--proto_path="+path)
	}

	// Add plugin path if specified
	if req.PluginPath != "" {
		pluginName := p.languageSpec.ProtocPlugin
		if pluginName == "" {
			pluginName = p.languageSpec.ID
		}
		cmd = append(cmd, fmt.Sprintf("--plugin=protoc-gen-%s=%s", pluginName, req.PluginPath))
	}

	// Add output flag
	outputFlag := fmt.Sprintf("--%s_out=%s", p.languageSpec.ID, req.OutputDir)
	cmd = append(cmd, outputFlag)

	// Add custom options if provided
	if len(req.Options) > 0 {
		var opts []string
		for key, value := range req.Options {
			if value == "" {
				opts = append(opts, key)
			} else {
				opts = append(opts, fmt.Sprintf("%s=%s", key, value))
			}
		}
		if len(opts) > 0 {
			optFlag := fmt.Sprintf("--%s_opt=%s", p.languageSpec.ID, strings.Join(opts, ","))
			cmd = append(cmd, optFlag)
		}
	}

	// Add proto files
	cmd = append(cmd, req.ProtoFiles...)

	return cmd, nil
}

// ValidateOutput validates that the expected output files were generated
func (p *BasicLanguagePlugin) ValidateOutput(ctx context.Context, files []string) error {
	if p.languageSpec == nil {
		return fmt.Errorf("language spec not loaded")
	}

	if len(files) == 0 {
		return fmt.Errorf("no files generated")
	}

	// Check that files have expected extensions
	for _, file := range files {
		hasValidExt := false
		for _, ext := range p.languageSpec.FileExtensions {
			if strings.HasSuffix(file, ext) {
				hasValidExt = true
				break
			}
		}

		if !hasValidExt {
			return fmt.Errorf("unexpected file extension: %s (expected: %v)",
				file, p.languageSpec.FileExtensions)
		}
	}

	return nil
}

// loadLanguageSpec loads the language specification from the manifest or a separate file
func (p *BasicLanguagePlugin) loadLanguageSpec() error {
	// Try to load from a separate language_spec.yaml file first
	specPath := filepath.Join(p.pluginDir, "language_spec.yaml")
	if _, err := os.Stat(specPath); err == nil {
		data, err := os.ReadFile(specPath)
		if err != nil {
			return fmt.Errorf("failed to read language spec file: %w", err)
		}

		var spec LanguageSpec
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return fmt.Errorf("failed to parse language spec: %w", err)
		}

		p.languageSpec = &spec
		return nil
	}

	// Otherwise, try to extract from manifest metadata
	if specData, ok := p.manifest.Metadata["language_spec"]; ok {
		var spec LanguageSpec
		if err := yaml.Unmarshal([]byte(specData), &spec); err != nil {
			return fmt.Errorf("failed to parse language spec from manifest: %w", err)
		}

		p.languageSpec = &spec
		return nil
	}

	// As a last resort, create a basic spec from manifest
	p.languageSpec = &LanguageSpec{
		ID:          p.manifest.ID,
		Name:        p.manifest.Name,
		DisplayName: p.manifest.Name,
		Enabled:     true,
		Stable:      p.manifest.SecurityLevel == SecurityLevelOfficial || p.manifest.SecurityLevel == SecurityLevelVerified,
		Description: p.manifest.Description,
	}

	return nil
}
