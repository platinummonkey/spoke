#!/bin/bash
set -e

# Build search index for Spoke web UI
# This script builds the search indexer and generates the search index

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "==> Building search indexer..."
cd "$PROJECT_ROOT"
go build -o bin/search-indexer ./cmd/search-indexer

echo "==> Generating search index..."
./bin/search-indexer \
  -storage-dir="${STORAGE_DIR:-./storage}" \
  -output="${OUTPUT:-./web/public/search-index.json}"

echo "==> Search index built successfully!"
echo "Output: ${OUTPUT:-./web/public/search-index.json}"
