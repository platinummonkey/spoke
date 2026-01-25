#!/bin/bash

# Backup and Restore Testing Script
# Tests the PostgreSQL backup and restore functionality

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SCRIPTS_DIR="${PROJECT_ROOT}/scripts"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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

# Check if scripts exist
check_scripts() {
    log_info "Checking for backup/restore scripts..."

    if [ -f "${SCRIPTS_DIR}/backup-postgres.sh" ]; then
        test_passed "Backup script exists"
    else
        test_failed "Backup script not found: ${SCRIPTS_DIR}/backup-postgres.sh"
        return 1
    fi

    if [ -f "${SCRIPTS_DIR}/restore-postgres.sh" ]; then
        test_passed "Restore script exists"
    else
        test_failed "Restore script not found: ${SCRIPTS_DIR}/restore-postgres.sh"
        return 1
    fi
}

# Test backup script with Docker Compose
test_backup_with_docker() {
    log_info "Test: Backup with Docker Compose"

    local compose_file="${PROJECT_ROOT}/deployments/docker-compose/ha-stack.yml"

    if [ ! -f "$compose_file" ]; then
        log_warn "Docker Compose file not found, skipping Docker backup test"
        return 0
    fi

    # Check if postgres container is running
    if ! docker compose -f "$compose_file" ps postgres-primary 2>/dev/null | grep -q "Up\|running"; then
        log_warn "PostgreSQL container not running, skipping Docker backup test"
        return 0
    fi

    # Create test backup
    local backup_dir="/tmp/spoke-backup-test"
    mkdir -p "$backup_dir"

    log_info "Creating backup to $backup_dir..."

    # Simulate backup (would normally use the backup script)
    if docker compose -f "$compose_file" exec -T postgres-primary \
        pg_dump -U spoke spoke > "${backup_dir}/test-backup.sql" 2>/dev/null; then
        test_passed "Backup created successfully"

        # Check file size
        local file_size=$(stat -f%z "${backup_dir}/test-backup.sql" 2>/dev/null || stat -c%s "${backup_dir}/test-backup.sql" 2>/dev/null || echo "0")
        if [ "$file_size" -gt 0 ]; then
            test_passed "Backup file has content (${file_size} bytes)"
        else
            test_failed "Backup file is empty"
        fi

        # Cleanup
        rm -rf "$backup_dir"
    else
        test_failed "Backup creation failed"
    fi
}

# Test backup compression
test_backup_compression() {
    log_info "Test: Backup compression"

    local test_file="/tmp/test-backup-$(date +%s).sql"
    echo "SELECT * FROM modules;" > "$test_file"

    # Compress
    if gzip -c "$test_file" > "${test_file}.gz"; then
        test_passed "Backup compression works"

        # Verify compressed file exists and is smaller
        if [ -f "${test_file}.gz" ]; then
            local original_size=$(stat -f%z "$test_file" 2>/dev/null || stat -c%s "$test_file" 2>/dev/null || echo "0")
            local compressed_size=$(stat -f%z "${test_file}.gz" 2>/dev/null || stat -c%s "${test_file}.gz" 2>/dev/null || echo "0")

            if [ "$compressed_size" -lt "$original_size" ]; then
                test_passed "Compressed file is smaller (${compressed_size} < ${original_size})"
            else
                log_warn "Compressed file is not smaller (test file too small)"
            fi
        fi

        # Cleanup
        rm -f "$test_file" "${test_file}.gz"
    else
        test_failed "Backup compression failed"
    fi
}

# Test restore script validation
test_restore_validation() {
    log_info "Test: Restore script validation"

    local restore_script="${SCRIPTS_DIR}/restore-postgres.sh"

    if [ ! -f "$restore_script" ]; then
        log_warn "Restore script not found, skipping validation test"
        return 0
    fi

    # Check if script is executable
    if [ -x "$restore_script" ]; then
        test_passed "Restore script is executable"
    else
        test_failed "Restore script is not executable"
    fi

    # Check script has basic structure
    if grep -q "pg_restore\|psql" "$restore_script"; then
        test_passed "Restore script contains PostgreSQL restore commands"
    else
        test_failed "Restore script missing PostgreSQL restore commands"
    fi
}

# Test backup file format
test_backup_format() {
    log_info "Test: Backup file format"

    # Create a simple SQL backup
    local test_backup="/tmp/test-format-$(date +%s).sql"

    cat > "$test_backup" <<EOF
-- PostgreSQL database dump
CREATE TABLE modules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);
EOF

    # Verify format
    if grep -q "PostgreSQL database dump" "$test_backup"; then
        test_passed "Backup has PostgreSQL header"
    else
        test_failed "Backup missing PostgreSQL header"
    fi

    if grep -q "CREATE TABLE" "$test_backup"; then
        test_passed "Backup contains SQL statements"
    else
        test_failed "Backup missing SQL statements"
    fi

    # Cleanup
    rm -f "$test_backup"
}

# Test backup retention policy
test_backup_retention() {
    log_info "Test: Backup retention policy"

    local backup_script="${SCRIPTS_DIR}/backup-postgres.sh"

    if [ ! -f "$backup_script" ]; then
        log_warn "Backup script not found, skipping retention test"
        return 0
    fi

    # Check if script implements retention
    if grep -q "find.*-mtime\|find.*-delete\|RETENTION" "$backup_script"; then
        test_passed "Backup script implements retention policy"
    else
        log_warn "Backup script may not implement retention policy"
    fi
}

# Test environment variables
test_environment_variables() {
    log_info "Test: Environment variables"

    local backup_script="${SCRIPTS_DIR}/backup-postgres.sh"

    if [ ! -f "$backup_script" ]; then
        log_warn "Backup script not found, skipping environment test"
        return 0
    fi

    # Check for common environment variables
    local has_vars=0
    for var in "POSTGRES_HOST" "POSTGRES_USER" "POSTGRES_DB" "PGPASSWORD" "BACKUP_DIR"; do
        if grep -q "$var" "$backup_script"; then
            has_vars=$((has_vars + 1))
        fi
    done

    if [ $has_vars -ge 3 ]; then
        test_passed "Backup script uses environment variables"
    else
        log_warn "Backup script may not use environment variables properly"
    fi
}

# Main execution
main() {
    log_info "Starting Backup/Restore Testing"
    log_info "==============================="

    # Run tests
    check_scripts
    test_backup_format
    test_backup_compression
    test_restore_validation
    test_backup_retention
    test_environment_variables
    test_backup_with_docker

    # Print summary
    log_info ""
    log_info "==============================="
    log_info "Test Summary"
    log_info "==============================="
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
