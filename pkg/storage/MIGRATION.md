# Storage Layer Migration Guide

## Overview

This document describes the migration from the original `Storage` interface to the new context-aware `StorageV2` interface, along with planned enhancements for S3 and Redis integration.

## Background

The original `Storage` interface was designed without context support, which prevents:
- Request cancellation propagation
- Timeout enforcement
- Distributed tracing
- Proper resource cleanup

The `StorageV2` interface addresses these issues by requiring `context.Context` as the first parameter in all operations.

## Current State

### ✅ Completed
- `StorageV2` interface defined in `pkg/storage/interfaces.go`
- `PostgresStorage` implements both `Storage` and `StorageV2`
- Legacy methods delegate to context-aware versions using `context.Background()`
- Connection pooling with primary/replica support
- OpenTelemetry tracing integration

### ⚠️ In Progress
The following features have TODOs and need completion:

#### 1. S3 Integration (Line 52-59)
**Current**: Placeholder with TODO comment
**Goal**: Store large proto files in S3 instead of PostgreSQL BYTEA columns
**Benefits**:
- Reduced database size
- Better performance for large files
- Cost savings (S3 cheaper than RDS storage)
- CDN integration possibilities

**Implementation Plan**:
```go
// TODO: Initialize S3 client
var s3Client *S3Client
if config.S3Endpoint != "" {
    s3Client, err = NewS3Client(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create s3 client: %w", err)
    }
}
```

**Required Config**:
- `S3Endpoint`: S3-compatible endpoint URL
- `S3Bucket`: Bucket name for proto files
- `S3AccessKey`: Access key ID
- `S3SecretKey`: Secret access key
- `S3Region`: AWS region (optional)

#### 2. Redis Cache Layer (Line 61-68)
**Current**: Placeholder with TODO comment
**Goal**: Cache frequently accessed modules/versions in Redis
**Benefits**:
- Reduced database load
- Faster response times (sub-ms vs 10-50ms)
- Read replica offloading

**Implementation Plan**:
```go
// TODO: Initialize Redis client
var redisClient *RedisClient
if config.CacheEnabled && config.RedisURL != "" {
    redisClient, err = NewRedisClient(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create redis client: %w", err)
    }
}
```

**Cache Strategy**:
- TTL: 5 minutes for module metadata, 1 hour for version data
- Cache-aside pattern (read-through)
- Invalidate on write operations
- LRU eviction policy

#### 3. Dependency Parsing (Lines 434, 534)
**Current**: Dependencies stored as TEXT, not parsed
**Goal**: Parse proto imports and store as structured data
**Benefits**:
- Recursive dependency resolution
- Circular dependency detection
- Dependency graph visualization
- Version compatibility checking

**Schema Enhancement Needed**:
```sql
CREATE TABLE IF NOT EXISTS module_dependencies (
    id SERIAL PRIMARY KEY,
    module_name VARCHAR(255) NOT NULL,
    version VARCHAR(100) NOT NULL,
    dependency_name VARCHAR(255) NOT NULL,
    dependency_version VARCHAR(100),
    import_path TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (module_name, version)
        REFERENCES versions(module_name, version) ON DELETE CASCADE,
    UNIQUE(module_name, version, dependency_name, import_path)
);

CREATE INDEX idx_dependencies_module ON module_dependencies(module_name, version);
CREATE INDEX idx_dependencies_dep ON module_dependencies(dependency_name);
```

#### 4. Version Update Operations (Lines 542, 588, 619, 624)
**Current**: Basic CRUD operations only
**Goal**: Support version updates, deprecation, and lifecycle management
**Benefits**:
- Update version metadata without recreating
- Mark versions as deprecated
- Version lifecycle states (draft, published, deprecated, archived)
- Audit trail for changes

**Proposed API**:
```go
// Update version metadata (description, tags, etc.)
UpdateVersion(ctx context.Context, moduleName, version string, updates *VersionUpdate) error

// Mark version as deprecated
DeprecateVersion(ctx context.Context, moduleName, version string, reason string) error

// Archive old versions
ArchiveVersion(ctx context.Context, moduleName, version string) error

// Get version changelog
GetVersionChangelog(ctx context.Context, moduleName, version string) ([]*VersionChange, error)
```

## Migration Timeline

### Phase 1: Foundation (Completed)
- ✅ Define StorageV2 interface
- ✅ Implement PostgresStorage with dual interface support
- ✅ Add connection pooling and replica support
- ✅ Integrate OpenTelemetry tracing

### Phase 2: Infrastructure (Current - 4 weeks)
Week 1-2:
- [ ] Implement S3Client for file storage
- [ ] Migrate existing proto files to S3 (background job)
- [ ] Update file operations to use S3

Week 3-4:
- [ ] Implement RedisClient for caching
- [ ] Add cache warming on startup
- [ ] Implement cache invalidation hooks

### Phase 3: Features (Next - 4 weeks)
Week 1-2:
- [ ] Implement dependency parser
- [ ] Create dependency graph storage
- [ ] Add recursive dependency resolution

Week 3-4:
- [ ] Add version update operations
- [ ] Implement version lifecycle management
- [ ] Create version changelog tracking

### Phase 4: Deprecation (Future - 2 weeks)
- [ ] Add deprecation warnings to legacy Storage methods
- [ ] Update all callers to use StorageV2
- [ ] Remove legacy Storage interface
- [ ] Update documentation

## API Changes

### Legacy (Deprecated)
```go
// No context, no cancellation, no tracing
storage.GetModule("mymodule")
storage.CreateVersion(version)
```

### StorageV2 (Current)
```go
// Context-aware, cancellable, traceable
storage.GetModuleContext(ctx, "mymodule")
storage.CreateVersionContext(ctx, version)
```

### Caller Migration Example

**Before**:
```go
func (h *Handler) GetModule(w http.ResponseWriter, r *http.Request) {
    module, err := h.storage.GetModule(name)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    json.NewEncoder(w).Encode(module)
}
```

**After**:
```go
func (h *Handler) GetModule(w http.ResponseWriter, r *http.Request) {
    // Use request context for proper cancellation
    module, err := h.storage.GetModuleContext(r.Context(), name)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    json.NewEncoder(w).Encode(module)
}
```

## Current Issues

### Issue 1: context.Background() in Legacy Methods
**Location**: `pkg/storage/postgres/postgres.go:81-99`

**Problem**: Legacy methods create `context.Background()` which:
- Loses request cancellation
- Breaks timeout enforcement
- Prevents tracing propagation

**Example**:
```go
func (s *PostgresStorage) CreateModule(module *api.Module) error {
    return s.CreateModuleContext(context.Background(), module)
}
```

**Impact**: If HTTP request is cancelled, database operation continues

**Solution**: Deprecate legacy methods, force callers to provide context

### Issue 2: No Cache Layer
**Problem**: Every request hits database, even for hot data

**Impact**:
- Database load: 1000+ QPS
- P99 latency: 50-100ms
- Cost: High RDS IOPS usage

**Solution**: Redis cache (Phase 2)
- Target latency: <5ms for cache hits
- Cache hit ratio: >80%
- Reduced DB load: 80%

### Issue 3: No S3 Integration
**Problem**: Large proto files stored in PostgreSQL BYTEA

**Impact**:
- Database bloat (10GB+ for large registries)
- Expensive storage ($0.115/GB/month RDS vs $0.023/GB/month S3)
- Slow backups and replication

**Solution**: S3 storage (Phase 2)
- Store files in S3, metadata in PostgreSQL
- Implement pre-signed URLs for downloads
- Background migration job

## Configuration

### Current Config
```go
type Config struct {
    PostgresURL           string        // Primary database URL
    PostgresReplicaURLs   string        // Comma-separated replica URLs
    PostgresMaxConns      int           // Max connections (default: 25)
    PostgresMinConns      int           // Min connections (default: 5)
    PostgresTimeout       time.Duration // Query timeout (default: 30s)
}
```

### Future Config (After Phase 2)
```go
type Config struct {
    // PostgreSQL
    PostgresURL           string
    PostgresReplicaURLs   string
    PostgresMaxConns      int
    PostgresMinConns      int
    PostgresTimeout       time.Duration

    // S3
    S3Endpoint            string // S3-compatible endpoint
    S3Bucket              string // Bucket for proto files
    S3AccessKey           string // Access key
    S3SecretKey           string // Secret key
    S3Region              string // AWS region

    // Redis
    CacheEnabled          bool          // Enable caching
    RedisURL              string        // Redis connection string
    CacheTTL              time.Duration // Default TTL
    CacheMaxSize          int64         // Max cache size in bytes
}
```

## Testing Strategy

### Unit Tests
- Mock S3Client and RedisClient
- Test cache hit/miss scenarios
- Test context cancellation propagation

### Integration Tests
- Real PostgreSQL, S3, Redis (testcontainers)
- Test migration from legacy to StorageV2
- Test cache invalidation
- Test S3 fallback on Redis failure

### Performance Tests
- Benchmark with/without cache
- Measure S3 vs PostgreSQL for large files
- Load testing (10k+ QPS)

## Rollout Plan

### Week 1-2: S3 Integration
1. Deploy S3Client code
2. Add feature flag `SPOKE_S3_ENABLED=false`
3. Test with small percentage of traffic
4. Monitor error rates, latency
5. Gradually increase to 100%
6. Background job to migrate old files

### Week 3-4: Redis Integration
1. Deploy RedisClient code
2. Add feature flag `SPOKE_CACHE_ENABLED=false`
3. Warm cache with top 1000 modules
4. Enable for reads only (cache-aside)
5. Monitor cache hit ratio
6. Enable cache invalidation on writes

### Week 5-6: Dependency Parsing
1. Deploy parser code
2. Background job to parse existing versions
3. Add dependency endpoints to API
4. Update UI to show dependency graph

### Week 7-8: Version Lifecycle
1. Add lifecycle columns to schema
2. Implement update operations
3. Add audit logging for changes
4. Deploy version management UI

## Monitoring

### Metrics to Track
- **Storage operations**: success/error rates, latency (p50, p95, p99)
- **Cache performance**: hit ratio, miss ratio, eviction rate
- **S3 operations**: upload/download success rate, bandwidth
- **Database**: connection pool usage, query duration, deadlocks

### Alerts
- Cache hit ratio < 70%
- Database connection pool exhaustion
- S3 upload failure rate > 1%
- Average query latency > 100ms

## FAQ

### Q: When will legacy Storage interface be removed?
**A**: After Phase 4 (approximately 6 months from now). Deprecation warnings will be added 3 months before removal.

### Q: Do I need to update my code now?
**A**: Yes, if you're adding new code. Use StorageV2 methods with proper context. Existing code will continue to work but should be migrated during next refactor.

### Q: What if S3 is unavailable?
**A**: Fallback to PostgreSQL BYTEA storage. S3 is optional and can be disabled via config.

### Q: What if Redis is unavailable?
**A**: Fallback to direct database queries. Cache layer is transparent to application logic.

### Q: How do I test locally without S3/Redis?
**A**: Use localstack for S3 and redis-server for Redis. Docker Compose config provided in `docker/dev-stack.yml`.

## References

- [StorageV2 Interface Definition](../storage/interfaces.go)
- [PostgresStorage Implementation](./postgres.go)
- [S3Client Design Doc](./S3_DESIGN.md) (TBD)
- [Redis Cache Design Doc](./REDIS_CACHE.md) (TBD)

## Contact

For questions or clarification, contact the Platform team or file an issue on GitHub.

---

*Last Updated: 2026-01-27*
*Status: In Progress*
*Owner: Platform Team*
