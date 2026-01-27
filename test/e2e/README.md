# Spoke E2E Testing Suite

Complete end-to-end testing infrastructure for the Spoke Plugin Ecosystem using Podman/Docker Compose and Playwright.

## Quick Start

### Prerequisites
```bash
# Install Podman (or Docker)
brew install podman podman-compose

# Or use Docker
brew install docker docker-compose

# Start Podman machine (if using Podman)
podman machine init
podman machine start
```

### Run Full E2E Test Suite

```bash
# Navigate to test directory
cd test/e2e

# Start infrastructure
podman-compose up -d

# Wait for services to be healthy
./scripts/wait-for-health.sh

# Run API tests
./scripts/test-api.sh

# Run complete workflow test
./scripts/test-plugin-workflow.sh

# Run UI tests (Playwright)
podman-compose --profile test run playwright

# Generate report
./scripts/generate-report.sh > test-report-$(date +%Y%m%d).md

# Stop infrastructure
podman-compose down
```

### Run Individual Test Suites

**API Tests Only:**
```bash
./scripts/test-api.sh
```

**UI Tests Only:**
```bash
podman-compose --profile test run playwright

# Or specific test file
podman-compose --profile test run playwright npx playwright test marketplace

# Or in headed mode for debugging
podman-compose --profile test run playwright npx playwright test --headed
```

**Plugin Workflow Test:**
```bash
./scripts/test-plugin-workflow.sh
```

## Architecture

### Services

| Service | Port | Purpose |
|---------|------|---------|
| MySQL | 3307 | Database |
| Redis | 6380 | Cache |
| MinIO | 9000/9001 | S3-compatible storage |
| Spoke API | 8080 | REST API server |
| Sprocket | - | Compilation service |
| Plugin Verifier | - | Security verification |
| Web UI | 5173 | React frontend |
| Playwright | - | UI test automation |

### Test Structure

```
test/e2e/
├── docker-compose.yml          # Infrastructure definition
├── Dockerfile.*                # Service containers
├── E2E_TEST_PLAN.md           # Complete test plan
├── README.md                   # This file
├── scripts/
│   ├── wait-for-health.sh     # Health check script
│   ├── test-api.sh            # API test automation
│   ├── test-plugin-workflow.sh # Complete workflow test
│   └── generate-report.sh      # Report generation
└── playwright/
    ├── tests/                  # Playwright test files
    │   ├── marketplace.spec.ts
    │   └── search.spec.ts
    ├── playwright.config.ts
    └── package.json
```

## Testing Phases

1. **Infrastructure Setup** (5 min) - Start all services
2. **API Testing** (10 min) - Test all REST endpoints
3. **Plugin Loader** (5 min) - Test plugin discovery
4. **UI Testing** (15 min) - Automated browser tests
5. **Security** (10 min) - Test malicious plugin detection
6. **Performance** (5 min) - Load testing
7. **Integration** (5 min) - End-to-end workflows
8. **Cleanup** (5 min) - Generate reports, stop services

**Total Time:** ~60 minutes for complete suite

## Common Tasks

### Debug Failed Service

```bash
# Check logs
podman-compose logs <service-name>

# Example: Check Spoke API logs
podman-compose logs spoke-api

# Follow logs in real-time
podman-compose logs -f spoke-api
```

### Access Service Directly

```bash
# MySQL
podman exec -it spoke-mysql-test mysql -uspoke -pspoke spoke

# Redis
podman exec -it spoke-redis-test redis-cli

# MinIO Console
open http://localhost:9001
# Login: spoke / spokespoke
```

### Reset Database

```bash
# Stop services
podman-compose down

# Remove volumes
podman-compose down -v

# Start fresh
podman-compose up -d
```

### Run Single Playwright Test

```bash
podman-compose --profile test run playwright \
  npx playwright test tests/marketplace.spec.ts
```

### Debug Playwright Test

```bash
# Run in headed mode
podman-compose --profile test run playwright \
  npx playwright test --headed

# Or use debug mode
podman-compose --profile test run playwright \
  npx playwright test --debug
```

## Filing Bugs

When tests fail, file bugs using the beads system:

```bash
# API bug
bd create \
  --title="API Endpoint Failure: [endpoint]" \
  --description="Test: [test name]
Error: [error message]
Expected: [expected behavior]
Actual: [actual behavior]" \
  --type=bug \
  --priority=1

# UI bug
bd create \
  --title="UI Bug: [component]" \
  --description="Test: [test name]
Steps to reproduce:
1. ...
2. ...
Expected: [expected]
Actual: [actual]
Screenshot: [path]" \
  --type=bug \
  --priority=2

# Security bug (CRITICAL)
bd create \
  --title="SECURITY: [issue]" \
  --description="[Details]" \
  --type=bug \
  --priority=0
```

## Continuous Integration

### GitHub Actions Integration

Create `.github/workflows/e2e-tests.yml`:

```yaml
name: E2E Tests

on:
  pull_request:
  push:
    branches: [main]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Podman
        run: |
          sudo apt-get update
          sudo apt-get install -y podman podman-compose

      - name: Start Infrastructure
        run: |
          cd test/e2e
          podman-compose up -d
          ./scripts/wait-for-health.sh

      - name: Run Tests
        run: |
          cd test/e2e
          ./scripts/test-api.sh
          ./scripts/test-plugin-workflow.sh
          podman-compose --profile test run playwright

      - name: Upload Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: test/e2e/playwright-results/

      - name: Cleanup
        if: always()
        run: podman-compose down
```

## Performance Benchmarks

Expected performance metrics:

| Metric | Target | Measured |
|--------|--------|----------|
| API Throughput | > 100 req/s | - |
| API Latency (p95) | < 100ms | - |
| Page Load Time | < 2s | - |
| Plugin Verification | < 30s | - |

## Troubleshooting

### Services Won't Start

**Problem:** Ports already in use

**Solution:**
```bash
# Check what's using the ports
lsof -i :8080,3307,6380,9000

# Kill processes or change ports in docker-compose.yml
```

**Problem:** Podman machine not running

**Solution:**
```bash
podman machine start
```

### Container Build Failures (Corporate Credential Helper)

**Problem:** Docker/Podman builds fail with credential helper errors:
```
[ddtool] retrieving identity jwt: context deadline exceeded
Error: error getting credentials - err: exit status 1
```

**Cause:** Corporate credential helper (ddtool, ecr-login, etc.) attempts to authenticate even for public Docker Hub images, causing timeouts when corporate VPN/vault is unreachable.

**Solution 1: Pre-pull Images (Recommended)**

Use the provided script to pre-pull all required images:

```bash
# Navigate to test/e2e directory
cd test/e2e

# Pre-pull all base images (works with docker or podman)
./scripts/pre-pull-images.sh

# Now build with docker-compose (pulls are cached)
docker-compose build

# Start services
docker-compose up -d
```

Or manually pre-pull each image:

```bash
docker pull golang:1.21-alpine
docker pull alpine:latest
docker pull mysql:8.0
docker pull redis:7-alpine
docker pull minio/minio:latest
```

**Solution 2: Temporary Credential Helper Bypass**

Temporarily disable credential helpers for public registries:

```bash
# Backup your Docker config
cp ~/.docker/config.json ~/.docker/config.json.backup

# Remove credHelpers for the session
jq 'del(.credHelpers)' ~/.docker/config.json > ~/.docker/config.json.tmp
mv ~/.docker/config.json.tmp ~/.docker/config.json

# Run your builds
cd test/e2e
docker-compose up -d

# Restore original config
mv ~/.docker/config.json.backup ~/.docker/config.json
```

**Solution 3: Use Plain Docker Build**

Build services individually without compose:

```bash
cd ../..  # Go to repo root

# Build each service
docker build -f test/e2e/Dockerfile.spoke -t spoke-api:local .
docker build -f test/e2e/Dockerfile.sprocket -t sprocket:local .
docker build -f test/e2e/Dockerfile.verifier -t verifier:local .
docker build -f test/e2e/Dockerfile.web -t spoke-web:local .
docker build -f test/e2e/Dockerfile.playwright -t playwright:local .

# Update docker-compose.yml to use local tags
# (See docker-compose.override.yml example below)
```

**Solution 4: Skip Container Builds**

Build binaries locally and run tests without containers:

```bash
# Build locally
cd ../..  # Go to repo root
make build

# Start only infrastructure services (MySQL, Redis, MinIO)
cd test/e2e
docker-compose up -d mysql redis minio

# Run binaries directly
export DATABASE_URL="spoke:spoke@tcp(localhost:3307)/spoke?parseTime=true"
export REDIS_URL="localhost:6380"
../../bin/spoke-api &
../../bin/sprocket &

# Run tests
./scripts/test-api.sh
```

**Prevention: Configure Docker for Corporate Networks**

Add to `~/.docker/config.json` to explicitly route only corporate registries through credential helpers:

```json
{
  "auths": {},
  "credHelpers": {
    "registry.company.com": "ddtool",
    "registry-internal.company.com": "ddtool"
  },
  "credStore": ""
}
```

This ensures only internal registries use credential helpers, while public registries (Docker Hub, etc.) use anonymous pulls.

### Tests Timing Out

**Problem:** Services not ready

**Solution:**
```bash
# Increase wait time in wait-for-health.sh
# Or manually wait longer before running tests
sleep 30
./scripts/test-api.sh
```

### Database Migration Errors

**Problem:** Tables not created

**Solution:**
```bash
# Check migration logs
podman-compose logs mysql

# Manually run migrations
podman exec spoke-mysql-test mysql -uspoke -pspoke spoke < ../../migrations/010_plugin_marketplace.up.sql
```

### Playwright Tests Failing

**Problem:** Elements not found

**Solution:**
```bash
# Run in headed mode to see what's happening
podman-compose --profile test run playwright npx playwright test --headed

# Or check screenshots in results
ls playwright-results/
```

## Test Maintenance

### Update Test Data

```bash
# Update test plugins
vi test/e2e/test-plugins/rust-test/plugin.yaml

# Restart Sprocket
podman-compose restart sprocket
```

### Add New Tests

1. Create new spec file: `playwright/tests/new-feature.spec.ts`
2. Follow existing test patterns
3. Run tests: `podman-compose --profile test run playwright`
4. Update documentation

### Update Dependencies

```bash
# Update Playwright
cd playwright
npm update @playwright/test

# Rebuild Playwright container
podman-compose build playwright
```

## Resources

- [Complete Test Plan](E2E_TEST_PLAN.md) - Detailed testing procedures
- [Playwright Docs](https://playwright.dev) - UI testing framework
- [Podman Compose](https://github.com/containers/podman-compose) - Container orchestration

## Support

For issues with E2E tests:
1. Check logs: `podman-compose logs <service>`
2. Review test plan: `cat E2E_TEST_PLAN.md`
3. File bug: `bd create --type=bug --title="E2E Test Issue: [description]"`
