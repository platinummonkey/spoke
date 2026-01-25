# Spoke Compiler Docker Images

This directory contains Docker images for compiling Protocol Buffer definitions to various programming languages.

## Architecture

All compiler images follow a consistent pattern:

1. **Base Image** (`base/`): Contains protoc compiler only
2. **Language Images**: Extend base or standalone, include language-specific plugins

## Available Images (15 Languages)

### Base
- **Image**: `spoke/compiler-base:25.1`
- **Contains**: protoc 25.1
- **Usage**: Foundation for custom compiler images

### Go
- **Image**: `spoke/compiler-go:1.31.0`
- **Contains**: protoc + protoc-gen-go v1.31.0 + protoc-gen-go-grpc v1.3.0
- **Generates**: `.pb.go` and `_grpc.pb.go` files

### Python
- **Image**: `spoke/compiler-python:4.24.0`
- **Contains**: protoc + protobuf 4.24.0 + grpcio-tools 1.59.0
- **Generates**: `_pb2.py` and `_pb2_grpc.py` files

### Java
- **Image**: `spoke/compiler-java:3.21.0`
- **Contains**: protoc + protoc-gen-grpc-java 1.59.0
- **Generates**: `.java` files

### C++
- **Image**: `spoke/compiler-cpp:3.21.0`
- **Contains**: protoc + g++ + grpc_cpp_plugin
- **Generates**: `.pb.h` and `.pb.cc` files

### C#
- **Image**: `spoke/compiler-csharp:3.21.0`
- **Contains**: protoc + .NET SDK + Grpc.Tools
- **Generates**: `.cs` files

### Rust
- **Image**: `spoke/compiler-rust:3.2.0`
- **Contains**: protoc + Rust + prost + tonic
- **Generates**: `.rs` files

### TypeScript
- **Image**: `spoke/compiler-typescript:5.0.1`
- **Contains**: protoc + Node.js + ts-proto
- **Generates**: `.ts` files

### JavaScript
- **Image**: `spoke/compiler-javascript:3.21.0`
- **Contains**: protoc + Node.js + protobufjs
- **Generates**: `_pb.js` files

### Dart
- **Image**: `spoke/compiler-dart:3.1.0`
- **Contains**: protoc + Dart SDK + protoc-gen-dart
- **Generates**: `.pb.dart` files

### Swift
- **Image**: `spoke/compiler-swift:1.25.0`
- **Contains**: protoc + Swift + swift-protobuf + grpc-swift
- **Generates**: `.pb.swift` and `.grpc.swift` files

### Kotlin
- **Image**: `spoke/compiler-kotlin:3.21.0`
- **Contains**: protoc + Kotlin + protoc-gen-kotlin
- **Generates**: `.kt` files

### Objective-C
- **Image**: `spoke/compiler-objc:3.21.0`
- **Contains**: protoc + Clang + grpc_objective_c_plugin
- **Generates**: `.pbobjc.h` and `.pbobjc.m` files

### Ruby
- **Image**: `spoke/compiler-ruby:3.21.0`
- **Contains**: protoc + Ruby + grpc-tools gem
- **Generates**: `_pb.rb` files

### PHP
- **Image**: `spoke/compiler-php:3.21.0`
- **Contains**: protoc + PHP + grpc extension
- **Generates**: `.php` files

### Scala
- **Image**: `spoke/compiler-scala:0.11.13`
- **Contains**: protoc + Scala + sbt + ScalaPB
- **Generates**: `.scala` files

## Building Images

Build all images:
```bash
cd deployments/docker/compilers
./build.sh
```

Build specific image:
```bash
docker build -t spoke/compiler-go:1.31.0 go/
docker build -t spoke/compiler-python:4.25.1 python/
docker build -t spoke/compiler-java:3.25.1 java/
```

## Usage

### Go Compilation
```bash
docker run --rm \
  -v /path/to/proto:/input:ro \
  -v /path/to/output:/output \
  spoke/compiler-go:1.31.0 \
  protoc --proto_path=/input \
         --go_out=/output \
         --go_opt=paths=source_relative \
         --go-grpc_out=/output \
         --go-grpc_opt=paths=source_relative \
         /input/*.proto
```

### Python Compilation
```bash
docker run --rm \
  -v /path/to/proto:/input:ro \
  -v /path/to/output:/output \
  spoke/compiler-python:4.25.1 \
  python -m grpc_tools.protoc \
         --proto_path=/input \
         --python_out=/output \
         --grpc_python_out=/output \
         /input/*.proto
```

### Java Compilation
```bash
docker run --rm \
  -v /path/to/proto:/input:ro \
  -v /path/to/output:/output \
  spoke/compiler-java:3.25.1 \
  protoc --proto_path=/input \
         --java_out=/output \
         --grpc-java_out=/output \
         /input/*.proto
```

## Resource Limits

Default resource limits for all containers:
- **Memory**: 512MB
- **CPU**: 1.0 core
- **Timeout**: 5 minutes

Apply limits:
```bash
docker run --rm \
  --memory=512m \
  --cpus=1.0 \
  -v /path/to/proto:/input:ro \
  -v /path/to/output:/output \
  spoke/compiler-go:1.31.0 \
  protoc ...
```

## Directory Structure

```
compilers/
├── base/
│   └── Dockerfile          # Base image with protoc only
├── go/
│   └── Dockerfile          # Go with protoc-gen-go + protoc-gen-go-grpc
├── python/
│   └── Dockerfile          # Python with grpcio-tools
├── java/
│   └── Dockerfile          # Java with protoc-gen-grpc-java
├── build.sh                # Build all images
└── README.md               # This file
```

## Adding New Languages

To add a new language compiler image:

1. Create a new directory: `compilers/{language}/`
2. Add `Dockerfile` with:
   - Base Ubuntu/Alpine image
   - Install protoc
   - Install language-specific protoc plugins
   - Verify installation
3. Update `build.sh` to include new image
4. Update language registry in `pkg/codegen/languages/defaults.go`
5. Add documentation to this README

Example template:
```dockerfile
FROM ubuntu:22.04

ARG PROTOC_VERSION=25.1
ARG PLUGIN_VERSION=1.0.0

# Install protoc
RUN curl -L "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip" -o protoc.zip && \
    unzip protoc.zip -d /usr/local && \
    rm protoc.zip

# Install language-specific plugin
RUN install-your-plugin-here

# Create directories
RUN mkdir -p /input /output

# Verify
RUN protoc --version && \
    your-plugin --version

WORKDIR /workspace
```

## Versioning

Images follow semantic versioning based on the primary plugin version:

- `spoke/compiler-go:1.31.0` - protoc-gen-go v1.31.0
- `spoke/compiler-python:4.25.1` - protobuf v4.25.1
- `spoke/compiler-java:3.25.1` - protoc v3.25.1 (built-in Java support)

Tag images with both specific versions and `latest`:
```bash
docker build -t spoke/compiler-go:1.31.0 -t spoke/compiler-go:latest go/
```

## Testing

Test each image:
```bash
# Create test proto file
cat > test.proto <<EOF
syntax = "proto3";
package test;

message TestMessage {
  string name = 1;
  int32 value = 2;
}
EOF

# Test Go compiler
docker run --rm -v $(pwd):/input -v $(pwd)/out:/output spoke/compiler-go:1.31.0 \
  protoc --proto_path=/input --go_out=/output --go_opt=paths=source_relative /input/test.proto

# Verify output
ls -la out/test.pb.go
```

## Troubleshooting

### Plugin not found
```
protoc-gen-go: program not found or is not executable
```
**Solution**: Ensure plugin is in `/usr/local/bin/` and has execute permissions

### Permission denied
```
cannot create /output/file.pb.go: Permission denied
```
**Solution**: Check output volume permissions, run with appropriate user:
```bash
docker run --user $(id -u):$(id -g) ...
```

### Architecture mismatch
```
exec format error
```
**Solution**: Build for correct architecture or use multi-arch builds:
```bash
docker buildx build --platform linux/amd64,linux/arm64 -t spoke/compiler-go:1.31.0 .
```

## CI/CD Integration

GitHub Actions workflow example:
```yaml
name: Build Compiler Images

on:
  push:
    paths:
      - 'deployments/docker/compilers/**'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build images
        run: |
          cd deployments/docker/compilers
          ./build.sh
      - name: Push to registry
        run: |
          echo "${{ secrets.DOCKER_PASSWORD }}" | docker login -u "${{ secrets.DOCKER_USERNAME }}" --password-stdin
          docker push spoke/compiler-go:1.31.0
          docker push spoke/compiler-python:4.25.1
          docker push spoke/compiler-java:3.25.1
```
