# Style Guide: Agent-Readable Code

This guide supplements existing Go style guides with rules specifically designed to make code safe for AI coding agents to modify.

## 1. Context Keys

**RULE**: All context keys MUST be defined in a central location with documentation.

✅ **GOOD**:
```go
// pkg/contextkeys/keys.go
const (
    // AuthKey contains *auth.AuthContext
    // Set by: AuthMiddleware
    // Required by: all protected endpoints
    AuthKey Key = "auth_context"
)
```

❌ **BAD**:
```go
// Scattered magic strings
ctx.Value("auth")
ctx.Value("auth_context")
ctx.Value("authentication")
```

**Rationale**: Agents cannot infer correct context key names. Centralization prevents typos and documents dependencies.

---

## 2. Invariants

**RULE**: Non-obvious invariants MUST be documented with INVARIANT comments.

✅ **GOOD**:
```go
// CacheKey represents a compilation cache key
//
// INVARIANT: Cache keys must be generated with sorted inputs.
// Changing String() invalidates entire cache.
type CacheKey struct { ... }
```

❌ **BAD**:
```go
// CacheKey for compilations
type CacheKey struct { ... }
```

**Rationale**: Agents don't understand implicit ordering requirements. Violations cause silent failures.

---

## 3. Magic Constants

**RULE**: Hardcoded values MUST be extracted to named constants with units and rationale.

✅ **GOOD**:
```go
const (
    // DefaultCacheMaxSize is 100MB - typical proto artifacts are ~100KB
    DefaultCacheMaxSize = 100 * 1024 * 1024
)
```

❌ **BAD**:
```go
maxSize := 100 * 1024 * 1024  // What is this?
```

**Rationale**: Agents cannot determine if values are tunable parameters or critical limits.

---

## 4. Middleware Ordering

**RULE**: Middleware with ordering dependencies MUST document required order.

✅ **GOOD**:
```go
// QuotaMiddleware enforces quotas
//
// MIDDLEWARE CHAIN ORDERING (CRITICAL):
// 1. auth.Middleware - sets auth context
// 2. OrgContextMiddleware - extracts org
// 3. QuotaMiddleware - checks quotas
type QuotaMiddleware struct { ... }
```

❌ **BAD**:
```go
// QuotaMiddleware enforces quotas
type QuotaMiddleware struct { ... }
```

**Rationale**: Agents cannot infer cross-cutting dependencies. Wrong ordering causes security bypasses.

---

## 5. Global State

**RULE**: Global variables MUST be documented with warnings about side effects.

✅ **GOOD**:
```go
// globalCache is DEPRECATED - has system-wide side effects
// DO NOT USE - migrate to cache.MemoryCache
var globalCache sync.Map
```

❌ **BAD**:
```go
var globalCache sync.Map
```

**Rationale**: Agents don't recognize implicit shared state. Modifications can break unrelated code.

---

## 6. Async Behavior

**RULE**: Asynchronous operations MUST be marked with ASYNC comments explaining behavior.

✅ **GOOD**:
```go
// CheckAPIRateLimit checks rate limits
//
// ASYNC BEHAVIOR: Spawns goroutine to increment counters
// Failures during increment don't block request
func (m *QuotaMiddleware) CheckAPIRateLimit(...) { ... }
```

❌ **BAD**:
```go
func (m *QuotaMiddleware) CheckAPIRateLimit(...) {
    go m.orgService.IncrementAPIRequests(orgID) // Hidden!
}
```

**Rationale**: Agents assume functions are synchronous. Hidden goroutines cause race conditions.

---

## 7. Cleanup Requirements

**RULE**: Types requiring manual cleanup MUST document this in godoc with CRITICAL warning.

✅ **GOOD**:
```go
// RateLimiter implements rate limiting
//
// CRITICAL: MUST CALL StartCleanup(ctx) after construction
// Without cleanup, old buckets accumulate (memory leak)
type RateLimiter struct { ... }
```

❌ **BAD**:
```go
// RateLimiter implements rate limiting
type RateLimiter struct { ... }
```

**Rationale**: Agents don't understand resource lifecycle. Missing cleanup causes memory leaks.

---

## 8. Panic Recovery

**RULE**: Panic recovery MUST use centralized helper with logging and metrics.

✅ **GOOD**:
```go
defer func() {
    if observability.RecoverPanic(ctx, logger, "worker") {
        resultCh <- errorResult
    }
}()
```

❌ **BAD**:
```go
defer func() {
    if r := recover(); r != nil {
        fmt.Printf("panic: %v\n", r) // Lost!
    }
}()
```

**Rationale**: Agents copy-paste patterns. Centralization ensures consistent error reporting.

---

## 9. Structured Logging

**RULE**: NEVER use fmt.Printf/fmt.Println in production code. Use structured logger.

✅ **GOOD**:
```go
logger.WithError(err).Warn("failed to initialize cache")
```

❌ **BAD**:
```go
fmt.Printf("Warning: failed to initialize cache: %v\n", err)
```

**Rationale**: Agents see fmt.Printf as acceptable. Output is lost in production systems.

---

## 10. Defer Execution Order

**RULE**: Functions with multiple defers MUST document execution order if it matters.

✅ **GOOD**:
```go
// Execute compiles code
//
// DEFER STACK (executes LIFO):
// 1. timing defer - records duration (includes cleanup)
// 2. outputDir cleanup
// 3. inputDir cleanup
func (r *Runner) Execute(...) {
    defer func() { result.Duration = time.Since(start) }()
    defer os.RemoveAll(outputDir)
    defer os.RemoveAll(inputDir)
}
```

❌ **BAD**:
```go
func (r *Runner) Execute(...) {
    defer func() { result.Duration = time.Since(start) }()
    defer os.RemoveAll(outputDir)
    defer os.RemoveAll(inputDir)
}
```

**Rationale**: Agents don't understand LIFO order. Can cause timing bugs or resource leaks.

---

## 11. Deprecated Code

**RULE**: Deprecated code MUST include concrete removal date and migration guide.

✅ **GOOD**:
```go
// Storage interface is DEPRECATED
//
// REMOVAL TIMELINE:
// - Deprecated: 2025-01-15
// - Removal: 2026-01-15 (12 months)
//
// MIGRATION: Use storage.Storage with context methods
// See pkg/storage/MIGRATION.md
type Storage interface { ... }
```

❌ **BAD**:
```go
// DEPRECATED: Use storage.Storage instead
type Storage interface { ... }
```

**Rationale**: Agents need concrete dates and migration paths. Vague warnings are ignored.

---

## 12. Options Maps

**RULE**: `map[string]string` parameters MUST document valid keys and value formats.

✅ **GOOD**:
```go
// CompileRequest represents a compilation request
//
// OPTIONS: Protoc plugin options (affects cache key)
// Valid keys:
//   - "go_opt" - Go plugin options (e.g., "paths=source_relative")
//   - "java_package" - Java package override
type CompileRequest struct {
    Options map[string]string
}
```

❌ **BAD**:
```go
type CompileRequest struct {
    Options map[string]string
}
```

**Rationale**: Agents cannot determine valid keys or value constraints. Leads to runtime errors.

---

## Enforcement

These rules should be enforced via:
1. **Code review** - reviewers check for INVARIANT, ASYNC, CRITICAL comments
2. **Linter rules** - detect global vars, fmt.Printf, panic without recovery
3. **Documentation tests** - verify required godoc sections exist

## Migration

Existing code should be updated incrementally:
1. Start with HIGH priority items (context keys, invariants, global state)
2. Add documentation when touching nearby code
3. Full compliance targeted for v2.0.0 release
