// Package contextkeys provides centralized context key definitions
//
// IMPORTANT: All context keys used across the application must be defined here.
// This prevents typos, documents dependencies, and makes key usage discoverable.
//
// USAGE PATTERN:
//   import "github.com/platinummonkey/spoke/pkg/contextkeys"
//   ctx = context.WithValue(ctx, contextkeys.AuthKey, authCtx)
//   authCtx := ctx.Value(contextkeys.AuthKey).(*auth.AuthContext)
package contextkeys

import "context"

// Key is the type for context keys to prevent collisions
type Key string

const (
	// AuthKey contains *auth.AuthContext
	// Set by: middleware.AuthMiddleware (pkg/middleware/auth.go)
	// Required by: All protected API endpoints, RBAC middleware
	// Type: *auth.AuthContext
	AuthKey Key = "auth_context"

	// OrgKey contains *orgs.Organization
	// Set by: middleware.OrgContextMiddleware (pkg/middleware/org.go)
	// Required by: Org-scoped endpoints, quota middleware
	// Type: *orgs.Organization
	OrgKey Key = "organization"

	// RequestIDKey contains request ID string (UUID)
	// Set by: HTTP middleware, observability layer
	// Used by: Logger, audit trail, distributed tracing
	// Type: string
	RequestIDKey Key = "request_id"

	// UserIDKey contains user ID string
	// Set by: Auth middleware after user authentication
	// Used by: Logger, audit trail, user-scoped operations
	// Type: string
	UserIDKey Key = "user_id"

	// LoggerKey contains *observability.Logger
	// Set by: Observability middleware
	// Used by: Handlers that need structured logging with request context
	// Type: *observability.Logger
	LoggerKey Key = "logger"

	// AuditLoggerKey contains audit.Logger interface
	// Set by: Audit middleware (pkg/audit/middleware.go)
	// Used by: Handlers that record audit events
	// Type: audit.Logger
	AuditLoggerKey Key = "audit_logger"

	// RequestStartTimeKey contains request start timestamp
	// Set by: Audit middleware
	// Used by: Duration calculation for audit logs
	// Type: time.Time
	RequestStartTimeKey Key = "request_start_time"
)

// Helper functions for type-safe context operations

// WithAuth adds authentication context to the context
func WithAuth(ctx context.Context, authCtx interface{}) context.Context {
	return context.WithValue(ctx, AuthKey, authCtx)
}

// WithOrg adds organization to the context
func WithOrg(ctx context.Context, org interface{}) context.Context {
	return context.WithValue(ctx, OrgKey, org)
}

// WithRequestID adds request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithUserID adds user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithLogger adds logger to the context
func WithLogger(ctx context.Context, logger interface{}) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// WithAuditLogger adds audit logger to the context
func WithAuditLogger(ctx context.Context, logger interface{}) context.Context {
	return context.WithValue(ctx, AuditLoggerKey, logger)
}

// WithRequestStartTime adds request start time to the context
func WithRequestStartTime(ctx context.Context, startTime interface{}) context.Context {
	return context.WithValue(ctx, RequestStartTimeKey, startTime)
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}
