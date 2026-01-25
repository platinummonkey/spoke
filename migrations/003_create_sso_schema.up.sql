-- Create SSO providers table
CREATE TABLE IF NOT EXISTS sso_providers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    provider_type VARCHAR(50) NOT NULL, -- saml, oauth2, oidc
    provider_name VARCHAR(100) NOT NULL, -- azuread, okta, google, generic
    enabled BOOLEAN NOT NULL DEFAULT true,
    auto_provision BOOLEAN NOT NULL DEFAULT true,
    default_role VARCHAR(50), -- Default role for new users
    saml_config JSONB, -- SAML configuration
    oauth2_config JSONB, -- OAuth2 configuration
    oidc_config JSONB, -- OIDC configuration
    group_mapping JSONB, -- Group to role mappings
    attribute_mapping JSONB NOT NULL, -- Attribute mappings
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_provider_type CHECK (provider_type IN ('saml', 'oauth2', 'oidc'))
);

CREATE INDEX idx_sso_providers_enabled ON sso_providers(enabled);
CREATE INDEX idx_sso_providers_provider_type ON sso_providers(provider_type);

-- Create SSO user mappings table
CREATE TABLE IF NOT EXISTS sso_user_mappings (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES sso_providers(id) ON DELETE CASCADE,
    external_user_id VARCHAR(255) NOT NULL, -- User ID from SSO provider
    internal_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_login_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_sso_user_mapping UNIQUE (provider_id, external_user_id)
);

CREATE INDEX idx_sso_user_mappings_provider ON sso_user_mappings(provider_id);
CREATE INDEX idx_sso_user_mappings_internal_user ON sso_user_mappings(internal_user_id);
CREATE INDEX idx_sso_user_mappings_external_user ON sso_user_mappings(external_user_id);

-- Create SSO sessions table
CREATE TABLE IF NOT EXISTS sso_sessions (
    id VARCHAR(255) PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES sso_providers(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_user_id VARCHAR(255) NOT NULL,
    saml_session_index VARCHAR(255), -- For SAML logout
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,

    CONSTRAINT valid_expiry CHECK (expires_at > created_at)
);

CREATE INDEX idx_sso_sessions_user ON sso_sessions(user_id);
CREATE INDEX idx_sso_sessions_provider ON sso_sessions(provider_id);
CREATE INDEX idx_sso_sessions_expires ON sso_sessions(expires_at);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_sso_providers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sso_providers_updated_at
    BEFORE UPDATE ON sso_providers
    FOR EACH ROW
    EXECUTE FUNCTION update_sso_providers_updated_at();

CREATE OR REPLACE FUNCTION update_sso_user_mappings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sso_user_mappings_updated_at
    BEFORE UPDATE ON sso_user_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_sso_user_mappings_updated_at();

-- Comments for documentation
COMMENT ON TABLE sso_providers IS 'SSO identity provider configurations';
COMMENT ON TABLE sso_user_mappings IS 'Mappings between SSO external user IDs and internal Spoke user IDs';
COMMENT ON TABLE sso_sessions IS 'Active SSO sessions for logged-in users';

COMMENT ON COLUMN sso_providers.provider_type IS 'Type of SSO provider: saml, oauth2, or oidc';
COMMENT ON COLUMN sso_providers.auto_provision IS 'Whether to automatically create users on first login (JIT provisioning)';
COMMENT ON COLUMN sso_providers.group_mapping IS 'JSON mapping of SSO groups to Spoke roles';
COMMENT ON COLUMN sso_user_mappings.external_user_id IS 'User identifier from SSO provider (e.g., sub claim, NameID)';
COMMENT ON COLUMN sso_sessions.saml_session_index IS 'SAML SessionIndex for logout requests';
