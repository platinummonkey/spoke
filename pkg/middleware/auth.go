package middleware

import (
	"net/http"
	"strings"

	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/contextkeys"
)

// AuthContextKey is DEPRECATED: Use contextkeys.AuthKey instead
// This alias is provided for backward compatibility and will be removed in v2.0.0
type ContextKey = contextkeys.Key

const (
	// AuthContextKey is DEPRECATED: Use contextkeys.AuthKey instead
	AuthContextKey = contextkeys.AuthKey
)

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	tokenManager *auth.TokenManager
	optional     bool // If true, allow requests without auth
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(tokenManager *auth.TokenManager, optional bool) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager: tokenManager,
		optional:     optional,
	}
}

// Handler wraps an HTTP handler with authentication
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		// Format: "Bearer <token>"
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			if m.optional {
				// Continue without auth
				next.ServeHTTP(w, r)
				return
			}
			m.unauthorizedResponse(w, "missing authorization header")
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			m.unauthorizedResponse(w, "invalid authorization header format")
			return
		}

		token := parts[1]

		// Validate token
		apiToken, err := m.tokenManager.ValidateToken(token)
		if err != nil {
			m.unauthorizedResponse(w, "invalid or expired token")
			return
		}

		// TODO: Load user and organization details
		authCtx := &auth.AuthContext{
			Token:  apiToken,
			Scopes: apiToken.Scopes,
		}

		// Add auth context to request
		ctx := contextkeys.WithAuth(r.Context(), authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) unauthorizedResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + message + `"}`))
}

// GetAuthContext extracts auth context from request
func GetAuthContext(r *http.Request) *auth.AuthContext {
	ctx := r.Context().Value(contextkeys.AuthKey)
	if ctx == nil {
		return nil
	}
	authCtx, ok := ctx.(*auth.AuthContext)
	if !ok {
		return nil
	}
	return authCtx
}

// RequireScope creates middleware that checks for a specific scope
func RequireScope(scope auth.Scope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			if authCtx == nil {
				forbiddenResponse(w, "authentication required")
				return
			}

			if !authCtx.HasScope(scope) {
				forbiddenResponse(w, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates middleware that checks for a specific organization role
func RequireRole(role auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			if authCtx == nil {
				forbiddenResponse(w, "authentication required")
				return
			}

			if !authCtx.HasRole(role) {
				forbiddenResponse(w, "insufficient role permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireModulePermission creates middleware that checks module-level permissions
func RequireModulePermission(moduleIDParam string, perm auth.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			if authCtx == nil {
				forbiddenResponse(w, "authentication required")
				return
			}

			// TODO: Extract module ID from request params
			// TODO: Check if user has permission on module
			moduleID := int64(0) // Placeholder
			_ = moduleIDParam

			if !authCtx.HasPermission(moduleID, perm) {
				forbiddenResponse(w, "insufficient module permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func forbiddenResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
