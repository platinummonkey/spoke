# Plugin Development Guide

This guide explains how to develop custom plugins for Spoke to extend its functionality.

## Table of Contents

- [Overview](#overview)
- [Plugin Types](#plugin-types)
- [Quick Start](#quick-start)
- [Language Plugins](#language-plugins)
- [Validator Plugins](#validator-plugins)
- [Runner Plugins](#runner-plugins)
- [Testing Plugins](#testing-plugins)
- [Publishing Plugins](#publishing-plugins)

## Overview

Spoke's plugin system allows you to:
- Add support for new programming languages
- Create custom validators and linters
- Implement custom execution environments
- Transform generated code

### Plugin Architecture

```
Spoke Core
    │
    ├─ Plugin Loader (discovers plugins)
    │     │
    │     ├─ ~/.spoke/plugins/
    │     ├─ /etc/spoke/plugins/
    │     └─ ./plugins/
    │
    ├─ Plugin Registry (manages loaded plugins)
    │
    └─ Language Registry (integrates language plugins)
```

### Plugin Format

Plugins are directories containing:
- `plugin.yaml` - Manifest with metadata
- `language_spec.yaml` - (Language plugins only) Language specification
- Optional: Native Go plugin (`.so` file) for advanced features
- Optional: gRPC service for remote execution

## Plugin Types

### Language Plugin
Adds code generation support for a programming language.

**Example**: Rust, Kotlin, Elm

### Validator Plugin
Validates and lints Protocol Buffer schemas.

**Example**: Style checker, breaking change detector

### Generator Plugin
Generates additional artifacts (documentation, package files).

**Example**: README generator, API documentation

### Runner Plugin
Custom execution environment for compilation.

**Example**: Cloud compilation, WebAssembly runtime

### Transform Plugin
Post-processes generated code.

**Example**: Code formatter, license header injector

## Quick Start

### Create a Basic Language Plugin

**1. Create plugin directory**
```bash
mkdir -p ~/.spoke/plugins/my-language
cd ~/.spoke/plugins/my-language
```

**2. Create `plugin.yaml`**
```yaml
id: my-language
name: My Language Plugin
version: 1.0.0
api_version: 1.0.0
description: Adds My Language support to Spoke
author: Your Name
license: MIT
type: language
security_level: community
permissions:
  - filesystem:read
  - filesystem:write
```

**3. Create `language_spec.yaml`**
```yaml
id: mylang
name: MyLang
display_name: My Language (v2.0)
supports_grpc: true
file_extensions:
  - .mylang
enabled: true
stable: false
description: My Language code generation
protoc_plugin: protoc-gen-mylang
docker_image: myorg/mylang-protoc:latest

package_manager:
  name: mylang-pkg
  config_files:
    - package.mylang
```

**4. Test the plugin**
```bash
# Validate manifest
spoke plugin validate ~/.spoke/plugins/my-language

# Test discovery
spoke plugin list
```

**5. Use the plugin**
```bash
spoke compile -module test -version v1.0.0 -lang mylang -out ./output
```

## Language Plugins

Language plugins enable Spoke to generate code for a specific programming language.

### Manifest Fields

```yaml
id: unique-language-id
name: Display Name
version: 1.0.0  # Plugin version
api_version: 1.0.0  # Spoke Plugin SDK version
type: language
```

### Language Specification

Create `language_spec.yaml`:

```yaml
# Identification
id: mylang  # Used in commands: -lang mylang
name: MyLang
display_name: MyLang (protoc-gen-mylang)

# Protoc integration
protoc_plugin: protoc-gen-mylang  # Plugin binary name
plugin_version: "1.5.0"  # Version of protoc plugin
docker_image: myorg/mylang-protoc:1.5  # Docker image with protoc + plugin

# Features
supports_grpc: true  # Can generate gRPC services
file_extensions:
  - .mylang  # Expected output file extensions
enabled: true  # Enable by default
stable: true  # Production-ready

# Documentation
description: Generate MyLang code from Protocol Buffers
documentation_url: https://mylang.org/docs/protobuf

# Package manager (optional)
package_manager:
  name: mylang-pkg
  config_files:
    - package.mylang
    - README.md
  dependency_map:
    protobuf: mylang-protobuf
    grpc: mylang-grpc
  default_versions:
    mylang-protobuf: "2.0.0"
    mylang-grpc: "1.5.0"

# Custom options (optional)
custom_options:
  optimize_for: speed
  generate_equals: "true"
```

### Protoc Command Building

Spoke builds commands like:
```bash
protoc \
  --proto_path=/path/to/protos \
  --mylang_out=/output/dir \
  --mylang_opt=optimize_for=speed,generate_equals=true \
  user.proto
```

### Docker Image Requirements

Your Docker image should:
1. Include `protoc` compiler (3.15.0+)
2. Include your `protoc-gen-mylang` plugin
3. Be publicly available or in your private registry

**Example Dockerfile**:
```dockerfile
FROM golang:1.21 AS builder
RUN apt-get update && apt-get install -y protobuf-compiler
RUN go install github.com/myorg/protoc-gen-mylang@v1.5.0

FROM debian:bookworm-slim
COPY --from=builder /usr/bin/protoc /usr/bin/
COPY --from=builder /go/bin/protoc-gen-mylang /usr/bin/
CMD ["/bin/bash"]
```

### Package Manager Integration

If your language has a package manager, provide templates:

**Create `templates/package.mylang.tmpl`**:
```
[package]
name = "{{ .ModuleName }}"
version = "{{ .Version }}"

[dependencies]
{{range .Dependencies}}
{{.Name}} = "{{.Version}}"
{{end}}
```

Spoke will generate this file alongside generated code.

### Advanced: Native Go Plugin

For complex logic, implement the `LanguagePlugin` interface:

**Create `plugin.go`**:
```go
package main

import (
    "context"
    "github.com/platinummonkey/spoke/pkg/plugins"
)

type MyLanguagePlugin struct {
    manifest *plugins.Manifest
    spec     *plugins.LanguageSpec
}

func (p *MyLanguagePlugin) Manifest() *plugins.Manifest {
    return p.manifest
}

func (p *MyLanguagePlugin) Load() error {
    // Initialize plugin
    return nil
}

func (p *MyLanguagePlugin) Unload() error {
    // Cleanup
    return nil
}

func (p *MyLanguagePlugin) GetLanguageSpec() *plugins.LanguageSpec {
    return p.spec
}

func (p *MyLanguagePlugin) BuildProtocCommand(ctx context.Context, req *plugins.CommandRequest) ([]string, error) {
    // Custom command building logic
    cmd := []string{"protoc"}
    // ... build command
    return cmd, nil
}

func (p *MyLanguagePlugin) ValidateOutput(ctx context.Context, files []string) error {
    // Validate generated files
    return nil
}

// Export plugin
var Plugin MyLanguagePlugin
```

**Build as shared library**:
```bash
go build -buildmode=plugin -o mylang.so plugin.go
```

## Validator Plugins

Validator plugins check Protocol Buffer schemas for errors and style issues.

### Manifest

```yaml
id: my-validator
name: My Validator
version: 1.0.0
api_version: 1.0.0
type: validator
security_level: community
permissions:
  - filesystem:read
```

### Implementation

```go
package main

import (
    "context"
    "github.com/platinummonkey/spoke/pkg/plugins"
)

type MyValidator struct {
    manifest *plugins.Manifest
}

func (v *MyValidator) Manifest() *plugins.Manifest {
    return v.manifest
}

func (v *MyValidator) Load() error {
    return nil
}

func (v *MyValidator) Unload() error {
    return nil
}

func (v *MyValidator) Validate(ctx context.Context, req *plugins.ValidationRequest) (*plugins.ValidationResult, error) {
    result := &plugins.ValidationResult{
        Valid:    true,
        Errors:   []plugins.ValidationError{},
        Warnings: []plugins.ValidationWarning{},
    }

    // Validate proto files
    for _, protoFile := range req.ProtoFiles {
        // Check for issues
        // Add errors/warnings to result
    }

    if len(result.Errors) > 0 {
        result.Valid = false
    }

    return result, nil
}

var Plugin MyValidator
```

## Runner Plugins

Runner plugins provide custom execution environments.

### Manifest

```yaml
id: cloud-runner
name: Cloud Compilation Runner
version: 1.0.0
api_version: 1.0.0
type: runner
security_level: community
permissions:
  - network:read
  - network:write
```

### Implementation

```go
func (r *CloudRunner) Execute(ctx context.Context, req *plugins.ExecutionRequest) (*plugins.ExecutionResult, error) {
    // Execute command in cloud environment
    // Return results
    return &plugins.ExecutionResult{
        ExitCode: 0,
        Stdout:   "compilation output",
        Stderr:   "",
        Duration: time.Second * 5,
    }, nil
}
```

## Testing Plugins

### Validation

```bash
# Validate manifest
spoke plugin validate ./my-plugin

# Check API compatibility
spoke plugin check-compatibility ./my-plugin

# Lint manifest
spoke plugin lint ./my-plugin/plugin.yaml
```

### Unit Tests

Write Go tests for your plugin:

```go
func TestMyPlugin(t *testing.T) {
    manifest, err := plugins.LoadManifest("./plugin.yaml")
    require.NoError(t, err)

    errors := plugins.ValidateManifest(manifest)
    assert.Empty(t, errors)
}
```

### Integration Tests

Test with real proto files:

```bash
# Create test proto
cat > test.proto <<EOF
syntax = "proto3";
message Test {
  string name = 1;
}
EOF

# Test compilation
spoke compile -module test -version v1.0.0 -lang mylang -out ./test-output

# Verify output
ls -la ./test-output
```

### Docker Testing

Test your Docker image:

```bash
# Pull image
docker pull myorg/mylang-protoc:latest

# Test protoc works
docker run myorg/mylang-protoc:latest protoc --version

# Test plugin works
docker run myorg/mylang-protoc:latest protoc-gen-mylang --version
```

## Publishing Plugins

### To Plugin Marketplace

1. **Prepare plugin**
   ```bash
   cd my-plugin
   tar -czf my-plugin-v1.0.0.tar.gz *
   ```

2. **Submit to marketplace**
   ```bash
   spoke plugin submit \
     --archive my-plugin-v1.0.0.tar.gz \
     --manifest plugin.yaml
   ```

3. **Verification process**
   - Spoke team reviews code
   - Security scan runs automatically
   - If approved, plugin gets "verified" badge

### To GitHub

1. Create repository
2. Add plugin files
3. Tag release
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

4. Users can install via:
   ```bash
   spoke plugin install github.com/myorg/my-plugin@v1.0.0
   ```

### To Private Registry

Host plugin files on your own server:
```bash
# users install via URL
spoke plugin install https://plugins.mycompany.com/my-plugin/v1.0.0.tar.gz
```

## Best Practices

### 1. Semantic Versioning
- Use semver: `MAJOR.MINOR.PATCH`
- Increment MAJOR for breaking changes
- Document changes in CHANGELOG.md

### 2. Security
- Request minimal permissions
- Validate all inputs
- Don't include secrets in plugin files
- Use signed releases

### 3. Documentation
- Include README.md
- Provide usage examples
- Document all options
- Add troubleshooting section

### 4. Testing
- Test on multiple platforms (Linux, macOS, Windows)
- Test with various protoc versions
- Include integration tests
- Test error cases

### 5. Performance
- Cache results when possible
- Minimize Docker image size
- Avoid unnecessary file I/O
- Stream large outputs

### 6. Compatibility
- Specify minimum protoc version
- Test with multiple Spoke versions
- Handle deprecated features gracefully
- Provide migration guides

## Troubleshooting

### Plugin not loading

**Check logs**:
```bash
spoke -log-level debug plugin list
```

**Common issues**:
- Wrong directory structure
- Invalid YAML syntax
- Missing required fields
- API version mismatch

### Compilation fails

**Verify protoc**:
```bash
docker run YOUR_IMAGE protoc --version
```

**Check plugin binary**:
```bash
docker run YOUR_IMAGE which protoc-gen-mylang
docker run YOUR_IMAGE protoc-gen-mylang --version
```

### Permission denied

**Check plugin permissions**:
```yaml
permissions:
  - filesystem:read  # Required for reading protos
  - filesystem:write  # Required for writing output
```

## Examples

Complete example plugins:
- [Rust Language Plugin](../plugins/rust-language/)
- [Buf Connect Plugin](../plugins/buf-connect-go/)
- [Protolint Validator](../plugins/protolint-validator/)

## Support

- **Documentation**: https://docs.spoke.dev/plugins
- **API Reference**: https://pkg.go.dev/github.com/platinummonkey/spoke/pkg/plugins
- **Community**: https://github.com/platinummonkey/spoke/discussions
- **Issues**: https://github.com/platinummonkey/spoke/issues

## See Also

- [Plugin Manifest Specification](PLUGIN_MANIFEST.md)
- [Plugin API Reference](PLUGIN_API.md)
- [Security Guidelines](SECURITY.md)
