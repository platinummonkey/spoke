# Redis High Availability with Sentinel

This guide covers setting up Redis Sentinel for automatic failover and high availability caching in Spoke.

## Architecture

```
┌───────────────┐
│ Spoke Instance│ ───┐
└───────────────┘    │
                     ├──► Sentinel 1 ─┐
┌───────────────┐    │   Sentinel 2  ├──► Redis Master ⟷ Redis Replica 1
│ Spoke Instance│ ───┤   Sentinel 3 ─┘                  ⟷ Redis Replica 2
└───────────────┘    │
                     │
┌───────────────┐    │
│ Spoke Instance│ ───┘
└───────────────┘
```

**Benefits**:
- **Automatic Failover**: No manual intervention on master failure
- **Service Discovery**: Clients auto-discover current master
- **Monitoring**: Health checks and notifications
- **High Availability**: 99.9%+ uptime

## Prerequisites

- Redis 7.0+ recommended
- Minimum 3 Sentinel instances (odd number)
- Network connectivity between all nodes
- Persistent storage for Redis data

## Part 1: Redis Master Setup

### 1.1 Install Redis

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install redis-server

# Or use official Docker image
docker pull redis:7
```

### 1.2 Configure Master

Edit `/etc/redis/redis.conf`:

```conf
# Network
bind 0.0.0.0
protected-mode yes
port 6379
requirepass "strong-redis-password"

# Persistence
save 900 1
save 300 10
save 60 10000
stop-writes-on-bgsave-error yes
rdbcompression yes
dbfilename dump.rdb
dir /var/lib/redis

# Replication
masterauth "strong-redis-password"
repl-diskless-sync no
repl-diskless-sync-delay 5
repl-ping-replica-period 10
repl-timeout 60

# Performance
maxmemory 2gb
maxmemory-policy allkeys-lru
maxclients 10000
timeout 300
tcp-keepalive 300

# Logging
loglevel notice
logfile /var/log/redis/redis-server.log
```

### 1.3 Start Redis Master

```bash
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

Verify:

```bash
redis-cli -a strong-redis-password ping
# Expected: PONG
```

## Part 2: Redis Replica Setup

### 2.1 Configure Replica

Edit `/etc/redis/redis.conf` on replica servers:

```conf
# Same as master, plus:
replicaof 10.0.1.10 6379  # Master IP and port
masterauth "strong-redis-password"
replica-read-only yes
```

### 2.2 Start Replicas

```bash
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

### 2.3 Verify Replication

On **master**:

```bash
redis-cli -a strong-redis-password INFO replication
```

Expected output:

```
# Replication
role:master
connected_slaves:2
slave0:ip=10.0.2.10,port=6379,state=online,offset=12345,lag=0
slave1:ip=10.0.2.11,port=6379,state=online,offset=12345,lag=0
```

On **replica**:

```bash
redis-cli -a strong-redis-password INFO replication
```

Expected output:

```
# Replication
role:slave
master_host:10.0.1.10
master_port:6379
master_link_status:up
```

## Part 3: Redis Sentinel Setup

### 3.1 Install Sentinel

Sentinel is included with Redis:

```bash
which redis-sentinel
# /usr/bin/redis-sentinel
```

### 3.2 Configure Sentinel

Create `/etc/redis/sentinel.conf` on all 3 Sentinel nodes:

```conf
# Network
bind 0.0.0.0
port 26379
protected-mode yes
sentinel auth-pass spoke-redis strong-redis-password

# Monitoring
sentinel monitor spoke-redis 10.0.1.10 6379 2
sentinel down-after-milliseconds spoke-redis 5000
sentinel parallel-syncs spoke-redis 1
sentinel failover-timeout spoke-redis 10000

# Notifications (optional)
sentinel notification-script spoke-redis /etc/redis/notify.sh
sentinel client-reconfig-script spoke-redis /etc/redis/reconfig.sh

# Logging
logfile /var/log/redis/sentinel.log
loglevel notice
```

**Key parameters**:
- `sentinel monitor <master-name> <ip> <port> <quorum>`
  - `quorum`: Minimum Sentinels that must agree for failover (2 out of 3)
- `down-after-milliseconds`: Time before marking master as down (5s)
- `failover-timeout`: Max time for failover process (10s)
- `parallel-syncs`: Max replicas to sync simultaneously (1 = safer)

### 3.3 Create Notification Script (Optional)

Create `/etc/redis/notify.sh`:

```bash
#!/bin/bash
EVENT_TYPE=$1
EVENT_NAME=$2
shift 2
ARGS="$@"

# Log to syslog
logger -t redis-sentinel "Event: $EVENT_TYPE $EVENT_NAME $ARGS"

# Send to monitoring system (example: webhook)
curl -X POST https://monitoring.example.com/alerts \
  -H "Content-Type: application/json" \
  -d "{\"event\":\"$EVENT_TYPE\",\"name\":\"$EVENT_NAME\",\"args\":\"$ARGS\"}"
```

Make executable:

```bash
chmod +x /etc/redis/notify.sh
```

### 3.4 Start Sentinels

On all 3 Sentinel nodes:

```bash
sudo systemctl start redis-sentinel
sudo systemctl enable redis-sentinel
```

Verify:

```bash
redis-cli -p 26379 SENTINEL masters
redis-cli -p 26379 SENTINEL slaves spoke-redis
redis-cli -p 26379 SENTINEL sentinels spoke-redis
```

## Part 4: Configure Spoke for Sentinel

### 4.1 Update Configuration

Set Sentinel URL format:

```bash
export SPOKE_REDIS_URL="redis://spoke-redis@sentinel1:26379,sentinel2:26379,sentinel3:26379/master/spoke-redis"
export SPOKE_REDIS_PASSWORD="strong-redis-password"
```

**URL format**:
```
redis://<master-name>@<sentinel1>:<port>,<sentinel2>:<port>,.../ master/<master-name>
```

**Note**: Sentinel URL support will be added in a future update. For now, use direct Redis URLs and manually update on failover.

### 4.2 Alternative: Direct Connection with Failover

For now, use direct Redis connection:

```bash
export SPOKE_REDIS_URL="redis://10.0.1.10:6379/0"
export SPOKE_REDIS_PASSWORD="strong-redis-password"
```

On failover, update the URL to point to new master and restart Spoke instances.

## Part 5: Testing Failover

### 5.1 Simulate Master Failure

On current **master**:

```bash
redis-cli -a strong-redis-password DEBUG sleep 30
```

Or stop Redis:

```bash
sudo systemctl stop redis-server
```

### 5.2 Monitor Sentinel Logs

```bash
tail -f /var/log/redis/sentinel.log
```

Expected sequence:

```
+sdown master spoke-redis 10.0.1.10 6379    # Subjectively down
+odown master spoke-redis 10.0.1.10 6379    # Objectively down (quorum reached)
+vote-for-leader <sentinel-id>               # Sentinels vote
+elected-leader <sentinel-id>                # Leader elected
+failover-state-select-slave spoke-redis     # Select new master
+selected-slave slave 10.0.2.10:6379         # Replica selected
+failover-state-send-slaveof-noone           # Promote replica
+failover-end master spoke-redis 10.0.2.10   # Failover complete
+switch-master spoke-redis 10.0.1.10 6379 10.0.2.10 6379  # Master switched
```

### 5.3 Verify New Master

```bash
redis-cli -h 10.0.2.10 -a strong-redis-password INFO replication
```

Should show:

```
role:master
```

Old master (when recovered) should show:

```
role:slave
master_host:10.0.2.10
```

### 5.4 Failover Timing

Expected timing:
- **Detection**: 5 seconds (down-after-milliseconds)
- **Quorum**: < 1 second (Sentinel communication)
- **Promotion**: 1-2 seconds
- **Total**: ~6-8 seconds

## Part 6: Monitoring

### Key Metrics

**Master status**:

```bash
redis-cli -p 26379 SENTINEL masters
```

**Replica lag**:

```bash
redis-cli -a strong-redis-password INFO replication | grep lag
```

**Sentinel health**:

```bash
redis-cli -p 26379 PING
redis-cli -p 26379 SENTINEL ckquorum spoke-redis
```

### Prometheus Exporter

Use [redis_exporter](https://github.com/oliver006/redis_exporter):

```bash
docker run -d \
  -e REDIS_ADDR=redis://10.0.1.10:6379 \
  -e REDIS_PASSWORD=strong-redis-password \
  -p 9121:9121 \
  oliver006/redis_exporter:latest
```

**Key Metrics**:
- `redis_up` - Redis instance availability
- `redis_connected_slaves` - Number of connected replicas
- `redis_master_repl_offset` - Replication offset
- `redis_commands_processed_total` - Commands per second
- `redis_memory_used_bytes` - Memory usage

### Alert Rules

```yaml
- alert: RedisMasterDown
  expr: redis_up{role="master"} == 0
  for: 30s
  labels:
    severity: critical
  annotations:
    summary: "Redis master is down"

- alert: RedisReplicationBroken
  expr: redis_connected_slaves < 1
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "Redis has no connected replicas"

- alert: RedisHighMemoryUsage
  expr: redis_memory_used_bytes / redis_memory_max_bytes > 0.9
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Redis memory usage > 90%"
```

## Part 7: Troubleshooting

### Sentinel Not Detecting Master

**Check network connectivity**:

```bash
telnet 10.0.1.10 6379
```

**Verify Sentinel config**:

```bash
redis-cli -p 26379 SENTINEL get-master-addr-by-name spoke-redis
```

**Check authentication**:

```bash
redis-cli -p 26379 SENTINEL masters
```

### Failover Not Happening

**Check quorum**:

```bash
redis-cli -p 26379 SENTINEL ckquorum spoke-redis
```

Must return OK. If not, check:
- At least 3 Sentinels running
- `quorum` parameter ≤ number of Sentinels / 2

**Check Sentinel logs**:

```bash
grep "failover" /var/log/redis/sentinel.log
```

### Split-Brain Scenario

If network partition causes multiple masters:

1. **Stop old master**:
   ```bash
   sudo systemctl stop redis-server
   ```

2. **Reconfigure as replica**:
   ```bash
   redis-cli -a strong-redis-password REPLICAOF 10.0.2.10 6379
   ```

3. **Restart**:
   ```bash
   sudo systemctl start redis-server
   ```

### Replica Lag

**Check replication status**:

```bash
redis-cli -a strong-redis-password INFO replication | grep -E "role|lag|offset"
```

**Causes**:
- Network bandwidth issues
- High write load on master
- Slow disk I/O on replica

**Solutions**:
- Increase network bandwidth
- Enable `repl-diskless-sync` for slow disks
- Reduce `repl-ping-replica-period`

## Part 8: Security

### TLS/SSL Encryption

Enable TLS in `redis.conf`:

```conf
tls-port 6379
tls-cert-file /etc/redis/certs/redis.crt
tls-key-file /etc/redis/certs/redis.key
tls-ca-cert-file /etc/redis/certs/ca.crt
tls-auth-clients yes
```

Update Spoke config:

```bash
export SPOKE_REDIS_URL="rediss://10.0.1.10:6379/0"  # Note: rediss://
```

### Network Isolation

- Run Redis in private VPC/network
- Use security groups/firewall rules
- Allow only Spoke instances and Sentinel nodes

### Access Control Lists (ACL)

Redis 6+ supports ACLs:

```bash
ACL SETUSER spoke on >password ~* &* +@all
ACL SETUSER replicator on >repl-password +psync +replconf +ping
```

## Part 9: Backup Strategy

### RDB Snapshots

Configure automatic snapshots:

```conf
save 900 1
save 300 10
save 60 10000
```

### Manual Backup

```bash
redis-cli -a strong-redis-password BGSAVE
```

Copy snapshot:

```bash
cp /var/lib/redis/dump.rdb /backup/redis-$(date +%Y%m%d-%H%M%S).rdb
```

### Restore from Backup

```bash
sudo systemctl stop redis-server
cp /backup/redis-20260125.rdb /var/lib/redis/dump.rdb
sudo chown redis:redis /var/lib/redis/dump.rdb
sudo systemctl start redis-server
```

## Further Reading

- [Redis Sentinel Documentation](https://redis.io/docs/management/sentinel/)
- [Redis Replication](https://redis.io/docs/management/replication/)
- [Redis Security](https://redis.io/docs/management/security/)
- [Spoke Cache Configuration](../configuration/cache.md)
