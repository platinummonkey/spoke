# Kotlin protobuf compiler image
FROM spoke/compiler-base:latest

# Install Java (required for Kotlin)
RUN apt-get update && \
    apt-get install -y openjdk-17-jdk && \
    rm -rf /var/lib/apt/lists/*

# Install Kotlin compiler
RUN wget https://github.com/JetBrains/kotlin/releases/download/v1.9.20/kotlin-compiler-1.9.20.zip && \
    unzip kotlin-compiler-1.9.20.zip && \
    mv kotlinc /usr/local/kotlinc && \
    rm kotlin-compiler-1.9.20.zip

ENV PATH="/usr/local/kotlinc/bin:${PATH}"

# Download protoc-gen-kotlin plugin
RUN wget https://repo1.maven.org/maven2/com/google/protobuf/protoc-gen-kotlin/3.21.0/protoc-gen-kotlin-3.21.0-linux-x86_64.exe && \
    chmod +x protoc-gen-kotlin-3.21.0-linux-x86_64.exe && \
    mv protoc-gen-kotlin-3.21.0-linux-x86_64.exe /usr/local/bin/protoc-gen-kotlin

# Download grpc-kotlin plugin
RUN wget https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-kotlin/1.4.0/protoc-gen-grpc-kotlin-1.4.0-linux-x86_64.exe && \
    chmod +x protoc-gen-grpc-kotlin-1.4.0-linux-x86_64.exe && \
    mv protoc-gen-grpc-kotlin-1.4.0-linux-x86_64.exe /usr/local/bin/protoc-gen-grpc-kotlin

# Verify installation
RUN protoc --version && \
    kotlinc -version && \
    which protoc-gen-kotlin && \
    which protoc-gen-grpc-kotlin

WORKDIR /workspace

CMD ["protoc", "--version"]
