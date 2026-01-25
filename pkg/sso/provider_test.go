package sso

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderFactory(t *testing.T) {
	factory := NewProviderFactory("https://spoke.example.com")
	assert.NotNil(t, factory)
	assert.Equal(t, "https://spoke.example.com", factory.baseURL)
}

func TestGetPresetConfig_AzureAD(t *testing.T) {
	config, err := GetPresetConfig(ProviderAzureAD)
	require.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, ProviderTypeOIDC, config.ProviderType)
	assert.Equal(t, ProviderAzureAD, config.ProviderName)
	assert.NotNil(t, config.OIDCConfig)
	assert.Contains(t, config.OIDCConfig.Scopes, "openid")
	assert.Equal(t, "oid", config.AttributeMapping.UserID)
	assert.Equal(t, "email", config.AttributeMapping.Email)
	assert.Equal(t, "groups", config.AttributeMapping.Groups)
}

func TestGetPresetConfig_Okta(t *testing.T) {
	config, err := GetPresetConfig(ProviderOkta)
	require.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, ProviderTypeOIDC, config.ProviderType)
	assert.Equal(t, ProviderOkta, config.ProviderName)
	assert.NotNil(t, config.OIDCConfig)
	assert.Contains(t, config.OIDCConfig.Scopes, "openid")
	assert.Contains(t, config.OIDCConfig.Scopes, "groups")
	assert.Equal(t, "sub", config.AttributeMapping.UserID)
}

func TestGetPresetConfig_Google(t *testing.T) {
	config, err := GetPresetConfig(ProviderGoogle)
	require.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, ProviderTypeOIDC, config.ProviderType)
	assert.Equal(t, ProviderGoogle, config.ProviderName)
	assert.NotNil(t, config.OIDCConfig)
	assert.Equal(t, "https://accounts.google.com", config.OIDCConfig.IssuerURL)
	assert.Contains(t, config.OIDCConfig.Scopes, "openid")
	assert.Contains(t, config.OIDCConfig.Scopes, "email")
}

func TestGetPresetConfig_Invalid(t *testing.T) {
	config, err := GetPresetConfig(ProviderName("invalid"))
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "no preset configuration")
}

func TestCreateProvider_Disabled(t *testing.T) {
	factory := NewProviderFactory("https://spoke.example.com")

	config := &ProviderConfig{
		Name:         "test",
		Enabled:      false,
		ProviderType: ProviderTypeOIDC,
	}

	provider, err := factory.CreateProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "disabled")
}

func TestCreateProvider_MissingConfig(t *testing.T) {
	factory := NewProviderFactory("https://spoke.example.com")

	tests := []struct {
		name         string
		config       *ProviderConfig
		expectedErr  string
	}{
		{
			name: "SAML without config",
			config: &ProviderConfig{
				Name:         "test-saml",
				Enabled:      true,
				ProviderType: ProviderTypeSAML,
			},
			expectedErr: "SAML config is required",
		},
		{
			name: "OAuth2 without config",
			config: &ProviderConfig{
				Name:         "test-oauth2",
				Enabled:      true,
				ProviderType: ProviderTypeOAuth2,
			},
			expectedErr: "OAuth2 config is required",
		},
		{
			name: "OIDC without config",
			config: &ProviderConfig{
				Name:         "test-oidc",
				Enabled:      true,
				ProviderType: ProviderTypeOIDC,
			},
			expectedErr: "OIDC config is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.config)
			assert.Error(t, err)
			assert.Nil(t, provider)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestCreateProvider_UnsupportedType(t *testing.T) {
	factory := NewProviderFactory("https://spoke.example.com")

	config := &ProviderConfig{
		Name:         "test",
		Enabled:      true,
		ProviderType: ProviderType("unsupported"),
	}

	provider, err := factory.CreateProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "unsupported provider type")
}
