package auth

import (
	"strings"
	"testing"
	"time"
)

func TestTokenGenerator_GenerateToken(t *testing.T) {
	tg := NewTokenGenerator()

	token, tokenHash, tokenPrefix, err := tg.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Check token format
	if !strings.HasPrefix(token, TokenPrefix) {
		t.Errorf("Token should start with %q, got %q", TokenPrefix, token)
	}

	// Check hash length (SHA256 = 64 hex chars)
	if len(tokenHash) != 64 {
		t.Errorf("TokenHash length = %d, want 64", len(tokenHash))
	}

	// Check prefix format
	if !strings.HasPrefix(tokenPrefix, TokenPrefix) {
		t.Errorf("TokenPrefix should start with %q, got %q", TokenPrefix, tokenPrefix)
	}

	// Token should be long enough
	if len(token) < len(TokenPrefix)+8 {
		t.Errorf("Token too short: %d chars", len(token))
	}
}

func TestTokenGenerator_GenerateToken_Uniqueness(t *testing.T) {
	tg := NewTokenGenerator()

	// Generate multiple tokens and ensure they're unique
	tokens := make(map[string]bool)
	hashes := make(map[string]bool)

	for i := 0; i < 100; i++ {
		token, tokenHash, _, err := tg.GenerateToken()
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		if tokens[token] {
			t.Errorf("Duplicate token generated: %s", token)
		}
		if hashes[tokenHash] {
			t.Errorf("Duplicate token hash generated: %s", tokenHash)
		}

		tokens[token] = true
		hashes[tokenHash] = true
	}
}

func TestTokenGenerator_HashToken(t *testing.T) {
	tg := NewTokenGenerator()

	token := "spoke_test123456789"
	hash1 := tg.HashToken(token)
	hash2 := tg.HashToken(token)

	// Same token should produce same hash
	if hash1 != hash2 {
		t.Error("Same token should produce same hash")
	}

	// Hash should be 64 chars (SHA256)
	if len(hash1) != 64 {
		t.Errorf("Hash length = %d, want 64", len(hash1))
	}

	// Different tokens should produce different hashes
	hash3 := tg.HashToken("spoke_different")
	if hash1 == hash3 {
		t.Error("Different tokens should produce different hashes")
	}
}

func TestTokenGenerator_ValidateTokenFormat(t *testing.T) {
	tg := NewTokenGenerator()

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   "spoke_abc123def456",
			wantErr: false,
		},
		{
			name:    "missing prefix",
			token:   "abc123def456",
			wantErr: true,
		},
		{
			name:    "wrong prefix",
			token:   "other_abc123def456",
			wantErr: true,
		},
		{
			name:    "empty token part",
			token:   "spoke_",
			wantErr: true,
		},
		{
			name:    "invalid base64",
			token:   "spoke_!!!invalid!!!",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tg.ValidateTokenFormat(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTokenFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTokenGenerator_ExtractPrefix(t *testing.T) {
	tg := NewTokenGenerator()

	tests := []struct {
		name   string
		token  string
		want   string
	}{
		{
			name:  "normal token",
			token: "spoke_abc123def456",
			want:  "spoke_abc123de",
		},
		{
			name:  "short token",
			token: "spoke_abc",
			want:  "spoke_abc",
		},
		{
			name:  "no prefix",
			token: "invalid",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tg.ExtractPrefix(tt.token)
			if got != tt.want {
				t.Errorf("ExtractPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTokenManager_CreateToken(t *testing.T) {
	tm := NewTokenManager()

	userID := int64(123)
	name := "Test Token"
	description := "Token for testing"
	scopes := []Scope{ScopeModuleRead, ScopeModuleWrite}
	expiresAt := time.Now().Add(24 * time.Hour)

	apiToken, token, err := tm.CreateToken(userID, name, description, scopes, &expiresAt)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	// Check API token fields
	if apiToken.UserID != userID {
		t.Errorf("UserID = %d, want %d", apiToken.UserID, userID)
	}
	if apiToken.Name != name {
		t.Errorf("Name = %q, want %q", apiToken.Name, name)
	}
	if apiToken.Description != description {
		t.Errorf("Description = %q, want %q", apiToken.Description, description)
	}
	if len(apiToken.Scopes) != len(scopes) {
		t.Errorf("Scopes count = %d, want %d", len(apiToken.Scopes), len(scopes))
	}

	// Check token format
	if !strings.HasPrefix(token, TokenPrefix) {
		t.Errorf("Token should start with %q", TokenPrefix)
	}

	// Check token hash is set
	if apiToken.TokenHash == "" {
		t.Error("TokenHash should not be empty")
	}

	// Check token prefix is set
	if apiToken.TokenPrefix == "" {
		t.Error("TokenPrefix should not be empty")
	}
}

func TestAuthContext_HasScope(t *testing.T) {
	tests := []struct {
		name       string
		userScopes []Scope
		checkScope Scope
		want       bool
	}{
		{
			name:       "has specific scope",
			userScopes: []Scope{ScopeModuleRead, ScopeModuleWrite},
			checkScope: ScopeModuleRead,
			want:       true,
		},
		{
			name:       "missing scope",
			userScopes: []Scope{ScopeModuleRead},
			checkScope: ScopeModuleWrite,
			want:       false,
		},
		{
			name:       "wildcard scope",
			userScopes: []Scope{ScopeAll},
			checkScope: ScopeModuleDelete,
			want:       true,
		},
		{
			name:       "no scopes",
			userScopes: []Scope{},
			checkScope: ScopeModuleRead,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authCtx := &AuthContext{
				Scopes: tt.userScopes,
			}
			got := authCtx.HasScope(tt.checkScope)
			if got != tt.want {
				t.Errorf("HasScope(%v) = %v, want %v", tt.checkScope, got, tt.want)
			}
		})
	}
}

func TestRole_Constants(t *testing.T) {
	// Ensure role constants are defined
	roles := []Role{RoleAdmin, RoleDeveloper, RoleViewer}
	if len(roles) != 3 {
		t.Error("Should have 3 role constants")
	}

	if string(RoleAdmin) != "admin" {
		t.Errorf("RoleAdmin = %q, want %q", RoleAdmin, "admin")
	}
	if string(RoleDeveloper) != "developer" {
		t.Errorf("RoleDeveloper = %q, want %q", RoleDeveloper, "developer")
	}
	if string(RoleViewer) != "viewer" {
		t.Errorf("RoleViewer = %q, want %q", RoleViewer, "viewer")
	}
}

func TestPermission_Constants(t *testing.T) {
	// Ensure permission constants are defined
	perms := []Permission{PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin}
	if len(perms) != 4 {
		t.Error("Should have 4 permission constants")
	}
}

func TestScope_Constants(t *testing.T) {
	// Ensure common scope constants are defined
	scopes := []Scope{
		ScopeModuleRead, ScopeModuleWrite, ScopeModuleDelete,
		ScopeVersionRead, ScopeVersionWrite, ScopeVersionDelete,
		ScopeTokenCreate, ScopeTokenRevoke,
		ScopeAll,
	}
	if len(scopes) < 9 {
		t.Error("Should have at least 9 scope constants")
	}

	if string(ScopeAll) != "*" {
		t.Errorf("ScopeAll = %q, want %q", ScopeAll, "*")
	}
}
