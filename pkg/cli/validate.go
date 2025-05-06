package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func newValidateCommand() *Command {
	cmd := &Command{
		Name:        "validate",
		Description: "Validate protobuf files",
		Flags:       flag.NewFlagSet("validate", flag.ExitOnError),
		Run:         runValidate,
	}

	cmd.Flags.String("dir", ".", "Directory containing protobuf files")
	cmd.Flags.Bool("recursive", false, "Validate dependencies recursively")

	return cmd
}

func runValidate(args []string) error {
	flags := flag.NewFlagSet("validate", flag.ExitOnError)
	dir := flags.String("dir", ".", "Directory containing protobuf files")
	recursive := flags.Bool("recursive", false, "Validate dependencies recursively")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Find all proto files in the directory
	var protoFiles []string
	err := filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".proto" {
			protoFiles = append(protoFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find proto files: %v", err)
	}

	if len(protoFiles) == 0 {
		return fmt.Errorf("no proto files found in %s", *dir)
	}

	// Create a temporary directory for validation output
	tmpDir, err := os.MkdirTemp("", "validate")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build protoc command
	protocArgs := []string{
		"--proto_path=" + *dir,
		"--include_imports",
		"--include_source_info",
		"--descriptor_set_out=" + filepath.Join(tmpDir, "validation.pb"),
	}

	// Add dependency paths if recursive
	if *recursive {
		protocArgs = append(protocArgs, "--proto_path="+filepath.Dir(*dir))
	}

	// Add proto files to validate
	for _, file := range protoFiles {
		relPath, err := filepath.Rel(*dir, file)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}
		protocArgs = append(protocArgs, relPath)
	}

	// Run protoc to validate
	cmd := exec.Command("protoc", protocArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("validation failed: %s", string(output))
		}
		return fmt.Errorf("validation failed: %v", err)
	}

	// Additional validations
	for _, file := range protoFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}

		// Check for common issues
		if err := validateProtoFile(string(content)); err != nil {
			return fmt.Errorf("validation failed for %s: %w", file, err)
		}
	}

	fmt.Println("All proto files are valid")
	return nil
}

func validateProtoFile(content string) error {
	// Check for package declaration
	if !strings.Contains(content, "package") {
		return fmt.Errorf("missing package declaration")
	}

	// Check for syntax version
	if !strings.Contains(content, "syntax =") {
		return fmt.Errorf("missing syntax version")
	}

	// Check for common issues
	lines := strings.Split(content, "\n")
	openBraces := 0
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Check for message/enum/service declarations
		if strings.HasPrefix(line, "message") || strings.HasPrefix(line, "enum") || strings.HasPrefix(line, "service") {
			// Look for opening brace in the same line or next line
			if !strings.Contains(line, "{") {
				if i+1 >= len(lines) || !strings.Contains(strings.TrimSpace(lines[i+1]), "{") {
					return fmt.Errorf("invalid declaration at line %d: missing opening brace", i+1)
				}
			}
		}

		// Count braces
		openBraces += strings.Count(line, "{")
		openBraces -= strings.Count(line, "}")

		// Check for unclosed braces
		if openBraces < 0 {
			return fmt.Errorf("unmatched closing brace at line %d", i+1)
		}
	}

	// Check for unclosed braces at the end
	if openBraces > 0 {
		return fmt.Errorf("unclosed braces at end of file")
	}

	return nil
} 