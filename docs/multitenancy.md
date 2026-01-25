# Multi-Tenancy and Billing

This document describes the multi-tenancy and billing features for Spoke, enabling SaaS offering.

## Overview

Spoke's multi-tenancy system provides:

- **Organization Isolation**: Separate namespaces and data isolation per organization
- **Resource Quotas**: Enforced limits on modules, versions, storage, compile jobs, and API requests
- **Usage Tracking**: Real-time tracking of resource consumption
- **Billing Integration**: Stripe integration for subscription management and payment processing
- **Team Management**: User invitations, role-based access control, and member management

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

#### Subscriptions
- Stripe customer and subscription IDs
- Plan tier and status tracking
- Trial and cancellation management

#### Invitations
- Email-based user invitations
- Token-based acceptance flow
- Expiration and role assignment

#### Payment Methods
- Stripe payment method IDs
- Card and bank account support
- Default payment method management

#### Invoices
- Usage-based billing records
- Stripe invoice integration
- Payment status tracking

## Plan Tiers

### Free Tier
- **Price**: $0/month
- **Modules**: 5
- **Versions per Module**: 50
- **Storage**: 1GB
- **Compile Jobs**: 100/month
- **API Rate Limit**: 1,000 req/hour

### Pro Tier
- **Price**: $49/month
- **Modules**: 50
- **Versions per Module**: 500
- **Storage**: 10GB
- **Compile Jobs**: 1,000/month
- **API Rate Limit**: 10,000 req/hour
- **Overage Pricing**:
  - Storage: $5/GB over quota
  - Compile Jobs: $0.05/job over quota
  - API Requests: $0.10/1000 requests over quota

### Enterprise Tier
- **Price**: $499/month
- **Modules**: 1,000
- **Versions per Module**: 10,000
- **Storage**: 100GB
- **Compile Jobs**: 100,000/month
- **API Rate Limit**: 100,000 req/hour
- **Overage Pricing**:
  - Storage: $3/GB over quota
  - Compile Jobs: $0.03/job over quota
  - API Requests: $0.05/1000 requests over quota

### Custom Tier
- Custom quotas negotiated per customer
- Contact sales for pricing

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

### Subscriptions

```
POST   /orgs/{id}/subscription       - Create subscription
GET    /orgs/{id}/subscription       - Get subscription
PUT    /orgs/{id}/subscription       - Update subscription
POST   /orgs/{id}/subscription/cancel    - Cancel subscription
POST   /orgs/{id}/subscription/reactivate - Reactivate subscription
```

### Invoices

```
GET    /orgs/{id}/invoices           - List invoices
GET    /invoices/{id}                - Get invoice
POST   /orgs/{id}/invoices/generate  - Generate invoice
```

### Payment Methods

```
POST   /orgs/{id}/payment-methods    - Add payment method
GET    /orgs/{id}/payment-methods    - List payment methods
PUT    /orgs/{id}/payment-methods/{id}/default - Set default
DELETE /orgs/{id}/payment-methods/{id}         - Remove method
```

### Webhooks

```
POST   /billing/webhook              - Stripe webhook handler
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
    "plan_tier": "pro"
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

### Creating a Subscription

```bash
curl -X POST https://spoke.example.com/orgs/1/subscription \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "plan": "pro",
    "payment_method_id": "pm_1234567890",
    "trial_period_days": 14
  }'
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
HTTP/1.1 403 Forbidden
Content-Type: application/json

{
  "error": "quota exceeded for modules",
  "current": 5,
  "limit": 5
}
```

For API rate limits:

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{
  "error": "API rate limit exceeded"
}
```

## Stripe Integration

### Setup

1. Create a Stripe account at https://stripe.com
2. Get your API keys from the Stripe Dashboard
3. Configure webhook endpoint: `https://spoke.example.com/billing/webhook`
4. Set webhook secret in environment variables

### Environment Variables

```bash
STRIPE_API_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
```

### Webhook Events

The system handles the following Stripe webhook events:

- `customer.subscription.created` - New subscription
- `customer.subscription.updated` - Subscription changes
- `customer.subscription.deleted` - Subscription canceled
- `invoice.paid` - Payment successful
- `invoice.payment_failed` - Payment failed

### Test Mode

For development, use Stripe test mode:

1. Use test API keys (starting with `sk_test_`)
2. Use test payment methods:
   - Success: `pm_card_visa` or card number `4242 4242 4242 4242`
   - Decline: `pm_card_chargeDecline` or card number `4000 0000 0000 0002`

## Usage-Based Billing

Billing is calculated monthly based on:

1. **Base subscription price** for the plan tier
2. **Overage charges** for resources exceeding quotas:
   - Storage: Charged per GB over quota
   - Compile Jobs: Charged per job over quota
   - API Requests: Charged per 1000 requests over quota

### Example Calculation (Pro Plan)

```
Base Price:           $49.00
Storage Overage:      $25.00  (5GB over 10GB quota at $5/GB)
Compile Job Overage:  $10.00  (200 jobs over 1000 at $0.05/job)
API Request Overage:   $1.00  (10k requests over quota at $0.10/1000)
------------------------
Total:                $85.00
```

### Invoice Generation

Invoices are automatically generated at the end of each billing period:

1. System calculates usage and overages
2. Creates invoice in database
3. Submits invoice to Stripe
4. Stripe charges the default payment method
5. Webhook confirms payment

Manual invoice generation:

```bash
curl -X POST https://spoke.example.com/orgs/1/invoices/generate \
  -H "Authorization: Bearer $TOKEN"
```

## Security Considerations

1. **Data Isolation**: All queries filter by `org_id` to prevent cross-org access
2. **RBAC Integration**: Role-based access control for org members
3. **Invitation Tokens**: Cryptographically random 32-byte tokens
4. **Webhook Verification**: Stripe signature verification on webhooks
5. **Payment Data**: Never stored directly, only Stripe IDs

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
├── orgs/                   - Organization management
│   ├── types.go           - Data types and interfaces
│   ├── service.go         - Core CRUD operations
│   ├── members.go         - Member and invitation management
│   ├── quotas.go          - Quota enforcement and usage tracking
│   └── service_test.go    - Unit tests
│
├── billing/               - Billing and subscriptions
│   ├── types.go          - Billing data types
│   ├── service.go        - Subscription and invoice management
│   ├── stripe.go         - Stripe API integration
│   └── types_test.go     - Unit tests
│
├── api/                   - HTTP handlers
│   ├── org_handlers.go   - Organization API endpoints
│   └── billing_handlers.go - Billing API endpoints
│
└── middleware/            - HTTP middleware
    └── quota.go          - Quota enforcement middleware
```

## Testing

Run tests:

```bash
go test ./pkg/orgs/...
go test ./pkg/billing/...
```

Integration tests require a test database:

```bash
export TEST_DATABASE_URL="postgresql://localhost/spoke_test"
go test -tags=integration ./...
```

## Future Enhancements

- [ ] Custom domain support for organizations
- [ ] Advanced usage analytics dashboard
- [ ] Automated quota increase requests
- [ ] Enterprise SSO integration per organization
- [ ] Multi-region support
- [ ] Reseller/partner accounts
- [ ] Usage alerts and notifications
- [ ] Budget caps and spending limits
- [ ] Annual billing discounts
- [ ] Volume discounts for large enterprises
