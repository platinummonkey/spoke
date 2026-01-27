# Storage Interface Deprecation and Migration Guide

## Overview

The `api.Storage` interface is being deprecated in favor of the new unified `storage.Storage` interface. This migration provides better context propagation, interface segregation, and modern Go practices.

## Timeline

- **v1.8.0** (Current): `storage.Storage` available, `api.Storage` deprecated
- **v1.9.0** (+3 months): Runtime deprecation warnings added
- **v1.10.0** (+6 months): Breaking change notices in logs
- **v2.0.0** (+12 months): `api.Storage` removed completely

## Why This Change?

### Problems with api.Storage

1. **No context support**: Methods couldn't be canceled or timed out
2. **Import cycles**: Difficult to compose with other packages
3. **Monolithic interface**: Clients needed full interface even for read-only operations
4. **No pagination**: List operations couldn't handle large datasets efficiently

### Benefits of storage.Storage

1. **Context-aware**: All methods accept `context.Context` for cancellation and timeouts
2. **Interface segregation**: Composed of focused sub-interfaces (ModuleReader, ModuleWriter, etc.)
3. **Better testability**: Mock only the methods you need
4. **Pagination support**: Built-in support for paginated list operations
5. **Extended capabilities**: File storage, artifact storage, cache management, health checks

## Migration Guide

### For Implementation Authors (storage backends)

Both `FileSystemStorage` and `PostgresStorage` already implement the new interface. If you have a custom storage implementation:

**Before:**
```go
type MyStorage struct {}

func (s *MyStorage) CreateModule(module *api.Module) error {
    // implementation
}

func (s *MyStorage) GetModule(name string) (*api.Module, error) {
    // implementation
}
// ... other methods
```

**After:**
```go
type MyStorage struct {}

// Keep old methods for backward compatibility
func (s *MyStorage) CreateModule(module *api.Module) error {
    // implementation
}

func (s *MyStorage) GetModule(name string) (*api.Module, error) {
    // implementation
}

// Add new context-aware methods
func (s *MyStorage) CreateModuleContext(ctx context.Context, module *api.Module) error {
    // Optionally check context cancellation
    if err := ctx.Err(); err != nil {
        return err
    }
    return s.CreateModule(module)
}

func (s *MyStorage) GetModuleContext(ctx context.Context, name string) (*api.Module, error) {
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    return s.GetModule(name)
}

// Implement additional interfaces
func (s *MyStorage) ListModulesPaginated(ctx context.Context, limit, offset int) ([]*api.Module, int64, error) {
    // implementation
}

func (s *MyStorage) HealthCheck(ctx context.Context) error {
    // implementation
}

// ... other new methods

// Verify interface compliance
var _ storage.Storage = (*MyStorage)(nil)
```

### For API Consumers (handlers, services)

**Internal packages** (pkg/api, pkg/dependencies, pkg/docs) should continue using `api.Storage` to avoid import cycles. The implementations satisfy both interfaces.

**Entry points** (cmd/spoke/main.go) can optionally use `storage.Storage` for type safety, but it's not required.

**No migration needed** - your code continues to work as-is.

### Method Migration Map

| Old Method (api.Storage) | New Method (storage.Storage) |
|--------------------------|------------------------------|
| `CreateModule(module)` | `CreateModuleContext(ctx, module)` |
| `GetModule(name)` | `GetModuleContext(ctx, name)` |
| `ListModules()` | `ListModulesContext(ctx)` |
| `CreateVersion(version)` | `CreateVersionContext(ctx, version)` |
| `GetVersion(moduleName, version)` | `GetVersionContext(ctx, moduleName, version)` |
| `ListVersions(moduleName)` | `ListVersionsContext(ctx, moduleName)` |
| `UpdateVersion(version)` | `UpdateVersionContext(ctx, version)` |
| `GetFile(moduleName, version, path)` | `GetFileContext(ctx, moduleName, version, path)` |

### New Methods (storage.Storage only)

| Method | Description |
|--------|-------------|
| `ListModulesPaginated(ctx, limit, offset)` | List modules with pagination |
| `ListVersionsPaginated(ctx, moduleName, limit, offset)` | List versions with pagination |
| `GetFileContent(ctx, hash)` | Get file by content hash (S3/object storage) |
| `PutFileContent(ctx, content, contentType)` | Store file by content (returns hash) |
| `GetCompiledArtifact(ctx, moduleName, version, language)` | Get compiled artifact |
| `PutCompiledArtifact(ctx, moduleName, version, language, artifact)` | Store compiled artifact |
| `InvalidateCache(ctx, patterns...)` | Invalidate cache entries |
| `HealthCheck(ctx)` | Check storage health |

## Interface Composition

The `storage.Storage` interface is composed of focused sub-interfaces:

```go
type Storage interface {
    ModuleReader      // GetModuleContext, ListModulesContext, ListModulesPaginated
    ModuleWriter      // CreateModuleContext
    VersionReader     // GetVersionContext, ListVersionsContext, ListVersionsPaginated, GetFileContext
    VersionWriter     // CreateVersionContext, UpdateVersionContext
    FileStorage       // GetFileContent, PutFileContent
    ArtifactStorage   // GetCompiledArtifact, PutCompiledArtifact
    CacheManager      // InvalidateCache
    HealthChecker     // HealthCheck
}
```

This allows functions to require only the capabilities they need:

```go
// Only needs read access
func AnalyzeDependencies(reader storage.VersionReader, module, version string) error {
    ver, err := reader.GetVersionContext(ctx, module, version)
    // Cannot accidentally write
}

// Only needs write access
func CreateBackup(writer storage.ModuleWriter, module *api.Module) error {
    return writer.CreateModuleContext(ctx, module)
}
```

## Context Usage Examples

### Basic Context Propagation

```go
func (h *Handler) GetModule(w http.ResponseWriter, r *http.Request) {
    // Pass request context for automatic cancellation
    module, err := h.storage.GetModuleContext(r.Context(), name)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(module)
}
```

### Timeout Example

```go
func GetModuleWithTimeout(store storage.Storage, name string) (*api.Module, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    return store.GetModuleContext(ctx, name)
}
```

### Cancellation Example

```go
func StreamModules(store storage.Storage, ctx context.Context) error {
    modules, err := store.ListModulesContext(ctx)
    if err != nil {
        return err
    }

    for _, module := range modules {
        // Check for cancellation
        if ctx.Err() != nil {
            return ctx.Err()
        }
        // Process module...
    }
    return nil
}
```

## Architecture Decisions

### Why Keep api.Storage Separate?

The `api.Storage` interface remains in the `pkg/api` package to avoid import cycles:

```
pkg/api → pkg/search (for search_adapter.go)
pkg/search → pkg/api (would create cycle if api imported storage)
```

The solution:
- `api.Storage` stays in `pkg/api` (deprecated but functional)
- `storage.Storage` lives in `pkg/storage` (canonical interface)
- Implementations satisfy BOTH interfaces
- Internal packages use `api.Storage`
- Entry points can use `storage.Storage`

### Why Not Merge search.StorageReader?

`search.StorageReader` must remain separate for the same reason - it would create an import cycle. The `SearchStorageAdapter` bridges between `api.Storage` and `search.StorageReader`.

### Why Not Merge marketplace.Storage?

`marketplace.Storage` handles plugin artifacts (archives, manifests, checksums) while registry storage handles protobuf schemas. These are different domains with no overlap.

## Rollback Plan

Until v2.0.0, rolling back is simple:

1. Both interfaces are supported
2. Implementations provide both old and new methods
3. No breaking changes for consumers
4. Gradually adopt new interface at your own pace

## Testing Your Migration

```go
func TestStorageImplementation(t *testing.T) {
    var store storage.Storage = NewMyStorage()

    ctx := context.Background()

    // Test module operations
    module := &api.Module{Name: "test"}
    err := store.CreateModuleContext(ctx, module)
    assert.NoError(t, err)

    retrieved, err := store.GetModuleContext(ctx, "test")
    assert.NoError(t, err)
    assert.Equal(t, module.Name, retrieved.Name)

    // Test context cancellation
    cancelCtx, cancel := context.WithCancel(ctx)
    cancel() // Cancel immediately

    _, err = store.GetModuleContext(cancelCtx, "test")
    assert.Error(t, err)
    assert.True(t, errors.Is(err, context.Canceled))
}
```

## FAQ

### Q: Do I need to update my code now?
**A:** No. The old `api.Storage` interface continues to work. Plan to migrate before v2.0.0 (12 months away).

### Q: What if I only need read access?
**A:** Use the `storage.ModuleReader` or `storage.VersionReader` sub-interfaces instead of the full `storage.Storage`.

### Q: Do context methods support cancellation in FileSystemStorage?
**A:** Currently, FileSystemStorage delegates to non-context methods for backward compatibility. PostgresStorage fully supports context cancellation.

### Q: Can I use both interfaces in the same codebase?
**A:** Yes! Implementations satisfy both interfaces. Mix and match as needed during migration.

### Q: What about performance?
**A:** The new interface has the same performance. Context propagation adds minimal overhead (~nanoseconds).

### Q: How do I know if I've migrated correctly?
**A:** Compile with `-race` flag and run tests. The compile-time interface checks will catch issues.

## Support

If you encounter migration issues:

1. Check this guide first
2. Review the code examples in `pkg/storage/`
3. Open an issue at https://github.com/platinummonkey/spoke/issues

## See Also

- [Interface Segregation Principle](https://en.wikipedia.org/wiki/Interface_segregation_principle)
- [Go Context Package](https://pkg.go.dev/context)
- [Effective Go - Interfaces](https://go.dev/doc/effective_go#interfaces)
