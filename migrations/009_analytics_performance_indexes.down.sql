-- Rollback Migration 009: Drop analytics performance indexes

DROP INDEX IF EXISTS idx_download_events_analytics_cover;
DROP INDEX IF EXISTS idx_module_view_events_analytics_cover;
DROP INDEX IF EXISTS idx_compilation_events_analytics_cover;
DROP INDEX IF EXISTS idx_download_events_time_module_lang;
DROP INDEX IF EXISTS idx_download_events_user_activity;
DROP INDEX IF EXISTS idx_download_events_org_activity;
DROP INDEX IF EXISTS idx_compilation_events_failures;
DROP INDEX IF EXISTS idx_compilation_events_slow;
DROP INDEX IF EXISTS idx_download_events_recent_90d;
DROP INDEX IF EXISTS idx_module_view_events_recent_90d;
DROP INDEX IF EXISTS idx_compilation_events_cache_stats;
DROP INDEX IF EXISTS idx_module_stats_daily_popular;
DROP INDEX IF EXISTS idx_language_stats_daily_performance;
DROP INDEX IF EXISTS idx_module_stats_daily_trending;
