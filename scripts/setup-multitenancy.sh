#!/bin/bash
set -e

# Setup script for multi-tenancy feature
# This script initializes the database, creates default organizations, and sets up test data

echo "Setting up Spoke multi-tenancy..."

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL environment variable is not set"
    echo "Example: export DATABASE_URL=postgresql://spoke:spoke@localhost:5432/spoke"
    exit 1
fi

# Run migrations
echo "Running database migrations..."
for migration in migrations/00*.up.sql; do
    echo "  Applying migration: $(basename $migration)"
    psql "$DATABASE_URL" < "$migration"
done

echo "Migrations completed successfully!"

# Create test organizations
echo "Creating test organizations..."

psql "$DATABASE_URL" <<EOF
-- Create test users
INSERT INTO users (username, email, full_name, is_bot, is_active)
VALUES
    ('testuser1', 'user1@example.com', 'Test User 1', false, true),
    ('testuser2', 'user2@example.com', 'Test User 2', false, true),
    ('botuser', 'bot@example.com', 'Bot User', true, true)
ON CONFLICT (username) DO NOTHING;

-- Create test organizations
INSERT INTO organizations (name, slug, display_name, description, plan_tier, status, is_active)
VALUES
    ('acme-corp', 'acme-corp', 'Acme Corporation', 'Test organization for Acme Corp', 'free', 'active', true),
    ('widgets-inc', 'widgets-inc', 'Widgets Inc', 'Test organization for Widgets Inc', 'pro', 'active', true),
    ('enterprise-co', 'enterprise-co', 'Enterprise Co', 'Test enterprise organization', 'enterprise', 'active', true)
ON CONFLICT (name) DO NOTHING;

-- Add users to organizations
INSERT INTO organization_members (organization_id, user_id, role)
SELECT o.id, u.id, 'admin'
FROM organizations o
CROSS JOIN users u
WHERE o.name = 'acme-corp' AND u.username = 'testuser1'
ON CONFLICT (organization_id, user_id) DO NOTHING;

INSERT INTO organization_members (organization_id, user_id, role)
SELECT o.id, u.id, 'developer'
FROM organizations o
CROSS JOIN users u
WHERE o.name = 'widgets-inc' AND u.username = 'testuser2'
ON CONFLICT (organization_id, user_id) DO NOTHING;

-- Create API tokens for testing
INSERT INTO api_tokens (user_id, token_hash, token_prefix, name, description, scopes)
SELECT
    u.id,
    encode(digest('test-token-' || u.username, 'sha256'), 'hex'),
    'spoke_',
    'Test Token',
    'Token for testing',
    ARRAY['*']
FROM users u
WHERE u.username IN ('testuser1', 'testuser2', 'botuser')
ON CONFLICT (token_hash) DO NOTHING;

EOF

echo "Test organizations created successfully!"

# Display summary
echo ""
echo "===================================="
echo "Multi-tenancy setup completed!"
echo "===================================="
echo ""
echo "Test Organizations:"
echo "  1. acme-corp (Free tier)"
echo "     - User: testuser1@example.com (admin)"
echo ""
echo "  2. widgets-inc (Pro tier)"
echo "     - User: testuser2@example.com (developer)"
echo ""
echo "  3. enterprise-co (Enterprise tier)"
echo ""
echo "Test API Tokens:"
echo "  - testuser1: spoke_test-token-testuser1"
echo "  - testuser2: spoke_test-token-testuser2"
echo "  - botuser: spoke_test-token-botuser"
echo ""
echo "You can now start the Spoke server with:"
echo "  ./spoke-server --postgres-url=\$DATABASE_URL --stripe-api-key=\$STRIPE_API_KEY"
echo ""
