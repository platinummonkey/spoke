---
title: "CLI Reference"
weight: 1
---

# CLI Reference

Complete reference for the `spoke-cli` command-line tool.

## Installation

```bash
# Build from source
go build -o spoke-cli ./cmd/spoke-cli/main.go

# Add to PATH
export PATH=$PATH:$(pwd)
```

## Global Flags

All commands support these global flags:

| Flag | Description | Default |
|------|-------------|---------|
| `-registry` | Registry URL | `http://localhost:8080` |
| `-timeout` | Request timeout | `30s` |
| `-v, -verbose` | Verbose output | `false` |
| `-h, -help` | Show help | - |

## Configuration File

Create `~/.spoke/config.yaml`:

```yaml
default_registry: http://spoke.example.com
timeout: 60s
retry_attempts: 3
```

## Commands

### push

Push protobuf files to the registry.

#### Syntax

```bash
spoke-cli push -module <name> -version <version> -dir <path> [options]
```

#### Required Flags

| Flag | Description |
|------|-------------|
| `-module` | Module name |
| `-version` | Version (semantic version or commit hash) |
| `-dir` | Directory containing protobuf files |

#### Optional Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-description` | Module description | "" |
| `-registry` | Registry URL | `http://localhost:8080` |
| `-org` | Organization ID (multi-tenancy) | "" |

#### Examples

```bash
# Basic push
spoke-cli push -module user -version v1.0.0 -dir ./proto

# With description
spoke-cli push \
  -module user \
  -version v1.0.0 \
  -dir ./proto \
  -description "User service API"

# To specific registry
spoke-cli push \
  -module user \
  -version v1.0.0 \
  -dir ./proto \
  -registry https://spoke.company.com

# With organization (multi-tenancy)
spoke-cli push \
  -module user \
  -version v1.0.0 \
  -dir ./proto \
  -org my-org
```

#### Output

```
Successfully pushed module 'user' version 'v1.0.0'
Files uploaded: 3
Module URL: http://localhost:8080/modules/user/versions/v1.0.0
```

---

### pull

Pull protobuf files from the registry.

#### Syntax

```bash
spoke-cli pull -module <name> -version <version> -dir <path> [options]
```

#### Required Flags

| Flag | Description |
|------|-------------|
| `-module` | Module name |
| `-version` | Version to pull |
| `-dir` | Output directory |

#### Optional Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-recursive` | Pull dependencies recursively | `false` |
| `-registry` | Registry URL | `http://localhost:8080` |
| `-org` | Organization ID | "" |

#### Examples

```bash
# Basic pull
spoke-cli pull -module user -version v1.0.0 -dir ./proto

# With dependencies
spoke-cli pull \
  -module user \
  -version v1.0.0 \
  -dir ./proto \
  -recursive

# From specific registry
spoke-cli pull \
  -module user \
  -version v1.0.0 \
  -dir ./proto \
  -registry https://spoke.company.com
```

#### Output

```
Successfully pulled module 'user' version 'v1.0.0'
Files downloaded: 3
Output directory: ./proto
```

---

### compile

Compile protobuf files to target language.

#### Syntax

```bash
spoke-cli compile -dir <input> -out <output> -lang <language> [options]
```

#### Required Flags

| Flag | Description |
|------|-------------|
| `-dir` | Input directory with proto files |
| `-out` | Output directory for generated files |
| `-lang` | Target language |

#### Supported Languages

- `go` - Go
- `python` - Python
- `java` - Java
- `cpp` - C++
- `csharp` - C#
- `ruby` - Ruby
- `php` - PHP
- `javascript` - JavaScript
- `typescript` - TypeScript
- `objc` - Objective-C
- `swift` - Swift

#### Optional Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-grpc` | Generate gRPC code | `true` |
| `-import-path` | Additional import paths | `[]` |

#### Examples

```bash
# Compile to Go
spoke-cli compile -dir ./proto -out ./generated/go -lang go

# Compile to Python
spoke-cli compile -dir ./proto -out ./generated/python -lang python

# Compile without gRPC
spoke-cli compile \
  -dir ./proto \
  -out ./generated/go \
  -lang go \
  -grpc=false

# With additional import paths
spoke-cli compile \
  -dir ./proto \
  -out ./generated/go \
  -lang go \
  -import-path ./common \
  -import-path ./vendor
```

#### Output

```
Compiling proto files...
✓ user.proto
✓ types.proto
Generated 4 files in ./generated/go
```

---

### validate

Validate protobuf files.

#### Syntax

```bash
spoke-cli validate -dir <path> [options]
```

#### Required Flags

| Flag | Description |
|------|-------------|
| `-dir` | Directory containing proto files |

#### Optional Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-recursive` | Validate dependencies | `false` |

#### Examples

```bash
# Validate proto files
spoke-cli validate -dir ./proto

# Validate with dependencies
spoke-cli validate -dir ./proto -recursive
```

#### Output

```
Validating proto files in ./proto...
✓ user.proto is valid
✓ types.proto is valid
All files validated successfully
```

---

### list

List modules or versions.

#### Syntax

```bash
# List all modules
spoke-cli list -registry <url>

# List versions of a module
spoke-cli list -module <name> -registry <url>
```

#### Examples

```bash
# List all modules
spoke-cli list

# List versions of a module
spoke-cli list -module user

# From specific registry
spoke-cli list -registry https://spoke.company.com
```

#### Output

```
Available modules:
- user (3 versions)
- common (2 versions)
- order (1 version)

Total: 3 modules
```

---

### download

Download pre-compiled code.

#### Syntax

```bash
spoke-cli download -module <name> -version <version> -lang <language> -out <path>
```

#### Required Flags

| Flag | Description |
|------|-------------|
| `-module` | Module name |
| `-version` | Version |
| `-lang` | Language of compiled code |
| `-out` | Output directory |

#### Examples

```bash
# Download pre-compiled Go code
spoke-cli download \
  -module user \
  -version v1.0.0 \
  -lang go \
  -out ./generated/go

# Download Python code
spoke-cli download \
  -module user \
  -version v1.0.0 \
  -lang python \
  -out ./generated/python
```

---

### delete

Delete a module version.

#### Syntax

```bash
spoke-cli delete -module <name> -version <version> [options]
```

#### Required Flags

| Flag | Description |
|------|-------------|
| `-module` | Module name |
| `-version` | Version to delete |

#### Optional Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-force` | Skip confirmation | `false` |

#### Examples

```bash
# Delete a version (with confirmation)
spoke-cli delete -module user -version v1.0.0

# Force delete without confirmation
spoke-cli delete -module user -version v1.0.0 -force
```

---

### batch-push

Push multiple modules from a directory tree.

#### Syntax

```bash
spoke-cli batch-push -dir <path> -version <version> [options]
```

#### Required Flags

| Flag | Description |
|------|-------------|
| `-dir` | Root directory containing modules |
| `-version` | Version for all modules |

#### Optional Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-recursive` | Scan subdirectories | `true` |
| `-pattern` | Glob pattern for proto files | `*.proto` |

#### Directory Structure

```
schemas/
├── common/
│   └── types.proto
├── user/
│   └── user.proto
└── order/
    └── order.proto
```

#### Examples

```bash
# Push all modules
spoke-cli batch-push -dir ./schemas -version v1.0.0

# With pattern
spoke-cli batch-push \
  -dir ./schemas \
  -version v1.0.0 \
  -pattern "*.proto"
```

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SPOKE_REGISTRY` | Default registry URL | `http://localhost:8080` |
| `SPOKE_TIMEOUT` | Request timeout | `30s` |
| `SPOKE_ORG` | Default organization | "" |
| `SPOKE_TOKEN` | Authentication token | "" |

### Example

```bash
export SPOKE_REGISTRY=https://spoke.company.com
export SPOKE_TOKEN=your-api-token

# Now you can omit -registry flag
spoke-cli push -module user -version v1.0.0 -dir ./proto
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Network error |
| 4 | Authentication error |
| 5 | Not found |
| 6 | Validation error |

## Scripting Examples

### CI/CD Pipeline

```bash
#!/bin/bash
set -e

# Push on release
if [[ "$GITHUB_REF" == refs/tags/* ]]; then
  VERSION=${GITHUB_REF#refs/tags/}

  spoke-cli push \
    -module $MODULE_NAME \
    -version $VERSION \
    -dir ./proto \
    -registry $SPOKE_REGISTRY
fi
```

### Version Comparison

```bash
#!/bin/bash

# Pull two versions
spoke-cli pull -module user -version v1.0.0 -dir ./v1
spoke-cli pull -module user -version v1.1.0 -dir ./v2

# Compare
diff -u ./v1/user.proto ./v2/user.proto
```

### Automated Validation

```bash
#!/bin/bash

# Pre-commit hook
spoke-cli validate -dir ./proto

if [ $? -ne 0 ]; then
  echo "Proto validation failed"
  exit 1
fi
```

## Troubleshooting

### Command Not Found

```bash
# Ensure spoke-cli is in PATH
which spoke-cli

# If not found, add to PATH
export PATH=$PATH:/path/to/spoke-cli
```

### Connection Refused

```bash
# Check if server is running
curl http://localhost:8080/health

# Use correct registry URL
spoke-cli push -module user -version v1.0.0 -dir ./proto -registry http://localhost:8080
```

### Authentication Failed

```bash
# Set authentication token
export SPOKE_TOKEN=your-token

# Or use CLI flag
spoke-cli push -module user -version v1.0.0 -dir ./proto -token your-token
```

## Next Steps

- [API Reference](/guides/api-reference/) - HTTP REST API
- [Module Management](/guides/module-management/) - Managing modules
- [CI/CD Integration](/guides/ci-cd/) - Automation examples
