package audit

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Logger is the interface for audit logging
type Logger interface {
	// Log logs an audit event
	Log(ctx context.Context, event *AuditEvent) error

	// LogAuthentication logs an authentication event
	LogAuthentication(ctx context.Context, eventType EventType, userID *int64, username string, status EventStatus, message string) error

	// LogAuthorization logs an authorization event
	LogAuthorization(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, status EventStatus, message string) error

	// LogDataMutation logs a data mutation event
	LogDataMutation(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, changes *ChangeDetails, message string) error

	// LogConfiguration logs a configuration change event
	LogConfiguration(ctx context.Context, eventType EventType, userID *int64, resourceID string, changes *ChangeDetails, message string) error

	// LogAdminAction logs an admin action event
	LogAdminAction(ctx context.Context, eventType EventType, adminUserID *int64, targetUserID *int64, message string) error

	// LogAccess logs a resource access event
	LogAccess(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, message string) error

	// LogHTTPRequest logs an HTTP request (for middleware)
	LogHTTPRequest(ctx context.Context, r *http.Request, statusCode int, duration time.Duration, err error) error

	// Close closes the logger and flushes any buffered logs
	Close() error
}

// contextKey is the type for context keys
type contextKey string

const (
	// AuditLoggerKey is the context key for the audit logger
	AuditLoggerKey contextKey = "audit_logger"

	// RequestStartTimeKey is the context key for request start time
	RequestStartTimeKey contextKey = "request_start_time"
)

// WithLogger adds an audit logger to the context
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, AuditLoggerKey, logger)
}

// FromContext retrieves the audit logger from context
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(AuditLoggerKey).(Logger); ok {
		return logger
	}
	// Return a no-op logger if none is set
	return &noOpLogger{}
}

// WithRequestStartTime adds the request start time to the context
func WithRequestStartTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, RequestStartTimeKey, t)
}

// GetRequestStartTime retrieves the request start time from context
func GetRequestStartTime(ctx context.Context) time.Time {
	if t, ok := ctx.Value(RequestStartTimeKey).(time.Time); ok {
		return t
	}
	return time.Now()
}

// noOpLogger is a logger that does nothing (used when no logger is configured)
type noOpLogger struct{}

func (l *noOpLogger) Log(ctx context.Context, event *AuditEvent) error {
	return nil
}

func (l *noOpLogger) LogAuthentication(ctx context.Context, eventType EventType, userID *int64, username string, status EventStatus, message string) error {
	return nil
}

func (l *noOpLogger) LogAuthorization(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, status EventStatus, message string) error {
	return nil
}

func (l *noOpLogger) LogDataMutation(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, changes *ChangeDetails, message string) error {
	return nil
}

func (l *noOpLogger) LogConfiguration(ctx context.Context, eventType EventType, userID *int64, resourceID string, changes *ChangeDetails, message string) error {
	return nil
}

func (l *noOpLogger) LogAdminAction(ctx context.Context, eventType EventType, adminUserID *int64, targetUserID *int64, message string) error {
	return nil
}

func (l *noOpLogger) LogAccess(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, message string) error {
	return nil
}

func (l *noOpLogger) LogHTTPRequest(ctx context.Context, r *http.Request, statusCode int, duration time.Duration, err error) error {
	return nil
}

func (l *noOpLogger) Close() error {
	return nil
}

// extractRequestInfo extracts common request information from context and HTTP request
func extractRequestInfo(ctx context.Context, r *http.Request) (userID *int64, username string, orgID *int64, tokenID *int64, ipAddress, userAgent, requestID string) {
	// Extract from observability context if available
	if reqID := getContextString(ctx, "request_id"); reqID != "" {
		requestID = reqID
	}

	// Extract IP address
	if r != nil {
		ipAddress = getClientIP(r)
		userAgent = r.UserAgent()
	}

	// Note: In real implementation, we'd extract user/org/token from auth context
	// This is a placeholder for the integration point
	return
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// getContextString safely extracts a string value from context
func getContextString(ctx context.Context, key string) string {
	if val := ctx.Value(contextKey(key)); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// buildBaseEvent creates a base audit event with common fields populated
func buildBaseEvent(ctx context.Context, r *http.Request, eventType EventType, status EventStatus) *AuditEvent {
	userID, username, orgID, tokenID, ipAddress, userAgent, requestID := extractRequestInfo(ctx, r)

	event := &AuditEvent{
		Timestamp:      time.Now().UTC(),
		EventType:      eventType,
		Status:         status,
		UserID:         userID,
		Username:       username,
		OrganizationID: orgID,
		TokenID:        tokenID,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		RequestID:      requestID,
		Metadata:       make(map[string]interface{}),
	}

	if r != nil {
		event.Method = r.Method
		event.Path = r.URL.Path
	}

	return event
}

// QuickLog is a convenience function for simple audit logging
func QuickLog(ctx context.Context, eventType EventType, status EventStatus, message string) error {
	logger := FromContext(ctx)
	event := &AuditEvent{
		Timestamp: time.Now().UTC(),
		EventType: eventType,
		Status:    status,
		Message:   message,
	}
	return logger.Log(ctx, event)
}

// LogSuccess logs a successful event with a message
func LogSuccess(ctx context.Context, eventType EventType, message string, metadata map[string]interface{}) error {
	logger := FromContext(ctx)
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.Message = message
	if metadata != nil {
		event.Metadata = metadata
	}
	return logger.Log(ctx, event)
}

// LogFailure logs a failed event with an error
func LogFailure(ctx context.Context, eventType EventType, message string, err error) error {
	logger := FromContext(ctx)
	event := buildBaseEvent(ctx, nil, eventType, EventStatusFailure)
	event.Message = message
	if err != nil {
		event.ErrorMessage = err.Error()
	}
	return logger.Log(ctx, event)
}

// LogDenied logs an access denied event
func LogDenied(ctx context.Context, eventType EventType, resourceType ResourceType, resourceID string, reason string) error {
	logger := FromContext(ctx)
	event := buildBaseEvent(ctx, nil, eventType, EventStatusDenied)
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.Message = fmt.Sprintf("Access denied: %s", reason)
	return logger.Log(ctx, event)
}
