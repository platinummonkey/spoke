#!/bin/bash
set -e

# This script initializes the PostgreSQL primary server for replication

# Create replication user if it doesn't exist
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create replication user
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$POSTGRES_REPLICATION_USER') THEN
            CREATE ROLE $POSTGRES_REPLICATION_USER WITH REPLICATION PASSWORD '$POSTGRES_REPLICATION_PASSWORD' LOGIN;
        END IF;
    END
    \$\$;

    -- Create physical replication slot for replica
    SELECT pg_create_physical_replication_slot('replica_slot') WHERE NOT EXISTS (
        SELECT 1 FROM pg_replication_slots WHERE slot_name = 'replica_slot'
    );

    -- Grant necessary permissions
    GRANT CONNECT ON DATABASE $POSTGRES_DB TO $POSTGRES_REPLICATION_USER;
EOSQL

# Update pg_hba.conf to allow replication connections
echo "host replication $POSTGRES_REPLICATION_USER all md5" >> "$PGDATA/pg_hba.conf"

# Reload PostgreSQL configuration
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -c "SELECT pg_reload_conf();"

echo "PostgreSQL primary initialized for replication"
