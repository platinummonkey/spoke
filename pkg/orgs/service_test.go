package orgs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "MyOrg",
			expected: "myorg",
		},
		{
			name:     "name with spaces",
			input:    "My Organization",
			expected: "my-organization",
		},
		{
			name:     "name with special chars",
			input:    "My-Org-123",
			expected: "my-org-123",
		},
		{
			name:     "name with invalid chars",
			input:    "My@Org!",
			expected: "myorg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := generateToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token1)
	assert.Equal(t, 64, len(token1)) // 32 bytes = 64 hex chars

	token2, err := generateToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, token1, token2) // Should be unique
}

func TestQuotaExceededError(t *testing.T) {
	err := &QuotaExceededError{
		Resource: "modules",
		Current:  10,
		Limit:    5,
	}

	assert.True(t, IsQuotaExceeded(err))
	assert.Contains(t, err.Error(), "quota exceeded")
	assert.Contains(t, err.Error(), "modules")
}

func TestGetDefaultQuotas(t *testing.T) {
	service := &PostgresService{}

	tests := []struct {
		name     string
		tier     QuotaTier
		expected *OrgQuotas
	}{
		{
			name: "small tier",
			tier: QuotaTierSmall,
			expected: &OrgQuotas{
				MaxModules:             10,
				MaxVersionsPerModule:   100,
				MaxStorageBytes:        5 * 1024 * 1024 * 1024,
				MaxCompileJobsPerMonth: 5000,
				APIRateLimitPerHour:    5000,
			},
		},
		{
			name: "medium tier",
			tier: QuotaTierMedium,
			expected: &OrgQuotas{
				MaxModules:             50,
				MaxVersionsPerModule:   500,
				MaxStorageBytes:        25 * 1024 * 1024 * 1024,
				MaxCompileJobsPerMonth: 25000,
				APIRateLimitPerHour:    25000,
			},
		},
		{
			name: "large tier",
			tier: QuotaTierLarge,
			expected: &OrgQuotas{
				MaxModules:             200,
				MaxVersionsPerModule:   2000,
				MaxStorageBytes:        100 * 1024 * 1024 * 1024,
				MaxCompileJobsPerMonth: 100000,
				APIRateLimitPerHour:    100000,
			},
		},
		{
			name: "unlimited tier",
			tier: QuotaTierUnlimited,
			expected: &OrgQuotas{
				MaxModules:             999999,
				MaxVersionsPerModule:   999999,
				MaxStorageBytes:        999999 * 1024 * 1024 * 1024,
				MaxCompileJobsPerMonth: 999999999,
				APIRateLimitPerHour:    999999999,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quotas := service.GetDefaultQuotas(tt.tier)
			assert.Equal(t, tt.expected.MaxModules, quotas.MaxModules)
			assert.Equal(t, tt.expected.MaxVersionsPerModule, quotas.MaxVersionsPerModule)
			assert.Equal(t, tt.expected.MaxStorageBytes, quotas.MaxStorageBytes)
			assert.Equal(t, tt.expected.MaxCompileJobsPerMonth, quotas.MaxCompileJobsPerMonth)
			assert.Equal(t, tt.expected.APIRateLimitPerHour, quotas.APIRateLimitPerHour)
		})
	}
}

func TestOrgStatuses(t *testing.T) {
	assert.Equal(t, OrgStatus("active"), OrgStatusActive)
	assert.Equal(t, OrgStatus("suspended"), OrgStatusSuspended)
	assert.Equal(t, OrgStatus("deleted"), OrgStatusDeleted)
}

func TestQuotaTiers(t *testing.T) {
	assert.Equal(t, QuotaTier("small"), QuotaTierSmall)
	assert.Equal(t, QuotaTier("medium"), QuotaTierMedium)
	assert.Equal(t, QuotaTier("large"), QuotaTierLarge)
	assert.Equal(t, QuotaTier("unlimited"), QuotaTierUnlimited)
}

func TestOrgInvitationExpiry(t *testing.T) {
	now := time.Now()
	invitation := &OrgInvitation{
		InvitedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}

	assert.True(t, invitation.ExpiresAt.After(now))
	assert.True(t, invitation.ExpiresAt.Before(now.Add(8 * 24 * time.Hour)))
}
