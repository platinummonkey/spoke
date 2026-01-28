package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
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
func parseProtoImports(content string) []protobuf.ProtoImport {
	imports, err := protobuf.ExtractImports(content)
	if err != nil {
		fmt.Printf("Warning: Failed to parse proto imports: %v\n", err)
		return []protobuf.ProtoImport{}
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
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Try running git rev-parse to find git repository
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = dir
		if _, err := cmd.Output(); err != nil {
			// Not a git repository - return default values
			fmt.Printf("Warning: Failed to get git information: %v\n", err)
			return info, nil
		}
		// Found git repository - continue with git commands
	}

	// Get repository URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = dir
	if output, err := cmd.Output(); err == nil {
		repo := strings.TrimSpace(string(output))
		// Convert git@github.com:user/repo.git to https://github.com/user/repo
		if strings.HasPrefix(repo, "git@") {
			// Parse git@github.com:user/repo.git format
			sshParts := strings.SplitN(repo, ":", 2)
			if len(sshParts) == 2 {
				host := strings.TrimPrefix(sshParts[0], "git@")
				path := sshParts[1]
				repo = fmt.Sprintf("https://%s/%s", host, path)
			}
		}
		// Remove .git suffix if present
		repo = strings.TrimSuffix(repo, ".git")
		info.Repository = repo
	} else {
		fmt.Printf("Warning: Failed to get repository URL: %v\n", err)
	}

	// Get current commit SHA
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	if output, err := cmd.Output(); err == nil {
		info.CommitSHA = strings.TrimSpace(string(output))
	} else {
		fmt.Printf("Warning: Failed to get commit SHA: %v\n", err)
	}

	// Get current branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	if output, err := cmd.Output(); err == nil {
		branch := strings.TrimSpace(string(output))
		// If we're in a detached HEAD state, try to get branch from env vars (CI systems often set this)
		if branch == "HEAD" {
			// Try CI environment variables
			if githubRef := os.Getenv("GITHUB_REF"); githubRef != "" {
				branch = strings.TrimPrefix(githubRef, "refs/heads/")
			} else if branchName := os.Getenv("CI_COMMIT_REF_NAME"); branchName != "" {
				branch = branchName
			}
		}
		info.Branch = branch
	} else {
		fmt.Printf("Warning: Failed to get branch name: %v\n", err)
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
			content, err := os.ReadFile(path)
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

			// Parse imports and add dependencies using the new parser
			imports := parseProtoImports(string(content))
			for _, imp := range imports {
				if imp.Module != "" && imp.Module != module {
					dependencies[imp.Module] = imp.Version
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
		fmt.Printf("Warning: Failed to get git information: %v\nUsing default values for source info.\n", err)
		sourceInfo = api.SourceInfo{
			Repository: "unknown",
			CommitSHA:  "unknown",
			Branch:     "unknown",
		}
	} else {
		fmt.Printf("Source info found:\n - Repository: %s\n - Branch: %s\n - Commit: %s\n", 
			sourceInfo.Repository, sourceInfo.Branch, sourceInfo.CommitSHA)
	}

	// Create version
	versionURL := fmt.Sprintf("%s/modules/%s/versions", registry, module)
	versionData := api.Version{
		Version:      version,
		Files:        files,
		Dependencies: deps,
		SourceInfo:   sourceInfo,
	}

	versionJSON, err := json.Marshal(versionData)
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	resp, err = http.Post(versionURL, "application/json", strings.NewReader(string(versionJSON)))
	if err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create version: %s", string(body))
	}

	fmt.Printf("Successfully pushed %d files to module %s version %s\n", len(files), module, version)
	return nil
} 