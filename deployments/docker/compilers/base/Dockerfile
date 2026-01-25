# Base image with protoc compiler
# This provides the foundation for all language-specific compiler images
FROM ubuntu:22.04

ARG PROTOC_VERSION=25.1
ARG TARGETARCH

LABEL maintainer="Spoke Team"
LABEL description="Base image with Protocol Buffer compiler (protoc)"
LABEL version="${PROTOC_VERSION}"

# Install required packages
RUN apt-get update && apt-get install -y \
    curl \
    unzip \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Download and install protoc
RUN ARCH=${TARGETARCH:-amd64} && \
    if [ "$ARCH" = "amd64" ]; then PROTOC_ARCH="x86_64"; \
    elif [ "$ARCH" = "arm64" ]; then PROTOC_ARCH="aarch_64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -L "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${PROTOC_ARCH}.zip" -o protoc.zip && \
    unzip protoc.zip -d /usr/local && \
    rm protoc.zip && \
    chmod +x /usr/local/bin/protoc

# Create directories for proto files and output
RUN mkdir -p /input /output

# Verify installation
RUN protoc --version

WORKDIR /workspace
