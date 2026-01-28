package auth

import (
	"testing"
	"time"
)

// MockPermissionChecker is a mock implementation of PermissionChecker for testing
type MockPermissionChecker struct {
	HasPermissionFunc func(userID int64, resource, action string, scope string, resourceID *string, organizationID *int64) bool
}

func (m *MockPermissionChecker) HasPermission(userID int64, resource, action string, scope string, resourceID *string, organizationID *int64) bool {
	if m.HasPermissionFunc != nil {
		return m.HasPermissionFunc(userID, resource, action, scope, resourceID, organizationID)
	}
	return false
}

func TestAuthContext_HasRole(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{
			name: "admin role",
			role: RoleAdmin,
			want: false, // Currently always returns false (legacy)
		},
		{
			name: "developer role",
			role: RoleDeveloper,
			want: false,
		},
		{
			name: "viewer role",
			role: RoleViewer,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &AuthContext{}
			if got := ac.HasRole(tt.role); got != tt.want {
				t.Errorf("AuthContext.HasRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthContext_HasPermission(t *testing.T) {
	tests := []struct {
		name       string
		moduleID   int64
		permission Permission
		want       bool
	}{
		{
			name:       "check read permission",
			moduleID:   123,
			permission: PermissionRead,
			want:       false, // Currently always returns false (legacy)
		},
		{
			name:       "check write permission",
			moduleID:   456,
			permission: PermissionWrite,
			want:       false,
		},
		{
			name:       "check delete permission",
			moduleID:   789,
			permission: PermissionDelete,
			want:       false,
		},
		{
			name:       "check admin permission",
			moduleID:   999,
			permission: PermissionAdmin,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &AuthContext{}
			if got := ac.HasPermission(tt.moduleID, tt.permission); got != tt.want {
				t.Errorf("AuthContext.HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_StructFields(t *testing.T) {
	now := time.Now()
	lastLogin := time.Now().Add(-1 * time.Hour)

	user := User{
		ID:          1,
		Username:    "testuser",
		Email:       "test@example.com",
		FullName:    "Test User",
		IsBot:       false,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
		LastLoginAt: &lastLogin,
	}

	if user.ID != 1 {
		t.Errorf("User.ID = %d, want 1", user.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("User.Username = %s, want testuser", user.Username)
	}
	if user.Email != "test@example.com" {
		t.Errorf("User.Email = %s, want test@example.com", user.Email)
	}
	if user.FullName != "Test User" {
		t.Errorf("User.FullName = %s, want Test User", user.FullName)
	}
	if user.IsBot != false {
		t.Errorf("User.IsBot = %v, want false", user.IsBot)
	}
	if user.IsActive != true {
		t.Errorf("User.IsActive = %v, want true", user.IsActive)
	}
	if user.LastLoginAt == nil {
		t.Error("User.LastLoginAt should not be nil")
	}
}

func TestOrganization_StructFields(t *testing.T) {
	now := time.Now()

	org := Organization{
		ID:          1,
		Name:        "test-org",
		DisplayName: "Test Organization",
		Description: "A test organization",
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if org.ID != 1 {
		t.Errorf("Organization.ID = %d, want 1", org.ID)
	}
	if org.Name != "test-org" {
		t.Errorf("Organization.Name = %s, want test-org", org.Name)
	}
	if org.DisplayName != "Test Organization" {
		t.Errorf("Organization.DisplayName = %s, want Test Organization", org.DisplayName)
	}
	if org.Description != "A test organization" {
		t.Errorf("Organization.Description = %s, want A test organization", org.Description)
	}
	if org.IsActive != true {
		t.Errorf("Organization.IsActive = %v, want true", org.IsActive)
	}
}

func TestAPIToken_StructFields(t *testing.T) {
	now := time.Now()
	expiresAt := time.Now().Add(24 * time.Hour)
	lastUsedAt := time.Now().Add(-1 * time.Hour)
	revokedAt := time.Now().Add(-30 * time.Minute)
	revokedBy := int64(999)

	token := APIToken{
		ID:           1,
		UserID:       123,
		TokenHash:    "hash123",
		TokenPrefix:  "spoke_abc123",
		Name:         "Test Token",
		Description:  "A test token",
		Scopes:       []Scope{ScopeModuleRead, ScopeModuleWrite},
		ExpiresAt:    &expiresAt,
		LastUsedAt:   &lastUsedAt,
		CreatedAt:    now,
		RevokedAt:    &revokedAt,
		RevokedBy:    &revokedBy,
		RevokeReason: "Test revocation",
	}

	if token.ID != 1 {
		t.Errorf("APIToken.ID = %d, want 1", token.ID)
	}
	if token.UserID != 123 {
		t.Errorf("APIToken.UserID = %d, want 123", token.UserID)
	}
	if token.TokenHash != "hash123" {
		t.Errorf("APIToken.TokenHash = %s, want hash123", token.TokenHash)
	}
	if token.TokenPrefix != "spoke_abc123" {
		t.Errorf("APIToken.TokenPrefix = %s, want spoke_abc123", token.TokenPrefix)
	}
	if token.Name != "Test Token" {
		t.Errorf("APIToken.Name = %s, want Test Token", token.Name)
	}
	if len(token.Scopes) != 2 {
		t.Errorf("APIToken.Scopes length = %d, want 2", len(token.Scopes))
	}
	if token.ExpiresAt == nil {
		t.Error("APIToken.ExpiresAt should not be nil")
	}
	if token.LastUsedAt == nil {
		t.Error("APIToken.LastUsedAt should not be nil")
	}
	if token.RevokedAt == nil {
		t.Error("APIToken.RevokedAt should not be nil")
	}
	if token.RevokedBy == nil || *token.RevokedBy != 999 {
		t.Errorf("APIToken.RevokedBy = %v, want 999", token.RevokedBy)
	}
	if token.RevokeReason != "Test revocation" {
		t.Errorf("APIToken.RevokeReason = %s, want Test revocation", token.RevokeReason)
	}
}

func TestModulePermission_StructFields(t *testing.T) {
	now := time.Now()
	userID := int64(123)
	orgID := int64(456)
	grantedBy := int64(789)

	mp := ModulePermission{
		ID:             1,
		ModuleID:       100,
		UserID:         &userID,
		OrganizationID: &orgID,
		Permission:     PermissionRead,
		GrantedAt:      now,
		GrantedBy:      &grantedBy,
	}

	if mp.ID != 1 {
		t.Errorf("ModulePermission.ID = %d, want 1", mp.ID)
	}
	if mp.ModuleID != 100 {
		t.Errorf("ModulePermission.ModuleID = %d, want 100", mp.ModuleID)
	}
	if mp.UserID == nil || *mp.UserID != 123 {
		t.Errorf("ModulePermission.UserID = %v, want 123", mp.UserID)
	}
	if mp.OrganizationID == nil || *mp.OrganizationID != 456 {
		t.Errorf("ModulePermission.OrganizationID = %v, want 456", mp.OrganizationID)
	}
	if mp.Permission != PermissionRead {
		t.Errorf("ModulePermission.Permission = %s, want %s", mp.Permission, PermissionRead)
	}
	if mp.GrantedBy == nil || *mp.GrantedBy != 789 {
		t.Errorf("ModulePermission.GrantedBy = %v, want 789", mp.GrantedBy)
	}
}

func TestAuditLog_StructFields(t *testing.T) {
	now := time.Now()
	userID := int64(123)
	orgID := int64(456)

	audit := AuditLog{
		ID:             1,
		UserID:         &userID,
		OrganizationID: &orgID,
		Action:         "create",
		ResourceType:   "module",
		ResourceID:     "module-123",
		IPAddress:      "192.168.1.1",
		UserAgent:      "TestAgent/1.0",
		Status:         "success",
		ErrorMessage:   "",
		CreatedAt:      now,
	}

	if audit.ID != 1 {
		t.Errorf("AuditLog.ID = %d, want 1", audit.ID)
	}
	if audit.UserID == nil || *audit.UserID != 123 {
		t.Errorf("AuditLog.UserID = %v, want 123", audit.UserID)
	}
	if audit.OrganizationID == nil || *audit.OrganizationID != 456 {
		t.Errorf("AuditLog.OrganizationID = %v, want 456", audit.OrganizationID)
	}
	if audit.Action != "create" {
		t.Errorf("AuditLog.Action = %s, want create", audit.Action)
	}
	if audit.ResourceType != "module" {
		t.Errorf("AuditLog.ResourceType = %s, want module", audit.ResourceType)
	}
	if audit.ResourceID != "module-123" {
		t.Errorf("AuditLog.ResourceID = %s, want module-123", audit.ResourceID)
	}
	if audit.IPAddress != "192.168.1.1" {
		t.Errorf("AuditLog.IPAddress = %s, want 192.168.1.1", audit.IPAddress)
	}
	if audit.Status != "success" {
		t.Errorf("AuditLog.Status = %s, want success", audit.Status)
	}
}

func TestRateLimitBucket_StructFields(t *testing.T) {
	now := time.Now()
	userID := int64(123)
	orgID := int64(456)

	bucket := RateLimitBucket{
		ID:                    1,
		UserID:                &userID,
		OrganizationID:        &orgID,
		Endpoint:              "/api/v1/modules",
		RequestsCount:         10,
		WindowStart:           now,
		WindowDurationSeconds: 60,
	}

	if bucket.ID != 1 {
		t.Errorf("RateLimitBucket.ID = %d, want 1", bucket.ID)
	}
	if bucket.UserID == nil || *bucket.UserID != 123 {
		t.Errorf("RateLimitBucket.UserID = %v, want 123", bucket.UserID)
	}
	if bucket.OrganizationID == nil || *bucket.OrganizationID != 456 {
		t.Errorf("RateLimitBucket.OrganizationID = %v, want 456", bucket.OrganizationID)
	}
	if bucket.Endpoint != "/api/v1/modules" {
		t.Errorf("RateLimitBucket.Endpoint = %s, want /api/v1/modules", bucket.Endpoint)
	}
	if bucket.RequestsCount != 10 {
		t.Errorf("RateLimitBucket.RequestsCount = %d, want 10", bucket.RequestsCount)
	}
	if bucket.WindowDurationSeconds != 60 {
		t.Errorf("RateLimitBucket.WindowDurationSeconds = %d, want 60", bucket.WindowDurationSeconds)
	}
}

func TestAuthContext_StructFields(t *testing.T) {
	user := &User{
		ID:       1,
		Username: "testuser",
	}
	org := &Organization{
		ID:   1,
		Name: "test-org",
	}
	token := &APIToken{
		ID:     1,
		UserID: 1,
		Name:   "test-token",
	}
	scopes := []Scope{ScopeModuleRead, ScopeModuleWrite}
	checker := &MockPermissionChecker{}

	ctx := AuthContext{
		User:            user,
		Organization:    org,
		Token:           token,
		Scopes:          scopes,
		PermissionCheck: checker,
	}

	if ctx.User == nil || ctx.User.ID != 1 {
		t.Errorf("AuthContext.User.ID = %v, want 1", ctx.User)
	}
	if ctx.Organization == nil || ctx.Organization.ID != 1 {
		t.Errorf("AuthContext.Organization.ID = %v, want 1", ctx.Organization)
	}
	if ctx.Token == nil || ctx.Token.ID != 1 {
		t.Errorf("AuthContext.Token.ID = %v, want 1", ctx.Token)
	}
	if len(ctx.Scopes) != 2 {
		t.Errorf("AuthContext.Scopes length = %d, want 2", len(ctx.Scopes))
	}
	if ctx.PermissionCheck == nil {
		t.Error("AuthContext.PermissionCheck should not be nil")
	}
}

func TestRole_Values(t *testing.T) {
	tests := []struct {
		role Role
		want string
	}{
		{RoleAdmin, "admin"},
		{RoleDeveloper, "developer"},
		{RoleViewer, "viewer"},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if string(tt.role) != tt.want {
				t.Errorf("Role value = %s, want %s", tt.role, tt.want)
			}
		})
	}
}

func TestPermission_Values(t *testing.T) {
	tests := []struct {
		perm Permission
		want string
	}{
		{PermissionRead, "read"},
		{PermissionWrite, "write"},
		{PermissionDelete, "delete"},
		{PermissionAdmin, "admin"},
	}

	for _, tt := range tests {
		t.Run(string(tt.perm), func(t *testing.T) {
			if string(tt.perm) != tt.want {
				t.Errorf("Permission value = %s, want %s", tt.perm, tt.want)
			}
		})
	}
}

func TestScope_Values(t *testing.T) {
	tests := []struct {
		scope Scope
		want  string
	}{
		{ScopeModuleRead, "module:read"},
		{ScopeModuleWrite, "module:write"},
		{ScopeModuleDelete, "module:delete"},
		{ScopeVersionRead, "version:read"},
		{ScopeVersionWrite, "version:write"},
		{ScopeVersionDelete, "version:delete"},
		{ScopeTokenCreate, "token:create"},
		{ScopeTokenRevoke, "token:revoke"},
		{ScopeOrgRead, "org:read"},
		{ScopeOrgWrite, "org:write"},
		{ScopeUserRead, "user:read"},
		{ScopeUserWrite, "user:write"},
		{ScopeAuditRead, "audit:read"},
		{ScopeAll, "*"},
	}

	for _, tt := range tests {
		t.Run(string(tt.scope), func(t *testing.T) {
			if string(tt.scope) != tt.want {
				t.Errorf("Scope value = %s, want %s", tt.scope, tt.want)
			}
		})
	}
}

func TestMockPermissionChecker(t *testing.T) {
	called := false
	checker := &MockPermissionChecker{
		HasPermissionFunc: func(userID int64, resource, action string, scope string, resourceID *string, organizationID *int64) bool {
			called = true
			return userID == 123 && resource == "module" && action == "read"
		},
	}

	result := checker.HasPermission(123, "module", "read", "test-scope", nil, nil)
	if !result {
		t.Error("MockPermissionChecker should return true for matching parameters")
	}
	if !called {
		t.Error("MockPermissionChecker.HasPermissionFunc was not called")
	}

	// Test with non-matching parameters
	result = checker.HasPermission(456, "module", "read", "test-scope", nil, nil)
	if result {
		t.Error("MockPermissionChecker should return false for non-matching parameters")
	}
}

func TestMockPermissionChecker_DefaultBehavior(t *testing.T) {
	checker := &MockPermissionChecker{}
	result := checker.HasPermission(123, "module", "read", "test-scope", nil, nil)
	if result {
		t.Error("MockPermissionChecker should return false by default when HasPermissionFunc is nil")
	}
}
