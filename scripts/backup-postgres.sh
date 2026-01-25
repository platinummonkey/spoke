#!/bin/bash
#
# PostgreSQL Backup Script for Spoke
#
# This script creates compressed backups of the Spoke PostgreSQL database
# and optionally uploads them to S3.
#
# Usage:
#   ./backup-postgres.sh
#
# Environment Variables:
#   SPOKE_POSTGRES_URL      - PostgreSQL connection string (required)
#   SPOKE_S3_BUCKET         - S3 bucket for backup storage (optional)
#   SPOKE_S3_REGION         - S3 region (default: us-east-1)
#   BACKUP_RETENTION_DAYS   - Days to retain backups (default: 7)
#   BACKUP_DIR              - Local backup directory (default: /var/backups/spoke)
#

set -e  # Exit on error
set -u  # Exit on undefined variable
set -o pipefail  # Exit on pipe failure

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/spoke}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-7}"
SPOKE_S3_REGION="${SPOKE_S3_REGION:-us-east-1}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="spoke-${TIMESTAMP}.sql.gz"
BACKUP_PATH="${BACKUP_DIR}/${BACKUP_FILE}"

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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if [ -z "${SPOKE_POSTGRES_URL:-}" ]; then
        log_error "SPOKE_POSTGRES_URL environment variable is not set"
        exit 1
    fi

    if ! command -v pg_dump &> /dev/null; then
        log_error "pg_dump not found. Install postgresql-client."
        exit 1
    fi

    if ! command -v gzip &> /dev/null; then
        log_error "gzip not found"
        exit 1
    fi

    if [ -n "${SPOKE_S3_BUCKET:-}" ]; then
        if ! command -v aws &> /dev/null; then
            log_warn "aws CLI not found. S3 upload will be skipped."
            SPOKE_S3_BUCKET=""
        fi
    fi

    log_info "Prerequisites check passed"
}

# Create backup directory
create_backup_dir() {
    if [ ! -d "${BACKUP_DIR}" ]; then
        log_info "Creating backup directory: ${BACKUP_DIR}"
        mkdir -p "${BACKUP_DIR}"
    fi
}

# Perform database backup
perform_backup() {
    log_info "Starting database backup to ${BACKUP_PATH}"

    # Create backup with pg_dump and compress with gzip
    if pg_dump "${SPOKE_POSTGRES_URL}" | gzip > "${BACKUP_PATH}"; then
        log_info "Backup created successfully"
    else
        log_error "Backup failed"
        exit 1
    fi

    # Verify backup file size
    FILE_SIZE=$(stat -f%z "${BACKUP_PATH}" 2>/dev/null || stat -c%s "${BACKUP_PATH}" 2>/dev/null)
    FILE_SIZE_MB=$((FILE_SIZE / 1024 / 1024))

    if [ "${FILE_SIZE}" -lt 1024 ]; then
        log_error "Backup file is suspiciously small (${FILE_SIZE} bytes)"
        exit 1
    fi

    log_info "Backup size: ${FILE_SIZE_MB} MB"
}

# Upload to S3
upload_to_s3() {
    if [ -z "${SPOKE_S3_BUCKET:-}" ]; then
        log_info "S3 upload skipped (SPOKE_S3_BUCKET not set)"
        return 0
    fi

    log_info "Uploading backup to S3: s3://${SPOKE_S3_BUCKET}/backups/${BACKUP_FILE}"

    if aws s3 cp "${BACKUP_PATH}" "s3://${SPOKE_S3_BUCKET}/backups/${BACKUP_FILE}" \
        --region "${SPOKE_S3_REGION}" \
        --storage-class STANDARD_IA; then
        log_info "S3 upload successful"
    else
        log_error "S3 upload failed"
        return 1
    fi
}

# Clean up old backups (local)
cleanup_local_backups() {
    log_info "Cleaning up local backups older than ${BACKUP_RETENTION_DAYS} days"

    find "${BACKUP_DIR}" -name "spoke-*.sql.gz" -type f -mtime +${BACKUP_RETENTION_DAYS} -delete

    REMAINING=$(find "${BACKUP_DIR}" -name "spoke-*.sql.gz" -type f | wc -l)
    log_info "Remaining local backups: ${REMAINING}"
}

# Clean up old backups (S3)
cleanup_s3_backups() {
    if [ -z "${SPOKE_S3_BUCKET:-}" ]; then
        return 0
    fi

    log_info "Cleaning up S3 backups older than ${BACKUP_RETENTION_DAYS} days"

    # Calculate cutoff date
    CUTOFF_DATE=$(date -d "${BACKUP_RETENTION_DAYS} days ago" +%Y%m%d 2>/dev/null || \
                  date -v-${BACKUP_RETENTION_DAYS}d +%Y%m%d 2>/dev/null)

    # List and delete old backups
    aws s3 ls "s3://${SPOKE_S3_BUCKET}/backups/" --region "${SPOKE_S3_REGION}" | \
        grep "spoke-" | \
        while read -r line; do
            BACKUP_NAME=$(echo "${line}" | awk '{print $4}')
            BACKUP_DATE=$(echo "${BACKUP_NAME}" | sed 's/spoke-\([0-9]\{8\}\).*/\1/')

            if [ "${BACKUP_DATE}" -lt "${CUTOFF_DATE}" ]; then
                log_info "Deleting old S3 backup: ${BACKUP_NAME}"
                aws s3 rm "s3://${SPOKE_S3_BUCKET}/backups/${BACKUP_NAME}" \
                    --region "${SPOKE_S3_REGION}"
            fi
        done
}

# Main execution
main() {
    log_info "=== Spoke PostgreSQL Backup ==="

    check_prerequisites
    create_backup_dir
    perform_backup
    upload_to_s3
    cleanup_local_backups
    cleanup_s3_backups

    log_info "=== Backup completed successfully ==="
    log_info "Backup location: ${BACKUP_PATH}"

    if [ -n "${SPOKE_S3_BUCKET:-}" ]; then
        log_info "S3 location: s3://${SPOKE_S3_BUCKET}/backups/${BACKUP_FILE}"
    fi
}

# Run main function
main "$@"
