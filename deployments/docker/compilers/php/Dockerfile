# PHP protobuf compiler image
FROM spoke/compiler-base:latest

# Install PHP
RUN apt-get update && \
    apt-get install -y \
    php \
    php-dev \
    php-pear \
    && rm -rf /var/lib/apt/lists/*

# Install PHP protobuf extension
RUN pecl install protobuf

# Install gRPC PHP extension
RUN pecl install grpc-1.59.0

# Install composer
RUN curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer

# Install grpc_php_plugin
RUN apt-get update && apt-get install -y \
    libgrpc++-dev \
    protobuf-compiler-grpc \
    && rm -rf /var/lib/apt/lists/*

# Verify installation
RUN protoc --version && \
    php --version && \
    which grpc_php_plugin

WORKDIR /workspace

CMD ["protoc", "--version"]
