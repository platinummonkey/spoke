# Spoke for Service Communication

This document describes how Spoke serves as a protobuf schema registry to enable communication between services, and distributed systems.

## What is Spoke?

Spoke is a **Protobuf Schema Registry** that provides centralized storage, versioning, and pre-compilation of Protocol Buffer definitions. It acts as a hub for schema-driven development, enabling multiple services, and systems to share and synchronize their communication contracts.

**Build Once. Connect Everywhere.**

## Purpose

Spoke solves the problem of schema management in distributed systems by:

1. **Centralized Storage**: Single source of truth for protobuf definitions
2. **Version Management**: Track and retrieve specific versions of schemas
3. **Pre-Compilation**: Automatically compile proto files to multiple programming languages
4. **Dependency Management**: Handle complex import relationships between proto modules
5. **Multi-Language Support**: Generate code for Go, Python, C++, Java, and more

## Use Cases for Service Communication


### 1. gRPC Service Communication

Spoke enables gRPC-based microservices by:

- **Service Definition Management**: Store `.proto` files defining RPC services
- **Automatic Code Generation**: Pre-compile to language-specific gRPC stubs
- **Versioned APIs**: Track service interface changes over time

**Example: Microservice Architecture**
```proto
// order.proto stored in Spoke
service OrderService {
  rpc CreateOrder(OrderRequest) returns (OrderResponse);
  rpc GetOrder(OrderID) returns (Order);
}
```

Services pull the compiled stubs:
```bash
spoke pull -module order -version v1.2.0 -dir ./proto
spoke compile -dir ./proto -out ./generated -lang go
```

### 2. Streaming Protocols

For streaming systems using protobuf:

- **Event Schema Management**: Define streaming event formats
- **Consumer Compatibility**: Ensure consumers have correct schema versions
- **Producer Coordination**: Producers reference the same event definitions

**Example: Event Streaming (Kafka, Pulsar)**
```proto
// events.proto
message UserAction {
  string user_id = 1;
  string action_type = 2;
  int64 timestamp = 3;
}
```

Producers and consumers both pull from Spoke:
```bash
# Producer service
spoke pull -module events -version v1.0.0

# Consumer service
spoke pull -module events -version v1.0.0
```

### 3. Protocol-Agnostic Communication

Any protocol using protobuf as content type:

- **HTTP/REST with Protobuf**: Binary proto payloads over HTTP
- **WebSocket Messaging**: Real-time communication with proto messages
- **Message Queues**: RabbitMQ, SQS with protobuf serialization
- **Custom Protocols**: Domain-specific protocols using protobuf encoding

## How Spoke Enables Service Communication

### Shared Schema Repository

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

### Version Management for API Evolution

As services evolve, Spoke tracks changes:

```bash
# Version 1: Initial release
spoke push -module user -version v1.0.0 -dir ./proto

# Version 2: Add new field
spoke push -module user -version v1.1.0 -dir ./proto

# Legacy services use v1.0.0, new services use v1.1.0
spoke pull -module user -version v1.0.0  # Legacy
spoke pull -module user -version v1.1.0  # Current
```

### Dependency Resolution

When services share common types:

```proto
// common/types.proto
message Timestamp { ... }
message UUID { ... }

// order/order.proto
import "common/types.proto";
message Order {
  UUID id = 1;
  Timestamp created_at = 2;
}
```

Spoke handles recursive dependencies:
```bash
spoke pull -module order -version v1.0.0 -recursive
# Automatically pulls common/types.proto as well
```

### Pre-Compilation for Multiple Languages

Spoke pre-compiles protobuf definitions to support polyglot systems:

**Go Service:**
```bash
spoke pull -module events -version v1.0.0
spoke compile -lang go -dir ./proto -out ./pkg/events
```

**Python Service:**
```bash
spoke pull -module events -version v1.0.0
spoke compile -lang python -dir ./proto -out ./events_pb
```

**C++ Service:**
```bash
spoke pull -module events -version v1.0.0
spoke compile -lang cpp -dir ./proto -out ./generated
```

All services now share the same message definitions but use language-native code.

## Architecture Components

### 1. HTTP API Server

The core registry service (`cmd/spoke/main.go`):

```bash
spoke-server -port 8080 -storage-dir /var/spoke/storage
```

**API Endpoints:**
- `POST /modules` - Register new protobuf module
- `GET /modules/{name}/versions/{version}` - Retrieve specific version
- `GET /modules/{name}/versions/{version}/files/{path}` - Download proto file
- `POST /modules/{name}/versions/{version}/compile/{language}` - Request compilation
- `GET /modules/{name}/versions/{version}/download/{language}` - Download compiled files

### 2. Command-Line Interface

The `spoke` CLI (`cmd/spoke-cli/main.go`) provides:

**Push schemas:**
```bash
spoke push -module user -version v1.0.0 -dir ./proto -description "User service API"
```

**Pull schemas:**
```bash
spoke pull -module user -version v1.0.0 -dir ./proto -recursive
```

**Compile locally:**
```bash
spoke compile -dir ./proto -out ./generated -lang go
```

**Validate schemas:**
```bash
spoke validate -dir ./proto -recursive
```

**Batch operations:**
```bash
spoke batch-push -dir ./schemas -recursive
```

### 3. Sprocket Compilation Service

Sprocket (`cmd/sprocket/main.go`) is an optional background service that automatically compiles protobuf files when they're added to the registry:

```bash
sprocket -storage-dir /var/spoke/storage -delay 10s
```

**What it does:**
- Monitors for new protobuf files in storage
- Automatically triggers compilation for Go, Python, and other languages
- Manages dependency graph and cascading compilation
- Pre-generates binaries so agents don't need protoc installed

**Why it's useful:**
- Agents can download pre-compiled code directly
- Reduces compilation time for consumers
- Ensures all versions are compiled consistently
- Handles complex dependency chains automatically

### 4. Storage Layer

Filesystem-based storage organizing modules:

```
storage/
├── common/
│   ├── module.json
│   └── versions/
│       ├── v1.0.0/
│       │   ├── version.json
│       │   ├── types.proto
│       │   ├── go/
│       │   │   └── types.pb.go
│       │   └── python/
│       │       └── types_pb2.py
│       └── v1.1.0/
│           └── ...
└── order/
    └── ...
```

### 5. Web UI

React-based interface (`web/`) for:
- Browsing available modules
- Viewing version history
- Inspecting message definitions
- Searching schemas

## Communication Patterns for Agents

### Pattern 1: Direct gRPC Communication

```
┌─────────┐                    ┌─────────┐
│ Agent A │ ←── gRPC Call ───→ │ Agent B │
└─────────┘                    └─────────┘
     ↑                              ↑
     └──── Spoke (service.proto) ───┘
```

Both agents pull the same service definition and communicate directly.

### Pattern 2: Event-Driven Communication

```
┌─────────┐      Publish       ┌──────────┐
│ Agent A │ ──→ Event Stream ──→│ Agent B  │
└─────────┘                     └──────────┘
     ↑                               ↑
     └──── Spoke (events.proto) ─────┘
```

Agents produce and consume events using shared schema definitions.

### Pattern 3: Hub-and-Spoke (Request-Response)

```
        ┌─────────┐
        │ Broker  │
        └────┬────┘
             │
    ┌────────┼────────┐
    ↓        ↓        ↓
┌───────┐ ┌───────┐ ┌───────┐
│Agent A│ │Agent B│ │Agent C│
└───────┘ └───────┘ └───────┘
    ↑         ↑         ↑
    └── Spoke (api.proto) ───┘
```

A central broker routes messages between agents using shared protobuf definitions.

### Pattern 4: Streaming Data Pipeline

```
Producer → Transform → Aggregate → Consumer
   ↓          ↓          ↓          ↓
   └─── Spoke (pipeline.proto) ────┘
```

Multi-stage pipeline where each stage uses the same message format.

## Benefits for Agent-Based Systems

### 1. Type Safety

Protobuf provides strong typing across agent boundaries:

```proto
message AgentCommand {
  enum CommandType {
    START = 0;
    STOP = 1;
    RESTART = 2;
  }
  CommandType type = 1;
  string agent_id = 2;
}
```

Agents can't send invalid commands - the type system enforces correctness.

### 2. Backward Compatibility

Protobuf's compatibility rules ensure agents can evolve independently:

```proto
// v1.0.0
message Config {
  string name = 1;
}

// v1.1.0 - backward compatible
message Config {
  string name = 1;
  int32 timeout = 2;  // New field, old agents ignore it
}
```

### 3. Performance

Binary protobuf encoding is efficient for agent communication:
- Smaller message sizes than JSON
- Fast serialization/deserialization
- Low memory overhead

### 4. Schema Evolution Tracking

Spoke maintains version history:

```bash
# See what changed between versions
spoke pull -module user -version v1.0.0 -dir ./v1
spoke pull -module user -version v1.1.0 -dir ./v2
diff ./v1/user.proto ./v2/user.proto
```

### 5. Multi-Language Agent Ecosystems

Build agents in different languages with guaranteed compatibility:

- Python agents for ML/data processing
- Go agents for high-performance services
- Java agents for enterprise integration
- C++ agents for embedded systems

All using the same protobuf schemas from Spoke.

## Real-World Agent Scenarios

### Scenario 1: Distributed AI Agent System

**Setup:**
- Coordinator agent (Python) orchestrates tasks
- Worker agents (Go) execute computationally intensive jobs
- Monitoring agent (Python) tracks metrics

**Spoke Usage:**
```bash
# Define agent communication protocol
cat > agent-protocol.proto <<EOF
syntax = "proto3";

message Task {
  string task_id = 1;
  bytes payload = 2;
  int32 priority = 3;
}

message TaskResult {
  string task_id = 1;
  bool success = 2;
  bytes result = 3;
}

service AgentCoordinator {
  rpc AssignTask(Task) returns (TaskAck);
  rpc ReportResult(TaskResult) returns (ResultAck);
}
EOF

# Push to Spoke
spoke push -module agent-protocol -version v1.0.0

# Coordinator pulls Python stubs
spoke pull -module agent-protocol -version v1.0.0
spoke compile -lang python -dir ./proto -out ./coordinator/proto

# Workers pull Go stubs
spoke pull -module agent-protocol -version v1.0.0
spoke compile -lang go -dir ./proto -out ./worker/proto
```

### Scenario 2: IoT Device Communication

**Setup:**
- Edge devices (C++) send sensor data
- Gateway service (Go) aggregates data
- Cloud processing (Python) analyzes data

**Spoke Usage:**
```proto
// sensor-data.proto
message SensorReading {
  string device_id = 1;
  double temperature = 2;
  double humidity = 3;
  int64 timestamp = 4;
}
```

Each component pulls the schema and compiles to its native language.

### Scenario 3: Microservice Mesh

**Setup:**
- 20+ microservices in Go, Python, Java
- gRPC for synchronous calls
- Kafka for asynchronous events

**Spoke Usage:**
```bash
# Common types shared across all services
spoke push -module common -version v1.0.0 -dir ./proto/common

# Each service pushes its API
spoke push -module user-service -version v1.0.0 -dir ./proto/user
spoke push -module order-service -version v1.0.0 -dir ./proto/order
spoke push -module payment-service -version v1.0.0 -dir ./proto/payment

# Services pull dependencies
spoke pull -module user-service -version v1.0.0 -recursive
```

## Deployment Patterns

### Development Environment

```bash
# Run local registry
spoke-server -port 8080 -storage-dir ./local-storage

# Run Sprocket for automatic compilation
sprocket -storage-dir ./local-storage -delay 5s

# Push schemas during development
spoke push -module myservice -version v0.1.0-dev
```

### Production Environment

```bash
# Run registry with persistent storage
spoke-server -port 8080 -storage-dir /var/spoke/storage

# Run Sprocket as systemd service
systemctl start sprocket

# Use semantic versions
spoke push -module myservice -version v1.0.0

# Services pull on deployment
spoke pull -module myservice -version v1.0.0
```

### CI/CD Integration

```yaml
# .github/workflows/schema-push.yml
- name: Push schemas to Spoke
  run: |
    spoke push \
      -module ${{ github.repository }} \
      -version ${{ github.ref_name }} \
      -dir ./proto \
      -registry https://spoke.company.com
```

## Best Practices for Agent Communication

### 1. Semantic Versioning

Use semantic versions for schema changes:
- `v1.0.0` → `v1.0.1`: Bug fixes (backward compatible)
- `v1.0.0` → `v1.1.0`: New features (backward compatible)
- `v1.0.0` → `v2.0.0`: Breaking changes

### 2. Explicit Dependencies

Always specify version dependencies:
```bash
spoke pull -module common -version v1.2.0  # Not "latest"
```

### 3. Proto Style Guide

Follow consistent naming:
```proto
syntax = "proto3";

// Package naming: domain.service.version
package company.userservice.v1;

// Message names: PascalCase
message UserProfile { ... }

// Field names: snake_case
string user_name = 1;

// Enum names: UPPER_CASE
enum Status {
  STATUS_UNKNOWN = 0;
  STATUS_ACTIVE = 1;
}
```

### 4. Separate Data from API

```
proto/
├── data/          # Shared data types
│   └── common.proto
├── api/           # Service definitions
│   └── service.proto
└── events/        # Event messages
    └── events.proto
```

### 5. Document Your Schemas

```proto
// User profile information.
//
// This message represents a user's public profile data
// that can be shared across services.
message UserProfile {
  // Unique user identifier (UUID format)
  string user_id = 1;

  // Display name (3-50 characters)
  string display_name = 2;
}
```

## Troubleshooting

### Issue: Agents have incompatible schemas

**Solution:** Ensure all agents pull the same version:
```bash
# Check what version each agent is using
spoke pull -module myapi -version v1.0.0 -dir ./agent-a-proto
spoke pull -module myapi -version v1.1.0 -dir ./agent-b-proto
diff -r ./agent-a-proto ./agent-b-proto
```

### Issue: Compilation fails

**Solution:** Validate proto files before pushing:
```bash
spoke validate -dir ./proto
protoc --proto_path=./proto --go_out=/tmp ./proto/*.proto
```

### Issue: Import not found

**Solution:** Use recursive pull:
```bash
spoke pull -module mymodule -version v1.0.0 -recursive
```

### Issue: Version conflicts

**Solution:** Use dependency lockfile:
```json
// spoke.lock
{
  "dependencies": {
    "common": "v1.2.0",
    "user": "v1.0.5",
    "order": "v2.1.0"
  }
}
```

## Alternatives Comparison

| Feature | Spoke | Buf Registry                   | Confluent Schema Registry |
|---------|-------|--------------------------------|---------------------------|
| Protobuf Support | ✅ | ✅                              | ❌ (Avro/JSON) |
| Self-Hosted | ✅ | ⚠️ (Cloud, enterprise-only for on-prem) | ✅ |
| Pre-Compilation | ✅ | ❌                              | N/A |
| Multi-Language | ✅ | ✅                              | ✅ |
| Dependency Graph | ✅ | ✅                              | ❌ |
| Open Source | ✅ | ⚠️                             | ✅ |
| Cost | Free Forever | $$                             | $$$ |

## Summary

Spoke is a protobuf schema registry that enables effective communication between agents, services, and distributed systems by:

- **Centralizing Schema Management**: Single source of truth for protobuf definitions
- **Version Control**: Track and retrieve specific schema versions
- **Pre-Compilation**: Generate language-specific code automatically
- **Dependency Resolution**: Handle complex import relationships
- **Multi-Language Support**: Enable polyglot agent ecosystems

Whether you're building multi-agent AI systems, service architectures, event-driven platforms, or IoT networks, Spoke provides the schema infrastructure needed for reliable, type-safe communication across heterogeneous systems.

**Build Once. Connect Everywhere.**

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
