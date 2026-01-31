package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/compatibility"
)

func newCheckCompatibilityCommand() *Command {
	cmd := &Command{
		Name:        "check-compatibility",
		Description: "Check compatibility between two proto schema versions",
		Flags:       flag.NewFlagSet("check-compatibility", flag.ExitOnError),
		Run:         runCheckCompatibility,
	}

	cmd.Flags.String("old", "", "Directory or file containing old proto schema (required)")
	cmd.Flags.String("new", "", "Directory or file containing new proto schema (required)")
	cmd.Flags.String("mode", "BACKWARD", "Compatibility mode: BACKWARD, FORWARD, FULL, BACKWARD_TRANSITIVE, FORWARD_TRANSITIVE, FULL_TRANSITIVE")
	cmd.Flags.Bool("verbose", false, "Show all violations including info level")
	cmd.Flags.String("format", "text", "Output format: text, json")

	return cmd
}

func runCheckCompatibility(args []string) error {
	flags := flag.NewFlagSet("check-compatibility", flag.ExitOnError)
	oldPath := flags.String("old", "", "Directory or file containing old proto schema (required)")
	newPath := flags.String("new", "", "Directory or file containing new proto schema (required)")
	mode := flags.String("mode", "BACKWARD", "Compatibility mode")
	verbose := flags.Bool("verbose", false, "Show all violations including info level")
	format := flags.String("format", "text", "Output format: text, json")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *oldPath == "" || *newPath == "" {
		return fmt.Errorf("both --old and --new are required")
	}

	// Parse compatibility mode
	compatMode, err := compatibility.ParseCompatibilityMode(*mode)
	if err != nil {
		return fmt.Errorf("invalid compatibility mode: %v", err)
	}

	// Parse old schema
	fmt.Printf("Parsing old schema from %s...\n", *oldPath)
	oldSchema, err := parseSchema(*oldPath)
	if err != nil {
		return fmt.Errorf("failed to parse old schema: %v", err)
	}

	// Parse new schema
	fmt.Printf("Parsing new schema from %s...\n", *newPath)
	newSchema, err := parseSchema(*newPath)
	if err != nil {
		return fmt.Errorf("failed to parse new schema: %v", err)
	}

	// Run compatibility check
	fmt.Printf("Checking compatibility (mode: %s)...\n\n", compatMode.String())
	result, err := compatibility.CheckCompatibility(oldSchema, newSchema, compatMode)
	if err != nil {
		return fmt.Errorf("compatibility check failed: %v", err)
	}

	// Output results
	if *format == "json" {
		return outputJSON(result)
	}

	return outputText(result, *verbose)
}

func parseSchema(path string) (*compatibility.SchemaGraph, error) {
	// Check if path is a file or directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %v", err)
	}

	var protoFiles []string
	if info.IsDir() {
		// Find all proto files in directory
		err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(p) == ".proto" {
				protoFiles = append(protoFiles, p)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %v", err)
		}
	} else {
		protoFiles = []string{path}
	}

	if len(protoFiles) == 0 {
		return nil, fmt.Errorf("no proto files found in %s", path)
	}

	// For now, parse the first file only
	// TODO: Merge multiple files into a single schema
	content, err := os.ReadFile(protoFiles[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Parse protobuf
	ast, err := protobuf.ParseString(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse proto: %v", err)
	}

	// Build schema graph
	builder := compatibility.NewSchemaGraphBuilder()
	schema, err := builder.BuildFromAST(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to build schema graph: %v", err)
	}

	return schema, nil
}

func outputText(result *compatibility.CheckResult, verbose bool) error {
	// Print summary
	fmt.Printf("Compatibility Check: %s\n", result.Mode)
	fmt.Printf("Result: ")
	if result.Compatible {
		fmt.Printf("\033[32mCOMPATIBLE\033[0m\n\n")
	} else {
		fmt.Printf("\033[31mINCOMPATIBLE\033[0m\n\n")
	}

	// Print summary statistics
	summary := result.Summary
	fmt.Printf("Summary:\n")
	fmt.Printf("  Total Violations: %d\n", summary.TotalViolations)
	if summary.Errors > 0 {
		fmt.Printf("  Errors:           \033[31m%d\033[0m\n", summary.Errors)
	} else {
		fmt.Printf("  Errors:           %d\n", summary.Errors)
	}
	if summary.Warnings > 0 {
		fmt.Printf("  Warnings:         \033[33m%d\033[0m\n", summary.Warnings)
	} else {
		fmt.Printf("  Warnings:         %d\n", summary.Warnings)
	}
	fmt.Printf("  Info:             %d\n", summary.Infos)
	fmt.Printf("  Wire Breaking:    %d\n", summary.WireBreaking)
	fmt.Printf("  Source Breaking:  %d\n\n", summary.SourceBreaking)

	// Print violations
	if len(result.Violations) > 0 {
		fmt.Printf("Violations:\n\n")
		for _, v := range result.Violations {
			// Skip info level if not verbose
			if !verbose && v.Level == compatibility.ViolationLevelInfo {
				continue
			}

			// Color code by level
			levelStr := v.Level.String()
			switch v.Level {
			case compatibility.ViolationLevelError:
				levelStr = fmt.Sprintf("\033[31m%s\033[0m", levelStr)
			case compatibility.ViolationLevelWarning:
				levelStr = fmt.Sprintf("\033[33m%s\033[0m", levelStr)
			case compatibility.ViolationLevelInfo:
				levelStr = fmt.Sprintf("\033[36m%s\033[0m", levelStr)
			}

			fmt.Printf("[%s] %s\n", levelStr, v.Rule)
			fmt.Printf("  Location: %s\n", v.Location)
			fmt.Printf("  Message:  %s\n", v.Message)
			if v.OldValue != "" || v.NewValue != "" {
				fmt.Printf("  Change:   %s → %s\n", v.OldValue, v.NewValue)
			}
			if v.WireBreaking || v.SourceBreaking {
				flags := []string{}
				if v.WireBreaking {
					flags = append(flags, "wire-breaking")
				}
				if v.SourceBreaking {
					flags = append(flags, "source-breaking")
				}
				fmt.Printf("  Breaking: %s\n", flags)
			}
			if v.Suggestion != "" {
				fmt.Printf("  Hint:     %s\n", v.Suggestion)
			}
			fmt.Println()
		}
	}

	// Exit with non-zero status if incompatible
	if !result.Compatible {
		return fmt.Errorf("compatibility check failed")
	}

	return nil
}

func outputJSON(result *compatibility.CheckResult) error {
	// TODO: Implement JSON output
	return fmt.Errorf("JSON output not yet implemented")
}
