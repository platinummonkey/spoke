package rbac

import (
	"context"
	"net/http"

	"github.com/platinummonkey/spoke/pkg/middleware"
)

// PermissionMiddleware provides middleware for permission checking
type PermissionMiddleware struct {
	checker *PermissionChecker
}

// NewPermissionMiddleware creates a new permission middleware
func NewPermissionMiddleware(checker *PermissionChecker) *PermissionMiddleware {
	return &PermissionMiddleware{
		checker: checker,
	}
}

// RequirePermission creates middleware that requires a specific permission
func (pm *PermissionMiddleware) RequirePermission(resource Resource, action Action, scope PermissionScope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := middleware.GetAuthContext(r)
			if authCtx == nil || authCtx.User == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Build permission check
			check := PermissionCheck{
				UserID: authCtx.User.ID,
				Permission: Permission{
					Resource: resource,
					Action:   action,
				},
				Scope: scope,
			}

			// Add organization context if available
			if authCtx.Organization != nil {
				check.OrganizationID = &authCtx.Organization.ID
			}

			// Check permission
			result, err := pm.checker.CheckPermission(r.Context(), check)
			if err != nil {
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireModulePermission creates middleware that checks module-specific permissions
func (pm *PermissionMiddleware) RequireModulePermission(action Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := middleware.GetAuthContext(r)
			if authCtx == nil || authCtx.User == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Extract module name from request context or URL
			moduleName := extractModuleFromRequest(r)
			if moduleName == "" {
				http.Error(w, "Module name required", http.StatusBadRequest)
				return
			}

			// Build permission check
			check := PermissionCheck{
				UserID: authCtx.User.ID,
				Permission: Permission{
					Resource: ResourceModule,
					Action:   action,
				},
				Scope:      ScopeModule,
				ResourceID: &moduleName,
			}

			// Add organization context if available
			if authCtx.Organization != nil {
				check.OrganizationID = &authCtx.Organization.ID
			}

			// Check permission
			result, err := pm.checker.CheckPermission(r.Context(), check)
			if err != nil {
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				http.Error(w, "Insufficient permissions for this module", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission creates middleware that requires any of the specified permissions
func (pm *PermissionMiddleware) RequireAnyPermission(permissions ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := middleware.GetAuthContext(r)
			if authCtx == nil || authCtx.User == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check each permission
			allowed := false
			for _, perm := range permissions {
				check := PermissionCheck{
					UserID:     authCtx.User.ID,
					Permission: perm,
					Scope:      ScopeOrganization,
				}

				if authCtx.Organization != nil {
					check.OrganizationID = &authCtx.Organization.ID
				}

				result, err := pm.checker.CheckPermission(r.Context(), check)
				if err == nil && result.Allowed {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAllPermissions creates middleware that requires all of the specified permissions
func (pm *PermissionMiddleware) RequireAllPermissions(permissions ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := middleware.GetAuthContext(r)
			if authCtx == nil || authCtx.User == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check each permission
			for _, perm := range permissions {
				check := PermissionCheck{
					UserID:     authCtx.User.ID,
					Permission: perm,
					Scope:      ScopeOrganization,
				}

				if authCtx.Organization != nil {
					check.OrganizationID = &authCtx.Organization.ID
				}

				result, err := pm.checker.CheckPermission(r.Context(), check)
				if err != nil || !result.Allowed {
					http.Error(w, "Insufficient permissions", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates middleware that requires a specific role
func (pm *PermissionMiddleware) RequireRole(roleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := middleware.GetAuthContext(r)
			if authCtx == nil || authCtx.User == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Get user roles
			var organizationID *int64
			if authCtx.Organization != nil {
				organizationID = &authCtx.Organization.ID
			}

			roles, err := pm.checker.GetUserRoles(r.Context(), authCtx.User.ID, organizationID)
			if err != nil {
				http.Error(w, "Failed to check roles", http.StatusInternalServerError)
				return
			}

			// Check if user has the required role
			hasRole := false
			for _, role := range roles {
				if role.Name == roleName {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, "Required role not found", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractModuleFromRequest extracts the module name from the request
func extractModuleFromRequest(r *http.Request) string {
	// Try to get from context first
	if moduleName := r.Context().Value("module_name"); moduleName != nil {
		if name, ok := moduleName.(string); ok {
			return name
		}
	}

	// Try to get from URL path variable (gorilla/mux)
	// This would need to be implemented based on your routing
	// For now, return empty string
	return ""
}

// WithModuleName adds module name to request context
func WithModuleName(ctx context.Context, moduleName string) context.Context {
	return context.WithValue(ctx, "module_name", moduleName)
}
