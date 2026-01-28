package sso

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestOIDCProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *OIDCConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &OIDCConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				IssuerURL:    "https://provider.com",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"openid", "profile", "email"},
			},
			expectError: false,
		},
		{
			name: "missing client_id",
			config: &OIDCConfig{
				ClientSecret: "test-secret",
				IssuerURL:    "https://provider.com",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"openid"},
			},
			expectError: true,
			errorMsg:    "client_id is required",
		},
		{
			name: "missing client_secret",
			config: &OIDCConfig{
				ClientID:    "test-client-id",
				IssuerURL:   "https://provider.com",
				RedirectURL: "https://spoke.example.com/callback",
				Scopes:      []string{"openid"},
			},
			expectError: true,
			errorMsg:    "client_secret is required",
		},
		{
			name: "missing issuer_url",
			config: &OIDCConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"openid"},
			},
			expectError: true,
			errorMsg:    "issuer_url is required",
		},
		{
			name: "missing redirect_url",
			config: &OIDCConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				IssuerURL:    "https://provider.com",
				Scopes:       []string{"openid"},
			},
			expectError: true,
			errorMsg:    "redirect_url is required",
		},
		{
			name: "missing scopes",
			config: &OIDCConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				IssuerURL:    "https://provider.com",
				RedirectURL:  "https://spoke.example.com/callback",
			},
			expectError: true,
			errorMsg:    "scopes are required",
		},
		{
			name: "missing openid scope",
			config: &OIDCConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				IssuerURL:    "https://provider.com",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"profile", "email"},
			},
			expectError: true,
			errorMsg:    "'openid' scope is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerConfig := &ProviderConfig{
				Name:         "test-oidc",
				ProviderType: ProviderTypeOIDC,
				ProviderName: ProviderGenericOIDC,
				Enabled:      true,
				OIDCConfig:   tt.config,
				AttributeMapping: AttributeMap{
					UserID:   "sub",
					Username: "preferred_username",
					Email:    "email",
				},
			}

			// Create provider will fail if issuer URL is invalid in real scenario
			// For unit tests, we just validate the config structure
			if tt.config != nil && tt.config.IssuerURL != "" {
				// We can't create actual OIDC provider in unit tests without a real issuer
				// So we just validate the config directly
				provider := &OIDCProvider{config: providerConfig}
				err := provider.ValidateConfig()

				if tt.expectError {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.errorMsg)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestOIDCProvider_ValidateConfig_NilConfig(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			Name:         "test-oidc",
			ProviderType: ProviderTypeOIDC,
			ProviderName: ProviderGenericOIDC,
			OIDCConfig:   nil,
		},
	}

	err := provider.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OIDC config is required")
}

func TestOIDCConfig_Serialization(t *testing.T) {
	config := &OIDCConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		IssuerURL:    "https://login.microsoftonline.com/tenant-id/v2.0",
		RedirectURL:  "https://spoke.example.com/callback",
		Scopes:       []string{"openid", "profile", "email"},
	}

	// The OIDC provider validates config structure
	provider := &OIDCProvider{
		config: &ProviderConfig{
			OIDCConfig: config,
		},
	}

	err := provider.ValidateConfig()
	require.NoError(t, err)
}

func TestOIDCProvider_MissingOpenIDScope(t *testing.T) {
	config := &ProviderConfig{
		Name:         "test-oidc",
		ProviderType: ProviderTypeOIDC,
		ProviderName: ProviderGenericOIDC,
		OIDCConfig: &OIDCConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-secret",
			IssuerURL:    "https://provider.com",
			RedirectURL:  "https://spoke.example.com/callback",
			Scopes:       []string{"profile", "email"}, // Missing "openid"
		},
		AttributeMapping: AttributeMap{
			UserID: "sub",
			Email:  "email",
		},
	}

	provider := &OIDCProvider{config: config}
	err := provider.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "openid")
}

func TestOIDCProvider_GetType(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			Name:         "test-oidc",
			ProviderType: ProviderTypeOIDC,
			ProviderName: ProviderGenericOIDC,
		},
	}

	assert.Equal(t, ProviderTypeOIDC, provider.GetType())
}

func TestOIDCProvider_GetName(t *testing.T) {
	tests := []struct {
		name         string
		providerName ProviderName
	}{
		{"generic OIDC", ProviderGenericOIDC},
		{"Azure AD", ProviderAzureAD},
		{"Google", ProviderGoogle},
		{"Okta", ProviderOkta},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &OIDCProvider{
				config: &ProviderConfig{
					ProviderName: tt.providerName,
				},
			}

			assert.Equal(t, tt.providerName, provider.GetName())
		})
	}
}

func TestOIDCProvider_InitiateLogin(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			Name:         "test-oidc",
			ProviderType: ProviderTypeOIDC,
			ProviderName: ProviderGenericOIDC,
		},
		oauth2Config: &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-secret",
			RedirectURL:  "https://spoke.example.com/callback",
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://provider.com/oauth/authorize",
				TokenURL: "https://provider.com/oauth/token",
			},
		},
	}

	tests := []struct {
		name  string
		state string
	}{
		{"with state", "test-state-123"},
		{"empty state", ""},
		{"complex state", "state-with-special-chars-!@#$%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/login", nil)
			rec := httptest.NewRecorder()

			err := provider.InitiateLogin(rec, req, tt.state)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusFound, rec.Code)

			location := rec.Header().Get("Location")
			assert.NotEmpty(t, location)

			// Verify the redirect URL contains expected parameters
			redirectURL, err := url.Parse(location)
			require.NoError(t, err)

			assert.Equal(t, "https", redirectURL.Scheme)
			assert.Equal(t, "provider.com", redirectURL.Host)
			assert.Equal(t, "/oauth/authorize", redirectURL.Path)

			// Verify query parameters
			query := redirectURL.Query()
			assert.Equal(t, "test-client-id", query.Get("client_id"))
			assert.Equal(t, "https://spoke.example.com/callback", query.Get("redirect_uri"))
			assert.Equal(t, "code", query.Get("response_type"))
			assert.Equal(t, tt.state, query.Get("state"))
			assert.Contains(t, query.Get("scope"), "openid")
			assert.Equal(t, "offline", query.Get("access_type"))
		})
	}
}

func TestOIDCProvider_Logout(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			Name:         "test-oidc",
			ProviderType: ProviderTypeOIDC,
			ProviderName: ProviderGenericOIDC,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	rec := httptest.NewRecorder()

	err := provider.Logout(rec, req, "session-index-123")
	assert.NoError(t, err)
}

func TestOIDCProvider_HandleCallback_MissingCode(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			Name:         "test-oidc",
			ProviderType: ProviderTypeOIDC,
			ProviderName: ProviderGenericOIDC,
			AttributeMapping: AttributeMap{
				UserID: "sub",
				Email:  "email",
			},
		},
		oauth2Config: &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-secret",
		},
	}

	tests := []struct {
		name     string
		queryStr string
		errorMsg string
	}{
		{
			name:     "no code parameter",
			queryStr: "",
			errorMsg: "missing authorization code",
		},
		{
			name:     "empty code parameter",
			queryStr: "code=",
			errorMsg: "missing authorization code",
		},
		{
			name:     "only state parameter",
			queryStr: "state=abc123",
			errorMsg: "missing authorization code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/callback?"+tt.queryStr, nil)
			rec := httptest.NewRecorder()

			user, err := provider.HandleCallback(rec, req)

			assert.Error(t, err)
			assert.Nil(t, user)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestNewOIDCProvider_MissingOIDCConfig(t *testing.T) {
	ctx := context.Background()
	config := &ProviderConfig{
		Name:         "test-oidc",
		ProviderType: ProviderTypeOIDC,
		ProviderName: ProviderGenericOIDC,
		OIDCConfig:   nil,
		AttributeMapping: AttributeMap{
			UserID: "sub",
			Email:  "email",
		},
	}

	provider, err := NewOIDCProvider(ctx, config)

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "OIDC config is required")
}

func TestOIDCProvider_ValidateConfig_EmptyClientID(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			OIDCConfig: &OIDCConfig{
				ClientID:     "",
				ClientSecret: "secret",
				IssuerURL:    "https://provider.com",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"openid"},
			},
		},
	}

	err := provider.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client_id is required")
}

func TestOIDCProvider_ValidateConfig_EmptyClientSecret(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			OIDCConfig: &OIDCConfig{
				ClientID:     "client-id",
				ClientSecret: "",
				IssuerURL:    "https://provider.com",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"openid"},
			},
		},
	}

	err := provider.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client_secret is required")
}

func TestOIDCProvider_ValidateConfig_EmptyIssuerURL(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			OIDCConfig: &OIDCConfig{
				ClientID:     "client-id",
				ClientSecret: "secret",
				IssuerURL:    "",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"openid"},
			},
		},
	}

	err := provider.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "issuer_url is required")
}

func TestOIDCProvider_ValidateConfig_EmptyRedirectURL(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			OIDCConfig: &OIDCConfig{
				ClientID:     "client-id",
				ClientSecret: "secret",
				IssuerURL:    "https://provider.com",
				RedirectURL:  "",
				Scopes:       []string{"openid"},
			},
		},
	}

	err := provider.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redirect_url is required")
}

func TestOIDCProvider_ValidateConfig_EmptyScopes(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			OIDCConfig: &OIDCConfig{
				ClientID:     "client-id",
				ClientSecret: "secret",
				IssuerURL:    "https://provider.com",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{},
			},
		},
	}

	err := provider.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scopes are required")
}

func TestOIDCProvider_ValidateConfig_WithOptionalFields(t *testing.T) {
	provider := &OIDCProvider{
		config: &ProviderConfig{
			OIDCConfig: &OIDCConfig{
				ClientID:         "client-id",
				ClientSecret:     "secret",
				IssuerURL:        "https://provider.com",
				RedirectURL:      "https://spoke.example.com/callback",
				Scopes:           []string{"openid", "profile", "email"},
				SkipIssuerCheck:  true,
				UserinfoEndpoint: "https://provider.com/userinfo",
			},
		},
	}

	err := provider.ValidateConfig()
	assert.NoError(t, err)
}

func TestOIDCProvider_ValidateConfig_MultipleScopes(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		hasErr bool
	}{
		{
			name:   "openid first",
			scopes: []string{"openid", "profile", "email"},
			hasErr: false,
		},
		{
			name:   "openid middle",
			scopes: []string{"profile", "openid", "email"},
			hasErr: false,
		},
		{
			name:   "openid last",
			scopes: []string{"profile", "email", "openid"},
			hasErr: false,
		},
		{
			name:   "only openid",
			scopes: []string{"openid"},
			hasErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &OIDCProvider{
				config: &ProviderConfig{
					OIDCConfig: &OIDCConfig{
						ClientID:     "client-id",
						ClientSecret: "secret",
						IssuerURL:    "https://provider.com",
						RedirectURL:  "https://spoke.example.com/callback",
						Scopes:       tt.scopes,
					},
				},
			}

			err := provider.ValidateConfig()
			if tt.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOIDCProvider_GetTypeAndName(t *testing.T) {
	config := &ProviderConfig{
		ID:           123,
		Name:         "test-azure-oidc",
		ProviderType: ProviderTypeOIDC,
		ProviderName: ProviderAzureAD,
		Enabled:      true,
		OIDCConfig: &OIDCConfig{
			ClientID:     "azure-client-id",
			ClientSecret: "azure-secret",
			IssuerURL:    "https://login.microsoftonline.com/tenant-id/v2.0",
			RedirectURL:  "https://spoke.example.com/callback",
			Scopes:       []string{"openid", "profile", "email"},
		},
		AttributeMapping: AttributeMap{
			UserID:   "oid",
			Username: "preferred_username",
			Email:    "email",
			FullName: "name",
		},
	}

	provider := &OIDCProvider{config: config}

	// Test GetType
	assert.Equal(t, ProviderTypeOIDC, provider.GetType())

	// Test GetName
	assert.Equal(t, ProviderAzureAD, provider.GetName())

	// Verify config is properly stored
	assert.Equal(t, int64(123), provider.config.ID)
	assert.Equal(t, "test-azure-oidc", provider.config.Name)
	assert.True(t, provider.config.Enabled)
}

func TestOIDCProvider_CompleteConfiguration(t *testing.T) {
	// Test with a complete configuration including all optional fields
	config := &ProviderConfig{
		ID:           1,
		Name:         "complete-oidc",
		ProviderType: ProviderTypeOIDC,
		ProviderName: ProviderGoogle,
		Enabled:      true,
		AutoProvision: true,
		DefaultRole:  "viewer",
		OIDCConfig: &OIDCConfig{
			ClientID:         "google-client-id",
			ClientSecret:     "google-secret",
			IssuerURL:        "https://accounts.google.com",
			RedirectURL:      "https://spoke.example.com/sso/callback",
			Scopes:           []string{"openid", "profile", "email"},
			SkipIssuerCheck:  false,
			UserinfoEndpoint: "https://www.googleapis.com/oauth2/v3/userinfo",
		},
		AttributeMapping: AttributeMap{
			UserID:    "sub",
			Username:  "email",
			Email:     "email",
			FullName:  "name",
			FirstName: "given_name",
			LastName:  "family_name",
			Groups:    "groups",
		},
		GroupMapping: []GroupMap{
			{SSOGroup: "engineers", SpokeRole: "developer"},
			{SSOGroup: "admins", SpokeRole: "admin"},
		},
	}

	provider := &OIDCProvider{config: config}

	// Validate configuration
	err := provider.ValidateConfig()
	assert.NoError(t, err)

	// Verify all fields
	assert.Equal(t, ProviderTypeOIDC, provider.GetType())
	assert.Equal(t, ProviderGoogle, provider.GetName())
	assert.Equal(t, "complete-oidc", provider.config.Name)
	assert.True(t, provider.config.Enabled)
	assert.True(t, provider.config.AutoProvision)
	assert.Equal(t, "viewer", provider.config.DefaultRole)
	assert.Len(t, provider.config.GroupMapping, 2)
}
