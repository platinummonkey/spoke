---
title: "What is Spoke?"
weight: 1
---

# What is Spoke?

Spoke is a **Protobuf Schema Registry** that provides centralized storage, versioning, and pre-compilation of Protocol Buffer definitions. It acts as a hub for schema-driven development, enabling multiple services and distributed systems to share and synchronize their communication contracts.

## The Problem

In modern distributed systems and microservice architectures, services need to communicate with each other using well-defined interfaces. Protocol Buffers (protobuf) is a popular choice for defining these interfaces, but managing protobuf schemas across multiple services presents challenges:

1. **Schema Distribution**: How do services get the protobuf definitions they need?
2. **Version Management**: How do you track changes and ensure compatibility?
3. **Dependency Resolution**: How do you handle proto files that import other proto files?
4. **Multi-Language Support**: How do you generate code for different programming languages?
5. **Compilation Overhead**: Do all clients need protoc and plugins installed?

## The Solution

Spoke solves these problems by providing:

### Centralized Storage
A single source of truth for all your protobuf definitions. Push schemas once, pull from anywhere.

```bash
# Push once
spoke push -module user-service -version v1.0.0 -dir ./proto

# Pull from any service
spoke pull -module user-service -version v1.0.0 -dir ./proto
```

### Version Management
Track schema versions using semantic versioning or commit hashes. Services can depend on specific versions.

```bash
# Evolution over time
spoke push -module user-service -version v1.0.0  # Initial release
spoke push -module user-service -version v1.1.0  # Add new fields
spoke push -module user-service -version v2.0.0  # Breaking changes
```

### Dependency Resolution
Automatically resolve and pull imported proto files recursively.

```proto
// user.proto
import "common/types.proto";  // Spoke resolves this automatically

message User {
  common.UUID id = 1;
}
```

```bash
# Pull with all dependencies
spoke pull -module user-service -version v1.0.0 -recursive
```

### Pre-Compilation
Optional Sprocket service pre-compiles protobuf files to multiple languages. Download ready-to-use code without needing protoc.

```bash
# Download pre-compiled Go code
spoke download -module user-service -version v1.0.0 -lang go -out ./generated
```

### Multi-Language Support
Generate code for any protobuf-supported language: Go, Python, Java, C++, JavaScript, Ruby, and more.

```bash
# Compile to different languages
spoke compile -dir ./proto -out ./go -lang go
spoke compile -dir ./proto -out ./python -lang python
spoke compile -dir ./proto -out ./java -lang java
```

## Architecture Components

### 1. Spoke Server
The core HTTP API server that stores and serves protobuf modules.

```bash
spoke-server -port 8080 -storage-dir /var/spoke/storage
```

### 2. Spoke CLI
Command-line interface for interacting with the registry.

```bash
spoke push -module mymodule -version v1.0.0
spoke pull -module mymodule -version v1.0.0
spoke compile -dir ./proto -out ./generated -lang go
```

### 3. Sprocket (Optional)
Background compilation service that automatically pre-compiles uploaded protobuf files.

```bash
sprocket -storage-dir /var/spoke/storage -delay 10s
```

### 4. Web UI
React-based interface for browsing modules, viewing versions, and inspecting schemas.

### 5. Storage Layer
File-based storage with organized module and version structure.

```
storage/
├── common/
│   ├── module.json
│   └── versions/
│       └── v1.0.0/
│           ├── version.json
│           ├── types.proto
│           ├── go/
│           │   └── types.pb.go
│           └── python/
│               └── types_pb2.py
```

## Use Cases

### gRPC Service Communication

Store gRPC service definitions and enable microservices to communicate with type-safe APIs.

```proto
service UserService {
  rpc GetUser(UserRequest) returns (User);
  rpc CreateUser(CreateUserRequest) returns (User);
}
```

### Event-Driven Architectures

Define event schemas for streaming platforms like Kafka or Pulsar.

```proto
message UserCreatedEvent {
  string user_id = 1;
  string email = 2;
  int64 timestamp = 3;
}
```

### Polyglot Microservices

Enable services written in different languages to share schemas.

```
Service A (Go) ←→ Spoke ←→ Service B (Python)
                    ↕
                Service C (Java)
```

### IoT & Edge Computing

Manage schemas for IoT devices and edge services with different language requirements.

## Key Benefits

### Type Safety
Protobuf provides strong typing across service boundaries, catching errors at compile time.

### Backward Compatibility
Protobuf's compatibility rules allow services to evolve independently.

### Performance
Binary protobuf encoding is compact and fast to serialize/deserialize.

### Schema Evolution Tracking
See what changed between versions and ensure compatibility.

### Multi-Language Agent Ecosystems
Build systems with components in different languages, all using the same schemas.

## Comparison with Alternatives

| Feature | Spoke | Buf Registry | Confluent Schema Registry |
|---------|-------|--------------|---------------------------|
| Protobuf Support | ✅ | ✅ | ❌ (Avro/JSON) |
| Self-Hosted | ✅ | ⚠️ Enterprise | ✅ |
| Pre-Compilation | ✅ | ❌ | N/A |
| Multi-Language | ✅ | ✅ | ✅ |
| Dependency Graph | ✅ | ✅ | ❌ |
| Open Source | ✅ | ⚠️ Limited | ✅ |
| Enterprise SSO | ✅ | ✅ | ✅ |
| RBAC | ✅ | ✅ | ✅ |
| Cost | Free | $$ | $$$ |

## When to Use Spoke

Spoke is ideal for:

- **Microservice architectures** with multiple services needing shared schemas
- **Event-driven systems** using Kafka, Pulsar, or other message brokers
- **Polyglot environments** with services in Go, Python, Java, etc.
- **Organizations** needing centralized schema governance
- **Teams** wanting to avoid "protoc installation hell" across environments

## When Not to Use Spoke

Spoke might not be the best fit if:

- You only have a single service or monolithic application
- You don't use Protocol Buffers
- You need Avro or JSON Schema support (use Confluent Schema Registry)
- You need production-hardened software with enterprise support (consider Buf Registry)

## Next Steps

- [Quick Start Guide](/getting-started/quick-start/) - Get started in 5 minutes
- [Installation Guide](/getting-started/installation/) - Detailed setup instructions
- [First Module Tutorial](/getting-started/first-module/) - Create your first module
