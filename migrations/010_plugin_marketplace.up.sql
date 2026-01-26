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
    download_count BIGINT NOT NULL DEFAULT 0,
    INDEX idx_plugins_type (type),
    INDEX idx_plugins_security (security_level),
    INDEX idx_plugins_enabled (enabled),
    INDEX idx_plugins_downloads (download_count DESC)
);

-- Plugin versions table
CREATE TABLE IF NOT EXISTS plugin_versions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    plugin_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,  -- Semver
    api_version VARCHAR(50) NOT NULL,  -- SDK API version
    manifest_url TEXT NOT NULL,  -- URL to plugin.yaml
    download_url TEXT NOT NULL,  -- URL to plugin archive (.tar.gz)
    checksum VARCHAR(64) NOT NULL,  -- SHA-256 checksum
    size_bytes BIGINT NOT NULL,
    downloads BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY unique_plugin_version (plugin_id, version),
    INDEX idx_plugin_versions_plugin (plugin_id, version),
    INDEX idx_plugin_versions_downloads (downloads DESC),
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

-- Plugin ratings and reviews
CREATE TABLE IF NOT EXISTS plugin_reviews (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    plugin_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,  -- User identifier
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY unique_plugin_user_review (plugin_id, user_id),
    INDEX idx_plugin_reviews_plugin (plugin_id, rating DESC),
    INDEX idx_plugin_reviews_user (user_id),
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

-- Plugin installations (track who installed what)
CREATE TABLE IF NOT EXISTS plugin_installations (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    plugin_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    user_id VARCHAR(255) NULL,  -- User identifier
    organization_id VARCHAR(255) NULL,  -- Organization identifier
    installed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    uninstalled_at TIMESTAMP NULL,
    INDEX idx_plugin_installations_plugin (plugin_id, version),
    INDEX idx_plugin_installations_user (user_id, installed_at DESC),
    INDEX idx_plugin_installations_org (organization_id, installed_at DESC),
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

-- Plugin statistics (aggregated daily)
CREATE TABLE IF NOT EXISTS plugin_stats_daily (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    plugin_id VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    downloads BIGINT NOT NULL DEFAULT 0,
    installations BIGINT NOT NULL DEFAULT 0,
    uninstallations BIGINT NOT NULL DEFAULT 0,
    active_installations BIGINT NOT NULL DEFAULT 0,
    avg_rating FLOAT,
    review_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY unique_plugin_date (plugin_id, date),
    INDEX idx_plugin_stats_daily_plugin (plugin_id, date DESC),
    INDEX idx_plugin_stats_daily_downloads (downloads DESC, date DESC),
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

-- Plugin dependencies
CREATE TABLE IF NOT EXISTS plugin_dependencies (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    plugin_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    depends_on_plugin_id VARCHAR(255) NOT NULL,
    depends_on_version VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_plugin_deps_plugin (plugin_id, version),
    INDEX idx_plugin_deps_depends_on (depends_on_plugin_id),
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE,
    FOREIGN KEY (depends_on_plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);

-- Plugin tags for better discoverability
CREATE TABLE IF NOT EXISTS plugin_tags (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    plugin_id VARCHAR(255) NOT NULL,
    tag VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY unique_plugin_tag (plugin_id, tag),
    INDEX idx_plugin_tags_tag (tag),
    INDEX idx_plugin_tags_plugin (plugin_id),
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
);
