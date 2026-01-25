#!/bin/bash
# Build all Spoke compiler Docker images

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building Spoke compiler images..."

# Build base image
echo "Building base image..."
docker build -t spoke/compiler-base:25.1 -t spoke/compiler-base:latest base/

# Build Go compiler
echo "Building Go compiler..."
docker build -t spoke/compiler-go:1.31.0 -t spoke/compiler-go:latest go/

# Build Python compiler
echo "Building Python compiler..."
docker build -t spoke/compiler-python:4.25.1 -t spoke/compiler-python:latest python/

# Build Java compiler
echo "Building Java compiler..."
docker build -t spoke/compiler-java:3.25.1 -t spoke/compiler-java:latest java/

echo ""
echo "All images built successfully!"
echo ""
echo "Available images:"
docker images | grep spoke/compiler

echo ""
echo "To push to registry:"
echo "  docker push spoke/compiler-go:1.31.0"
echo "  docker push spoke/compiler-python:4.25.1"
echo "  docker push spoke/compiler-java:3.25.1"
