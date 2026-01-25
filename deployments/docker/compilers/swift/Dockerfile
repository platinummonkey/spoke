# Swift protobuf compiler image
FROM spoke/compiler-base:latest

# Install Swift
RUN apt-get update && \
    apt-get install -y \
    binutils \
    git \
    gnupg2 \
    libc6-dev \
    libcurl4-openssl-dev \
    libedit2 \
    libgcc-9-dev \
    libpython3.9 \
    libsqlite3-0 \
    libstdc++-9-dev \
    libxml2-dev \
    libz3-dev \
    pkg-config \
    tzdata \
    unzip \
    zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

# Download and install Swift
RUN wget https://download.swift.org/swift-5.9-release/ubuntu2004/swift-5.9-RELEASE/swift-5.9-RELEASE-ubuntu20.04.tar.gz && \
    tar xzf swift-5.9-RELEASE-ubuntu20.04.tar.gz && \
    mv swift-5.9-RELEASE-ubuntu20.04 /usr/share/swift && \
    rm swift-5.9-RELEASE-ubuntu20.04.tar.gz

ENV PATH="/usr/share/swift/usr/bin:${PATH}"

# Install protoc-gen-swift
RUN git clone https://github.com/apple/swift-protobuf.git /tmp/swift-protobuf && \
    cd /tmp/swift-protobuf && \
    git checkout 1.25.0 && \
    swift build -c release && \
    cp .build/release/protoc-gen-swift /usr/local/bin/ && \
    rm -rf /tmp/swift-protobuf

# Install grpc-swift plugin
RUN git clone https://github.com/grpc/grpc-swift.git /tmp/grpc-swift && \
    cd /tmp/grpc-swift && \
    git checkout 1.21.0 && \
    swift build -c release --product protoc-gen-grpc-swift && \
    cp .build/release/protoc-gen-grpc-swift /usr/local/bin/ && \
    rm -rf /tmp/grpc-swift

# Verify installation
RUN protoc --version && \
    swift --version && \
    which protoc-gen-swift && \
    which protoc-gen-grpc-swift

WORKDIR /workspace

CMD ["protoc", "--version"]
