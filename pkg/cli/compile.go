package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func newCompileCommand() *Command {
	cmd := &Command{
		Name:        "compile",
		Description: "Compile protobuf files using protoc",
		Flags:       flag.NewFlagSet("compile", flag.ExitOnError),
		Run:         runCompile,
	}

	cmd.Flags.String("dir", ".", "Directory containing protobuf files")
	cmd.Flags.String("out", ".", "Output directory for generated files")
	cmd.Flags.String("lang", "go", "Output language (go, cpp, java, etc.)")
	cmd.Flags.Bool("recursive", false, "Pull dependencies recursively")

	return cmd
}

func runCompile(args []string) error {
	cmd := newCompileCommand()
	if err := cmd.Flags.Parse(args); err != nil {
		return err
	}

	dir := cmd.Flags.Lookup("dir").Value.String()
	out := cmd.Flags.Lookup("out").Value.String()
	lang := cmd.Flags.Lookup("lang").Value.String()
	recursive := cmd.Flags.Lookup("recursive").Value.String() == "true"

	// Create output directory
	if err := os.MkdirAll(out, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Find all proto files
	var protoFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".proto") {
			protoFiles = append(protoFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find proto files: %w", err)
	}

	if len(protoFiles) == 0 {
		return fmt.Errorf("no proto files found in directory: %s", dir)
	}

	// Build protoc command
	protocArgs := []string{
		"--proto_path=" + dir,
	}

	// Add dependency paths
	if recursive {
		// Add the parent directory as a proto path to handle imports
		parentDir := filepath.Dir(dir)
		if parentDir != "." {
			protocArgs = append(protocArgs, "--proto_path="+parentDir)
		}
	}

	// Add language-specific output
	switch lang {
	case "go":
		protocArgs = append(protocArgs,
			"--go_out="+out,
			"--go_opt=paths=source_relative",
		)
	case "cpp":
		protocArgs = append(protocArgs, "--cpp_out="+out)
	case "java":
		protocArgs = append(protocArgs, "--java_out="+out)
	default:
		return fmt.Errorf("unsupported language: %s", lang)
	}

	// Add proto files
	protocArgs = append(protocArgs, protoFiles...)

	// Run protoc
	protocCmd := exec.Command("protoc", protocArgs...)
	protocCmd.Stdout = os.Stdout
	protocCmd.Stderr = os.Stderr

	if err := protocCmd.Run(); err != nil {
		return fmt.Errorf("failed to run protoc: %w", err)
	}

	fmt.Printf("Successfully compiled proto files to %s\n", out)
	return nil
} 