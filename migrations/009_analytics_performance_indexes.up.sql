-- Migration 009: Analytics Performance Indexes
-- Add covering indexes and optimizations for fast analytics queries

-- Covering index for download analytics queries (avoid table lookups)
CREATE INDEX IF NOT EXISTS idx_download_events_analytics_cover
ON download_events(module_name, downloaded_at DESC)
INCLUDE (user_id, organization_id, language, file_size, success);

-- Covering index for view analytics queries
CREATE INDEX IF NOT EXISTS idx_module_view_events_analytics_cover
ON module_view_events(module_name, viewed_at DESC)
INCLUDE (user_id, organization_id, source);

-- Covering index for compilation analytics queries
CREATE INDEX IF NOT EXISTS idx_compilation_events_analytics_cover
ON compilation_events(language, started_at DESC)
INCLUDE (module_name, version, duration_ms, success, cache_hit);

-- Composite index for time-range + module queries (common pattern)
CREATE INDEX IF NOT EXISTS idx_download_events_time_module_lang
ON download_events(downloaded_at DESC, module_name, language);

-- Index for user activity queries
CREATE INDEX IF NOT EXISTS idx_download_events_user_activity
ON download_events(user_id, downloaded_at DESC)
WHERE user_id IS NOT NULL;

-- Index for organization activity queries
CREATE INDEX IF NOT EXISTS idx_download_events_org_activity
ON download_events(organization_id, downloaded_at DESC)
WHERE organization_id IS NOT NULL;

-- Index for failed compilations (for alerting)
CREATE INDEX IF NOT EXISTS idx_compilation_events_failures
ON compilation_events(language, started_at DESC)
WHERE success = false;

-- Index for slow compilations (for alerting)
CREATE INDEX IF NOT EXISTS idx_compilation_events_slow
ON compilation_events(language, started_at DESC, duration_ms DESC)
WHERE duration_ms > 5000;

-- Partial index for recent downloads (last 90 days, for health scoring)
CREATE INDEX IF NOT EXISTS idx_download_events_recent_90d
ON download_events(module_name, language, downloaded_at DESC)
WHERE downloaded_at >= CURRENT_DATE - INTERVAL '90 days';

-- Partial index for recent views (last 90 days)
CREATE INDEX IF NOT EXISTS idx_module_view_events_recent_90d
ON module_view_events(module_name, viewed_at DESC)
WHERE viewed_at >= CURRENT_DATE - INTERVAL '90 days';

-- Index for cache hit rate calculations
CREATE INDEX IF NOT EXISTS idx_compilation_events_cache_stats
ON compilation_events(language, started_at DESC, cache_hit);

-- Optimize aggregation table queries
CREATE INDEX IF NOT EXISTS idx_module_stats_daily_popular
ON module_stats_daily(date DESC, download_count DESC)
INCLUDE (module_name, view_count, unique_users);

CREATE INDEX IF NOT EXISTS idx_language_stats_daily_performance
ON language_stats_daily(date DESC, language)
INCLUDE (avg_duration_ms, p95_duration_ms, compilation_count);

-- Optimize trending queries (growth rate calculation)
CREATE INDEX IF NOT EXISTS idx_module_stats_daily_trending
ON module_stats_daily(module_name, date DESC)
INCLUDE (download_count)
WHERE date >= CURRENT_DATE - INTERVAL '14 days';

-- VACUUM ANALYZE to update statistics after index creation
VACUUM ANALYZE download_events;
VACUUM ANALYZE module_view_events;
VACUUM ANALYZE compilation_events;
VACUUM ANALYZE module_stats_daily;
VACUUM ANALYZE language_stats_daily;
