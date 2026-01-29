package orgs

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test QuotaTier type conversion
func TestQuotaTierTypeConversion(t *testing.T) {
	tests := []struct {
		name     string
		tier     QuotaTier
		expected string
	}{
		{"small tier", QuotaTierSmall, "small"},
		{"medium tier", QuotaTierMedium, "medium"},
		{"large tier", QuotaTierLarge, "large"},
		{"unlimited tier", QuotaTierUnlimited, "unlimited"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.tier))
		})
	}
}

// Test OrgStatus type conversion
func TestOrgStatusTypeConversion(t *testing.T) {
	tests := []struct {
		name     string
		status   OrgStatus
		expected string
	}{
		{"active status", OrgStatusActive, "active"},
		{"suspended status", OrgStatusSuspended, "suspended"},
		{"deleted status", OrgStatusDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

// Test Organization struct
func TestOrganization(t *testing.T) {
	now := time.Now()
	ownerID := int64(123)

	org := &Organization{
		ID:          1,
		Name:        "Test Org",
		Slug:        "test-org",
		DisplayName: "Test Organization",
		Description: "A test organization",
		OwnerID:     &ownerID,
		QuotaTier:   QuotaTierSmall,
		Status:      OrgStatusActive,
		IsActive:    true,
		Settings:    map[string]any{"theme": "dark", "notifications": true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, int64(1), org.ID)
	assert.Equal(t, "Test Org", org.Name)
	assert.Equal(t, "test-org", org.Slug)
	assert.Equal(t, "Test Organization", org.DisplayName)
	assert.Equal(t, "A test organization", org.Description)
	assert.NotNil(t, org.OwnerID)
	assert.Equal(t, int64(123), *org.OwnerID)
	assert.Equal(t, QuotaTierSmall, org.QuotaTier)
	assert.Equal(t, OrgStatusActive, org.Status)
	assert.True(t, org.IsActive)
	assert.NotNil(t, org.Settings)
	assert.Equal(t, "dark", org.Settings["theme"])
	assert.Equal(t, true, org.Settings["notifications"])
	assert.Equal(t, now, org.CreatedAt)
	assert.Equal(t, now, org.UpdatedAt)
}

// Test Organization JSON serialization
func TestOrganizationJSONSerialization(t *testing.T) {
	now := time.Now()
	ownerID := int64(456)

	org := &Organization{
		ID:          2,
		Name:        "JSON Test",
		Slug:        "json-test",
		DisplayName: "JSON Test Org",
		Description: "Testing JSON",
		OwnerID:     &ownerID,
		QuotaTier:   QuotaTierMedium,
		Status:      OrgStatusActive,
		IsActive:    true,
		Settings:    map[string]any{"key": "value"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Marshal to JSON
	data, err := json.Marshal(org)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded Organization
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, org.ID, decoded.ID)
	assert.Equal(t, org.Name, decoded.Name)
	assert.Equal(t, org.Slug, decoded.Slug)
	assert.Equal(t, org.DisplayName, decoded.DisplayName)
	assert.Equal(t, org.Description, decoded.Description)
	assert.NotNil(t, decoded.OwnerID)
	assert.Equal(t, *org.OwnerID, *decoded.OwnerID)
	assert.Equal(t, org.QuotaTier, decoded.QuotaTier)
	assert.Equal(t, org.Status, decoded.Status)
	assert.Equal(t, org.IsActive, decoded.IsActive)
	assert.Equal(t, "value", decoded.Settings["key"])
}

// Test Organization with nil OwnerID
func TestOrganizationNilOwner(t *testing.T) {
	now := time.Now()

	org := &Organization{
		ID:          3,
		Name:        "No Owner",
		Slug:        "no-owner",
		DisplayName: "No Owner Org",
		OwnerID:     nil,
		QuotaTier:   QuotaTierSmall,
		Status:      OrgStatusActive,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Nil(t, org.OwnerID)

	// Test JSON serialization with nil owner
	data, err := json.Marshal(org)
	require.NoError(t, err)

	var decoded Organization
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Nil(t, decoded.OwnerID)
}

// Test OrgQuotas struct
func TestOrgQuotas(t *testing.T) {
	now := time.Now()

	quotas := &OrgQuotas{
		ID:                       1,
		OrgID:                    10,
		MaxModules:               50,
		MaxVersionsPerModule:     500,
		MaxStorageBytes:          25 * 1024 * 1024 * 1024,
		MaxCompileJobsPerMonth:   25000,
		APIRateLimitPerHour:      25000,
		CustomSettings:           map[string]any{"custom_key": "custom_value"},
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	assert.Equal(t, int64(1), quotas.ID)
	assert.Equal(t, int64(10), quotas.OrgID)
	assert.Equal(t, 50, quotas.MaxModules)
	assert.Equal(t, 500, quotas.MaxVersionsPerModule)
	assert.Equal(t, int64(25*1024*1024*1024), quotas.MaxStorageBytes)
	assert.Equal(t, 25000, quotas.MaxCompileJobsPerMonth)
	assert.Equal(t, 25000, quotas.APIRateLimitPerHour)
	assert.NotNil(t, quotas.CustomSettings)
	assert.Equal(t, "custom_value", quotas.CustomSettings["custom_key"])
}

// Test OrgQuotas JSON serialization
func TestOrgQuotasJSONSerialization(t *testing.T) {
	now := time.Now()

	quotas := &OrgQuotas{
		ID:                       2,
		OrgID:                    20,
		MaxModules:               100,
		MaxVersionsPerModule:     1000,
		MaxStorageBytes:          50 * 1024 * 1024 * 1024,
		MaxCompileJobsPerMonth:   50000,
		APIRateLimitPerHour:      50000,
		CustomSettings:           map[string]any{"feature_flag": true},
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	// Marshal to JSON
	data, err := json.Marshal(quotas)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded OrgQuotas
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, quotas.ID, decoded.ID)
	assert.Equal(t, quotas.OrgID, decoded.OrgID)
	assert.Equal(t, quotas.MaxModules, decoded.MaxModules)
	assert.Equal(t, quotas.MaxVersionsPerModule, decoded.MaxVersionsPerModule)
	assert.Equal(t, quotas.MaxStorageBytes, decoded.MaxStorageBytes)
	assert.Equal(t, quotas.MaxCompileJobsPerMonth, decoded.MaxCompileJobsPerMonth)
	assert.Equal(t, quotas.APIRateLimitPerHour, decoded.APIRateLimitPerHour)
	assert.Equal(t, true, decoded.CustomSettings["feature_flag"])
}

// Test OrgUsage struct
func TestOrgUsage(t *testing.T) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	usage := &OrgUsage{
		ID:               1,
		OrgID:            10,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		ModulesCount:     5,
		VersionsCount:    50,
		StorageBytes:     1024 * 1024 * 1024,
		CompileJobsCount: 100,
		APIRequestsCount: 1000,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	assert.Equal(t, int64(1), usage.ID)
	assert.Equal(t, int64(10), usage.OrgID)
	assert.Equal(t, periodStart, usage.PeriodStart)
	assert.Equal(t, periodEnd, usage.PeriodEnd)
	assert.Equal(t, 5, usage.ModulesCount)
	assert.Equal(t, 50, usage.VersionsCount)
	assert.Equal(t, int64(1024*1024*1024), usage.StorageBytes)
	assert.Equal(t, 100, usage.CompileJobsCount)
	assert.Equal(t, int64(1000), usage.APIRequestsCount)
	assert.Equal(t, now, usage.CreatedAt)
	assert.Equal(t, now, usage.UpdatedAt)
}

// Test OrgUsage JSON serialization
func TestOrgUsageJSONSerialization(t *testing.T) {
	now := time.Now()
	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	usage := &OrgUsage{
		ID:               2,
		OrgID:            20,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		ModulesCount:     10,
		VersionsCount:    100,
		StorageBytes:     2 * 1024 * 1024 * 1024,
		CompileJobsCount: 200,
		APIRequestsCount: 2000,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Marshal to JSON
	data, err := json.Marshal(usage)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded OrgUsage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, usage.ID, decoded.ID)
	assert.Equal(t, usage.OrgID, decoded.OrgID)
	assert.Equal(t, usage.ModulesCount, decoded.ModulesCount)
	assert.Equal(t, usage.VersionsCount, decoded.VersionsCount)
	assert.Equal(t, usage.StorageBytes, decoded.StorageBytes)
	assert.Equal(t, usage.CompileJobsCount, decoded.CompileJobsCount)
	assert.Equal(t, usage.APIRequestsCount, decoded.APIRequestsCount)
}

// Test OrgInvitation struct
func TestOrgInvitation(t *testing.T) {
	now := time.Now()
	invitedAt := now.Add(-24 * time.Hour)
	expiresAt := now.Add(7 * 24 * time.Hour)

	invitation := &OrgInvitation{
		ID:        1,
		OrgID:     10,
		Email:     "test@example.com",
		Role:      auth.RoleDeveloper,
		Token:     "abcdef123456",
		InvitedBy: 100,
		InvitedAt: invitedAt,
		ExpiresAt: expiresAt,
	}

	assert.Equal(t, int64(1), invitation.ID)
	assert.Equal(t, int64(10), invitation.OrgID)
	assert.Equal(t, "test@example.com", invitation.Email)
	assert.Equal(t, auth.RoleDeveloper, invitation.Role)
	assert.Equal(t, "abcdef123456", invitation.Token)
	assert.Equal(t, int64(100), invitation.InvitedBy)
	assert.Equal(t, invitedAt, invitation.InvitedAt)
	assert.Equal(t, expiresAt, invitation.ExpiresAt)
	assert.Nil(t, invitation.AcceptedAt)
	assert.Nil(t, invitation.AcceptedBy)
}

// Test OrgInvitation with accepted fields
func TestOrgInvitationAccepted(t *testing.T) {
	now := time.Now()
	invitedAt := now.Add(-48 * time.Hour)
	expiresAt := now.Add(5 * 24 * time.Hour)
	acceptedAt := now.Add(-24 * time.Hour)
	acceptedBy := int64(200)

	invitation := &OrgInvitation{
		ID:         2,
		OrgID:      20,
		Email:      "accepted@example.com",
		Role:       auth.RoleAdmin,
		Token:      "token789",
		InvitedBy:  100,
		InvitedAt:  invitedAt,
		ExpiresAt:  expiresAt,
		AcceptedAt: &acceptedAt,
		AcceptedBy: &acceptedBy,
	}

	assert.NotNil(t, invitation.AcceptedAt)
	assert.Equal(t, acceptedAt, *invitation.AcceptedAt)
	assert.NotNil(t, invitation.AcceptedBy)
	assert.Equal(t, int64(200), *invitation.AcceptedBy)
}

// Test OrgInvitation JSON serialization
func TestOrgInvitationJSONSerialization(t *testing.T) {
	now := time.Now()
	invitedAt := now.Add(-24 * time.Hour)
	expiresAt := now.Add(7 * 24 * time.Hour)

	invitation := &OrgInvitation{
		ID:        3,
		OrgID:     30,
		Email:     "json@example.com",
		Role:      auth.RoleViewer,
		Token:     "jsontoken",
		InvitedBy: 300,
		InvitedAt: invitedAt,
		ExpiresAt: expiresAt,
	}

	// Marshal to JSON
	data, err := json.Marshal(invitation)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded OrgInvitation
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, invitation.ID, decoded.ID)
	assert.Equal(t, invitation.OrgID, decoded.OrgID)
	assert.Equal(t, invitation.Email, decoded.Email)
	assert.Equal(t, invitation.Role, decoded.Role)
	assert.Equal(t, invitation.Token, decoded.Token)
	assert.Equal(t, invitation.InvitedBy, decoded.InvitedBy)
}

// Test OrgMember struct
func TestOrgMember(t *testing.T) {
	now := time.Now()
	joinedAt := now.Add(-30 * 24 * time.Hour)
	invitedBy := int64(100)

	member := &OrgMember{
		ID:             1,
		OrganizationID: 10,
		UserID:         50,
		Role:           auth.RoleDeveloper,
		InvitedBy:      &invitedBy,
		JoinedAt:       joinedAt,
		CreatedAt:      now,
		Username:       "testuser",
		Email:          "testuser@example.com",
		FullName:       "Test User",
		IsBot:          false,
	}

	assert.Equal(t, int64(1), member.ID)
	assert.Equal(t, int64(10), member.OrganizationID)
	assert.Equal(t, int64(50), member.UserID)
	assert.Equal(t, auth.RoleDeveloper, member.Role)
	assert.NotNil(t, member.InvitedBy)
	assert.Equal(t, int64(100), *member.InvitedBy)
	assert.Equal(t, joinedAt, member.JoinedAt)
	assert.Equal(t, now, member.CreatedAt)
	assert.Equal(t, "testuser", member.Username)
	assert.Equal(t, "testuser@example.com", member.Email)
	assert.Equal(t, "Test User", member.FullName)
	assert.False(t, member.IsBot)
}

// Test OrgMember bot account
func TestOrgMemberBot(t *testing.T) {
	now := time.Now()

	member := &OrgMember{
		ID:             2,
		OrganizationID: 20,
		UserID:         60,
		Role:           auth.RoleViewer,
		InvitedBy:      nil,
		JoinedAt:       now,
		CreatedAt:      now,
		Username:       "bot-user",
		Email:          "",
		FullName:       "Bot User",
		IsBot:          true,
	}

	assert.True(t, member.IsBot)
	assert.Nil(t, member.InvitedBy)
	assert.Empty(t, member.Email)
}

// Test OrgMember JSON serialization
func TestOrgMemberJSONSerialization(t *testing.T) {
	now := time.Now()
	invitedBy := int64(100)

	member := &OrgMember{
		ID:             3,
		OrganizationID: 30,
		UserID:         70,
		Role:           auth.RoleAdmin,
		InvitedBy:      &invitedBy,
		JoinedAt:       now,
		CreatedAt:      now,
		Username:       "jsonuser",
		Email:          "jsonuser@example.com",
		FullName:       "JSON User",
		IsBot:          false,
	}

	// Marshal to JSON
	data, err := json.Marshal(member)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded OrgMember
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, member.ID, decoded.ID)
	assert.Equal(t, member.OrganizationID, decoded.OrganizationID)
	assert.Equal(t, member.UserID, decoded.UserID)
	assert.Equal(t, member.Role, decoded.Role)
	assert.Equal(t, member.Username, decoded.Username)
	assert.Equal(t, member.Email, decoded.Email)
	assert.Equal(t, member.FullName, decoded.FullName)
	assert.Equal(t, member.IsBot, decoded.IsBot)
}

// Test CreateOrgRequest struct
func TestCreateOrgRequest(t *testing.T) {
	req := &CreateOrgRequest{
		Name:        "New Org",
		DisplayName: "New Organization",
		Description: "A new organization",
		QuotaTier:   QuotaTierMedium,
		Settings:    map[string]any{"theme": "light"},
	}

	assert.Equal(t, "New Org", req.Name)
	assert.Equal(t, "New Organization", req.DisplayName)
	assert.Equal(t, "A new organization", req.Description)
	assert.Equal(t, QuotaTierMedium, req.QuotaTier)
	assert.NotNil(t, req.Settings)
	assert.Equal(t, "light", req.Settings["theme"])
}

// Test CreateOrgRequest JSON serialization
func TestCreateOrgRequestJSONSerialization(t *testing.T) {
	req := &CreateOrgRequest{
		Name:        "JSON Org",
		DisplayName: "JSON Organization",
		Description: "JSON test",
		QuotaTier:   QuotaTierLarge,
		Settings:    map[string]any{"key": "value"},
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded CreateOrgRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Name, decoded.Name)
	assert.Equal(t, req.DisplayName, decoded.DisplayName)
	assert.Equal(t, req.Description, decoded.Description)
	assert.Equal(t, req.QuotaTier, decoded.QuotaTier)
	assert.Equal(t, "value", decoded.Settings["key"])
}

// Test UpdateOrgRequest struct
func TestUpdateOrgRequest(t *testing.T) {
	displayName := "Updated Display Name"
	description := "Updated Description"
	settings := map[string]any{"updated": true}

	req := &UpdateOrgRequest{
		DisplayName: &displayName,
		Description: &description,
		Settings:    settings,
	}

	assert.NotNil(t, req.DisplayName)
	assert.Equal(t, "Updated Display Name", *req.DisplayName)
	assert.NotNil(t, req.Description)
	assert.Equal(t, "Updated Description", *req.Description)
	assert.NotNil(t, req.Settings)
	assert.Equal(t, true, req.Settings["updated"])
}

// Test UpdateOrgRequest with nil fields
func TestUpdateOrgRequestNilFields(t *testing.T) {
	req := &UpdateOrgRequest{}

	assert.Nil(t, req.DisplayName)
	assert.Nil(t, req.Description)
	assert.Nil(t, req.Settings)
}

// Test UpdateOrgRequest JSON serialization
func TestUpdateOrgRequestJSONSerialization(t *testing.T) {
	displayName := "JSON Updated"
	description := "JSON Description"

	req := &UpdateOrgRequest{
		DisplayName: &displayName,
		Description: &description,
		Settings:    map[string]any{"json": true},
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded UpdateOrgRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.NotNil(t, decoded.DisplayName)
	assert.Equal(t, *req.DisplayName, *decoded.DisplayName)
	assert.NotNil(t, decoded.Description)
	assert.Equal(t, *req.Description, *decoded.Description)
	assert.Equal(t, true, decoded.Settings["json"])
}

// Test InviteMemberRequest struct
func TestInviteMemberRequest(t *testing.T) {
	req := &InviteMemberRequest{
		Email: "invite@example.com",
		Role:  auth.RoleDeveloper,
	}

	assert.Equal(t, "invite@example.com", req.Email)
	assert.Equal(t, auth.RoleDeveloper, req.Role)
}

// Test InviteMemberRequest JSON serialization
func TestInviteMemberRequestJSONSerialization(t *testing.T) {
	req := &InviteMemberRequest{
		Email: "jsoninvite@example.com",
		Role:  auth.RoleAdmin,
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded InviteMemberRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Email, decoded.Email)
	assert.Equal(t, req.Role, decoded.Role)
}

// Test UpdateMemberRequest struct
func TestUpdateMemberRequest(t *testing.T) {
	req := &UpdateMemberRequest{
		Role: auth.RoleAdmin,
	}

	assert.Equal(t, auth.RoleAdmin, req.Role)
}

// Test UpdateMemberRequest JSON serialization
func TestUpdateMemberRequestJSONSerialization(t *testing.T) {
	req := &UpdateMemberRequest{
		Role: auth.RoleViewer,
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded UpdateMemberRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Role, decoded.Role)
}

// Test QuotaExceededError struct fields
func TestQuotaExceededErrorFields(t *testing.T) {
	err := &QuotaExceededError{
		Resource: "modules",
		Current:  15,
		Limit:    10,
	}

	assert.Equal(t, "modules", err.Resource)
	assert.Equal(t, int64(15), err.Current)
	assert.Equal(t, int64(10), err.Limit)
}

// Test QuotaExceededError Error method variations
func TestQuotaExceededErrorMethodVariations(t *testing.T) {
	tests := []struct {
		name     string
		err      *QuotaExceededError
		contains string
	}{
		{
			name: "storage quota exceeded",
			err: &QuotaExceededError{
				Resource: "storage",
				Current:  1024,
				Limit:    512,
			},
			contains: "quota exceeded for storage",
		},
		{
			name: "api rate limit exceeded",
			err: &QuotaExceededError{
				Resource: "api_requests",
				Current:  10000,
				Limit:    5000,
			},
			contains: "quota exceeded for api_requests",
		},
		{
			name: "compile jobs exceeded",
			err: &QuotaExceededError{
				Resource: "compile_jobs",
				Current:  6000,
				Limit:    5000,
			},
			contains: "quota exceeded for compile_jobs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			assert.Contains(t, msg, tt.contains)
		})
	}
}

// Test IsQuotaExceeded with different error types
func TestIsQuotaExceededDifferentTypes(t *testing.T) {
	quotaErr := &QuotaExceededError{
		Resource: "modules",
		Current:  20,
		Limit:    10,
	}

	regularErr := errors.New("regular error")

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"quota exceeded error", quotaErr, true},
		{"regular error", regularErr, false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsQuotaExceeded(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test different QuotaTier values with quotas
func TestQuotaTierValues(t *testing.T) {
	tests := []struct {
		name string
		tier QuotaTier
	}{
		{"small tier", QuotaTierSmall},
		{"medium tier", QuotaTierMedium},
		{"large tier", QuotaTierLarge},
		{"unlimited tier", QuotaTierUnlimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, string(tt.tier))
		})
	}
}

// Test different OrgStatus values
func TestOrgStatusValues(t *testing.T) {
	tests := []struct {
		name   string
		status OrgStatus
	}{
		{"active status", OrgStatusActive},
		{"suspended status", OrgStatusSuspended},
		{"deleted status", OrgStatusDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, string(tt.status))
		})
	}
}

// Test Organization with all quota tiers
func TestOrganizationAllQuotaTiers(t *testing.T) {
	now := time.Now()
	ownerID := int64(100)

	tiers := []QuotaTier{
		QuotaTierSmall,
		QuotaTierMedium,
		QuotaTierLarge,
		QuotaTierUnlimited,
	}

	for i, tier := range tiers {
		t.Run(string(tier), func(t *testing.T) {
			org := &Organization{
				ID:          int64(i + 1),
				Name:        "Test Org " + string(tier),
				Slug:        "test-org-" + string(tier),
				DisplayName: "Test Organization",
				OwnerID:     &ownerID,
				QuotaTier:   tier,
				Status:      OrgStatusActive,
				IsActive:    true,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			assert.Equal(t, tier, org.QuotaTier)
			assert.Equal(t, OrgStatusActive, org.Status)
		})
	}
}

// Test Organization with all statuses
func TestOrganizationAllStatuses(t *testing.T) {
	now := time.Now()
	ownerID := int64(100)

	statuses := []OrgStatus{
		OrgStatusActive,
		OrgStatusSuspended,
		OrgStatusDeleted,
	}

	for i, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			org := &Organization{
				ID:          int64(i + 1),
				Name:        "Test Org",
				Slug:        "test-org",
				DisplayName: "Test Organization",
				OwnerID:     &ownerID,
				QuotaTier:   QuotaTierSmall,
				Status:      status,
				IsActive:    status == OrgStatusActive,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			assert.Equal(t, status, org.Status)
			assert.Equal(t, status == OrgStatusActive, org.IsActive)
		})
	}
}

// Test OrgInvitation with different roles
func TestOrgInvitationAllRoles(t *testing.T) {
	now := time.Now()

	roles := []auth.Role{
		auth.RoleAdmin,
		auth.RoleDeveloper,
		auth.RoleViewer,
	}

	for i, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			invitation := &OrgInvitation{
				ID:        int64(i + 1),
				OrgID:     10,
				Email:     "test@example.com",
				Role:      role,
				Token:     "token" + string(role),
				InvitedBy: 100,
				InvitedAt: now,
				ExpiresAt: now.Add(7 * 24 * time.Hour),
			}

			assert.Equal(t, role, invitation.Role)
		})
	}
}

// Test OrgMember with different roles
func TestOrgMemberAllRoles(t *testing.T) {
	now := time.Now()

	roles := []auth.Role{
		auth.RoleAdmin,
		auth.RoleDeveloper,
		auth.RoleViewer,
	}

	for i, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			member := &OrgMember{
				ID:             int64(i + 1),
				OrganizationID: 10,
				UserID:         int64(50 + i),
				Role:           role,
				JoinedAt:       now,
				CreatedAt:      now,
				Username:       "user" + string(role),
				IsBot:          false,
			}

			assert.Equal(t, role, member.Role)
		})
	}
}

// Test empty and nil Settings
func TestOrganizationEmptySettings(t *testing.T) {
	now := time.Now()
	ownerID := int64(100)

	tests := []struct {
		name     string
		settings map[string]any
	}{
		{"nil settings", nil},
		{"empty settings", map[string]any{}},
		{"with settings", map[string]any{"key": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := &Organization{
				ID:          1,
				Name:        "Test",
				Slug:        "test",
				DisplayName: "Test Org",
				OwnerID:     &ownerID,
				QuotaTier:   QuotaTierSmall,
				Status:      OrgStatusActive,
				IsActive:    true,
				Settings:    tt.settings,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			if tt.settings == nil {
				assert.Nil(t, org.Settings)
			} else {
				assert.NotNil(t, org.Settings)
			}
		})
	}
}

// Test OrgQuotas with nil CustomSettings
func TestOrgQuotasNilCustomSettings(t *testing.T) {
	now := time.Now()

	quotas := &OrgQuotas{
		ID:                     1,
		OrgID:                  10,
		MaxModules:             50,
		MaxVersionsPerModule:   500,
		MaxStorageBytes:        25 * 1024 * 1024 * 1024,
		MaxCompileJobsPerMonth: 25000,
		APIRateLimitPerHour:    25000,
		CustomSettings:         nil,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	assert.Nil(t, quotas.CustomSettings)

	// Test JSON serialization with nil custom settings
	data, err := json.Marshal(quotas)
	require.NoError(t, err)

	var decoded OrgQuotas
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Nil(t, decoded.CustomSettings)
}

// Test struct field tags
func TestOrganizationJSONTags(t *testing.T) {
	now := time.Now()
	ownerID := int64(123)

	org := &Organization{
		ID:          1,
		Name:        "Test",
		Slug:        "test",
		DisplayName: "Test Org",
		Description: "",
		OwnerID:     &ownerID,
		QuotaTier:   QuotaTierSmall,
		Status:      OrgStatusActive,
		IsActive:    true,
		Settings:    nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(org)
	require.NoError(t, err)

	// Verify JSON contains expected fields
	jsonStr := string(data)
	assert.Contains(t, jsonStr, "\"id\":")
	assert.Contains(t, jsonStr, "\"name\":")
	assert.Contains(t, jsonStr, "\"slug\":")
	assert.Contains(t, jsonStr, "\"display_name\":")
	assert.Contains(t, jsonStr, "\"quota_tier\":")
	assert.Contains(t, jsonStr, "\"status\":")
	assert.Contains(t, jsonStr, "\"is_active\":")
	assert.Contains(t, jsonStr, "\"created_at\":")
	assert.Contains(t, jsonStr, "\"updated_at\":")
}

// Test CreateOrgRequest with minimal fields
func TestCreateOrgRequestMinimal(t *testing.T) {
	req := &CreateOrgRequest{
		Name:        "Minimal Org",
		DisplayName: "Minimal Organization",
	}

	assert.Equal(t, "Minimal Org", req.Name)
	assert.Equal(t, "Minimal Organization", req.DisplayName)
	assert.Empty(t, req.Description)
	assert.Empty(t, req.QuotaTier)
	assert.Nil(t, req.Settings)
}

// Test UpdateOrgRequest partial updates
func TestUpdateOrgRequestPartialUpdates(t *testing.T) {
	t.Run("only display name", func(t *testing.T) {
		displayName := "New Name"
		req := &UpdateOrgRequest{
			DisplayName: &displayName,
		}
		assert.NotNil(t, req.DisplayName)
		assert.Nil(t, req.Description)
		assert.Nil(t, req.Settings)
	})

	t.Run("only description", func(t *testing.T) {
		description := "New Description"
		req := &UpdateOrgRequest{
			Description: &description,
		}
		assert.Nil(t, req.DisplayName)
		assert.NotNil(t, req.Description)
		assert.Nil(t, req.Settings)
	})

	t.Run("only settings", func(t *testing.T) {
		req := &UpdateOrgRequest{
			Settings: map[string]any{"key": "value"},
		}
		assert.Nil(t, req.DisplayName)
		assert.Nil(t, req.Description)
		assert.NotNil(t, req.Settings)
	})
}

// Test complex nested Settings
func TestOrganizationComplexSettings(t *testing.T) {
	now := time.Now()
	ownerID := int64(100)

	complexSettings := map[string]any{
		"theme": "dark",
		"notifications": map[string]any{
			"email":  true,
			"slack":  false,
			"webhook": "https://example.com/webhook",
		},
		"features": []string{"feature1", "feature2", "feature3"},
		"limits": map[string]any{
			"max_users": 100,
			"max_projects": 50,
		},
	}

	org := &Organization{
		ID:          1,
		Name:        "Complex Settings Org",
		Slug:        "complex-settings",
		DisplayName: "Complex Settings Organization",
		OwnerID:     &ownerID,
		QuotaTier:   QuotaTierLarge,
		Status:      OrgStatusActive,
		IsActive:    true,
		Settings:    complexSettings,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Marshal to JSON
	data, err := json.Marshal(org)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Organization
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "dark", decoded.Settings["theme"])
	assert.NotNil(t, decoded.Settings["notifications"])
	assert.NotNil(t, decoded.Settings["features"])
	assert.NotNil(t, decoded.Settings["limits"])
}

// Test zero values for numeric fields
func TestOrgUsageZeroValues(t *testing.T) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	usage := &OrgUsage{
		ID:               1,
		OrgID:            10,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		ModulesCount:     0,
		VersionsCount:    0,
		StorageBytes:     0,
		CompileJobsCount: 0,
		APIRequestsCount: 0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	assert.Equal(t, 0, usage.ModulesCount)
	assert.Equal(t, 0, usage.VersionsCount)
	assert.Equal(t, int64(0), usage.StorageBytes)
	assert.Equal(t, 0, usage.CompileJobsCount)
	assert.Equal(t, int64(0), usage.APIRequestsCount)
}

// Test OrgQuotas with large values
func TestOrgQuotasLargeValues(t *testing.T) {
	now := time.Now()

	quotas := &OrgQuotas{
		ID:                       1,
		OrgID:                    1,
		MaxModules:               999999,
		MaxVersionsPerModule:     999999,
		MaxStorageBytes:          999999 * 1024 * 1024 * 1024,
		MaxCompileJobsPerMonth:   999999999,
		APIRateLimitPerHour:      999999999,
		CustomSettings:           nil,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	assert.Equal(t, 999999, quotas.MaxModules)
	assert.Equal(t, 999999, quotas.MaxVersionsPerModule)
	assert.Equal(t, int64(999999*1024*1024*1024), quotas.MaxStorageBytes)
	assert.Equal(t, 999999999, quotas.MaxCompileJobsPerMonth)
	assert.Equal(t, 999999999, quotas.APIRateLimitPerHour)
}
