package sso

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuth2Provider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *OAuth2Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &OAuth2Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				AuthURL:      "https://provider.com/oauth/authorize",
				TokenURL:     "https://provider.com/oauth/token",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"openid", "profile"},
			},
			expectError: false,
		},
		{
			name: "missing client_id",
			config: &OAuth2Config{
				ClientSecret: "test-secret",
				AuthURL:      "https://provider.com/oauth/authorize",
				TokenURL:     "https://provider.com/oauth/token",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"openid"},
			},
			expectError: true,
			errorMsg:    "client_id is required",
		},
		{
			name: "missing client_secret",
			config: &OAuth2Config{
				ClientID:    "test-client-id",
				AuthURL:     "https://provider.com/oauth/authorize",
				TokenURL:    "https://provider.com/oauth/token",
				RedirectURL: "https://spoke.example.com/callback",
				Scopes:      []string{"openid"},
			},
			expectError: true,
			errorMsg:    "client_secret is required",
		},
		{
			name: "missing auth_url",
			config: &OAuth2Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				TokenURL:     "https://provider.com/oauth/token",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"openid"},
			},
			expectError: true,
			errorMsg:    "auth_url is required",
		},
		{
			name: "missing token_url",
			config: &OAuth2Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				AuthURL:      "https://provider.com/oauth/authorize",
				RedirectURL:  "https://spoke.example.com/callback",
				Scopes:       []string{"openid"},
			},
			expectError: true,
			errorMsg:    "token_url is required",
		},
		{
			name: "missing redirect_url",
			config: &OAuth2Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				AuthURL:      "https://provider.com/oauth/authorize",
				TokenURL:     "https://provider.com/oauth/token",
				Scopes:       []string{"openid"},
			},
			expectError: true,
			errorMsg:    "redirect_url is required",
		},
		{
			name: "missing scopes",
			config: &OAuth2Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				AuthURL:      "https://provider.com/oauth/authorize",
				TokenURL:     "https://provider.com/oauth/token",
				RedirectURL:  "https://spoke.example.com/callback",
			},
			expectError: true,
			errorMsg:    "scopes are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerConfig := &ProviderConfig{
				Name:          "test-oauth2",
				ProviderType:  ProviderTypeOAuth2,
				ProviderName:  ProviderGenericOAuth2,
				Enabled:       true,
				OAuth2Config:  tt.config,
				AttributeMapping: AttributeMap{
					UserID:   "sub",
					Username: "username",
					Email:    "email",
				},
			}

			provider, err := NewOAuth2Provider(providerConfig)

			if tt.expectError {
				require.NoError(t, err) // Provider creation should succeed
				err = provider.ValidateConfig()
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				err = provider.ValidateConfig()
				assert.NoError(t, err)
			}
		})
	}
}

func TestOAuth2Provider_GetType(t *testing.T) {
	config := &ProviderConfig{
		Name:         "test-oauth2",
		ProviderType: ProviderTypeOAuth2,
		ProviderName: ProviderGenericOAuth2,
		OAuth2Config: &OAuth2Config{
			ClientID:     "test",
			ClientSecret: "test",
			AuthURL:      "https://example.com/auth",
			TokenURL:     "https://example.com/token",
			RedirectURL:  "https://spoke.example.com/callback",
			Scopes:       []string{"openid"},
		},
		AttributeMapping: AttributeMap{
			UserID: "sub",
			Email:  "email",
		},
	}

	provider, err := NewOAuth2Provider(config)
	require.NoError(t, err)

	assert.Equal(t, ProviderTypeOAuth2, provider.GetType())
}

func TestOAuth2Provider_GetName(t *testing.T) {
	config := &ProviderConfig{
		Name:         "test-oauth2",
		ProviderType: ProviderTypeOAuth2,
		ProviderName: ProviderGenericOAuth2,
		OAuth2Config: &OAuth2Config{
			ClientID:     "test",
			ClientSecret: "test",
			AuthURL:      "https://example.com/auth",
			TokenURL:     "https://example.com/token",
			RedirectURL:  "https://spoke.example.com/callback",
			Scopes:       []string{"openid"},
		},
		AttributeMapping: AttributeMap{
			UserID: "sub",
			Email:  "email",
		},
	}

	provider, err := NewOAuth2Provider(config)
	require.NoError(t, err)

	assert.Equal(t, ProviderGenericOAuth2, provider.GetName())
}

func TestGetStringValue(t *testing.T) {
	data := map[string]interface{}{
		"string_field": "value",
		"number_field": 123,
		"bool_field":   true,
		"nil_field":    nil,
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"existing string", "string_field", "value"},
		{"number field", "number_field", ""},
		{"bool field", "bool_field", ""},
		{"nil field", "nil_field", ""},
		{"missing field", "missing", ""},
		{"empty key", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringValue(data, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetArrayValue(t *testing.T) {
	data := map[string]interface{}{
		"string_array": []interface{}{"value1", "value2", "value3"},
		"mixed_array":  []interface{}{"string", 123, true},
		"empty_array":  []interface{}{},
		"string_field": "not an array",
	}

	tests := []struct {
		name     string
		key      string
		expected []string
	}{
		{"string array", "string_array", []string{"value1", "value2", "value3"}},
		{"mixed array", "mixed_array", []string{"string"}},
		{"empty array", "empty_array", []string{}},
		{"not an array", "string_field", nil},
		{"missing field", "missing", nil},
		{"empty key", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getArrayValue(data, tt.key)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
