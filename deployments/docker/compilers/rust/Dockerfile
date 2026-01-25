# Rust protobuf compiler image
FROM spoke/compiler-base:latest

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Install protoc-gen-rust (prost)
RUN cargo install protoc-gen-prost --version 0.3.1

# Install protoc-gen-tonic for gRPC
RUN cargo install protoc-gen-tonic --version 0.4.0

# Verify installation
RUN protoc --version && \
    which protoc-gen-prost && \
    which protoc-gen-tonic

WORKDIR /workspace

CMD ["protoc", "--version"]
