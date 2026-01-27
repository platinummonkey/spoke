#!/usr/bin/env bash
set -e

# Pre-pull all base images required for E2E tests
# This helps avoid credential helper issues in corporate environments

echo "Pre-pulling base images for E2E tests..."
echo "This may take a few minutes depending on your connection."
echo ""

# Detect whether to use docker or podman
if command -v podman &> /dev/null; then
    CONTAINER_CMD="podman"
elif command -v docker &> /dev/null; then
    CONTAINER_CMD="docker"
else
    echo "Error: Neither docker nor podman found in PATH"
    exit 1
fi

echo "Using: $CONTAINER_CMD"
echo ""

# Array of required images
images=(
    "golang:1.21-alpine"
    "alpine:latest"
    "mysql:8.0"
    "redis:7-alpine"
    "minio/minio:latest"
)

# Pull each image
for image in "${images[@]}"; do
    echo "Pulling $image..."
    if $CONTAINER_CMD pull "$image"; then
        echo "✓ Successfully pulled $image"
    else
        echo "✗ Failed to pull $image"
        echo "  You may need to check your network connection or docker configuration"
    fi
    echo ""
done

echo "Pre-pull complete!"
echo ""
echo "You can now run: $CONTAINER_CMD-compose build"
