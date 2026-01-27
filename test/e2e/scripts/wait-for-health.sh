#!/bin/bash
# Wait for all services to be healthy

set -e

echo "Waiting for services to be healthy..."

MAX_WAIT=120  # 2 minutes
WAIT_INTERVAL=5

elapsed=0

while [ $elapsed -lt $MAX_WAIT ]; do
    echo "Checking service health (${elapsed}s/${MAX_WAIT}s)..."

    # Check PostgreSQL
    if podman exec spoke-postgres-test pg_isready -U spoke -d spoke 2>/dev/null | grep -q "accepting connections"; then
        echo "✓ PostgreSQL is healthy"
    else
        echo "  PostgreSQL not ready yet..."
        sleep $WAIT_INTERVAL
        elapsed=$((elapsed + WAIT_INTERVAL))
        continue
    fi

    # Check Redis
    if podman exec spoke-redis-test redis-cli ping 2>/dev/null | grep -q PONG; then
        echo "✓ Redis is healthy"
    else
        echo "  Redis not ready yet..."
        sleep $WAIT_INTERVAL
        elapsed=$((elapsed + WAIT_INTERVAL))
        continue
    fi

    # Check MinIO
    if curl -sf http://localhost:9000/minio/health/live > /dev/null 2>&1; then
        echo "✓ MinIO is healthy"
    else
        echo "  MinIO not ready yet..."
        sleep $WAIT_INTERVAL
        elapsed=$((elapsed + WAIT_INTERVAL))
        continue
    fi

    # Check Spoke API
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        echo "✓ Spoke API is healthy"
    else
        echo "  Spoke API not ready yet..."
        sleep $WAIT_INTERVAL
        elapsed=$((elapsed + WAIT_INTERVAL))
        continue
    fi

    # Check Web UI
    if curl -sf http://localhost:5173 > /dev/null 2>&1; then
        echo "✓ Web UI is healthy"
    else
        echo "  Web UI not ready yet..."
        sleep $WAIT_INTERVAL
        elapsed=$((elapsed + WAIT_INTERVAL))
        continue
    fi

    # All services healthy
    echo ""
    echo "✅ All services are healthy!"
    exit 0
done

echo ""
echo "❌ Timeout waiting for services to be healthy"
echo "Check logs with: podman-compose logs"
exit 1
