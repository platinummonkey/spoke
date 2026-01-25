package sso

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCProvider implements OpenID Connect SSO
type OIDCProvider struct {
	config       *ProviderConfig
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config *oauth2.Config
}

// NewOIDCProvider creates a new OIDC provider
func NewOIDCProvider(ctx context.Context, config *ProviderConfig) (*OIDCProvider, error) {
	if config.OIDCConfig == nil {
		return nil, fmt.Errorf("OIDC config is required")
	}

	// Discover OIDC provider
	provider, err := oidc.NewProvider(ctx, config.OIDCConfig.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	// Create ID token verifier
	verifierConfig := &oidc.Config{
		ClientID:          config.OIDCConfig.ClientID,
		SkipIssuerCheck:   config.OIDCConfig.SkipIssuerCheck,
	}
	verifier := provider.Verifier(verifierConfig)

	// Create OAuth2 config
	oauth2Config := &oauth2.Config{
		ClientID:     config.OIDCConfig.ClientID,
		ClientSecret: config.OIDCConfig.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  config.OIDCConfig.RedirectURL,
		Scopes:       config.OIDCConfig.Scopes,
	}

	return &OIDCProvider{
		config:       config,
		provider:     provider,
		verifier:     verifier,
		oauth2Config: oauth2Config,
	}, nil
}

// GetType returns the provider type
func (p *OIDCProvider) GetType() ProviderType {
	return ProviderTypeOIDC
}

// GetName returns the provider name
func (p *OIDCProvider) GetName() ProviderName {
	return p.config.ProviderName
}

// InitiateLogin redirects to OIDC authorization endpoint
func (p *OIDCProvider) InitiateLogin(w http.ResponseWriter, r *http.Request, state string) error {
	authURL := p.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusFound)
	return nil
}

// HandleCallback processes OIDC callback
func (p *OIDCProvider) HandleCallback(w http.ResponseWriter, r *http.Request) (*SSOUser, error) {
	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}

	// Exchange code for token
	ctx := context.Background()
	oauth2Token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("missing id_token in response")
	}

	// Verify ID token
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Map to SSOUser
	ssoUser := &SSOUser{
		ProviderID:   p.config.ID,
		ProviderName: p.config.Name,
		Attributes:   make(map[string]string),
	}

	// Convert all claims to strings
	for k, v := range claims {
		if str, ok := v.(string); ok {
			ssoUser.Attributes[k] = str
		}
	}

	// Extract mapped fields
	ssoUser.ExternalID = getStringValue(claims, p.config.AttributeMapping.UserID)
	ssoUser.Username = getStringValue(claims, p.config.AttributeMapping.Username)
	ssoUser.Email = getStringValue(claims, p.config.AttributeMapping.Email)
	ssoUser.FullName = getStringValue(claims, p.config.AttributeMapping.FullName)
	ssoUser.FirstName = getStringValue(claims, p.config.AttributeMapping.FirstName)
	ssoUser.LastName = getStringValue(claims, p.config.AttributeMapping.LastName)

	// Extract groups
	if p.config.AttributeMapping.Groups != "" {
		groups := getArrayValue(claims, p.config.AttributeMapping.Groups)
		ssoUser.Groups = groups
	}

	// Fetch additional user info if endpoint is configured
	if p.config.OIDCConfig.UserinfoEndpoint != "" {
		userInfo, err := p.fetchUserInfo(ctx, oauth2Token)
		if err == nil {
			// Merge additional attributes
			for k, v := range userInfo {
				if str, ok := v.(string); ok {
					if _, exists := ssoUser.Attributes[k]; !exists {
						ssoUser.Attributes[k] = str
					}
				}
			}

			// Override with userinfo if available
			if email := getStringValue(userInfo, "email"); email != "" {
				ssoUser.Email = email
			}
			if groups := getArrayValue(userInfo, p.config.AttributeMapping.Groups); len(groups) > 0 {
				ssoUser.Groups = groups
			}
		}
	}

	// Use email as fallback for username
	if ssoUser.Username == "" && ssoUser.Email != "" {
		ssoUser.Username = ssoUser.Email
	}

	// Use subject claim as fallback for user ID
	if ssoUser.ExternalID == "" {
		ssoUser.ExternalID = idToken.Subject
	}

	// Validate required fields
	if ssoUser.ExternalID == "" {
		return nil, fmt.Errorf("missing user ID in OIDC token")
	}
	if ssoUser.Email == "" {
		return nil, fmt.Errorf("missing email in OIDC token")
	}

	return ssoUser, nil
}

// fetchUserInfo fetches additional user information from userinfo endpoint
func (p *OIDCProvider) fetchUserInfo(ctx context.Context, token *oauth2.Token) (map[string]interface{}, error) {
	userInfo, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return nil, err
	}

	var claims map[string]interface{}
	if err := userInfo.Claims(&claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// Logout handles OIDC logout
func (p *OIDCProvider) Logout(w http.ResponseWriter, r *http.Request, sessionIndex string) error {
	// OIDC has optional logout support via end_session_endpoint
	// For now, just clear local session
	// TODO: Implement RP-initiated logout if provider supports it
	return nil
}

// ValidateConfig validates the OIDC configuration
func (p *OIDCProvider) ValidateConfig() error {
	if p.config.OIDCConfig == nil {
		return fmt.Errorf("OIDC config is required")
	}

	cfg := p.config.OIDCConfig

	if cfg.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if cfg.ClientSecret == "" {
		return fmt.Errorf("client_secret is required")
	}
	if cfg.IssuerURL == "" {
		return fmt.Errorf("issuer_url is required")
	}
	if cfg.RedirectURL == "" {
		return fmt.Errorf("redirect_url is required")
	}
	if len(cfg.Scopes) == 0 {
		return fmt.Errorf("scopes are required")
	}

	// Verify "openid" scope is present
	hasOpenID := false
	for _, scope := range cfg.Scopes {
		if scope == "openid" {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		return fmt.Errorf("'openid' scope is required for OIDC")
	}

	return nil
}
