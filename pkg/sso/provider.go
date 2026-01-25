package sso

import (
	"context"
	"fmt"
	"net/http"
)

// Provider defines the interface for SSO providers
type Provider interface {
	// GetType returns the provider type (SAML, OAuth2, OIDC)
	GetType() ProviderType

	// GetName returns the provider name
	GetName() ProviderName

	// InitiateLogin generates the URL/redirect for SSO login
	InitiateLogin(w http.ResponseWriter, r *http.Request, state string) error

	// HandleCallback processes the SSO callback and returns user information
	HandleCallback(w http.ResponseWriter, r *http.Request) (*SSOUser, error)

	// Logout handles SSO logout (if supported)
	Logout(w http.ResponseWriter, r *http.Request, sessionIndex string) error

	// ValidateConfig validates the provider configuration
	ValidateConfig() error
}

// ProviderFactory creates SSO providers based on configuration
type ProviderFactory struct {
	baseURL string
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(baseURL string) *ProviderFactory {
	return &ProviderFactory{
		baseURL: baseURL,
	}
}

// CreateProvider creates a provider instance from configuration
func (f *ProviderFactory) CreateProvider(config *ProviderConfig) (Provider, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("provider %s is disabled", config.Name)
	}

	switch config.ProviderType {
	case ProviderTypeSAML:
		if config.SAMLConfig == nil {
			return nil, fmt.Errorf("SAML config is required for SAML provider")
		}
		return NewSAMLProvider(config, f.baseURL)

	case ProviderTypeOAuth2:
		if config.OAuth2Config == nil {
			return nil, fmt.Errorf("OAuth2 config is required for OAuth2 provider")
		}
		return NewOAuth2Provider(config)

	case ProviderTypeOIDC:
		if config.OIDCConfig == nil {
			return nil, fmt.Errorf("OIDC config is required for OIDC provider")
		}
		return NewOIDCProvider(context.Background(), config)

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.ProviderType)
	}
}

// GetPresetConfig returns preset configuration for well-known providers
func GetPresetConfig(providerName ProviderName) (*ProviderConfig, error) {
	switch providerName {
	case ProviderAzureAD:
		return &ProviderConfig{
			ProviderType: ProviderTypeOIDC,
			ProviderName: ProviderAzureAD,
			AttributeMapping: AttributeMap{
				UserID:    "oid",
				Username:  "preferred_username",
				Email:     "email",
				FullName:  "name",
				FirstName: "given_name",
				LastName:  "family_name",
				Groups:    "groups",
			},
			OIDCConfig: &OIDCConfig{
				Scopes: []string{"openid", "profile", "email"},
			},
		}, nil

	case ProviderOkta:
		return &ProviderConfig{
			ProviderType: ProviderTypeOIDC,
			ProviderName: ProviderOkta,
			AttributeMapping: AttributeMap{
				UserID:    "sub",
				Username:  "preferred_username",
				Email:     "email",
				FullName:  "name",
				FirstName: "given_name",
				LastName:  "family_name",
				Groups:    "groups",
			},
			OIDCConfig: &OIDCConfig{
				Scopes: []string{"openid", "profile", "email", "groups"},
			},
		}, nil

	case ProviderGoogle:
		return &ProviderConfig{
			ProviderType: ProviderTypeOIDC,
			ProviderName: ProviderGoogle,
			AttributeMapping: AttributeMap{
				UserID:    "sub",
				Username:  "email",
				Email:     "email",
				FullName:  "name",
				FirstName: "given_name",
				LastName:  "family_name",
			},
			OIDCConfig: &OIDCConfig{
				IssuerURL: "https://accounts.google.com",
				Scopes:    []string{"openid", "profile", "email"},
			},
		}, nil

	default:
		return nil, fmt.Errorf("no preset configuration for provider: %s", providerName)
	}
}
