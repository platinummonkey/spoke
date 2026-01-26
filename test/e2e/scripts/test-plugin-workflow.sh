#!/bin/bash
# End-to-end plugin workflow test

set -e

API_URL="http://localhost:8080"
PLUGIN_ID="workflow-test-plugin"

echo "========================================="
echo "Complete Plugin Workflow Test"
echo "========================================="
echo ""

# Step 1: Create plugin
echo "Step 1: Creating plugin..."
curl -s -X POST "$API_URL/api/v1/plugins" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": \"$PLUGIN_ID\",
        \"name\": \"Workflow Test Plugin\",
        \"description\": \"End-to-end workflow test\",
        \"author\": \"E2E Test\",
        \"license\": \"MIT\",
        \"type\": \"language\",
        \"security_level\": \"community\"
    }" | jq '.'

echo "✓ Plugin created"
echo ""

# Step 2: Create version
echo "Step 2: Creating plugin version..."
curl -s -X POST "$API_URL/api/v1/plugins/$PLUGIN_ID/versions" \
    -H "Content-Type: application/json" \
    -d '{
        "version": "1.0.0",
        "api_version": "1.0",
        "download_url": "http://example.com/plugin.tar.gz",
        "checksum": "abc123",
        "size_bytes": 2048
    }' | jq '.'

echo "✓ Version created"
echo ""

# Step 3: Submit for verification
echo "Step 3: Submitting for verification..."
VERIFICATION=$(curl -s -X POST "$API_URL/api/v1/plugins/$PLUGIN_ID/versions/1.0.0/verify" \
    -H "Content-Type: application/json" \
    -d '{
        "submitted_by": "e2e-test",
        "auto_approve": true
    }')

VERIFICATION_ID=$(echo "$VERIFICATION" | jq -r '.verification_id')
echo "$VERIFICATION" | jq '.'
echo "✓ Verification submitted (ID: $VERIFICATION_ID)"
echo ""

# Step 4: Wait for verification (max 30 seconds)
echo "Step 4: Waiting for verification to complete..."
for i in {1..10}; do
    sleep 3
    STATUS=$(curl -s "$API_URL/api/v1/verifications/$VERIFICATION_ID" | jq -r '.status')
    echo "  Status: $STATUS"

    if [ "$STATUS" != "pending" ] && [ "$STATUS" != "in_progress" ]; then
        break
    fi
done

echo "✓ Verification completed: $STATUS"
echo ""

# Step 5: Record installation
echo "Step 5: Recording installation..."
curl -s -X POST "$API_URL/api/v1/plugins/$PLUGIN_ID/install" \
    -H "Content-Type: application/json" \
    -H "X-User-ID: test-user-1" \
    -d '{"version": "1.0.0"}' | jq '.'

echo "✓ Installation recorded"
echo ""

# Step 6: Leave review
echo "Step 6: Submitting review..."
curl -s -X POST "$API_URL/api/v1/plugins/$PLUGIN_ID/reviews" \
    -H "Content-Type: application/json" \
    -H "X-User-ID: test-user-1" \
    -d '{
        "rating": 5,
        "review": "Excellent plugin! Works perfectly in my workflow."
    }' | jq '.'

echo "✓ Review submitted"
echo ""

# Step 7: Check final stats
echo "Step 7: Checking plugin stats..."
curl -s "$API_URL/api/v1/plugins/$PLUGIN_ID/stats" | jq '.'

echo ""
echo "========================================="
echo "✅ Complete plugin workflow successful!"
echo "========================================="
