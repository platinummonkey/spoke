-- Authentication and Authorization Schema
-- Migration: 002_create_auth_schema

-- Users table (both human users and bot users)
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) UNIQUE,
    full_name VARCHAR(255),
    is_bot BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'
);

-- Organizations table
CREATE TABLE organizations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

-- Organization memberships
CREATE TABLE organization_members (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- admin, developer, viewer
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, user_id)
);

-- API tokens (both user tokens and bot tokens)
CREATE TABLE api_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE, -- SHA256 of token
    token_prefix VARCHAR(20) NOT NULL, -- First 8 chars for identification
    name VARCHAR(255) NOT NULL, -- User-friendly name for the token
    description TEXT,
    scopes TEXT[] NOT NULL DEFAULT '{}', -- Array of permission scopes
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by BIGINT REFERENCES users(id),
    revoke_reason TEXT
);

-- Module-level permissions
CREATE TABLE module_permissions (
    id BIGSERIAL PRIMARY KEY,
    module_id BIGINT NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE,
    permission VARCHAR(50) NOT NULL, -- read, write, delete, admin
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    granted_by BIGINT REFERENCES users(id),
    CONSTRAINT module_perm_check CHECK (
        (user_id IS NOT NULL AND organization_id IS NULL) OR
        (user_id IS NULL AND organization_id IS NOT NULL)
    ),
    UNIQUE(module_id, user_id, permission),
    UNIQUE(module_id, organization_id, permission)
);

-- Rate limiting buckets
CREATE TABLE rate_limits (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE,
    endpoint VARCHAR(255) NOT NULL,
    requests_count INT NOT NULL DEFAULT 0,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    window_duration_seconds INT NOT NULL,
    CONSTRAINT rate_limit_check CHECK (
        (user_id IS NOT NULL AND organization_id IS NULL) OR
        (user_id IS NULL AND organization_id IS NOT NULL)
    )
);

-- Audit log for security events
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id),
    organization_id BIGINT REFERENCES organizations(id),
    action VARCHAR(100) NOT NULL, -- e.g., "module.create", "version.push", "token.create"
    resource_type VARCHAR(50) NOT NULL, -- module, version, token, user
    resource_id VARCHAR(255), -- ID of the affected resource
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(20) NOT NULL, -- success, failure, denied
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- IP allowlist for enterprise security
CREATE TABLE ip_allowlist (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    ip_cidr CIDR NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES users(id)
);

-- Indexes for performance
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_is_bot ON users(is_bot);
CREATE INDEX idx_users_is_active ON users(is_active);

CREATE INDEX idx_org_members_org ON organization_members(organization_id);
CREATE INDEX idx_org_members_user ON organization_members(user_id);
CREATE INDEX idx_org_members_role ON organization_members(role);

CREATE INDEX idx_api_tokens_user ON api_tokens(user_id);
CREATE INDEX idx_api_tokens_hash ON api_tokens(token_hash);
CREATE INDEX idx_api_tokens_prefix ON api_tokens(token_prefix);
CREATE INDEX idx_api_tokens_expires ON api_tokens(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_api_tokens_active ON api_tokens(revoked_at) WHERE revoked_at IS NULL;

CREATE INDEX idx_module_perms_module ON module_permissions(module_id);
CREATE INDEX idx_module_perms_user ON module_permissions(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_module_perms_org ON module_permissions(organization_id) WHERE organization_id IS NOT NULL;

CREATE INDEX idx_rate_limits_user ON rate_limits(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_rate_limits_org ON rate_limits(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX idx_rate_limits_endpoint ON rate_limits(endpoint);
CREATE INDEX idx_rate_limits_window ON rate_limits(window_start);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_logs_org ON audit_logs(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_status ON audit_logs(status);

CREATE INDEX idx_ip_allowlist_org ON ip_allowlist(organization_id);

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Default admin user (password should be changed on first login)
INSERT INTO users (username, email, full_name, is_bot, is_active)
VALUES ('admin', 'admin@spoke.local', 'System Administrator', false, true);

-- Default organization
INSERT INTO organizations (name, display_name, description)
VALUES ('default', 'Default Organization', 'Default organization for Spoke registry');

-- Make admin user an admin of default org
INSERT INTO organization_members (organization_id, user_id, role)
VALUES (
    (SELECT id FROM organizations WHERE name = 'default'),
    (SELECT id FROM users WHERE username = 'admin'),
    'admin'
);
