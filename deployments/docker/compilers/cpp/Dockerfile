# C++ protobuf compiler image
FROM spoke/compiler-base:latest

# Install C++ development tools
RUN apt-get update && apt-get install -y \
    g++ \
    cmake \
    make \
    && rm -rf /var/lib/apt/lists/*

# Install gRPC C++ plugin
RUN apt-get update && apt-get install -y \
    libgrpc++-dev \
    protobuf-compiler-grpc \
    && rm -rf /var/lib/apt/lists/*

# Verify installation
RUN protoc --version && \
    which grpc_cpp_plugin

WORKDIR /workspace

# Default command
CMD ["protoc", "--version"]
