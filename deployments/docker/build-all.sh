#!/bin/bash
# Build all Spoke compiler Docker images

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Building Spoke Compiler Docker Images${NC}"
echo "========================================"

# Build base image first
echo -e "\n${YELLOW}[1/16] Building base image...${NC}"
cd compilers/base
docker build -t spoke/compiler-base:latest .
echo -e "${GREEN}✓ Base image built${NC}"

# Array of languages to build
declare -a languages=(
    "go:1.31.0"
    "python:4.24.0"
    "java:3.21.0"
    "cpp:3.21.0"
    "csharp:3.21.0"
    "rust:3.2.0"
    "typescript:5.0.1"
    "javascript:3.21.0"
    "dart:3.1.0"
    "swift:1.25.0"
    "kotlin:3.21.0"
    "objc:3.21.0"
    "ruby:3.21.0"
    "php:3.21.0"
    "scala:0.11.13"
)

# Build each language image
count=2
total=$((${#languages[@]} + 1))

for lang_version in "${languages[@]}"; do
    IFS=':' read -r lang version <<< "$lang_version"
    echo -e "\n${YELLOW}[$count/$total] Building $lang image...${NC}"

    cd "../compilers/$lang"
    if [ -f Dockerfile ]; then
        docker build -t "spoke/compiler-$lang:$version" -t "spoke/compiler-$lang:latest" .
        echo -e "${GREEN}✓ $lang image built${NC}"
    else
        echo -e "${RED}✗ Dockerfile not found for $lang${NC}"
    fi

    ((count++))
done

echo -e "\n${GREEN}========================================"
echo -e "All images built successfully!${NC}"
echo ""
echo "Available images:"
docker images | grep spoke/compiler
