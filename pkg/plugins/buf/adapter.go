package buf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/platinummonkey/spoke/pkg/plugins"
)

// BufPluginAdapter wraps a Buf plugin as a Spoke LanguagePlugin
type BufPluginAdapter struct {
	manifest     *plugins.Manifest
	pluginRef    string // buf.build/library/connect-go
	version      string // v1.5.0
	binaryPath   string // ~/.buf/plugins/connect-go/v1.5.0/protoc-gen-connect-go
	downloader   *Downloader
	languageSpec *plugins.LanguageSpec
	loaded       bool
}

// NewBufPluginAdapter creates a new Buf plugin adapter
func NewBufPluginAdapter(pluginRef, version string) *BufPluginAdapter {
	return &BufPluginAdapter{
		pluginRef:  pluginRef,
		version:    version,
		downloader: NewDownloader(),
	}
}

// NewBufPluginAdapterFromManifest creates a Buf plugin adapter from a manifest
func NewBufPluginAdapterFromManifest(manifest *plugins.Manifest) (*BufPluginAdapter, error) {
	// Extract Buf plugin info from manifest metadata
	bufRegistry, ok := manifest.Metadata["buf_registry"]
	if !ok {
		return nil, fmt.Errorf("manifest missing buf_registry metadata")
	}

	bufVersion, ok := manifest.Metadata["buf_version"]
	if !ok {
		bufVersion = manifest.Version
	}

	adapter := &BufPluginAdapter{
		manifest:   manifest,
		pluginRef:  bufRegistry,
		version:    bufVersion,
		downloader: NewDownloader(),
	}

	return adapter, nil
}

// Manifest returns the plugin manifest
func (a *BufPluginAdapter) Manifest() *plugins.Manifest {
	return a.manifest
}

// Load downloads and initializes the Buf plugin
func (a *BufPluginAdapter) Load() error {
	if a.loaded {
		return nil
	}

	// Check if already cached
	if a.isCached() {
		a.binaryPath = a.getCachedPath()
	} else {
		// Download plugin from Buf registry
		binary, err := a.downloader.Download(a.pluginRef, a.version)
		if err != nil {
			return fmt.Errorf("failed to download Buf plugin: %w", err)
		}
		a.binaryPath = binary
	}

	// Verify binary exists and is executable
	if err := a.verifyBinary(); err != nil {
		return fmt.Errorf("binary verification failed: %w", err)
	}

	// Initialize language spec
	a.languageSpec = a.buildLanguageSpec()

	a.loaded = true
	return nil
}

// Unload cleans up plugin resources
func (a *BufPluginAdapter) Unload() error {
	a.loaded = false
	return nil
}

// GetLanguageSpec returns the language specification for this Buf plugin
func (a *BufPluginAdapter) GetLanguageSpec() *plugins.LanguageSpec {
	if a.languageSpec == nil {
		a.languageSpec = a.buildLanguageSpec()
	}
	return a.languageSpec
}

// BuildProtocCommand builds a protoc command using this Buf plugin
func (a *BufPluginAdapter) BuildProtocCommand(ctx context.Context, req *plugins.CommandRequest) ([]string, error) {
	if !a.loaded {
		return nil, fmt.Errorf("plugin not loaded")
	}

	cmd := []string{"protoc"}

	// Add plugin binary path
	pluginName := a.derivePluginName()
	cmd = append(cmd, fmt.Sprintf("--plugin=protoc-gen-%s=%s", pluginName, a.binaryPath))

	// Add import paths
	for _, path := range req.ImportPaths {
		cmd = append(cmd, "--proto_path="+path)
	}

	// Add output directory
	cmd = append(cmd, fmt.Sprintf("--%s_out=%s", pluginName, req.OutputDir))

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
			optFlag := fmt.Sprintf("--%s_opt=%s", pluginName, strings.Join(opts, ","))
			cmd = append(cmd, optFlag)
		}
	}

	// Add proto files
	cmd = append(cmd, req.ProtoFiles...)

	return cmd, nil
}

// ValidateOutput validates that the expected output files were generated
func (a *BufPluginAdapter) ValidateOutput(ctx context.Context, files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files generated")
	}

	// Basic validation - check files exist
	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("expected file not found: %s", file)
		}
	}

	return nil
}

// isCached checks if the plugin is already downloaded and cached
func (a *BufPluginAdapter) isCached() bool {
	cachePath := a.getCachedPath()
	_, err := os.Stat(cachePath)
	return err == nil
}

// getCachedPath returns the expected cache path for this plugin
func (a *BufPluginAdapter) getCachedPath() string {
	homeDir, _ := os.UserHomeDir()
	pluginName := a.derivePluginName()
	return filepath.Join(homeDir, ".buf", "plugins", pluginName, a.version, "protoc-gen-"+pluginName)
}

// deriveLanguageID derives a language ID from the Buf plugin reference
func (a *BufPluginAdapter) deriveLanguageID() string {
	// buf.build/library/connect-go -> connect-go
	parts := strings.Split(a.pluginRef, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

// derivePluginName derives the plugin name for protoc
func (a *BufPluginAdapter) derivePluginName() string {
	// buf.build/library/connect-go -> connect-go
	return a.deriveLanguageID()
}

// buildLanguageSpec builds a LanguageSpec from the Buf plugin
func (a *BufPluginAdapter) buildLanguageSpec() *plugins.LanguageSpec {
	pluginName := a.derivePluginName()
	languageID := a.deriveLanguageID()

	spec := &plugins.LanguageSpec{
		ID:               languageID,
		Name:             pluginName,
		DisplayName:      fmt.Sprintf("%s (Buf Plugin)", pluginName),
		ProtocPlugin:     "protoc-gen-" + pluginName,
		PluginVersion:    a.version,
		DockerImage:      "", // Buf plugins run natively, not in Docker
		SupportsGRPC:     strings.Contains(strings.ToLower(pluginName), "grpc") || strings.Contains(strings.ToLower(pluginName), "connect"),
		FileExtensions:   a.guessFileExtensions(pluginName),
		Enabled:          true,
		Stable:           true,
		Description:      fmt.Sprintf("Buf plugin for %s (from %s)", pluginName, a.pluginRef),
		DocumentationURL: fmt.Sprintf("https://%s", a.pluginRef),
	}

	return spec
}

// guessFileExtensions guesses file extensions based on plugin name
func (a *BufPluginAdapter) guessFileExtensions(pluginName string) []string {
	lower := strings.ToLower(pluginName)

	// Common patterns
	if strings.Contains(lower, "go") {
		return []string{".pb.go", ".go"}
	}
	if strings.Contains(lower, "python") || strings.Contains(lower, "py") {
		return []string{"_pb2.py", ".py"}
	}
	if strings.Contains(lower, "typescript") || strings.Contains(lower, "ts") {
		return []string{".pb.ts", ".ts"}
	}
	if strings.Contains(lower, "javascript") || strings.Contains(lower, "js") {
		return []string{".pb.js", ".js"}
	}
	if strings.Contains(lower, "java") {
		return []string{".java"}
	}
	if strings.Contains(lower, "kotlin") || strings.Contains(lower, "kt") {
		return []string{".kt"}
	}
	if strings.Contains(lower, "swift") {
		return []string{".swift"}
	}
	if strings.Contains(lower, "rust") || strings.Contains(lower, "rs") {
		return []string{".rs"}
	}

	// Default
	return []string{".pb"}
}

// verifyBinary verifies that the plugin binary exists and is executable
func (a *BufPluginAdapter) verifyBinary() error {
	info, err := os.Stat(a.binaryPath)
	if err != nil {
		return fmt.Errorf("binary not found: %w", err)
	}

	// Check if it's executable
	if info.Mode()&0111 == 0 {
		// Not executable, try to make it executable
		if err := os.Chmod(a.binaryPath, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	return nil
}
