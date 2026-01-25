# PostgreSQL High Availability with Streaming Replication

This guide covers setting up PostgreSQL streaming replication for Spoke to achieve high availability and read scalability.

## Architecture

```
┌──────────────┐
│ Spoke (writes)│ ───────► PostgreSQL Primary
└──────────────┘               │
                               │ WAL streaming
┌──────────────┐               ▼
│ Spoke (reads) │ ◄──────── PostgreSQL Replica 1
└──────────────┘

┌──────────────┐          PostgreSQL Replica 2
│ Spoke (reads) │ ◄────────
└──────────────┘
```

**Benefits**:
- **High Availability**: Automatic failover with minimal downtime
- **Read Scalability**: Distribute read queries across replicas
- **Data Redundancy**: Multiple copies of data
- **Zero Data Loss**: Synchronous replication option

## Prerequisites

- PostgreSQL 14+ (16+ recommended)
- Root or sudo access on all servers
- Network connectivity between primary and replicas
- Sufficient disk space for WAL archiving

## Part 1: Configure Primary Server

### 1.1 Edit postgresql.conf

```bash
sudo vi /etc/postgresql/16/main/postgresql.conf
```

Add/modify:

```ini
# Replication settings
wal_level = replica
max_wal_senders = 10
max_replication_slots = 10
wal_keep_size = 1GB
hot_standby = on
hot_standby_feedback = on

# Performance tuning
shared_buffers = 4GB
effective_cache_size = 12GB
maintenance_work_mem = 1GB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 10MB
min_wal_size = 1GB
max_wal_size = 4GB

# Connection settings
listen_addresses = '*'
max_connections = 200

# Logging
logging_collector = on
log_directory = 'log'
log_filename = 'postgresql-%Y-%m-%d_%H%M%S.log'
log_rotation_age = 1d
log_rotation_size = 100MB
log_line_prefix = '%m [%p] %u@%d '
log_checkpoints = on
log_connections = on
log_disconnections = on
log_duration = off
log_lock_waits = on
```

### 1.2 Configure Authentication

Edit `pg_hba.conf`:

```bash
sudo vi /etc/postgresql/16/main/pg_hba.conf
```

Add replication access:

```
# Replication connections
host    replication     replicator      10.0.1.0/24             scram-sha-256
host    replication     replicator      10.0.2.0/24             scram-sha-256
```

Replace `10.0.1.0/24` with your replica network.

### 1.3 Create Replication User

```sql
CREATE ROLE replicator WITH REPLICATION PASSWORD 'strong-password-here' LOGIN;
```

Store password securely (e.g., in Vault, AWS Secrets Manager).

### 1.4 Create Replication Slot

```sql
SELECT pg_create_physical_replication_slot('replica_1_slot');
SELECT pg_create_physical_replication_slot('replica_2_slot');
```

### 1.5 Restart Primary

```bash
sudo systemctl restart postgresql
```

Verify:

```bash
sudo -u postgres psql -c "SELECT * FROM pg_stat_replication;"
```

(Should be empty until replicas connect)

## Part 2: Set Up Replica Servers

### 2.1 Stop PostgreSQL on Replica

```bash
sudo systemctl stop postgresql
```

### 2.2 Remove Existing Data Directory

```bash
sudo rm -rf /var/lib/postgresql/16/main/*
```

### 2.3 Create Base Backup from Primary

```bash
sudo -u postgres pg_basebackup \
  -h 10.0.1.10 \
  -D /var/lib/postgresql/16/main \
  -U replicator \
  -P \
  -v \
  -R \
  -X stream \
  -C \
  -S replica_1_slot
```

Options:
- `-h`: Primary server IP
- `-D`: Data directory
- `-U`: Replication user
- `-P`: Show progress
- `-v`: Verbose
- `-R`: Create standby.signal and replication config
- `-X stream`: Stream WAL during backup
- `-C`: Create replication slot
- `-S`: Replication slot name

Enter replication password when prompted.

### 2.4 Configure Replica

PostgreSQL will auto-create `standby.signal` and update `postgresql.auto.conf` with:

```ini
primary_conninfo = 'host=10.0.1.10 port=5432 user=replicator password=XXX'
primary_slot_name = 'replica_1_slot'
```

Optionally edit `postgresql.conf`:

```ini
hot_standby = on
hot_standby_feedback = on
```

### 2.5 Start Replica

```bash
sudo systemctl start postgresql
```

### 2.6 Verify Replication

On **replica**:

```sql
SELECT pg_is_in_recovery();  -- Should return: t (true)
```

On **primary**:

```sql
SELECT * FROM pg_stat_replication;
```

Expected output:

```
 pid  | application_name | client_addr |  state    | sync_state |
------|------------------|-------------|-----------|------------|
 1234 | replica_1_slot   | 10.0.2.10   | streaming | async      |
```

**Replication lag**:

```sql
SELECT
    client_addr,
    state,
    pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) AS lag_bytes,
    pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)) AS lag
FROM pg_stat_replication;
```

## Part 3: Configure Spoke for Read Replicas

### 3.1 Update Configuration

Set environment variables:

```bash
# Primary database (for writes)
export SPOKE_POSTGRES_URL="postgresql://spoke:password@10.0.1.10:5432/spoke"

# Read replicas (comma-separated, for reads)
export SPOKE_POSTGRES_REPLICA_URLS="postgresql://spoke:password@10.0.2.10:5432/spoke,postgresql://spoke:password@10.0.2.11:5432/spoke"
```

**Note**: Read replica support is implemented in Phase 2 (Task 6). For now, Spoke will use primary for all queries.

### 3.2 Connection Pooling

For production, use PgBouncer for connection pooling (see below).

## Part 4: PgBouncer Setup

PgBouncer reduces connection overhead and provides connection pooling.

### 4.1 Install PgBouncer

```bash
sudo apt-get install pgbouncer
```

### 4.2 Configure PgBouncer

Edit `/etc/pgbouncer/pgbouncer.ini`:

```ini
[databases]
spoke = host=10.0.1.10 port=5432 dbname=spoke

[pgbouncer]
listen_addr = *
listen_port = 6432
auth_type = scram-sha-256
auth_file = /etc/pgbouncer/userlist.txt
admin_users = postgres
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
min_pool_size = 5
reserve_pool_size = 5
reserve_pool_timeout = 3
max_db_connections = 50
max_user_connections = 50
server_lifetime = 3600
server_idle_timeout = 600
server_connect_timeout = 15
server_login_retry = 15
log_connections = 1
log_disconnections = 1
log_pooler_errors = 1
```

### 4.3 Create User List

Edit `/etc/pgbouncer/userlist.txt`:

```
"spoke" "SCRAM-SHA-256$..."
```

Generate hash:

```bash
echo -n "your-password" | openssl dgst -sha256 -binary | openssl enc -base64
```

Or use PostgreSQL to extract:

```sql
SELECT rolname, rolpassword FROM pg_authid WHERE rolname = 'spoke';
```

### 4.4 Start PgBouncer

```bash
sudo systemctl start pgbouncer
sudo systemctl enable pgbouncer
```

### 4.5 Update Spoke Configuration

```bash
export SPOKE_POSTGRES_URL="postgresql://spoke:password@localhost:6432/spoke"
```

## Part 5: Failover Procedures

### Manual Failover

**Scenario**: Primary fails, promote replica to new primary.

#### Step 1: Promote Replica

On the **replica** to promote:

```bash
sudo -u postgres pg_ctl promote -D /var/lib/postgresql/16/main
```

Or:

```sql
SELECT pg_promote();
```

#### Step 2: Verify Promotion

```sql
SELECT pg_is_in_recovery();  -- Should return: f (false)
```

#### Step 3: Update Spoke Configuration

Point writes to the new primary:

```bash
export SPOKE_POSTGRES_URL="postgresql://spoke:password@10.0.2.10:5432/spoke"
```

Restart Spoke instances or use dynamic config reload.

#### Step 4: Reconfigure Old Primary (if recovered)

When the old primary comes back online, make it a replica:

1. Stop PostgreSQL
2. Remove data directory
3. Run `pg_basebackup` from new primary
4. Start as replica

### Automatic Failover with Patroni

For automatic failover, use [Patroni](https://patroni.readthedocs.io/):

- Monitors PostgreSQL health
- Automatic failover on primary failure
- Maintains cluster state in etcd/Consul
- REST API for health checks

See [Patroni documentation](https://patroni.readthedocs.io/) for setup.

## Part 6: Monitoring Replication

### Key Metrics

**Replication Lag** (seconds):

```sql
SELECT
    EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp()))::int AS lag_seconds
FROM pg_stat_replication;
```

**WAL Lag** (bytes):

```sql
SELECT
    pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) AS lag_bytes
FROM pg_stat_replication;
```

**Replication Slots**:

```sql
SELECT * FROM pg_replication_slots;
```

### Prometheus Exporter

Use [postgres_exporter](https://github.com/prometheus-community/postgres_exporter):

```bash
docker run -d \
  -e DATA_SOURCE_NAME="postgresql://spoke:password@localhost:5432/spoke?sslmode=disable" \
  -p 9187:9187 \
  prometheuscommunity/postgres-exporter
```

**Key Metrics**:
- `pg_replication_lag` - Replication lag in seconds
- `pg_stat_replication_pg_wal_lsn_diff` - WAL lag in bytes
- `pg_stat_database_tup_inserted` - Inserts per second

### Alert Rules

```yaml
- alert: PostgreSQLReplicationLag
  expr: pg_replication_lag_seconds > 60
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Replication lag > 60 seconds"

- alert: PostgreSQLReplicationDown
  expr: pg_stat_replication_pg_wal_lsn_diff == 0
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "Replication is not working"
```

## Part 7: Backup Strategy

### Continuous Archiving

Configure WAL archiving for point-in-time recovery:

```ini
# postgresql.conf
archive_mode = on
archive_command = 'test ! -f /mnt/wal_archive/%f && cp %p /mnt/wal_archive/%f'
```

Or use pgBackRest:

```bash
pgbackrest --stanza=spoke --type=full backup
```

See [automated backup scripts](../../scripts/backup-postgres.sh) for automation.

### Backup Schedule

- **Full backup**: Daily at 2 AM
- **Incremental backup**: Every 6 hours
- **WAL archiving**: Continuous
- **Retention**: 7 days local, 30 days S3

## Troubleshooting

### Replica Not Connecting

**Check primary logs**:

```bash
sudo tail -f /var/log/postgresql/postgresql-16-main.log
```

**Check network connectivity**:

```bash
telnet 10.0.1.10 5432
```

**Verify pg_hba.conf**:

```sql
SELECT * FROM pg_hba_file_rules WHERE database = '{replication}';
```

### High Replication Lag

**Causes**:
1. Network bandwidth issues
2. High write load on primary
3. Slow disk I/O on replica
4. Long-running queries on replica

**Solutions**:
- Increase `wal_sender_timeout`
- Use synchronous replication for critical data
- Tune `max_wal_senders` and `wal_keep_size`
- Upgrade replica hardware

### Replication Slot Full

```
ERROR:  replication slot "replica_1_slot" is active but not consuming WAL
```

**Solution**:

```sql
-- Check slot status
SELECT * FROM pg_replication_slots WHERE slot_name = 'replica_1_slot';

-- Drop and recreate if needed
SELECT pg_drop_replication_slot('replica_1_slot');
SELECT pg_create_physical_replication_slot('replica_1_slot');
```

Then re-run pg_basebackup on replica.

## Further Reading

- [PostgreSQL Replication Documentation](https://www.postgresql.org/docs/current/high-availability.html)
- [PgBouncer Documentation](https://www.pgbouncer.org/)
- [Patroni Documentation](https://patroni.readthedocs.io/)
- [Backup Scripts](../../scripts/backup-postgres.sh)
