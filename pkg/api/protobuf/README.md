# Protobuf AST Parser

This package provides an Abstract Syntax Tree (AST) parser for Protocol Buffer (protobuf) files.

## Features

- Full AST parsing of protobuf files
- Support for reading and extracting comments
- Validation of protobuf syntax and structure
- Extraction of imports, package names, and dependencies
- Support for nested message types and versioned imports

## Usage

### Parsing a Proto File

```go
import "github.com/platinummonkey/spoke/pkg/api/protobuf"

// Parse from a file
ast, err := protobuf.ParseFile("path/to/file.proto")
if err != nil {
    // Handle error
}

// Parse from a string
content := `syntax = "proto3";
package example;

message Test {
    string id = 1;
}`

ast, err := protobuf.ParseString(content)
if err != nil {
    // Handle error
}
```

### Extracting Imports

```go
imports, err := protobuf.ExtractImports(protoContent)
if err != nil {
    // Handle error
}

for _, imp := range imports {
    fmt.Printf("Module: %s, Version: %s, Path: %s\n", imp.Module, imp.Version, imp.Path)
}
```

### Extracting Package Name

```go
packageName, err := protobuf.ExtractPackageName(protoContent)
if err != nil {
    // Handle error
}
fmt.Printf("Package: %s\n", packageName)
```

### Extracting Comments

```go
comments, err := protobuf.ExtractComments(protoContent)
if err != nil {
    // Handle error
}

for _, comment := range comments {
    fmt.Printf("Comment: %s\n", comment)
}
```

### Validating Proto Files

```go
err := protobuf.ValidateProtoFile(protoContent)
if err != nil {
    fmt.Printf("Validation failed: %v\n", err)
} else {
    fmt.Println("Proto file is valid")
}
```

## Implementation Details

The parser consists of:

1. **Scanner (Tokenizer)**: Breaks down the input into tokens like identifiers, strings, numbers, and punctuation
2. **Parser**: Builds an AST from the tokens, following the protobuf grammar
3. **AST Nodes**: Represent different elements of the protobuf file (syntax, package, imports, messages, etc.)
4. **Utility Functions**: Extract specific information from the AST for common use cases 