# Spoke HA Testing Suite

This directory contains comprehensive tests for the Spoke High Availability implementation.

## Test Structure

```
tests/
├── integration/        # Integration tests for HA components
│   ├── ha_test.go     # Database, Redis, rate limiting, OTel tests
│   └── backup_test.sh # Backup and restore validation
├── chaos/             # Chaos engineering tests
│   └── docker_compose_test.sh
├── performance/       # Performance benchmarks
│   └── benchmark_test.go
├── run_all_tests.sh   # Comprehensive test runner
└── README.md          # This file
```

## Test Categories

### 1. Integration Tests

Tests core HA functionality:
- **Database Connection Management**: Primary/replica setup, round-robin selection, health checks
- **Redis Caching**: Cache hit/miss, pattern invalidation, fail-open behavior
- **Distributed Rate Limiting**: Redis-backed rate limiting across instances
- **OpenTelemetry**: Span creation, attributes, error recording
- **Health Checks**: Liveness and readiness endpoints

**Run:**
```bash
go test -v ./tests/integration/...
```

**Requirements:**
- PostgreSQL running on `localhost:5432` (optional - tests will skip if not available)
- Redis running on `localhost:6379` (optional - tests will skip if not available)

### 2. Backup/Restore Tests

Validates backup and restore procedures:
- Backup script existence and executability
- File format and compression
- Restore script validation
- Retention policy implementation

**Run:**
```bash
bash tests/integration/backup_test.sh
```

**Requirements:**
- Backup scripts in `scripts/` directory

### 3. Chaos Tests

Tests system resilience under failure conditions:
- Single Spoke instance failure
- Load balancer distribution
- PostgreSQL replication
- Redis Sentinel failover
- Random instance failures
- Load testing

**Run:**
```bash
# Start the HA stack first
cd deployments/docker-compose
docker compose -f ha-stack.yml up -d

# Wait for services to be healthy (30-60 seconds)
docker compose -f ha-stack.yml ps

# Run chaos tests
cd ../../tests/chaos
bash docker_compose_test.sh

# Or run from anywhere:
RUN_CHAOS_TESTS=yes bash tests/run_all_tests.sh
```

**Requirements:**
- Docker and Docker Compose installed
- HA stack running (postgres, redis, spoke instances, nginx)

### 4. Performance Benchmarks

Measures performance of HA components:
- Module creation/retrieval
- Cache hit/miss performance
- Version creation
- Redis operations
- Connection pool performance
- Replica round-robin selection

**Run:**
```bash
go test -bench=. -benchmem ./tests/performance/...
```

**Requirements:**
- PostgreSQL and Redis running (tests will skip if not available)

## Quick Start

### Run All Tests (Without Chaos)

```bash
bash tests/run_all_tests.sh
```

This will:
1. Run integration tests
2. Run backup/restore tests
3. Skip chaos tests (requires Docker stack)
4. Run performance benchmarks
5. Generate a comprehensive test report in `docs/testing/TEST_RESULTS.md`

### Run All Tests Including Chaos

```bash
# Terminal 1: Start HA stack
cd deployments/docker-compose
docker compose -f ha-stack.yml up

# Terminal 2: Run tests
RUN_CHAOS_TESTS=yes bash tests/run_all_tests.sh
```

### Run Specific Test Categories

```bash
# Integration tests only
go test -v ./tests/integration/...

# Specific test
go test -v -run TestOpenTelemetry ./tests/integration/...

# Backup tests
bash tests/integration/backup_test.sh

# Chaos tests (requires HA stack)
bash tests/chaos/docker_compose_test.sh

# Benchmarks
go test -bench=. ./tests/performance/...

# Specific benchmark
go test -bench=BenchmarkModuleCreation ./tests/performance/...
```

## Test Environment Variables

Configure test environment:

```bash
export TEST_POSTGRES_PRIMARY="postgres://spoke:spoke@localhost:5432/spoke?sslmode=disable"
export TEST_POSTGRES_REPLICA="postgres://spoke:spoke@localhost:5433/spoke?sslmode=disable"
export TEST_REDIS_URL="redis://localhost:6379/0"
```

## Docker Compose HA Stack

The chaos tests use the HA stack defined in `deployments/docker-compose/ha-stack.yml`.

**Services:**
- `postgres-primary`: PostgreSQL primary database
- `postgres-replica-1`: PostgreSQL read replica (streaming replication)
- `redis-master`: Redis cache
- `redis-sentinel`: Redis Sentinel for automatic failover
- `spoke-1`, `spoke-2`, `spoke-3`: Three Spoke instances
- `nginx`: Load balancer distributing traffic across Spoke instances
- `otel-collector`: OpenTelemetry collector for traces/metrics

**Start the stack:**
```bash
cd deployments/docker-compose
docker compose -f ha-stack.yml up -d
```

**Check status:**
```bash
docker compose -f ha-stack.yml ps
docker compose -f ha-stack.yml logs -f spoke-1
```

**Stop the stack:**
```bash
docker compose -f ha-stack.yml down
```

**Clean up (including volumes):**
```bash
docker compose -f ha-stack.yml down -v
```

## Chaos Test Scenarios

The chaos test script validates:

1. **Startup**: All services start successfully
2. **Load Balancer**: NGINX distributes traffic across instances
3. **Single Instance Failure**: Traffic continues when one Spoke instance fails
4. **PostgreSQL Replication**: Primary and replica are accessible
5. **Redis Sentinel**: Monitoring and failover capability
6. **OTel Collector**: Telemetry collection is working
7. **Load Test**: System handles 50+ concurrent requests
8. **Random Failure**: System handles random instance failures
9. **Health Endpoints**: Liveness and readiness checks work

## Test Results

After running `tests/run_all_tests.sh`, view the comprehensive report:

```bash
cat docs/testing/TEST_RESULTS.md
```

Test logs are saved in `/tmp/spoke-*-tests.log`:
- `/tmp/spoke-integration-tests.log`
- `/tmp/spoke-backup-tests.log`
- `/tmp/spoke-chaos-tests.log`
- `/tmp/spoke-benchmarks.log`

## Expected Test Behavior

### Without Infrastructure

When PostgreSQL and Redis are not running:
- ✅ Integration tests: Some tests SKIP (expected)
- ✅ Backup tests: PASS (validates scripts only)
- ⏭️ Chaos tests: SKIP (requires Docker stack)
- ⏭️ Benchmarks: SKIP (requires infrastructure)

**This is normal!** The tests are designed to skip gracefully when dependencies are unavailable.

### With Docker Compose Stack

When HA stack is running:
- ✅ Integration tests: All tests PASS
- ✅ Backup tests: All tests PASS
- ✅ Chaos tests: All tests PASS (validates resilience)
- ✅ Benchmarks: All benchmarks RUN (provides performance baseline)

## Continuous Integration

To run tests in CI/CD:

```yaml
# .github/workflows/test.yml
- name: Run HA Tests
  run: |
    # Start infrastructure
    docker compose -f deployments/docker-compose/ha-stack.yml up -d

    # Wait for services
    sleep 30

    # Run tests
    RUN_CHAOS_TESTS=yes bash tests/run_all_tests.sh

    # Cleanup
    docker compose -f deployments/docker-compose/ha-stack.yml down -v
```

## Troubleshooting

### Tests Fail to Connect to PostgreSQL

```bash
# Check if PostgreSQL is running
psql postgres://spoke:spoke@localhost:5432/spoke -c "SELECT 1"

# Start with Docker Compose
cd deployments/docker-compose
docker compose -f ha-stack.yml up -d postgres-primary
```

### Tests Fail to Connect to Redis

```bash
# Check if Redis is running
redis-cli -u redis://localhost:6379 ping

# Start with Docker Compose
cd deployments/docker-compose
docker compose -f ha-stack.yml up -d redis-master
```

### Chaos Tests Fail

```bash
# Check HA stack status
cd deployments/docker-compose
docker compose -f ha-stack.yml ps

# View logs
docker compose -f ha-stack.yml logs

# Restart stack
docker compose -f ha-stack.yml down
docker compose -f ha-stack.yml up -d
```

### Benchmarks Run Slowly

This is expected! Benchmarks measure performance and may take several minutes to complete.

```bash
# Run specific benchmark
go test -bench=BenchmarkModuleCreation -benchtime=5s ./tests/performance/...

# Run with less iterations
go test -bench=. -benchtime=1s ./tests/performance/...
```

## Contributing

When adding new tests:

1. **Integration tests**: Add to `tests/integration/ha_test.go`
2. **Chaos tests**: Add scenarios to `tests/chaos/docker_compose_test.sh`
3. **Benchmarks**: Add to `tests/performance/benchmark_test.go`
4. **Update documentation**: Update this README and test report template

### Test Writing Guidelines

- Use `testing.Short()` to skip long-running tests in short mode
- Skip gracefully when dependencies unavailable (don't fail)
- Add descriptive test names and comments
- Use subtests (`t.Run()`) for related test cases
- Clean up resources (defer cleanup functions)
- Use environment variables for configuration
- Document expected behavior

### Example Test

```go
func TestNewFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    t.Run("SuccessCase", func(t *testing.T) {
        // Setup
        // Test
        // Assert
        // Cleanup
    })

    t.Run("FailureCase", func(t *testing.T) {
        // Test failure scenario
    })
}
```

## References

- [Main HA Documentation](../deployments/docker-compose/README.md)
- [PostgreSQL Replication](../docs/operations/postgresql-replication.md)
- [Redis Sentinel](../docs/operations/redis-sentinel.md)
- [OpenTelemetry Setup](../docs/operations/opentelemetry-setup.md)
- [Test Results](../docs/testing/TEST_RESULTS.md)
