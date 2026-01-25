# Python compiler image with protobuf and grpcio-tools
FROM python:3.11-slim

ARG PROTOBUF_VERSION=4.25.1
ARG GRPCIO_VERSION=1.60.0
ARG PROTOC_VERSION=25.1
ARG TARGETARCH

LABEL maintainer="Spoke Team"
LABEL description="Python compiler with protobuf and grpcio-tools"
LABEL protobuf_version="${PROTOBUF_VERSION}"
LABEL grpcio_version="${GRPCIO_VERSION}"

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl \
    unzip \
    ca-certificates \
    build-essential \
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

# Install Python protobuf packages
RUN pip install --no-cache-dir \
    protobuf==${PROTOBUF_VERSION} \
    grpcio==${GRPCIO_VERSION} \
    grpcio-tools==${GRPCIO_VERSION}

# Create directories
RUN mkdir -p /input /output

# Verify installation
RUN protoc --version && \
    python -c "import google.protobuf; print('protobuf version:', google.protobuf.__version__)" && \
    python -c "import grpc; print('grpc version:', grpc.__version__)"

WORKDIR /workspace
