-- Migration 011: Plugin Verification System
-- Creates tables and views for plugin security validation and verification workflow

-- Plugin verification requests table
CREATE TABLE IF NOT EXISTS plugin_verifications (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
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
    UNIQUE KEY unique_plugin_verification (plugin_id, version),
    INDEX idx_verification_status (status, submitted_at DESC),
    INDEX idx_verification_plugin (plugin_id, version),
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Validation errors table (manifest validation)
CREATE TABLE IF NOT EXISTS plugin_validation_errors (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    verification_id BIGINT NOT NULL,
    field VARCHAR(255),
    message TEXT NOT NULL,
    severity VARCHAR(50) NOT NULL DEFAULT 'error', -- error, warning
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_validation_verification (verification_id),
    FOREIGN KEY (verification_id) REFERENCES plugin_verifications(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Security issues table (security scan results)
CREATE TABLE IF NOT EXISTS plugin_security_issues (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    verification_id BIGINT NOT NULL,
    severity VARCHAR(50) NOT NULL, -- critical, high, medium, low, warning
    category VARCHAR(100) NOT NULL, -- imports, hardcoded-secrets, sql-injection, etc.
    description TEXT NOT NULL,
    file VARCHAR(500),
    line_number INT,
    recommendation TEXT,
    cwe_id VARCHAR(50), -- Common Weakness Enumeration ID
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_security_verification (verification_id, severity),
    INDEX idx_security_severity (severity, created_at DESC),
    FOREIGN KEY (verification_id) REFERENCES plugin_verifications(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Plugin permission requests (track what permissions plugins request)
CREATE TABLE IF NOT EXISTS plugin_permissions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    permission VARCHAR(255) NOT NULL,
    approved BOOLEAN NOT NULL DEFAULT false,
    approved_by VARCHAR(255),
    approved_at TIMESTAMP,
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_permissions_plugin (plugin_id, version),
    INDEX idx_permissions_status (approved),
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Verification audit log (track all verification actions)
CREATE TABLE IF NOT EXISTS plugin_verification_audit (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    verification_id BIGINT NOT NULL,
    action VARCHAR(100) NOT NULL, -- submitted, started, approved, rejected, review_requested, comment_added
    actor VARCHAR(255) NOT NULL, -- User or system identifier
    details TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_audit_verification (verification_id, created_at DESC),
    INDEX idx_audit_actor (actor, created_at DESC),
    FOREIGN KEY (verification_id) REFERENCES plugin_verifications(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Plugin scan history (track all security scans)
CREATE TABLE IF NOT EXISTS plugin_scan_history (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
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
    INDEX idx_scan_plugin (plugin_id, version, started_at DESC),
    INDEX idx_scan_status (status, started_at DESC),
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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
    TIMESTAMPDIFF(SECOND, v.started_at, v.completed_at) AS processing_time_seconds,
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
DELIMITER //
CREATE TRIGGER update_plugin_security_level_on_approval
AFTER UPDATE ON plugin_verifications
FOR EACH ROW
BEGIN
    IF NEW.status = 'approved' AND OLD.status != 'approved' AND NEW.security_level IS NOT NULL THEN
        UPDATE plugins
        SET
            security_level = NEW.security_level,
            verified_at = NEW.completed_at,
            verified_by = NEW.verified_by
        WHERE id = NEW.plugin_id;
    END IF;
END//
DELIMITER ;

-- Trigger: Record audit log entry on verification status change
DELIMITER //
CREATE TRIGGER record_verification_audit_on_update
AFTER UPDATE ON plugin_verifications
FOR EACH ROW
BEGIN
    IF NEW.status != OLD.status THEN
        INSERT INTO plugin_verification_audit (verification_id, action, actor, details)
        VALUES (
            NEW.id,
            CONCAT('status_changed_to_', NEW.status),
            COALESCE(NEW.verified_by, 'system'),
            CONCAT('Status changed from ', OLD.status, ' to ', NEW.status)
        );
    END IF;
END//
DELIMITER ;

-- Index optimizations for common queries
CREATE INDEX idx_verifications_completed ON plugin_verifications(completed_at DESC) WHERE completed_at IS NOT NULL;
CREATE INDEX idx_security_issues_critical ON plugin_security_issues(severity, verification_id) WHERE severity IN ('critical', 'high');
CREATE INDEX idx_scan_recent ON plugin_scan_history(plugin_id, completed_at DESC) WHERE completed_at IS NOT NULL;

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
ON DUPLICATE KEY UPDATE approved = true;

-- Comments for documentation
ALTER TABLE plugin_verifications
    COMMENT 'Tracks plugin verification requests and their approval workflow';
ALTER TABLE plugin_validation_errors
    COMMENT 'Stores manifest validation errors found during verification';
ALTER TABLE plugin_security_issues
    COMMENT 'Stores security issues discovered by automated scans (gosec, etc.)';
ALTER TABLE plugin_permissions
    COMMENT 'Tracks plugin permission requests and approvals';
ALTER TABLE plugin_verification_audit
    COMMENT 'Audit log for all verification-related actions';
ALTER TABLE plugin_scan_history
    COMMENT 'Historical record of all security scans performed on plugins';
