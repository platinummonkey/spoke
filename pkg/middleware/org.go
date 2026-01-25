package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/orgs"
)

// OrgContextKey is the key for organization context
type OrgContextKey string

const (
	// OrgKey is the context key for organization
	OrgKey OrgContextKey = "organization"
)

// OrgContextMiddleware adds organization context to the request
func OrgContextMiddleware(orgService orgs.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request has org_id parameter
			vars := mux.Vars(r)
			if orgIDStr, ok := vars["org_id"]; ok {
				orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
				if err != nil {
					http.Error(w, "Invalid organization ID", http.StatusBadRequest)
					return
				}

				org, err := orgService.GetOrganization(orgID)
				if err != nil {
					http.Error(w, "Organization not found", http.StatusNotFound)
					return
				}

				// Add organization to context
				ctx := context.WithValue(r.Context(), OrgKey, org)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Check if request has org_slug parameter
			if orgSlug, ok := vars["org_slug"]; ok {
				org, err := orgService.GetOrganizationBySlug(orgSlug)
				if err != nil {
					http.Error(w, "Organization not found", http.StatusNotFound)
					return
				}

				// Add organization to context
				ctx := context.WithValue(r.Context(), OrgKey, org)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No organization context needed
			next.ServeHTTP(w, r)
		})
	}
}

// QuotaCheckMiddleware checks quotas before allowing operations
func QuotaCheckMiddleware(orgService orgs.Service, quotaType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip quota check for GET requests
			if r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Get organization from context
			org, ok := r.Context().Value(OrgKey).(*orgs.Organization)
			if !ok {
				// No organization context, skip quota check
				next.ServeHTTP(w, r)
				return
			}

			var err error
			switch quotaType {
			case "module":
				err = orgService.CheckModuleQuota(org.ID)
			case "version":
				// Get module name from URL
				vars := mux.Vars(r)
				moduleName := vars["module_name"]
				err = orgService.CheckVersionQuota(org.ID, moduleName)
			case "compile":
				err = orgService.CheckCompileJobQuota(org.ID)
			case "api":
				err = orgService.CheckAPIRateLimit(org.ID)
			}

			if err != nil {
				if orgs.IsQuotaExceeded(err) {
					quotaErr := err.(*orgs.QuotaExceededError)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusTooManyRequests)
					w.Write([]byte(`{"error":"quota_exceeded","resource":"` + quotaErr.Resource + `","current":` + strconv.FormatInt(quotaErr.Current, 10) + `,"limit":` + strconv.FormatInt(quotaErr.Limit, 10) + `}`))
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UsageTrackingMiddleware tracks API usage
func UsageTrackingMiddleware(orgService orgs.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get organization from context
			org, ok := r.Context().Value(OrgKey).(*orgs.Organization)
			if ok {
				// Track API request (fire and forget)
				go orgService.IncrementAPIRequests(org.ID)
			}

			next.ServeHTTP(w, r)
		})
	}
}
