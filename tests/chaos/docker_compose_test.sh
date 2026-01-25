#!/bin/bash

# Chaos Testing for Spoke HA Stack
# This script tests the HA implementation by simulating failures

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
COMPOSE_DIR="${PROJECT_ROOT}/deployments/docker-compose"
COMPOSE_FILE="${COMPOSE_DIR}/ha-stack.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

test_passed() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    log_info "✓ $1"
}

test_failed() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    log_error "✗ $1"
}

# Wait for service to be healthy
wait_for_service() {
    local service=$1
    local max_attempts=${2:-30}
    local attempt=1

    log_info "Waiting for $service to be healthy..."

    while [ $attempt -le $max_attempts ]; do
        if docker compose -f "$COMPOSE_FILE" ps "$service" | grep -q "healthy\|running"; then
            log_info "$service is healthy"
            return 0
        fi

        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done

    log_error "$service failed to become healthy after ${max_attempts} attempts"
    return 1
}

# Check if service is responding
check_http_service() {
    local url=$1
    local expected_code=${2:-200}

    local status_code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")

    if [ "$status_code" = "$expected_code" ]; then
        return 0
    else
        log_warn "Expected HTTP $expected_code, got $status_code from $url"
        return 1
    fi
}

# Test: Verify all services start successfully
test_startup() {
    log_info "Test: Verify all services start successfully"

    cd "$COMPOSE_DIR"

    # Start the stack
    log_info "Starting HA stack..."
    docker compose -f ha-stack.yml up -d

    sleep 10

    # Check all services are running
    local services=("postgres-primary" "spoke-1" "spoke-2" "spoke-3" "nginx" "redis-master")

    for service in "${services[@]}"; do
        if docker compose -f ha-stack.yml ps "$service" | grep -q "Up\|running"; then
            test_passed "$service is running"
        else
            test_failed "$service is not running"
            docker compose -f ha-stack.yml logs "$service" | tail -20
        fi
    done
}

# Test: Verify NGINX load balancer distributes traffic
test_load_balancer() {
    log_info "Test: Verify NGINX load balancer"

    sleep 5

    # Check if NGINX is accessible
    if check_http_service "http://localhost:8080/health/live"; then
        test_passed "NGINX is accessible and routing traffic"
    else
        test_failed "NGINX is not accessible"
        return 1
    fi

    # Make multiple requests and check responses
    local success_count=0
    for i in {1..10}; do
        if check_http_service "http://localhost:8080/health/live"; then
            success_count=$((success_count + 1))
        fi
        sleep 0.5
    done

    if [ $success_count -ge 8 ]; then
        test_passed "Load balancer successfully handled $success_count/10 requests"
    else
        test_failed "Load balancer only handled $success_count/10 requests"
    fi
}

# Test: Kill one Spoke instance and verify traffic continues
test_single_instance_failure() {
    log_info "Test: Single Spoke instance failure"

    # Kill spoke-1
    log_info "Stopping spoke-1..."
    docker compose -f "$COMPOSE_FILE" stop spoke-1

    sleep 3

    # Verify traffic still works
    local success_count=0
    for i in {1..10}; do
        if check_http_service "http://localhost:8080/health/live"; then
            success_count=$((success_count + 1))
        fi
        sleep 0.5
    done

    if [ $success_count -ge 8 ]; then
        test_passed "Traffic continued after spoke-1 failure ($success_count/10 requests successful)"
    else
        test_failed "Traffic disrupted after spoke-1 failure ($success_count/10 requests successful)"
    fi

    # Restart spoke-1
    log_info "Restarting spoke-1..."
    docker compose -f "$COMPOSE_FILE" start spoke-1
    sleep 5
}

# Test: Verify PostgreSQL replication
test_postgres_replication() {
    log_info "Test: PostgreSQL replication"

    # Check if primary is accessible
    if docker compose -f "$COMPOSE_FILE" exec -T postgres-primary pg_isready -U spoke >/dev/null 2>&1; then
        test_passed "PostgreSQL primary is accessible"
    else
        test_failed "PostgreSQL primary is not accessible"
        return 1
    fi

    # Check if replica is accessible (if configured)
    if docker compose -f "$COMPOSE_FILE" ps | grep -q "postgres-replica-1"; then
        if docker compose -f "$COMPOSE_FILE" exec -T postgres-replica-1 pg_isready -U spoke >/dev/null 2>&1; then
            test_passed "PostgreSQL replica is accessible"
        else
            test_failed "PostgreSQL replica is not accessible"
        fi
    else
        log_info "No PostgreSQL replica configured (this is OK)"
    fi
}

# Test: Verify Redis Sentinel
test_redis_sentinel() {
    log_info "Test: Redis Sentinel"

    # Check if Redis master is accessible
    if docker compose -f "$COMPOSE_FILE" exec -T redis-master redis-cli ping 2>/dev/null | grep -q "PONG"; then
        test_passed "Redis master is accessible"
    else
        log_warn "Redis master is not accessible (may be using Sentinel)"
    fi

    # Check if Sentinel is running
    if docker compose -f "$COMPOSE_FILE" ps | grep -q "redis-sentinel"; then
        if docker compose -f "$COMPOSE_FILE" exec -T redis-sentinel redis-cli -p 26379 ping 2>/dev/null | grep -q "PONG"; then
            test_passed "Redis Sentinel is running"
        else
            test_failed "Redis Sentinel is not responding"
        fi
    else
        log_info "Redis Sentinel not configured (using standalone Redis)"
    fi
}

# Test: Verify OTel Collector
test_otel_collector() {
    log_info "Test: OpenTelemetry Collector"

    if docker compose -f "$COMPOSE_FILE" ps | grep -q "otel-collector"; then
        if docker compose -f "$COMPOSE_FILE" ps otel-collector | grep -q "Up\|running"; then
            test_passed "OTel Collector is running"

            # Check if OTel Collector is receiving telemetry (check health endpoint if available)
            if check_http_service "http://localhost:13133" 2>/dev/null; then
                test_passed "OTel Collector health endpoint is accessible"
            else
                log_info "OTel Collector health endpoint not accessible (may not be configured)"
            fi
        else
            test_failed "OTel Collector is not running"
        fi
    else
        log_info "OTel Collector not configured"
    fi
}

# Test: Simple load test
test_load() {
    log_info "Test: Simple load test"

    local total_requests=50
    local success_count=0
    local start_time=$(date +%s)

    log_info "Making $total_requests requests..."

    for i in $(seq 1 $total_requests); do
        if check_http_service "http://localhost:8080/health/live" 2>/dev/null; then
            success_count=$((success_count + 1))
        fi
    done

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local success_rate=$((success_count * 100 / total_requests))

    log_info "Completed $success_count/$total_requests requests in ${duration}s (${success_rate}% success rate)"

    if [ $success_rate -ge 95 ]; then
        test_passed "Load test passed with ${success_rate}% success rate"
    else
        test_failed "Load test failed with only ${success_rate}% success rate"
    fi
}

# Test: Chaos - Kill random Spoke instance
test_random_failure() {
    log_info "Test: Random Spoke instance failure"

    local instances=("spoke-1" "spoke-2" "spoke-3")
    local random_index=$((RANDOM % 3))
    local instance="${instances[$random_index]}"

    log_info "Killing $instance randomly..."
    docker compose -f "$COMPOSE_FILE" stop "$instance"

    sleep 2

    # Verify traffic continues
    local success_count=0
    for i in {1..10}; do
        if check_http_service "http://localhost:8080/health/live"; then
            success_count=$((success_count + 1))
        fi
        sleep 0.5
    done

    if [ $success_count -ge 8 ]; then
        test_passed "Traffic continued after $instance failure"
    else
        test_failed "Traffic disrupted after $instance failure"
    fi

    # Restart instance
    log_info "Restarting $instance..."
    docker compose -f "$COMPOSE_FILE" start "$instance"
    sleep 3
}

# Test: Health check endpoints
test_health_endpoints() {
    log_info "Test: Health check endpoints"

    # Test liveness endpoint
    if check_http_service "http://localhost:8080/health/live" 200; then
        test_passed "Liveness endpoint returns 200"
    else
        test_failed "Liveness endpoint failed"
    fi

    # Test readiness endpoint
    if check_http_service "http://localhost:8080/health/ready" 200; then
        test_passed "Readiness endpoint returns 200"
    else
        log_warn "Readiness endpoint may not be configured or dependencies are unhealthy"
    fi
}

# Main execution
main() {
    log_info "Starting Spoke HA Chaos Testing"
    log_info "================================"

    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker first."
        exit 1
    fi

    # Check if compose file exists
    if [ ! -f "$COMPOSE_FILE" ]; then
        log_error "Docker Compose file not found: $COMPOSE_FILE"
        exit 1
    fi

    # Run tests
    test_startup
    test_load_balancer
    test_postgres_replication
    test_redis_sentinel
    test_otel_collector
    test_health_endpoints
    test_load
    test_single_instance_failure
    test_random_failure

    # Cleanup (optional - comment out to keep stack running)
    log_info ""
    log_info "To stop the HA stack, run:"
    log_info "  cd $COMPOSE_DIR && docker compose -f ha-stack.yml down"

    # Print summary
    log_info ""
    log_info "================================"
    log_info "Test Summary"
    log_info "================================"
    log_info "Passed: $TESTS_PASSED"
    log_info "Failed: $TESTS_FAILED"

    if [ $TESTS_FAILED -eq 0 ]; then
        log_info "All tests passed!"
        exit 0
    else
        log_error "Some tests failed"
        exit 1
    fi
}

# Run main function
main "$@"
