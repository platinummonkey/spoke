# Spoke - Protobuf Schema Registry

![Spoke Logo](./web/public/logos/logo_main.png)

THIS IS A POC HACK WEEK PROJECT. NOT FOR PRODUCTION USE.

Spoke is a Protobuf Schema Registry that helps manage and version your Protocol Buffer definitions. It provides a simple way to store, retrieve, and compile protobuf files with dependency management.

Build Once. Connect Everywhere. The Hub of Schema-Driven Development.


## Features

- Store and version protobuf files
- Pull dependencies recursively
- Compile protobuf files to multiple languages
- Validate protobuf files
- AST parsing of protobuf files with comment support
- Simple HTTP API
- Command-line interface

## Prerequisites

- Go 1.16 or later
- Protocol Buffers compiler (protoc)
- Language-specific protoc plugins (e.g., protoc-gen-go for Go)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/platinummonkey/spoke.git
cd spoke
```

2. Build the server and CLI:
```bash
go build -o bin/spoke-server cmd/spoke/main.go
go build -o bin/spoke cmd/spoke-cli/main.go
```

3. Add the binaries to your PATH:
```bash
export PATH=$PATH:$(pwd)/bin
```

## Usage

### Starting the Server

Start the Spoke server with default settings:
```bash
spoke-server
```

Or customize the port and storage directory:
```bash
spoke-server -port 8080 -storage-dir /path/to/storage
```

### CLI Commands

#### Push a Module

Push protobuf files to the registry:
```bash
spoke push -module mymodule -version v1.0.0 -dir ./proto
```

Options:
- `-module`: Module name (required)
- `-version`: Version (semantic version or commit hash) (required)
- `-dir`: Directory containing protobuf files (default: ".")
- `-registry`: Registry URL (default: "http://localhost:8080")
- `-description`: Module description

#### Pull a Module

Pull protobuf files from the registry:
```bash
spoke pull -module mymodule -version v1.0.0 -dir ./proto
```

Options:
- `-module`: Module name (required)
- `-version`: Version (semantic version or commit hash) (required)
- `-dir`: Directory to save protobuf files (default: ".")
- `-registry`: Registry URL (default: "http://localhost:8080")
- `-recursive`: Pull dependencies recursively (default: false)

#### Compile Protobuf Files

Compile protobuf files using protoc:
```bash
spoke compile -dir ./proto -out ./generated -lang go
```

Options:
- `-dir`: Directory containing protobuf files (default: ".")
- `-out`: Output directory for generated files (default: ".")
- `-lang`: Output language (go, cpp, java) (default: "go")
- `-registry`: Registry URL (default: "http://localhost:8080")
- `-recursive`: Pull dependencies recursively (default: false)

#### Validate Protobuf Files

Validate protobuf files:
```bash
spoke validate -dir ./proto
```

Options:
- `-dir`: Directory containing protobuf files (default: ".")
- `-registry`: Registry URL (default: "http://localhost:8080")
- `-recursive`: Validate dependencies recursively (default: false)

## Example Workflow

1. Create a new module:
```bash
# Create your protobuf files
mkdir -p proto
cat > proto/example.proto << EOF
syntax = "proto3";
package example;

message Hello {
  string message = 1;
}
EOF

# Push to registry
spoke push -module example -version v1.0.0 -dir ./proto
```

2. Pull and use the module:
```bash
# Pull the module
spoke pull -module example -version v1.0.0 -dir ./myproject/proto

# Compile to Go
spoke compile -dir ./myproject/proto -out ./myproject/generated -lang go
```

## API Endpoints

The server provides the following HTTP endpoints:

- `POST /modules` - Create a new module
- `GET /modules` - List all modules
- `GET /modules/{name}` - Get a specific module
- `POST /modules/{name}/versions` - Create a new version of a module
- `GET /modules/{name}/versions` - List all versions of a module
- `GET /modules/{name}/versions/{version}` - Get a specific version
- `GET /modules/{name}/versions/{version}/files/{path}` - Get a specific file from a version
- `POST /modules/{name}/versions/{version}/compile/{language}` - Compile a version for a specific language
- `GET /modules/{name}/versions/{version}/download/{language}`