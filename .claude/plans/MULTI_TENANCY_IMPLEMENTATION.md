# Multi-Tenancy Implementation Summary

**Implementation Date**: 2026-01-24
**Status**: Complete
**Billing Integration**: None - Spoke is entirely free and open-source

## Overview

Implemented comprehensive multi-tenancy support for Spoke without any billing integration. This feature enables organizations to:

- Isolate their protobuf schemas
- Manage team members with roles
- Enforce configurable resource quotas
- Track usage for monitoring
- Share public modules across organizations

## Key Changes

### 1. Database Schema (Migration 005)

**File**: `migrations/005_create_multitenancy_schema.up.sql`

**Added Tables**:
- `org_quotas` - Resource limits per organization
- `org_usage` - Usage tracking per organization
- `org_invitations` - Email-based team invitations

**Modified Tables**:
- `organizations` - Added `slug`, `owner_id`, `quota_tier`, `status`, `settings`
- `modules` - Added `org_id`, `is_public`, `owner_id`
- `organization_members` - Added `invited_by`, `joined_at`

**Removed Tables** (from original design):
- `subscriptions` - No billing
- `invoices` - No billing
- `payment_methods` - No billing

### 2. Quota Tiers (Free, No Billing)

**Small (Default)**:
- 10 modules
- 100 versions per module
- 5GB storage
- 5,000 compile jobs/month
- 5,000 API requests/hour

**Medium**:
- 50 modules
- 500 versions per module
- 25GB storage
- 25,000 compile jobs/month
- 25,000 API requests/hour

**Large**:
- 200 modules
- 2,000 versions per module
- 100GB storage
- 100,000 compile jobs/month
- 100,000 API requests/hour

**Unlimited**:
- No limits (for self-hosted deployments)

### 3. Code Structure

**New Files**:
- `pkg/middleware/org.go` - Organization context and quota enforcement middleware
- `pkg/middleware/org_test.go` - Middleware tests

**Modified Files**:
- `pkg/orgs/types.go` - Changed `PlanTier` to `QuotaTier`, removed billing references
- `pkg/orgs/service.go` - Updated to use `QuotaTier`, added custom settings support
- `pkg/orgs/quotas.go` - Quota enforcement implementation
- `pkg/orgs/members.go` - Member and invitation management
- `pkg/orgs/service_test.go` - Updated tests for quota tiers
- `pkg/api/org_handlers.go` - Organization API endpoints
- `pkg/billing/*` - Updated for compatibility (not used)

**Documentation**:
- `docs/multitenancy.md` - Comprehensive multi-tenancy guide (updated to remove billing)

### 4. API Endpoints

**Organizations**:
- `POST /orgs` - Create organization
- `GET /orgs` - List user's organizations
- `GET /orgs/{id}` - Get organization
- `PUT /orgs/{id}` - Update organization
- `DELETE /orgs/{id}` - Delete organization

**Quotas & Usage**:
- `GET /orgs/{id}/quotas` - Get quotas
- `PUT /orgs/{id}/quotas` - Update quotas (admin only)
- `GET /orgs/{id}/usage` - Get current usage
- `GET /orgs/{id}/usage/history` - Get usage history

**Members**:
- `GET /orgs/{id}/members` - List members
- `POST /orgs/{id}/members` - Add member
- `PUT /orgs/{id}/members/{user_id}` - Update role
- `DELETE /orgs/{id}/members/{user_id}` - Remove member

**Invitations**:
- `POST /orgs/{id}/invitations` - Create invitation
- `GET /orgs/{id}/invitations` - List invitations
- `DELETE /orgs/{id}/invitations/{id}` - Revoke invitation
- `POST /invitations/{token}/accept` - Accept invitation

**Module Access**:
- `POST /orgs/{id}/modules` - Create module
- `GET /orgs/{id}/modules` - List organization modules
- `GET /modules/{org_slug}/{name}` - Access public module

### 5. Middleware

**OrgContextMiddleware**:
- Adds organization to request context
- Supports org_id and org_slug parameters
- Validates organization existence

**QuotaCheckMiddleware**:
- Enforces quotas before operations
- Checks module, version, compile, and API quotas
- Returns 429 with quota details when exceeded
- Skips GET requests

**UsageTrackingMiddleware**:
- Tracks API requests asynchronously
- Updates usage counters

### 6. Features

**Organization Isolation**:
- All modules scoped to organizations via `org_id`
- Row-level security in database queries
- Public modules can be shared across organizations

**Team Management**:
- Three roles: Admin, Developer, Viewer
- Email-based invitations with 7-day expiry
- Cryptographically random invitation tokens

**Quota Enforcement**:
- Enforced via middleware before operations
- Configurable per organization
- Clear error messages with current/limit values

**Usage Tracking**:
- Real-time tracking of resource consumption
- Monthly usage periods
- Historical data retention

## Testing

All tests passing:

```bash
# Middleware tests
go test ./pkg/middleware/... -v
PASS (2.138s)

# Organization tests
go test ./pkg/orgs/... -v
PASS (0.330s)

# Build verification
go build ./cmd/... ./pkg/...
SUCCESS
```

## Configuration

### Environment Variables

```bash
# Database connection
DATABASE_URL=postgresql://user:pass@localhost:5432/spoke

# Optional: Default quota tier for new organizations
DEFAULT_QUOTA_TIER=small  # small, medium, large, unlimited
```

### Server Setup

```go
import (
    "github.com/platinummonkey/spoke/pkg/orgs"
    "github.com/platinummonkey/spoke/pkg/middleware"
)

// Initialize organization service
orgService := orgs.NewPostgresService(db)

// Add middleware
router.Use(middleware.OrgContextMiddleware(orgService))
router.Use(middleware.UsageTrackingMiddleware(orgService))

// Add quota enforcement for module operations
modulesRouter := router.PathPrefix("/orgs/{org_id}/modules").Subrouter()
modulesRouter.Use(middleware.QuotaCheckMiddleware(orgService, "module"))
```

## Migration Instructions

### For New Installations

```bash
# Run migrations
psql $DATABASE_URL -f migrations/001_create_base_schema.up.sql
psql $DATABASE_URL -f migrations/002_create_auth_schema.up.sql
psql $DATABASE_URL -f migrations/003_create_sso_schema.up.sql
psql $DATABASE_URL -f migrations/004_create_audit_schema.up.sql
psql $DATABASE_URL -f migrations/005_create_multitenancy_schema.up.sql
```

### For Existing Installations

The migration automatically:
1. Adds multi-tenancy columns to existing tables
2. Backfills existing modules to "default" organization
3. Creates default quotas for all organizations
4. Initializes usage tracking

## Security Considerations

1. **Data Isolation**: All queries filtered by `org_id`
2. **RBAC Integration**: Role-based access control for all operations
3. **Invitation Tokens**: 32-byte cryptographically random tokens
4. **Public Module Control**: Only owners can modify public modules
5. **Audit Logging**: All operations logged for compliance

## Performance Considerations

1. **Indexes**: Added on `org_id`, `slug`, `is_public`
2. **Usage Tracking**: Asynchronous to avoid blocking requests
3. **Quota Checks**: Cached quota values (consider adding caching)
4. **View Optimization**: `org_members_view` for efficient member queries

## Future Enhancements

- [ ] Quota increase automation based on usage patterns
- [ ] Organization transfer/migration tools
- [ ] Usage alerts and notifications
- [ ] Module dependency visualization per organization
- [ ] Organization-level webhooks
- [ ] Module marketplace for public schemas

## API Examples

### Create Organization

```bash
curl -X POST https://spoke.example.com/orgs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acme-corp",
    "display_name": "Acme Corporation",
    "quota_tier": "medium"
  }'
```

### Invite Team Member

```bash
curl -X POST https://spoke.example.com/orgs/1/invitations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "developer@acme.com",
    "role": "developer"
  }'
```

### Check Usage

```bash
curl https://spoke.example.com/orgs/1/usage \
  -H "Authorization: Bearer $TOKEN"
```

### Create Public Module

```bash
curl -X POST https://spoke.example.com/orgs/1/modules \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "common-types",
    "description": "Shared common types",
    "is_public": true
  }'
```

## Rollback Instructions

If needed, rollback the migration:

```bash
psql $DATABASE_URL -f migrations/005_create_multitenancy_schema.down.sql
```

**Warning**: This will delete all multi-tenancy data including:
- Organization quotas and usage
- Invitations
- Organization settings

## Support

For questions or issues:
- GitHub Issues: https://github.com/platinummonkey/spoke/issues
- Documentation: See `docs/multitenancy.md`

## License

Spoke is open-source and completely free under the MIT License.
