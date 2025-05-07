package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api"
)

func newPushCommand() *Command {
	cmd := &Command{
		Name:        "push",
		Description: "Push protobuf files to the registry",
		Flags:       flag.NewFlagSet("push", flag.ExitOnError),
		Run:         runPush,
	}

	cmd.Flags.String("module", "", "Module name")
	cmd.Flags.String("version", "", "Version (semantic version or commit hash)")
	cmd.Flags.String("dir", ".", "Directory containing protobuf files")
	cmd.Flags.String("registry", "http://localhost:8080", "Registry URL")
	cmd.Flags.String("description", "", "Module description")

	return cmd
}

// parseProtoImports extracts import statements from a proto file
func parseProtoImports(content string) []string {
	imports := make([]string, 0)
	importRegex := regexp.MustCompile(`import\s+"([^"]+)"`)
	matches := importRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, match[1])
		}
	}
	return imports
}

// extractModuleFromImport extracts the module name from an import path
func extractModuleFromImport(importPath string) string {
	parts := strings.Split(importPath, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// getGitInfo attempts to get git information from the current directory
func getGitInfo(dir string) (api.SourceInfo, error) {
	info := api.SourceInfo{
		Repository: "unknown",
		CommitSHA:  "unknown",
		Branch:     "unknown",
	}

	// Check if directory is a git repository
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		return info, nil
	}

	// Get repository URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = dir
	if output, err := cmd.Output(); err == nil {
		repo := strings.TrimSpace(string(output))
		// Convert git@github.com:user/repo.git to https://github.com/user/repo
		if strings.HasPrefix(repo, "git@") {
			// Remove git@ prefix
			repo = strings.TrimPrefix(repo, "git@")
			// Replace : with / for the path separator
			repo = strings.Replace(repo, ":", "/", 1)
			// Add https:// prefix
			repo = "https://" + repo
		}
		// Remove .git suffix if present
		repo = strings.TrimSuffix(repo, ".git")
		info.Repository = repo
	}

	// Get current commit SHA
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	if output, err := cmd.Output(); err == nil {
		info.CommitSHA = strings.TrimSpace(string(output))
	}

	// Get current branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	if output, err := cmd.Output(); err == nil {
		info.Branch = strings.TrimSpace(string(output))
	}

	return info, nil
}

func runPush(args []string) error {
	cmd := newPushCommand()
	if err := cmd.Flags.Parse(args); err != nil {
		return err
	}

	module := cmd.Flags.Lookup("module").Value.String()
	version := cmd.Flags.Lookup("version").Value.String()
	dir := cmd.Flags.Lookup("dir").Value.String()
	registry := cmd.Flags.Lookup("registry").Value.String()
	description := cmd.Flags.Lookup("description").Value.String()

	if module == "" || version == "" {
		return fmt.Errorf("module and version are required")
	}

	// Create module if it doesn't exist
	moduleURL := fmt.Sprintf("%s/modules", registry)
	moduleData := api.Module{
		Name:        module,
		Description: description,
	}

	moduleJSON, err := json.Marshal(moduleData)
	if err != nil {
		return fmt.Errorf("failed to marshal module: %w", err)
	}

	resp, err := http.Post(moduleURL, "application/json", strings.NewReader(string(moduleJSON)))
	if err != nil {
		return fmt.Errorf("failed to create module: %w", err)
	}
	resp.Body.Close()

	// Collect proto files and dependencies
	var files []api.File
	dependencies := make(map[string]string) // module -> version

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".proto") {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}

			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}

			files = append(files, api.File{
				Path:    relPath,
				Content: string(content),
			})

			// Parse imports and add dependencies
			imports := parseProtoImports(string(content))
			for _, imp := range imports {
				if depModule := extractModuleFromImport(imp); depModule != "" && depModule != module {
					// For now, we'll use v1.0.0 as the default version for dependencies
					// In a real implementation, you might want to get this from a config file or command line
					dependencies[depModule] = "v1.0.0"
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to collect proto files: %w", err)
	}

	// Convert dependencies map to slice
	var deps []string
	for depModule, depVersion := range dependencies {
		deps = append(deps, fmt.Sprintf("%s@%s", depModule, depVersion))
	}

	// Get git information if available
	sourceInfo, err := getGitInfo(dir)
	if err != nil {
		// Log the error but continue with default values
		fmt.Printf("Warning: Failed to get git information: %v\n", err)
	}

	// Create version
	versionURL := fmt.Sprintf("%s/modules/%s/versions", registry, module)
	versionData := api.Version{
		ModuleName:    module,
		Version:       version,
		Files:         files,
		Dependencies:  deps,
		SourceInfo:    sourceInfo,
	}

	versionJSON, err := json.Marshal(versionData)
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	resp, err = http.Post(versionURL, "application/json", strings.NewReader(string(versionJSON)))
	if err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}
	resp.Body.Close()

	fmt.Printf("Successfully pushed module %s version %s\n", module, version)
	return nil
} 