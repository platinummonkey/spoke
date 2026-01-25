package observability

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownManager handles graceful shutdown of services
type ShutdownManager struct {
	logger         *Logger
	server         *http.Server
	shutdownFuncs  []ShutdownFunc
	shutdownTimeout time.Duration
	mu             sync.Mutex
}

// ShutdownFunc is a function to call during shutdown
type ShutdownFunc func(context.Context) error

// NewShutdownManager creates a new shutdown manager
func NewShutdownManager(logger *Logger, server *http.Server, timeout time.Duration) *ShutdownManager {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &ShutdownManager{
		logger:          logger,
		server:          server,
		shutdownFuncs:   make([]ShutdownFunc, 0),
		shutdownTimeout: timeout,
	}
}

// RegisterShutdownFunc registers a function to call during shutdown
func (sm *ShutdownManager) RegisterShutdownFunc(fn ShutdownFunc) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.shutdownFuncs = append(sm.shutdownFuncs, fn)
}

// WaitForShutdown blocks until shutdown signal is received
func (sm *ShutdownManager) WaitForShutdown() error {
	// Create signal channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	sm.logger.Infof("Received signal %s, starting graceful shutdown", sig)

	// Create shutdown context with timeout
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

// GracefulShutdown performs a graceful shutdown
func GracefulShutdown(logger *Logger, server *http.Server, shutdownFuncs ...ShutdownFunc) error {
	manager := NewShutdownManager(logger, server, 30*time.Second)

	for _, fn := range shutdownFuncs {
		manager.RegisterShutdownFunc(fn)
	}

	return manager.WaitForShutdown()
}
