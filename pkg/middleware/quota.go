package middleware

import (
	"context"
	"net/http"

	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/orgs"
)

// QuotaMiddleware enforces quotas for API requests
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
func IncrementModuleUsage(orgService orgs.Service, orgID int64) {
	go func() {
		if err := orgService.IncrementModules(orgID); err != nil {
			// Log error but don't block
		}
	}()
}

// IncrementVersionUsage increments version usage for an organization
func IncrementVersionUsage(orgService orgs.Service, orgID int64) {
	go func() {
		if err := orgService.IncrementVersions(orgID); err != nil {
			// Log error but don't block
		}
	}()
}

// IncrementStorageUsage increments storage usage for an organization
func IncrementStorageUsage(orgService orgs.Service, orgID int64, bytes int64) {
	go func() {
		if err := orgService.IncrementStorage(orgID, bytes); err != nil {
			// Log error but don't block
		}
	}()
}

// IncrementCompileJobUsage increments compile job usage for an organization
func IncrementCompileJobUsage(orgService orgs.Service, orgID int64) {
	go func() {
		if err := orgService.IncrementCompileJobs(orgID); err != nil {
			// Log error but don't block
		}
	}()
}
