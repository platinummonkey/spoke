package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/linter"
	"github.com/platinummonkey/spoke/pkg/linter/rules"
)

// newLintCommand creates a new lint command
func newLintCommand() *Command {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)

	var (
		dir           = fs.String("dir", ".", "Directory containing proto files")
		configFile    = fs.String("config", "", "Path to lint config file (spoke-lint.yaml)")
		format        = fs.String("format", "text", "Output format: text, json, github")
		autoFix       = fs.Bool("fix", false, "Automatically fix violations")
		failOnError   = fs.Bool("fail-on-error", true, "Exit with error code on lint errors")
		failOnWarning = fs.Bool("fail-on-warning", false, "Exit with error code on lint warnings")
		verbose       = fs.Bool("verbose", false, "Verbose output")
		rulesOnly     = fs.Bool("rules", false, "List available rules and exit")
	)

	return &Command{
		Name:        "lint",
		Description: "Lint protobuf files for style and quality",
		Flags:       fs,
		Run: func(args []string) error {
			if err := fs.Parse(args); err != nil {
				return err
			}

			return runLint(*dir, *configFile, *format, *autoFix, *failOnError, *failOnWarning, *verbose, *rulesOnly)
		},
	}
}

func runLint(dir, configFile, format string, autoFix, failOnError, failOnWarning, verbose, rulesOnly bool) error {
	// Load configuration
	var config *linter.Config
	var err error
	if configFile != "" {
		config, err = linter.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		config, err = linter.LoadConfigFromDir(dir)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create linter engine
	engine := linter.NewLintEngine(config)

	// Register default rules
	for _, rule := range rules.DefaultRules() {
		engine.Registry().Register(rule)
	}

	// List rules if requested
	if rulesOnly {
		return lintListRules(engine)
	}

	// Find proto files
	protoFiles, err := lintFindProtoFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to find proto files: %w", err)
	}

	if len(protoFiles) == 0 {
		fmt.Printf("No proto files found in %s\n", dir)
		return nil
	}

	if verbose {
		fmt.Printf("Linting %d proto files...\n", len(protoFiles))
	}

	// Parse and lint files
	files := make(map[string]*protobuf.RootNode)
	for _, file := range protoFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		ast, err := protobuf.ParseString(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", file, err)
		}

		files[file] = ast
	}

	results := engine.LintFiles(files)

	// Apply auto-fix if requested
	if autoFix {
		if verbose {
			fmt.Println("Applying auto-fixes...")
		}
		// TODO: Implement auto-fix application
		fmt.Println("Auto-fix not yet implemented")
	}

	// Generate summary
	summary := engine.GenerateSummary(results)

	// Output results
	switch format {
	case "json":
		return lintOutputJSON(results, summary)
	case "github":
		return lintOutputGitHub(results)
	default:
		return lintOutputText(results, summary, verbose, failOnError, failOnWarning)
	}
}

func lintFindProtoFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and vendor directories
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "third_party" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only include .proto files
		if filepath.Ext(path) == ".proto" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func lintListRules(engine *linter.LintEngine) error {
	allRules := engine.Registry().GetAllRules()

	fmt.Printf("Available lint rules (%d):\n\n", len(allRules))

	// Group by category
	byCategory := make(map[linter.Category][]linter.Rule)
	for _, rule := range allRules {
		cat := rule.Category()
		byCategory[cat] = append(byCategory[cat], rule)
	}

	for _, cat := range []linter.Category{
		linter.CategoryNaming,
		linter.CategoryStyle,
		linter.CategoryDocumentation,
		linter.CategoryStructure,
	} {
		rules := byCategory[cat]
		if len(rules) == 0 {
			continue
		}

		// Capitalize category name
		catName := string(cat)
		if len(catName) > 0 {
			catName = strings.ToUpper(string(catName[0])) + catName[1:]
		}

		fmt.Printf("%s Rules:\n", catName)
		for _, rule := range rules {
			autofix := ""
			if rule.CanAutoFix() {
				autofix = " [auto-fix]"
			}
			fmt.Printf("  - %-25s [%s]%s\n    %s\n",
				rule.Name(),
				rule.Severity(),
				autofix,
				rule.Description(),
			)
		}
		fmt.Println()
	}

	return nil
}

func lintOutputText(results []linter.LintResult, summary linter.Summary, verbose, failOnError, failOnWarning bool) error {
	hasViolations := false

	for _, result := range results {
		if len(result.Violations) == 0 {
			continue
		}

		hasViolations = true
		fmt.Printf("\n%s:\n", result.FilePath)

		for _, v := range result.Violations {
			fmt.Printf("  %s:%d:%d: [%s] %s (%s)\n",
				result.FilePath,
				v.Position.Line,
				v.Position.Column,
				v.Severity,
				v.Message,
				v.Rule,
			)

			if v.SuggestedFix != nil && verbose {
				fmt.Printf("    Fix: %s\n", v.SuggestedFix.Description)
			}
		}
	}

	// Print summary
	fmt.Printf("\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("  Files:      %d\n", summary.TotalFiles)
	fmt.Printf("  Violations: %d\n", summary.TotalViolations)
	fmt.Printf("  Errors:     %d\n", summary.Errors)
	fmt.Printf("  Warnings:   %d\n", summary.Warnings)
	fmt.Printf("  Infos:      %d\n", summary.Infos)

	// Exit with error if needed
	if failOnError && summary.Errors > 0 {
		return fmt.Errorf("lint failed with %d errors", summary.Errors)
	}

	if failOnWarning && summary.Warnings > 0 {
		return fmt.Errorf("lint failed with %d warnings", summary.Warnings)
	}

	if !hasViolations {
		fmt.Println("\n✓ All files passed linting")
	}

	return nil
}

func lintOutputJSON(results []linter.LintResult, summary linter.Summary) error {
	output := struct {
		Results []linter.LintResult `json:"results"`
		Summary linter.Summary      `json:"summary"`
	}{
		Results: results,
		Summary: summary,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func lintOutputGitHub(results []linter.LintResult) error {
	// GitHub Actions annotation format
	// ::error file={name},line={line},col={col}::{message}
	for _, result := range results {
		for _, v := range result.Violations {
			level := "error"
			if v.Severity == linter.SeverityWarning {
				level = "warning"
			} else if v.Severity == linter.SeverityInfo {
				level = "notice"
			}

			fmt.Printf("::%s file=%s,line=%d,col=%d::[%s] %s\n",
				level,
				result.FilePath,
				v.Position.Line,
				v.Position.Column,
				v.Rule,
				v.Message,
			)
		}
	}

	return nil
}
