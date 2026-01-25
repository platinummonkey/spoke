# C# protobuf compiler image
FROM spoke/compiler-base:latest

# Install .NET SDK
RUN wget https://packages.microsoft.com/config/debian/11/packages-microsoft-prod.deb -O packages-microsoft-prod.deb && \
    dpkg -i packages-microsoft-prod.deb && \
    rm packages-microsoft-prod.deb && \
    apt-get update && \
    apt-get install -y dotnet-sdk-7.0 && \
    rm -rf /var/lib/apt/lists/*

# Install Grpc.Tools for protoc plugins
RUN dotnet tool install --global Grpc.Tools --version 2.59.0

# Add dotnet tools to PATH
ENV PATH="/root/.dotnet/tools:${PATH}"

# Verify installation
RUN protoc --version && \
    which grpc_csharp_plugin

WORKDIR /workspace

CMD ["protoc", "--version"]
