-- Plugin Marketplace Schema Rollback
-- Migration 010: Drop plugin marketplace tables

DROP TABLE IF EXISTS plugin_tags;
DROP TABLE IF EXISTS plugin_dependencies;
DROP TABLE IF EXISTS plugin_stats_daily;
DROP TABLE IF EXISTS plugin_installations;
DROP TABLE IF EXISTS plugin_reviews;
DROP TABLE IF EXISTS plugin_versions;
DROP TABLE IF EXISTS plugins;
