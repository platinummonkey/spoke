# Spoke Operations Runbook

Operations guide for managing Spoke Schema Registry in production.

---

## Table of Contents

1. [Daily Operations](#daily-operations)
2. [Incident Response](#incident-response)
3. [Backup and Restore](#backup-and-restore)
4. [Scaling Operations](#scaling-operations)
5. [Monitoring and Alerts](#monitoring-and-alerts)
6. [Maintenance Windows](#maintenance-windows)
7. [Common Tasks](#common-tasks)

---

## Daily Operations

### Health Check Routine

Perform daily health checks to ensure system stability.

**Morning Health Check** (5 minutes):

```bash
# 1. Check pod status
kubectl get pods -n spoke
# Expected: All pods Running, 3/3 READY

# 2. Check HPA status
kubectl get hpa -n spoke
# Expected: Current replicas between min and max

# 3. Test API endpoint
curl https://spoke.example.com/modules
# Expected: 200 OK response

# 4. Test health endpoints
kubectl port-forward -n spoke deployment/spoke-server 9090:9090 &
curl http://localhost:9090/health/ready | jq .
# Expected: "status": "healthy"

# 5. Check metrics
curl http://localhost:9090/metrics | grep -E "(http_requests_total|db_connections)"
# Expected: Reasonable values, no anomalies
```

**Weekly Health Check** (15 minutes):

```bash
# 1. Review error rates
kubectl logs -n spoke -l app=spoke --since=7d | grep -i error | wc -l
# Expected: Low error count

# 2. Check database replication lag (if using replicas)
# Connect to PostgreSQL and run:
# SELECT EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())) AS lag_seconds;
# Expected: < 10 seconds

# 3. Check Redis Sentinel status (if using Sentinel)
kubectl exec -n redis redis-sentinel-1 -- redis-cli -p 26379 SENTINEL masters
# Expected: Master status OK, all sentinels agree

# 4. Review backup status
aws s3 ls s3://spoke-backups/ | tail -5
# Expected: Recent backups (within 24 hours)

# 5. Check disk usage (S3 bucket size)
aws s3 ls s3://spoke-schemas/ --recursive --summarize | grep "Total Size"
# Expected: Growing but within budget

# 6. Review resource utilization
kubectl top pods -n spoke
kubectl top nodes
# Expected: CPU/memory within acceptable ranges
```

### Log Monitoring

**Real-time log tailing**:

```bash
# Tail all Spoke pods
kubectl logs -n spoke -l app=spoke -f

# Tail specific pod
kubectl logs -n spoke spoke-server-xxx -f

# Filter errors only
kubectl logs -n spoke -l app=spoke --since=1h | grep '"level":"error"'
```

**Log aggregation** (if using Loki):

```promql
# Query errors in last hour
{namespace="spoke", app="spoke"} |= "error" | json | __error__ = ""

# Query slow requests (> 1s)
{namespace="spoke", app="spoke"} | json | duration > 1.0

# Query by trace ID
{namespace="spoke"} | json | trace_id="abc123..."
```

### Metrics Dashboard

**Key metrics to monitor daily**:

| Metric | Threshold | Action |
|--------|-----------|--------|
| Request rate | - | Normal fluctuation expected |
| Error rate | < 1% | Investigate if > 1% |
| p95 latency | < 500ms | Investigate if > 500ms |
| p99 latency | < 1s | Investigate if > 1s |
| Database connections | < 80% of max | Scale if approaching limit |
| Cache hit rate | > 80% | Investigate if < 80% |
| Pod restarts | 0 | Investigate any restarts |
| Disk usage | < 80% | Plan capacity if > 80% |

**Prometheus queries**:

```promql
# Request rate
rate(spoke_http_requests_total[5m])

# Error rate
rate(spoke_http_requests_total{status=~"5.."}[5m]) / rate(spoke_http_requests_total[5m])

# p95 latency
histogram_quantile(0.95, rate(spoke_http_request_duration_seconds_bucket[5m]))

# Database connection usage
spoke_db_connections_active / spoke_db_connections_max

# Cache hit rate
rate(spoke_cache_hits_total[5m]) / (rate(spoke_cache_hits_total[5m]) + rate(spoke_cache_misses_total[5m]))
```

---

## Incident Response

### Severity Levels

| Severity | Definition | Response Time | Example |
|----------|------------|---------------|---------|
| **P1** | Service down, data loss | Immediate | All pods crashed, database corruption |
| **P2** | Major degradation | < 15 min | High error rate, severe latency |
| **P3** | Minor degradation | < 1 hour | Single pod down, cache unavailable |
| **P4** | No user impact | Next business day | Slow background job |

### Incident Response Checklist

For all incidents:

1. **Acknowledge** - Acknowledge alert in monitoring system
2. **Assess** - Determine severity and impact
3. **Communicate** - Notify stakeholders (for P1/P2)
4. **Investigate** - Identify root cause
5. **Mitigate** - Implement immediate fix
6. **Resolve** - Confirm issue resolved
7. **Document** - Create incident report
8. **Follow-up** - Implement long-term fixes

### P1: Complete Service Outage

**Symptoms**: All API requests failing, all pods down

**Investigation**:

```bash
# 1. Check pod status
kubectl get pods -n spoke
kubectl describe pods -n spoke

# 2. Check recent events
kubectl get events -n spoke --sort-by='.lastTimestamp' | tail -20

# 3. Check deployment
kubectl get deployment -n spoke
kubectl describe deployment spoke-server -n spoke

# 4. Check node health
kubectl get nodes
kubectl describe node <node-name>
```

**Common Causes**:

1. **Deployment failure**: Bad configuration in recent deployment
2. **Node failure**: All nodes hosting Spoke pods failed
3. **Database failure**: PostgreSQL completely unavailable
4. **Configuration error**: Invalid secrets or config

**Mitigation**:

```bash
# Option 1: Rollback deployment
kubectl rollout undo deployment/spoke-server -n spoke
kubectl rollout status deployment/spoke-server -n spoke

# Option 2: Scale to zero and back (reset)
kubectl scale deployment spoke-server -n spoke --replicas=0
kubectl scale deployment spoke-server -n spoke --replicas=3

# Option 3: Delete and recreate pods
kubectl delete pods -n spoke -l app=spoke

# Option 4: Check and fix configuration
kubectl get configmap spoke-config -n spoke -o yaml
kubectl get secret spoke-secrets -n spoke -o yaml

# Emergency: Restore from backup if data corruption
./scripts/restore-postgres.sh s3://spoke-backups/spoke-backup-latest.sql.gz
```

### P2: High Error Rate

**Symptoms**: 5xx errors > 5%, elevated latency

**Investigation**:

```bash
# 1. Check error logs
kubectl logs -n spoke -l app=spoke --since=10m | grep '"level":"error"'

# 2. Check metrics
curl http://localhost:9090/metrics | grep spoke_http_requests_total

# 3. Check database
kubectl run -it --rm psql --image=postgres:16 -- psql "$SPOKE_POSTGRES_URL" -c "SELECT count(*) FROM modules;"

# 4. Check Redis
kubectl run -it --rm redis --image=redis:7 -- redis-cli -u "$SPOKE_REDIS_URL" ping
```

**Common Causes**:

1. **Database connection exhaustion**: Pool full
2. **Slow queries**: Database performance degradation
3. **Redis failure**: Cache layer unavailable
4. **Resource limits**: Pods being throttled or OOMKilled

**Mitigation**:

```bash
# If database connection exhaustion
kubectl edit configmap spoke-config -n spoke
# Increase SPOKE_POSTGRES_MAX_CONNS
kubectl rollout restart deployment/spoke-server -n spoke

# If resource constraints
kubectl edit deployment spoke-server -n spoke
# Increase resources.limits.cpu and resources.limits.memory
# Or scale horizontally
kubectl scale deployment spoke-server -n spoke --replicas=5

# If Redis issue (Spoke fails open, but will be slower)
# Fix Redis or disable caching temporarily
kubectl edit configmap spoke-config -n spoke
# Set SPOKE_CACHE_ENABLED: "false"
kubectl rollout restart deployment/spoke-server -n spoke
```

### P2: Database Primary Failure

**Symptoms**: Write operations failing, reads may work (if replicas exist)

**Investigation**:

```bash
# Check database connectivity
kubectl run -it --rm psql --image=postgres:16 -- psql "$SPOKE_POSTGRES_URL"

# Check database logs (managed service)
# AWS RDS: Check CloudWatch logs
# Google Cloud SQL: Check Cloud Logging
```

**Mitigation**:

**If using managed service with auto-failover**:
1. Wait for automatic failover (usually < 2 minutes)
2. Monitor connection recovery in Spoke logs
3. Verify new primary is healthy

**If using self-managed PostgreSQL with Patroni**:

```bash
# Check Patroni cluster status
patronictl -c /etc/patroni/patroni.yml list

# If automatic failover hasn't occurred, trigger manually
patronictl -c /etc/patroni/patroni.yml failover

# Update Spoke connection string if needed
kubectl edit secret spoke-secrets -n spoke
# Update SPOKE_POSTGRES_URL to new primary

kubectl rollout restart deployment/spoke-server -n spoke
```

**If no automatic failover**:

```bash
# Emergency: Promote replica to primary manually
# Connect to replica and run:
# SELECT pg_promote();

# Update Spoke configuration
kubectl edit secret spoke-secrets -n spoke
# Update SPOKE_POSTGRES_URL to new primary

kubectl rollout restart deployment/spoke-server -n spoke
```

### P2: Redis Master Failure

**Symptoms**: Cache not working, increased database load, slower responses

**Investigation**:

```bash
# Check Redis connectivity
kubectl run -it --rm redis --image=redis:7 -- redis-cli -u "$SPOKE_REDIS_URL" ping

# If using Sentinel, check failover status
kubectl exec -n redis redis-sentinel-1 -- redis-cli -p 26379 SENTINEL masters
```

**Mitigation**:

**If using Redis Sentinel**:
1. Sentinel automatically promotes replica (usually < 30 seconds)
2. Spoke reconnects automatically
3. Verify new master is healthy

```bash
# Check new master
kubectl exec -n redis redis-sentinel-1 -- redis-cli -p 26379 SENTINEL get-master-addr-by-name spoke-redis
```

**If Sentinel doesn't failover**:

```bash
# Manually trigger failover
kubectl exec -n redis redis-sentinel-1 -- redis-cli -p 26379 SENTINEL failover spoke-redis

# If Sentinel is completely broken, manually update Spoke config
kubectl edit configmap spoke-config -n spoke
# Update SPOKE_REDIS_URL to new master

kubectl rollout restart deployment/spoke-server -n spoke
```

**Workaround**: Spoke fails open - service continues without cache (slower but functional)

### P3: Single Pod Failure

**Symptoms**: 1 of 3 pods not running, requests still succeed

**Investigation**:

```bash
# Get pod status
kubectl get pods -n spoke

# Describe failed pod
kubectl describe pod -n spoke <pod-name>

# Check logs
kubectl logs -n spoke <pod-name> -p  # Previous container
```

**Common Causes**:

1. **OOMKilled**: Memory limit too low
2. **CrashLoopBackOff**: Application error
3. **Node eviction**: Node pressure

**Mitigation**:

Usually self-heals via Kubernetes restart policy. If persists:

```bash
# Delete pod (will be recreated)
kubectl delete pod -n spoke <pod-name>

# If OOMKilled, increase memory
kubectl edit deployment spoke-server -n spoke
# Increase resources.limits.memory

# If application error, check logs and rollback if needed
kubectl rollout undo deployment/spoke-server -n spoke
```

### P3: High Latency

**Symptoms**: p95 > 500ms, p99 > 1s, but no errors

**Investigation**:

```bash
# Check metrics
curl http://localhost:9090/metrics | grep http_request_duration

# Check resource usage
kubectl top pods -n spoke
kubectl top nodes

# Check database query performance
# Connect to PostgreSQL and check slow queries

# Check cache hit rate
curl http://localhost:9090/metrics | grep cache
```

**Common Causes**:

1. **CPU throttling**: Pods hitting CPU limits
2. **Database slow**: Slow queries or high load
3. **Cache misses**: Redis not working efficiently
4. **Network latency**: Slow connection to PostgreSQL/S3

**Mitigation**:

```bash
# Scale horizontally
kubectl scale deployment spoke-server -n spoke --replicas=5

# Or increase CPU limits
kubectl edit deployment spoke-server -n spoke
# Increase resources.limits.cpu

# If database slow, add read replicas
kubectl edit configmap spoke-config -n spoke
# Add SPOKE_POSTGRES_REPLICA_URLS

# If cache issues
kubectl logs -n redis <redis-pod> | tail -100
# Check Redis memory usage, may need to increase Redis resources
```

---

## Backup and Restore

### Daily Backup Verification

```bash
# Check latest backup
aws s3 ls s3://spoke-backups/ --recursive | sort | tail -1

# Verify backup size (should be > 0)
aws s3 ls s3://spoke-backups/spoke-backup-latest.sql.gz --summarize

# Test backup integrity (weekly)
./scripts/restore-postgres.sh --dry-run s3://spoke-backups/spoke-backup-latest.sql.gz
```

### Manual Backup

```bash
# Create immediate backup
export SPOKE_POSTGRES_URL="postgresql://..."
export SPOKE_S3_BUCKET="spoke-backups"
./scripts/backup-postgres.sh

# Verify
aws s3 ls s3://spoke-backups/ | tail -1
```

### Restore from Backup

**WARNING**: This is a destructive operation. Creates downtime.

**Procedure**:

```bash
# 1. Create maintenance window
kubectl scale deployment spoke-server -n spoke --replicas=0

# 2. Verify all pods stopped
kubectl get pods -n spoke

# 3. Download backup
aws s3 cp s3://spoke-backups/spoke-backup-YYYYMMDD-HHMMSS.sql.gz /tmp/

# 4. Restore database
./scripts/restore-postgres.sh /tmp/spoke-backup-YYYYMMDD-HHMMSS.sql.gz

# 5. Verify restoration
psql "$SPOKE_POSTGRES_URL" -c "SELECT COUNT(*) FROM modules;"

# 6. Restart Spoke
kubectl scale deployment spoke-server -n spoke --replicas=3

# 7. Verify functionality
kubectl wait --for=condition=ready pod -l app=spoke -n spoke --timeout=60s
curl https://spoke.example.com/modules
```

### Point-in-Time Recovery

If using PostgreSQL WAL archiving:

```bash
# Restore to specific timestamp
export RECOVERY_TARGET_TIME="2026-01-25 12:00:00"

# Connect to PostgreSQL and create recovery.conf
cat > recovery.conf <<EOF
restore_command = 'aws s3 cp s3://spoke-wal-archive/%f %p'
recovery_target_time = '$RECOVERY_TARGET_TIME'
recovery_target_action = 'promote'
EOF

# Restart PostgreSQL with recovery.conf
# PostgreSQL will replay WAL until target time
```

---

## Scaling Operations

### Scale Up (Horizontal)

**When to scale up**:
- CPU utilization > 70% for > 5 minutes
- Memory utilization > 80%
- Request latency p95 > 500ms
- Request queue depth increasing

**Procedure**:

```bash
# Manual scale
kubectl scale deployment spoke-server -n spoke --replicas=5

# Verify
kubectl get pods -n spoke --watch

# Or adjust HPA
kubectl edit hpa spoke-hpa -n spoke
# Increase minReplicas or decrease CPU target percentage
```

### Scale Down (Horizontal)

**When to scale down**:
- CPU utilization < 30% for > 30 minutes
- Low traffic period
- Cost optimization

**Procedure**:

```bash
# Never scale below 3 replicas (HA requirement)
kubectl scale deployment spoke-server -n spoke --replicas=3

# Or adjust HPA
kubectl edit hpa spoke-hpa -n spoke
# Decrease maxReplicas
```

### Scale Up (Vertical)

**When to scale up**:
- Pods frequently OOMKilled
- CPU throttling observed
- Single-request workloads (can't benefit from horizontal scaling)

**Procedure**:

```bash
# Edit deployment
kubectl edit deployment spoke-server -n spoke

# Update resources:
# resources:
#   requests:
#     memory: "1Gi"  # Increase
#     cpu: "1000m"   # Increase
#   limits:
#     memory: "4Gi"  # Increase
#     cpu: "4000m"   # Increase

# Apply (triggers rolling update)
kubectl rollout status deployment/spoke-server -n spoke
```

### Database Scaling

**Add Read Replica**:

```bash
# Create read replica (managed service)
# AWS RDS example:
aws rds create-db-instance-read-replica \
  --db-instance-identifier spoke-db-replica-2 \
  --source-db-instance-identifier spoke-db

# Wait for replica to be available
aws rds wait db-instance-available --db-instance-identifier spoke-db-replica-2

# Update Spoke configuration
kubectl edit configmap spoke-config -n spoke
# Add new replica to SPOKE_POSTGRES_REPLICA_URLS (comma-separated)

# Restart Spoke to pick up new replica
kubectl rollout restart deployment/spoke-server -n spoke
```

**Increase Database Resources**:

```bash
# AWS RDS example
aws rds modify-db-instance \
  --db-instance-identifier spoke-db \
  --db-instance-class db.t3.large \
  --apply-immediately

# Note: May cause brief downtime for single-AZ deployments
```

---

## Monitoring and Alerts

### Recommended Alerts

Configure these alerts in your monitoring system (Prometheus AlertManager, Datadog, etc.):

#### Critical Alerts (P1)

```yaml
# All pods down
- alert: SpokeServiceDown
  expr: sum(up{job="spoke"}) == 0
  for: 1m
  severity: critical
  annotations:
    summary: "Spoke service is completely down"

# High error rate
- alert: SpokeHighErrorRate
  expr: rate(spoke_http_requests_total{status=~"5.."}[5m]) / rate(spoke_http_requests_total[5m]) > 0.05
  for: 5m
  severity: critical
  annotations:
    summary: "Spoke error rate > 5%"

# Database unreachable
- alert: SpokeDatabaseDown
  expr: spoke_health_check{check="database"} == 0
  for: 2m
  severity: critical
  annotations:
    summary: "Spoke cannot connect to database"
```

#### Warning Alerts (P2)

```yaml
# High latency
- alert: SpokeHighLatency
  expr: histogram_quantile(0.95, rate(spoke_http_request_duration_seconds_bucket[5m])) > 0.5
  for: 10m
  severity: warning
  annotations:
    summary: "Spoke p95 latency > 500ms"

# High CPU usage
- alert: SpokeHighCPU
  expr: rate(container_cpu_usage_seconds_total{pod=~"spoke-.*"}[5m]) > 0.8
  for: 10m
  severity: warning
  annotations:
    summary: "Spoke pod CPU usage > 80%"

# Database connection pool near limit
- alert: SpokeDBPoolNearLimit
  expr: spoke_db_connections_active / spoke_db_connections_max > 0.8
  for: 5m
  severity: warning
  annotations:
    summary: "Database connection pool > 80% utilized"

# Cache unavailable
- alert: SpokeCacheDown
  expr: spoke_health_check{check="redis"} == 0
  for: 5m
  severity: warning
  annotations:
    summary: "Spoke cache (Redis) is unavailable"
```

#### Info Alerts (P3)

```yaml
# Pod restarted
- alert: SpokePodRestarted
  expr: increase(kube_pod_container_status_restarts_total{pod=~"spoke-.*"}[1h]) > 0
  for: 0m
  severity: info
  annotations:
    summary: "Spoke pod restarted"

# Low cache hit rate
- alert: SpokeLowCacheHitRate
  expr: rate(spoke_cache_hits_total[5m]) / (rate(spoke_cache_hits_total[5m]) + rate(spoke_cache_misses_total[5m])) < 0.7
  for: 30m
  severity: info
  annotations:
    summary: "Spoke cache hit rate < 70%"
```

### On-Call Playbook

**When you receive an alert**:

1. **Acknowledge** - Acknowledge in monitoring system
2. **Check dashboard** - View Grafana dashboard for context
3. **Check pods** - `kubectl get pods -n spoke`
4. **Check logs** - `kubectl logs -n spoke -l app=spoke --tail=100`
5. **Assess severity** - Use severity matrix above
6. **Follow runbook** - Use incident response procedures
7. **Escalate if needed** - Contact senior engineer for P1/P2

**On-call rotation best practices**:
- Keep laptop charged and available
- Have kubectl access configured
- Bookmark Grafana dashboard
- Test alert delivery monthly
- Document non-standard incidents

---

## Maintenance Windows

### Planned Maintenance

**Schedule**: Monthly, during low-traffic window (e.g., Sunday 2-4 AM UTC)

**Activities**:
- Database maintenance (VACUUM, REINDEX)
- Kubernetes node upgrades
- PostgreSQL minor version upgrades
- Security patches

**Procedure**:

```bash
# 1. Notify users (24-48 hours advance notice)
# Post maintenance window notification

# 2. Create backup before maintenance
./scripts/backup-postgres.sh

# 3. Perform maintenance
# Example: Kubernetes node upgrade
kubectl cordon <node-name>
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data
# Upgrade node OS/Kubernetes version
kubectl uncordon <node-name>

# 4. Verify service health
kubectl get pods -n spoke
curl https://spoke.example.com/health/ready

# 5. Monitor for 30 minutes post-maintenance
watch -n 10 'kubectl get pods -n spoke'
```

### Emergency Maintenance

**When**: Critical security patch, data corruption, etc.

**Procedure**:

```bash
# 1. Notify stakeholders immediately
# Send emergency maintenance notification

# 2. Create backup
./scripts/backup-postgres.sh

# 3. Perform maintenance quickly
# Apply fix

# 4. Verify and monitor closely
kubectl get pods -n spoke --watch
```

---

## Common Tasks

### Update Spoke to New Version

```bash
# 1. Review changelog
# Check release notes for breaking changes

# 2. Test in staging first
kubectl set image deployment/spoke-server spoke=spoke:v1.2.0 -n spoke-staging
kubectl rollout status deployment/spoke-server -n spoke-staging

# 3. Deploy to production
kubectl set image deployment/spoke-server spoke=spoke:v1.2.0 -n spoke
kubectl rollout status deployment/spoke-server -n spoke

# 4. Monitor
kubectl get pods -n spoke --watch
curl https://spoke.example.com/modules

# 5. Rollback if issues
kubectl rollout undo deployment/spoke-server -n spoke
```

### Update Configuration

```bash
# 1. Edit ConfigMap
kubectl edit configmap spoke-config -n spoke

# 2. Restart pods to pick up changes
kubectl rollout restart deployment/spoke-server -n spoke

# 3. Verify
kubectl get pods -n spoke
```

### Update Secrets

```bash
# 1. Update secret
kubectl create secret generic spoke-secrets -n spoke \
  --from-literal=SPOKE_POSTGRES_URL="..." \
  --from-literal=SPOKE_REDIS_URL="..." \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Restart pods
kubectl rollout restart deployment/spoke-server -n spoke

# 3. Verify connectivity
kubectl logs -n spoke -l app=spoke | grep "initialized"
```

### View Traces (with Jaeger)

```bash
# Port-forward to Jaeger
kubectl port-forward -n monitoring svc/jaeger-query 16686:16686

# Open browser
open http://localhost:16686

# Search for traces:
# - Service: spoke-registry
# - Operation: HTTP GET /modules
# - Lookback: 1h
```

### Query Metrics (with Prometheus)

```bash
# Port-forward to Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090

# Open browser
open http://localhost:9090

# Example queries:
# - spoke_http_requests_total
# - rate(spoke_http_requests_total[5m])
# - histogram_quantile(0.95, rate(spoke_http_request_duration_seconds_bucket[5m]))
```

### Database Query Performance

```bash
# Connect to database
kubectl run -it --rm psql --image=postgres:16 -- psql "$SPOKE_POSTGRES_URL"

# Check slow queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active' AND now() - pg_stat_activity.query_start > interval '5 seconds';

# Check connection count
SELECT count(*) FROM pg_stat_activity;

# Check database size
SELECT pg_size_pretty(pg_database_size('spoke'));

# Check table sizes
SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

### Redis Operations

```bash
# Connect to Redis
kubectl run -it --rm redis --image=redis:7 -- redis-cli -u "$SPOKE_REDIS_URL"

# Check memory usage
INFO memory

# Check hit rate
INFO stats

# Check connected clients
CLIENT LIST

# Flush cache (use with caution!)
FLUSHDB
```

---

## Disaster Recovery Scenarios

### Complete Data Center Failure

**Scenario**: Entire AWS region unavailable

**Recovery**:

1. **Activate DR region** (if multi-region deployment)
2. **Update DNS** to point to DR region
3. **Restore from latest backup** in DR region
4. **Verify data integrity**
5. **Resume operations** in DR region

**RTO**: 4 hours | **RPO**: 1 hour (based on backup frequency)

### Database Corruption

**Scenario**: Database corruption detected, service degraded

**Recovery**:

```bash
# 1. Stop all Spoke instances immediately
kubectl scale deployment spoke-server -n spoke --replicas=0

# 2. Assess corruption extent
psql "$SPOKE_POSTGRES_URL" -c "SELECT * FROM modules LIMIT 10;"

# 3. If corruption is recent, restore from latest backup
./scripts/restore-postgres.sh s3://spoke-backups/spoke-backup-latest.sql.gz

# 4. If corruption is in backup, restore from earlier backup
./scripts/restore-postgres.sh s3://spoke-backups/spoke-backup-20260124-020000.sql.gz

# 5. Verify data integrity
psql "$SPOKE_POSTGRES_URL" -c "SELECT COUNT(*) FROM modules;"

# 6. Restart Spoke
kubectl scale deployment spoke-server -n spoke --replicas=3
```

### S3 Bucket Deletion

**Scenario**: S3 bucket accidentally deleted

**Recovery**:

If versioning enabled:
```bash
# Restore all objects from deleted markers
aws s3api list-object-versions --bucket spoke-schemas \
  --query 'DeleteMarkers[].{Key:Key,VersionId:VersionId}' \
  | jq -r '.[] | "\(.Key) \(.VersionId)"' \
  | while read key versionId; do
      aws s3api delete-object --bucket spoke-schemas --key "$key" --version-id "$versionId"
    done
```

If versioning not enabled:
1. Restore from most recent database backup
2. Re-upload all proto files from database to S3
3. Verify file integrity

---

## Support Contacts

- **On-Call Engineer**: [PagerDuty/Opsgenie rotation]
- **Platform Team**: platform@example.com
- **Database Team**: dba@example.com
- **Security Team**: security@example.com

## Related Documentation

- [Deployment Guide](../deployment/DEPLOYMENT_GUIDE.md)
- [PostgreSQL HA Guide](../ha/postgresql-replication.md)
- [Redis Sentinel Guide](../ha/redis-sentinel.md)
- [OpenTelemetry Setup](../observability/otel-setup.md)
- [Verification Checklist](../verification/VERIFICATION_CHECKLIST.md)
