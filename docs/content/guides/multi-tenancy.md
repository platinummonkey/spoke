---
title: "Multi-Tenancy"
weight: 11
---

# Multi-Tenancy

Spoke supports multi-tenancy through organization-based isolation.

## Overview

Multi-tenancy features:
- **Organization isolation**: Resources are scoped to organizations
- **Quotas**: Resource limits per organization
- **Usage tracking**: Monitor resource consumption
- **Billing integration**: Stripe integration for SaaS deployments
- **Team management**: Invite and manage organization members

## Configuration

### Enable Multi-Tenancy

```yaml
# config.yaml
multi_tenancy:
  enabled: true
  default_plan: free

database:
  driver: postgres
  host: localhost
  port: 5432
  database: spoke
  user: spoke
  password: your-password

billing:
  enabled: true
  stripe_key: sk_test_...
  stripe_webhook_secret: whsec_...
```

### Plan Tiers

Define plan tiers:

```yaml
plans:
  - name: free
    modules_limit: 5
    versions_per_module: 10
    storage_gb: 1
    api_requests_per_hour: 100
    price: 0

  - name: professional
    modules_limit: 50
    versions_per_module: 100
    storage_gb: 50
    api_requests_per_hour: 1000
    price: 49

  - name: enterprise
    modules_limit: -1  # unlimited
    versions_per_module: -1
    storage_gb: -1
    api_requests_per_hour: -1
    price: 499
```

## Organizations

### Create Organization

```http
POST /organizations
Authorization: Bearer <token>

{
  "name": "Acme Corp",
  "slug": "acme",
  "plan": "professional"
}
```

**Response:**

```json
{
  "id": "org-123",
  "name": "Acme Corp",
  "slug": "acme",
  "plan": "professional",
  "owner_id": "user-456",
  "created_at": "2025-01-24T10:00:00Z",
  "quotas": {
    "modules": 50,
    "versions_per_module": 100,
    "storage_gb": 50,
    "api_requests_per_hour": 1000
  }
}
```

### Get Organization

```http
GET /organizations/{org_id}
Authorization: Bearer <token>
```

### Update Organization

```http
PUT /organizations/{org_id}
Authorization: Bearer <token>

{
  "name": "Acme Corporation",
  "settings": {
    "allow_public_modules": false
  }
}
```

### Delete Organization

```http
DELETE /organizations/{org_id}
Authorization: Bearer <token>
```

## Members

### Invite Member

```http
POST /organizations/{org_id}/invitations
Authorization: Bearer <token>

{
  "email": "developer@example.com",
  "role": "org:developer"
}
```

**Response:**

```json
{
  "id": "invite-789",
  "email": "developer@example.com",
  "role": "org:developer",
  "status": "pending",
  "expires_at": "2025-01-31T10:00:00Z",
  "invitation_url": "https://spoke.company.com/invitations/invite-789"
}
```

### List Members

```http
GET /organizations/{org_id}/members
Authorization: Bearer <token>
```

**Response:**

```json
{
  "members": [
    {
      "user_id": "user-456",
      "email": "admin@example.com",
      "role": "org:admin",
      "joined_at": "2025-01-24T10:00:00Z"
    },
    {
      "user_id": "user-789",
      "email": "developer@example.com",
      "role": "org:developer",
      "joined_at": "2025-01-25T09:00:00Z"
    }
  ]
}
```

### Remove Member

```http
DELETE /organizations/{org_id}/members/{user_id}
Authorization: Bearer <token>
```

### Update Member Role

```http
PUT /organizations/{org_id}/members/{user_id}/role
Authorization: Bearer <token>

{
  "role": "org:admin"
}
```

## Quotas

### Check Quota Usage

```http
GET /organizations/{org_id}/usage
Authorization: Bearer <token>
```

**Response:**

```json
{
  "organization_id": "org-123",
  "plan": "professional",
  "quotas": {
    "modules": {
      "limit": 50,
      "used": 12,
      "remaining": 38
    },
    "storage_gb": {
      "limit": 50,
      "used": 8.5,
      "remaining": 41.5
    },
    "api_requests_per_hour": {
      "limit": 1000,
      "used": 234,
      "remaining": 766
    }
  },
  "billing_period": {
    "start": "2025-01-01T00:00:00Z",
    "end": "2025-02-01T00:00:00Z"
  }
}
```

### Quota Enforcement

Spoke automatically enforces quotas:

```bash
# Attempt to exceed module limit
spoke-cli push -module new-module -version v1.0.0 -dir ./proto

# Error: Quota exceeded
# Organization 'acme' has reached the module limit (50/50)
# Please upgrade your plan or delete unused modules
```

## Billing

### Stripe Integration

#### Create Subscription

```http
POST /organizations/{org_id}/subscription
Authorization: Bearer <token>

{
  "plan": "professional",
  "payment_method_id": "pm_xxx"
}
```

#### Get Billing Info

```http
GET /organizations/{org_id}/billing
Authorization: Bearer <token>
```

**Response:**

```json
{
  "subscription": {
    "id": "sub-xxx",
    "status": "active",
    "plan": "professional",
    "amount": 4900,
    "currency": "usd",
    "current_period_start": "2025-01-01T00:00:00Z",
    "current_period_end": "2025-02-01T00:00:00Z"
  },
  "payment_method": {
    "type": "card",
    "brand": "visa",
    "last4": "4242",
    "exp_month": 12,
    "exp_year": 2026
  }
}
```

#### Upgrade Plan

```http
POST /organizations/{org_id}/subscription/upgrade
Authorization: Bearer <token>

{
  "plan": "enterprise"
}
```

#### Cancel Subscription

```http
DELETE /organizations/{org_id}/subscription
Authorization: Bearer <token>
```

## Module Scoping

### Push Module to Organization

```bash
spoke-cli push \
  -module user-service \
  -version v1.0.0 \
  -dir ./proto \
  -org acme \
  -registry https://spoke.company.com
```

### API with Organization Context

```http
POST /modules
Authorization: Bearer <token>
X-Organization-ID: org-123

{
  "name": "user-service",
  "description": "User service API"
}
```

### List Organization Modules

```http
GET /modules?organization_id=org-123
Authorization: Bearer <token>
```

## CLI with Organizations

### Configure Default Organization

```bash
# Set default organization
spoke-cli config set organization acme

# Use in commands
spoke-cli push -module user-service -version v1.0.0 -dir ./proto
```

### Override Organization

```bash
spoke-cli push \
  -module user-service \
  -version v1.0.0 \
  -dir ./proto \
  -org other-org
```

## Usage Tracking

### Track Module Pushes

```sql
-- Usage tracking in database
SELECT
  organization_id,
  COUNT(*) as module_count,
  SUM(storage_bytes) / (1024*1024*1024.0) as storage_gb
FROM modules
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY organization_id;
```

### API Request Tracking

```yaml
# Middleware tracks API requests per organization
middleware:
  - name: rate_limiter
    config:
      per_organization: true
      requests_per_hour: 1000
```

## Migration from Single-Tenant

### Migrate Existing Data

```sql
-- Create default organization
INSERT INTO organizations (id, name, slug, plan, owner_id)
VALUES ('org-default', 'Default Organization', 'default', 'enterprise', 'admin-user');

-- Assign all modules to default org
UPDATE modules SET organization_id = 'org-default' WHERE organization_id IS NULL;

-- Assign all users to default org
INSERT INTO organization_members (organization_id, user_id, role)
SELECT 'org-default', id, 'org:admin'
FROM users
WHERE id NOT IN (SELECT user_id FROM organization_members);
```

## Best Practices

### 1. Organization Naming

Use clear, unique slugs:

```json
{
  "name": "Acme Corporation",
  "slug": "acme-corp"  // Used in URLs
}
```

### 2. Quota Monitoring

Monitor quota usage:

```bash
# Check usage regularly
curl -H "Authorization: Bearer $TOKEN" \
  https://spoke.company.com/organizations/org-123/usage

# Set up alerts for 80% usage
```

### 3. Member Management

Use teams instead of individual permissions:

```bash
# Create team
curl -X POST /organizations/org-123/teams \
  -d '{"name": "Backend Team"}'

# Assign role to team
curl -X POST /organizations/org-123/teams/team-456/roles \
  -d '{"role_id": "role-developer"}'
```

### 4. Billing Webhooks

Handle Stripe webhooks:

```javascript
// webhook-handler.js
app.post('/webhooks/stripe', async (req, res) => {
  const event = req.body;

  switch (event.type) {
    case 'customer.subscription.updated':
      // Update organization plan
      break;
    case 'invoice.payment_failed':
      // Notify organization
      break;
  }

  res.json({received: true});
});
```

## Troubleshooting

### Quota Exceeded

**Problem**: Cannot push module, quota exceeded

**Solution**: Check usage and upgrade plan:

```bash
# Check usage
curl /organizations/org-123/usage

# Upgrade plan
curl -X POST /organizations/org-123/subscription/upgrade \
  -d '{"plan": "enterprise"}'
```

### Member Cannot Access Organization

**Problem**: Invited member cannot access

**Solution**: Check invitation status:

```bash
# List invitations
curl /organizations/org-123/invitations

# Resend invitation
curl -X POST /organizations/org-123/invitations/invite-789/resend
```

## Next Steps

- [RBAC Configuration](/guides/rbac/) - Role-based access control
- [Billing Integration](/guides/billing/) - Stripe setup
- [Audit Logging](/guides/audit/) - Compliance and auditing
