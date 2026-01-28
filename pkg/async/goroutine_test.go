package async

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSafeGo_Success(t *testing.T) {
	ctx := context.Background()
	executed := atomic.Bool{}

	SafeGo(ctx, 1*time.Second, "test task", func(ctx context.Context) error {
		executed.Store(true)
		return nil
	})

	// Wait for goroutine to complete
	time.Sleep(100 * time.Millisecond)

	if !executed.Load() {
		t.Error("SafeGo did not execute function")
	}
}

func TestSafeGo_WithError(t *testing.T) {
	ctx := context.Background()
	executed := atomic.Bool{}

	SafeGo(ctx, 1*time.Second, "test task", func(ctx context.Context) error {
		executed.Store(true)
		return errors.New("test error")
	})

	// Wait for goroutine to complete
	time.Sleep(100 * time.Millisecond)

	if !executed.Load() {
		t.Error("SafeGo did not execute function despite error")
	}
	// Error should be logged but not crash
}

func TestSafeGo_Timeout(t *testing.T) {
	ctx := context.Background()
	started := atomic.Bool{}
	completed := atomic.Bool{}

	SafeGo(ctx, 50*time.Millisecond, "test task", func(ctx context.Context) error {
		started.Store(true)
		select {
		case <-time.After(200 * time.Millisecond):
			completed.Store(true)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	if !started.Load() {
		t.Error("Function did not start")
	}
	if completed.Load() {
		t.Error("Function should have been canceled by timeout")
	}
}

func TestSafeGo_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	executed := atomic.Bool{}

	SafeGo(ctx, 1*time.Second, "test task", func(ctx context.Context) error {
		executed.Store(true)
		panic("test panic")
	})

	// Wait for goroutine to complete
	time.Sleep(100 * time.Millisecond)

	if !executed.Load() {
		t.Error("Function did not execute before panic")
	}
	// Panic should be recovered and logged
}

func TestSafeGo_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := atomic.Bool{}
	completed := atomic.Bool{}

	SafeGo(ctx, 5*time.Second, "test task", func(ctx context.Context) error {
		started.Store(true)
		select {
		case <-time.After(1 * time.Second):
			completed.Store(true)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	// Cancel context quickly
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	if !started.Load() {
		t.Error("Function did not start")
	}
	if completed.Load() {
		t.Error("Function should have been canceled")
	}
}

func TestSafeGoNoError(t *testing.T) {
	ctx := context.Background()
	executed := atomic.Bool{}

	SafeGoNoError(ctx, 1*time.Second, "test task", func(ctx context.Context) {
		executed.Store(true)
	})

	// Wait for goroutine to complete
	time.Sleep(100 * time.Millisecond)

	if !executed.Load() {
		t.Error("SafeGoNoError did not execute function")
	}
}

func TestWorkerPool_Basic(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool(ctx, 2, "test pool", 1*time.Second)
	defer pool.Shutdown(1 * time.Second)

	executed := atomic.Int32{}
	for i := 0; i < 10; i++ {
		err := pool.Submit(func(ctx context.Context) error {
			executed.Add(1)
			return nil
		})
		if err != nil {
			t.Errorf("Failed to submit task: %v", err)
		}
	}

	// Wait for tasks to complete
	time.Sleep(200 * time.Millisecond)

	if executed.Load() != 10 {
		t.Errorf("Expected 10 executions, got %d", executed.Load())
	}
}

func TestWorkerPool_WithErrors(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool(ctx, 2, "test pool", 1*time.Second)
	defer pool.Shutdown(1 * time.Second)

	// Submit tasks that return errors
	for i := 0; i < 5; i++ {
		err := pool.Submit(func(ctx context.Context) error {
			return errors.New("test error")
		})
		if err != nil {
			t.Errorf("Failed to submit task: %v", err)
		}
	}

	// Wait for tasks to complete
	time.Sleep(200 * time.Millisecond)

	// Check errors channel
	errorCount := 0
	for {
		select {
		case <-pool.Errors():
			errorCount++
		default:
			goto done
		}
	}
done:

	if errorCount != 5 {
		t.Errorf("Expected 5 errors, got %d", errorCount)
	}
}

func TestWorkerPool_Shutdown(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool(ctx, 2, "test pool", 1*time.Second)

	executed := atomic.Int32{}
	for i := 0; i < 5; i++ {
		err := pool.Submit(func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			executed.Add(1)
			return nil
		})
		if err != nil {
			t.Errorf("Failed to submit task: %v", err)
		}
	}

	// Shutdown and wait
	err := pool.Shutdown(1 * time.Second)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// All tasks should have completed
	if executed.Load() != 5 {
		t.Errorf("Expected 5 executions, got %d", executed.Load())
	}

	// Submitting after shutdown should fail
	err = pool.Submit(func(ctx context.Context) error {
		return nil
	})
	if err == nil {
		t.Error("Expected error when submitting after shutdown")
	}
}

func TestWorkerPool_Timeout(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool(ctx, 1, "test pool", 50*time.Millisecond)
	defer pool.Shutdown(1 * time.Second)

	timedOut := atomic.Bool{}
	err := pool.Submit(func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			timedOut.Store(true)
			return ctx.Err()
		}
	})
	if err != nil {
		t.Errorf("Failed to submit task: %v", err)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	if !timedOut.Load() {
		t.Error("Task should have timed out")
	}
}

func TestBatch(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}
	executed := atomic.Int32{}

	errs := Batch(ctx, items, 2, "test batch", 1*time.Second, func(ctx context.Context, item int) error {
		executed.Add(1)
		return nil
	})

	if len(errs) > 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}

	if executed.Load() != 5 {
		t.Errorf("Expected 5 executions, got %d", executed.Load())
	}
}

func TestBatch_WithErrors(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}

	errs := Batch(ctx, items, 2, "test batch", 1*time.Second, func(ctx context.Context, item int) error {
		if item%2 == 0 {
			return errors.New("even number error")
		}
		return nil
	})

	// Should have 2 errors (items 2 and 4)
	if len(errs) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errs))
	}
}

func TestBatch_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	items := []int{1, 2, 3, 4, 5}
	executed := atomic.Int32{}

	// Cancel context immediately
	cancel()

	errs := Batch(ctx, items, 2, "test batch", 1*time.Second, func(ctx context.Context, item int) error {
		executed.Add(1)
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	// Should fail to submit tasks or execute very few
	if executed.Load() == 5 {
		t.Error("All tasks executed despite context cancellation")
	}

	// Should have at least one error
	if len(errs) == 0 {
		t.Error("Expected errors due to context cancellation")
	}
}
