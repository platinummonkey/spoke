# Concurrency Guidelines for Spoke

## Overview

This document establishes patterns and guidelines for concurrent programming in the Spoke codebase. Following these patterns prevents goroutine leaks, panics, and resource exhaustion.

## Core Principles

1. **Never use bare `go func()` in production code** - Always use `async.SafeGo` or `async.WorkerPool`
2. **Always propagate context** - Never use `context.Background()` in request handlers
3. **Always recover from panics** - Unhandled panics crash the entire process
4. **Always have a cleanup strategy** - Goroutines must be stoppable
5. **Always set timeouts** - Unbounded operations leak resources

## When to Use Goroutines

### ✅ Good Use Cases

1. **Async logging/analytics** - Non-critical operations that shouldn't block responses
2. **Background workers** - Long-running tasks (cleanup, health checks, retries)
3. **Concurrent processing** - Parallelizing independent operations
4. **Non-blocking notifications** - Webhooks, events, cache warming

### ❌ Anti-Patterns

1. **Short synchronous operations** - Function calls are fast enough
2. **Database queries in HTTP handlers** - Use synchronous calls with context
3. **File I/O** - Usually fast enough to be synchronous
4. **Critical path operations** - Must complete before responding to user

## Required Patterns

### 1. Async Operations in HTTP Handlers

**Use `async.SafeGo` for fire-and-forget operations:**

```go
func (s *Server) createModule(w http.ResponseWriter, r *http.Request) {
    // ... create module ...

    // Track analytics asynchronously (don't block response)
    if s.eventTracker != nil {
        async.SafeGo(r.Context(), 5*time.Second, "track module creation", func(ctx context.Context) error {
            return s.eventTracker.Track(ctx, analytics.ModuleCreated{
                ModuleName: module.Name,
                UserID:     getUserID(r),
            })
        })
    }

    httputil.WriteCreated(w, module)
}
```

**Key points:**
- Use request context `r.Context()` not `context.Background()`
- Set reasonable timeout (usually 5-10s for async operations)
- Provide descriptive task name for debugging
- Don't crash on errors - log and continue

### 2. Background Workers

**Use context-aware workers with panic recovery:**

```go
func (s *Service) StartCleanup(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    go func() {
        // Always recover from panics
        defer func() {
            if r := recover(); r != nil {
                log.Printf("[Cleanup] PANIC: %v\n%s", r, debug.Stack())
            }
        }()

        for {
            select {
            case <-ctx.Done():
                log.Println("[Cleanup] Shutting down")
                return

            case <-ticker.C:
                if err := s.cleanup(); err != nil {
                    log.Printf("[Cleanup] Error: %v", err)
                }
            }
        }
    }()
}
```

**Better: Use SafeGo wrapper:**

```go
func (s *Service) StartCleanup(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)

    async.SafeGo(ctx, 0, "cleanup worker", func(ctx context.Context) error {
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-ticker.C:
                if err := s.cleanup(); err != nil {
                    return fmt.Errorf("cleanup failed: %w", err)
                }
            }
        }
    })
}
```

### 3. Worker Pools

**Use `async.WorkerPool` for bounded concurrency:**

```go
func (s *Service) ProcessImages(ctx context.Context, imageIDs []string) error {
    pool := async.NewWorkerPool(ctx, 10, "image processing", 30*time.Second)
    defer pool.Shutdown(5 * time.Second)

    // Submit tasks
    for _, imageID := range imageIDs {
        imageID := imageID // Capture loop variable
        if err := pool.Submit(func(ctx context.Context) error {
            return s.processImage(ctx, imageID)
        }); err != nil {
            return fmt.Errorf("failed to submit task: %w", err)
        }
    }

    // Collect errors
    var errs []error
    for err := range pool.Errors() {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("processing failed with %d errors", len(errs))
    }

    return nil
}
```

**Better: Use `async.Batch` for slice processing:**

```go
func (s *Service) ProcessImages(ctx context.Context, imageIDs []string) error {
    errs := async.Batch(ctx, imageIDs, 10, "image processing", 30*time.Second,
        func(ctx context.Context, imageID string) error {
            return s.processImage(ctx, imageID)
        },
    )

    if len(errs) > 0 {
        return fmt.Errorf("processing failed with %d errors: %v", len(errs), errs)
    }

    return nil
}
```

## Context Propagation Rules

### ✅ Correct Context Usage

```go
// HTTP Handler - use request context
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
    result, err := h.service.Fetch(r.Context())
    // Context canceled if client disconnects
}

// Service method - accept context parameter
func (s *Service) Fetch(ctx context.Context) (*Result, error) {
    return s.storage.GetData(ctx)
}

// Async operation - propagate parent context
func (s *Service) LogEvent(parentCtx context.Context, event Event) {
    async.SafeGo(parentCtx, 5*time.Second, "log event", func(ctx context.Context) error {
        return s.logger.Log(ctx, event)
    })
}
```

### ❌ Wrong Context Usage

```go
// WRONG: Using Background in HTTP handler
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
    ctx := context.Background() // ❌ Ignores client cancellation
    result, err := h.service.Fetch(ctx)
}

// WRONG: Creating new context instead of propagating
func (s *Service) LogEvent(parentCtx context.Context, event Event) {
    ctx := context.Background() // ❌ Loses parent cancellation
    async.SafeGo(ctx, 5*time.Second, "log event", func(ctx context.Context) error {
        return s.logger.Log(ctx, event)
    })
}

// WRONG: Not accepting context parameter
func (s *Service) Fetch() (*Result, error) { // ❌ Can't be canceled
    return s.storage.GetData()
}
```

## Panic Recovery

**All goroutines must recover from panics:**

```go
// Good: Using SafeGo (panic recovery built-in)
async.SafeGo(ctx, 5*time.Second, "task", func(ctx context.Context) error {
    // Panics are automatically recovered and logged
    panic("oops") // Won't crash the process
})

// Manual recovery (only if you can't use SafeGo)
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("[Worker] PANIC: %v\n%s", r, debug.Stack())
        }
    }()

    // ... work ...
}()
```

**Why this matters:**
- Unhandled panics crash the entire process (all HTTP handlers, all workers)
- In production, this means service outage
- Panics in one goroutine shouldn't affect others

## Graceful Shutdown

**All long-running goroutines must respect context cancellation:**

```go
func (s *Service) Start(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    async.SafeGo(ctx, 0, "worker", func(ctx context.Context) error {
        for {
            select {
            case <-ctx.Done():
                log.Println("Worker shutting down gracefully")
                return ctx.Err()

            case <-ticker.C:
                if err := s.doWork(); err != nil {
                    log.Printf("Work failed: %v", err)
                }
            }
        }
    })
}
```

**Shutdown coordination:**

```go
// In main.go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start services
    service.Start(ctx)
    worker.Start(ctx)

    // Wait for signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    // Cancel context to stop all workers
    cancel()

    // Give workers time to cleanup
    time.Sleep(2 * time.Second)
}
```

## Error Handling

### Fire-and-Forget Operations

```go
// Non-critical: Log error but don't crash
async.SafeGo(ctx, 5*time.Second, "track analytics", func(ctx context.Context) error {
    if err := tracker.Track(ctx, event); err != nil {
        // SafeGo logs this automatically
        return err
    }
    return nil
})
```

### Critical Operations

```go
// Critical: Collect errors and fail if any occur
pool := async.NewWorkerPool(ctx, 10, "critical processing", 30*time.Second)
defer pool.Shutdown(5 * time.Second)

for _, item := range items {
    pool.Submit(func(ctx context.Context) error {
        return processItem(ctx, item)
    })
}

// Check for errors
var errs []error
for err := range pool.Errors() {
    errs = append(errs, err)
}

if len(errs) > 0 {
    return fmt.Errorf("processing failed: %v", errs)
}
```

## Common Patterns

### Pattern: Async Logging

```go
func (h *Handler) CreateModule(w http.ResponseWriter, r *http.Request) {
    module, err := h.service.Create(r.Context(), req)
    if err != nil {
        httputil.WriteError(w, err, http.StatusInternalServerError)
        return
    }

    // Log asynchronously - don't block response
    async.SafeGo(r.Context(), 5*time.Second, "audit log", func(ctx context.Context) error {
        return h.auditLog.Log(ctx, audit.Event{
            Action: "module.created",
            UserID: getUserID(r),
            Data:   module,
        })
    })

    httputil.WriteCreated(w, module)
}
```

### Pattern: Cache Warming

```go
func (s *Service) WarmCache(ctx context.Context, keys []string) {
    async.SafeGoNoError(ctx, 30*time.Second, "cache warming", func(ctx context.Context) {
        for _, key := range keys {
            if ctx.Err() != nil {
                return // Stop if canceled
            }

            if val, err := s.compute(key); err == nil {
                s.cache.Set(key, val)
            }
        }
    })
}
```

### Pattern: Concurrent HTTP Requests

```go
func (s *Service) FetchMultiple(ctx context.Context, urls []string) ([]Response, error) {
    return async.Batch(ctx, urls, 5, "fetch URLs", 10*time.Second,
        func(ctx context.Context, url string) error {
            resp, err := s.fetch(ctx, url)
            if err != nil {
                return fmt.Errorf("failed to fetch %s: %w", url, err)
            }
            results = append(results, resp)
            return nil
        },
    )
}
```

### Pattern: Periodic Cleanup

```go
func (s *Service) StartPeriodicCleanup(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)

    async.SafeGo(ctx, 0, "periodic cleanup", func(ctx context.Context) error {
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return ctx.Err()

            case <-ticker.C:
                cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
                if err := s.cleanup(cleanupCtx); err != nil {
                    log.Printf("Cleanup failed: %v", err)
                }
                cancel()
            }
        }
    })
}
```

## Testing Concurrent Code

### Test for Race Conditions

```bash
# Always run tests with race detector
go test -race ./...

# In CI, make this mandatory
make test-race
```

### Test for Goroutine Leaks

```go
func TestNoGoroutineLeak(t *testing.T) {
    before := runtime.NumGoroutine()

    ctx, cancel := context.WithCancel(context.Background())
    service := NewService()
    service.Start(ctx)

    // Do work...

    // Stop service
    cancel()
    time.Sleep(100 * time.Millisecond) // Give goroutines time to exit

    after := runtime.NumGoroutine()
    if after > before {
        t.Errorf("Goroutine leak: before=%d, after=%d", before, after)
    }
}
```

### Test Context Cancellation

```go
func TestContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())

    executed := atomic.Bool{}
    completed := atomic.Bool{}

    async.SafeGo(ctx, 10*time.Second, "test", func(ctx context.Context) error {
        executed.Store(true)

        select {
        case <-time.After(1 * time.Second):
            completed.Store(true)
        case <-ctx.Done():
            return ctx.Err()
        }

        return nil
    })

    // Cancel immediately
    cancel()
    time.Sleep(100 * time.Millisecond)

    if !executed.Load() {
        t.Error("Function should have started")
    }
    if completed.Store() {
        t.Error("Function should have been canceled")
    }
}
```

## Performance Considerations

### Worker Pool Sizing

```go
// CPU-bound work: Use number of CPUs
workers := runtime.NumCPU()
pool := async.NewWorkerPool(ctx, workers, "cpu work", timeout)

// I/O-bound work: Use higher number (10-100)
workers := 20
pool := async.NewWorkerPool(ctx, workers, "http requests", timeout)

// Database queries: Match connection pool size
workers := 10 // If DB pool is 10 connections
pool := async.NewWorkerPool(ctx, workers, "db queries", timeout)
```

### Channel Buffer Sizing

```go
// Unbuffered: Synchronous handoff
ch := make(chan Task)

// Small buffer: Smooth out bursts
ch := make(chan Task, 10)

// Large buffer: Decouple producer/consumer
ch := make(chan Task, 1000)
```

### Timeout Guidelines

```go
// Quick operations (logging, metrics)
timeout := 5 * time.Second

// Normal I/O operations (HTTP, DB queries)
timeout := 10 * time.Second

// Long operations (compilation, file processing)
timeout := 30 * time.Second

// Very long operations (backups, migrations)
timeout := 5 * time.Minute

// Infinite workers (until context canceled)
timeout := 0
```

## Debugging

### Enable Verbose Logging

```go
// SafeGo logs all panics and errors automatically
async.SafeGo(ctx, timeout, "task name", fn)

// Check logs for:
// [SafeGo] Error in task name: ...
// [SafeGo] PANIC in task name: ...
```

### Trace Goroutine Leaks

```bash
# Get goroutine dump
curl http://localhost:9090/debug/pprof/goroutine?debug=2

# Profile goroutines
go tool pprof http://localhost:9090/debug/pprof/goroutine
```

### Monitor Goroutine Count

```go
// Add metrics
go func() {
    ticker := time.NewTicker(10 * time.Second)
    for range ticker.C {
        count := runtime.NumGoroutine()
        metrics.RecordGoroutineCount(count)

        if count > 1000 {
            log.Printf("WARNING: High goroutine count: %d", count)
        }
    }
}()
```

## Migration Checklist

When updating old code:

- [ ] Replace `go func()` with `async.SafeGo`
- [ ] Add context parameter if missing
- [ ] Propagate context from caller (don't use `context.Background()`)
- [ ] Set appropriate timeout (5-30s typical)
- [ ] Add descriptive task name
- [ ] Test with `-race` flag
- [ ] Test context cancellation
- [ ] Test panic recovery

## References

- [pkg/async/goroutine.go](../../pkg/async/goroutine.go) - SafeGo, WorkerPool, Batch implementations
- [pkg/async/goroutine_test.go](../../pkg/async/goroutine_test.go) - Usage examples and tests
- [Go Concurrency Patterns](https://go.dev/blog/pipelines) - Official Go blog
- [Context Package](https://pkg.go.dev/context) - Context documentation

## Questions?

If you're unsure about a concurrency pattern:

1. Check existing code using `async.SafeGo` for examples
2. Review this document's patterns section
3. Ask in team chat or PR review
4. When in doubt, use `async.SafeGo` - it's the safe default
