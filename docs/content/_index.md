---
title: "Spoke - Protobuf Schema Registry"
type: docs
weight: 1
---

# Spoke - Protobuf Schema Registry

![Spoke Logo](/logos/logo_main.png)

**Build Once. Connect Everywhere.**

Spoke is a **Protobuf Schema Registry** that provides centralized storage, versioning, and pre-compilation of Protocol Buffer definitions. It acts as a hub for schema-driven development, enabling multiple services and distributed systems to share and synchronize their communication contracts.

## What is Spoke?

Spoke solves the problem of schema management in distributed systems by providing:

- **Centralized Storage**: Single source of truth for protobuf definitions
- **Version Management**: Track and retrieve specific versions of schemas
- **Pre-Compilation**: Automatically compile proto files to multiple programming languages
- **Dependency Management**: Handle complex import relationships between proto modules
- **Multi-Language Support**: Generate code for Go, Python, C++, Java, and more
- **Enterprise Features**: SSO, RBAC, multi-tenancy, audit logging

## Key Features

### Core Schema Management
- Store and version protobuf files with semantic versioning
- Pull dependencies recursively
- Validate protobuf files before pushing
- AST parsing of protobuf files with comment support

### Compilation & Code Generation
- Compile protobuf files to multiple languages (Go, Python, Java, C++, etc.)
- Pre-compiled binaries via Sprocket background service
- Download ready-to-use generated code

### API & Integration
- Simple HTTP REST API
- Command-line interface (CLI)
- Webhook notifications for module updates
- CI/CD integration (GitHub Actions, GitLab CI)

### Enterprise Features
- **SSO Integration**: SAML 2.0, OAuth2, and OpenID Connect
- **RBAC**: Fine-grained permissions with custom roles and teams
- **Multi-Tenancy**: Organization-based isolation with quotas
- **Audit Logging**: Complete audit trail of all operations
- **Billing**: Stripe integration for SaaS deployments

## Quick Start

```bash
# Start the Spoke server
spoke-server -port 8080 -storage-dir ./storage

# Push a module
spoke push -module myservice -version v1.0.0 -dir ./proto

# Pull a module
spoke pull -module myservice -version v1.0.0 -dir ./output

# Compile to Go
spoke compile -dir ./output -out ./generated -lang go
```

## Use Cases

### gRPC Service Communication
Store and version gRPC service definitions, enabling microservices to communicate with type-safe, versioned APIs.

### Event Streaming
Define event schemas for streaming platforms (Kafka, Pulsar), ensuring producers and consumers use compatible message formats.

### Multi-Service Communication
Enable polyglot microservices (Go, Python, Java, C++) to share protobuf definitions with guaranteed compatibility.

### IoT & Edge Computing
Manage protobuf schemas for IoT devices and edge services with different language requirements.

## Why Spoke?

| Feature | Spoke | Buf Registry | Confluent Schema Registry |
|---------|-------|--------------|---------------------------|
| Protobuf Support | ✅ | ✅ | ❌ (Avro/JSON) |
| Self-Hosted | ✅ | ⚠️ Enterprise Only | ✅ |
| Pre-Compilation | ✅ | ❌ | N/A |
| Multi-Language | ✅ | ✅ | ✅ |
| Open Source | ✅ | ⚠️ Limited | ✅ |
| Enterprise SSO | ✅ | ✅ | ✅ |
| Cost | Free | $$ | $$$ |

## Architecture Overview

```
┌─────────────────────────────────────────┐
│         Spoke Schema Registry           │
│                                         │
│  common/v1.0.0  →  user/v1.0.0         │
│  order/v1.0.0   →  payment/v1.0.0      │
│  events/v2.0.0  →  analytics/v1.0.0    │
└─────────────────────────────────────────┘
         ↓              ↓              ↓
    Service A      Service B       Service C
   (Python)          (Go)         (Java)
```

## Next Steps

- [Getting Started Guide](/getting-started/quick-start/) - Set up Spoke in 5 minutes
- [Installation](/getting-started/installation/) - Install Spoke server and CLI
- [CLI Reference](/guides/cli-reference/) - Complete CLI command documentation
- [API Reference](/api/rest-api/) - HTTP API endpoints
- [Tutorials](/tutorials/) - Step-by-step guides

## Community & Support

- GitHub: [github.com/platinummonkey/spoke](https://github.com/platinummonkey/spoke)
- Issues: [GitHub Issues](https://github.com/platinummonkey/spoke/issues)
- License: [MIT License](https://github.com/platinummonkey/spoke/blob/main/LICENSE.md)

---

**Note**: Spoke is currently a POC project and not recommended for production use without thorough evaluation.
