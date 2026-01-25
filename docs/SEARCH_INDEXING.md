# Search Indexing

## Overview

Spoke's advanced search feature provides full-text search across all proto entities (messages, enums, services, methods, fields) using PostgreSQL Full-Text Search (FTS).

## Architecture

### Database Schema

The search system uses four main tables:

1. **proto_search_index** - Stores individual proto entities with full-text search vectors
2. **saved_searches** - User-saved search queries
3. **search_history** - Tracks search queries for suggestions
4. **bookmarks** - User bookmarks for frequently accessed modules/entities

### Search Vector

Each indexed entity has a `search_vector` (tsvector) automatically updated by a PostgreSQL trigger with weighted search terms:

- **A (highest)**: Entity name
- **B**: Full path (e.g., `user.v1.UserProfile.email`)
- **C**: Description, field type, method input/output types
- **D (lowest)**: Comments

### Indexes

- **GIN index** on `search_vector` for fast full-text search
- **GIN index** on `metadata` JSONB for flexible metadata queries
- **B-tree indexes** on commonly filtered columns (version_id, entity_type, full_path)

## Automatic Indexing

When a new version is pushed to Spoke, the search indexer automatically:

1. Parses all proto files using the AST parser
2. Extracts entities: messages, fields, enums, enum values, services, RPC methods
3. Builds full paths for each entity (e.g., `package.Message.field`)
4. Extracts comments and descriptions
5. Inserts entities into `proto_search_index` table
6. PostgreSQL trigger updates `search_vector` for FTS

## Manual Re-indexing

### Re-index a Specific Version

```bash
# Using psql
psql -d spoke -c "DELETE FROM proto_search_index WHERE version_id = <version_id>"
# Then push the version again to trigger re-indexing
```

### Re-index All Versions

Coming in Phase 2: CLI command `spoke index rebuild`

## Entity Types

The indexer extracts the following entity types:

- **message** - Protobuf message definitions
- **field** - Message fields (with type, number, repeated/optional flags)
- **enum** - Enum definitions
- **enum_value** - Enum values (with number)
- **service** - gRPC service definitions
- **method** - RPC methods (with input/output types, streaming flags)

## Entity Attributes

### All Entities
- `version_id` - Reference to versions table
- `entity_type` - Type of entity
- `entity_name` - Name of the entity
- `full_path` - Fully qualified path (e.g., `user.v1.UserProfile.email`)
- `parent_path` - Parent path (e.g., `user.v1.UserProfile`)
- `proto_file_path` - Source proto file
- `line_number` - Line number in source file
- `description` - First line of comments
- `comments` - All comment text
- `metadata` - JSONB for flexible attributes

### Fields
- `field_type` - Field type (string, int32, etc.)
- `field_number` - Proto field number
- `is_repeated` - Boolean flag
- `is_optional` - Boolean flag

### Methods
- `method_input_type` - Input message type
- `method_output_type` - Output message type
- `metadata.client_streaming` - Boolean flag
- `metadata.server_streaming` - Boolean flag

## Search Performance

### Optimization Strategies

1. **GIN Indexes**: Fast lookups for tsvector searches
2. **Covering Indexes**: Composite indexes for common query patterns
3. **Batch Inserts**: Entities inserted in batches of 100
4. **Materialized Views**: Pre-computed popular searches and entities

### Expected Performance

- Search query latency: <200ms (p95)
- Index creation: ~1000 entities/second
- Storage: ~1KB per entity (average)

## Migration

### Applying the Migration

```bash
# Using golang-migrate
migrate -path migrations -database "postgres://user:pass@localhost:5432/spoke" up

# Or using psql
psql -d spoke -f migrations/006_create_search_schema.up.sql
```

### Rolling Back

```bash
# Using golang-migrate
migrate -path migrations -database "postgres://user:pass@localhost:5432/spoke" down 1

# Or using psql
psql -d spoke -f migrations/006_create_search_schema.down.sql
```

## Monitoring

### Useful Queries

**Check index size:**
```sql
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE tablename = 'proto_search_index';
```

**Check entity counts by type:**
```sql
SELECT entity_type, COUNT(*) as count
FROM proto_search_index
GROUP BY entity_type
ORDER BY count DESC;
```

**Most recent indexed versions:**
```sql
SELECT
    m.name,
    v.version,
    COUNT(psi.id) as entity_count,
    MAX(psi.created_at) as last_indexed
FROM proto_search_index psi
JOIN versions v ON psi.version_id = v.id
JOIN modules m ON v.module_id = m.id
GROUP BY m.name, v.version
ORDER BY last_indexed DESC
LIMIT 10;
```

**Slow search queries:**
```sql
SELECT
    query,
    COUNT(*) as executions,
    AVG(search_duration_ms) as avg_duration_ms,
    MAX(search_duration_ms) as max_duration_ms
FROM search_history
WHERE search_duration_ms > 500
GROUP BY query
ORDER BY avg_duration_ms DESC
LIMIT 10;
```

## Troubleshooting

### Indexing Failures

**Symptom**: Version pushed but not searchable

**Solution**:
1. Check server logs for indexing errors
2. Verify proto files are valid
3. Manually re-index the version

**Common Causes**:
- Invalid proto syntax
- Missing proto files
- Database connection issues
- Insufficient database permissions

### Search Not Working

**Symptom**: No search results returned

**Diagnostics**:
```sql
-- Check if data exists
SELECT COUNT(*) FROM proto_search_index;

-- Check search_vector
SELECT entity_name, search_vector
FROM proto_search_index
LIMIT 10;

-- Test search manually
SELECT entity_name, full_path
FROM proto_search_index
WHERE search_vector @@ to_tsquery('english', 'user');
```

### Performance Issues

**Symptom**: Slow search queries

**Solutions**:
1. Analyze query patterns: `EXPLAIN ANALYZE SELECT ...`
2. Verify GIN indexes are being used
3. Update statistics: `ANALYZE proto_search_index;`
4. Consider increasing `work_mem` for complex queries
5. Enable query result caching (coming in Phase 2)

## Next Steps

Phase 1 provides the foundation. Future phases will add:

- **Phase 2**: Enhanced search service with advanced query syntax
- **Phase 3**: Dependency graph visualization
- **Phase 4**: Advanced query UI with autocomplete
- **Phase 5**: Saved searches and bookmarks UI
- **Phase 6**: Impact analysis UI

## References

- [PostgreSQL Full-Text Search](https://www.postgresql.org/docs/current/textsearch.html)
- [GIN Indexes](https://www.postgresql.org/docs/current/gin.html)
- [Migration 006](../migrations/006_create_search_schema.up.sql)
