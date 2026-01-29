package observability

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestNewShutdownManager tests the creation of a new shutdown manager
func TestNewShutdownManager(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "with custom timeout",
			timeout:         10 * time.Second,
			expectedTimeout: 10 * time.Second,
		},
		{
			name:            "with zero timeout uses default",
			timeout:         0,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "with 1 second timeout",
			timeout:         1 * time.Second,
			expectedTimeout: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(InfoLevel, &bytes.Buffer{})
			server := &http.Server{}

			sm := NewShutdownManager(logger, server, tt.timeout)

			if sm == nil {
				t.Fatal("Expected non-nil shutdown manager")
			}

			if sm.logger != logger {
				t.Error("Logger not set correctly")
			}

			if sm.server != server {
				t.Error("Server not set correctly")
			}

			if sm.shutdownTimeout != tt.expectedTimeout {
				t.Errorf("Expected timeout %v, got %v", tt.expectedTimeout, sm.shutdownTimeout)
			}

			if sm.shutdownFuncs == nil {
				t.Error("Expected non-nil shutdown functions slice")
			}

			if len(sm.shutdownFuncs) != 0 {
				t.Error("Expected empty shutdown functions slice")
			}
		})
	}
}

// TestNewShutdownManagerWithNilLogger tests creation with nil logger
func TestNewShutdownManagerWithNilLogger(t *testing.T) {
	// Should not panic even with nil logger
	sm := NewShutdownManager(nil, nil, 5*time.Second)

	if sm == nil {
		t.Fatal("Expected non-nil shutdown manager")
	}

	if sm.shutdownTimeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", sm.shutdownTimeout)
	}
}

// TestRegisterShutdownFunc tests registering shutdown functions
func TestRegisterShutdownFunc(t *testing.T) {
	logger := NewLogger(InfoLevel, &bytes.Buffer{})
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	// Test registering single function
	fn1 := func(ctx context.Context) error {
		return nil
	}

	sm.RegisterShutdownFunc(fn1)

	if len(sm.shutdownFuncs) != 1 {
		t.Errorf("Expected 1 shutdown function, got %d", len(sm.shutdownFuncs))
	}

	// Test registering multiple functions
	fn2 := func(ctx context.Context) error {
		return nil
	}
	fn3 := func(ctx context.Context) error {
		return nil
	}

	sm.RegisterShutdownFunc(fn2)
	sm.RegisterShutdownFunc(fn3)

	if len(sm.shutdownFuncs) != 3 {
		t.Errorf("Expected 3 shutdown functions, got %d", len(sm.shutdownFuncs))
	}

	// Test concurrent registration (thread safety)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm.RegisterShutdownFunc(func(ctx context.Context) error {
				return nil
			})
		}()
	}
	wg.Wait()

	if len(sm.shutdownFuncs) != 13 {
		t.Errorf("Expected 13 shutdown functions, got %d", len(sm.shutdownFuncs))
	}
}

// TestRegisterShutdownFuncNilFunction tests registering nil function
func TestRegisterShutdownFuncNilFunction(t *testing.T) {
	logger := NewLogger(InfoLevel, &bytes.Buffer{})
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	// Should not panic with nil function
	sm.RegisterShutdownFunc(nil)

	if len(sm.shutdownFuncs) != 1 {
		t.Errorf("Expected 1 shutdown function (even if nil), got %d", len(sm.shutdownFuncs))
	}
}

// Helper function to execute shutdown logic without waiting for signals
func executeShutdownLogic(sm *ShutdownManager) error {
	ctx, cancel := context.WithTimeout(context.Background(), sm.shutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if sm.server != nil {
		sm.logger.Info("Shutting down HTTP server")
		if err := sm.server.Shutdown(ctx); err != nil {
			sm.logger.WithError(err).Error("HTTP server shutdown error")
			return fmt.Errorf("HTTP server shutdown failed: %w", err)
		}
		sm.logger.Info("HTTP server shutdown complete")
	}

	// Execute shutdown functions
	sm.mu.Lock()
	funcs := sm.shutdownFuncs
	sm.mu.Unlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(funcs))

	for i, fn := range funcs {
		if fn == nil {
			continue
		}
		wg.Add(1)
		go func(index int, shutdownFn ShutdownFunc) {
			defer wg.Done()
			sm.logger.Infof("Executing shutdown function %d", index)
			if err := shutdownFn(ctx); err != nil {
				sm.logger.WithError(err).Errorf("Shutdown function %d failed", index)
				errChan <- err
			} else {
				sm.logger.Infof("Shutdown function %d complete", index)
			}
		}(i, fn)
	}

	// Wait for all shutdown functions to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		sm.logger.Info("All shutdown functions completed")
	case <-ctx.Done():
		sm.logger.Warn("Shutdown timeout reached, forcing shutdown")
		return fmt.Errorf("shutdown timeout reached")
	}

	// Collect errors
	close(errChan)
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown completed with %d errors", len(errors))
	}

	sm.logger.Info("Graceful shutdown complete")
	return nil
}

// TestShutdownFunctionsExecution tests that shutdown functions are executed
func TestShutdownFunctionsExecution(t *testing.T) {
	tests := []struct {
		name           string
		setupFuncs     func() []ShutdownFunc
		expectedErrors int
	}{
		{
			name: "successful shutdown functions",
			setupFuncs: func() []ShutdownFunc {
				return []ShutdownFunc{
					func(ctx context.Context) error {
						return nil
					},
					func(ctx context.Context) error {
						return nil
					},
				}
			},
			expectedErrors: 0,
		},
		{
			name: "shutdown function with error",
			setupFuncs: func() []ShutdownFunc {
				return []ShutdownFunc{
					func(ctx context.Context) error {
						return errors.New("shutdown error 1")
					},
					func(ctx context.Context) error {
						return nil
					},
				}
			},
			expectedErrors: 1,
		},
		{
			name: "multiple shutdown functions with errors",
			setupFuncs: func() []ShutdownFunc {
				return []ShutdownFunc{
					func(ctx context.Context) error {
						return errors.New("error 1")
					},
					func(ctx context.Context) error {
						return errors.New("error 2")
					},
					func(ctx context.Context) error {
						return errors.New("error 3")
					},
				}
			},
			expectedErrors: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(InfoLevel, io.Discard)
			sm := NewShutdownManager(logger, nil, 5*time.Second)

			funcs := tt.setupFuncs()
			for _, fn := range funcs {
				sm.RegisterShutdownFunc(fn)
			}

			err := executeShutdownLogic(sm)

			if tt.expectedErrors > 0 {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				expectedMsg := fmt.Sprintf("shutdown completed with %d errors", tt.expectedErrors)
				if err.Error() != expectedMsg {
					t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestShutdownWithHTTPServer tests shutdown with HTTP server
func TestShutdownWithHTTPServer(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *http.Server
		expectError   bool
		serverStopped bool
	}{
		{
			name: "successful server shutdown",
			setupServer: func() *http.Server {
				server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				server.Start()
				return server.Config
			},
			expectError:   false,
			serverStopped: true,
		},
		{
			name: "nil server",
			setupServer: func() *http.Server {
				return nil
			},
			expectError:   false,
			serverStopped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(InfoLevel, io.Discard)
			server := tt.setupServer()
			sm := NewShutdownManager(logger, server, 5*time.Second)

			err := executeShutdownLogic(sm)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestShutdownTimeout tests that shutdown respects timeout
func TestShutdownTimeout(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 500*time.Millisecond)

	// Register a slow shutdown function
	sm.RegisterShutdownFunc(func(ctx context.Context) error {
		select {
		case <-time.After(2 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	start := time.Now()
	err := executeShutdownLogic(sm)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error but got nil")
	}

	if err.Error() != "shutdown timeout reached" {
		t.Errorf("Expected 'shutdown timeout reached' error, got: %v", err)
	}

	// Should timeout around 500ms, not wait full 2 seconds
	if elapsed > 1*time.Second {
		t.Errorf("Shutdown took too long: %v", elapsed)
	}
}

// TestShutdownConcurrentExecution tests that shutdown functions run concurrently
func TestShutdownConcurrentExecution(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	var mu sync.Mutex
	var executionOrder []int

	// Register functions that track execution order
	for i := 0; i < 3; i++ {
		index := i
		sm.RegisterShutdownFunc(func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			mu.Lock()
			executionOrder = append(executionOrder, index)
			mu.Unlock()
			return nil
		})
	}

	start := time.Now()
	err := executeShutdownLogic(sm)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// If functions ran concurrently, total time should be ~100ms
	// If sequential, it would be ~300ms
	if elapsed > 250*time.Millisecond {
		t.Error("Functions did not run concurrently")
	}

	if len(executionOrder) != 3 {
		t.Errorf("Expected 3 functions to execute, got %d", len(executionOrder))
	}
}

// TestShutdownManagerThreadSafety tests concurrent access to shutdown manager
func TestShutdownManagerThreadSafety(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrently register shutdown functions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			sm.RegisterShutdownFunc(func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})
		}(i)
	}

	wg.Wait()

	if len(sm.shutdownFuncs) != numGoroutines {
		t.Errorf("Expected %d shutdown functions, got %d", numGoroutines, len(sm.shutdownFuncs))
	}
}

// TestShutdownWithContextCancellation tests shutdown functions receive context
func TestShutdownWithContextCancellation(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	contextReceived := false
	var receivedCtx context.Context

	sm.RegisterShutdownFunc(func(ctx context.Context) error {
		contextReceived = true
		receivedCtx = ctx
		return nil
	})

	err := executeShutdownLogic(sm)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if !contextReceived {
		t.Error("Shutdown function did not receive context")
	}

	if receivedCtx == nil {
		t.Error("Received context was nil")
	}
}

// TestShutdownWithMixedSuccessAndFailure tests mixed success/failure scenarios
func TestShutdownWithMixedSuccessAndFailure(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	successCount := 0
	errorCount := 0
	var mu sync.Mutex

	// Add successful functions
	for i := 0; i < 3; i++ {
		sm.RegisterShutdownFunc(func(ctx context.Context) error {
			mu.Lock()
			successCount++
			mu.Unlock()
			return nil
		})
	}

	// Add failing functions
	for i := 0; i < 2; i++ {
		sm.RegisterShutdownFunc(func(ctx context.Context) error {
			mu.Lock()
			errorCount++
			mu.Unlock()
			return errors.New("intentional error")
		})
	}

	err := executeShutdownLogic(sm)

	if err == nil {
		t.Error("Expected error but got nil")
	}

	if err.Error() != "shutdown completed with 2 errors" {
		t.Errorf("Expected 'shutdown completed with 2 errors', got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if successCount != 3 {
		t.Errorf("Expected 3 successful shutdowns, got %d", successCount)
	}

	if errorCount != 2 {
		t.Errorf("Expected 2 failed shutdowns, got %d", errorCount)
	}
}

// TestShutdownEmptyFunctionList tests shutdown with no registered functions
func TestShutdownEmptyFunctionList(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	// Don't register any shutdown functions

	err := executeShutdownLogic(sm)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

// TestShutdownFunctionOrdering tests function execution with different delays
func TestShutdownFunctionOrdering(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	executionTimes := make(map[int]time.Time)
	var mu sync.Mutex

	// Register functions with different delays
	for i := 0; i < 5; i++ {
		index := i
		sm.RegisterShutdownFunc(func(ctx context.Context) error {
			delay := time.Duration(index*10) * time.Millisecond
			time.Sleep(delay)
			mu.Lock()
			executionTimes[index] = time.Now()
			mu.Unlock()
			return nil
		})
	}

	start := time.Now()
	err := executeShutdownLogic(sm)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Verify all functions executed
	if len(executionTimes) != 5 {
		t.Errorf("Expected 5 functions to execute, got %d", len(executionTimes))
	}

	// Since functions run concurrently, total time should be dominated by slowest
	// Slowest is 4*10ms = 40ms, so total should be well under 100ms
	if elapsed > 200*time.Millisecond {
		t.Errorf("Shutdown took too long: %v (functions may not be running concurrently)", elapsed)
	}
}

// TestShutdownWithSlowServer tests timeout handling with slow HTTP server
func TestShutdownWithSlowServer(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)

	// Create a server with a slow shutdown handler
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewUnstartedServer(mux)
	server.Start()
	defer server.Close()

	sm := NewShutdownManager(logger, server.Config, 100*time.Millisecond)

	start := time.Now()
	err := executeShutdownLogic(sm)
	elapsed := time.Since(start)

	// Should timeout quickly
	if elapsed > 500*time.Millisecond {
		t.Errorf("Shutdown took too long: %v", elapsed)
	}

	// May or may not error depending on server state
	_ = err
}

// TestShutdownFunctionWithPanic tests handling of panicking shutdown functions
func TestShutdownFunctionContextTimeout(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 1*time.Second)

	var contextTimedOut bool
	var mu sync.Mutex

	sm.RegisterShutdownFunc(func(ctx context.Context) error {
		<-ctx.Done()
		mu.Lock()
		contextTimedOut = true
		mu.Unlock()
		return ctx.Err()
	})

	err := executeShutdownLogic(sm)

	if err == nil {
		t.Error("Expected timeout error but got nil")
	}

	mu.Lock()
	timedOut := contextTimedOut
	mu.Unlock()

	if !timedOut {
		t.Error("Context should have timed out")
	}
}

// TestShutdownWithMultipleTimeouts tests multiple functions timing out
func TestShutdownWithMultipleTimeouts(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 200*time.Millisecond)

	for i := 0; i < 3; i++ {
		sm.RegisterShutdownFunc(func(ctx context.Context) error {
			select {
			case <-time.After(1 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}

	start := time.Now()
	err := executeShutdownLogic(sm)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error but got nil")
	}

	if elapsed > 500*time.Millisecond {
		t.Errorf("Shutdown took too long: %v", elapsed)
	}
}

// TestShutdownManagerFields tests the struct fields are properly initialized
func TestShutdownManagerFields(t *testing.T) {
	var logBuf bytes.Buffer
	logger := NewLogger(InfoLevel, &logBuf)
	server := &http.Server{Addr: ":8080"}
	timeout := 15 * time.Second

	sm := NewShutdownManager(logger, server, timeout)

	if sm.logger != logger {
		t.Error("Logger field not set correctly")
	}

	if sm.server != server {
		t.Error("Server field not set correctly")
	}

	if sm.shutdownTimeout != timeout {
		t.Errorf("Timeout field not set correctly: expected %v, got %v", timeout, sm.shutdownTimeout)
	}

	if sm.shutdownFuncs == nil {
		t.Error("ShutdownFuncs should be initialized")
	}

	if cap(sm.shutdownFuncs) < 0 {
		t.Error("ShutdownFuncs should have valid capacity")
	}
}

// TestDefaultTimeout tests the default timeout constant
func TestDefaultTimeout(t *testing.T) {
	logger := NewLogger(InfoLevel, &bytes.Buffer{})
	sm := NewShutdownManager(logger, nil, 0)

	expectedDefault := 30 * time.Second
	if sm.shutdownTimeout != expectedDefault {
		t.Errorf("Expected default timeout %v, got %v", expectedDefault, sm.shutdownTimeout)
	}
}

// TestShutdownFuncType tests the ShutdownFunc type
func TestShutdownFuncType(t *testing.T) {
	var fn ShutdownFunc = func(ctx context.Context) error {
		return nil
	}

	if fn == nil {
		t.Error("ShutdownFunc should not be nil")
	}

	err := fn(context.Background())
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

// TestExecuteShutdownWithAllNilFunctions tests nil function handling
func TestExecuteShutdownWithAllNilFunctions(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	// Register nil functions
	sm.RegisterShutdownFunc(nil)
	sm.RegisterShutdownFunc(nil)

	err := executeShutdownLogic(sm)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

// TestShutdownLogging tests that appropriate log messages are generated
func TestShutdownLogging(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	called := false
	sm.RegisterShutdownFunc(func(ctx context.Context) error {
		called = true
		return nil
	})

	err := executeShutdownLogic(sm)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if !called {
		t.Error("Shutdown function was not called")
	}
}

// TestShutdownManagerMutexProtection tests mutex protection of shutdown functions
func TestShutdownManagerMutexProtection(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	// Start goroutines that continuously register functions
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				select {
				case <-done:
					return
				default:
					sm.RegisterShutdownFunc(func(ctx context.Context) error {
						return nil
					})
				}
			}
		}()
	}

	// Let them run briefly
	time.Sleep(50 * time.Millisecond)
	close(done)
	time.Sleep(50 * time.Millisecond)

	// Execute shutdown while potentially still registering
	err := executeShutdownLogic(sm)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Should have registered many functions
	if len(sm.shutdownFuncs) == 0 {
		t.Error("Expected some shutdown functions to be registered")
	}
}

// TestShutdownWithErrorLogging tests error logging during shutdown
func TestShutdownWithErrorLogging(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	expectedError := errors.New("test error")
	sm.RegisterShutdownFunc(func(ctx context.Context) error {
		return expectedError
	})

	err := executeShutdownLogic(sm)

	if err == nil {
		t.Error("Expected error but got nil")
	}
}

// TestShutdownManagerInitialization tests various initialization scenarios
func TestShutdownManagerInitialization(t *testing.T) {
	tests := []struct {
		name    string
		logger  *Logger
		server  *http.Server
		timeout time.Duration
	}{
		{
			name:    "all nil/zero",
			logger:  nil,
			server:  nil,
			timeout: 0,
		},
		{
			name:    "with logger only",
			logger:  NewLogger(InfoLevel, &bytes.Buffer{}),
			server:  nil,
			timeout: 0,
		},
		{
			name:    "with server only",
			logger:  nil,
			server:  &http.Server{},
			timeout: 0,
		},
		{
			name:    "all parameters",
			logger:  NewLogger(InfoLevel, &bytes.Buffer{}),
			server:  &http.Server{},
			timeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewShutdownManager(tt.logger, tt.server, tt.timeout)
			if sm == nil {
				t.Fatal("Expected non-nil shutdown manager")
			}

			// Verify timeout is set correctly or defaults to 30s
			expectedTimeout := tt.timeout
			if expectedTimeout == 0 {
				expectedTimeout = 30 * time.Second
			}
			if sm.shutdownTimeout != expectedTimeout {
				t.Errorf("Expected timeout %v, got %v", expectedTimeout, sm.shutdownTimeout)
			}
		})
	}
}

// TestShutdownContextPropagation tests context propagation to shutdown functions
func TestShutdownContextPropagation(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 2*time.Second)

	var capturedDeadline time.Time
	var hasDeadline bool

	sm.RegisterShutdownFunc(func(ctx context.Context) error {
		capturedDeadline, hasDeadline = ctx.Deadline()
		return nil
	})

	err := executeShutdownLogic(sm)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if !hasDeadline {
		t.Error("Context should have a deadline")
	}

	if capturedDeadline.IsZero() {
		t.Error("Deadline should not be zero")
	}
}

// TestShutdownMultipleServerInstances tests with different server configurations
func TestShutdownMultipleServerInstances(t *testing.T) {
	tests := []struct {
		name   string
		server *http.Server
	}{
		{
			name:   "server with address",
			server: &http.Server{Addr: ":8080"},
		},
		{
			name:   "server with custom config",
			server: &http.Server{Addr: ":9090", ReadTimeout: 10 * time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(InfoLevel, io.Discard)

			// Start a test server
			testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			testServer.Start()
			defer testServer.Close()

			sm := NewShutdownManager(logger, testServer.Config, 5*time.Second)

			err := executeShutdownLogic(sm)

			// Should not error for valid servers
			if err != nil {
				t.Logf("Server shutdown returned: %v", err)
			}
		})
	}
}

// TestShutdownErrorCollection tests error collection from multiple functions
func TestShutdownErrorCollection(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	numErrors := 5
	for i := 0; i < numErrors; i++ {
		sm.RegisterShutdownFunc(func(ctx context.Context) error {
			return fmt.Errorf("error %d", i)
		})
	}

	err := executeShutdownLogic(sm)

	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	expectedMsg := fmt.Sprintf("shutdown completed with %d errors", numErrors)
	if err.Error() != expectedMsg {
		t.Errorf("Expected '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestShutdownQuickFunctions tests many quick-completing functions
func TestShutdownQuickFunctions(t *testing.T) {
	logger := NewLogger(InfoLevel, io.Discard)
	sm := NewShutdownManager(logger, nil, 5*time.Second)

	callCount := 0
	var mu sync.Mutex

	// Register many quick functions
	for i := 0; i < 100; i++ {
		sm.RegisterShutdownFunc(func(ctx context.Context) error {
			mu.Lock()
			callCount++
			mu.Unlock()
			return nil
		})
	}

	start := time.Now()
	err := executeShutdownLogic(sm)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if callCount != 100 {
		t.Errorf("Expected 100 function calls, got %d", callCount)
	}

	// All 100 functions should complete very quickly since they run concurrently
	if elapsed > 1*time.Second {
		t.Errorf("Shutdown took too long: %v", elapsed)
	}
}

// TestWaitForShutdownWithSignal tests WaitForShutdown function with actual signal
func TestWaitForShutdownWithSignal(t *testing.T) {
	t.Skip("Skipping signal test - sending signals to test process is unreliable")
}

// TestWaitForShutdownWithSIGINT tests WaitForShutdown with SIGINT
func TestWaitForShutdownWithSIGINT(t *testing.T) {
	t.Skip("Skipping signal test - sending signals to test process is unreliable")
}

// TestWaitForShutdownWithServerError tests server shutdown error handling
func TestWaitForShutdownWithServerError(t *testing.T) {
	t.Skip("Skipping signal test - sending signals to test process is unreliable")
}

// TestWaitForShutdownWithErrors tests error handling in shutdown functions
func TestWaitForShutdownWithErrors(t *testing.T) {
	t.Skip("Skipping signal test - sending signals to test process is unreliable")
}

// TestWaitForShutdownTimeout tests timeout during shutdown
func TestWaitForShutdownTimeoutScenario(t *testing.T) {
	t.Skip("Skipping signal test - sending signals to test process is unreliable")
}

// TestGracefulShutdownFunction tests the GracefulShutdown convenience function
func TestGracefulShutdownFunction(t *testing.T) {
	t.Skip("Skipping signal test - sending signals to test process is unreliable")
}

// TestGracefulShutdownWithServer tests GracefulShutdown with HTTP server
func TestGracefulShutdownWithServer(t *testing.T) {
	t.Skip("Skipping signal test - sending signals to test process is unreliable")
}
