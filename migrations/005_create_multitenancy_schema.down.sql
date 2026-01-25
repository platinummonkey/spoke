-- Rollback Multi-Tenancy Schema (NO BILLING)
-- Migration: 005_create_multitenancy_schema

-- Drop views
DROP VIEW IF EXISTS org_members_view;

-- Drop triggers
DROP TRIGGER IF EXISTS update_org_usage_updated_at ON org_usage;
DROP TRIGGER IF EXISTS update_org_quotas_updated_at ON org_quotas;

-- Drop new tables
DROP TABLE IF EXISTS org_invitations;
DROP TABLE IF EXISTS org_usage;
DROP TABLE IF EXISTS org_quotas;

-- Remove columns from organization_members
ALTER TABLE organization_members DROP COLUMN IF EXISTS invited_by;
ALTER TABLE organization_members DROP COLUMN IF EXISTS joined_at;

-- Remove columns from organizations
ALTER TABLE organizations DROP COLUMN IF EXISTS settings;
ALTER TABLE organizations DROP COLUMN IF EXISTS status;
ALTER TABLE organizations DROP COLUMN IF EXISTS quota_tier;
ALTER TABLE organizations DROP COLUMN IF EXISTS owner_id;
ALTER TABLE organizations DROP COLUMN IF EXISTS slug;

-- Remove columns from modules
DROP INDEX IF EXISTS idx_modules_owner;
DROP INDEX IF EXISTS idx_modules_public;
DROP INDEX IF EXISTS idx_modules_org_id;

ALTER TABLE modules DROP COLUMN IF EXISTS owner_id;
ALTER TABLE modules DROP COLUMN IF EXISTS is_public;
ALTER TABLE modules DROP COLUMN IF EXISTS org_id;
