-- Migration 011 Rollback: Drop Plugin Verification System

-- Drop triggers first
DROP TRIGGER IF EXISTS update_plugin_security_level_on_approval;
DROP TRIGGER IF EXISTS record_verification_audit_on_update;

-- Drop views
DROP VIEW IF EXISTS plugin_security_scores;
DROP VIEW IF EXISTS recent_verifications;
DROP VIEW IF EXISTS pending_verifications;

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS plugin_scan_history;
DROP TABLE IF EXISTS plugin_verification_audit;
DROP TABLE IF EXISTS plugin_permissions;
DROP TABLE IF EXISTS plugin_security_issues;
DROP TABLE IF EXISTS plugin_validation_errors;
DROP TABLE IF EXISTS plugin_verifications;

-- Remove verification columns from plugins table (if they were added)
ALTER TABLE plugins
    DROP COLUMN IF EXISTS verified_at,
    DROP COLUMN IF EXISTS verified_by;
