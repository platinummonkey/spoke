package async

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

// SafeGo executes a function in a goroutine with:
// - Context cancellation support
// - Panic recovery
// - Timeout enforcement
// - Error logging
//
// Use this instead of bare `go func()` to prevent goroutine leaks and crashes.
//
// Example:
//
//	SafeGo(r.Context(), 5*time.Second, "analytics tracking", func(ctx context.Context) error {
//	    return tracker.TrackEvent(ctx, event)
//	})
func SafeGo(parentCtx context.Context, timeout time.Duration, taskName string, fn func(context.Context) error) {
	go func() {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(parentCtx, timeout)
		defer cancel()

		// Recover from panics
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SafeGo] PANIC in %s: %v\nStack trace:\n%s",
					taskName, r, string(debug.Stack()))
			}
		}()

		// Execute function
		if err := fn(ctx); err != nil {
			// Log error but don't crash
			// Caller can decide if this is critical or not
			log.Printf("[SafeGo] Error in %s: %v", taskName, err)
		}
	}()
}

// SafeGoNoError is like SafeGo but for functions that don't return errors.
// Still provides panic recovery and context support.
//
// Example:
//
//	SafeGoNoError(r.Context(), 5*time.Second, "cache warming", func(ctx context.Context) {
//	    cache.Warm(ctx, keys)
//	})
func SafeGoNoError(parentCtx context.Context, timeout time.Duration, taskName string, fn func(context.Context)) {
	SafeGo(parentCtx, timeout, taskName, func(ctx context.Context) error {
		fn(ctx)
		return nil
	})
}

// WorkerPool manages a pool of workers that process tasks from a channel.
// Provides graceful shutdown and error collection.
type WorkerPool struct {
	workers   int
	taskName  string
	timeout   time.Duration
	workCh    chan func(context.Context) error
	doneCh    chan struct{}
	errCh     chan error
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewWorkerPool creates a new worker pool.
//
// Example:
//
//	pool := NewWorkerPool(ctx, 10, "image processing", 30*time.Second)
//	defer pool.Shutdown(5 * time.Second)
//
//	pool.Submit(func(ctx context.Context) error {
//	    return processImage(ctx, imageID)
//	})
func NewWorkerPool(ctx context.Context, workers int, taskName string, timeout time.Duration) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)

	pool := &WorkerPool{
		workers:  workers,
		taskName: taskName,
		timeout:  timeout,
		workCh:   make(chan func(context.Context) error, workers*2),
		doneCh:   make(chan struct{}),
		errCh:    make(chan error, workers*10), // Larger buffer to avoid drops
		ctx:      ctx,
		cancel:   cancel,
	}

	// Start workers and wait for them to finish in background
	go func() {
		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				pool.worker(id)
			}(i)
		}
		wg.Wait()
		close(pool.doneCh)
	}()

	return pool
}

// Submit adds a task to the worker pool.
// Returns error if pool is shut down.
func (p *WorkerPool) Submit(fn func(context.Context) error) error {
	select {
	case p.workCh <- fn:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool shut down")
	}
}

// Shutdown gracefully shuts down the worker pool.
// Waits up to timeout for workers to finish current tasks.
func (p *WorkerPool) Shutdown(timeout time.Duration) error {
	// Signal shutdown and close work channel
	p.cancel()

	// Close work channel (may already be closed by Batch)
	select {
	case <-p.ctx.Done():
		// Already cancelled, don't close again
	default:
		close(p.workCh)
	}

	// Wait for workers to finish with timeout
	select {
	case <-p.doneCh:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("worker pool shutdown timed out after %v", timeout)
	}
}

// Errors returns a channel that receives worker errors.
// Non-blocking, use select to check for errors.
func (p *WorkerPool) Errors() <-chan error {
	return p.errCh
}

func (p *WorkerPool) worker(id int) {
	defer func() {
		// Recover from panics first
		if r := recover(); r != nil {
			log.Printf("[WorkerPool] PANIC in worker %d (%s): %v\nStack trace:\n%s",
				id, p.taskName, r, string(debug.Stack()))
		}
	}()

	for {
		select {
		case <-p.ctx.Done():
			return

		case fn, ok := <-p.workCh:
			if !ok {
				return
			}

			// Create context with timeout for this task
			ctx, cancel := context.WithTimeout(p.ctx, p.timeout)

			// Execute task with panic recovery
			func() {
				defer cancel()
				defer func() {
					if r := recover(); r != nil {
						err := fmt.Errorf("panic: %v", r)
						select {
						case p.errCh <- err:
						default:
							log.Printf("[WorkerPool] Error channel full, dropping error: %v", err)
						}
					}
				}()

				if err := fn(ctx); err != nil {
					select {
					case p.errCh <- err:
					default:
						log.Printf("[WorkerPool] Error channel full, dropping error: %v", err)
					}
				}
			}()
		}
	}
}

// Batch processes a slice of items concurrently using a worker pool.
// Returns all errors encountered.
//
// Example:
//
//	items := []string{"file1.txt", "file2.txt", "file3.txt"}
//	errs := Batch(ctx, items, 5, "file processing", 10*time.Second, func(ctx context.Context, item string) error {
//	    return processFile(ctx, item)
//	})
//	if len(errs) > 0 {
//	    log.Printf("Failed to process %d files", len(errs))
//	}
func Batch[T any](ctx context.Context, items []T, workers int, taskName string, timeout time.Duration,
	fn func(context.Context, T) error) []error {

	pool := NewWorkerPool(ctx, workers, taskName, timeout)
	defer pool.Shutdown(5 * time.Second)

	// Submit all tasks
	for _, item := range items {
		item := item // Capture loop variable
		if err := pool.Submit(func(ctx context.Context) error {
			return fn(ctx, item)
		}); err != nil {
			return []error{err}
		}
	}

	// Wait for completion by closing work channel
	pool.cancel()
	close(pool.workCh)
	<-pool.doneCh

	// Collect errors
	var errs []error
	for {
		select {
		case err := <-pool.errCh:
			errs = append(errs, err)
		default:
			return errs
		}
	}
}
