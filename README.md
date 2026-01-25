# Spoke - Protobuf Schema Registry

![Spoke Logo](./web/public/logos/logo_main.png)

[![CI](https://github.com/platinummonkey/spoke/actions/workflows/ci.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/ci.yml)
[![Lint](https://github.com/platinummonkey/spoke/actions/workflows/lint.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/lint.yml)
[![Security](https://github.com/platinummonkey/spoke/actions/workflows/security.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/security.yml)
[![Coverage](https://github.com/platinummonkey/spoke/actions/workflows/coverage.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/coverage.yml)
[![Build](https://github.com/platinummonkey/spoke/actions/workflows/build.yml/badge.svg)](https://github.com/platinummonkey/spoke/actions/workflows/build.yml)

THIS IS A POC HACK WEEK PROJECT. NOT FOR PRODUCTION USE.

Spoke is a Protobuf Schema Registry that helps manage and version your Protocol Buffer definitions. It provides a simple way to store, retrieve, and compile protobuf files with dependency management.

Build Once. Connect Everywhere. The Hub of Schema-Driven Development.


## Features

### Core Features
- Store and version protobuf files
- Pull dependencies recursively
- Compile protobuf files to multiple languages (Go, Python, C++, Java)
- Validate protobuf files
- AST parsing of protobuf files with comment support
- RESTful HTTP API
- Command-line interface

### Documentation & Developer Experience
- **Interactive API Explorer**: Browse services, methods, and message schemas with expandable UI
- **Auto-Generated Code Examples**: Working code snippets for 15+ languages (Go, Python, Rust, TypeScript, Java, etc.)
- **Schema Comparison Tool**: Visual diff showing breaking changes, additions, and modifications between versions
- **Migration Guides**: Step-by-step upgrade instructions with code examples
- **Full-Text Search**: Fast client-side search across modules, messages, services, fields, and enums (CMD+K)
- **Request/Response Playground**: Build and validate JSON requests against proto schemas
- **Multi-Language Support**: Generate examples for Go, Python, Java, C++, C#, Rust, TypeScript, JavaScript, Dart, Swift, Kotlin, Objective-C, Ruby, PHP, Scala

### High Availability Features
- **Horizontal Scaling**: Stateless architecture supports unlimited replicas
- **Database HA**: PostgreSQL with streaming replication and read replicas
- **Distributed Caching**: Redis with Sentinel for automatic failover
- **Zero-Downtime Deployments**: Rolling updates with health checks
- **Automated Backups**: Daily PostgreSQL backups to S3 with point-in-time recovery
- **Observability**: OpenTelemetry integration (traces, metrics, logs)
- **Rate Limiting**: Distributed rate limiting across instances
- **Health Checks**: Liveness, readiness, and startup probes
- **Graceful Shutdown**: Clean termination with request draining

### Production Ready
- Environment-based configuration
- Kubernetes deployment with HPA
- Docker Compose HA stack for testing
- Comprehensive monitoring and alerting
- Security hardening (non-root, read-only filesystem)
- 99.9% SLA achievable

## Prerequisites

- Go 1.16 or later
- Protocol Buffers compiler (protoc)
- Language-specific protoc plugins (e.g., protoc-gen-go for Go)

## Quick Start

### Local Development

1. Clone the repository:
```bash
git clone https://github.com/platinummonkey/spoke.git
cd spoke
```

2. Build the server and CLI:
```bash
go build -o bin/spoke-server cmd/spoke/main.go
go build -o bin/spoke cmd/spoke-cli/main.go
```

3. Run with filesystem storage:
```bash
export SPOKE_STORAGE_TYPE=filesystem
export SPOKE_FILESYSTEM_ROOT=./data/storage
./bin/spoke-server
```

Server available at:
- API: http://localhost:8080
- Health: http://localhost:9090

### Production Deployment

Deploy to Kubernetes with high availability:

```bash
# Apply configuration
kubectl create namespace spoke
kubectl apply -f deployments/kubernetes/spoke-config.yaml
kubectl apply -f deployments/kubernetes/spoke-deployment.yaml
kubectl apply -f deployments/kubernetes/spoke-hpa.yaml

# Verify deployment
kubectl get pods -n spoke
# Expected: 3/3 pods running
```

See [Deployment Guide](docs/deployment/DEPLOYMENT_GUIDE.md) for complete instructions.

### Local HA Testing

Test the full HA stack locally with Docker Compose:

```bash
cd deployments/docker-compose
docker-compose -f ha-stack.yml up -d
```

Includes:
- 3 Spoke instances behind NGINX load balancer
- PostgreSQL with replication
- Redis with Sentinel
- MinIO (S3-compatible storage)
- OpenTelemetry Collector

Access at http://localhost:8080

## Usage

### Starting the Server

Start the Spoke server with default settings:
```bash
spoke-server
```

Or customize the port and storage directory:
```bash
spoke-server -port 8080 -storage-dir /path/to/storage
```

### CLI Commands

#### Push a Module

Push protobuf files to the registry:
```bash
spoke push -module mymodule -version v1.0.0 -dir ./proto
```

Options:
- `-module`: Module name (required)
- `-version`: Version (semantic version or commit hash) (required)
- `-dir`: Directory containing protobuf files (default: ".")
- `-registry`: Registry URL (default: "http://localhost:8080")
- `-description`: Module description

#### Pull a Module

Pull protobuf files from the registry:
```bash
spoke pull -module mymodule -version v1.0.0 -dir ./proto
```

Options:
- `-module`: Module name (required)
- `-version`: Version (semantic version or commit hash) (required)
- `-dir`: Directory to save protobuf files (default: ".")
- `-registry`: Registry URL (default: "http://localhost:8080")
- `-recursive`: Pull dependencies recursively (default: false)

#### Compile Protobuf Files

Compile protobuf files using protoc:
```bash
spoke compile -dir ./proto -out ./generated -lang go
```

Options:
- `-dir`: Directory containing protobuf files (default: ".")
- `-out`: Output directory for generated files (default: ".")
- `-lang`: Output language (go, cpp, java) (default: "go")
- `-registry`: Registry URL (default: "http://localhost:8080")
- `-recursive`: Pull dependencies recursively (default: false)

#### Validate Protobuf Files

Validate protobuf files:
```bash
spoke validate -dir ./proto
```

Options:
- `-dir`: Directory containing protobuf files (default: ".")
- `-registry`: Registry URL (default: "http://localhost:8080")
- `-recursive`: Validate dependencies recursively (default: false)

## Example Workflow

1. Create a new module:
```bash
# Create your protobuf files
mkdir -p proto
cat > proto/example.proto << EOF
syntax = "proto3";
package example;

message Hello {
  string message = 1;
}
EOF

# Push to registry
spoke push -module example -version v1.0.0 -dir ./proto
```

2. Pull and use the module:
```bash
# Pull the module
spoke pull -module example -version v1.0.0 -dir ./myproject/proto

# Compile to Go
spoke compile -dir ./myproject/proto -out ./myproject/generated -lang go
```

## Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Load Balancer                          │
│                   (NGINX/ALB/Ingress)                       │
└──────────────────────┬──────────────────────────────────────┘
                       │
       ┌───────────────┼───────────────┐
       │               │               │
┌──────▼─────┐  ┌──────▼─────┐  ┌──────▼─────┐
│  Spoke 1   │  │  Spoke 2   │  │  Spoke 3   │
│  (Pod)     │  │  (Pod)     │  │  (Pod)     │
└──────┬─────┘  └──────┬─────┘  └──────┬─────┘
       │               │               │
       └───────────────┼───────────────┘
                       │
       ┌───────────────┼───────────────┐
       │               │               │
┌──────▼─────┐  ┌──────▼─────┐  ┌──────▼─────┐
│ PostgreSQL │  │   Redis    │  │     S3     │
│  Primary   │  │   Master   │  │  Storage   │
│   + Read   │  │ + Sentinel │  │            │
│  Replicas  │  │  Failover  │  │            │
└────────────┘  └────────────┘  └────────────┘
```

### Component Responsibilities

- **Spoke Instances**: Stateless API servers (horizontally scalable)
- **PostgreSQL**: Metadata storage (modules, versions, dependencies)
- **Redis**: Distributed cache and rate limiting
- **S3**: Proto files and compiled binaries

See [HA Architecture](docs/architecture/HA_ARCHITECTURE.md) for detailed documentation.

## Configuration

Spoke is configured via environment variables:

```bash
# Server
export SPOKE_PORT=8080
export SPOKE_HEALTH_PORT=9090

# Storage
export SPOKE_STORAGE_TYPE=postgres  # or "filesystem", "hybrid"
export SPOKE_POSTGRES_URL="postgresql://user:pass@host:5432/spoke"
export SPOKE_POSTGRES_REPLICA_URLS="postgresql://..." # Optional read replicas
export SPOKE_S3_ENDPOINT="https://s3.us-west-2.amazonaws.com"
export SPOKE_S3_BUCKET="spoke-schemas"
export SPOKE_S3_ACCESS_KEY="..."
export SPOKE_S3_SECRET_KEY="..."

# Caching
export SPOKE_REDIS_URL="redis://localhost:6379/0"
export SPOKE_CACHE_ENABLED=true

# Observability
export SPOKE_LOG_LEVEL=info
export SPOKE_OTEL_ENABLED=true
export SPOKE_OTEL_ENDPOINT="otel-collector:4317"
```

See [Deployment Guide](docs/deployment/DEPLOYMENT_GUIDE.md) for complete configuration reference.

## API Endpoints

### Core Endpoints

- `POST /modules` - Create a new module
- `GET /modules` - List all modules
- `GET /modules/{name}` - Get a specific module
- `POST /modules/{name}/versions` - Create a new version
- `GET /modules/{name}/versions` - List versions
- `GET /modules/{name}/versions/{version}` - Get version details
- `GET /modules/{name}/versions/{version}/files/{path}` - Download proto file
- `POST /modules/{name}/versions/{version}/compile/{language}` - Trigger compilation
- `GET /modules/{name}/versions/{version}/download/{language}` - Download compiled binaries

### Health Endpoints

- `GET /health/live` - Liveness probe (always returns 200 when running)
- `GET /health/ready` - Readiness probe (checks dependencies)
- `GET /metrics` - Prometheus metrics
## Documentation

### Getting Started
- [What is Spoke?](docs/content/getting-started/what-is-spoke.md)
- [Installation](docs/content/getting-started/installation.md)
- [Quick Start](docs/content/getting-started/quick-start.md)
- [First Module](docs/content/getting-started/first-module.md)

### Deployment & Operations
- **[Deployment Guide](docs/deployment/DEPLOYMENT_GUIDE.md)** - Complete deployment reference
- **[Operations Runbook](docs/operations/RUNBOOK.md)** - Daily operations and incident response
- **[Verification Checklist](docs/verification/VERIFICATION_CHECKLIST.md)** - Testing and validation
- [Kubernetes Deployment](deployments/kubernetes/README.md)
- [Docker Compose HA Stack](deployments/docker-compose/README.md)

### High Availability
- **[HA Architecture](docs/architecture/HA_ARCHITECTURE.md)** - System design and failure modes
- [PostgreSQL Replication](docs/ha/postgresql-replication.md) - Database HA setup
- [Redis Sentinel](docs/ha/redis-sentinel.md) - Cache HA setup
- [HA Implementation Status](docs/HA_IMPLEMENTATION_STATUS.md)

### Observability
- [OpenTelemetry Setup](docs/observability/otel-setup.md) - Monitoring configuration
- [CLI Reference](docs/content/guides/cli-reference.md)
- [API Reference](docs/content/guides/api-reference.md)

### Examples & Tutorials
- [gRPC Integration](docs/content/tutorials/grpc-integration.md)
- [CI/CD Integration](docs/content/guides/ci-cd.md)
- [gRPC Service Example](docs/content/examples/grpc-service.md)

## Production Readiness Checklist

Before deploying to production, verify:

- [ ] **Infrastructure**: PostgreSQL, Redis, S3 configured and tested
- [ ] **Configuration**: Environment variables set correctly
- [ ] **Secrets**: Credentials stored securely (not in ConfigMap)
- [ ] **Deployment**: Kubernetes manifests applied, 3+ replicas running
- [ ] **Health Checks**: Liveness and readiness probes passing
- [ ] **Backups**: Automated backups configured and tested
- [ ] **Monitoring**: OpenTelemetry Collector receiving metrics/traces/logs
- [ ] **Alerts**: Critical alerts configured (service down, high error rate)
- [ ] **Load Testing**: Performance baseline established
- [ ] **Runbooks**: Operations team familiar with incident procedures
- [ ] **Disaster Recovery**: Restore procedure tested

See [Verification Checklist](docs/verification/VERIFICATION_CHECKLIST.md) for detailed verification steps.

## Monitoring & Observability

### Key Metrics

Monitor these metrics for production health:

```promql
# Request rate
rate(spoke_http_requests_total[5m])

# Error rate (should be < 1%)
rate(spoke_http_requests_total{status=~"5.."}[5m]) / rate(spoke_http_requests_total[5m])

# p95 latency (should be < 500ms)
histogram_quantile(0.95, rate(spoke_http_request_duration_seconds_bucket[5m]))

# Database connections (should be < 80% of max)
spoke_db_connections_active / spoke_db_connections_max

# Cache hit rate (should be > 80%)
rate(spoke_cache_hits_total[5m]) / (rate(spoke_cache_hits_total[5m]) + rate(spoke_cache_misses_total[5m]))
```

### Dashboards

Create Grafana dashboards for:
- **Overview**: Request rate, error rate, latency percentiles
- **Database**: Connection pool, query performance, replication lag
- **Cache**: Hit rate, memory usage, evictions
- **Infrastructure**: Pod status, CPU, memory, network

See [OpenTelemetry Setup](docs/observability/otel-setup.md) for dashboard examples.

## Scaling Guide

### When to Scale

- **Horizontal (Add Pods)**:
  - CPU > 70% for > 5 minutes
  - Request latency p95 > 500ms
  - Request queue depth increasing

- **Vertical (Increase Resources)**:
  - Pods frequently OOMKilled
  - CPU throttling observed

- **Database**:
  - Connection pool > 80% utilized
  - Query latency increasing
  - Replication lag > 10 seconds

### How to Scale

```bash
# Horizontal scaling (manual)
kubectl scale deployment spoke-server -n spoke --replicas=5

# Or adjust HPA
kubectl edit hpa spoke-hpa -n spoke

# Vertical scaling
kubectl edit deployment spoke-server -n spoke
# Increase resources.limits.cpu and resources.limits.memory

# Add database read replica
# Configure SPOKE_POSTGRES_REPLICA_URLS and restart
```

See [Scaling Guide](docs/deployment/DEPLOYMENT_GUIDE.md#scaling-guide) for detailed instructions.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

See [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/platinummonkey/spoke/issues)
- **Discussions**: [GitHub Discussions](https://github.com/platinummonkey/spoke/discussions)
