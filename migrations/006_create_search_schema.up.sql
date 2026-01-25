-- Advanced Search and Dependency Visualization Schema
-- Migration: 006_create_search_schema
-- Description: Creates search index, saved searches, search history, and bookmarks tables

-- Proto search index table
-- Stores individual proto entities (messages, enums, services, methods, fields) for full-text search
CREATE TABLE IF NOT EXISTS proto_search_index (
    id BIGSERIAL PRIMARY KEY,
    version_id BIGINT NOT NULL REFERENCES versions(id) ON DELETE CASCADE,

    -- Entity identification
    entity_type VARCHAR(50) NOT NULL, -- 'message', 'enum', 'service', 'method', 'field'
    entity_name VARCHAR(255) NOT NULL,
    full_path TEXT NOT NULL, -- e.g., 'user.v1.UserProfile.email'
    parent_path TEXT, -- e.g., 'user.v1.UserProfile'

    -- Proto file context
    proto_file_path VARCHAR(512),
    line_number INTEGER,

    -- Entity content
    description TEXT,
    comments TEXT, -- Extracted from proto comments

    -- Field-specific attributes
    field_type VARCHAR(255), -- For fields: 'string', 'int32', etc.
    field_number INTEGER, -- For fields: field number
    is_repeated BOOLEAN DEFAULT false,
    is_optional BOOLEAN DEFAULT false,

    -- Method-specific attributes
    method_input_type VARCHAR(255), -- For methods: input message type
    method_output_type VARCHAR(255), -- For methods: output message type

    -- Full-text search vector (automatically updated by trigger)
    search_vector tsvector,

    -- Flexible metadata (tags, labels, custom fields)
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for proto_search_index
CREATE INDEX idx_proto_search_vector ON proto_search_index USING GIN(search_vector);
CREATE INDEX idx_proto_search_metadata ON proto_search_index USING GIN(metadata jsonb_path_ops);
CREATE INDEX idx_proto_search_version_id ON proto_search_index(version_id);
CREATE INDEX idx_proto_search_entity_type ON proto_search_index(entity_type);
CREATE INDEX idx_proto_search_full_path ON proto_search_index(full_path);
CREATE INDEX idx_proto_search_parent_path ON proto_search_index(parent_path) WHERE parent_path IS NOT NULL;
CREATE INDEX idx_proto_search_field_type ON proto_search_index(field_type) WHERE field_type IS NOT NULL;
CREATE INDEX idx_proto_search_entity_name ON proto_search_index(entity_name);

-- Composite index for common query patterns
CREATE INDEX idx_proto_search_version_entity ON proto_search_index(version_id, entity_type);
CREATE INDEX idx_proto_search_entity_field_type ON proto_search_index(entity_type, field_type) WHERE field_type IS NOT NULL;

-- Trigger function to update search_vector automatically
-- Weights: A (highest) for entity_name, B for full_path, C for description, D (lowest) for comments
CREATE OR REPLACE FUNCTION proto_search_index_trigger() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.entity_name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.full_path, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.comments, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.field_type, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.method_input_type, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.method_output_type, '')), 'C');
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER proto_search_index_update_trigger
    BEFORE INSERT OR UPDATE ON proto_search_index
    FOR EACH ROW EXECUTE FUNCTION proto_search_index_trigger();

-- Saved searches table
-- Allows users to save frequently used search queries
CREATE TABLE IF NOT EXISTS saved_searches (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE, -- NULL for anonymous (stored in localStorage)
    name VARCHAR(255) NOT NULL,
    query TEXT NOT NULL,
    filters JSONB DEFAULT '{}', -- JSON object with filter parameters
    description TEXT,
    is_shared BOOLEAN NOT NULL DEFAULT false, -- For future team sharing feature
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_saved_searches_user_id ON saved_searches(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_saved_searches_is_shared ON saved_searches(is_shared) WHERE is_shared = true;
CREATE INDEX idx_saved_searches_created_at ON saved_searches(created_at DESC);

CREATE TRIGGER update_saved_searches_updated_at
    BEFORE UPDATE ON saved_searches
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Search history table
-- Tracks search queries for suggestions and analytics
CREATE TABLE IF NOT EXISTS search_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE, -- NULL for anonymous
    query TEXT NOT NULL,
    filters JSONB DEFAULT '{}',
    result_count INTEGER,
    search_duration_ms INTEGER, -- Query execution time in milliseconds
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_search_history_user_id ON search_history(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_search_history_created_at ON search_history(created_at DESC);
CREATE INDEX idx_search_history_query ON search_history USING GIN(to_tsvector('english', query));

-- Bookmarks table
-- Allows users to bookmark frequently accessed modules/entities
CREATE TABLE IF NOT EXISTS bookmarks (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE, -- NULL for anonymous (stored in localStorage)
    module_name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    entity_path TEXT, -- Optional: specific entity (e.g., 'UserService.CreateUser')
    entity_type VARCHAR(50), -- Optional: 'message', 'enum', 'service', etc.
    notes TEXT,
    tags TEXT[] DEFAULT '{}', -- User-defined tags for organization
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, module_name, version, entity_path)
);

CREATE INDEX idx_bookmarks_user_id ON bookmarks(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_bookmarks_module_version ON bookmarks(module_name, version);
CREATE INDEX idx_bookmarks_entity_type ON bookmarks(entity_type) WHERE entity_type IS NOT NULL;
CREATE INDEX idx_bookmarks_tags ON bookmarks USING GIN(tags);
CREATE INDEX idx_bookmarks_created_at ON bookmarks(created_at DESC);

CREATE TRIGGER update_bookmarks_updated_at
    BEFORE UPDATE ON bookmarks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Search suggestions materialized view
-- Pre-computed popular searches for autocomplete
CREATE MATERIALIZED VIEW IF NOT EXISTS search_suggestions AS
SELECT
    query,
    COUNT(*) as search_count,
    AVG(result_count)::INTEGER as avg_results,
    MAX(created_at) as last_searched_at
FROM search_history
WHERE created_at > NOW() - INTERVAL '30 days'
  AND result_count > 0
GROUP BY query
HAVING COUNT(*) > 1
ORDER BY search_count DESC, last_searched_at DESC
LIMIT 1000;

CREATE INDEX idx_search_suggestions_query ON search_suggestions(query);
CREATE INDEX idx_search_suggestions_count ON search_suggestions(search_count DESC);

-- Function to refresh search suggestions (can be called periodically)
CREATE OR REPLACE FUNCTION refresh_search_suggestions() RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY search_suggestions;
END;
$$ LANGUAGE plpgsql;

-- Popular entities view for discovery
CREATE MATERIALIZED VIEW IF NOT EXISTS popular_entities AS
SELECT
    psi.entity_type,
    psi.entity_name,
    psi.full_path,
    COUNT(DISTINCT b.user_id) as bookmark_count,
    COUNT(DISTINCT sh.user_id) as search_count,
    MAX(b.created_at) as last_bookmarked_at,
    MAX(sh.created_at) as last_searched_at
FROM proto_search_index psi
LEFT JOIN bookmarks b ON psi.full_path = b.entity_path
LEFT JOIN search_history sh ON psi.entity_name = sh.query
WHERE psi.created_at > NOW() - INTERVAL '90 days'
GROUP BY psi.entity_type, psi.entity_name, psi.full_path
HAVING COUNT(DISTINCT b.user_id) > 0 OR COUNT(DISTINCT sh.user_id) > 1
ORDER BY (COUNT(DISTINCT b.user_id) * 2 + COUNT(DISTINCT sh.user_id)) DESC
LIMIT 500;

CREATE INDEX idx_popular_entities_type ON popular_entities(entity_type);
CREATE INDEX idx_popular_entities_name ON popular_entities(entity_name);

-- Function to refresh popular entities (can be called periodically)
CREATE OR REPLACE FUNCTION refresh_popular_entities() RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY popular_entities;
END;
$$ LANGUAGE plpgsql;

-- View for recent searches (for quick access)
CREATE VIEW recent_searches AS
SELECT DISTINCT ON (user_id, query)
    id,
    user_id,
    query,
    filters,
    result_count,
    created_at
FROM search_history
WHERE created_at > NOW() - INTERVAL '7 days'
ORDER BY user_id, query, created_at DESC;
