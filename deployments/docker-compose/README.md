# Docker Compose HA Stack for Spoke

This Docker Compose stack provides a complete high-availability testing environment for Spoke with:

- **PostgreSQL**: Primary + replica with streaming replication
- **Redis**: Master + replica with 3 Sentinel instances for automatic failover
- **MinIO**: S3-compatible object storage
- **Spoke**: 3 instances behind NGINX load balancer
- **OpenTelemetry Collector**: Observability with Prometheus metrics

## Quick Start

### 1. Build and Start the Stack

```bash
cd deployments/docker-compose
docker-compose -f ha-stack.yml up --build
```

This will:
- Build the Spoke image
- Start all services with health checks
- Configure replication and failover
- Initialize the database

**First startup takes 2-3 minutes** as services wait for dependencies.

### 2. Verify the Stack

```bash
# Check all services are healthy
docker-compose -f ha-stack.yml ps

# Expected: All services showing "healthy"
```

**Access Points:**
- **Load Balanced API**: http://localhost:8080
- **Direct Spoke Instances**:
  - Spoke 1: http://localhost:8081
  - Spoke 2: http://localhost:8082
  - Spoke 3: http://localhost:8083
- **Health Endpoints**:
  - Spoke 1: http://localhost:9091/health/ready
  - Spoke 2: http://localhost:9092/health/ready
  - Spoke 3: http://localhost:9093/health/ready
- **PostgreSQL Primary**: localhost:5432
- **PostgreSQL Replica**: localhost:5433
- **Redis Master**: localhost:6379
- **Redis Sentinels**: localhost:26379, 26380, 26381
- **MinIO Console**: http://localhost:9001 (spoke/spoke-password)
- **OpenTelemetry Metrics**: http://localhost:8889/metrics

### 3. Test the API

```bash
# Test load balancer
curl http://localhost:8080/modules

# Check health
curl http://localhost:8080/health/ready | jq .

# Create a test module
curl -X POST http://localhost:8080/modules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-module",
    "description": "Test module for HA stack"
  }'
```

## Testing High Availability

### Test 1: Load Balancing

```bash
# Make multiple requests and observe which instance handles them
for i in {1..10}; do
  curl -s http://localhost:8080/modules | jq -r '.[] | .name' | head -1
  echo "Request $i completed"
done
```

Check NGINX logs to see distribution:
```bash
docker logs spoke-nginx
```

### Test 2: Spoke Instance Failure

**Simulate failure:**
```bash
# Stop one Spoke instance
docker stop spoke-1

# Verify others still serve traffic
curl http://localhost:8080/health/ready

# Check NGINX marks it as down
docker exec spoke-nginx cat /var/log/nginx/error.log | grep spoke-1
```

**Restart:**
```bash
docker start spoke-1

# Verify it rejoins the cluster
curl http://localhost:9091/health/ready
```

### Test 3: Redis Master Failover

**Simulate Redis master failure:**
```bash
# Stop Redis master
docker stop spoke-redis-master

# Wait for Sentinel to detect failure (5-10 seconds)
sleep 10

# Check Sentinel promoted replica
docker exec spoke-redis-sentinel-1 \
  redis-cli -p 26379 SENTINEL get-master-addr-by-name spoke-redis

# Spoke instances should automatically reconnect to new master
curl http://localhost:8080/health/ready
```

**View Sentinel logs:**
```bash
docker logs spoke-redis-sentinel-1 | grep -E "failover|switch-master"
```

**Restart old master (becomes replica):**
```bash
docker start spoke-redis-master

# It will rejoin as replica
docker exec spoke-redis-master redis-cli -a spoke-redis-password INFO replication
```

### Test 4: PostgreSQL Failover

**Check replication status:**
```bash
# On primary
docker exec spoke-postgres-primary psql -U spoke -d spoke -c \
  "SELECT * FROM pg_stat_replication;"

# On replica
docker exec spoke-postgres-replica psql -U spoke -d spoke -c \
  "SELECT pg_is_in_recovery();"
```

**Manual failover (promote replica):**
```bash
# Promote replica to primary
docker exec spoke-postgres-replica pg_ctl promote -D /var/lib/postgresql/data

# Update Spoke instances to use new primary
# (In production, this would be automated or use connection pooler)
```

### Test 5: Cache Performance

```bash
# First request (cache miss)
time curl -s http://localhost:8080/modules/test-module/versions/1.0.0

# Second request (cache hit - should be faster)
time curl -s http://localhost:8080/modules/test-module/versions/1.0.0

# Check Redis for cached data
docker exec spoke-redis-master redis-cli -a spoke-redis-password \
  KEYS "version:*"
```

### Test 6: OpenTelemetry Metrics

```bash
# View Prometheus metrics
curl http://localhost:8889/metrics | grep spoke

# Key metrics to check:
# - http_server_requests
# - http_server_duration
# - db_connections_active
# - cache_hits_total
# - cache_misses_total
```

## Monitoring

### View Logs

```bash
# All services
docker-compose -f ha-stack.yml logs -f

# Specific service
docker-compose -f ha-stack.yml logs -f spoke-1

# Nginx access log
docker exec spoke-nginx tail -f /var/log/nginx/access.log
```

### Check Service Health

```bash
# All services
docker-compose -f ha-stack.yml ps

# Specific service
docker inspect spoke-1 --format='{{.State.Health.Status}}'
```

### Database Monitoring

```bash
# Replication lag
docker exec spoke-postgres-primary psql -U spoke -d spoke -c \
  "SELECT client_addr, state, replay_lag FROM pg_stat_replication;"

# Connection count
docker exec spoke-postgres-primary psql -U spoke -d spoke -c \
  "SELECT count(*) FROM pg_stat_activity;"
```

### Redis Monitoring

```bash
# Master info
docker exec spoke-redis-master redis-cli -a spoke-redis-password INFO replication

# Sentinel status
docker exec spoke-redis-sentinel-1 redis-cli -p 26379 SENTINEL masters

# Memory usage
docker exec spoke-redis-master redis-cli -a spoke-redis-password INFO memory
```

## Cleanup

```bash
# Stop all services
docker-compose -f ha-stack.yml down

# Remove volumes (WARNING: deletes all data)
docker-compose -f ha-stack.yml down -v

# Remove images
docker-compose -f ha-stack.yml down --rmi all -v
```

## Troubleshooting

### Services Not Starting

**Check logs:**
```bash
docker-compose -f ha-stack.yml logs <service-name>
```

**Common issues:**
1. **Port already in use**: Stop conflicting services
2. **Build failures**: Run `docker-compose build --no-cache`
3. **Health check failures**: Increase `start_period` in compose file

### PostgreSQL Replica Not Syncing

**Check replica logs:**
```bash
docker logs spoke-postgres-replica
```

**Verify replication slot:**
```bash
docker exec spoke-postgres-primary psql -U spoke -d spoke -c \
  "SELECT * FROM pg_replication_slots;"
```

**Recreate replica:**
```bash
docker-compose -f ha-stack.yml stop postgres-replica
docker-compose -f ha-stack.yml rm -f postgres-replica
docker volume rm docker-compose_postgres-replica-data
docker-compose -f ha-stack.yml up -d postgres-replica
```

### Redis Sentinel Not Detecting Master

**Check Sentinel config:**
```bash
docker exec spoke-redis-sentinel-1 redis-cli -p 26379 SENTINEL masters
```

**Reset Sentinel:**
```bash
docker-compose -f ha-stack.yml restart redis-sentinel-1 redis-sentinel-2 redis-sentinel-3
```

### Spoke Instance Crashes

**Check logs:**
```bash
docker logs spoke-1
```

**Common causes:**
- Database connection failure
- Redis connection failure
- Missing environment variables
- S3 bucket not created

**Verify dependencies:**
```bash
# Database
docker exec spoke-1 wget -O- http://postgres-primary:5432 || echo "DB unreachable"

# Redis
docker exec spoke-1 wget -O- http://redis-master:6379 || echo "Redis unreachable"

# MinIO
docker exec spoke-1 wget -O- http://minio:9000 || echo "S3 unreachable"
```

### Load Balancer Not Distributing

**Check NGINX config:**
```bash
docker exec spoke-nginx nginx -t
```

**View upstream status:**
```bash
curl http://localhost:8080/nginx_status
```

**Check backend health:**
```bash
for port in 9091 9092 9093; do
  curl -s http://localhost:$port/health/ready | jq .
done
```

## Performance Tuning

### Increase Replica Count

Edit `ha-stack.yml`:
```yaml
# Add more Spoke instances
spoke-4:
  # ... same config as spoke-1 with different ports
```

Update `nginx.conf`:
```nginx
upstream spoke_backend {
    server spoke-1:8080;
    server spoke-2:8080;
    server spoke-3:8080;
    server spoke-4:8080;  # Add new instance
}
```

### Adjust Resource Limits

Add resource limits to services in `ha-stack.yml`:
```yaml
services:
  spoke-1:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### Database Connection Pooling

Increase connections in environment:
```yaml
SPOKE_POSTGRES_MAX_CONNS: "50"
```

### Redis Memory Limit

Add to Redis command:
```yaml
command: >
  redis-server
  --maxmemory 256mb
  --maxmemory-policy allkeys-lru
```

## Production Differences

This stack is for **testing only**. For production:

1. **Security**:
   - Use strong passwords (not hardcoded)
   - Enable TLS for all services
   - Use secrets management (Vault, AWS Secrets Manager)

2. **Persistence**:
   - Use named volumes or host mounts
   - Configure backup schedules
   - Enable WAL archiving for PostgreSQL

3. **Scaling**:
   - Use Kubernetes instead of Docker Compose
   - Implement proper service discovery
   - Use managed services (RDS, ElastiCache, S3)

4. **Monitoring**:
   - Add Prometheus for metrics
   - Add Grafana for dashboards
   - Add Jaeger for distributed tracing
   - Configure alerting

See [Kubernetes deployment](../kubernetes/README.md) for production setup.

## Further Reading

- [PostgreSQL Replication Guide](../../docs/ha/postgresql-replication.md)
- [Redis Sentinel Guide](../../docs/ha/redis-sentinel.md)
- [OpenTelemetry Setup](../../docs/observability/otel-setup.md)
- [Kubernetes Deployment](../kubernetes/README.md)
