# End-to-End Testing Plan: Plugin Ecosystem

**Version:** 1.0
**Date:** 2026-01-25
**Status:** Initial Implementation

---

## Overview

This document defines the complete end-to-end testing strategy for the Spoke Plugin Ecosystem. It provides a repeatable process for validating the entire system from infrastructure setup through UI verification.

## Testing Goals

1. **Infrastructure Validation:** All services start correctly and communicate
2. **Database Integrity:** Migrations apply correctly, data persists
3. **API Functionality:** All endpoints work as specified
4. **UI Functionality:** User workflows complete successfully
5. **Plugin Lifecycle:** Plugins can be loaded, verified, installed, and used
6. **Security:** Verification catches malicious plugins
7. **Performance:** System handles expected load

---

## Prerequisites

### Required Tools
```bash
# Podman (or Docker)
podman --version  # >= 4.0

# Node.js (for Playwright)
node --version  # >= 18.0

# curl (for API testing)
curl --version

# jq (for JSON parsing)
jq --version
```

### Environment Setup
```bash
# Clone repository
cd /Users/cody.lee/go/src/github.com/platinummonkey/spoke

# Ensure test directory exists
mkdir -p test/e2e/playwright-results
mkdir -p test/e2e/playwright-report
mkdir -p test/e2e/test-plugins
```

---

## Phase 1: Infrastructure Setup (5 minutes)

### Step 1.1: Start Infrastructure
```bash
cd test/e2e

# Start all services with Podman Compose
podman-compose up -d

# Or with Docker Compose
docker-compose up -d

# Expected: 8 containers running
# - postgres, redis, minio
# - spoke-api, sprocket, plugin-verifier
# - spoke-web
# - playwright (profile=test, not started yet)
```

### Step 1.2: Wait for Health Checks
```bash
# Wait for all services to be healthy (max 2 minutes)
./scripts/wait-for-health.sh

# Or manually check
podman-compose ps

# Expected output:
# spoke-postgres-test    Up (healthy)
# spoke-redis-test       Up (healthy)
# spoke-minio-test       Up (healthy)
# spoke-api-test         Up (healthy)
# spoke-sprocket-test    Up
# spoke-verifier-test    Up
# spoke-web-test         Up
```

### Step 1.3: Verify Database Migrations
```bash
# Check migrations applied
podman exec spoke-postgres-test psql -U spoke -d spoke -c "\dt"

# Expected tables:
# - plugins
# - plugin_versions
# - plugin_reviews
# - plugin_installations
# - plugin_stats_daily
# - plugin_dependencies
# - plugin_tags
# - plugin_verifications
# - plugin_validation_errors
# - plugin_security_issues
# - plugin_permissions
# - plugin_verification_audit
# - plugin_scan_history
```

### Step 1.4: Initialize MinIO Bucket
```bash
# Create S3 bucket for plugins
podman exec spoke-minio-test mc alias set myminio http://localhost:9000 spoke spokespoke
podman exec spoke-minio-test mc mb myminio/spoke-plugins
podman exec spoke-minio-test mc policy set download myminio/spoke-plugins
```

**Success Criteria:**
- [x] All services running
- [x] All health checks passing
- [x] Database migrations applied (13 tables)
- [x] S3 bucket created

**If Failed:**
- Check logs: `podman-compose logs <service>`
- Verify ports not in use: `lsof -i :8080,3307,6380,9000`
- Restart services: `podman-compose restart <service>`

---

## Phase 2: API Testing (10 minutes)

### Step 2.1: Test Health Endpoints
```bash
# Spoke API health
curl -f http://localhost:8080/health
# Expected: {"status":"ok"}

# Check API version
curl http://localhost:8080/api/v1/version
# Expected: {"version":"..."}
```

### Step 2.2: Test Plugin Marketplace Endpoints

**List Plugins (should be empty initially):**
```bash
curl -s http://localhost:8080/api/v1/plugins | jq '.'
# Expected: {"plugins":[],"total":0}
```

**Create Test Plugin:**
```bash
# Create plugin metadata
curl -X POST http://localhost:8080/api/v1/plugins \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-plugin",
    "name": "Test Plugin",
    "description": "E2E test plugin",
    "author": "E2E Test",
    "license": "MIT",
    "type": "language",
    "security_level": "community"
  }' | jq '.'

# Expected: {"id":"test-plugin","name":"Test Plugin",...}
```

**Create Plugin Version:**
```bash
curl -X POST http://localhost:8080/api/v1/plugins/test-plugin/versions \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1.0.0",
    "api_version": "1.0",
    "download_url": "http://minio:9000/spoke-plugins/test-plugin-1.0.0.tar.gz",
    "checksum": "abc123"
  }' | jq '.'
```

**List Plugins Again:**
```bash
curl -s http://localhost:8080/api/v1/plugins | jq '.'
# Expected: 1 plugin
```

**Search Plugins:**
```bash
curl -s "http://localhost:8080/api/v1/plugins/search?q=test" | jq '.'
# Expected: Find test-plugin
```

**Get Plugin Details:**
```bash
curl -s http://localhost:8080/api/v1/plugins/test-plugin | jq '.'
# Expected: Full plugin details
```

### Step 2.3: Test Review System

**Create Review:**
```bash
curl -X POST http://localhost:8080/api/v1/plugins/test-plugin/reviews \
  -H "Content-Type: application/json" \
  -H "X-User-ID: test-user-1" \
  -d '{
    "rating": 5,
    "review": "Excellent plugin for testing!"
  }' | jq '.'
```

**List Reviews:**
```bash
curl -s http://localhost:8080/api/v1/plugins/test-plugin/reviews | jq '.'
# Expected: 1 review with 5-star rating
```

### Step 2.4: Test Installation Tracking

**Record Installation:**
```bash
curl -X POST http://localhost:8080/api/v1/plugins/test-plugin/install \
  -H "Content-Type: application/json" \
  -H "X-User-ID: test-user-1" \
  -d '{"version": "1.0.0"}' | jq '.'
```

**Get Plugin Stats:**
```bash
curl -s http://localhost:8080/api/v1/plugins/test-plugin/stats | jq '.'
# Expected: download_count: 1
```

### Step 2.5: Test Verification System

**Submit for Verification:**
```bash
curl -X POST http://localhost:8080/api/v1/plugins/test-plugin/versions/1.0.0/verify \
  -H "Content-Type: application/json" \
  -d '{
    "submitted_by": "test-user",
    "auto_approve": false
  }' | jq '.'
# Expected: {"verification_id":1,"status":"pending"}
```

**Check Verification Status:**
```bash
curl -s http://localhost:8080/api/v1/verifications/1 | jq '.'
# Expected: status changes from pending -> in_progress -> approved/rejected
```

**Get Verification Stats:**
```bash
curl -s http://localhost:8080/api/v1/verifications/stats | jq '.'
# Expected: Statistics about verifications
```

**Success Criteria:**
- [x] Health endpoints respond
- [x] Plugin CRUD operations work
- [x] Search returns correct results
- [x] Review system functional
- [x] Installation tracking works
- [x] Verification workflow executes

**If Failed, File Bug:**
```bash
bd create \
  --title="API Endpoint Failure: [endpoint name]" \
  --description="[Error details, expected vs actual]" \
  --type=bug \
  --priority=1
```

---

## Phase 3: Plugin Loader Testing (5 minutes)

### Step 3.1: Deploy Test Plugin

**Create Rust Plugin:**
```bash
mkdir -p test/e2e/test-plugins/rust-test

cat > test/e2e/test-plugins/rust-test/plugin.yaml <<EOF
id: rust-test
name: Rust Test Plugin
version: 1.0.0
api_version: 1.0.0
description: Test Rust language plugin
author: E2E Test
license: MIT
type: language
security_level: community

language_spec:
  id: rust
  name: Rust
  display_name: Rust (E2E Test)
  supports_grpc: true
  file_extensions: [".rs"]
  enabled: true
EOF

# Restart Sprocket to pick up new plugin
podman-compose restart sprocket
```

### Step 3.2: Verify Plugin Loaded

**Check Sprocket Logs:**
```bash
podman-compose logs sprocket | grep -i "rust-test"
# Expected: "Loaded plugin: rust-test"
```

**List Available Languages:**
```bash
curl -s http://localhost:8080/api/v1/languages | jq '.[] | select(.id=="rust")'
# Expected: Rust language available
```

**Success Criteria:**
- [x] Plugin discovered from filesystem
- [x] Plugin loaded successfully
- [x] Language available for compilation

---

## Phase 4: UI Testing with Playwright (15 minutes)

### Step 4.1: Run Automated UI Tests

**Start Playwright Tests:**
```bash
cd test/e2e

# Run all tests
podman-compose --profile test run playwright

# Or run specific test suite
podman-compose --profile test run playwright npx playwright test marketplace

# Or run in headed mode for debugging
podman-compose --profile test run playwright npx playwright test --headed
```

### Step 4.2: Review Test Results

**Check Test Report:**
```bash
# View HTML report
open test/e2e/playwright-report/index.html

# Or view in terminal
cat test/e2e/playwright-results/results.json | jq '.'
```

### Step 4.3: Manual UI Verification (if automated tests incomplete)

**Open Browser:**
```
http://localhost:5173/plugins
```

**Test Scenarios:**

**Scenario 1: Browse Marketplace**
1. Navigate to `/plugins`
2. Verify plugin grid displays
3. Verify search bar present
4. Verify filter dropdowns (Type, Security Level, Sort)
5. Verify pagination controls

**Expected:**
- ✅ Grid layout with plugin cards
- ✅ Search input functional
- ✅ Filters work
- ✅ Empty state if no plugins

**If Failed:** File bug with screenshot

**Scenario 2: Search Plugins**
1. Enter "test" in search bar
2. Verify results filter in real-time
3. Clear search
4. Verify all plugins return

**Expected:**
- ✅ Real-time filtering
- ✅ "test-plugin" appears in results
- ✅ Clear button works

**Scenario 3: View Plugin Details**
1. Click on "test-plugin" card
2. Navigate to plugin detail page
3. Verify tabs: Overview, Versions, Reviews
4. Click each tab

**Expected:**
- ✅ Detail page loads
- ✅ Three tabs visible
- ✅ Tab switching works
- ✅ Overview shows metadata
- ✅ Versions shows version table
- ✅ Reviews shows review form

**Scenario 4: Submit Review**
1. Go to Reviews tab
2. Click "Write a Review"
3. Select star rating (5 stars)
4. Enter review text
5. Click "Submit Review"

**Expected:**
- ✅ Form appears
- ✅ Star rating interactive
- ✅ Text area functional
- ✅ Submit button enabled
- ✅ Success notification
- ✅ Review appears in list

**Scenario 5: Install Plugin**
1. Click "Install" button on detail page
2. Verify installation confirmation
3. Check installation tracking

**Expected:**
- ✅ Install button clickable
- ✅ Success message displayed
- ✅ Download count increments

**Scenario 6: Filter and Sort**
1. Go back to marketplace
2. Select "Type: Language"
3. Select "Security: Community"
4. Select "Sort: Downloads"

**Expected:**
- ✅ Filters apply correctly
- ✅ Only matching plugins show
- ✅ Sort order changes

**Success Criteria:**
- [x] Marketplace page loads
- [x] Plugin cards render
- [x] Search works
- [x] Filters work
- [x] Detail page loads
- [x] Reviews can be submitted
- [x] Installation tracking works

**If Failed, File Bug:**
```bash
bd create \
  --title="UI Bug: [component/feature]" \
  --description="Steps to reproduce:\n1. ...\n2. ...\n\nExpected: ...\nActual: ...\n\nScreenshot: [attach]" \
  --type=bug \
  --priority=2
```

---

## Phase 5: Security Validation Testing (10 minutes)

### Step 5.1: Test Malicious Plugin Detection

**Create Malicious Plugin:**
```bash
mkdir -p /tmp/malicious-plugin

cat > /tmp/malicious-plugin/plugin.yaml <<EOF
id: malicious-plugin
name: Malicious Plugin
version: 1.0.0
api_version: 1.0.0
type: language
security_level: community
EOF

cat > /tmp/malicious-plugin/main.go <<'EOF'
package main

import (
    "os/exec"
    "syscall"
)

const apiKey = "sk-1234567890abcdef"

func main() {
    exec.Command("rm", "-rf", "/").Run()
    syscall.Exit(0)
}
EOF

# Package plugin
tar -czf /tmp/malicious-plugin.tar.gz -C /tmp/malicious-plugin .
```

**Upload to MinIO:**
```bash
podman exec spoke-minio-test mc cp /tmp/malicious-plugin.tar.gz myminio/spoke-plugins/
```

**Submit for Verification:**
```bash
curl -X POST http://localhost:8080/api/v1/plugins \
  -H "Content-Type: application/json" \
  -d '{
    "id": "malicious-plugin",
    "name": "Malicious Plugin",
    "type": "language",
    "security_level": "community"
  }'

curl -X POST http://localhost:8080/api/v1/plugins/malicious-plugin/versions \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1.0.0",
    "download_url": "http://minio:9000/spoke-plugins/malicious-plugin.tar.gz"
  }'

curl -X POST http://localhost:8080/api/v1/plugins/malicious-plugin/versions/1.0.0/verify \
  -H "Content-Type: application/json" \
  -d '{"submitted_by": "test-user", "auto_approve": true}' | jq '.'
```

**Wait for Verification (30 seconds):**
```bash
sleep 30

# Check verification result
VERIFICATION_ID=$(curl -s "http://localhost:8080/api/v1/verifications?status=completed&limit=1" | jq -r '.verifications[0].verification_id')

curl -s "http://localhost:8080/api/v1/verifications/$VERIFICATION_ID" | jq '.'
```

**Expected Results:**
```json
{
  "status": "rejected",
  "security_issues": [
    {
      "severity": "high",
      "category": "dangerous-import",
      "description": "Plugin imports potentially dangerous package: os/exec"
    },
    {
      "severity": "high",
      "category": "dangerous-import",
      "description": "Plugin imports potentially dangerous package: syscall"
    },
    {
      "severity": "high",
      "category": "hardcoded-secret",
      "description": "Potential hardcoded API Key detected"
    },
    {
      "severity": "high",
      "category": "suspicious-file-operation",
      "description": "Shell command execution detected"
    }
  ]
}
```

**Success Criteria:**
- [x] Malicious plugin detected
- [x] Status: rejected
- [x] All security issues found:
  - [x] os/exec import detected
  - [x] syscall import detected
  - [x] Hardcoded secret detected
  - [x] Shell command detected

**If Failed, File Critical Bug:**
```bash
bd create \
  --title="SECURITY: Malicious plugin not detected" \
  --description="Verification failed to detect [specific issue]\nPlugin: malicious-plugin\nVerification ID: $VERIFICATION_ID" \
  --type=bug \
  --priority=0
```

### Step 5.2: Test Safe Plugin Approval

**Create Safe Plugin:**
```bash
mkdir -p /tmp/safe-plugin

cat > /tmp/safe-plugin/plugin.yaml <<EOF
id: safe-plugin
name: Safe Plugin
version: 1.0.0
api_version: 1.0.0
type: language
security_level: community
EOF

cat > /tmp/safe-plugin/main.go <<'EOF'
package main

import (
    "fmt"
)

func main() {
    fmt.Println("Hello from safe plugin")
}
EOF

tar -czf /tmp/safe-plugin.tar.gz -C /tmp/safe-plugin .
```

**Submit and Verify:**
```bash
# Upload, create, submit (similar to malicious)
# ...

# Check result - should be approved
curl -s "http://localhost:8080/api/v1/verifications/$VERIFICATION_ID" | jq '.'
```

**Expected:**
```json
{
  "status": "approved",
  "security_level": "verified",
  "security_issues": [],
  "manifest_errors": []
}
```

**Success Criteria:**
- [x] Safe plugin approved
- [x] No security issues
- [x] Security level: verified

---

## Phase 6: Performance Testing (5 minutes)

### Step 6.1: Load Test Plugin Marketplace API

**Install Apache Bench:**
```bash
# macOS
brew install apache-bench

# Linux
sudo apt-get install apache2-utils
```

**Run Load Test:**
```bash
# Test plugin listing endpoint
ab -n 1000 -c 10 http://localhost:8080/api/v1/plugins

# Expected:
# - Requests per second: > 100
# - Mean time per request: < 100ms
# - Failed requests: 0
```

**Test Database Under Load:**
```bash
# Concurrent searches
for i in {1..50}; do
  curl -s "http://localhost:8080/api/v1/plugins/search?q=test" &
done
wait

# Check PostgreSQL connections
podman exec spoke-postgres-test psql -U spoke -d spoke -c "SELECT * FROM pg_stat_activity;"
```

**Success Criteria:**
- [x] > 100 req/s throughput
- [x] < 100ms average latency
- [x] 0 failed requests
- [x] Database connections stable

**If Failed:**
```bash
bd create \
  --title="Performance: [endpoint] slow" \
  --description="Load test results:\n- Throughput: X req/s (expected > 100)\n- Latency: Xms (expected < 100ms)" \
  --type=task \
  --priority=2
```

---

## Phase 7: Integration Testing (5 minutes)

### Step 7.1: Test Complete Plugin Workflow

**End-to-End Plugin Journey:**
```bash
# 1. Author creates plugin
# 2. Submit to marketplace
# 3. Verification runs
# 4. User discovers plugin
# 5. User installs plugin
# 6. User leaves review
# 7. Plugin used in compilation
```

**Automated Workflow Test:**
```bash
./test/e2e/scripts/test-plugin-workflow.sh

# This script automates:
# - Plugin creation
# - Marketplace submission
# - Verification
# - Installation
# - Review submission
# - Compilation test
```

**Success Criteria:**
- [x] Plugin created
- [x] Submitted successfully
- [x] Verification completes
- [x] Installation tracked
- [x] Review posted
- [x] Plugin usable in compilation

---

## Phase 8: Cleanup and Reporting (5 minutes)

### Step 8.1: Generate Test Report

**Create Summary Report:**
```bash
./test/e2e/scripts/generate-report.sh > test-report.md

# Report includes:
# - Infrastructure status
# - API test results
# - UI test results
# - Security test results
# - Performance metrics
# - Bugs filed
```

### Step 8.2: Stop Infrastructure

```bash
# Stop all services
podman-compose down

# Remove volumes (optional)
podman-compose down -v

# Clean up test data
rm -rf test/e2e/playwright-results/*
rm -rf /tmp/*-plugin
```

### Step 8.3: Review and File Bugs

**Review All Findings:**
```bash
bd list --status=open --type=bug | grep "E2E"
```

**Prioritize Bugs:**
- P0 (Critical): Security vulnerabilities
- P1 (High): API failures, data corruption
- P2 (Medium): UI bugs, performance issues
- P3 (Low): Minor UI glitches
- P4 (Backlog): Enhancements

---

## Automated Test Suite

### Playwright Test Structure

```
test/e2e/playwright/
├── tests/
│   ├── marketplace.spec.ts       # Marketplace browsing tests
│   ├── plugin-detail.spec.ts     # Plugin detail page tests
│   ├── search.spec.ts             # Search functionality tests
│   ├── reviews.spec.ts            # Review system tests
│   ├── installation.spec.ts       # Installation tracking tests
│   └── filters.spec.ts            # Filter and sort tests
├── fixtures/
│   ├── test-plugin.yaml          # Test plugin manifest
│   └── mock-data.json            # Mock API responses
├── utils/
│   ├── api-helpers.ts            # API testing utilities
│   └── test-helpers.ts           # Common test functions
├── playwright.config.ts          # Playwright configuration
└── package.json                  # Dependencies
```

### Key Test Scenarios

**1. Marketplace.spec.ts:**
- Test plugin grid rendering
- Test pagination
- Test empty state
- Test loading state
- Test error state

**2. Plugin-detail.spec.ts:**
- Test tab navigation
- Test metadata display
- Test version list
- Test review list
- Test install button

**3. Search.spec.ts:**
- Test search input
- Test debouncing
- Test result filtering
- Test no results state

**4. Reviews.spec.ts:**
- Test review form
- Test star rating
- Test review submission
- Test review validation
- Test review display

**5. Installation.spec.ts:**
- Test install button
- Test installation tracking
- Test download count increment

---

## Bug Template

When filing bugs during E2E testing, use this template:

```bash
bd create \
  --title="[Component]: [Brief description]" \
  --description="## Summary
[Brief description of the issue]

## Steps to Reproduce
1. [First step]
2. [Second step]
3. [...]

## Expected Behavior
[What should happen]

## Actual Behavior
[What actually happens]

## Environment
- Test Phase: [Phase number]
- Service: [spoke-api/spoke-web/etc.]
- Browser: [Chrome/Firefox/etc. if UI bug]
- Container Status: [Running/Failed/etc.]

## Logs
\`\`\`
[Relevant log output]
\`\`\`

## Screenshots
[If applicable]

## Severity
[Critical/High/Medium/Low]" \
  --type=bug \
  --priority=[0-4]
```

---

## Continuous Testing Integration

### Add to CI/CD Pipeline

```yaml
# .github/workflows/e2e-tests.yml
name: E2E Tests

on:
  pull_request:
  push:
    branches: [main]

jobs:
  e2e-tests:
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

      - name: Run API Tests
        run: ./test/e2e/scripts/test-api.sh

      - name: Run UI Tests
        run: |
          cd test/e2e
          podman-compose --profile test run playwright

      - name: Upload Test Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: test/e2e/playwright-results/

      - name: Stop Infrastructure
        if: always()
        run: podman-compose down
```

---

## Success Metrics

### Phase Completion Checklist

| Phase | Success Rate Target | Actual | Status |
|-------|-------------------|--------|--------|
| Infrastructure | 100% | - | ⬜ |
| API Testing | > 95% | - | ⬜ |
| Plugin Loader | 100% | - | ⬜ |
| UI Testing | > 90% | - | ⬜ |
| Security | 100% | - | ⬜ |
| Performance | > 90% | - | ⬜ |
| Integration | > 95% | - | ⬜ |

### Bug Severity Distribution Target

- P0 (Critical): 0 bugs
- P1 (High): < 5 bugs
- P2 (Medium): < 10 bugs
- P3 (Low): < 20 bugs
- P4 (Backlog): Unlimited

---

## Repeat Testing Schedule

**Per PR:** Automated API + UI tests
**Nightly:** Full E2E suite including performance
**Weekly:** Manual security testing + load testing
**Release:** Complete manual + automated suite

---

## Conclusion

This E2E test plan provides a repeatable, comprehensive validation process for the Plugin Ecosystem. Follow this plan for each iteration to ensure quality and catch regressions early.

**Next Steps:**
1. Implement missing Playwright tests
2. Create automation scripts
3. Set up CI/CD pipeline
4. Schedule regular test runs
5. Track metrics over time
