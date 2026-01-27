---
title: "Installation"
weight: 3
---

# Installation Guide

This guide covers detailed installation instructions for Spoke server, CLI, and Sprocket compilation service.

## System Requirements

### Server Requirements
- **Operating System**: Linux, macOS, or Windows
- **Go**: 1.21 or later (for building from source)
- **Memory**: Minimum 512MB RAM, 2GB+ recommended for production
- **Storage**: Varies based on number of modules (10GB+ recommended)
- **Network**: Port 8080 (default) or custom port

### Client Requirements
- **Go**: 1.21+ (for building CLI from source)
- **protoc**: Protocol Buffers compiler
- **Language Plugins**: Depends on target languages

## Installing from Source

### 1. Clone the Repository

```bash
git clone https://github.com/platinummonkey/spoke.git
cd spoke
```

### 2. Build All Components

```bash
# Build everything
make build

# Or build individual components
make build-server  # Spoke server
make build-cli     # Spoke CLI
make build-sprocket # Sprocket compilation service
```

Binaries will be created in the project root:
- `spoke` - Server
- `spoke-cli` - CLI
- `sprocket` - Compilation service

### 3. Install to System Path (Optional)

```bash
# Copy binaries to /usr/local/bin
sudo cp spoke spoke-cli sprocket /usr/local/bin/

# Or add to PATH
export PATH=$PATH:$(pwd)
```

## Installing Protocol Buffers Compiler

### macOS

```bash
# Using Homebrew
brew install protobuf

# Verify installation
protoc --version
# Should show: libprotoc 3.x.x or later
```

### Ubuntu/Debian

```bash
# Install from package manager
sudo apt-get update
sudo apt-get install -y protobuf-compiler

# Verify installation
protoc --version
```

### CentOS/RHEL

```bash
# Install from package manager
sudo yum install -y protobuf-compiler

# Or install from GitHub releases
PROTOC_VERSION=3.20.0
wget https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
unzip protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local
```

### Windows

Download the latest release from [Protocol Buffers Releases](https://github.com/protocolbuffers/protobuf/releases) and add to PATH.

## Installing Language-Specific Plugins

### Go

```bash
# Install Go protobuf plugin
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Install Go gRPC plugin
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure Go bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Verify installation
which protoc-gen-go
which protoc-gen-go-grpc
```

### Python

```bash
# Install Python gRPC tools
pip install grpcio-tools

# Verify installation
python -m grpc_tools.protoc --version
```

### Java

```bash
# protoc-gen-java is included with protoc
# No additional installation needed

# For gRPC Java, add to your build.gradle or pom.xml
# See: https://github.com/grpc/grpc-java
```

### C++

```bash
# protoc-gen-cpp is included with protoc
# No additional installation needed

# On Ubuntu/Debian
sudo apt-get install -y libprotobuf-dev protobuf-compiler
```

### JavaScript/TypeScript

```bash
# Install protoc-gen-js
npm install -g protoc-gen-js

# For TypeScript
npm install -g protoc-gen-ts
```

## Configuration

### Server Configuration

Create a configuration file (optional):

```bash
# Create config directory
mkdir -p /etc/spoke

# Create config file
cat > /etc/spoke/config.yaml << 'EOF'
server:
  port: 8080
  host: 0.0.0.0

storage:
  type: filesystem
  directory: /var/lib/spoke/storage

database:
  driver: postgres
  host: localhost
  port: 5432
  database: spoke
  user: spoke
  password: your-password
  ssl_mode: disable

auth:
  jwt_secret: your-secret-key
  session_timeout: 24h

logging:
  level: info
  format: json
  output: /var/log/spoke/server.log

metrics:
  enabled: true
  port: 9090

sso:
  enabled: false
  # See SSO configuration guide

rbac:
  enabled: false
  # See RBAC configuration guide
EOF
```

### CLI Configuration

Create CLI config in your home directory:

```bash
mkdir -p ~/.spoke

cat > ~/.spoke/config.yaml << 'EOF'
default_registry: http://localhost:8080
timeout: 30s
retry_attempts: 3
EOF
```

## Running Spoke

### Development Mode

```bash
# Start server with default settings
./spoke -port 8080 -storage-dir ./storage

# Start with verbose logging
./spoke -port 8080 -storage-dir ./storage -log-level debug
```

### Production Mode

See the [Deployment Guide](/deployment/) for production deployment options including:
- Docker deployment
- Kubernetes deployment
- Systemd service
- High availability setup

## Verifying Installation

### 1. Check Server

```bash
# Start the server
./spoke -port 8080 -storage-dir ./storage &

# Wait a few seconds, then test
curl http://localhost:8080/health

# Expected output:
# {"status":"ok","timestamp":"2025-01-01T00:00:00Z"}
```

### 2. Check CLI

```bash
# Test CLI connection
./spoke-cli push --help

# Should show usage information
```

### 3. Test Complete Workflow

```bash
# Create test proto file
mkdir -p test/proto
cat > test/proto/test.proto << 'EOF'
syntax = "proto3";
package test;

message TestMessage {
  string content = 1;
}
EOF

# Push to registry
./spoke-cli push -module test -version v1.0.0 -dir test/proto

# Pull from registry
mkdir -p test/output
./spoke-cli pull -module test -version v1.0.0 -dir test/output

# Verify file exists
cat test/output/test.proto

# Cleanup
rm -rf test
```

## Optional: Installing Sprocket

Sprocket is an optional background service that automatically compiles uploaded protobuf files.

### Running Sprocket

```bash
# Start Sprocket (requires same storage directory as server)
./sprocket -storage-dir ./storage -delay 10s

# Run in background
./sprocket -storage-dir ./storage -delay 10s &
```

### Sprocket Configuration

```yaml
# /etc/spoke/sprocket.yaml
storage:
  directory: /var/lib/spoke/storage

compilation:
  delay: 10s
  languages:
    - go
    - python
    - java
    - cpp

concurrency:
  workers: 4

logging:
  level: info
```

## Database Setup (Optional)

Spoke can use PostgreSQL for metadata storage.

### 1. Install PostgreSQL

```bash
# macOS
brew install postgresql

# Ubuntu/Debian
sudo apt-get install postgresql postgresql-contrib

# Start PostgreSQL
brew services start postgresql  # macOS
sudo systemctl start postgresql # Linux
```

### 2. Create Database

```sql
-- Connect to PostgreSQL
psql postgres

-- Create database and user
CREATE DATABASE spoke;
CREATE USER spoke WITH ENCRYPTED PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE spoke TO spoke;
```

### 3. Run Migrations

```bash
# Apply database migrations
./spoke migrate -db-url "postgres://spoke:your-password@localhost:5432/spoke?sslmode=disable"

# Or use migration tool
go run migrations/*.go
```

## Troubleshooting

### Build Errors

**Problem**: `go: command not found`

**Solution**: Install Go from [golang.org](https://golang.org/dl/)

**Problem**: Build fails with missing dependencies

**Solution**: Update dependencies and retry:

```bash
go mod tidy
go mod download
make build
```

### Runtime Errors

**Problem**: `permission denied` when creating storage directory

**Solution**: Ensure proper permissions:

```bash
# Create directory with proper permissions
sudo mkdir -p /var/lib/spoke/storage
sudo chown $USER:$USER /var/lib/spoke/storage

# Or use local directory
./spoke -port 8080 -storage-dir ./storage
```

**Problem**: `address already in use`

**Solution**: Port is already taken:

```bash
# Use different port
./spoke -port 8081 -storage-dir ./storage

# Or find and kill process
lsof -ti:8080 | xargs kill -9  # macOS/Linux
netstat -ano | findstr :8080   # Windows
```

### Compilation Errors

**Problem**: `protoc: command not found`

**Solution**: Install protoc as shown in [Installing Protocol Buffers Compiler](#installing-protocol-buffers-compiler)

**Problem**: `protoc-gen-go: program not found or is not executable`

**Solution**: Install plugin and ensure it's in PATH:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
export PATH=$PATH:$(go env GOPATH)/bin
```

## Next Steps

- [Quick Start Guide](/getting-started/quick-start/) - Create your first module
- [First Module Tutorial](/getting-started/first-module/) - Detailed tutorial
- [CLI Reference](/guides/cli-reference/) - Complete CLI documentation
- [Deployment Guide](/deployment/) - Production deployment options
