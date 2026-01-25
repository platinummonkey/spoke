-- Migration 007: Analytics Events
-- Purpose: Track download events, module views, and compilation events for analytics

-- Download event tracking (partitioned by time)
CREATE TABLE download_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id),
    organization_id BIGINT REFERENCES organizations(id),
    module_name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    language VARCHAR(50) NOT NULL,
    downloaded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    file_size BIGINT NOT NULL,
    duration_ms INTEGER,
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    ip_address INET,
    user_agent TEXT,
    client_sdk VARCHAR(100),
    client_version VARCHAR(50),
    cache_hit BOOLEAN DEFAULT false
) PARTITION BY RANGE (downloaded_at);

-- Create monthly partitions for download_events (current and next 2 months)
CREATE TABLE download_events_2026_01 PARTITION OF download_events
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE download_events_2026_02 PARTITION OF download_events
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE download_events_2026_03 PARTITION OF download_events
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

-- Indexes for download_events
CREATE INDEX idx_download_events_module ON download_events(module_name, version);
CREATE INDEX idx_download_events_user ON download_events(user_id, downloaded_at DESC);
CREATE INDEX idx_download_events_org ON download_events(organization_id, downloaded_at DESC);
CREATE INDEX idx_download_events_language ON download_events(language, downloaded_at DESC);
CREATE INDEX idx_download_events_time ON download_events(downloaded_at DESC);

-- Covering index for analytics queries
CREATE INDEX idx_download_events_analytics_cover ON download_events(module_name, downloaded_at DESC)
    INCLUDE (user_id, organization_id, language, file_size, success);

-- Module view tracking (partitioned by time)
CREATE TABLE module_view_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id),
    organization_id BIGINT REFERENCES organizations(id),
    module_name VARCHAR(255) NOT NULL,
    version VARCHAR(50),
    viewed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    source VARCHAR(50) NOT NULL, -- 'web', 'api', 'cli'
    page_type VARCHAR(50), -- 'list', 'detail', 'search'
    referrer TEXT,
    ip_address INET,
    user_agent TEXT
) PARTITION BY RANGE (viewed_at);

-- Create monthly partitions for module_view_events
CREATE TABLE module_view_events_2026_01 PARTITION OF module_view_events
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE module_view_events_2026_02 PARTITION OF module_view_events
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE module_view_events_2026_03 PARTITION OF module_view_events
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

-- Indexes for module_view_events
CREATE INDEX idx_module_view_events_module ON module_view_events(module_name, viewed_at DESC);
CREATE INDEX idx_module_view_events_user ON module_view_events(user_id, viewed_at DESC);
CREATE INDEX idx_module_view_events_time ON module_view_events(viewed_at DESC);

-- Compilation event tracking (partitioned by time)
CREATE TABLE compilation_events (
    id BIGSERIAL PRIMARY KEY,
    module_name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    language VARCHAR(50) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    success BOOLEAN NOT NULL DEFAULT false,
    error_message TEXT,
    error_type VARCHAR(100),
    cache_hit BOOLEAN DEFAULT false,
    file_count INTEGER,
    output_size BIGINT,
    compiler_version VARCHAR(50)
) PARTITION BY RANGE (started_at);

-- Create monthly partitions for compilation_events
CREATE TABLE compilation_events_2026_01 PARTITION OF compilation_events
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE compilation_events_2026_02 PARTITION OF compilation_events
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE compilation_events_2026_03 PARTITION OF compilation_events
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

-- Indexes for compilation_events
CREATE INDEX idx_compilation_events_module ON compilation_events(module_name, version);
CREATE INDEX idx_compilation_events_language ON compilation_events(language, started_at DESC);
CREATE INDEX idx_compilation_events_time ON compilation_events(started_at DESC);
CREATE INDEX idx_compilation_events_success ON compilation_events(success, started_at DESC);

-- Comment on tables
COMMENT ON TABLE download_events IS 'Tracks all module download events for analytics';
COMMENT ON TABLE module_view_events IS 'Tracks module page views and access patterns';
COMMENT ON TABLE compilation_events IS 'Tracks compilation job execution and performance';
