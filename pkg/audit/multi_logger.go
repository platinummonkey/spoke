package audit

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// MultiLogger logs to multiple audit loggers simultaneously
type MultiLogger struct {
	loggers []Logger
	async   bool // If true, log asynchronously
	wg      sync.WaitGroup
	errChan chan error
}

// NewMultiLogger creates a new multi-logger that writes to multiple destinations
func NewMultiLogger(loggers ...Logger) *MultiLogger {
	return &MultiLogger{
		loggers: loggers,
		async:   true,
		errChan: make(chan error, len(loggers)),
	}
}

// SetAsync sets whether logging should be asynchronous
func (m *MultiLogger) SetAsync(async bool) {
	m.async = async
}

// Log logs an audit event to all configured loggers
func (m *MultiLogger) Log(ctx context.Context, event *AuditEvent) error {
	if len(m.loggers) == 0 {
		return nil
	}

	if m.async {
		return m.logAsync(ctx, event)
	}

	return m.logSync(ctx, event)
}

// logSync logs synchronously to all loggers
func (m *MultiLogger) logSync(ctx context.Context, event *AuditEvent) error {
	var firstErr error

	for _, logger := range m.loggers {
		if err := logger.Log(ctx, event); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			// Continue logging to other loggers even if one fails
		}
	}

	return firstErr
}

// logAsync logs asynchronously to all loggers
func (m *MultiLogger) logAsync(ctx context.Context, event *AuditEvent) error {
	for _, logger := range m.loggers {
		m.wg.Add(1)
		go func(l Logger) {
			defer m.wg.Done()
			if err := l.Log(ctx, event); err != nil {
				select {
				case m.errChan <- err:
				default:
					// Channel full, drop error
				}
			}
		}(logger)
	}

	return nil
}

// LogAuthentication logs an authentication event
func (m *MultiLogger) LogAuthentication(ctx context.Context, eventType EventType, userID *int64, username string, status EventStatus, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, status)
	event.UserID = userID
	event.Username = username
	event.Message = message
	event.ResourceType = ResourceTypeUser

	return m.Log(ctx, event)
}

// LogAuthorization logs an authorization event
func (m *MultiLogger) LogAuthorization(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, status EventStatus, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, status)
	event.UserID = userID
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.Message = message

	return m.Log(ctx, event)
}

// LogDataMutation logs a data mutation event
func (m *MultiLogger) LogDataMutation(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, changes *ChangeDetails, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = userID
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.Changes = changes
	event.Message = message

	return m.Log(ctx, event)
}

// LogConfiguration logs a configuration change event
func (m *MultiLogger) LogConfiguration(ctx context.Context, eventType EventType, userID *int64, resourceID string, changes *ChangeDetails, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = userID
	event.ResourceType = ResourceTypeConfig
	event.ResourceID = resourceID
	event.Changes = changes
	event.Message = message

	return m.Log(ctx, event)
}

// LogAdminAction logs an admin action event
func (m *MultiLogger) LogAdminAction(ctx context.Context, eventType EventType, adminUserID *int64, targetUserID *int64, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = adminUserID
	event.Message = message
	if targetUserID != nil {
		event.Metadata["target_user_id"] = *targetUserID
	}

	return m.Log(ctx, event)
}

// LogAccess logs a resource access event
func (m *MultiLogger) LogAccess(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = userID
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.Message = message

	return m.Log(ctx, event)
}

// LogHTTPRequest logs an HTTP request
func (m *MultiLogger) LogHTTPRequest(ctx context.Context, r *http.Request, statusCode int, duration time.Duration, err error) error {
	eventType := EventTypeAccessModuleRead
	status := EventStatusSuccess

	if statusCode >= 400 {
		status = EventStatusFailure
	}
	if statusCode == 403 {
		status = EventStatusDenied
	}

	event := buildBaseEvent(ctx, r, eventType, status)
	event.StatusCode = statusCode
	event.Metadata["duration_ms"] = duration.Milliseconds()

	if err != nil {
		event.ErrorMessage = err.Error()
	}

	return m.Log(ctx, event)
}

// Wait waits for all async logging operations to complete
func (m *MultiLogger) Wait() {
	m.wg.Wait()
}

// GetErrors returns any errors that occurred during async logging
func (m *MultiLogger) GetErrors() []error {
	var errors []error
	for {
		select {
		case err := <-m.errChan:
			errors = append(errors, err)
		default:
			return errors
		}
	}
}

// Close closes all loggers
func (m *MultiLogger) Close() error {
	// Wait for any pending async operations
	m.wg.Wait()

	var firstErr error
	for _, logger := range m.loggers {
		if err := logger.Close(); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to close logger: %w", err)
			}
		}
	}

	close(m.errChan)
	return firstErr
}
