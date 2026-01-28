// Package orgs provides multi-tenant organization management for the Spoke registry.
//
// # Overview
//
// This package manages organizations, membership, resource quotas, and usage tracking.
// It supports tiered quota systems to control resource consumption per organization.
//
// # Quota Tiers
//
// Small (Free Tier):
//   - 10 modules
//   - 100 versions per module
//   - 5 GB storage
//   - 100 compile jobs/day
//   - 1000 API requests/hour
//
// Medium ($49/month):
//   - 50 modules
//   - 500 versions per module
//   - 25 GB storage
//   - 500 compile jobs/day
//   - 5000 API requests/hour
//
// Large ($499/month):
//   - 200 modules
//   - 2000 versions per module
//   - 100 GB storage
//   - 2000 compile jobs/day
//   - 20000 API requests/hour
//
// Unlimited (Enterprise):
//   - Unlimited resources
//   - Custom limits
//
// # Usage Example
//
// Create organization:
//
//	org := &orgs.Organization{
//		Name:  "Acme Corp",
//		Slug:  "acme",
//		Tier:  orgs.TierSmall,
//	}
//	service.CreateOrganization(ctx, org)
//
// Quota enforcement:
//
//	err := service.CheckQuota(ctx, orgID, orgs.QuotaTypeModules)
//	if err == orgs.ErrQuotaExceeded {
//		return errors.New("upgrade plan to create more modules")
//	}
//
// Track usage:
//
//	service.IncrementUsage(ctx, orgID, orgs.UsageTypeModules, 1)
//
// # Related Packages
//
//   - pkg/auth: User roles (admin, developer, viewer)
//   - pkg/billing: Subscription management
package orgs
