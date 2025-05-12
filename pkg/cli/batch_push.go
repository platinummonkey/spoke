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
	"time"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

func newBatchPushCommand() *Command {
	cmd := &Command{
		Name:        "batch-push",
		Description: "Recursively scan directories for proto files and push them to the registry",
		Flags:       flag.NewFlagSet("batch-push", flag.ExitOnError),
		Run:         runBatchPush,
	}

	cmd.Flags.String("module", "", "Module name (optional - will be inferred from proto package if not provided)")
	cmd.Flags.String("dir", ".", "Root directory to scan for protobuf files")
	cmd.Flags.String("registry", "http://localhost:8080", "Registry URL")
	cmd.Flags.String("description", "", "Module description")
	cmd.Flags.String("exclude", "", "Comma-separated list of directory patterns to exclude")

	return cmd
}

// generateVersionName creates a version name using the format: 
// ${current_git_branch}-${short-timestamp-in-UTC-YYYY-MM-DD-HH-mm}-${short git sha}
func generateVersionName(dir string) (string, error) {
	// Get git information
	info, err := getGitInfo(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get git information: %w", err)
	}

	// Get current timestamp in UTC
	now := time.Now().UTC()
	timestamp := now.Format("2006-01-02-15-04")

	// Get short git SHA (first 7 characters)
	shortSHA := info.CommitSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	// Generate version name
	version := fmt.Sprintf("%s-%s-%s", info.Branch, timestamp, shortSHA)

	// Replace any invalid characters
	version = strings.ReplaceAll(version, "/", "-")
	version = strings.ReplaceAll(version, "\\", "-")
	version = strings.ReplaceAll(version, ":", "-")

	return version, nil
}

// isExcluded checks if a path should be excluded based on patterns
func isExcluded(path string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if pattern == "" {
			continue
		}
		
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
		
		// Also check parent directories
		dirs := strings.Split(path, string(filepath.Separator))
		for _, dir := range dirs {
			matched, err := filepath.Match(pattern, dir)
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}

// extractPackageName gets the package name from a proto file content
func extractPackageName(content string) string {
	packageName, err := protobuf.ExtractPackageName(content)
	if err != nil {
		fmt.Printf("Warning: Failed to extract package name: %v\n", err)
		return ""
	}
	return packageName
}

// getFileGitInfo gets git information specific to a file
func getFileGitInfo(filePath string) (api.SourceInfo, error) {
	info := api.SourceInfo{
		Repository: "unknown",
		CommitSHA:  "unknown",
		Branch:     "unknown",
	}

	// Get the directory containing the file
	dir := filepath.Dir(filePath)

	// Check if this directory is within a git repository
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return info, fmt.Errorf("not in a git repository: %w", err)
	}

	// Get the git repository root
	repoRoot := strings.TrimSpace(string(output))

	// Get repository URL
	cmd = exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = repoRoot
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
	}

	// Get the last commit that touched this file
	cmd = exec.Command("git", "log", "-n", "1", "--pretty=format:%H", "--", filePath)
	cmd.Dir = repoRoot
	if output, err := cmd.Output(); err == nil {
		info.CommitSHA = strings.TrimSpace(string(output))
	}

	// Get the branch for this file
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoRoot
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
	}

	return info, nil
}

// generateVersionNameFromFileInfo creates a version name for a specific file's git info
func generateVersionNameFromFileInfo(filePath string) (string, error) {
	// Get git information specific to this file
	info, err := getFileGitInfo(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get git information for %s: %w", filePath, err)
	}

	// Get current timestamp in UTC
	now := time.Now().UTC()
	timestamp := now.Format("2006-01-02-15-04")

	// Get short git SHA (first 7 characters)
	shortSHA := info.CommitSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	// Generate version name
	version := fmt.Sprintf("%s-%s-%s", info.Branch, timestamp, shortSHA)

	// Replace any invalid characters
	version = strings.ReplaceAll(version, "/", "-")
	version = strings.ReplaceAll(version, "\\", "-")
	version = strings.ReplaceAll(version, ":", "-")

	return version, nil
}

func runBatchPush(args []string) error {
	cmd := newBatchPushCommand()
	if err := cmd.Flags.Parse(args); err != nil {
		return err
	}

	dir := cmd.Flags.Lookup("dir").Value.String()
	registry := cmd.Flags.Lookup("registry").Value.String()
	description := cmd.Flags.Lookup("description").Value.String()
	excludeStr := cmd.Flags.Lookup("exclude").Value.String()
	// We'll ignore the module flag and use package names directly

	// Parse exclude patterns
	excludePatterns := []string{}
	if excludeStr != "" {
		excludePatterns = strings.Split(excludeStr, ",")
		for i := range excludePatterns {
			excludePatterns[i] = strings.TrimSpace(excludePatterns[i])
		}
	}

	// Organize proto files by package name
	type ProtoModule struct {
		PackageName  string
		Files        []api.File
		Dependencies map[string]string // module -> version
	}
	packageMap := make(map[string]*ProtoModule)
	protoFiles := 0

	// Store git info for each repo we find
	gitInfoFound := false
	var lastGitInfo api.SourceInfo
	var firstProtoFile string

	fmt.Println("Scanning for proto files...")
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if isExcluded(path, excludePatterns) {
				fmt.Printf("Skipping excluded directory: %s\n", path)
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".proto") {
			return nil
		}

		fmt.Printf("Processing proto file: %s\n", path)

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if isExcluded(relPath, excludePatterns) {
			fmt.Printf("Skipping excluded file: %s\n", relPath)
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Save first proto file for later use
		if firstProtoFile == "" {
			firstProtoFile = path
		}

		// Try to get git info for this file
		if !gitInfoFound {
			if gitInfo, err := getFileGitInfo(path); err == nil {
				lastGitInfo = gitInfo
				gitInfoFound = true
				fmt.Printf("Found git information for %s:\n", path)
				fmt.Printf("  Repository: %s\n", gitInfo.Repository)
				fmt.Printf("  Branch: %s\n", gitInfo.Branch)
				fmt.Printf("  Commit: %s\n", gitInfo.CommitSHA)
			}
		}

		// Extract package name
		packageName := extractPackageName(string(content))
		if packageName == "" {
			fmt.Printf("Warning: No package name found in %s, skipping\n", path)
			return nil
		}

		// Get or create module for this package
		module, exists := packageMap[packageName]
		if !exists {
			module = &ProtoModule{
				PackageName:  packageName,
				Files:        []api.File{},
				Dependencies: make(map[string]string),
			}
			packageMap[packageName] = module
		}

		// Add file to module
		module.Files = append(module.Files, api.File{
			Path:    relPath,
			Content: string(content),
		})
		protoFiles++

		// Parse imports and add dependencies
		imports := parseProtoImports(string(content))
		for _, imp := range imports {
			if imp.Module != "" && imp.Module != packageName {
				fmt.Printf("\tAdding dependency: %s (%s)\n", imp.Module, imp.Version)
				module.Dependencies[imp.Module] = imp.Version
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to collect proto files: %w", err)
	}

	if protoFiles == 0 {
		return fmt.Errorf("no proto files found in directory: %s", dir)
	}

	fmt.Printf("\n\nFound %d proto files in %d package(s)\n", protoFiles, len(packageMap))

	// Generate version name based on first proto file
	var version string
	var sourceInfo api.SourceInfo

	if firstProtoFile != "" && gitInfoFound {
		// Use the git info from the first proto file with valid git info
		version, err = generateVersionNameFromFileInfo(firstProtoFile)
		sourceInfo = lastGitInfo
		if err != nil {
			fmt.Printf("Warning: Failed to generate version name from git: %v\nUsing default timestamp instead.\n", err)
			// Fallback to just timestamp if git info isn't available
			version = time.Now().UTC().Format("2006-01-02-15-04-05")
			sourceInfo = api.SourceInfo{
				Repository: "unknown",
				CommitSHA:  "unknown",
				Branch:     "unknown",
			}
		}
	} else {
		// Fallback if no git info was found
		version = time.Now().UTC().Format("2006-01-02-15-04-05")
		sourceInfo = api.SourceInfo{
			Repository: "unknown",
			CommitSHA:  "unknown",
			Branch:     "unknown",
		}
	}

	fmt.Printf("Using version: %s\n", version)
	fmt.Printf("Source info:\n - Repository: %s\n - Branch: %s\n - Commit: %s\n", 
		sourceInfo.Repository, sourceInfo.Branch, sourceInfo.CommitSHA)

	// Upload each package as a separate module
	for packageName, module := range packageMap {
		fmt.Printf("\nUploading package: %s (%d files)\n", packageName, len(module.Files))
		
		// Create module if it doesn't exist
		moduleURL := fmt.Sprintf("%s/modules", registry)
		moduleData := api.Module{
			Name:        packageName,
			Description: description,
		}

		moduleJSON, err := json.Marshal(moduleData)
		if err != nil {
			return fmt.Errorf("failed to marshal module %s: %w", packageName, err)
		}

		resp, err := http.Post(moduleURL, "application/json", strings.NewReader(string(moduleJSON)))
		if err != nil {
			return fmt.Errorf("failed to create module %s: %w", packageName, err)
		}
		resp.Body.Close()

		// Convert dependencies map to slice
		var deps []string
		for depModule, depVersion := range module.Dependencies {
			deps = append(deps, fmt.Sprintf("%s@%s", depModule, depVersion))
		}

		if len(deps) > 0 {
			fmt.Printf("  Dependencies for %s:\n", packageName)
			for _, dep := range deps {
				fmt.Printf("    - %s\n", dep)
			}
		}

		// Create version
		versionURL := fmt.Sprintf("%s/modules/%s/versions", registry, packageName)
		versionData := api.Version{
			ModuleName:    packageName,
			Version:       version,
			Files:         module.Files,
			Dependencies:  deps,
			SourceInfo:    sourceInfo,
		}

		versionJSON, err := json.Marshal(versionData)
		if err != nil {
			return fmt.Errorf("failed to marshal version for %s: %w", packageName, err)
		}

		resp, err = http.Post(versionURL, "application/json", strings.NewReader(string(versionJSON)))
		if err != nil {
			return fmt.Errorf("failed to create version for %s: %w", packageName, err)
		}

		// Check response status
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("failed to create version for %s: %s: %s", packageName, resp.Status, string(body))
		}
		resp.Body.Close()

		fmt.Printf("  Successfully pushed module %s version %s\n", packageName, version)
		fmt.Printf("  Version URL: %s/modules/%s/versions/%s\n", registry, packageName, version)
	}
	
	return nil
} 