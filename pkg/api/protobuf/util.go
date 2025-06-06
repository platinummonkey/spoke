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

// ParseFile parses a proto file and returns its AST
// This is currently a stub as the full parser is not implemented yet
func ParseFile(path string) (*RootNode, error) {
	// Temporary implementation until the full parser is ready
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	return parseProtoContent(string(content))
}

// ParseReader parses a proto file from a reader and returns its AST
// This is currently a stub as the full parser is not implemented yet
func ParseReader(r io.Reader) (*RootNode, error) {
	// Read the entire content
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}
	
	return parseProtoContent(string(content))
}

// ParseString parses a proto file from a string and returns its AST
// This is currently a stub as the full parser is not implemented yet
func ParseString(s string) (*RootNode, error) {
	return parseProtoContent(s)
}

// parseProtoContent is a temporary implementation that uses regex to extract information
// until the full parser is implemented
func parseProtoContent(content string) (*RootNode, error) {
	return NewStringParser(content).Parse()
}

// ExtractPackageName extracts the package name from a protobuf file content
func ExtractPackageName(content string) (string, error) {
	parser := NewStringParser(content)
	ast, err := parser.Parse()
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
	parser := NewStringParser(content)
	ast, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	
	var imports []ProtoImport
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
	parser := NewStringParser(content)
	_, err := parser.Parse()
	return err
}
