-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_changes;
DROP INDEX IF EXISTS idx_audit_logs_metadata;
DROP INDEX IF EXISTS idx_audit_logs_status_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_event_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_org_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_user_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_ip_address;
DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_audit_logs_organization_id;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP INDEX IF EXISTS idx_audit_logs_status;
DROP INDEX IF EXISTS idx_audit_logs_event_type;
DROP INDEX IF EXISTS idx_audit_logs_timestamp;

-- Drop table
DROP TABLE IF EXISTS audit_logs;
