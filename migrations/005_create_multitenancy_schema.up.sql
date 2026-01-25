-- Multi-Tenancy and Billing Schema
-- Migration: 005_create_multitenancy_schema
-- Description: Adds organization-level quotas, usage tracking, invitations, subscriptions, and billing

-- Add org_id to existing tables for data isolation
ALTER TABLE modules ADD COLUMN org_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE modules ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE modules ADD COLUMN owner_id BIGINT REFERENCES users(id);

-- Backfill existing modules to default organization
UPDATE modules SET org_id = (SELECT id FROM organizations WHERE name = 'default' LIMIT 1)
WHERE org_id IS NULL;

-- Make org_id NOT NULL after backfill
ALTER TABLE modules ALTER COLUMN org_id SET NOT NULL;

-- Add indexes for org-based queries
CREATE INDEX idx_modules_org_id ON modules(org_id);
CREATE INDEX idx_modules_public ON modules(is_public) WHERE is_public = true;
CREATE INDEX idx_modules_owner ON modules(owner_id) WHERE owner_id IS NOT NULL;

-- Organization settings and status
ALTER TABLE organizations ADD COLUMN slug VARCHAR(255) UNIQUE;
ALTER TABLE organizations ADD COLUMN owner_id BIGINT REFERENCES users(id);
ALTER TABLE organizations ADD COLUMN plan_tier VARCHAR(50) NOT NULL DEFAULT 'free';
ALTER TABLE organizations ADD COLUMN status VARCHAR(50) NOT NULL DEFAULT 'active';
ALTER TABLE organizations ADD COLUMN settings JSONB DEFAULT '{}';

-- Backfill slugs from names
UPDATE organizations SET slug = LOWER(REGEXP_REPLACE(name, '[^a-zA-Z0-9]+', '-', 'g'));
ALTER TABLE organizations ALTER COLUMN slug SET NOT NULL;

-- Organization quotas table
CREATE TABLE org_quotas (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    max_modules INT NOT NULL DEFAULT 5,
    max_versions_per_module INT NOT NULL DEFAULT 50,
    max_storage_bytes BIGINT NOT NULL DEFAULT 1073741824, -- 1GB
    max_compile_jobs_per_month INT NOT NULL DEFAULT 100,
    api_rate_limit_per_hour INT NOT NULL DEFAULT 1000,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Organization usage tracking (current period)
CREATE TABLE org_usage (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    modules_count INT NOT NULL DEFAULT 0,
    versions_count INT NOT NULL DEFAULT 0,
    storage_bytes BIGINT NOT NULL DEFAULT 0,
    compile_jobs_count INT NOT NULL DEFAULT 0,
    api_requests_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, period_start)
);

-- Organization invitations
CREATE TABLE org_invitations (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL, -- admin, developer, viewer
    token VARCHAR(255) NOT NULL UNIQUE,
    invited_by BIGINT NOT NULL REFERENCES users(id),
    invited_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    accepted_at TIMESTAMP WITH TIME ZONE,
    accepted_by BIGINT REFERENCES users(id),
    UNIQUE(org_id, email)
);

-- Subscriptions table (Stripe integration)
CREATE TABLE subscriptions (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    plan VARCHAR(50) NOT NULL, -- free, pro, enterprise, custom
    stripe_customer_id VARCHAR(255) UNIQUE,
    stripe_subscription_id VARCHAR(255) UNIQUE,
    status VARCHAR(50) NOT NULL, -- active, canceled, past_due, incomplete
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    cancel_at TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Invoices table
CREATE TABLE invoices (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invoice_number VARCHAR(255) UNIQUE,
    stripe_invoice_id VARCHAR(255) UNIQUE,
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'usd',
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(50) NOT NULL, -- draft, open, paid, void, uncollectible
    paid_at TIMESTAMP WITH TIME ZONE,
    due_date TIMESTAMP WITH TIME ZONE,
    invoice_url VARCHAR(512),
    invoice_pdf_url VARCHAR(512),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Payment methods table
CREATE TABLE payment_methods (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    stripe_payment_method_id VARCHAR(255) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL, -- card, bank_account
    is_default BOOLEAN NOT NULL DEFAULT false,
    card_brand VARCHAR(50),
    card_last4 VARCHAR(4),
    card_exp_month INT,
    card_exp_year INT,
    bank_name VARCHAR(255),
    bank_last4 VARCHAR(4),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for multi-tenancy
CREATE INDEX idx_org_quotas_org_id ON org_quotas(org_id);
CREATE INDEX idx_org_usage_org_id ON org_usage(org_id);
CREATE INDEX idx_org_usage_period ON org_usage(period_start, period_end);
CREATE INDEX idx_org_invitations_org_id ON org_invitations(org_id);
CREATE INDEX idx_org_invitations_email ON org_invitations(email);
CREATE INDEX idx_org_invitations_token ON org_invitations(token);
CREATE INDEX idx_org_invitations_expires ON org_invitations(expires_at);
CREATE INDEX idx_subscriptions_org_id ON subscriptions(org_id);
CREATE INDEX idx_subscriptions_stripe_customer ON subscriptions(stripe_customer_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_invoices_org_id ON invoices(org_id);
CREATE INDEX idx_invoices_stripe_id ON invoices(stripe_invoice_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_period ON invoices(period_start, period_end);
CREATE INDEX idx_payment_methods_org_id ON payment_methods(org_id);
CREATE INDEX idx_payment_methods_default ON payment_methods(is_default) WHERE is_default = true;

-- Triggers for updated_at
CREATE TRIGGER update_org_quotas_updated_at
    BEFORE UPDATE ON org_quotas
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_org_usage_updated_at
    BEFORE UPDATE ON org_usage
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_payment_methods_updated_at
    BEFORE UPDATE ON payment_methods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Default quotas for existing organizations
INSERT INTO org_quotas (org_id, max_modules, max_versions_per_module, max_storage_bytes, max_compile_jobs_per_month, api_rate_limit_per_hour)
SELECT id, 5, 50, 1073741824, 100, 1000
FROM organizations
ON CONFLICT (org_id) DO NOTHING;

-- Default subscription for existing organizations
INSERT INTO subscriptions (org_id, plan, status)
SELECT id, 'free', 'active'
FROM organizations
ON CONFLICT (org_id) DO NOTHING;

-- Initialize current usage period for existing organizations
INSERT INTO org_usage (org_id, period_start, period_end)
SELECT
    id,
    DATE_TRUNC('month', NOW()),
    DATE_TRUNC('month', NOW() + INTERVAL '1 month')
FROM organizations
ON CONFLICT (org_id, period_start) DO NOTHING;

-- Function to ensure only one default payment method per org
CREATE OR REPLACE FUNCTION ensure_one_default_payment_method()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_default = true THEN
        UPDATE payment_methods
        SET is_default = false
        WHERE org_id = NEW.org_id AND id != NEW.id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ensure_default_payment_method
    BEFORE INSERT OR UPDATE ON payment_methods
    FOR EACH ROW
    WHEN (NEW.is_default = true)
    EXECUTE FUNCTION ensure_one_default_payment_method();

-- Add org member role update trigger
ALTER TABLE organization_members ADD COLUMN invited_by BIGINT REFERENCES users(id);
ALTER TABLE organization_members ADD COLUMN joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();

-- Create view for active org members with user details
CREATE VIEW org_members_view AS
SELECT
    om.id,
    om.organization_id,
    om.user_id,
    om.role,
    om.invited_by,
    om.joined_at,
    om.created_at,
    u.username,
    u.email,
    u.full_name,
    u.is_bot,
    u.is_active as user_is_active
FROM organization_members om
JOIN users u ON om.user_id = u.id
WHERE u.is_active = true;
