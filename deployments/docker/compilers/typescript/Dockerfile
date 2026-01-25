# TypeScript protobuf compiler image
FROM spoke/compiler-base:latest

# Install Node.js and npm
RUN curl -fsSL https://deb.nodesource.com/setup_18.x | bash - && \
    apt-get install -y nodejs && \
    rm -rf /var/lib/apt/lists/*

# Install ts-proto
RUN npm install -g ts-proto@1.165.0

# Install grpc-web for gRPC support
RUN npm install -g grpc-web@1.4.2

# Verify installation
RUN protoc --version && \
    node --version && \
    npm --version

WORKDIR /workspace

CMD ["protoc", "--version"]
