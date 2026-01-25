-- Base schema for Spoke protobuf registry
-- Migration: 001_create_base_schema
-- Description: Creates core tables for modules, versions, files, and compiled artifacts

-- Modules table: Central registry of protobuf modules
CREATE TABLE IF NOT EXISTS modules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',

    CONSTRAINT modules_name_valid CHECK (name ~ '^[a-z][a-z0-9_-]*$')
);

CREATE INDEX idx_modules_name ON modules(name);
CREATE INDEX idx_modules_created_at ON modules(created_at DESC);
CREATE INDEX idx_modules_updated_at ON modules(updated_at DESC);

-- Versions table: Semantic versions for each module
CREATE TABLE IF NOT EXISTS versions (
    id BIGSERIAL PRIMARY KEY,
    module_id BIGINT NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    version VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Source information
    source_repository VARCHAR(512),
    source_commit_sha VARCHAR(64),
    source_branch VARCHAR(255),

    -- Dependency management
    dependencies JSONB DEFAULT '[]',

    -- Storage location (S3 paths)
    proto_files_object_key VARCHAR(512),
    proto_files_checksum VARCHAR(64),

    -- Metadata
    metadata JSONB DEFAULT '{}',

    UNIQUE(module_id, version),
    CONSTRAINT versions_version_valid CHECK (version ~ '^v?[0-9]+\.[0-9]+\.[0-9]+.*')
);

CREATE INDEX idx_versions_module_id ON versions(module_id);
CREATE INDEX idx_versions_created_at ON versions(created_at DESC);
CREATE INDEX idx_versions_dependencies ON versions USING GIN(dependencies);
CREATE INDEX idx_versions_module_version ON versions(module_id, version);

-- Proto files table: Individual proto file metadata
CREATE TABLE IF NOT EXISTS proto_files (
    id BIGSERIAL PRIMARY KEY,
    version_id BIGINT NOT NULL REFERENCES versions(id) ON DELETE CASCADE,
    file_path VARCHAR(512) NOT NULL,

    -- Content-addressed storage
    content_hash VARCHAR(64) NOT NULL,
    object_key VARCHAR(512) NOT NULL,
    file_size BIGINT NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    UNIQUE(version_id, file_path)
);

CREATE INDEX idx_proto_files_version_id ON proto_files(version_id);
CREATE INDEX idx_proto_files_content_hash ON proto_files(content_hash);
CREATE INDEX idx_proto_files_file_path ON proto_files(file_path);

-- Compiled artifacts table: Generated language code
CREATE TABLE IF NOT EXISTS compiled_artifacts (
    id BIGSERIAL PRIMARY KEY,
    version_id BIGINT NOT NULL REFERENCES versions(id) ON DELETE CASCADE,
    language VARCHAR(50) NOT NULL,

    -- Package metadata
    package_name VARCHAR(255),
    package_version VARCHAR(100),

    -- Storage location
    object_key VARCHAR(512) NOT NULL,
    file_size BIGINT NOT NULL,
    checksum VARCHAR(64) NOT NULL,

    -- Compilation metadata
    compiled_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    compiler_version VARCHAR(50),

    UNIQUE(version_id, language)
);

CREATE INDEX idx_compiled_artifacts_version_id ON compiled_artifacts(version_id);
CREATE INDEX idx_compiled_artifacts_language ON compiled_artifacts(language);

-- Updated_at trigger for modules
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_modules_updated_at
BEFORE UPDATE ON modules
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Module with latest version view
CREATE VIEW modules_with_latest AS
SELECT
    m.*,
    v.version as latest_version,
    v.created_at as latest_version_created_at
FROM modules m
LEFT JOIN LATERAL (
    SELECT version, created_at
    FROM versions
    WHERE module_id = m.id
    ORDER BY created_at DESC
    LIMIT 1
) v ON true;
