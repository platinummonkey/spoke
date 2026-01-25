#!/bin/bash
#
# PostgreSQL Restore Script for Spoke
#
# This script restores a Spoke PostgreSQL database from a backup file.
#
# Usage:
#   ./restore-postgres.sh <backup-file>
#   ./restore-postgres.sh s3://<bucket>/backups/<backup-file>
#
# Examples:
#   ./restore-postgres.sh /var/backups/spoke/spoke-20260125-020000.sql.gz
#   ./restore-postgres.sh s3://spoke-backups/backups/spoke-20260125-020000.sql.gz
#
# Environment Variables:
#   SPOKE_POSTGRES_URL      - PostgreSQL connection string (required)
#   SPOKE_S3_REGION         - S3 region (default: us-east-1)
#   SKIP_CONFIRMATION       - Skip confirmation prompt (default: false)
#

set -e  # Exit on error
set -u  # Exit on undefined variable
set -o pipefail  # Exit on pipe failure

# Configuration
SPOKE_S3_REGION="${SPOKE_S3_REGION:-us-east-1}"
SKIP_CONFIRMATION="${SKIP_CONFIRMATION:-false}"
TEMP_DIR="/tmp/spoke-restore-$$"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Cleanup temporary files
cleanup() {
    if [ -d "${TEMP_DIR}" ]; then
        log_info "Cleaning up temporary files"
        rm -rf "${TEMP_DIR}"
    fi
}

# Register cleanup on exit
trap cleanup EXIT

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if [ $# -eq 0 ]; then
        log_error "Usage: $0 <backup-file>"
        log_error "       $0 s3://<bucket>/backups/<backup-file>"
        exit 1
    fi

    if [ -z "${SPOKE_POSTGRES_URL:-}" ]; then
        log_error "SPOKE_POSTGRES_URL environment variable is not set"
        exit 1
    fi

    if ! command -v psql &> /dev/null; then
        log_error "psql not found. Install postgresql-client."
        exit 1
    fi

    if ! command -v gunzip &> /dev/null; then
        log_error "gunzip not found"
        exit 1
    fi

    log_info "Prerequisites check passed"
}

# Download backup from S3 if needed
download_backup() {
    local BACKUP_SOURCE="$1"

    if [[ "${BACKUP_SOURCE}" == s3://* ]]; then
        log_info "Downloading backup from S3: ${BACKUP_SOURCE}"

        if ! command -v aws &> /dev/null; then
            log_error "aws CLI not found but S3 URL provided"
            exit 1
        fi

        mkdir -p "${TEMP_DIR}"
        local BACKUP_FILE="${TEMP_DIR}/$(basename ${BACKUP_SOURCE})"

        if aws s3 cp "${BACKUP_SOURCE}" "${BACKUP_FILE}" --region "${SPOKE_S3_REGION}"; then
            log_info "Download successful"
            echo "${BACKUP_FILE}"
        else
            log_error "S3 download failed"
            exit 1
        fi
    else
        if [ ! -f "${BACKUP_SOURCE}" ]; then
            log_error "Backup file not found: ${BACKUP_SOURCE}"
            exit 1
        fi
        echo "${BACKUP_SOURCE}"
    fi
}

# Verify backup file
verify_backup() {
    local BACKUP_FILE="$1"

    log_info "Verifying backup file: ${BACKUP_FILE}"

    # Check if file is gzipped
    if ! file "${BACKUP_FILE}" | grep -q "gzip compressed"; then
        log_error "Backup file is not gzipped"
        exit 1
    fi

    # Check file size
    FILE_SIZE=$(stat -f%z "${BACKUP_FILE}" 2>/dev/null || stat -c%s "${BACKUP_FILE}" 2>/dev/null)
    FILE_SIZE_MB=$((FILE_SIZE / 1024 / 1024))

    if [ "${FILE_SIZE}" -lt 1024 ]; then
        log_error "Backup file is suspiciously small (${FILE_SIZE} bytes)"
        exit 1
    fi

    log_info "Backup file size: ${FILE_SIZE_MB} MB"

    # Test gunzip
    if ! gunzip -t "${BACKUP_FILE}"; then
        log_error "Backup file is corrupted (gunzip test failed)"
        exit 1
    fi

    log_info "Backup file verification passed"
}

# Confirm restoration
confirm_restore() {
    if [ "${SKIP_CONFIRMATION}" = "true" ]; then
        return 0
    fi

    log_warn "================================================"
    log_warn "WARNING: This will DESTROY all existing data!"
    log_warn "================================================"
    log_warn ""
    log_warn "Database: ${SPOKE_POSTGRES_URL}"
    log_warn ""
    echo -n "Are you sure you want to continue? (type 'yes' to confirm): "
    read -r CONFIRMATION

    if [ "${CONFIRMATION}" != "yes" ]; then
        log_info "Restoration cancelled"
        exit 0
    fi

    log_info "Confirmation received, proceeding with restore"
}

# Get database name from connection string
get_db_name() {
    echo "${SPOKE_POSTGRES_URL}" | sed -n 's/.*\/\([^?]*\).*/\1/p'
}

# Drop and recreate database
recreate_database() {
    local DB_NAME=$(get_db_name)

    log_info "Dropping database: ${DB_NAME}"

    # Connect to postgres database to drop spoke database
    local POSTGRES_URL="${SPOKE_POSTGRES_URL/${DB_NAME}/postgres}"

    # Drop database (disconnect all users first)
    psql "${POSTGRES_URL}" -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DB_NAME}';" || true
    psql "${POSTGRES_URL}" -c "DROP DATABASE IF EXISTS ${DB_NAME};"

    log_info "Creating database: ${DB_NAME}"
    psql "${POSTGRES_URL}" -c "CREATE DATABASE ${DB_NAME};"
}

# Restore database
restore_database() {
    local BACKUP_FILE="$1"
    local DB_NAME=$(get_db_name)

    log_info "Restoring database from ${BACKUP_FILE}"
    log_info "Target database: ${DB_NAME}"

    # Restore with gunzip piped to psql
    if gunzip -c "${BACKUP_FILE}" | psql "${SPOKE_POSTGRES_URL}"; then
        log_info "Database restored successfully"
    else
        log_error "Database restore failed"
        exit 1
    fi
}

# Verify restored database
verify_restore() {
    local DB_NAME=$(get_db_name)

    log_info "Verifying restored database"

    # Count tables
    TABLE_COUNT=$(psql "${SPOKE_POSTGRES_URL}" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")

    if [ "${TABLE_COUNT}" -eq 0 ]; then
        log_error "No tables found in restored database"
        exit 1
    fi

    log_info "Found ${TABLE_COUNT} tables in database"

    # Count modules
    MODULE_COUNT=$(psql "${SPOKE_POSTGRES_URL}" -t -c "SELECT COUNT(*) FROM modules;" 2>/dev/null || echo "0")
    log_info "Found ${MODULE_COUNT} modules in database"

    log_info "Database verification passed"
}

# Main execution
main() {
    log_info "=== Spoke PostgreSQL Restore ==="

    check_prerequisites "$@"

    local BACKUP_SOURCE="$1"
    local BACKUP_FILE=$(download_backup "${BACKUP_SOURCE}")

    verify_backup "${BACKUP_FILE}"
    confirm_restore
    recreate_database
    restore_database "${BACKUP_FILE}"
    verify_restore

    log_info "=== Restore completed successfully ==="
}

# Run main function
main "$@"
