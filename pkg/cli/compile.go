package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// ProtoFileInfo holds metadata about a protobuf file
type ProtoFileInfo struct {
	Path        string
	PackageName string
	Domain      string
}

// parseProtoFileForSpokeDirectives parses a proto file and extracts spoke directives
func parseProtoFileForSpokeDirectives(filePath string) (*ProtoFileInfo, error) {
	ast, err := protobuf.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proto file %s: %w", filePath, err)
	}

	info := &ProtoFileInfo{
		Path: filePath,
	}

	// Extract package name
	if ast.Package != nil {
		info.PackageName = ast.Package.Name
	}

	// Look for domain directives on root node
	for _, directive := range ast.SpokeDirectives {
		if directive.Option == "domain" {
			info.Domain = directive.Value
			break // Use the first domain directive found
		}
	}

	// Also check package node for directives (they're often attached there)
	if ast.Package != nil {
		for _, directive := range ast.Package.SpokeDirectives {
			if directive.Option == "domain" {
				info.Domain = directive.Value
				break
			}
		}
	}

	return info, nil
}

func newCompileCommand() *Command {
	cmd := &Command{
		Name:        "compile",
		Description: "Compile protobuf files using protoc (supports single or multiple languages)",
		Flags:       flag.NewFlagSet("compile", flag.ExitOnError),
		Run:         runCompile,
	}

	cmd.Flags.String("dir", ".", "Directory containing protobuf files")
	cmd.Flags.String("out", ".", "Output directory for generated files")
	cmd.Flags.String("lang", "go", "Output language for single language compilation (deprecated, use --languages)")
	cmd.Flags.String("languages", "", "Comma-separated list of languages to compile for (e.g., go,python,java)")
	cmd.Flags.Bool("grpc", false, "Include gRPC code generation")
	cmd.Flags.Bool("parallel", false, "Compile multiple languages in parallel")
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
	languagesStr := cmd.Flags.Lookup("languages").Value.String()
	includeGRPC := cmd.Flags.Lookup("grpc").Value.String() == "true"
	parallel := cmd.Flags.Lookup("parallel").Value.String() == "true"
	recursive := cmd.Flags.Lookup("recursive").Value.String() == "true"

	// Determine which languages to compile
	var languages []string
	if languagesStr != "" {
		// New multi-language mode
		languages = strings.Split(languagesStr, ",")
		for i, l := range languages {
			languages[i] = strings.TrimSpace(l)
		}
	} else {
		// Legacy single language mode
		languages = []string{lang}
	}

	fmt.Printf("Compiling for languages: %v\n", languages)
	if includeGRPC {
		fmt.Println("Including gRPC code generation")
	}

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

	// Parse proto files for spoke directives
	var domainToPackageMap = make(map[string][]string) // domain -> list of packages

	for _, protoFile := range protoFiles {
		info, err := parseProtoFileForSpokeDirectives(protoFile)
		if err != nil {
			fmt.Printf("Warning: failed to parse spoke directives from %s: %v\n", protoFile, err)
			// Continue with compilation even if spoke directive parsing fails
			info = &ProtoFileInfo{Path: protoFile}
		}

		// Group packages by domain
		if info.Domain != "" && info.PackageName != "" {
			domainToPackageMap[info.Domain] = append(domainToPackageMap[info.Domain], info.PackageName)
		}
	}

	// Compile for each language
	for _, language := range languages {
		fmt.Printf("\n=== Compiling for %s ===\n", language)

		// Create language-specific output directory
		langOut := filepath.Join(out, language)
		if err := os.MkdirAll(langOut, 0755); err != nil {
			return fmt.Errorf("failed to create output directory for %s: %w", language, err)
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
		switch language {
		case "go":
			protocArgs = append(protocArgs,
				"--go_out="+langOut,
				"--go_opt=paths=source_relative",
			)

			// Add module mapping based on spoke domain directives
			for domain, packages := range domainToPackageMap {
				for _, pkg := range packages {
					// Map proto package to domain/package import path
					moduleMapping := fmt.Sprintf("--go_opt=M%s=%s/%s", pkg+".proto", domain, pkg)
					protocArgs = append(protocArgs, moduleMapping)
				}
			}

			if includeGRPC {
				protocArgs = append(protocArgs,
					"--go-grpc_out="+langOut,
					"--go-grpc_opt=paths=source_relative",
				)
			}

		case "python":
			protocArgs = append(protocArgs, "--python_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc_python_out="+langOut)
			}

		case "java":
			protocArgs = append(protocArgs, "--java_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc-java_out="+langOut)
			}

		case "cpp":
			protocArgs = append(protocArgs, "--cpp_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc_out="+langOut, "--plugin=protoc-gen-grpc=grpc_cpp_plugin")
			}

		case "csharp":
			protocArgs = append(protocArgs, "--csharp_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc_out="+langOut, "--plugin=protoc-gen-grpc=grpc_csharp_plugin")
			}

		case "rust":
			protocArgs = append(protocArgs, "--rust_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--tonic_out="+langOut)
			}

		case "typescript", "ts":
			protocArgs = append(protocArgs, "--ts_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc-web_out="+langOut)
			}

		case "javascript", "js":
			protocArgs = append(protocArgs, "--js_out=import_style=commonjs:"+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc-web_out="+langOut)
			}

		case "dart":
			protocArgs = append(protocArgs, "--dart_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--dart_out=grpc:"+langOut)
			}

		case "swift":
			protocArgs = append(protocArgs, "--swift_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc-swift_out="+langOut)
			}

		case "kotlin":
			protocArgs = append(protocArgs, "--kotlin_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc-kotlin_out="+langOut)
			}

		case "objc":
			protocArgs = append(protocArgs, "--objc_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc_out="+langOut, "--plugin=protoc-gen-grpc=grpc_objective_c_plugin")
			}

		case "ruby":
			protocArgs = append(protocArgs, "--ruby_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc_out="+langOut, "--plugin=protoc-gen-grpc=grpc_ruby_plugin")
			}

		case "php":
			protocArgs = append(protocArgs, "--php_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--grpc_out="+langOut, "--plugin=protoc-gen-grpc=grpc_php_plugin")
			}

		case "scala":
			protocArgs = append(protocArgs, "--scala_out="+langOut)
			if includeGRPC {
				protocArgs = append(protocArgs, "--scala_out=grpc:"+langOut)
			}

		default:
			fmt.Printf("Warning: unsupported language %s, skipping\n", language)
			continue
		}

		// Add proto files
		protocArgs = append(protocArgs, protoFiles...)

		// Run protoc
		protocCmd := exec.Command("protoc", protocArgs...)
		protocCmd.Stdout = os.Stdout
		protocCmd.Stderr = os.Stderr

		fmt.Printf("Running: protoc %s\n", strings.Join(protocArgs, " "))

		if err := protocCmd.Run(); err != nil {
			fmt.Printf("Error: failed to compile for %s: %v\n", language, err)
			if !parallel {
				return fmt.Errorf("failed to run protoc for %s: %w", language, err)
			}
			// In parallel mode, continue with other languages
			continue
		}

		fmt.Printf("✓ Successfully compiled %s to %s\n", language, langOut)
	}

	fmt.Printf("\n=== Compilation Complete ===\n")
	fmt.Printf("Output directory: %s\n", out)
	return nil
} 