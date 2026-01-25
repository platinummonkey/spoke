#!/bin/bash
# Multi-Tenancy API Examples
# This script demonstrates the multi-tenancy API endpoints

API_URL="${API_URL:-http://localhost:8080}"
TOKEN="${TOKEN:-your-api-token-here}"

echo "Spoke Multi-Tenancy API Examples"
echo "================================="
echo ""

# Helper function to make API calls
call_api() {
    local method=$1
    local endpoint=$2
    local data=$3

    echo ">>> $method $endpoint"
    if [ -n "$data" ]; then
        echo "Request body: $data"
    fi

    if [ -n "$data" ]; then
        response=$(curl -s -X "$method" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$API_URL$endpoint")
    else
        response=$(curl -s -X "$method" \
            -H "Authorization: Bearer $TOKEN" \
            "$API_URL$endpoint")
    fi

    echo "Response: $(echo "$response" | jq . 2>/dev/null || echo "$response")"
    echo ""
}

# 1. Create an organization
echo "1. Creating an organization..."
call_api POST "/orgs" '{
  "name": "my-startup",
  "display_name": "My Startup Inc",
  "description": "Our awesome startup protobuf schemas",
  "plan_tier": "free"
}'

# 2. List organizations
echo "2. Listing organizations..."
call_api GET "/orgs"

# 3. Get organization details
ORG_ID=1
echo "3. Getting organization details..."
call_api GET "/orgs/$ORG_ID"

# 4. Get quotas
echo "4. Getting organization quotas..."
call_api GET "/orgs/$ORG_ID/quotas"

# 5. Get usage
echo "5. Getting current usage..."
call_api GET "/orgs/$ORG_ID/usage"

# 6. Get usage history
echo "6. Getting usage history..."
call_api GET "/orgs/$ORG_ID/usage/history?limit=3"

# 7. Invite a team member
echo "7. Inviting a team member..."
call_api POST "/orgs/$ORG_ID/invitations" '{
  "email": "developer@example.com",
  "role": "developer"
}'

# 8. List invitations
echo "8. Listing invitations..."
call_api GET "/orgs/$ORG_ID/invitations"

# 9. List members
echo "9. Listing organization members..."
call_api GET "/orgs/$ORG_ID/members"

# 10. Create a subscription
echo "10. Creating a subscription (upgrade to Pro)..."
call_api POST "/orgs/$ORG_ID/subscription" '{
  "plan": "pro",
  "trial_period_days": 14
}'

# 11. Get subscription
echo "11. Getting subscription details..."
call_api GET "/orgs/$ORG_ID/subscription"

# 12. List invoices
echo "12. Listing invoices..."
call_api GET "/orgs/$ORG_ID/invoices?limit=10"

# 13. Add a payment method
echo "13. Adding a payment method..."
call_api POST "/orgs/$ORG_ID/payment-methods" '{
  "stripe_payment_method_id": "pm_card_visa",
  "set_as_default": true
}'

# 14. List payment methods
echo "14. Listing payment methods..."
call_api GET "/orgs/$ORG_ID/payment-methods"

# 15. Update organization
echo "15. Updating organization..."
call_api PUT "/orgs/$ORG_ID" '{
  "display_name": "My Startup Inc (Updated)",
  "description": "Updated description"
}'

# 16. Update member role
echo "16. Updating member role..."
USER_ID=2
call_api PUT "/orgs/$ORG_ID/members/$USER_ID" '{
  "role": "admin"
}'

# 17. Cancel subscription
echo "17. Canceling subscription at period end..."
call_api POST "/orgs/$ORG_ID/subscription/cancel" '{
  "immediately": false
}'

# 18. Reactivate subscription
echo "18. Reactivating subscription..."
call_api POST "/orgs/$ORG_ID/subscription/reactivate"

echo "================================="
echo "API Examples completed!"
echo ""
echo "Note: Some examples may fail if:"
echo "  - The organization doesn't exist"
echo "  - You don't have proper permissions"
echo "  - Billing is not configured"
echo ""
echo "Set environment variables:"
echo "  export API_URL=http://localhost:8080"
echo "  export TOKEN=your-api-token"
