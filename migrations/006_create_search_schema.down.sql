-- Rollback for Advanced Search and Dependency Visualization Schema
-- Migration: 006_create_search_schema
-- Description: Drops all tables, views, and functions created in the up migration

-- Drop views first (they depend on tables)
DROP VIEW IF EXISTS recent_searches;
DROP MATERIALIZED VIEW IF EXISTS popular_entities;
DROP MATERIALIZED VIEW IF EXISTS search_suggestions;

-- Drop functions
DROP FUNCTION IF EXISTS refresh_popular_entities();
DROP FUNCTION IF EXISTS refresh_search_suggestions();
DROP FUNCTION IF EXISTS proto_search_index_trigger();

-- Drop tables (CASCADE will handle any remaining dependencies)
DROP TABLE IF EXISTS bookmarks CASCADE;
DROP TABLE IF EXISTS search_history CASCADE;
DROP TABLE IF EXISTS saved_searches CASCADE;
DROP TABLE IF EXISTS proto_search_index CASCADE;
