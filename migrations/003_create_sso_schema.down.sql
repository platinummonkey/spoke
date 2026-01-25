-- Drop SSO tables in reverse order
DROP TABLE IF EXISTS sso_sessions;
DROP TABLE IF EXISTS sso_user_mappings;
DROP TABLE IF EXISTS sso_providers;

-- Drop triggers and functions
DROP TRIGGER IF EXISTS sso_user_mappings_updated_at ON sso_user_mappings;
DROP TRIGGER IF EXISTS sso_providers_updated_at ON sso_providers;
DROP FUNCTION IF EXISTS update_sso_user_mappings_updated_at();
DROP FUNCTION IF EXISTS update_sso_providers_updated_at();
