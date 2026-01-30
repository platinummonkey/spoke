package observability

import (
	"fmt"
	"runtime/debug"
)

// RecoverPanic recovers from a panic and logs it with structured logging
//
// Usage in defer statements:
//
//	func riskyOperation() {
//	    defer observability.RecoverPanic(logger, "risky operation")
//	    // ... code that might panic
//	}
//
// The function should be called in a defer statement. If a panic occurs,
// it will be recovered and logged at Error level with:
//   - panic value
//   - full stack trace
//   - context about where the panic occurred
//
// After logging, the panic is NOT re-raised - the function returns normally.
// This prevents the panic from crashing the process but may leave the system
// in an inconsistent state. Use carefully.
func RecoverPanic(logger *Logger, context string) {
	if r := recover(); r != nil {
		logger.WithField("panic", r).
			WithField("stack", string(debug.Stack())).
			WithField("context", context).
			Error("PANIC recovered")
	}
}

// RecoverPanicWithCallback recovers from a panic, logs it, and executes a callback
//
// Usage when cleanup is needed after panic:
//
//	func worker() {
//	    defer observability.RecoverPanicWithCallback(logger, "worker goroutine", func() {
//	        close(resultCh)  // Cleanup action
//	    })
//	    // ... code that might panic
//	}
//
// The callback is executed AFTER logging the panic, regardless of whether
// a panic occurred. This allows cleanup actions like closing channels,
// releasing locks, or updating state.
//
// Common use cases:
//   - Close channels to unblock waiting goroutines
//   - Release mutex locks to prevent deadlock
//   - Set error flags to indicate failure
//   - Update metrics counters
func RecoverPanicWithCallback(logger *Logger, context string, callback func()) {
	if r := recover(); r != nil {
		logger.WithField("panic", r).
			WithField("stack", string(debug.Stack())).
			WithField("context", context).
			Error("PANIC recovered")
		if callback != nil {
			callback()
		}
	}
}

// MustRecover recovers from a panic and converts it to an error
//
// Usage when you want to convert panics to errors:
//
//	func parseData() (result Data, err error) {
//	    defer func() {
//	        err = observability.MustRecover(recover())
//	    }()
//	    // ... code that might panic
//	    return data, nil
//	}
//
// If a panic occurred, returns an error describing the panic.
// If no panic (r is nil), returns nil.
//
// This is useful when:
//   - Calling third-party code that might panic
//   - You want to treat panics as errors in your API
//   - You need to propagate the failure up the call stack
//
// Note: The stack trace is NOT included in the error - use RecoverPanic
// for structured logging with full stack traces.
func MustRecover(r interface{}) error {
	if r != nil {
		return fmt.Errorf("panic: %v", r)
	}
	return nil
}
