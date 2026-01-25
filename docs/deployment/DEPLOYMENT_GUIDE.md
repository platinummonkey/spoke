# Spoke Deployment Guide

Comprehensive guide for deploying Spoke Schema Registry in development, staging, and production environments.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Environment Variables Reference](#environment-variables-reference)
3. [Development Deployment](#development-deployment)
4. [Production Deployment with Kubernetes](#production-deployment-with-kubernetes)
5. [High Availability Setup](#high-availability-setup)
6. [Monitoring and Observability](#monitoring-and-observability)
7. [Scaling Guide](#scaling-guide)
8. [Troubleshooting](#troubleshooting)

---

## Quick Start

### Minimal Local Deployment

Run Spoke locally with filesystem storage:

```bash
# Build the server
go build -o bin/spoke-server cmd/spoke/main.go

# Configure
export SPOKE_STORAGE_TYPE=filesystem
export SPOKE_FILESYSTEM_ROOT=./data/storage

# Run
./bin/spoke-server
```

Server will be available at:
- API: `http://localhost:8080`
- Health: `http://localhost:9090`

### Docker Deployment

```bash
# Pull image (when available)
docker pull platinummonkey/spoke:latest

# Or build locally
docker build -t spoke:latest .

# Run with filesystem storage
docker run -d \
  -p 8080:8080 \
  -p 9090:9090 \
  -e SPOKE_STORAGE_TYPE=filesystem \
  -e SPOKE_FILESYSTEM_ROOT=/data/storage \
  -v $(pwd)/data:/data \
  spoke:latest
```

### Docker Compose HA Stack

For local testing with full HA features:

```bash
cd deployments/docker-compose
docker-compose -f ha-stack.yml up -d
```

This starts:
- 3 Spoke instances behind NGINX load balancer
- PostgreSQL with replication
- Redis with Sentinel
- MinIO for S3 storage
- OpenTelemetry Collector

Access:
- Load-balanced API: `http://localhost:8080`
- Individual instances: `http://localhost:8081-8083`
- MinIO console: `http://localhost:9001`
- OTel Collector metrics: `http://localhost:8888/metrics`

See [Docker Compose README](../../deployments/docker-compose/README.md) for details.

---

## Environment Variables Reference

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SPOKE_HOST` | `0.0.0.0` | Server bind address |
| `SPOKE_PORT` | `8080` | Main API server port |
| `SPOKE_HEALTH_PORT` | `9090` | Health/metrics server port |
| `SPOKE_READ_TIMEOUT` | `15s` | HTTP read timeout |
| `SPOKE_WRITE_TIMEOUT` | `15s` | HTTP write timeout |
| `SPOKE_IDLE_TIMEOUT` | `60s` | HTTP idle timeout |
| `SPOKE_SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout |

### Storage Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SPOKE_STORAGE_TYPE` | `filesystem` | Storage backend: `filesystem`, `postgres`, `hybrid` |
| `SPOKE_FILESYSTEM_ROOT` | `./storage` | Filesystem storage directory |

### PostgreSQL Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SPOKE_POSTGRES_URL` | - | Primary database connection URL |
| `SPOKE_POSTGRES_REPLICA_URLS` | - | Read replica URLs (comma-separated) |
| `SPOKE_POSTGRES_MAX_CONNS` | `20` | Maximum connection pool size |
| `SPOKE_POSTGRES_MIN_CONNS` | `2` | Minimum connection pool size |
| `SPOKE_POSTGRES_TIMEOUT` | `10s` | Connection timeout |

**Example**:
```bash
export SPOKE_POSTGRES_URL="postgresql://spoke:password@primary.db.local:5432/spoke?sslmode=require"
export SPOKE_POSTGRES_REPLICA_URLS="postgresql://spoke:password@replica1.db.local:5432/spoke?sslmode=require,postgresql://spoke:password@replica2.db.local:5432/spoke?sslmode=require"
```

### S3 Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SPOKE_S3_ENDPOINT` | - | S3 endpoint URL |
| `SPOKE_S3_REGION` | `us-east-1` | S3 region |
| `SPOKE_S3_BUCKET` | - | S3 bucket name |
| `SPOKE_S3_ACCESS_KEY` | - | AWS access key ID |
| `SPOKE_S3_SECRET_KEY` | - | AWS secret access key |
| `SPOKE_S3_USE_PATH_STYLE` | `false` | Use path-style URLs |
| `SPOKE_S3_FORCE_PATH_STYLE` | `false` | Force path-style URLs |

**AWS S3 Example**:
```bash
export SPOKE_S3_ENDPOINT="https://s3.us-west-2.amazonaws.com"
export SPOKE_S3_REGION="us-west-2"
export SPOKE_S3_BUCKET="spoke-schemas"
export SPOKE_S3_ACCESS_KEY="AKIAIOSFODNN7EXAMPLE"
export SPOKE_S3_SECRET_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

**MinIO Example**:
```bash
export SPOKE_S3_ENDPOINT="http://minio:9000"
export SPOKE_S3_REGION="us-east-1"
export SPOKE_S3_BUCKET="spoke"
export SPOKE_S3_ACCESS_KEY="minioadmin"
export SPOKE_S3_SECRET_KEY="minioadmin"
export SPOKE_S3_USE_PATH_STYLE="true"
```

### Redis Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SPOKE_REDIS_URL` | - | Redis connection URL |
| `SPOKE_REDIS_PASSWORD` | - | Redis password |
| `SPOKE_REDIS_DB` | `0` | Redis database number |
| `SPOKE_REDIS_MAX_RETRIES` | `3` | Maximum retry attempts |
| `SPOKE_REDIS_POOL_SIZE` | `10` | Connection pool size |
| `SPOKE_CACHE_ENABLED` | `true` | Enable caching (when Redis configured) |
| `SPOKE_L1_CACHE_SIZE` | `10000` | In-memory L1 cache size (bytes) |

**Standalone Redis**:
```bash
export SPOKE_REDIS_URL="redis://localhost:6379/0"
export SPOKE_REDIS_PASSWORD="mypassword"
```

**Redis Sentinel** (for HA):
```bash
export SPOKE_REDIS_URL="redis://spoke-redis@sentinel1:26379,sentinel2:26379,sentinel3:26379/master/spoke-redis"
```

### Observability Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SPOKE_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `SPOKE_METRICS_ENABLED` | `true` | Enable Prometheus metrics |
| `SPOKE_OTEL_ENABLED` | `false` | Enable OpenTelemetry |
| `SPOKE_OTEL_ENDPOINT` | `localhost:4317` | OTel Collector gRPC endpoint |
| `SPOKE_OTEL_SERVICE_NAME` | `spoke-registry` | Service name for traces |
| `SPOKE_OTEL_SERVICE_VERSION` | `1.0.0` | Service version for traces |
| `SPOKE_OTEL_INSECURE` | `true` | Use insecure gRPC connection |

**Example**:
```bash
export SPOKE_OTEL_ENABLED=true
export SPOKE_OTEL_ENDPOINT="otel-collector.monitoring.svc.cluster.local:4317"
export SPOKE_OTEL_SERVICE_NAME="spoke-registry"
export SPOKE_OTEL_SERVICE_VERSION="1.2.0"
export SPOKE_OTEL_INSECURE=false  # Use TLS in production
```

---

## Development Deployment

### Local Development

1. **Clone repository**:
```bash
git clone https://github.com/platinummonkey/spoke.git
cd spoke
```

2. **Build**:
```bash
make build
# Or manually:
go build -o bin/spoke-server cmd/spoke/main.go
```

3. **Run with filesystem storage**:
```bash
export SPOKE_STORAGE_TYPE=filesystem
export SPOKE_FILESYSTEM_ROOT=./data/storage
export SPOKE_LOG_LEVEL=debug
./bin/spoke-server
```

4. **Test**:
```bash
# Health check
curl http://localhost:9090/health/live

# List modules
curl http://localhost:8080/modules
```

### Local Development with PostgreSQL

1. **Start PostgreSQL**:
```bash
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=spoke \
  -p 5432:5432 \
  postgres:16
```

2. **Start MinIO** (for S3 storage):
```bash
docker run -d \
  --name minio \
  -p 9000:9000 \
  -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Create bucket
docker exec minio mc mb /data/spoke
```

3. **Configure and run Spoke**:
```bash
export SPOKE_STORAGE_TYPE=postgres
export SPOKE_POSTGRES_URL="postgresql://postgres:postgres@localhost:5432/spoke?sslmode=disable"
export SPOKE_S3_ENDPOINT="http://localhost:9000"
export SPOKE_S3_BUCKET="spoke"
export SPOKE_S3_ACCESS_KEY="minioadmin"
export SPOKE_S3_SECRET_KEY="minioadmin"
export SPOKE_S3_USE_PATH_STYLE="true"
export SPOKE_LOG_LEVEL=debug

./bin/spoke-server
```

### Local Development with Full Stack

Use Docker Compose for complete local environment:

```bash
cd deployments/docker-compose
docker-compose -f ha-stack.yml up -d

# Tail logs
docker-compose -f ha-stack.yml logs -f spoke-1

# Stop
docker-compose -f ha-stack.yml down
```

---

## Production Deployment with Kubernetes

### Prerequisites

- Kubernetes cluster (1.25+)
- kubectl configured
- Managed PostgreSQL (AWS RDS, Google Cloud SQL, etc.)
- Managed Redis (AWS ElastiCache, etc.)
- S3 or compatible object storage
- NGINX Ingress Controller (optional)
- cert-manager for TLS (optional)

### Step 1: Prepare Infrastructure

#### PostgreSQL Setup

Create managed PostgreSQL instance with:
- PostgreSQL 16+
- Multi-AZ deployment
- Automated backups
- Read replicas (optional, for HA)

**AWS RDS Example**:
```bash
aws rds create-db-instance \
  --db-instance-identifier spoke-db \
  --db-instance-class db.t3.medium \
  --engine postgres \
  --engine-version 16.1 \
  --master-username spoke \
  --master-user-password <password> \
  --allocated-storage 100 \
  --storage-type gp3 \
  --multi-az \
  --backup-retention-period 7 \
  --vpc-security-group-ids sg-xxx \
  --db-subnet-group-name spoke-db-subnet
```

#### Redis Setup

Create managed Redis with:
- Redis 7+
- Cluster mode or Sentinel for HA
- Multi-AZ deployment
- Automated backups

**AWS ElastiCache Example**:
```bash
aws elasticache create-replication-group \
  --replication-group-id spoke-redis \
  --replication-group-description "Spoke cache" \
  --engine redis \
  --engine-version 7.0 \
  --cache-node-type cache.t3.medium \
  --num-cache-clusters 3 \
  --automatic-failover-enabled \
  --multi-az-enabled
```

#### S3 Bucket Setup

```bash
aws s3 mb s3://spoke-schemas

# Enable versioning
aws s3api put-bucket-versioning \
  --bucket spoke-schemas \
  --versioning-configuration Status=Enabled

# Set lifecycle policy (optional)
aws s3api put-bucket-lifecycle-configuration \
  --bucket spoke-schemas \
  --lifecycle-configuration file://s3-lifecycle.json
```

### Step 2: Create Kubernetes Namespace

```bash
kubectl create namespace spoke
```

### Step 3: Create Secrets

**Never commit secrets to git!** Use one of these methods:

#### Method 1: Kubernetes Secrets (basic)

```bash
# Create secret from literal values
kubectl create secret generic spoke-secrets \
  --namespace spoke \
  --from-literal=SPOKE_POSTGRES_URL="postgresql://spoke:password@db.example.com:5432/spoke?sslmode=require" \
  --from-literal=SPOKE_REDIS_URL="redis://redis.example.com:6379/0" \
  --from-literal=SPOKE_S3_ACCESS_KEY="AKIAIOSFODNN7EXAMPLE" \
  --from-literal=SPOKE_S3_SECRET_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

#### Method 2: External Secrets Operator (recommended)

Install External Secrets Operator:
```bash
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets-system --create-namespace
```

Create SecretStore and ExternalSecret:
```yaml
# secretstore.yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets-manager
  namespace: spoke
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-west-2
      auth:
        jwt:
          serviceAccountRef:
            name: spoke

---
# externalsecret.yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: spoke-secrets
  namespace: spoke
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: spoke-secrets
  data:
  - secretKey: SPOKE_POSTGRES_URL
    remoteRef:
      key: spoke/postgres-url
  - secretKey: SPOKE_REDIS_URL
    remoteRef:
      key: spoke/redis-url
  - secretKey: SPOKE_S3_ACCESS_KEY
    remoteRef:
      key: spoke/s3-access-key
  - secretKey: SPOKE_S3_SECRET_KEY
    remoteRef:
      key: spoke/s3-secret-key
```

### Step 4: Update Configuration

Edit `deployments/kubernetes/spoke-config.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: spoke-config
  namespace: spoke
data:
  SPOKE_STORAGE_TYPE: "postgres"
  SPOKE_S3_ENDPOINT: "https://s3.us-west-2.amazonaws.com"
  SPOKE_S3_REGION: "us-west-2"
  SPOKE_S3_BUCKET: "spoke-schemas"
  SPOKE_POSTGRES_MAX_CONNS: "50"
  SPOKE_POSTGRES_MIN_CONNS: "5"
  SPOKE_REDIS_POOL_SIZE: "20"
  SPOKE_CACHE_ENABLED: "true"
  SPOKE_LOG_LEVEL: "info"
  SPOKE_METRICS_ENABLED: "true"
  SPOKE_OTEL_ENABLED: "true"
  SPOKE_OTEL_ENDPOINT: "otel-collector.monitoring.svc.cluster.local:4317"
  SPOKE_OTEL_SERVICE_NAME: "spoke-registry"
  SPOKE_OTEL_SERVICE_VERSION: "1.0.0"
```

### Step 5: Deploy

```bash
# Apply configuration
kubectl apply -f deployments/kubernetes/spoke-config.yaml

# Deploy Spoke
kubectl apply -f deployments/kubernetes/spoke-deployment.yaml

# Enable autoscaling
kubectl apply -f deployments/kubernetes/spoke-hpa.yaml

# Optional: Create Ingress
kubectl apply -f deployments/kubernetes/spoke-ingress.yaml
```

### Step 6: Verify Deployment

```bash
# Check pods
kubectl get pods -n spoke
# Expected: 3/3 pods running

# Check deployment
kubectl get deployment -n spoke
# Expected: 3/3 READY

# Check services
kubectl get svc -n spoke

# Check HPA
kubectl get hpa -n spoke
# Expected: 3-10 replicas, based on load

# Test health
kubectl port-forward -n spoke deployment/spoke-server 9090:9090
curl http://localhost:9090/health/ready
```

### Step 7: Configure Ingress (Optional)

Edit `deployments/kubernetes/spoke-ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: spoke-ingress
  namespace: spoke
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - spoke.yourdomain.com
    secretName: spoke-tls
  rules:
  - host: spoke.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: spoke
            port:
              number: 8080
```

Apply:
```bash
kubectl apply -f deployments/kubernetes/spoke-ingress.yaml
```

### Step 8: Setup Monitoring

See [OpenTelemetry Setup Guide](../observability/otel-setup.md) for detailed monitoring setup.

Quick setup:
```bash
# Deploy OTel Collector
kubectl apply -f https://raw.githubusercontent.com/open-telemetry/opentelemetry-operator/main/bundle.yaml

# Create collector instance
kubectl apply -f deployments/kubernetes/otel-collector.yaml
```

---

## High Availability Setup

### Multi-Region Deployment

For global HA, deploy Spoke in multiple regions:

```
Region 1 (us-west-2)          Region 2 (eu-west-1)
┌─────────────────┐           ┌─────────────────┐
│ Spoke (3 pods)  │           │ Spoke (3 pods)  │
│ PostgreSQL      │◄─────────►│ PostgreSQL      │
│ Redis Sentinel  │   Sync    │ Redis Sentinel  │
│ S3 (primary)    │           │ S3 (replica)    │
└─────────────────┘           └─────────────────┘
         │                             │
         └──────── Route53 ────────────┘
              (latency-based)
```

**Steps**:

1. Deploy Spoke in each region
2. Setup PostgreSQL cross-region replication
3. Enable S3 cross-region replication
4. Configure Route53 for latency-based routing
5. Use Redis Sentinel in each region (separate clusters)

### Database High Availability

**Option 1: Managed Service with Auto-Failover**
- AWS RDS Multi-AZ
- Google Cloud SQL HA
- Azure Database for PostgreSQL HA

**Option 2: Self-Managed with Patroni**

See [PostgreSQL HA Guide](../ha/postgresql-replication.md) for detailed setup.

### Redis High Availability

**Option 1: Managed Service**
- AWS ElastiCache with Auto-Failover
- Google Cloud Memorystore HA
- Azure Cache for Redis

**Option 2: Self-Managed with Sentinel**

See [Redis Sentinel Guide](../ha/redis-sentinel.md) for detailed setup.

### Zero-Downtime Deployments

Spoke supports zero-downtime deployments with proper configuration:

1. **Rolling Updates**: Kubernetes gradually replaces pods
2. **Health Checks**: Traffic only sent to ready pods
3. **preStop Hook**: Waits for connections to drain
4. **maxUnavailable: 0**: Never remove pods before replacement ready

**Deploy new version**:
```bash
# Update image
kubectl set image deployment/spoke-server spoke=spoke:v1.2.0 -n spoke

# Watch rollout
kubectl rollout status deployment/spoke-server -n spoke

# Verify
kubectl get pods -n spoke
```

**Rollback if needed**:
```bash
kubectl rollout undo deployment/spoke-server -n spoke
```

---

## Monitoring and Observability

### Health Checks

Spoke exposes health endpoints on port 9090:

- **Liveness**: `/health/live` - Returns 200 if server is running
- **Readiness**: `/health/ready` - Returns 200 if dependencies are healthy

**Kubernetes probes**:
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 9090
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health/ready
    port: 9090
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Metrics

Spoke exposes Prometheus metrics at `/metrics` on port 9090.

**Key metrics**:
- `spoke_http_requests_total` - Total HTTP requests
- `spoke_http_request_duration_seconds` - Request latency histogram
- `spoke_db_connections_active` - Active database connections
- `spoke_db_connections_idle` - Idle database connections
- `spoke_cache_hits_total` - Cache hits
- `spoke_cache_misses_total` - Cache misses
- `spoke_storage_operations_total` - Storage operations

**Prometheus scrape config**:
```yaml
scrape_configs:
  - job_name: 'spoke'
    kubernetes_sd_configs:
    - role: pod
      namespaces:
        names:
        - spoke
    relabel_configs:
    - source_labels: [__meta_kubernetes_pod_label_app]
      action: keep
      regex: spoke
    - source_labels: [__meta_kubernetes_pod_container_port_number]
      action: keep
      regex: "9090"
```

### OpenTelemetry

Spoke supports OpenTelemetry for distributed tracing, metrics, and logs.

**Enable**:
```bash
export SPOKE_OTEL_ENABLED=true
export SPOKE_OTEL_ENDPOINT=otel-collector:4317
```

**What's instrumented**:
- HTTP requests (automatic via otelhttp)
- Database queries
- S3 operations
- Cache operations
- Custom business metrics

See [OpenTelemetry Setup](../observability/otel-setup.md) for complete guide.

### Logs

Spoke outputs structured JSON logs to stdout.

**Log fields**:
- `level`: Log level (debug, info, warn, error)
- `message`: Log message
- `timestamp`: RFC3339 timestamp
- `trace_id`: OpenTelemetry trace ID (when in request context)
- `span_id`: OpenTelemetry span ID (when in request context)
- Additional contextual fields

**Aggregation with Loki**:
```bash
# Install Loki and Promtail
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack -n monitoring

# Query logs
curl -G -s "http://loki:3100/loki/api/v1/query" \
  --data-urlencode 'query={namespace="spoke",app="spoke"}'
```

---

## Scaling Guide

### Horizontal Scaling

#### Manual Scaling

```bash
# Scale up
kubectl scale deployment spoke-server -n spoke --replicas=5

# Scale down
kubectl scale deployment spoke-server -n spoke --replicas=3
```

#### Auto-Scaling with HPA

HPA automatically scales based on CPU and memory:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: spoke-hpa
  namespace: spoke
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: spoke-server
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

**Monitor HPA**:
```bash
kubectl get hpa -n spoke --watch
```

#### Custom Metrics Scaling

Scale based on custom metrics (e.g., request rate):

```yaml
metrics:
- type: Pods
  pods:
    metric:
      name: http_requests_per_second
    target:
      type: AverageValue
      averageValue: "1000"
```

### Vertical Scaling

Increase resources per pod:

```yaml
resources:
  requests:
    memory: "1Gi"    # Increased from 512Mi
    cpu: "1000m"     # Increased from 500m
  limits:
    memory: "4Gi"    # Increased from 2Gi
    cpu: "4000m"     # Increased from 2000m
```

Apply changes:
```bash
kubectl apply -f deployments/kubernetes/spoke-deployment.yaml
kubectl rollout status deployment/spoke-server -n spoke
```

### Database Scaling

#### Connection Pool Tuning

Adjust based on pod count and load:

```bash
# If running 5 pods with max 50 connections each
# Total = 5 * 50 = 250 connections
# Ensure PostgreSQL max_connections > 250

export SPOKE_POSTGRES_MAX_CONNS=50
export SPOKE_POSTGRES_MIN_CONNS=10
```

#### Read Replicas

Add read replicas for read-heavy workloads:

```bash
export SPOKE_POSTGRES_REPLICA_URLS="postgresql://spoke:pass@replica1:5432/spoke,postgresql://spoke:pass@replica2:5432/spoke"
```

Spoke automatically routes:
- **Reads**: Load-balanced across replicas
- **Writes**: Always to primary

### Redis Scaling

#### Connection Pool Tuning

```bash
export SPOKE_REDIS_POOL_SIZE=30  # Increase from 10
```

#### Redis Cluster (for very large deployments)

Switch from Sentinel to Redis Cluster:

```bash
export SPOKE_REDIS_URL="redis://node1:7000,node2:7000,node3:7000"
```

### S3 Scaling

S3 automatically scales. No action needed.

For very high throughput, use S3 Transfer Acceleration:

```bash
export SPOKE_S3_ENDPOINT="https://spoke-schemas.s3-accelerate.amazonaws.com"
```

---

## Troubleshooting

### Pods Not Starting

**Symptoms**: Pods stuck in `Pending` or `ContainerCreating`

**Diagnosis**:
```bash
kubectl describe pod -n spoke <pod-name>
```

**Common causes**:
1. **Image pull errors**: Check image name and registry credentials
2. **Resource constraints**: Not enough CPU/memory on nodes
3. **PVC issues**: Storage class not available
4. **Security policies**: PSP/PSA blocking pod

**Solutions**:
```bash
# Check node resources
kubectl top nodes

# Check events
kubectl get events -n spoke --sort-by='.lastTimestamp'

# Adjust resource requests if needed
```

### CrashLoopBackOff

**Symptoms**: Pods repeatedly crashing

**Diagnosis**:
```bash
# View current logs
kubectl logs -n spoke <pod-name>

# View previous container logs
kubectl logs -n spoke <pod-name> -p

# Check exit code
kubectl describe pod -n spoke <pod-name> | grep "Exit Code"
```

**Common causes**:
1. **Database connection failure**: Check PostgreSQL URL and credentials
2. **Invalid configuration**: Check ConfigMap and Secrets
3. **Missing dependencies**: PostgreSQL or Redis not available
4. **Resource limits**: OOMKilled (out of memory)

**Solutions**:
```bash
# Test database connection
kubectl run -it --rm psql --image=postgres:16 -- psql "$SPOKE_POSTGRES_URL"

# Test Redis connection
kubectl run -it --rm redis --image=redis:7 -- redis-cli -u "$SPOKE_REDIS_URL" ping

# Increase memory limits if OOMKilled
```

### High Latency

**Symptoms**: Slow API responses

**Diagnosis**:
```bash
# Check pod CPU/memory
kubectl top pods -n spoke

# Check HPA
kubectl get hpa -n spoke

# Check metrics
curl http://localhost:9090/metrics | grep http_request_duration
```

**Common causes**:
1. **CPU throttling**: Pods hitting CPU limits
2. **Database slow**: Connection pool exhausted or slow queries
3. **Cache misses**: Redis not working
4. **Network issues**: High latency to PostgreSQL/S3

**Solutions**:
```bash
# Scale up
kubectl scale deployment spoke-server -n spoke --replicas=5

# Increase CPU limits
# Edit spoke-deployment.yaml and increase limits.cpu

# Check database connection pool
kubectl logs -n spoke <pod-name> | grep "connection pool"

# Verify Redis working
kubectl exec -n spoke <pod-name> -- sh -c 'curl localhost:9090/health/ready | jq .checks.redis'
```

### Database Connection Pool Exhausted

**Symptoms**: Errors about "too many connections"

**Diagnosis**:
```bash
# Check current connections
kubectl exec -n spoke <pod-name> -- sh -c 'curl localhost:9090/metrics | grep db_connections'
```

**Solution**:
```bash
# Increase pool size
kubectl edit configmap spoke-config -n spoke
# Set SPOKE_POSTGRES_MAX_CONNS: "50"

# Restart pods
kubectl rollout restart deployment/spoke-server -n spoke
```

### Redis Connection Issues

**Symptoms**: Cache not working, high database load

**Diagnosis**:
```bash
# Check Redis health
kubectl exec -n spoke <pod-name> -- sh -c 'curl localhost:9090/health/ready | jq .checks.redis'

# Test Redis directly
kubectl run -it --rm redis --image=redis:7 -- redis-cli -u "$SPOKE_REDIS_URL" ping
```

**Solution**:
Spoke fails open - if Redis unavailable, requests still succeed (just slower).

```bash
# Check Redis logs
kubectl logs -n <redis-namespace> <redis-pod>

# Restart Redis (if self-hosted)
kubectl rollout restart statefulset/redis -n <redis-namespace>
```

### S3 Upload Failures

**Symptoms**: Errors uploading protobuf files

**Diagnosis**:
```bash
# Check S3 credentials
kubectl get secret spoke-secrets -n spoke -o yaml

# Test S3 access
aws s3 ls s3://spoke-schemas/
```

**Common causes**:
1. **Invalid credentials**: Wrong access key or secret key
2. **Permission denied**: IAM policy doesn't allow PutObject
3. **Bucket doesn't exist**: Bucket not created
4. **Network issues**: Can't reach S3 endpoint

**Solution**:
```bash
# Update credentials
kubectl create secret generic spoke-secrets -n spoke \
  --from-literal=SPOKE_S3_ACCESS_KEY="<new-key>" \
  --from-literal=SPOKE_S3_SECRET_KEY="<new-secret>" \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart pods
kubectl rollout restart deployment/spoke-server -n spoke
```

### Ingress Not Working

**Symptoms**: Can't access Spoke via domain

**Diagnosis**:
```bash
# Check Ingress
kubectl get ingress -n spoke
kubectl describe ingress spoke-ingress -n spoke

# Check NGINX logs
kubectl logs -n ingress-nginx <nginx-pod>
```

**Common causes**:
1. **DNS not configured**: Domain doesn't point to load balancer
2. **TLS cert not ready**: cert-manager still issuing certificate
3. **Ingress controller not installed**: NGINX Ingress not running

**Solution**:
```bash
# Check load balancer IP
kubectl get ingress spoke-ingress -n spoke -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# Update DNS to point to this IP

# Check cert-manager
kubectl get certificate -n spoke
kubectl describe certificate spoke-tls -n spoke
```

---

## Backup and Disaster Recovery

See [Backup Scripts](../../scripts/) for automated backup procedures.

### Automated Backups

Schedule backups as Kubernetes CronJob:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: spoke-backup
  namespace: spoke
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:16
            command: ["/scripts/backup-postgres.sh"]
            env:
            - name: SPOKE_POSTGRES_URL
              valueFrom:
                secretKeyRef:
                  name: spoke-secrets
                  key: SPOKE_POSTGRES_URL
            - name: SPOKE_S3_BUCKET
              value: "spoke-backups"
            volumeMounts:
            - name: scripts
              mountPath: /scripts
          restartPolicy: OnFailure
          volumes:
          - name: scripts
            configMap:
              name: backup-scripts
              defaultMode: 0755
```

### Manual Restore

```bash
# Download latest backup
aws s3 cp s3://spoke-backups/spoke-backup-latest.sql.gz /tmp/

# Restore
./scripts/restore-postgres.sh /tmp/spoke-backup-latest.sql.gz

# Restart Spoke pods
kubectl rollout restart deployment/spoke-server -n spoke
```

---

## Next Steps

After deployment:

1. **Setup Monitoring**: See [OpenTelemetry Setup](../observability/otel-setup.md)
2. **Configure Backups**: Setup automated backups with CronJob
3. **Test Failover**: Verify HA features work as expected
4. **Load Testing**: Establish baseline performance
5. **Create Runbooks**: Document operational procedures
6. **Setup Alerts**: Configure alerting for critical metrics

---

## Additional Resources

- [Kubernetes Deployment README](../../deployments/kubernetes/README.md)
- [Docker Compose HA Stack](../../deployments/docker-compose/README.md)
- [PostgreSQL HA Guide](../ha/postgresql-replication.md)
- [Redis Sentinel Guide](../ha/redis-sentinel.md)
- [OpenTelemetry Setup](../observability/otel-setup.md)
- [Verification Checklist](../verification/VERIFICATION_CHECKLIST.md)
