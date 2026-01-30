// Package middleware provides HTTP middleware for quota enforcement
//
// # CRITICAL: Middleware Ordering Requirements
//
// Quota middleware has strict ordering dependencies. Incorrect order will cause
// quota checks to silently fail (returning 0 for org ID) or panic.
//
// REQUIRED ORDERING (outer to inner):
//  1. AuthMiddleware - Sets auth context with user and organization info
//  2. OrgContextMiddleware - Extracts org ID from auth context
//  3. Quota check middleware - CheckAPIRateLimit, EnforceModuleQuota, etc.
//
// Example (correct):
//
//	router.Use(authMiddleware.Handler)           // 1. Sets auth context
//	router.Use(quotaMiddleware.OrgContextMiddleware)  // 2. Extracts org ID
//	router.HandleFunc("/api/modules", handler).
//	    Methods("POST").
//	    Handler(quotaMiddleware.EnforceModuleQuota(...))  // 3. Checks quota
//
// Example (WRONG - will not work):
//
//	router.Use(quotaMiddleware.EnforceModuleQuota(...))  // FAILS: No org ID in context yet
//	router.Use(quotaMiddleware.OrgContextMiddleware)
//	router.Use(authMiddleware.Handler)
//
// WHY THIS MATTERS:
//   - If quota middleware runs before OrgContextMiddleware, getOrgIDFromContext()
//     returns 0, and quota checks are silently skipped (security vulnerability)
//   - If OrgContextMiddleware runs before AuthMiddleware, it cannot extract
//     organization info (auth context is nil)
//
// See pkg/middleware/auth.go for AuthMiddleware documentation
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/platinummonkey/spoke/pkg/async"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/orgs"
)

// QuotaMiddleware enforces quotas for API requests
//
// IMPORTANT: See package documentation for middleware ordering requirements.
// Quota middleware will not work correctly if ordering is wrong.
type QuotaMiddleware struct {
	orgService orgs.Service
}

// NewQuotaMiddleware creates a new QuotaMiddleware
func NewQuotaMiddleware(orgService orgs.Service) *QuotaMiddleware {
	return &QuotaMiddleware{
		orgService: orgService,
	}
}

// CheckAPIRateLimit checks if the organization is within API rate limits
//
// REQUIRES: OrgContextMiddleware must run before this middleware
// Returns: 429 Too Many Requests if rate limit exceeded
//
// If org_id is not in context (OrgContextMiddleware not run), quota check is skipped.
func (m *QuotaMiddleware) CheckAPIRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get organization from context
		orgID := getOrgIDFromContext(r.Context())
		if orgID == 0 {
			// No org context, skip quota check
			next.ServeHTTP(w, r)
			return
		}

		// Check rate limit
		if err := m.orgService.CheckAPIRateLimit(orgID); err != nil {
			if orgs.IsQuotaExceeded(err) {
				http.Error(w, "API rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			// Log error but don't block request
		}

		// Increment API request counter
		go m.orgService.IncrementAPIRequests(orgID)

		next.ServeHTTP(w, r)
	})
}

// EnforceModuleQuota checks if organization can create a new module
//
// REQUIRES: OrgContextMiddleware must run before this middleware
// Returns: 403 Forbidden if quota exceeded
//
// If org_id is not in context (OrgContextMiddleware not run), quota check is skipped.
func (m *QuotaMiddleware) EnforceModuleQuota(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := getOrgIDFromContext(r.Context())
		if orgID == 0 {
			next.ServeHTTP(w, r)
			return
		}

		if err := m.orgService.CheckModuleQuota(orgID); err != nil {
			if orgs.IsQuotaExceeded(err) {
				qe := err.(*orgs.QuotaExceededError)
				http.Error(w, qe.Error(), http.StatusForbidden)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// EnforceCompileJobQuota checks if organization can run a compile job
//
// REQUIRES: OrgContextMiddleware must run before this middleware
// Returns: 403 Forbidden if quota exceeded
//
// If org_id is not in context (OrgContextMiddleware not run), quota check is skipped.
func (m *QuotaMiddleware) EnforceCompileJobQuota(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := getOrgIDFromContext(r.Context())
		if orgID == 0 {
			next.ServeHTTP(w, r)
			return
		}

		if err := m.orgService.CheckCompileJobQuota(orgID); err != nil {
			if orgs.IsQuotaExceeded(err) {
				qe := err.(*orgs.QuotaExceededError)
				http.Error(w, qe.Error(), http.StatusForbidden)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// OrgContextMiddleware extracts organization ID from auth context and adds it to request context
//
// MIDDLEWARE ORDERING REQUIREMENT:
//   - MUST run after AuthMiddleware (requires auth context to be set)
//   - MUST run before any quota check middleware (they need org_id in context)
//
// Dependencies:
//   - Reads: auth context (set by AuthMiddleware)
//   - Sets: "org_id" in context (used by quota check middleware)
//
// If auth context is missing or has no organization, silently continues without
// setting org_id (quota checks will be skipped downstream).
func (m *QuotaMiddleware) OrgContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx, ok := r.Context().Value("auth").(*auth.AuthContext)
		if !ok || authCtx == nil || authCtx.Organization == nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), "org_id", authCtx.Organization.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getOrgIDFromContext retrieves organization ID from context
func getOrgIDFromContext(ctx context.Context) int64 {
	orgID, ok := ctx.Value("org_id").(int64)
	if !ok {
		return 0
	}
	return orgID
}

// IncrementModuleUsage increments module usage for an organization
// Context-aware version that properly manages goroutine lifecycle
func IncrementModuleUsage(ctx context.Context, orgService orgs.Service, orgID int64) {
	async.SafeGo(ctx, 5*time.Second, "increment module usage", func(ctx context.Context) error {
		return orgService.IncrementModules(orgID)
	})
}

// IncrementVersionUsage increments version usage for an organization
// Context-aware version that properly manages goroutine lifecycle
func IncrementVersionUsage(ctx context.Context, orgService orgs.Service, orgID int64) {
	async.SafeGo(ctx, 5*time.Second, "increment version usage", func(ctx context.Context) error {
		return orgService.IncrementVersions(orgID)
	})
}

// IncrementStorageUsage increments storage usage for an organization
// Context-aware version that properly manages goroutine lifecycle
func IncrementStorageUsage(ctx context.Context, orgService orgs.Service, orgID int64, bytes int64) {
	async.SafeGo(ctx, 5*time.Second, "increment storage usage", func(ctx context.Context) error {
		return orgService.IncrementStorage(orgID, bytes)
	})
}

// IncrementCompileJobUsage increments compile job usage for an organization
// Context-aware version that properly manages goroutine lifecycle
func IncrementCompileJobUsage(ctx context.Context, orgService orgs.Service, orgID int64) {
	async.SafeGo(ctx, 5*time.Second, "increment compile job usage", func(ctx context.Context) error {
		return orgService.IncrementCompileJobs(orgID)
	})
}
