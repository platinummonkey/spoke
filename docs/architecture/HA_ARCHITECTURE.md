# High Availability Architecture

Comprehensive documentation of Spoke's high availability architecture, components, and failure modes.

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Diagram](#architecture-diagram)
3. [Component Details](#component-details)
4. [Data Flow](#data-flow)
5. [Failure Modes and Recovery](#failure-modes-and-recovery)
6. [Scaling Strategy](#scaling-strategy)
7. [Observability Architecture](#observability-architecture)
8. [Security Architecture](#security-architecture)
9. [Network Architecture](#network-architecture)

---

## System Overview

Spoke is designed as a **stateless, horizontally scalable** schema registry with **persistent storage** in PostgreSQL and **object storage** in S3.

### Key Design Principles

1. **Stateless Application**: Spoke instances share no state, enabling unlimited horizontal scaling
2. **Shared Persistence**: All state in PostgreSQL (metadata) and S3 (files)
3. **Distributed Caching**: Redis for cross-instance cache coherence
4. **Fail-Open**: Graceful degradation when non-critical dependencies fail
5. **Observable by Design**: Full OpenTelemetry instrumentation

### Availability Goals

- **SLA**: 99.9% uptime (max 43.2 minutes downtime/month)
- **RTO**: < 4 hours (Recovery Time Objective)
- **RPO**: < 1 hour (Recovery Point Objective)
- **Scalability**: Horizontally scalable to handle 10,000+ req/s

---

## Architecture Diagram

### Production Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         External Clients                        │
│                    (CI/CD, Developers, Services)                │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                    ┌───────▼────────┐
                    │  Load Balancer │ (NGINX/ALB/GLB)
                    │  + TLS Termination
                    └───────┬────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
┌───────▼────────┐  ┌───────▼────────┐  ┌──────▼─────────┐
│ Spoke Instance │  │ Spoke Instance │  │ Spoke Instance │
│   (Pod 1)      │  │   (Pod 2)      │  │   (Pod 3)      │
│                │  │                │  │                │
│ • API Server   │  │ • API Server   │  │ • API Server   │
│ • Health :9090 │  │ • Health :9090 │  │ • Health :9090 │
│ • OTel Client  │  │ • OTel Client  │  │ • OTel Client  │
└───────┬────────┘  └───────┬────────┘  └────────┬───────┘
        │                   │                     │
        └───────────────────┼─────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
┌───────▼────────┐  ┌───────▼────────┐  ┌──────▼─────────┐
│  PostgreSQL    │  │     Redis      │  │   S3 Storage   │
│   Primary      │  │    Master      │  │                │
│                │  │                │  │ • Proto Files  │
│ • Modules      │  │ • Cache Layer  │  │ • Compiled     │
│ • Versions     │  │ • Rate Limits  │  │   Binaries     │
│ • Metadata     │  │                │  │                │
└───────┬────────┘  └───────┬────────┘  └────────────────┘
        │                   │
┌───────▼────────┐  ┌───────▼────────┐
│  PostgreSQL    │  │     Redis      │
│   Replica(s)   │  │   Replica(s)   │
│                │  │                │
│ • Read Queries │  │ • Automatic    │
│ • Async Repln  │  │   Failover     │
│                │  │   (Sentinel)   │
└────────────────┘  └────────────────┘
        │
┌───────▼────────┐
│ Sentinel x3    │ (monitors Redis, triggers failover)
└────────────────┘
```

### Observability Architecture

```
┌────────────────────────────────────────────────────┐
│              Spoke Instances (1-N)                 │
│  ┌──────────────────────────────────────────────┐ │
│  │ OTel Instrumentation:                        │ │
│  │ • HTTP Spans (method, route, status)         │ │
│  │ • Database Spans (queries, duration)         │ │
│  │ • S3 Spans (operations, size)                │ │
│  │ • Custom Metrics (requests, latency, cache)  │ │
│  │ • Structured Logs (JSON, trace context)      │ │
│  └───────────────────┬──────────────────────────┘ │
└────────────────────────┼───────────────────────────┘
                         │ OTLP/gRPC (4317)
                         │
                ┌────────▼─────────┐
                │ OTel Collector   │
                │                  │
                │ • Receivers      │
                │ • Processors     │
                │ • Exporters      │
                └────────┬─────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
┌───────▼────────┐ ┌─────▼──────┐ ┌──────▼─────┐
│   Prometheus   │ │   Jaeger   │ │    Loki    │
│   (Metrics)    │ │  (Traces)  │ │   (Logs)   │
└───────┬────────┘ └─────┬──────┘ └──────┬─────┘
        │                │                │
        └────────────────┼────────────────┘
                         │
                  ┌──────▼──────┐
                  │   Grafana   │
                  │  (Dashboards)
                  └─────────────┘
```

### Multi-Region Architecture (Optional)

```
┌──────────────────────────────────────────────────────────────────┐
│                      Global Load Balancer                        │
│               (Route53 Latency-Based Routing)                    │
└─────────────────────────┬────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                                   │
┌───────▼────────────────────┐    ┌─────────▼──────────────────────┐
│   Region 1 (us-west-2)     │    │   Region 2 (eu-west-1)         │
│                            │    │                                │
│ • 3 Spoke Instances        │    │ • 3 Spoke Instances            │
│ • PostgreSQL Primary       │◄───┤ • PostgreSQL Replica (async)   │
│ • Redis Sentinel Cluster   │    │ • Redis Sentinel Cluster       │
│ • S3 Primary Bucket        │◄───┤ • S3 Replica Bucket (CRR)      │
│ • OTel Collector           │    │ • OTel Collector               │
└────────────────────────────┘    └────────────────────────────────┘
      (Primary Region)                  (Disaster Recovery)
```

---

## Component Details

### Spoke Application

**Type**: Stateless, horizontally scalable Go application

**Responsibilities**:
- Serve HTTP API (port 8080)
- Health checks and metrics (port 9090)
- Read/write to PostgreSQL
- Read/write to S3
- Cache operations via Redis
- Emit telemetry to OTel Collector

**Configuration**:
- Environment variables (see [Deployment Guide](../deployment/DEPLOYMENT_GUIDE.md))
- ConfigMap in Kubernetes
- Secrets for credentials

**Scaling**:
- Horizontal: Add/remove pods via HPA
- Vertical: Increase CPU/memory limits

**Health**:
- Liveness: `/health/live` - Process alive
- Readiness: `/health/ready` - Dependencies healthy
- Startup: Allows 60s for slow startup

### PostgreSQL Database

**Type**: Relational database for structured metadata

**Schema**:
- `modules`: Module definitions
- `versions`: Version metadata
- `files`: File metadata (pointers to S3)
- `dependencies`: Module dependency graph
- `users`: Authentication (if enabled)

**Data Stored**:
- Module names and descriptions
- Version information
- File paths and checksums
- Dependency relationships
- User accounts and permissions

**NOT Stored**:
- Actual proto file contents (in S3)
- Compiled binaries (in S3)

**Replication**:
- Streaming replication (async)
- Read replicas for read queries
- Primary for all writes

**Backup**:
- Daily pg_dump backups
- Point-in-time recovery via WAL archiving (optional)
- Uploaded to S3

**Configuration**:
- Connection pooling: 20 max connections per pod (default)
- Timeouts: 10s connect, 30s query
- SSL/TLS required in production

### S3 Object Storage

**Type**: Content-addressable blob storage

**Data Stored**:
- `.proto` files (raw protobuf definitions)
- Compiled binaries (`.pb.go`, `_pb2.py`, etc.)
- Backups (PostgreSQL dumps)

**Organization**:
```
s3://spoke-schemas/
├── modules/
│   └── {module}/
│       └── {version}/
│           ├── files/
│           │   └── {sha256}.proto
│           └── compiled/
│               ├── go/
│               ├── python/
│               └── cpp/
└── backups/
    └── spoke-backup-{timestamp}.sql.gz
```

**Features**:
- Versioning enabled
- Lifecycle policies (transition to Glacier, expire)
- Cross-region replication (optional)
- Server-side encryption

**Access Pattern**:
- Writes: Only during module push
- Reads: Frequent during module pull and compilation

### Redis Cache

**Type**: In-memory key-value store for caching and rate limiting

**Data Stored**:
- Module metadata (cache of PostgreSQL data)
- Version metadata (cache of PostgreSQL data)
- Rate limit counters (token bucket state)
- Session data (if applicable)

**TTL**:
- Module cache: 5 minutes
- Version cache: 5 minutes
- Rate limit counters: Sliding window

**Eviction Policy**:
- LRU (Least Recently Used)
- Max memory: 1GB (configurable)

**Availability**:
- Redis Sentinel for automatic failover
- 3 Sentinel instances (quorum of 2)
- Fail-open: If Redis unavailable, requests succeed (slower)

**Invalidation**:
- Explicit invalidation on writes
- Pattern-based invalidation (e.g., `module:{name}:*`)
- Automatic expiration via TTL

### OpenTelemetry Collector

**Type**: Telemetry aggregation and export

**Receivers**:
- OTLP gRPC (port 4317)
- OTLP HTTP (port 4318)

**Processors**:
- Batch: Batch telemetry for efficiency
- Memory Limiter: Prevent OOM
- Resource Detection: Add environment metadata

**Exporters**:
- Prometheus (metrics)
- Jaeger (traces)
- Loki (logs)
- OTLP (forward to external services)

**Configuration**:
Users configure their own backends. Spoke only emits telemetry.

### Load Balancer

**Type**: Layer 7 load balancer (NGINX, AWS ALB, GCP GLB)

**Responsibilities**:
- Distribute traffic across Spoke instances
- TLS termination
- Health check integration
- Rate limiting (optional, in addition to app-level)
- DDoS protection

**Algorithm**:
- Round-robin or least-connections
- Sticky sessions not required (stateless app)

**Health Checks**:
- Path: `/health/ready`
- Interval: 10s
- Timeout: 5s
- Unhealthy threshold: 2 failures

---

## Data Flow

### Module Push (Write Path)

```
1. Client → Load Balancer → Spoke Instance
   POST /modules/{name}/versions
   Body: {version: "v1.0.0", files: [...]}

2. Spoke Instance:
   a. Validate request
   b. Check authentication/authorization
   c. Begin database transaction
   d. Insert module metadata into PostgreSQL (PRIMARY)
   e. Upload .proto files to S3
   f. Commit transaction
   g. Invalidate Redis cache for this module
   h. Return response

3. Redis Cache:
   - DELETE module:{name}:*
   - All Spoke instances see fresh data on next read

4. Other Spoke Instances:
   - Cache invalidated
   - Next read fetches from PostgreSQL/S3
```

**Consistency**:
- Strong consistency via PostgreSQL transaction
- Cache invalidation ensures eventual consistency across instances
- S3 is eventually consistent (typically < 1s)

### Module Pull (Read Path)

```
1. Client → Load Balancer → Spoke Instance
   GET /modules/{name}/versions/{version}

2. Spoke Instance:
   a. Check Redis cache: module:{name}:{version}
   b. If cache HIT:
      - Return cached data (fast path)
   c. If cache MISS:
      - Query PostgreSQL REPLICA (or PRIMARY if no replicas)
      - Fetch file list from database
      - Populate Redis cache
      - Return data

3. Client requests files:
   GET /modules/{name}/versions/{version}/files/{path}

4. Spoke Instance:
   a. Resolve file path to S3 key
   b. Generate signed URL (optional)
   c. Stream file from S3 → Client
```

**Performance**:
- Cache hit: < 10ms
- Cache miss + DB query: 50-100ms
- S3 file download: Depends on file size and network

### Compilation (Write + Read)

```
1. Client → Spoke Instance
   POST /modules/{name}/versions/{version}/compile/{language}

2. Spoke Instance:
   a. Fetch .proto files from S3
   b. Run protoc compiler with language plugin
   c. Upload compiled files to S3
   d. Update database with compilation status
   e. Invalidate cache
   f. Return success

3. Client downloads compiled files:
   GET /modules/{name}/versions/{version}/download/{language}

4. Spoke Instance:
   a. Check if already compiled
   b. Stream compiled archive from S3
```

**Async Compilation** (via Sprocket service):
```
1. Sprocket watches for new versions in PostgreSQL
2. Automatically triggers compilation for all languages
3. Compiles and uploads to S3
4. Clients download pre-compiled files (faster)
```

---

## Failure Modes and Recovery

### Spoke Instance Failure

**Failure**: Single pod crashes or is terminated

**Impact**: None (traffic routed to other instances)

**Detection**:
- Liveness probe failure
- Kubernetes restarts pod
- Alert: Pod restart count > 0

**Recovery**:
- Automatic via Kubernetes
- New pod starts within 30s
- Readiness probe ensures traffic only when healthy

**Prevention**:
- Run minimum 3 replicas
- Pod anti-affinity (different nodes)
- Resource limits prevent OOM

### Multiple Spoke Instances Failure

**Failure**: All pods down (e.g., bad deployment)

**Impact**: Service completely unavailable

**Detection**:
- All readiness probes failing
- Alert: Spoke service down

**Recovery**:
```bash
# Rollback to previous version
kubectl rollout undo deployment/spoke-server -n spoke

# Or scale to zero and back (reset)
kubectl scale deployment spoke-server -n spoke --replicas=0
kubectl scale deployment spoke-server -n spoke --replicas=3
```

**Prevention**:
- Rolling update strategy (maxUnavailable: 0)
- Test deployments in staging first
- Blue/green or canary deployments

### PostgreSQL Primary Failure

**Failure**: Database primary becomes unavailable

**Impact**: All writes fail, reads may continue (if replicas exist)

**Detection**:
- Database health check failure
- Alert: Database unreachable

**Recovery**:

**Managed Service (Auto-failover)**:
1. Cloud provider promotes replica to primary (1-2 minutes)
2. Connection string automatically updates
3. Spoke reconnects to new primary
4. Writes resume

**Self-Managed (Patroni)**:
```bash
# Check cluster status
patronictl -c /etc/patroni/patroni.yml list

# Automatic failover triggers within 30-60s
# Or manual failover
patronictl -c /etc/patroni/patroni.yml failover
```

**Manual Failover**:
```bash
# 1. Promote replica
psql $REPLICA_URL -c "SELECT pg_promote();"

# 2. Update Spoke configuration
kubectl edit secret spoke-secrets -n spoke
# Update SPOKE_POSTGRES_URL to new primary

# 3. Restart Spoke pods
kubectl rollout restart deployment/spoke-server -n spoke
```

**RTO**: 2-5 minutes | **RPO**: 0 seconds (synchronous replication) or seconds (async)

### PostgreSQL Replica Failure

**Failure**: Read replica becomes unavailable

**Impact**: Read queries fallback to primary (increased load)

**Detection**:
- Replica connection failure
- Spoke logs: "Replica unhealthy, using primary"

**Recovery**:
- Automatic fallback to primary
- Bring replica back online
- Spoke automatically detects and resumes using replica

**Prevention**:
- Monitor replication lag
- Alert on replica lag > 10 seconds
- Multiple replicas for redundancy

### Redis Master Failure

**Failure**: Redis master becomes unavailable

**Impact**: Cache unavailable, rate limiting degraded, slower responses

**Detection**:
- Redis health check failure
- Alert: Redis unavailable
- Increased database load (cache misses)

**Recovery** (with Sentinel):
1. Sentinel detects failure (down-after-milliseconds: 5s)
2. Quorum of Sentinels agree (quorum: 2)
3. Sentinel promotes replica to master (< 30s)
4. Spoke reconnects to new master automatically

**Graceful Degradation**:
- Spoke continues to serve requests (fail-open)
- Cache misses hit database (slower but functional)
- Rate limiting becomes per-instance (less accurate)

**RTO**: < 30 seconds | **RPO**: 0 seconds (cache is ephemeral)

### S3 Outage

**Failure**: S3 unavailable or bucket deleted

**Impact**: Cannot read/write proto files, compilation fails

**Detection**:
- S3 operation failures
- Alert: S3 upload/download errors

**Recovery**:
- Wait for S3 to recover (usually < 30 minutes for AWS)
- If bucket deleted: Restore from backup
- If region-wide outage: Failover to replica bucket (multi-region)

**Mitigation**:
- S3 versioning enabled (protects against accidental deletion)
- Cross-region replication (optional)
- Local caching of frequently accessed files (future enhancement)

**RTO**: 30 minutes - 4 hours | **RPO**: 0 (S3 is durable)

### Network Partition

**Failure**: Network split between Spoke and dependencies

**Impact**: Depends on which dependency is unreachable

**Scenarios**:

1. **Spoke ↔ PostgreSQL**:
   - Writes fail immediately
   - Reads fail after timeout
   - Health check fails → pod removed from load balancer

2. **Spoke ↔ Redis**:
   - Cache misses
   - Slower performance
   - Service continues (fail-open)

3. **Spoke ↔ S3**:
   - File operations fail
   - Module push/pull unavailable
   - Metadata queries still work

**Recovery**:
- Network heals automatically (cloud providers)
- Pods with healthy connections continue serving
- Unhealthy pods recover when network restores

### Complete Data Center Failure

**Failure**: Entire AWS region/data center unavailable

**Impact**: Service completely down (single-region deployment)

**Recovery** (Multi-Region):
1. Update DNS to point to DR region (Route53 health check)
2. Promote PostgreSQL replica in DR region
3. Update S3 bucket configuration
4. Verify service health
5. Resume operations in DR region

**Recovery** (Single-Region)**:
1. Wait for region to recover
2. Or restore from backup in different region:
   ```bash
   # Create new database in new region
   # Restore from S3 backup
   ./scripts/restore-postgres.sh s3://spoke-backups/spoke-backup-latest.sql.gz

   # Update Spoke configuration
   # Deploy Spoke to new region
   ```

**RTO**: 4 hours | **RPO**: 1 hour (backup frequency)

---

## Scaling Strategy

### Horizontal Scaling

**When to scale**:
- CPU > 70% for > 5 minutes
- Memory > 80%
- Request latency p95 > 500ms

**How to scale**:
- Automatic: HPA scales based on CPU/memory
- Manual: `kubectl scale deployment spoke-server --replicas=N`

**Limits**:
- Minimum: 3 replicas (HA requirement)
- Maximum: 10 replicas (default), unlimited (adjust HPA)
- Bottleneck: Database connection pool

**Database Connection Scaling**:
```
Total Connections = Replicas × Max Connections per Pod
Example: 5 pods × 50 = 250 total connections
Ensure PostgreSQL max_connections > 250
```

### Vertical Scaling

**When to scale**:
- Pods frequently OOMKilled
- CPU throttling observed

**How to scale**:
- Increase resource requests/limits
- Rolling update applies changes

**Considerations**:
- More expensive than horizontal scaling
- Limited by node size
- Prefer horizontal scaling for stateless apps

### Database Scaling

**Read Scaling**:
- Add read replicas
- Spoke automatically distributes read queries
- Linear scaling for read-heavy workloads

**Write Scaling**:
- Limited to primary capacity
- Scale vertically (larger instance)
- Optimize queries and indexes
- Connection pooling with PgBouncer

**Schema Changes**:
- Use online schema migration tools (e.g., gh-ost)
- Avoid locking migrations during peak hours

### Cache Scaling

**Redis Scaling**:
- Increase memory (vertical)
- Redis Cluster for sharding (horizontal)
- Spoke supports Redis Cluster URLs

**Cache Eviction**:
- Monitor eviction rate
- Increase max memory if high eviction
- Tune TTLs to reduce memory usage

### S3 Scaling

S3 automatically scales. No action required.

For very high throughput:
- Use S3 Transfer Acceleration
- Multi-part uploads for large files
- CloudFront CDN for frequently accessed files

---

## Observability Architecture

### Metrics (Prometheus)

**Categories**:
1. **HTTP Metrics**: Request count, duration, status codes
2. **Database Metrics**: Connection pool, query duration
3. **Cache Metrics**: Hit rate, eviction rate
4. **Storage Metrics**: S3 operations, sizes
5. **System Metrics**: CPU, memory, disk

**Collection**:
- Spoke exposes `/metrics` on port 9090
- Prometheus scrapes every 30s
- OTel Collector can also collect and forward

**Retention**:
- Prometheus: 15 days (default)
- Long-term: Export to remote storage (Thanos, Cortex, Mimir)

### Traces (Jaeger)

**Instrumentation**:
- HTTP requests (automatic via otelhttp)
- Database queries (manual spans)
- S3 operations (manual spans)
- Custom business logic

**Trace Context Propagation**:
- W3C Trace Context headers
- Propagated to downstream services
- Logs include trace_id for correlation

**Sampling**:
- Development: 100% (always sample)
- Production: 10% (reduce volume)
- Head-based sampling (decision at root span)

### Logs (Loki)

**Format**:
- Structured JSON
- Fields: level, message, timestamp, trace_id, span_id

**Collection**:
- Promtail (if using Loki)
- Fluentd/Fluent Bit (alternative)
- CloudWatch Logs (AWS)

**Retention**:
- 30 days (default)
- Archive to S3 for compliance

### Dashboards (Grafana)

**Recommended Dashboards**:
1. **Overview**: Request rate, error rate, latency
2. **Database**: Connection pool, query performance, replication lag
3. **Cache**: Hit rate, memory usage, evictions
4. **Infrastructure**: Pod status, CPU, memory, network
5. **SLA**: Availability percentage, error budget

**Alerts**:
- Configured in Prometheus AlertManager
- Sent to PagerDuty/Opsgenie for on-call
- Severity-based routing (P1 → page, P3 → email)

---

## Security Architecture

### Authentication & Authorization

**API Authentication**:
- JWT tokens (stateless)
- OAuth2/OIDC integration (optional)
- API keys (legacy)

**Authorization**:
- RBAC (Role-Based Access Control)
- Permissions: read, write, admin
- Module-level access control

### Network Security

**In Kubernetes**:
- Network Policies restrict pod-to-pod traffic
- Ingress from load balancer only
- Egress to PostgreSQL, Redis, S3 only

**TLS/SSL**:
- TLS termination at load balancer
- PostgreSQL connections with SSL required
- Redis connections with TLS (optional)
- S3 connections over HTTPS

### Secrets Management

**Development**:
- Kubernetes Secrets (base64 encoded)

**Production**:
- External Secrets Operator
- AWS Secrets Manager / HashiCorp Vault
- Automatic rotation

### Container Security

**Pod Security**:
- Run as non-root user (UID 65532)
- Read-only root filesystem
- Drop all capabilities
- No privilege escalation

**Image Security**:
- Base image: Distroless or Alpine
- Scan for vulnerabilities (Trivy, Snyk)
- Sign images (Sigstore)

### Data Security

**Encryption at Rest**:
- PostgreSQL: Encrypted volumes (cloud provider)
- S3: Server-side encryption (AES-256)
- Backups: Encrypted in S3

**Encryption in Transit**:
- HTTPS for API
- TLS for PostgreSQL
- TLS for Redis (optional)

---

## Network Architecture

### Kubernetes Networking

```
┌────────────────────────────────────────────────────────┐
│                    Ingress Controller                  │
│              (NGINX, port 80/443)                      │
└───────────────────────┬────────────────────────────────┘
                        │
                        │ (ClusterIP)
                        │
┌───────────────────────▼────────────────────────────────┐
│              Spoke Service (ClusterIP)                 │
│                    Port 8080                           │
└───────────────────────┬────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
┌───────▼────┐   ┌──────▼─────┐   ┌────▼──────┐
│  Pod 1     │   │   Pod 2    │   │   Pod 3   │
│  10.1.1.1  │   │  10.1.1.2  │   │ 10.1.1.3  │
└────────────┘   └────────────┘   └───────────┘
```

**Service Types**:
- Main API: ClusterIP (internal) → Ingress (external)
- Health/Metrics: Headless Service (for direct pod access)

**DNS**:
- Internal: `spoke.spoke.svc.cluster.local`
- External: `spoke.example.com` (via Ingress)

### Multi-AZ Deployment

**Node Placement**:
- Kubernetes nodes spread across 3 availability zones
- Pod anti-affinity ensures pods on different nodes/AZs
- Tolerations for node taints

**Topology**:
```
AZ 1 (us-west-2a)    AZ 2 (us-west-2b)    AZ 3 (us-west-2c)
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│ Spoke Pod 1   │    │ Spoke Pod 2   │    │ Spoke Pod 3   │
│ PostgreSQL    │    │ PostgreSQL    │    │ PostgreSQL    │
│   Primary     │    │   Replica 1   │    │   Replica 2   │
│ Redis Master  │    │ Redis Replica │    │ Redis Replica │
└───────────────┘    └───────────────┘    └───────────────┘
```

**Failure Isolation**:
- Single AZ failure: Service continues with pods in other AZs
- Database failover to replica in healthy AZ
- Redis Sentinel promotes replica in healthy AZ

---

## Summary

Spoke's high availability architecture provides:

✅ **Stateless application layer** - unlimited horizontal scaling
✅ **Redundant storage** - PostgreSQL replication, S3 durability
✅ **Distributed caching** - Redis with Sentinel for failover
✅ **Graceful degradation** - fail-open for non-critical dependencies
✅ **Comprehensive observability** - metrics, traces, logs via OpenTelemetry
✅ **Multi-AZ deployment** - survive zone failures
✅ **Automated recovery** - Kubernetes self-healing, database failover
✅ **Zero-downtime deployments** - rolling updates with health checks

The architecture supports **99.9% availability** with proper operational practices and infrastructure.

---

## Related Documentation

- [Deployment Guide](../deployment/DEPLOYMENT_GUIDE.md)
- [PostgreSQL HA Guide](../ha/postgresql-replication.md)
- [Redis Sentinel Guide](../ha/redis-sentinel.md)
- [OpenTelemetry Setup](../observability/otel-setup.md)
- [Operations Runbook](../operations/RUNBOOK.md)
- [Verification Checklist](../verification/VERIFICATION_CHECKLIST.md)
