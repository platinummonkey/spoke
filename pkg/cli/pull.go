package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api"
)

func newPullCommand() *Command {
	cmd := &Command{
		Name:        "pull",
		Description: "Pull protobuf files from the registry",
		Flags:       flag.NewFlagSet("pull", flag.ExitOnError),
		Run:         runPull,
	}

	cmd.Flags.String("module", "", "Module name")
	cmd.Flags.String("version", "", "Version (semantic version or commit hash)")
	cmd.Flags.String("dir", ".", "Directory to save protobuf files")
	cmd.Flags.String("registry", "http://localhost:8080", "Registry URL")
	cmd.Flags.Bool("recursive", false, "Pull dependencies recursively")

	return cmd
}

func runPull(args []string) error {
	cmd := newPullCommand()
	if err := cmd.Flags.Parse(args); err != nil {
		return err
	}

	module := cmd.Flags.Lookup("module").Value.String()
	version := cmd.Flags.Lookup("version").Value.String()
	dir := cmd.Flags.Lookup("dir").Value.String()
	registry := cmd.Flags.Lookup("registry").Value.String()
	recursive := cmd.Flags.Lookup("recursive").Value.String() == "true"

	if module == "" || version == "" {
		return fmt.Errorf("module and version are required")
	}

	// Create base directory for the module
	baseDir := filepath.Join(dir, module)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Get version
	versionURL := fmt.Sprintf("%s/modules/%s/versions/%s", registry, module, version)
	resp, err := http.Get(versionURL)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("version not found")
	}

	var versionData api.Version
	if err := json.NewDecoder(resp.Body).Decode(&versionData); err != nil {
		return fmt.Errorf("failed to decode version: %w", err)
	}

	// Save files
	for _, file := range versionData.Files {
		filePath := filepath.Join(baseDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", file.Path, err)
		}

		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.Path, err)
		}
	}

	// Pull dependencies recursively if requested
	if recursive {
		for _, dep := range versionData.Dependencies {
			parts := strings.Split(dep, "@")
			if len(parts) != 2 {
				return fmt.Errorf("invalid dependency format: %s", dep)
			}

			depModule := parts[0]
			depVersion := parts[1]

			// Create dependency directory structure that matches import paths
			depDir := filepath.Join(dir, depModule)
			if err := os.MkdirAll(depDir, 0755); err != nil {
				return fmt.Errorf("failed to create dependency directory: %w", err)
			}

			// Recursively pull dependency
			depArgs := []string{
				"-module", depModule,
				"-version", depVersion,
				"-dir", dir, // Use the same base directory
				"-registry", registry,
				"-recursive",
			}
			if err := runPull(depArgs); err != nil {
				return fmt.Errorf("failed to pull dependency %s: %w", dep, err)
			}
		}
	}

	fmt.Printf("Successfully pulled module %s version %s\n", module, version)
	return nil
} 