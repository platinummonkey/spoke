# Multi-Tenancy (NO BILLING)

**Spoke is entirely free and open-source with no billing integration.**

This document describes the multi-tenancy features for Spoke, enabling organization-based schema management.

## Overview

Spoke's multi-tenancy system provides:

- **Organization Isolation**: Separate namespaces and data isolation per organization
- **Resource Quotas**: Configurable limits on modules, versions, storage, compile jobs, and API requests
- **Usage Tracking**: Real-time tracking of resource consumption (for monitoring, not billing)
- **Team Management**: User invitations, role-based access control, and member management
- **Public Modules**: Share modules across organizations

## Architecture

### Database Schema

#### Organizations
- Core organization data with slug, plan tier, and status
- Settings stored as JSONB for flexibility
- Owner tracking for primary contact

#### Quotas
- Per-organization resource limits
- Configurable based on plan tier
- Includes: modules, versions, storage, compile jobs, API rate limits

#### Usage Tracking
- Real-time usage metrics per organization
- Monthly periods for billing calculations
- Tracks: module count, version count, storage bytes, compile jobs, API requests

#### Invitations
- Email-based user invitations
- Token-based acceptance flow
- Expiration and role assignment

## Quota Tiers (Free, No Billing)

All tiers are **completely free**. Choose the tier that matches your deployment needs.

### Small (Default)
- **Modules**: 10
- **Versions per Module**: 100
- **Storage**: 5GB
- **Compile Jobs**: 5,000/month
- **API Rate Limit**: 5,000 req/hour

*Ideal for small teams and side projects*

### Medium
- **Modules**: 50
- **Versions per Module**: 500
- **Storage**: 25GB
- **Compile Jobs**: 25,000/month
- **API Rate Limit**: 25,000 req/hour

*Ideal for growing teams and startups*

### Large
- **Modules**: 200
- **Versions per Module**: 2,000
- **Storage**: 100GB
- **Compile Jobs**: 100,000/month
- **API Rate Limit**: 100,000 req/hour

*Ideal for large teams and companies*

### Unlimited
- **Modules**: No limit
- **Versions per Module**: No limit
- **Storage**: No limit
- **Compile Jobs**: No limit
- **API Rate Limit**: No limit

*Ideal for self-hosted deployments with dedicated infrastructure*

**Note**: All tiers are configurable. Administrators can adjust quotas for any organization based on deployment capacity.

## API Endpoints

### Organizations

```
POST   /orgs                         - Create organization
GET    /orgs                         - List user's organizations
GET    /orgs/{id}                    - Get organization
PUT    /orgs/{id}                    - Update organization
DELETE /orgs/{id}                    - Delete organization
```

### Quotas & Usage

```
GET    /orgs/{id}/quotas             - Get organization quotas
PUT    /orgs/{id}/quotas             - Update quotas (admin only)
GET    /orgs/{id}/usage              - Get current usage
GET    /orgs/{id}/usage/history      - Get usage history
```

### Members

```
GET    /orgs/{id}/members            - List members
POST   /orgs/{id}/members            - Add member
PUT    /orgs/{id}/members/{user_id}  - Update member role
DELETE /orgs/{id}/members/{user_id}  - Remove member
```

### Invitations

```
POST   /orgs/{id}/invitations        - Create invitation
GET    /orgs/{id}/invitations        - List invitations
DELETE /orgs/{id}/invitations/{id}   - Revoke invitation
POST   /invitations/{token}/accept   - Accept invitation
```

## Module Access

### Organization-Scoped Modules (Private)

```
POST   /orgs/{id}/modules            - Create module
GET    /orgs/{id}/modules            - List organization modules
GET    /orgs/{id}/modules/{name}     - Get module
PUT    /orgs/{id}/modules/{name}     - Update module (toggle public)
DELETE /orgs/{id}/modules/{name}     - Delete module
```

### Public Module Access

```
GET    /modules/{org_slug}/{name}    - Access public module from any org
GET    /modules/{org_slug}/{name}/versions/{version} - Get public version
```

## Usage Examples

### Creating an Organization

```bash
curl -X POST https://spoke.example.com/orgs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acme-corp",
    "display_name": "Acme Corporation",
    "description": "Acme Corp protobuf schemas",
    "quota_tier": "medium"
  }'
```

### Inviting a Team Member

```bash
curl -X POST https://spoke.example.com/orgs/1/invitations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "developer@acme.com",
    "role": "developer"
  }'
```

### Checking Usage

```bash
curl https://spoke.example.com/orgs/1/usage \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "id": 1,
  "org_id": 1,
  "period_start": "2026-01-01T00:00:00Z",
  "period_end": "2026-02-01T00:00:00Z",
  "modules_count": 12,
  "versions_count": 45,
  "storage_bytes": 5368709120,
  "compile_jobs_count": 234,
  "api_requests_count": 45678
}
```

### Creating a Public Module

```bash
# Create a module
curl -X POST https://spoke.example.com/orgs/1/modules \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "common-types",
    "description": "Shared common types",
    "is_public": true
  }'

# Other organizations can now access it
curl https://spoke.example.com/modules/acme-corp/common-types/versions/v1.0.0 \
  -H "Authorization: Bearer $TOKEN"
```

## Quota Enforcement

Quotas are enforced at the middleware level:

1. **API Rate Limiting**: Checked on every request via middleware
2. **Module Creation**: Checked before allowing new modules
3. **Version Publishing**: Checked before allowing new versions
4. **Storage**: Checked before uploading files
5. **Compile Jobs**: Checked before starting compilation

### Quota Exceeded Responses

When a quota is exceeded, the API returns:

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{
  "error": "quota_exceeded",
  "resource": "modules",
  "current": 11,
  "limit": 10
}
```

For API rate limits:

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{
  "error": "quota_exceeded",
  "resource": "api_requests",
  "current": 5001,
  "limit": 5000
}
```

### Adjusting Quotas

Administrators can adjust quotas for any organization:

```bash
curl -X PUT https://spoke.example.com/orgs/1/quotas \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "max_modules": 100,
    "max_versions_per_module": 1000,
    "max_storage_bytes": 107374182400,
    "max_compile_jobs_per_month": 50000,
    "api_rate_limit_per_hour": 10000,
    "custom_settings": {
      "note": "Increased for production use"
    }
  }'
```

## Security Considerations

1. **Data Isolation**: All queries filter by `org_id` to prevent cross-org access
2. **RBAC Integration**: Role-based access control for org members
3. **Invitation Tokens**: Cryptographically random 32-byte tokens with 7-day expiry
4. **Public Module Control**: Only owners can modify public modules
5. **Audit Logging**: All organization operations are logged

## Database Migrations

Run migrations to set up multi-tenancy:

```bash
# Apply migration
psql $DATABASE_URL < migrations/005_create_multitenancy_schema.up.sql

# Rollback if needed
psql $DATABASE_URL < migrations/005_create_multitenancy_schema.down.sql
```

## Code Structure

```
pkg/
├── orgs/                   - Organization management (NO BILLING)
│   ├── types.go           - Data types and interfaces
│   ├── service.go         - Core CRUD operations
│   ├── members.go         - Member and invitation management
│   ├── quotas.go          - Quota enforcement and usage tracking
│   └── service_test.go    - Unit tests
│
├── api/                   - HTTP handlers
│   └── org_handlers.go   - Organization API endpoints
│
└── middleware/            - HTTP middleware
    ├── org.go            - Organization context middleware
    └── org_test.go       - Middleware tests
```

## Testing

Run tests:

```bash
go test ./pkg/orgs/...
go test ./pkg/middleware/...
```

Integration tests require a test database:

```bash
export TEST_DATABASE_URL="postgresql://localhost/spoke_test"
go test -tags=integration ./...
```

## Future Enhancements

- [ ] Custom domain support for organizations
- [ ] Advanced usage analytics dashboard
- [ ] Automated quota adjustment based on usage patterns
- [ ] Per-organization SSO integration
- [ ] Multi-region support
- [ ] Organization transfer/migration tools
- [ ] Usage alerts and notifications (quota warnings)
- [ ] Module dependency visualization per organization
- [ ] Organization-level webhooks
- [ ] Module marketplace for public schemas
