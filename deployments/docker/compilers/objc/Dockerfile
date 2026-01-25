# Objective-C protobuf compiler image
FROM spoke/compiler-base:latest

# Install necessary build tools
RUN apt-get update && \
    apt-get install -y \
    clang \
    libc++-dev \
    libc++abi-dev \
    && rm -rf /var/lib/apt/lists/*

# Install gRPC Objective-C plugin
# Note: The plugin is built into protoc for objc
RUN apt-get update && apt-get install -y \
    libgrpc++-dev \
    protobuf-compiler-grpc \
    && rm -rf /var/lib/apt/lists/*

# Verify installation
RUN protoc --version && \
    which grpc_objective_c_plugin

WORKDIR /workspace

CMD ["protoc", "--version"]
