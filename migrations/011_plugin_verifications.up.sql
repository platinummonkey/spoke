-- Migration 011: Plugin Verification System
-- Creates tables and views for plugin security validation and verification workflow

-- Plugin verification requests table
CREATE TABLE IF NOT EXISTS plugin_verifications (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, in_progress, approved, rejected, review_required
    security_level VARCHAR(50), -- verified (if approved)
    submitted_by VARCHAR(255),
    submitted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    verified_by VARCHAR(255),
    reason TEXT, -- Rejection or review reason
    notes TEXT, -- Additional notes from reviewer
    CONSTRAINT unique_plugin_verification UNIQUE (plugin_id, version),
    CONSTRAINT fk_plugin_verifications_plugin FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_verification_status ON plugin_verifications(status, submitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_verification_plugin ON plugin_verifications(plugin_id, version);

-- Validation errors table (manifest validation)
CREATE TABLE IF NOT EXISTS plugin_validation_errors (
    id BIGSERIAL PRIMARY KEY,
    verification_id BIGINT NOT NULL,
    field VARCHAR(255),
    message TEXT NOT NULL,
    severity VARCHAR(50) NOT NULL DEFAULT 'error', -- error, warning
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_plugin_validation_errors_verification FOREIGN KEY (verification_id) REFERENCES plugin_verifications(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_validation_verification ON plugin_validation_errors(verification_id);

-- Security issues table (security scan results)
CREATE TABLE IF NOT EXISTS plugin_security_issues (
    id BIGSERIAL PRIMARY KEY,
    verification_id BIGINT NOT NULL,
    severity VARCHAR(50) NOT NULL, -- critical, high, medium, low, warning
    category VARCHAR(100) NOT NULL, -- imports, hardcoded-secrets, sql-injection, etc.
    description TEXT NOT NULL,
    file VARCHAR(500),
    line_number INT,
    recommendation TEXT,
    cwe_id VARCHAR(50), -- Common Weakness Enumeration ID
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_plugin_security_issues_verification FOREIGN KEY (verification_id) REFERENCES plugin_verifications(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_security_verification ON plugin_security_issues(verification_id, severity);
CREATE INDEX IF NOT EXISTS idx_security_severity ON plugin_security_issues(severity, created_at DESC);

-- Plugin permission requests (track what permissions plugins request)
CREATE TABLE IF NOT EXISTS plugin_permissions (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    permission VARCHAR(255) NOT NULL,
    approved BOOLEAN NOT NULL DEFAULT false,
    approved_by VARCHAR(255),
    approved_at TIMESTAMP,
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_plugin_permissions_plugin FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_permissions_plugin ON plugin_permissions(plugin_id, version);
CREATE INDEX IF NOT EXISTS idx_permissions_status ON plugin_permissions(approved);

-- Verification audit log (track all verification actions)
CREATE TABLE IF NOT EXISTS plugin_verification_audit (
    id BIGSERIAL PRIMARY KEY,
    verification_id BIGINT NOT NULL,
    action VARCHAR(100) NOT NULL, -- submitted, started, approved, rejected, review_requested, comment_added
    actor VARCHAR(255) NOT NULL, -- User or system identifier
    details TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_plugin_verification_audit_verification FOREIGN KEY (verification_id) REFERENCES plugin_verifications(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_audit_verification ON plugin_verification_audit(verification_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON plugin_verification_audit(actor, created_at DESC);

-- Plugin scan history (track all security scans)
CREATE TABLE IF NOT EXISTS plugin_scan_history (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    scan_type VARCHAR(50) NOT NULL, -- gosec, manifest, dependency, etc.
    status VARCHAR(50) NOT NULL, -- completed, failed, in_progress
    issues_found INT NOT NULL DEFAULT 0,
    critical_issues INT NOT NULL DEFAULT 0,
    scan_duration_ms INT,
    error_message TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    CONSTRAINT fk_plugin_scan_history_plugin FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_scan_plugin ON plugin_scan_history(plugin_id, version, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_scan_status ON plugin_scan_history(status, started_at DESC);

-- View: Pending verifications (for verifier dashboard)
CREATE OR REPLACE VIEW pending_verifications AS
SELECT
    v.id,
    v.plugin_id,
    v.version,
    v.status,
    v.submitted_by,
    v.submitted_at,
    p.name AS plugin_name,
    p.type AS plugin_type,
    COUNT(DISTINCT ve.id) AS validation_errors,
    COUNT(DISTINCT si.id) FILTER (WHERE si.severity IN ('critical', 'high')) AS critical_issues,
    COUNT(DISTINCT si.id) AS total_issues
FROM plugin_verifications v
JOIN plugins p ON v.plugin_id = p.id
LEFT JOIN plugin_validation_errors ve ON v.id = ve.verification_id
LEFT JOIN plugin_security_issues si ON v.id = si.verification_id
WHERE v.status IN ('pending', 'review_required')
GROUP BY v.id, v.plugin_id, v.version, v.status, v.submitted_by, v.submitted_at, p.name, p.type
ORDER BY v.submitted_at ASC;

-- View: Recent verifications (for admin dashboard)
CREATE OR REPLACE VIEW recent_verifications AS
SELECT
    v.id,
    v.plugin_id,
    v.version,
    v.status,
    v.submitted_by,
    v.verified_by,
    v.submitted_at,
    v.completed_at,
    v.security_level,
    p.name AS plugin_name,
    p.type AS plugin_type,
    EXTRACT(EPOCH FROM (v.completed_at - v.started_at))::INTEGER AS processing_time_seconds,
    COUNT(DISTINCT si.id) FILTER (WHERE si.severity IN ('critical', 'high')) AS critical_issues
FROM plugin_verifications v
JOIN plugins p ON v.plugin_id = p.id
LEFT JOIN plugin_security_issues si ON v.id = si.verification_id
WHERE v.completed_at IS NOT NULL
GROUP BY v.id, v.plugin_id, v.version, v.status, v.submitted_by, v.verified_by,
         v.submitted_at, v.completed_at, v.security_level, p.name, p.type, v.started_at
ORDER BY v.completed_at DESC
LIMIT 100;

-- View: Plugin security scores (aggregate security metrics)
CREATE OR REPLACE VIEW plugin_security_scores AS
SELECT
    p.id AS plugin_id,
    p.name AS plugin_name,
    p.security_level,
    COUNT(DISTINCT v.id) AS total_verifications,
    COUNT(DISTINCT v.id) FILTER (WHERE v.status = 'approved') AS approved_verifications,
    COUNT(DISTINCT v.id) FILTER (WHERE v.status = 'rejected') AS rejected_verifications,
    AVG(CASE WHEN v.status = 'approved' THEN 100 ELSE 0 END) AS approval_rate,
    COUNT(DISTINCT si.id) AS total_security_issues,
    COUNT(DISTINCT si.id) FILTER (WHERE si.severity = 'critical') AS critical_issues,
    COUNT(DISTINCT si.id) FILTER (WHERE si.severity = 'high') AS high_issues,
    MAX(v.completed_at) AS last_verified_at
FROM plugins p
LEFT JOIN plugin_verifications v ON p.id = v.plugin_id
LEFT JOIN plugin_security_issues si ON v.id = si.verification_id
GROUP BY p.id, p.name, p.security_level;

-- Trigger: Update plugin security level on verification approval
CREATE OR REPLACE FUNCTION update_plugin_security_level_on_approval()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'approved' AND OLD.status != 'approved' AND NEW.security_level IS NOT NULL THEN
        UPDATE plugins
        SET
            security_level = NEW.security_level,
            verified_at = NEW.completed_at,
            verified_by = NEW.verified_by
        WHERE id = NEW.plugin_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_plugin_security_level_on_approval
AFTER UPDATE ON plugin_verifications
FOR EACH ROW
EXECUTE FUNCTION update_plugin_security_level_on_approval();

-- Trigger: Record audit log entry on verification status change
CREATE OR REPLACE FUNCTION record_verification_audit_on_update()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status != OLD.status THEN
        INSERT INTO plugin_verification_audit (verification_id, action, actor, details)
        VALUES (
            NEW.id,
            'status_changed_to_' || NEW.status,
            COALESCE(NEW.verified_by, 'system'),
            'Status changed from ' || OLD.status || ' to ' || NEW.status
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER record_verification_audit_on_update
AFTER UPDATE ON plugin_verifications
FOR EACH ROW
EXECUTE FUNCTION record_verification_audit_on_update();

-- Index optimizations for common queries
CREATE INDEX IF NOT EXISTS idx_verifications_completed ON plugin_verifications(completed_at DESC) WHERE completed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_security_issues_critical ON plugin_security_issues(severity, verification_id) WHERE severity IN ('critical', 'high');
CREATE INDEX IF NOT EXISTS idx_scan_recent ON plugin_scan_history(plugin_id, completed_at DESC) WHERE completed_at IS NOT NULL;

-- Insert default allowed permissions
INSERT INTO plugin_permissions (plugin_id, version, permission, approved, approved_by, approved_at, reason)
SELECT DISTINCT
    p.id,
    '*', -- Applies to all versions
    perm.permission,
    true,
    'system',
    CURRENT_TIMESTAMP,
    'Default safe permission'
FROM plugins p
CROSS JOIN (
    SELECT 'filesystem:read' AS permission
    UNION ALL SELECT 'filesystem:write'
    UNION ALL SELECT 'network:read'
    UNION ALL SELECT 'process:exec'
    UNION ALL SELECT 'env:read'
) perm
WHERE p.security_level IN ('official', 'verified')
ON CONFLICT DO NOTHING;

-- Comments for documentation
COMMENT ON TABLE plugin_verifications IS 'Tracks plugin verification requests and their approval workflow';
COMMENT ON TABLE plugin_validation_errors IS 'Stores manifest validation errors found during verification';
COMMENT ON TABLE plugin_security_issues IS 'Stores security issues discovered by automated scans (gosec, etc.)';
COMMENT ON TABLE plugin_permissions IS 'Tracks plugin permission requests and approvals';
COMMENT ON TABLE plugin_verification_audit IS 'Audit log for all verification-related actions';
COMMENT ON TABLE plugin_scan_history IS 'Historical record of all security scans performed on plugins';
