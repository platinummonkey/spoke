-- Plugin Marketplace Schema
-- Migration 010: Create plugin marketplace tables

-- Plugin registry table
CREATE TABLE IF NOT EXISTS plugins (
    id VARCHAR(255) PRIMARY KEY,  -- rust-language, buf-connect-go
    name VARCHAR(255) NOT NULL,
    description TEXT,
    author VARCHAR(255) NOT NULL,
    license VARCHAR(50),
    homepage TEXT,
    repository TEXT,
    type VARCHAR(50) NOT NULL,  -- language, validator, generator, runner
    security_level VARCHAR(50) NOT NULL DEFAULT 'community',  -- official, verified, community
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMP NULL,
    verified_by VARCHAR(255) NULL,
    download_count BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_plugins_type ON plugins(type);
CREATE INDEX IF NOT EXISTS idx_plugins_security ON plugins(security_level);
CREATE INDEX IF NOT EXISTS idx_plugins_enabled ON plugins(enabled);
CREATE INDEX IF NOT EXISTS idx_plugins_downloads ON plugins(download_count DESC);

-- Plugin versions table
CREATE TABLE IF NOT EXISTS plugin_versions (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,  -- Semver
    api_version VARCHAR(50) NOT NULL,  -- SDK API version
    manifest_url TEXT NOT NULL,  -- URL to plugin.yaml
    download_url TEXT NOT NULL,  -- URL to plugin archive (.tar.gz)
    checksum VARCHAR(64) NOT NULL,  -- SHA-256 checksum
    size_bytes BIGINT NOT NULL,
    downloads BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_plugin_version UNIQUE (plugin_id, version),
    CONSTRAINT fk_plugin_versions_plugin FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_plugin_versions_plugin ON plugin_versions(plugin_id, version);
CREATE INDEX IF NOT EXISTS idx_plugin_versions_downloads ON plugin_versions(downloads DESC);

-- Plugin ratings and reviews
CREATE TABLE IF NOT EXISTS plugin_reviews (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,  -- User identifier
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_plugin_user_review UNIQUE (plugin_id, user_id),
    CONSTRAINT fk_plugin_reviews_plugin FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_plugin_reviews_plugin ON plugin_reviews(plugin_id, rating DESC);
CREATE INDEX IF NOT EXISTS idx_plugin_reviews_user ON plugin_reviews(user_id);

-- Plugin installations (track who installed what)
CREATE TABLE IF NOT EXISTS plugin_installations (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    user_id VARCHAR(255) NULL,  -- User identifier
    organization_id VARCHAR(255) NULL,  -- Organization identifier
    installed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    uninstalled_at TIMESTAMP NULL,
    CONSTRAINT fk_plugin_installations_plugin FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_plugin_installations_plugin ON plugin_installations(plugin_id, version);
CREATE INDEX IF NOT EXISTS idx_plugin_installations_user ON plugin_installations(user_id, installed_at DESC);
CREATE INDEX IF NOT EXISTS idx_plugin_installations_org ON plugin_installations(organization_id, installed_at DESC);

-- Plugin statistics (aggregated daily)
CREATE TABLE IF NOT EXISTS plugin_stats_daily (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    downloads BIGINT NOT NULL DEFAULT 0,
    installations BIGINT NOT NULL DEFAULT 0,
    uninstallations BIGINT NOT NULL DEFAULT 0,
    active_installations BIGINT NOT NULL DEFAULT 0,
    avg_rating FLOAT,
    review_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_plugin_date UNIQUE (plugin_id, date),
    CONSTRAINT fk_plugin_stats_daily_plugin FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_plugin_stats_daily_plugin ON plugin_stats_daily(plugin_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_plugin_stats_daily_downloads ON plugin_stats_daily(downloads DESC, date DESC);

-- Plugin dependencies
CREATE TABLE IF NOT EXISTS plugin_dependencies (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    depends_on_plugin_id VARCHAR(255) NOT NULL,
    depends_on_version VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_plugin_dependencies_plugin FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE,
    CONSTRAINT fk_plugin_dependencies_depends_on FOREIGN KEY (depends_on_plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_plugin_deps_plugin ON plugin_dependencies(plugin_id, version);
CREATE INDEX IF NOT EXISTS idx_plugin_deps_depends_on ON plugin_dependencies(depends_on_plugin_id);

-- Plugin tags for better discoverability
CREATE TABLE IF NOT EXISTS plugin_tags (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL,
    tag VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_plugin_tag UNIQUE (plugin_id, tag),
    CONSTRAINT fk_plugin_tags_plugin FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_plugin_tags_tag ON plugin_tags(tag);
CREATE INDEX IF NOT EXISTS idx_plugin_tags_plugin ON plugin_tags(plugin_id);
