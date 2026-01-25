# Kubernetes Deployment for Spoke

This directory contains Kubernetes manifests for deploying Spoke in a production environment with high availability.

## Prerequisites

- Kubernetes 1.25+
- kubectl configured
- PostgreSQL instance (managed service or self-hosted)
- Redis instance (managed service or self-hosted)
- S3-compatible object storage (AWS S3, MinIO, etc.)
- NGINX Ingress Controller (optional)
- cert-manager (optional, for TLS)

## Quick Start

### 1. Create Namespace

```bash
kubectl create namespace spoke
```

### 2. Update Configuration

Edit `spoke-config.yaml` and `spoke-config.yaml` (secrets section):

- Update PostgreSQL URLs
- Update Redis URL
- Update S3 credentials
- Adjust replica counts if needed

**Important**: Never commit secrets to git. Use external secret management:
- Kubernetes Secrets (encrypted at rest)
- HashiCorp Vault
- AWS Secrets Manager
- Google Secret Manager

### 3. Apply Manifests

```bash
# Apply in order
kubectl apply -f spoke-config.yaml
kubectl apply -f spoke-deployment.yaml
kubectl apply -f spoke-hpa.yaml
kubectl apply -f spoke-ingress.yaml  # Optional
```

### 4. Verify Deployment

```bash
# Check pods
kubectl get pods -n spoke

# Check deployment
kubectl get deployment -n spoke

# Check services
kubectl get svc -n spoke

# Check HPA
kubectl get hpa -n spoke
```

Expected output:

```
NAME             READY   UP-TO-DATE   AVAILABLE   AGE
spoke-server     3/3     3            3           2m

NAME                 REFERENCE                 TARGETS         MINPODS   MAXPODS   REPLICAS
spoke-hpa            Deployment/spoke-server   45%/70%, 60%/80%   3         10        3
```

### 5. Test Health Endpoints

```bash
# Port-forward to a pod
kubectl port-forward -n spoke deployment/spoke-server 9090:9090

# Test liveness
curl http://localhost:9090/health/live

# Test readiness
curl http://localhost:9090/health/ready
```

### 6. Access the API

```bash
# Port-forward to access API
kubectl port-forward -n spoke svc/spoke 8080:8080

# Test API
curl http://localhost:8080/modules
```

## Configuration

### Environment Variables

All configuration is done via ConfigMap and Secret in `spoke-config.yaml`.

**Key settings**:
- `SPOKE_STORAGE_TYPE`: Set to `postgres` for production
- `SPOKE_POSTGRES_URL`: Primary database URL
- `SPOKE_POSTGRES_REPLICA_URLS`: Read replica URLs (comma-separated)
- `SPOKE_REDIS_URL`: Redis connection string
- `SPOKE_OTEL_ENABLED`: Enable OpenTelemetry (recommended)

### Resource Limits

Adjust in `spoke-deployment.yaml`:

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "2000m"
```

**Guidelines**:
- Small cluster (<100 modules): 512Mi/500m request
- Medium cluster (100-1000 modules): 1Gi/1000m request
- Large cluster (>1000 modules): 2Gi/2000m request

### Scaling

**Manual scaling**:

```bash
kubectl scale deployment spoke-server -n spoke --replicas=5
```

**Auto-scaling** (via HPA):

- Minimum: 3 replicas (for HA)
- Maximum: 10 replicas (adjust based on load)
- CPU target: 70%
- Memory target: 80%

Edit `spoke-hpa.yaml` to adjust thresholds.

### Ingress

Update `spoke-ingress.yaml`:

- Replace `spoke.example.com` with your domain
- Configure TLS certificate
- Adjust rate limits if needed

**Without Ingress**:

Use LoadBalancer service:

```bash
kubectl patch svc spoke -n spoke -p '{"spec":{"type":"LoadBalancer"}}'
```

## Monitoring

### Prometheus

Spoke exposes metrics on port 9090 at `/metrics`.

**ServiceMonitor** (if using Prometheus Operator):

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: spoke
  namespace: spoke
spec:
  selector:
    matchLabels:
      app: spoke
  endpoints:
  - port: health
    path: /metrics
    interval: 30s
```

### OpenTelemetry

Enable in `spoke-config.yaml`:

```yaml
SPOKE_OTEL_ENABLED: "true"
SPOKE_OTEL_ENDPOINT: "otel-collector:4317"
```

Deploy OpenTelemetry Collector separately (see [OTel setup guide](../../docs/observability/otel-setup.md)).

### Logs

View logs:

```bash
# All pods
kubectl logs -n spoke -l app=spoke -f

# Specific pod
kubectl logs -n spoke <pod-name> -f

# Previous container (after crash)
kubectl logs -n spoke <pod-name> -p
```

## High Availability

### Pod Distribution

The deployment uses `podAntiAffinity` to prefer running pods on different nodes.

**Hard anti-affinity** (prevents pods on same node):

```yaml
podAntiAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
  - labelSelector:
      matchExpressions:
      - key: app
        operator: In
        values:
        - spoke
    topologyKey: kubernetes.io/hostname
```

### Zero-Downtime Deployments

The deployment uses:
- `maxSurge: 1` - Add 1 new pod before removing old
- `maxUnavailable: 0` - Never remove pods before replacement is ready
- `preStop` hook - Wait 15s before terminating (for load balancer drain)

### Health Checks

- **Liveness**: Restarts container if unhealthy
- **Readiness**: Removes from load balancer if not ready
- **Startup**: Allows slow startup (up to 60s)

### Database Failover

If using managed PostgreSQL with automatic failover:
1. Database fails over to replica
2. Connection string updated by cloud provider
3. Spoke reconnects automatically (connection pool refresh)

If self-hosted:
- Use Patroni for automatic failover
- Update `SPOKE_POSTGRES_URL` secret
- Rolling restart: `kubectl rollout restart deployment/spoke-server -n spoke`

## Backup and Disaster Recovery

### Database Backups

See [backup scripts](../../scripts/backup-postgres.sh) for automated backups.

Schedule as CronJob:

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

### Disaster Recovery

**RTO** (Recovery Time Objective): < 4 hours
**RPO** (Recovery Point Objective): < 1 hour

**Recovery procedure**:
1. Restore PostgreSQL from latest backup (see [PostgreSQL HA guide](../../docs/ha/postgresql-replication.md))
2. Update database connection in secrets
3. Apply Kubernetes manifests
4. Verify health checks
5. Test API endpoints

## Troubleshooting

### Pods Not Starting

**Check events**:

```bash
kubectl describe pod -n spoke <pod-name>
```

**Common issues**:
- Image pull errors: Check image name and registry access
- Init container failures: Check database/Redis connectivity
- Resource constraints: Check node capacity

### CrashLoopBackOff

**Check logs**:

```bash
kubectl logs -n spoke <pod-name> -p
```

**Common causes**:
- Database connection failure
- Invalid configuration
- Missing secrets
- Resource limits too low

### Degraded Performance

**Check HPA**:

```bash
kubectl get hpa -n spoke
kubectl describe hpa spoke-hpa -n spoke
```

**Check resource usage**:

```bash
kubectl top pods -n spoke
kubectl top nodes
```

**Check database connections**:

```bash
kubectl exec -n spoke <pod-name> -- sh -c 'curl localhost:9090/health/ready | jq .'
```

### Database Connection Pool Exhausted

Increase `SPOKE_POSTGRES_MAX_CONNS` in `spoke-config.yaml`:

```yaml
SPOKE_POSTGRES_MAX_CONNS: "50"  # Increase from 20
```

Then restart:

```bash
kubectl rollout restart deployment/spoke-server -n spoke
```

## Security

### Network Policies

Restrict pod-to-pod communication:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: spoke-network-policy
  namespace: spoke
spec:
  podSelector:
    matchLabels:
      app: spoke
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
    - protocol: TCP
      port: 6379  # Redis
```

### Pod Security

Enabled by default in `spoke-deployment.yaml`:
- Run as non-root user
- Read-only root filesystem
- Drop all capabilities
- Prevent privilege escalation

### Secrets Management

Use external secrets operator:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: spoke-secrets
  namespace: spoke
spec:
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: spoke-secrets
  data:
  - secretKey: SPOKE_POSTGRES_URL
    remoteRef:
      key: spoke/postgres-url
```

## Cleanup

```bash
kubectl delete namespace spoke
```

This removes all Spoke resources.

## Further Reading

- [OpenTelemetry Setup](../../docs/observability/otel-setup.md)
- [PostgreSQL HA](../../docs/ha/postgresql-replication.md)
- [Redis Sentinel](../../docs/ha/redis-sentinel.md)
- [Backup Scripts](../../scripts/)
