# Dart protobuf compiler image
FROM spoke/compiler-base:latest

# Install Dart SDK
RUN apt-get update && \
    apt-get install -y apt-transport-https && \
    wget -qO- https://dl-ssl.google.com/linux/linux_signing_key.pub | gpg --dearmor -o /usr/share/keyrings/dart.gpg && \
    echo 'deb [signed-by=/usr/share/keyrings/dart.gpg arch=amd64] https://storage.googleapis.com/download.dartlang.org/linux/debian stable main' | tee /etc/apt/sources.list.d/dart_stable.list && \
    apt-get update && \
    apt-get install -y dart && \
    rm -rf /var/lib/apt/lists/*

ENV PATH="/usr/lib/dart/bin:${PATH}"

# Install protoc-gen-dart
RUN dart pub global activate protoc_plugin 21.1.2

ENV PATH="/root/.pub-cache/bin:${PATH}"

# Verify installation
RUN protoc --version && \
    dart --version && \
    which protoc-gen-dart

WORKDIR /workspace

CMD ["protoc", "--version"]
