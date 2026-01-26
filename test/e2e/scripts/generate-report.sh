#!/bin/bash
# Generate E2E test report

cat <<EOF
# E2E Test Report

**Date:** $(date)
**Test Suite:** Plugin Ecosystem End-to-End Tests

---

## Infrastructure Status

EOF

# Check service status
echo "### Service Health"
echo ""
echo "| Service | Status |"
echo "|---------|--------|"

services=("spoke-mysql-test:MySQL" "spoke-redis-test:Redis" "spoke-minio-test:MinIO" "spoke-api-test:Spoke API" "spoke-sprocket-test:Sprocket" "spoke-verifier-test:Verifier" "spoke-web-test:Web UI")

for service in "${services[@]}"; do
    container="${service%%:*}"
    name="${service##*:}"

    if podman ps --filter "name=$container" --format "{{.Names}}" | grep -q "$container"; then
        echo "| $name | ✅ Running |"
    else
        echo "| $name | ❌ Stopped |"
    fi
done

echo ""
echo "---"
echo ""

# API Test Results
echo "## API Test Results"
echo ""

if [ -f "test/e2e/api-test-results.txt" ]; then
    cat test/e2e/api-test-results.txt
else
    echo "No API test results available. Run: ./scripts/test-api.sh"
fi

echo ""
echo "---"
echo ""

# UI Test Results
echo "## UI Test Results"
echo ""

if [ -f "test/e2e/playwright-results/results.json" ]; then
    echo "Playwright tests executed:"
    jq -r '.suites[] | "- \(.title): \(.specs | length) tests"' test/e2e/playwright-results/results.json
else
    echo "No UI test results available. Run: podman-compose --profile test run playwright"
fi

echo ""
echo "---"
echo ""

# Bug Summary
echo "## Bugs Filed During Testing"
echo ""

# List bugs created today
bd list --status=open --type=bug | grep "$(date +%Y-%m-%d)" || echo "No bugs filed today"

echo ""
echo "---"
echo ""

# Performance Metrics
echo "## Performance Metrics"
echo ""

if [ -f "test/e2e/performance-results.txt" ]; then
    cat test/e2e/performance-results.txt
else
    echo "No performance test results available."
fi

echo ""
echo "---"
echo ""

# Recommendations
echo "## Recommendations"
echo ""

# Check for failed services
failed_services=0
for service in "${services[@]}"; do
    container="${service%%:*}"
    if ! podman ps --filter "name=$container" --format "{{.Names}}" | grep -q "$container"; then
        failed_services=$((failed_services + 1))
    fi
done

if [ $failed_services -gt 0 ]; then
    echo "⚠️ **Action Required:** $failed_services service(s) not running"
    echo ""
fi

# Check for bugs
bug_count=$(bd list --status=open --type=bug | grep -c "$(date +%Y-%m-%d)" || echo "0")

if [ "$bug_count" -gt 0 ]; then
    echo "📋 **Action Required:** Review and prioritize $bug_count bug(s) filed today"
    echo ""
fi

echo "✅ End-to-end testing completed"
echo ""
echo "---"
echo ""
echo "**Next Steps:**"
echo "1. Review filed bugs and assign priorities"
echo "2. Address P0/P1 bugs before next release"
echo "3. Re-run tests after fixes"
echo "4. Update test plan based on findings"
EOF
