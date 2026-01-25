---
title: "Quick Start"
weight: 2
---

# Quick Start Guide

Get up and running with Spoke in 5 minutes.

## Prerequisites

- Go 1.16 or later
- Protocol Buffers compiler (`protoc`)
- Git

## Step 1: Install Spoke

Clone and build Spoke:

```bash
# Clone the repository
git clone https://github.com/platinummonkey/spoke.git
cd spoke

# Build the server and CLI
go build -o spoke ./cmd/spoke/main.go
go build -o spoke-cli ./cmd/spoke-cli/main.go

# Add to PATH (optional)
export PATH=$PATH:$(pwd)
```

## Step 2: Start the Server

Start the Spoke server:

```bash
./spoke -port 8080 -storage-dir ./storage
```

You should see output like:

```
Starting Spoke server on :8080
Storage directory: ./storage
Server ready and listening...
```

The server is now running at `http://localhost:8080`.

## Step 3: Create Your First Protobuf Module

Create a simple protobuf definition:

```bash
# Create a directory for your proto files
mkdir -p examples/hello/proto

# Create a simple proto file
cat > examples/hello/proto/hello.proto << 'EOF'
syntax = "proto3";

package hello;

option go_package = "github.com/example/hello;hello";

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string message = 1;
  int64 timestamp = 2;
}

service Greeter {
  rpc SayHello(HelloRequest) returns (HelloResponse);
}
EOF
```

## Step 4: Push to Registry

Push your module to the Spoke registry:

```bash
./spoke-cli push \
  -module hello \
  -version v1.0.0 \
  -dir examples/hello/proto \
  -registry http://localhost:8080 \
  -description "Hello world gRPC service"
```

You should see:

```
Successfully pushed module 'hello' version 'v1.0.0'
Files uploaded: 1
```

## Step 5: Pull from Registry

Pull the module from the registry to a different location:

```bash
# Create output directory
mkdir -p output/hello

# Pull the module
./spoke-cli pull \
  -module hello \
  -version v1.0.0 \
  -dir output/hello \
  -registry http://localhost:8080
```

You should see:

```
Successfully pulled module 'hello' version 'v1.0.0'
Files downloaded: 1
```

Verify the file was downloaded:

```bash
cat output/hello/hello.proto
```

## Step 6: Compile to Go

Compile the protobuf files to Go code:

```bash
./spoke-cli compile \
  -dir output/hello \
  -out output/generated/go \
  -lang go
```

You should see generated `.pb.go` files in `output/generated/go/`.

## Step 7: Explore the Web UI (Optional)

If the web UI is enabled, open your browser to:

```
http://localhost:8080
```

You should see the Spoke web interface where you can browse modules, versions, and view proto files.

## What's Next?

Now that you have Spoke running, you can:

### Create More Complex Modules

```bash
# Module with dependencies
mkdir -p proto/common
mkdir -p proto/user

# Create common types
cat > proto/common/types.proto << 'EOF'
syntax = "proto3";
package common;

message UUID {
  string value = 1;
}

message Timestamp {
  int64 seconds = 1;
  int32 nanos = 2;
}
EOF

# Create user service that imports common types
cat > proto/user/user.proto << 'EOF'
syntax = "proto3";
package user;

import "common/types.proto";

message User {
  common.UUID id = 1;
  string email = 2;
  common.Timestamp created_at = 3;
}
EOF

# Push common module first
./spoke-cli push -module common -version v1.0.0 -dir proto/common

# Push user module
./spoke-cli push -module user -version v1.0.0 -dir proto/user

# Pull with dependencies
./spoke-cli pull -module user -version v1.0.0 -dir output/user -recursive
```

### Version Your Schemas

```bash
# Make changes to your proto file
echo 'message GoodbyeRequest { string name = 1; }' >> examples/hello/proto/hello.proto

# Push new version
./spoke-cli push -module hello -version v1.1.0 -dir examples/hello/proto
```

### Compile to Different Languages

```bash
# Python
./spoke-cli compile -dir output/hello -out output/generated/python -lang python

# Java
./spoke-cli compile -dir output/hello -out output/generated/java -lang java

# C++
./spoke-cli compile -dir output/hello -out output/generated/cpp -lang cpp
```

### Validate Schemas

```bash
# Validate proto files before pushing
./spoke-cli validate -dir examples/hello/proto
```

## Common Commands

Here are the most commonly used Spoke commands:

```bash
# Push a module
spoke-cli push -module <name> -version <version> -dir <path>

# Pull a module
spoke-cli pull -module <name> -version <version> -dir <path>

# Pull with dependencies
spoke-cli pull -module <name> -version <version> -dir <path> -recursive

# Compile proto files
spoke-cli compile -dir <input> -out <output> -lang <language>

# Validate proto files
spoke-cli validate -dir <path>

# List all modules
curl http://localhost:8080/modules

# Get module versions
curl http://localhost:8080/modules/<name>/versions
```

## Troubleshooting

### Server won't start

**Problem**: Port already in use

**Solution**: Use a different port or stop the process using port 8080:

```bash
# Use different port
./spoke -port 8081 -storage-dir ./storage

# Find and kill process on port 8080 (macOS/Linux)
lsof -ti:8080 | xargs kill -9
```

### Push fails

**Problem**: Cannot connect to registry

**Solution**: Ensure the server is running and the registry URL is correct:

```bash
# Check if server is running
curl http://localhost:8080/health

# Use correct registry URL
./spoke-cli push -module hello -version v1.0.0 -dir ./proto -registry http://localhost:8080
```

### Compilation fails

**Problem**: protoc not found

**Solution**: Install Protocol Buffers compiler:

```bash
# macOS
brew install protobuf

# Ubuntu/Debian
apt-get install protobuf-compiler

# Verify installation
protoc --version
```

**Problem**: Language plugin not found (e.g., protoc-gen-go)

**Solution**: Install the language-specific plugin:

```bash
# Go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Python
pip install grpcio-tools

# Ensure plugins are in PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

## Next Steps

- [Installation Guide](/getting-started/installation/) - Detailed installation instructions
- [First Module Tutorial](/getting-started/first-module/) - Deep dive into module creation
- [CLI Reference](/guides/cli-reference/) - Complete CLI documentation
- [API Reference](/api/rest-api/) - HTTP API documentation
