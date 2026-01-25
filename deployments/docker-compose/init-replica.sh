#!/bin/bash
set -e

# This script initializes the PostgreSQL replica server

# Wait for primary to be ready
echo "Waiting for primary PostgreSQL server..."
until PGPASSWORD=$POSTGRES_PASSWORD psql -h postgres-primary -U spoke -d spoke -c '\q' 2>/dev/null; do
  echo "Primary not ready yet, waiting..."
  sleep 2
done

echo "Primary is ready, setting up replica..."

# Stop PostgreSQL if it's running
pg_ctl stop -D "$PGDATA" -m fast || true

# Remove existing data directory
rm -rf "$PGDATA"/*

# Create base backup from primary
echo "Creating base backup from primary..."
PGPASSWORD=replicator-password pg_basebackup \
  -h postgres-primary \
  -D "$PGDATA" \
  -U replicator \
  -v \
  -P \
  -R \
  -X stream \
  -S replica_slot

# Ensure proper permissions
chmod 0700 "$PGDATA"

echo "PostgreSQL replica initialized successfully"

# Start PostgreSQL
exec postgres
