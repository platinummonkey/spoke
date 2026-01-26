#!/bin/bash
# Automated API testing script

set -e

API_URL="http://localhost:8080"
PASSED=0
FAILED=0

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

function test_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    local expected_status="$5"

    echo -n "Testing: $name... "

    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$API_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$API_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi

    status=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)

    if [ "$status" = "$expected_status" ]; then
        echo -e "${GREEN}✓ PASS${NC} (Status: $status)"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC} (Expected: $expected_status, Got: $status)"
        echo "Response: $body"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

echo "========================================="
echo "Spoke Plugin Ecosystem API Tests"
echo "========================================="
echo ""

# Test 1: Health endpoint
test_endpoint \
    "Health Check" \
    "GET" \
    "/health" \
    "" \
    "200"

# Test 2: List plugins (empty)
test_endpoint \
    "List Plugins (empty)" \
    "GET" \
    "/api/v1/plugins" \
    "" \
    "200"

# Test 3: Create plugin
test_endpoint \
    "Create Plugin" \
    "POST" \
    "/api/v1/plugins" \
    '{"id":"test-plugin","name":"Test Plugin","description":"Test","author":"E2E","license":"MIT","type":"language","security_level":"community"}' \
    "201"

# Test 4: Get plugin
test_endpoint \
    "Get Plugin" \
    "GET" \
    "/api/v1/plugins/test-plugin" \
    "" \
    "200"

# Test 5: Search plugins
test_endpoint \
    "Search Plugins" \
    "GET" \
    "/api/v1/plugins/search?q=test" \
    "" \
    "200"

# Test 6: Create plugin version
test_endpoint \
    "Create Plugin Version" \
    "POST" \
    "/api/v1/plugins/test-plugin/versions" \
    '{"version":"1.0.0","api_version":"1.0","download_url":"http://example.com/plugin.tar.gz","checksum":"abc123","size_bytes":1024}' \
    "201"

# Test 7: List versions
test_endpoint \
    "List Plugin Versions" \
    "GET" \
    "/api/v1/plugins/test-plugin/versions" \
    "" \
    "200"

# Test 8: Create review
test_endpoint \
    "Create Review" \
    "POST" \
    "/api/v1/plugins/test-plugin/reviews" \
    '{"rating":5,"review":"Great plugin!"}' \
    "201"

# Test 9: List reviews
test_endpoint \
    "List Reviews" \
    "GET" \
    "/api/v1/plugins/test-plugin/reviews" \
    "" \
    "200"

# Test 10: Record installation
test_endpoint \
    "Record Installation" \
    "POST" \
    "/api/v1/plugins/test-plugin/install" \
    '{"version":"1.0.0"}' \
    "200"

# Test 11: Get plugin stats
test_endpoint \
    "Get Plugin Stats" \
    "GET" \
    "/api/v1/plugins/test-plugin/stats" \
    "" \
    "200"

# Test 12: Submit verification
test_endpoint \
    "Submit Verification" \
    "POST" \
    "/api/v1/plugins/test-plugin/versions/1.0.0/verify" \
    '{"submitted_by":"test-user","auto_approve":false}' \
    "201"

# Test 13: Get verification stats
test_endpoint \
    "Get Verification Stats" \
    "GET" \
    "/api/v1/verifications/stats" \
    "" \
    "200"

echo ""
echo "========================================="
echo "Test Results"
echo "========================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All API tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some API tests failed${NC}"
    echo ""
    echo "To file bugs:"
    echo "bd create --title=\"API Test Failure: [test name]\" --type=bug --priority=1"
    exit 1
fi
