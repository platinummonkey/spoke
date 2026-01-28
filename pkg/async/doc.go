// Package async provides safe concurrent execution primitives for background tasks.
//
// # Overview
//
// This package handles goroutine lifecycle management with panic recovery, timeout
// enforcement, context cancellation, and error collection.
//
// # Key Functions
//
// SafeGo: Execute function in goroutine with safety features
//
//	async.SafeGo(ctx, 30*time.Second, errChan, func(ctx context.Context) error {
//		// Task code with automatic panic recovery and timeout
//		return processData(ctx)
//	})
//
// WorkerPool: Managed pool of concurrent workers
//
//	pool := async.NewWorkerPool(10, 100) // 10 workers, 100 task buffer
//	pool.Start()
//	defer pool.Stop()
//
//	pool.Submit(func(ctx context.Context) error {
//		return compileModule(ctx)
//	})
//
// Batch: Concurrent batch processing
//
//	results := async.Batch(items, 5, func(ctx context.Context, item Item) error {
//		return processItem(ctx, item)
//	})
//
// # Features
//
// Panic Recovery: Captures panics with stack traces
// Timeout Enforcement: Per-task timeouts
// Context Cancellation: Respects context cancellation
// Error Collection: Non-blocking error channels
// Graceful Shutdown: Worker draining
//
// # Use Cases
//
// Webhook delivery, background compilation, batch imports, analytics, cache warming
//
// # Related Packages
//
//   - pkg/webhooks: Uses SafeGo for delivery
//   - pkg/codegen: Uses WorkerPool for compilation
package async
