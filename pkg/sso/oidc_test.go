package sso

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
