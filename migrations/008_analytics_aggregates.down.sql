-- Rollback Migration 008: Analytics Aggregates

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS trending_modules;
DROP MATERIALIZED VIEW IF EXISTS top_modules_30d;

-- Drop aggregate tables
DROP TABLE IF EXISTS org_stats_daily;
DROP TABLE IF EXISTS language_stats_daily;
DROP TABLE IF EXISTS module_stats_monthly;
DROP TABLE IF EXISTS module_stats_weekly;
DROP TABLE IF EXISTS module_stats_daily;
