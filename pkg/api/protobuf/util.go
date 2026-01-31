package protobuf

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// ProtoImport represents a parsed import statement with version information
type ProtoImport struct {
	Module  string
	Version string
	Path    string
	Public  bool
	Weak    bool
}

// ParseFile parses a proto file and returns its AST using the descriptor parser
func ParseFile(path string) (*RootNode, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return ParseWithDescriptor(path, string(content))
}

// ParseReader parses a proto file from a reader and returns its AST using the descriptor parser
func ParseReader(r io.Reader) (*RootNode, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	return ParseWithDescriptor("input.proto", string(content))
}

// ParseString parses a proto file from a string and returns its AST using the descriptor parser
func ParseString(s string) (*RootNode, error) {
	return ParseWithDescriptor("input.proto", s)
}

// ExtractPackageName extracts the package name from a protobuf file content
func ExtractPackageName(content string) (string, error) {
	ast, err := ParseWithDescriptor("input.proto", content)
	if err != nil {
		return "", err
	}

	if ast.Package == nil {
		return "", errors.New("no package statement found")
	}

	return ast.Package.Name, nil
}

// ExtractImports extracts import statements from a protobuf file content
func ExtractImports(content string) ([]ProtoImport, error) {
	ast, err := ParseWithDescriptor("input.proto", content)
	if err != nil {
		return nil, err
	}

	imports := make([]ProtoImport, 0)
	for _, imp := range ast.Imports {
		imports = append(imports, ProtoImport{
			Path:   imp.Path,
			Public: imp.Public,
			Weak:   imp.Weak,
			// Module and Version can be extracted from Path if needed
			// For now, leave them empty or extract from path
		})
	}

	return imports, nil
}

// ValidateProtoFile validates the syntax of a protobuf file
func ValidateProtoFile(content string) error {
	_, err := ParseWithDescriptor("input.proto", content)
	return err
}
