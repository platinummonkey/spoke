-- Rollback base schema
-- Migration: 001_create_base_schema

DROP VIEW IF EXISTS modules_with_latest;
DROP TRIGGER IF EXISTS update_modules_updated_at ON modules;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS compiled_artifacts CASCADE;
DROP TABLE IF EXISTS proto_files CASCADE;
DROP TABLE IF EXISTS versions CASCADE;
DROP TABLE IF EXISTS modules CASCADE;
