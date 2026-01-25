-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,

    -- Actor information
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    username VARCHAR(255),
    organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL,
    token_id BIGINT REFERENCES api_tokens(id) ON DELETE SET NULL,

    -- Resource information
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    resource_name VARCHAR(255),

    -- Request context
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(100),
    method VARCHAR(10),
    path TEXT,
    status_code INTEGER,

    -- Additional details
    message TEXT,
    error_message TEXT,
    metadata JSONB,
    changes JSONB,

    -- Timestamp tracking
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for common query patterns
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_status ON audit_logs(status);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_logs_organization_id ON audit_logs(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id) WHERE resource_type IS NOT NULL;
CREATE INDEX idx_audit_logs_ip_address ON audit_logs(ip_address) WHERE ip_address IS NOT NULL;
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Create composite indexes for common filter combinations
CREATE INDEX idx_audit_logs_user_timestamp ON audit_logs(user_id, timestamp DESC) WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_logs_org_timestamp ON audit_logs(organization_id, timestamp DESC) WHERE organization_id IS NOT NULL;
CREATE INDEX idx_audit_logs_event_timestamp ON audit_logs(event_type, timestamp DESC);
CREATE INDEX idx_audit_logs_status_timestamp ON audit_logs(status, timestamp DESC);

-- Create GIN index for metadata and changes JSONB columns
CREATE INDEX idx_audit_logs_metadata ON audit_logs USING GIN(metadata) WHERE metadata IS NOT NULL;
CREATE INDEX idx_audit_logs_changes ON audit_logs USING GIN(changes) WHERE changes IS NOT NULL;

-- Add comment for documentation
COMMENT ON TABLE audit_logs IS 'Audit log for all security-relevant events and data mutations';
COMMENT ON COLUMN audit_logs.event_type IS 'Type of event (e.g., auth.login, data.module_create)';
COMMENT ON COLUMN audit_logs.status IS 'Event outcome: success, failure, or denied';
COMMENT ON COLUMN audit_logs.metadata IS 'Additional event-specific metadata as JSON';
COMMENT ON COLUMN audit_logs.changes IS 'Before/after values for update operations';
