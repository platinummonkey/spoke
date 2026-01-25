-- Migration 008: Analytics Aggregates
-- Purpose: Pre-computed aggregation tables for fast dashboard queries

-- Daily aggregates for modules
CREATE TABLE module_stats_daily (
    id BIGSERIAL PRIMARY KEY,
    module_name VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    view_count BIGINT NOT NULL DEFAULT 0,
    download_count BIGINT NOT NULL DEFAULT 0,
    unique_users BIGINT NOT NULL DEFAULT 0,
    unique_orgs BIGINT NOT NULL DEFAULT 0,
    compilation_count BIGINT NOT NULL DEFAULT 0,
    compilation_success_count BIGINT NOT NULL DEFAULT 0,
    total_download_bytes BIGINT NOT NULL DEFAULT 0,
    avg_compilation_duration_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(module_name, date)
);

CREATE INDEX idx_module_stats_daily_date ON module_stats_daily(date DESC);
CREATE INDEX idx_module_stats_daily_module ON module_stats_daily(module_name, date DESC);
CREATE INDEX idx_module_stats_daily_downloads ON module_stats_daily(download_count DESC, date DESC);

-- Weekly aggregates
CREATE TABLE module_stats_weekly (
    id BIGSERIAL PRIMARY KEY,
    module_name VARCHAR(255) NOT NULL,
    week_start DATE NOT NULL,
    week_end DATE NOT NULL,
    view_count BIGINT NOT NULL DEFAULT 0,
    download_count BIGINT NOT NULL DEFAULT 0,
    unique_users BIGINT NOT NULL DEFAULT 0,
    unique_orgs BIGINT NOT NULL DEFAULT 0,
    compilation_count BIGINT NOT NULL DEFAULT 0,
    compilation_success_count BIGINT NOT NULL DEFAULT 0,
    total_download_bytes BIGINT NOT NULL DEFAULT 0,
    avg_compilation_duration_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(module_name, week_start)
);

CREATE INDEX idx_module_stats_weekly_date ON module_stats_weekly(week_start DESC);
CREATE INDEX idx_module_stats_weekly_module ON module_stats_weekly(module_name, week_start DESC);

-- Monthly aggregates
CREATE TABLE module_stats_monthly (
    id BIGSERIAL PRIMARY KEY,
    module_name VARCHAR(255) NOT NULL,
    month DATE NOT NULL, -- First day of month
    view_count BIGINT NOT NULL DEFAULT 0,
    download_count BIGINT NOT NULL DEFAULT 0,
    unique_users BIGINT NOT NULL DEFAULT 0,
    unique_orgs BIGINT NOT NULL DEFAULT 0,
    compilation_count BIGINT NOT NULL DEFAULT 0,
    compilation_success_count BIGINT NOT NULL DEFAULT 0,
    total_download_bytes BIGINT NOT NULL DEFAULT 0,
    avg_compilation_duration_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(module_name, month)
);

CREATE INDEX idx_module_stats_monthly_date ON module_stats_monthly(month DESC);
CREATE INDEX idx_module_stats_monthly_module ON module_stats_monthly(module_name, month DESC);

-- Language-specific compilation stats (daily)
CREATE TABLE language_stats_daily (
    id BIGSERIAL PRIMARY KEY,
    language VARCHAR(50) NOT NULL,
    date DATE NOT NULL,
    compilation_count BIGINT NOT NULL DEFAULT 0,
    compilation_success_count BIGINT NOT NULL DEFAULT 0,
    compilation_failure_count BIGINT NOT NULL DEFAULT 0,
    avg_duration_ms INTEGER,
    p50_duration_ms INTEGER,
    p95_duration_ms INTEGER,
    p99_duration_ms INTEGER,
    total_output_bytes BIGINT NOT NULL DEFAULT 0,
    cache_hit_count BIGINT NOT NULL DEFAULT 0,
    cache_miss_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(language, date)
);

CREATE INDEX idx_language_stats_daily_date ON language_stats_daily(date DESC);
CREATE INDEX idx_language_stats_daily_language ON language_stats_daily(language, date DESC);

-- Organization-level daily stats
CREATE TABLE org_stats_daily (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    date DATE NOT NULL,
    api_requests BIGINT NOT NULL DEFAULT 0,
    downloads BIGINT NOT NULL DEFAULT 0,
    compilations BIGINT NOT NULL DEFAULT 0,
    storage_bytes BIGINT NOT NULL DEFAULT 0,
    active_users BIGINT NOT NULL DEFAULT 0,
    modules_created BIGINT NOT NULL DEFAULT 0,
    versions_created BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, date)
);

CREATE INDEX idx_org_stats_daily_date ON org_stats_daily(date DESC);
CREATE INDEX idx_org_stats_daily_org ON org_stats_daily(organization_id, date DESC);

-- Materialized view for top modules (last 30 days)
CREATE MATERIALIZED VIEW top_modules_30d AS
SELECT
    module_name,
    SUM(view_count) AS total_views,
    SUM(download_count) AS total_downloads,
    COUNT(DISTINCT date) AS active_days,
    AVG(download_count) AS avg_daily_downloads
FROM module_stats_daily
WHERE date >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY module_name
ORDER BY total_downloads DESC
LIMIT 100;

CREATE UNIQUE INDEX idx_top_modules_30d ON top_modules_30d(module_name);

-- Materialized view for trending modules (last 7 days vs previous 7)
CREATE MATERIALIZED VIEW trending_modules AS
WITH current_week AS (
    SELECT module_name, SUM(download_count) AS downloads
    FROM module_stats_daily
    WHERE date >= CURRENT_DATE - INTERVAL '7 days'
    GROUP BY module_name
),
previous_week AS (
    SELECT module_name, SUM(download_count) AS downloads
    FROM module_stats_daily
    WHERE date >= CURRENT_DATE - INTERVAL '14 days'
      AND date < CURRENT_DATE - INTERVAL '7 days'
    GROUP BY module_name
)
SELECT
    c.module_name,
    c.downloads AS current_downloads,
    COALESCE(p.downloads, 0) AS previous_downloads,
    ((c.downloads - COALESCE(p.downloads, 0))::float / NULLIF(COALESCE(p.downloads, 1), 0)) AS growth_rate
FROM current_week c
LEFT JOIN previous_week p ON c.module_name = p.module_name
WHERE c.downloads > 10
ORDER BY growth_rate DESC
LIMIT 50;

CREATE UNIQUE INDEX idx_trending_modules ON trending_modules(module_name);

-- Comments
COMMENT ON TABLE module_stats_daily IS 'Daily aggregated statistics per module';
COMMENT ON TABLE module_stats_weekly IS 'Weekly aggregated statistics per module';
COMMENT ON TABLE module_stats_monthly IS 'Monthly aggregated statistics per module';
COMMENT ON TABLE language_stats_daily IS 'Daily compilation performance statistics per language';
COMMENT ON TABLE org_stats_daily IS 'Daily usage statistics per organization';
COMMENT ON MATERIALIZED VIEW top_modules_30d IS 'Top 100 modules by downloads in last 30 days';
COMMENT ON MATERIALIZED VIEW trending_modules IS 'Top 50 trending modules by growth rate (7d vs previous 7d)';
