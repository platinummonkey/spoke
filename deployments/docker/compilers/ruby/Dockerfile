# Ruby protobuf compiler image
FROM spoke/compiler-base:latest

# Install Ruby
RUN apt-get update && \
    apt-get install -y \
    ruby \
    ruby-dev \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install grpc-tools gem (includes protoc-gen-ruby and grpc_ruby_plugin)
RUN gem install grpc-tools -v 1.59.0

# Verify installation
RUN protoc --version && \
    ruby --version && \
    which grpc_ruby_plugin

WORKDIR /workspace

CMD ["protoc", "--version"]
