package audit

import (
	"context"
	"net/http"
	"time"
)

// Middleware provides HTTP middleware for audit logging
type Middleware struct {
	logger       Logger
	logAllRequests bool // If false, only log mutations and sensitive operations
}

// NewMiddleware creates a new audit middleware
func NewMiddleware(logger Logger, logAllRequests bool) *Middleware {
	return &Middleware{
		logger:       logger,
		logAllRequests: logAllRequests,
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Handler wraps an HTTP handler with audit logging
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record start time
		startTime := time.Now()

		// Add logger and start time to context
		ctx := WithLogger(r.Context(), m.logger)
		ctx = WithRequestStartTime(ctx, startTime)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Serve the request
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Calculate duration
		duration := time.Since(startTime)

		// Determine if we should log this request
		shouldLog := m.logAllRequests || m.shouldLogRequest(r, wrapped.statusCode)

		if shouldLog {
			// Log the request
			if err := m.logger.LogHTTPRequest(ctx, r, wrapped.statusCode, duration, nil); err != nil {
				// Log error but don't fail the request
				// In production, you might want to send this to a separate error logging system
				_ = err
			}
		}
	})
}

// shouldLogRequest determines if a request should be logged
func (m *Middleware) shouldLogRequest(r *http.Request, statusCode int) bool {
	// Always log mutations (POST, PUT, PATCH, DELETE)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return true
	}

	// Always log errors and denials
	if statusCode >= 400 {
		return true
	}

	// Log access to sensitive endpoints
	if m.isSensitiveEndpoint(r.URL.Path) {
		return true
	}

	return false
}

// isSensitiveEndpoint checks if an endpoint is considered sensitive
func (m *Middleware) isSensitiveEndpoint(path string) bool {
	// Check for auth-related endpoints
	if len(path) >= 5 && path[:5] == "/auth" {
		return true
	}

	// Check for admin endpoints
	if len(path) >= 6 && path[:6] == "/admin" {
		return true
	}

	// Check for audit endpoints
	if len(path) >= 6 && path[:6] == "/audit" {
		return true
	}

	// Check for config endpoints
	if len(path) >= 7 && path[:7] == "/config" {
		return true
	}

	return false
}

// LogSuccess is a helper for logging successful operations from handlers
func (m *Middleware) LogSuccess(ctx context.Context, eventType EventType, message string, metadata map[string]interface{}) error {
	return LogSuccess(ctx, eventType, message, metadata)
}

// LogFailure is a helper for logging failed operations from handlers
func (m *Middleware) LogFailure(ctx context.Context, eventType EventType, message string, err error) error {
	return LogFailure(ctx, eventType, message, err)
}

// LogDenied is a helper for logging access denied from handlers
func (m *Middleware) LogDenied(ctx context.Context, eventType EventType, resourceType ResourceType, resourceID string, reason string) error {
	return LogDenied(ctx, eventType, resourceType, resourceID, reason)
}

// WithAuditContext adds audit-relevant information to the context
func WithAuditContext(ctx context.Context, userID *int64, username string, orgID *int64, tokenID *int64) context.Context {
	if userID != nil {
		ctx = context.WithValue(ctx, contextKey("audit_user_id"), *userID)
	}
	if username != "" {
		ctx = context.WithValue(ctx, contextKey("audit_username"), username)
	}
	if orgID != nil {
		ctx = context.WithValue(ctx, contextKey("audit_org_id"), *orgID)
	}
	if tokenID != nil {
		ctx = context.WithValue(ctx, contextKey("audit_token_id"), *tokenID)
	}
	return ctx
}

// GetAuditContext retrieves audit context from the request context
func GetAuditContext(ctx context.Context) (userID *int64, username string, orgID *int64, tokenID *int64) {
	if val := ctx.Value(contextKey("audit_user_id")); val != nil {
		if id, ok := val.(int64); ok {
			userID = &id
		}
	}
	if val := ctx.Value(contextKey("audit_username")); val != nil {
		if name, ok := val.(string); ok {
			username = name
		}
	}
	if val := ctx.Value(contextKey("audit_org_id")); val != nil {
		if id, ok := val.(int64); ok {
			orgID = &id
		}
	}
	if val := ctx.Value(contextKey("audit_token_id")); val != nil {
		if id, ok := val.(int64); ok {
			tokenID = &id
		}
	}
	return
}
