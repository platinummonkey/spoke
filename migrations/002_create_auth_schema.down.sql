-- Rollback Authentication and Authorization Schema
-- Migration: 002_create_auth_schema

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS ip_allowlist;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS rate_limits;
DROP TABLE IF EXISTS module_permissions;
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
