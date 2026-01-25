# Spoke HA Tests - Quick Start Guide

## TL;DR - Run All Tests

```bash
# From project root
bash tests/run_all_tests.sh
```

**Output:** Test report in `docs/testing/TEST_RESULTS.md`

## Run Tests With Full HA Stack

```bash
# Terminal 1: Start HA infrastructure
cd deployments/docker-compose
docker compose -f ha-stack.yml up

# Terminal 2: Run tests (wait 30-60 seconds after starting stack)
RUN_CHAOS_TESTS=yes bash tests/run_all_tests.sh
```

## Run Specific Test Types

```bash
# Integration tests only (no infrastructure needed)
go test -v ./tests/integration/...

# Backup tests only
bash tests/integration/backup_test.sh

# Chaos tests only (requires HA stack running)
bash tests/chaos/docker_compose_test.sh

# Performance benchmarks only
go test -bench=. ./tests/performance/...
```

## View Results

```bash
# Full test report
cat docs/testing/TEST_RESULTS.md

# Test logs
tail -100 /tmp/spoke-integration-tests.log
tail -100 /tmp/spoke-chaos-tests.log
```

## What Each Test Does

### Integration Tests (606 lines)
- Database connection management (primary + replicas)
- Redis caching (hit/miss, invalidation)
- Distributed rate limiting
- OpenTelemetry tracing
- Health checks

### Chaos Tests (371 lines)
- Service startup validation
- Load balancer distribution
- Single instance failure recovery
- PostgreSQL replication checks
- Redis Sentinel validation
- Load testing (50 requests)
- Random failure injection

### Backup Tests (271 lines)
- Backup script validation
- File format and compression
- Restore script validation
- Retention policy checks

### Performance Benchmarks (351 lines)
- Module creation/retrieval
- Cache performance
- Version operations
- Redis operations
- Connection pool behavior

## Expected Behavior

**Without Infrastructure:**
- Integration tests: Some SKIP (expected)
- Backup tests: PASS
- Chaos tests: SKIP
- Benchmarks: SKIP

**With HA Stack:**
- All tests: PASS/RUN

## Troubleshooting

**Tests skip due to missing PostgreSQL/Redis?**
- This is expected and normal
- Tests gracefully skip when dependencies unavailable
- Full test coverage requires HA stack running

**HA stack won't start?**
```bash
cd deployments/docker-compose
docker compose -f ha-stack.yml down -v
docker compose -f ha-stack.yml up -d
```

**Chaos tests fail?**
- Wait 60 seconds after starting HA stack
- Check service health: `docker compose -f ha-stack.yml ps`
- View logs: `docker compose -f ha-stack.yml logs`

## More Information

- Full documentation: `tests/README.md`
- Test results: `docs/testing/TEST_RESULTS.md`
- Testing summary: `docs/testing/TESTING_SUMMARY.md`
- HA stack guide: `deployments/docker-compose/README.md`
