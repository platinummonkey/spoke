# Code Generation Guide

This guide covers Spoke's advanced code generation system with support for 15+ programming languages.

## Table of Contents

- [Overview](#overview)
- [Supported Languages](#supported-languages)
- [Quick Start](#quick-start)
- [CLI Usage](#cli-usage)
- [API Usage](#api-usage)
- [Advanced Features](#advanced-features)
- [Performance](#performance)
- [Troubleshooting](#troubleshooting)

## Overview

Spoke's code generation system automatically compiles Protocol Buffer definitions into language-specific code. Key features:

- **15+ Languages**: Go, Python, Java, C++, C#, Rust, TypeScript, JavaScript, Dart, Swift, Kotlin, Objective-C, Ruby, PHP, Scala
- **gRPC Support**: Automatic gRPC stub generation for all languages
- **Multi-Level Caching**: 80%+ cache hit rate for blazing-fast repeat compilations
- **Parallel Compilation**: Compile for multiple languages simultaneously
- **Docker Isolation**: Each language compiles in its own isolated Docker container
- **Package Managers**: Auto-generates go.mod, setup.py, package.json, pom.xml, etc.

## Supported Languages

| Language | Plugin Version | gRPC | Package Manager | File Extensions |
|----------|----------------|------|-----------------|-----------------|
| Go | v1.31.0 | ✓ | go-modules | `.pb.go` |
| Python | 4.24.0 | ✓ | pip | `_pb2.py` |
| Java | 3.21.0 | ✓ | maven | `.java` |
| C++ | 3.21.0 | ✓ | - | `.pb.h`, `.pb.cc` |
| C# | 3.21.0 | ✓ | nuget | `.cs` |
| Rust | 3.2.0 | ✓ | cargo | `.rs` |
| TypeScript | 5.0.1 | ✓ | npm | `.ts` |
| JavaScript | 3.21.0 | ✓ | npm | `_pb.js` |
| Dart | 3.1.0 | ✓ | pub | `.pb.dart` |
| Swift | 1.25.0 | ✓ | swift-package | `.pb.swift` |
| Kotlin | 3.21.0 | ✓ | gradle | `.kt` |
| Objective-C | 3.21.0 | ✓ | cocoapods | `.pbobjc.h` |
| Ruby | 3.21.0 | ✓ | gem | `_pb.rb` |
| PHP | 3.21.0 | ✓ | composer | `.php` |
| Scala | 0.11.13 | ✓ | sbt | `.scala` |

## Quick Start

### 1. Push your proto files to Spoke

```bash
spoke push -module user-service -version v1.0.0 -dir ./proto
```

### 2. Compile for your language

```bash
# Single language
spoke compile --languages go --grpc --dir ./proto --out ./gen

# Multiple languages
spoke compile --languages go,python,java --grpc --parallel --dir ./proto --out ./gen
```

### 3. Download pre-compiled artifacts

```bash
# Via API
curl http://localhost:8080/modules/user-service/versions/v1.0.0/download/go > user-service-go.tar.gz
```

## CLI Usage

### List Available Languages

```bash
# Table view
spoke languages list

# JSON output
spoke languages list --json

# Get details for specific language
spoke languages show rust
```

**Example Output:**
```
ID          NAME            VERSION    GRPC   STABLE   FILE EXTENSIONS
──          ────            ───────    ────   ──────   ───────────────
go          Go              v1.31.0    ✓      ✓        .pb.go
python      Python          4.24.0     ✓      ✓        _pb2.py, ...
rust        Rust            3.2.0      ✓      ✓        .rs
...

Total: 15 languages
```

### Compile Command

#### Basic Compilation

```bash
# Single language
spoke compile --languages go --dir ./proto --out ./gen

# Multiple languages
spoke compile --languages go,python,java --dir ./proto --out ./gen
```

#### With gRPC

```bash
spoke compile --languages go,python --grpc --dir ./proto --out ./gen
```

#### Parallel Compilation

```bash
# Compile 5 languages in parallel
spoke compile --languages go,python,java,rust,typescript --parallel --grpc --dir ./proto
```

#### Output Structure

```
gen/
├── go/
│   ├── user.pb.go
│   ├── user_grpc.pb.go
│   └── go.mod
├── python/
│   ├── user_pb2.py
│   ├── user_pb2_grpc.py
│   └── setup.py
└── java/
    ├── User.java
    └── pom.xml
```

### Advanced Options

```bash
# Custom proto path
spoke compile --languages go --dir ./proto --proto-path ./common --out ./gen

# Recursive dependencies
spoke compile --languages go --recursive --dir ./proto --out ./gen

# Custom registry URL
spoke compile --languages go --registry https://spoke.company.com --dir ./proto
```

## API Usage

### List Languages

**Request:**
```bash
GET /api/v1/languages
```

**Response:**
```json
[
  {
    "id": "go",
    "name": "Go",
    "display_name": "Go (Protocol Buffers)",
    "supports_grpc": true,
    "file_extensions": [".pb.go"],
    "enabled": true,
    "stable": true,
    "description": "Go language support with protoc-gen-go",
    "documentation_url": "https://protobuf.dev/reference/go/go-generated/",
    "plugin_version": "v1.31.0",
    "package_manager": {
      "name": "go-modules",
      "config_files": ["go.mod"]
    }
  },
  ...
]
```

### Get Language Details

**Request:**
```bash
GET /api/v1/languages/rust
```

**Response:**
```json
{
  "id": "rust",
  "name": "Rust",
  "display_name": "Rust (Protocol Buffers)",
  "supports_grpc": true,
  "file_extensions": [".rs"],
  "enabled": true,
  "stable": true,
  "description": "Rust language support with prost and tonic",
  "documentation_url": "https://github.com/tokio-rs/prost",
  "plugin_version": "3.2.0",
  "package_manager": {
    "name": "cargo",
    "config_files": ["Cargo.toml"]
  }
}
```

### Trigger Compilation

**Request:**
```bash
POST /api/v1/modules/user-service/versions/v1.0.0/compile
Content-Type: application/json

{
  "languages": ["go", "python", "rust"],
  "include_grpc": true,
  "options": {
    "go_package": "github.com/company/user-service"
  }
}
```

**Response:**
```json
{
  "job_id": "user-service-v1.0.0",
  "results": [
    {
      "id": "user-service-v1.0.0-go",
      "language": "go",
      "status": "completed",
      "started_at": "2026-01-25T10:00:00Z",
      "completed_at": "2026-01-25T10:00:02Z",
      "duration_ms": 1850,
      "cache_hit": false,
      "s3_key": "compiled/user-service/v1.0.0/go.tar.gz",
      "s3_bucket": "spoke-artifacts"
    },
    {
      "id": "user-service-v1.0.0-python",
      "language": "python",
      "status": "completed",
      "duration_ms": 2100,
      "cache_hit": false,
      "s3_key": "compiled/user-service/v1.0.0/python.tar.gz"
    },
    {
      "id": "user-service-v1.0.0-rust",
      "language": "rust",
      "status": "completed",
      "duration_ms": 3200,
      "cache_hit": false,
      "s3_key": "compiled/user-service/v1.0.0/rust.tar.gz"
    }
  ]
}
```

### Get Compilation Job Status

**Request:**
```bash
GET /api/v1/modules/user-service/versions/v1.0.0/compile/user-service-v1.0.0-go
```

**Response:**
```json
{
  "id": "user-service-v1.0.0-go",
  "language": "go",
  "status": "completed",
  "started_at": "2026-01-25T10:00:00Z",
  "completed_at": "2026-01-25T10:00:02Z",
  "duration_ms": 1850,
  "cache_hit": false,
  "s3_key": "compiled/user-service/v1.0.0/go.tar.gz",
  "s3_bucket": "spoke-artifacts"
}
```

## Advanced Features

### Caching

Spoke uses a multi-level caching system for maximum performance:

**L1 Cache (Memory):**
- 10MB maximum size
- 5-minute TTL
- Sub-millisecond access time

**L2 Cache (Redis):**
- 24-hour TTL
- <10ms access time
- Shared across all server instances

**L3 Cache (S3):**
- Permanent storage
- ~100ms access time
- Content-addressable (deduplication)

**Cache Key Generation:**

Cache keys are computed from:
- Proto file content (SHA256)
- Dependency versions
- Plugin versions
- Compilation options

This ensures that identical inputs always hit the cache, while any change invalidates it.

**Performance:**
- First compilation: ~2s
- Cached compilation: ~200ms (10x faster!)
- Target cache hit rate: >80%

### Parallel Compilation

Compile multiple languages simultaneously using a worker pool:

```bash
# Compile 5 languages in parallel (default: 5 workers)
spoke compile --languages go,python,java,rust,typescript --parallel --grpc
```

**Configuration:**
```yaml
# Server configuration
max_parallel_workers: 5
compilation_timeout: 300  # seconds
```

**Performance:**
- Sequential: 5 languages × 2s = 10s
- Parallel (5 workers): ~3s (3.3x faster!)

### Package Manager Integration

Spoke automatically generates package manager configuration files:

**Go (go.mod):**
```go
module github.com/company/user-service

go 1.21

require (
    google.golang.org/protobuf v1.31.0
    google.golang.org/grpc v1.60.0
)
```

**Python (setup.py):**
```python
setup(
    name='user-service',
    version='1.0.0',
    install_requires=[
        'protobuf==4.24.0',
        'grpcio==1.59.0',
    ],
)
```

**TypeScript (package.json):**
```json
{
  "name": "user-service",
  "version": "1.0.0",
  "dependencies": {
    "google-protobuf": "^3.21.0",
    "@grpc/grpc-js": "^1.9.0"
  }
}
```

### Docker Isolation

Each language compiles in its own Docker container with:
- **Memory limit**: 512MB
- **CPU limit**: 1.0 core
- **Timeout**: 5 minutes

Benefits:
- Reproducible builds
- No local protoc installation required
- Isolated environments prevent conflicts
- Easy versioning via Docker images

## Performance

### Benchmarks

**Single Language Compilation:**
- Go: ~1.8s
- Python: ~2.1s
- Java: ~2.3s
- Rust: ~3.2s (larger toolchain)

**Parallel Compilation (5 languages):**
- Sequential: ~10s
- Parallel: ~3s (3.3x speedup)

**Caching:**
- Cache miss: ~2s
- Cache hit: ~200ms (10x faster)
- L1 hit: <1ms
- L2 hit: <10ms

**Cache Hit Rate:**
- Development: 60-70%
- CI/CD: 80-90%
- Production: >90%

### Optimization Tips

1. **Enable caching** - Set up Redis for L2 cache
2. **Use parallel compilation** - Compile multiple languages at once
3. **Pin versions** - Consistent versions = better cache hits
4. **Pre-compile** - Compile during CI/CD, not at runtime
5. **Use S3** - Store artifacts for long-term reuse

## Troubleshooting

### Docker Not Available

**Error:** `failed to connect to Docker daemon`

**Solution:**
```bash
# Check Docker is running
docker ps

# Start Docker daemon
sudo systemctl start docker
```

### Image Not Found

**Error:** `image not found: spoke/compiler-go:1.31.0`

**Solution:**
```bash
# Build images locally
cd deployments/docker
./build-all.sh

# Or pull from registry
docker pull spoke/compiler-go:1.31.0
```

### Compilation Timeout

**Error:** `compilation exceeded timeout: 5m0s`

**Solution:**
```yaml
# Increase timeout in config
compilation_timeout: 600  # 10 minutes
```

### Cache Not Working

**Issue:** Every compilation is slow (cache misses)

**Checklist:**
- [ ] Redis is running and accessible
- [ ] Cache is enabled in configuration
- [ ] Proto files haven't changed
- [ ] Plugin versions are consistent

**Debug:**
```bash
# Check Redis connection
redis-cli ping

# Monitor cache hits
curl http://localhost:8080/api/v1/cache/stats
```

### Out of Memory

**Error:** `docker: OOM killed`

**Solution:**
```yaml
# Increase Docker memory limit
docker_memory_limit: 1024  # 1GB
```

Or compile fewer languages in parallel:
```yaml
max_parallel_workers: 3  # Reduce from 5
```

### Language Not Supported

**Error:** `language xyz not found`

**Solution:**
```bash
# List available languages
spoke languages list

# Check if language is enabled
spoke languages show xyz
```

## Best Practices

### 1. Version Pinning

Always specify exact versions for reproducibility:

```bash
spoke push -module user-service -version v1.0.0  # Not "latest"
```

### 2. Pre-Compilation in CI/CD

Compile during build, not at runtime:

```yaml
# .github/workflows/compile.yml
- name: Compile protos
  run: |
    spoke compile --languages go,python --grpc --dir ./proto
    spoke push -module $SERVICE -version $VERSION
```

### 3. Cache Warming

Prime the cache during deployment:

```bash
# Warm cache for common languages
for lang in go python java; do
  spoke compile --languages $lang --dir ./proto
done
```

### 4. Monitoring

Track compilation metrics:
- Cache hit rate
- Compilation duration
- Error rates
- S3 storage usage

### 5. Graceful Degradation

Handle compilation failures gracefully:

```python
try:
    compile_all(languages=['go', 'python', 'java'])
except CompilationError:
    # Fall back to essential languages
    compile_all(languages=['go'])
```

## Additional Resources

- [Spoke Architecture](./ARCHITECTURE.md)
- [Docker Images](../deployments/docker/compilers/README.md)
- [API Reference](./API_REFERENCE.md)
- [Protocol Buffers Style Guide](https://protobuf.dev/programming-guides/style/)
- [gRPC Documentation](https://grpc.io/docs/)

## Support

- **Issues**: https://github.com/platinummonkey/spoke/issues
- **Discussions**: https://github.com/platinummonkey/spoke/discussions
- **Slack**: #spoke-support
