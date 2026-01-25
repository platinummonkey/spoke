# Go compiler image with protoc-gen-go and protoc-gen-go-grpc
FROM golang:1.21-alpine AS builder

ARG PROTOC_GEN_GO_VERSION=v1.31.0
ARG PROTOC_GEN_GO_GRPC_VERSION=v1.3.0

# Install protoc-gen-go and protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION} && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}

# Final image
FROM ubuntu:22.04

ARG PROTOC_VERSION=25.1
ARG TARGETARCH

LABEL maintainer="Spoke Team"
LABEL description="Go compiler with protoc-gen-go and protoc-gen-go-grpc"
LABEL protoc_gen_go_version="${PROTOC_GEN_GO_VERSION}"
LABEL protoc_gen_go_grpc_version="${PROTOC_GEN_GO_GRPC_VERSION}"

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

# Copy Go plugins from builder
COPY --from=builder /go/bin/protoc-gen-go /usr/local/bin/
COPY --from=builder /go/bin/protoc-gen-go-grpc /usr/local/bin/
RUN chmod +x /usr/local/bin/protoc-gen-go /usr/local/bin/protoc-gen-go-grpc

# Create directories
RUN mkdir -p /input /output

# Verify installation
RUN protoc --version && \
    protoc-gen-go --version

WORKDIR /workspace
