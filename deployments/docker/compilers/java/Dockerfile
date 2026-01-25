# Java compiler image with protoc and protoc-gen-grpc-java
FROM eclipse-temurin:17-jdk-alpine AS builder

ARG PROTOC_GEN_GRPC_JAVA_VERSION=1.60.0

# Download protoc-gen-grpc-java plugin
RUN apk add --no-cache curl && \
    curl -L "https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/${PROTOC_GEN_GRPC_JAVA_VERSION}/protoc-gen-grpc-java-${PROTOC_GEN_GRPC_JAVA_VERSION}-linux-x86_64.exe" \
    -o /tmp/protoc-gen-grpc-java && \
    chmod +x /tmp/protoc-gen-grpc-java

# Final image
FROM ubuntu:22.04

ARG PROTOC_VERSION=25.1
ARG TARGETARCH

LABEL maintainer="Spoke Team"
LABEL description="Java compiler with protoc and protoc-gen-grpc-java"
LABEL protoc_gen_grpc_java_version="${PROTOC_GEN_GRPC_JAVA_VERSION}"

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    unzip \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install protoc
RUN ARCH=${TARGETARCH:-amd64} && \
    if [ "$ARCH" = "amd64" ]; then PROTOC_ARCH="x86_64"; \
    elif [ "$ARCH" = "arm64" ]; then PROTOC_ARCH="aarch_64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -L "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${PROTOC_ARCH}.zip" -o protoc.zip && \
    unzip protoc.zip -d /usr/local && \
    rm protoc.zip && \
    chmod +x /usr/local/bin/protoc

# Copy gRPC Java plugin from builder
COPY --from=builder /tmp/protoc-gen-grpc-java /usr/local/bin/
RUN chmod +x /usr/local/bin/protoc-gen-grpc-java

# Create directories
RUN mkdir -p /input /output

# Verify installation
RUN protoc --version && \
    protoc-gen-grpc-java --version || echo "protoc-gen-grpc-java installed"

WORKDIR /workspace
