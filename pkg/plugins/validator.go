package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Validator performs security validation and scanning on plugins
type Validator struct {
	allowedPermissions map[string]bool
	dangerousImports   []string
	logger             *logrus.Logger
	gosecPath          string // Path to gosec binary
}

// NewValidator creates a new plugin validator
func NewValidator(logger *logrus.Logger) *Validator {
	// Default allowed permissions
	allowedPerms := map[string]bool{
		"filesystem:read":  true,
		"filesystem:write": true,
		"network:read":     true,
		"network:write":    true,
		"process:exec":     true,
		"env:read":         true,
	}

	// Dangerous Go imports that require scrutiny
	dangerousImports := []string{
		"os/exec",      // Command execution
		"syscall",      // Low-level system calls
		"unsafe",       // Unsafe pointer operations
		"plugin",       // Dynamic loading
		"net/http",     // Network access (can be dangerous)
		"database/sql", // Database access
		"crypto/md5",   // Weak cryptography
		"crypto/sha1",  // Weak cryptography
		"math/rand",    // Non-cryptographic random (if used for security)
	}

	return &Validator{
		allowedPermissions: allowedPerms,
		dangerousImports:   dangerousImports,
		logger:             logger,
		gosecPath:          findGosecPath(),
	}
}

// ValidateManifest validates a plugin manifest for correctness and safety
func (v *Validator) ValidateManifest(manifest *Manifest) []ValidationError {
	var errors []ValidationError

	// Required fields
	if manifest.ID == "" {
		errors = append(errors, ValidationError{
			Field:    "id",
			Message:  "Plugin ID is required",
			Severity: "error",
		})
	} else if !isValidPluginID(manifest.ID) {
		errors = append(errors, ValidationError{
			Field:    "id",
			Message:  "Plugin ID must be lowercase alphanumeric with hyphens (e.g., 'rust-language')",
			Severity: "error",
		})
	}

	if manifest.Name == "" {
		errors = append(errors, ValidationError{
			Field:    "name",
			Message:  "Plugin name is required",
			Severity: "error",
		})
	}

	if manifest.Version == "" {
		errors = append(errors, ValidationError{
			Field:    "version",
			Message:  "Version is required",
			Severity: "error",
		})
	} else if !isValidSemver(manifest.Version) {
		errors = append(errors, ValidationError{
			Field:    "version",
			Message:  "Version must be valid semantic version (e.g., '1.0.0')",
			Severity: "error",
		})
	}

	if manifest.APIVersion == "" {
		errors = append(errors, ValidationError{
			Field:    "api_version",
			Message:  "API version is required",
			Severity: "error",
		})
	}

	if manifest.Author == "" {
		errors = append(errors, ValidationError{
			Field:    "author",
			Message:  "Author is required",
			Severity: "warning",
		})
	}

	if manifest.License == "" {
		errors = append(errors, ValidationError{
			Field:    "license",
			Message:  "License should be specified",
			Severity: "warning",
		})
	}

	// Validate plugin type
	validTypes := map[PluginType]bool{
		PluginTypeLanguage:  true,
		PluginTypeValidator: true,
		PluginTypeGenerator: true,
		PluginTypeRunner:    true,
		PluginTypeTransform: true,
	}
	if !validTypes[manifest.Type] {
		errors = append(errors, ValidationError{
			Field:    "type",
			Message:  fmt.Sprintf("Invalid plugin type: %s", manifest.Type),
			Severity: "error",
		})
	}

	// Validate security level
	validSecurityLevels := map[SecurityLevel]bool{
		SecurityLevelOfficial:  true,
		SecurityLevelVerified:  true,
		SecurityLevelCommunity: true,
	}
	if !validSecurityLevels[manifest.SecurityLevel] {
		errors = append(errors, ValidationError{
			Field:    "security_level",
			Message:  fmt.Sprintf("Invalid security level: %s", manifest.SecurityLevel),
			Severity: "error",
		})
	}

	// Validate permissions
	for _, perm := range manifest.Permissions {
		if !v.isAllowedPermission(perm) {
			errors = append(errors, ValidationError{
				Field:    "permissions",
				Message:  fmt.Sprintf("Unknown or dangerous permission: %s", perm),
				Severity: "error",
			})
		}
	}

	// Check for suspicious patterns in metadata
	if manifest.Repository != "" && !isValidURL(manifest.Repository) {
		errors = append(errors, ValidationError{
			Field:    "repository",
			Message:  "Repository URL appears invalid",
			Severity: "warning",
		})
	}

	if manifest.Homepage != "" && !isValidURL(manifest.Homepage) {
		errors = append(errors, ValidationError{
			Field:    "homepage",
			Message:  "Homepage URL appears invalid",
			Severity: "warning",
		})
	}

	return errors
}

// ScanForSecurityIssues performs comprehensive security scanning on plugin code
func (v *Validator) ScanForSecurityIssues(ctx context.Context, pluginPath string) ([]SecurityIssue, error) {
	startTime := time.Now()
	var allIssues []SecurityIssue

	v.logger.Infof("Starting security scan for plugin at %s", pluginPath)

	// 1. Check for dangerous imports
	importIssues, err := v.checkDangerousImports(pluginPath)
	if err != nil {
		v.logger.Warnf("Import check failed: %v", err)
	} else {
		allIssues = append(allIssues, importIssues...)
	}

	// 2. Run gosec security scanner
	if v.gosecPath != "" {
		gosecIssues, err := v.runGosec(ctx, pluginPath)
		if err != nil {
			v.logger.Warnf("Gosec scan failed: %v", err)
		} else {
			allIssues = append(allIssues, gosecIssues...)
		}
	} else {
		v.logger.Warn("Gosec not found, skipping static analysis")
		allIssues = append(allIssues, SecurityIssue{
			Severity:    "warning",
			Category:    "scan-incomplete",
			Description: "Gosec security scanner not available, static analysis incomplete",
		})
	}

	// 3. Check for hardcoded secrets (basic pattern matching)
	secretIssues, err := v.checkHardcodedSecrets(pluginPath)
	if err != nil {
		v.logger.Warnf("Secret check failed: %v", err)
	} else {
		allIssues = append(allIssues, secretIssues...)
	}

	// 4. Check for suspicious file operations
	fileOpIssues, err := v.checkSuspiciousFileOperations(pluginPath)
	if err != nil {
		v.logger.Warnf("File operation check failed: %v", err)
	} else {
		allIssues = append(allIssues, fileOpIssues...)
	}

	v.logger.Infof("Security scan completed in %v, found %d issues", time.Since(startTime), len(allIssues))
	return allIssues, nil
}

// checkDangerousImports scans for potentially dangerous Go imports
func (v *Validator) checkDangerousImports(pluginPath string) ([]SecurityIssue, error) {
	var issues []SecurityIssue
	foundImports := make(map[string][]string) // import -> []files

	err := filepath.Walk(pluginPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, dangerousImport := range v.dangerousImports {
			// Check for both quoted and unquoted imports
			patterns := []string{
				fmt.Sprintf(`import\s+"%s"`, dangerousImport),
				fmt.Sprintf(`import\s+\(\s*[^)]*"%s"[^)]*\)`, dangerousImport),
			}

			for _, pattern := range patterns {
				matched, _ := regexp.Match(pattern, content)
				if matched {
					relPath, _ := filepath.Rel(pluginPath, path)
					foundImports[dangerousImport] = append(foundImports[dangerousImport], relPath)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk plugin directory: %w", err)
	}

	// Generate issues for each dangerous import
	for imp, files := range foundImports {
		severity := "medium"
		recommendation := fmt.Sprintf("Review usage of %s package for security implications", imp)

		// Escalate severity for particularly dangerous imports
		if imp == "unsafe" || imp == "syscall" {
			severity = "high"
			recommendation = fmt.Sprintf("Usage of %s requires careful security review. Ensure no unsafe operations are performed.", imp)
		} else if imp == "os/exec" {
			severity = "high"
			recommendation = "Command execution via os/exec can be dangerous. Ensure input validation and no shell injection vulnerabilities."
		} else if imp == "crypto/md5" || imp == "crypto/sha1" {
			severity = "medium"
			recommendation = fmt.Sprintf("%s is cryptographically weak. Use SHA-256 or stronger algorithms.", imp)
		}

		issues = append(issues, SecurityIssue{
			Severity:       severity,
			Category:       "dangerous-import",
			Description:    fmt.Sprintf("Plugin imports potentially dangerous package: %s (found in %d file(s))", imp, len(files)),
			File:           files[0], // Report first file
			Recommendation: recommendation,
		})
	}

	return issues, nil
}

// runGosec executes the gosec security scanner on the plugin
func (v *Validator) runGosec(ctx context.Context, pluginPath string) ([]SecurityIssue, error) {
	if v.gosecPath == "" {
		return nil, fmt.Errorf("gosec not available")
	}

	// Create temporary file for gosec output
	tmpFile, err := os.CreateTemp("", "gosec-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run gosec with JSON output
	cmd := exec.CommandContext(ctx, v.gosecPath,
		"-fmt=json",
		"-out="+tmpFile.Name(),
		"-no-fail", // Don't fail on findings
		pluginPath+"/...",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Gosec returns non-zero on findings, which is expected
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means findings were detected, which is OK
			if exitErr.ExitCode() != 1 {
				return nil, fmt.Errorf("gosec failed: %s", stderr.String())
			}
		}
	}

	// Parse gosec JSON output
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read gosec output: %w", err)
	}

	var gosecReport struct {
		Issues []struct {
			Severity   string `json:"severity"`
			Confidence string `json:"confidence"`
			RuleID     string `json:"rule_id"`
			Details    string `json:"details"`
			File       string `json:"file"`
			Line       string `json:"line"`
			Column     string `json:"column"`
			CWE        struct {
				ID string `json:"id"`
			} `json:"cwe"`
		} `json:"Issues"`
	}

	if err := json.Unmarshal(content, &gosecReport); err != nil {
		return nil, fmt.Errorf("failed to parse gosec output: %w", err)
	}

	// Convert gosec issues to SecurityIssue format
	var issues []SecurityIssue
	for _, gosecIssue := range gosecReport.Issues {
		// Map gosec severity to our severity levels
		severity := strings.ToLower(gosecIssue.Severity)
		if severity == "high" && gosecIssue.Confidence == "HIGH" {
			severity = "critical"
		}

		relPath, _ := filepath.Rel(pluginPath, gosecIssue.File)

		issues = append(issues, SecurityIssue{
			Severity:    severity,
			Category:    gosecIssue.RuleID,
			Description: gosecIssue.Details,
			File:        relPath,
			Line:        parseLineNumber(gosecIssue.Line),
			CWEID:       gosecIssue.CWE.ID,
		})
	}

	return issues, nil
}

// checkHardcodedSecrets looks for hardcoded API keys, tokens, passwords
func (v *Validator) checkHardcodedSecrets(pluginPath string) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	// Common secret patterns
	secretPatterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"API Key", regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*["']([a-zA-Z0-9]{20,})["']`)},
		{"Password", regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["']([^"']{8,})["']`)},
		{"Token", regexp.MustCompile(`(?i)(token|auth[_-]?token)\s*[:=]\s*["']([a-zA-Z0-9]{20,})["']`)},
		{"AWS Key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
		{"Private Key", regexp.MustCompile(`-----BEGIN (RSA |EC )?PRIVATE KEY-----`)},
	}

	err := filepath.Walk(pluginPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, secret := range secretPatterns {
			matches := secret.pattern.FindAll(content, -1)
			if len(matches) > 0 {
				relPath, _ := filepath.Rel(pluginPath, path)
				issues = append(issues, SecurityIssue{
					Severity:       "high",
					Category:       "hardcoded-secret",
					Description:    fmt.Sprintf("Potential hardcoded %s detected", secret.name),
					File:           relPath,
					Recommendation: "Remove hardcoded secrets. Use environment variables or secure configuration.",
					CWEID:          "CWE-798",
				})
			}
		}

		return nil
	})

	return issues, err
}

// checkSuspiciousFileOperations looks for risky file operations
func (v *Validator) checkSuspiciousFileOperations(pluginPath string) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	suspiciousPatterns := []struct {
		name        string
		pattern     *regexp.Regexp
		severity    string
		description string
	}{
		{
			"Path Traversal",
			regexp.MustCompile(`\.\./`),
			"medium",
			"Potential path traversal vulnerability detected",
		},
		{
			"Dangerous File Write",
			regexp.MustCompile(`(?i)os\.WriteFile.*(/etc/|/usr/|/sys/|C:\\Windows)`),
			"high",
			"Writing to system directories detected",
		},
		{
			"Shell Command",
			regexp.MustCompile(`(?i)(sh\s+-c|bash\s+-c|cmd\.exe)`),
			"high",
			"Shell command execution detected - potential command injection",
		},
	}

	err := filepath.Walk(pluginPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, pattern := range suspiciousPatterns {
			if pattern.pattern.Match(content) {
				relPath, _ := filepath.Rel(pluginPath, path)
				issues = append(issues, SecurityIssue{
					Severity:       pattern.severity,
					Category:       "suspicious-file-operation",
					Description:    pattern.description,
					File:           relPath,
					Recommendation: "Review file operations for security implications",
				})
			}
		}

		return nil
	})

	return issues, err
}

// isAllowedPermission checks if a permission is in the allowed list
func (v *Validator) isAllowedPermission(permission string) bool {
	return v.allowedPermissions[permission]
}

// Helper functions

func isValidPluginID(id string) bool {
	// Plugin IDs must be lowercase alphanumeric with hyphens
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`, id)
	return matched
}

func isValidURL(url string) bool {
	matched, _ := regexp.MatchString(`^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, url)
	return matched
}

func parseLineNumber(lineStr string) int {
	var line int
	fmt.Sscanf(lineStr, "%d", &line)
	return line
}

func findGosecPath() string {
	// Try to find gosec in PATH
	path, err := exec.LookPath("gosec")
	if err == nil {
		return path
	}

	// Try common installation locations
	commonPaths := []string{
		"/usr/local/bin/gosec",
		"/usr/bin/gosec",
		os.Getenv("GOPATH") + "/bin/gosec",
		os.Getenv("HOME") + "/go/bin/gosec",
	}

	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return "" // Not found
}
