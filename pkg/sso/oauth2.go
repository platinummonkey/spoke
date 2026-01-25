package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// OAuth2Provider implements OAuth2 SSO
type OAuth2Provider struct {
	config       *ProviderConfig
	oauth2Config *oauth2.Config
}

// NewOAuth2Provider creates a new OAuth2 provider
func NewOAuth2Provider(config *ProviderConfig) (*OAuth2Provider, error) {
	if config.OAuth2Config == nil {
		return nil, fmt.Errorf("OAuth2 config is required")
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     config.OAuth2Config.ClientID,
		ClientSecret: config.OAuth2Config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.OAuth2Config.AuthURL,
			TokenURL: config.OAuth2Config.TokenURL,
		},
		RedirectURL: config.OAuth2Config.RedirectURL,
		Scopes:      config.OAuth2Config.Scopes,
	}

	return &OAuth2Provider{
		config:       config,
		oauth2Config: oauth2Cfg,
	}, nil
}

// GetType returns the provider type
func (p *OAuth2Provider) GetType() ProviderType {
	return ProviderTypeOAuth2
}

// GetName returns the provider name
func (p *OAuth2Provider) GetName() ProviderName {
	return p.config.ProviderName
}

// InitiateLogin redirects to OAuth2 authorization endpoint
func (p *OAuth2Provider) InitiateLogin(w http.ResponseWriter, r *http.Request, state string) error {
	authURL := p.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusFound)
	return nil
}

// HandleCallback processes OAuth2 callback
func (p *OAuth2Provider) HandleCallback(w http.ResponseWriter, r *http.Request) (*SSOUser, error) {
	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}

	// Exchange code for token
	ctx := context.Background()
	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	// Fetch user info
	client := p.oauth2Config.Client(ctx, token)
	userInfoURL := p.config.OAuth2Config.UserInfoURL
	if userInfoURL == "" {
		return nil, fmt.Errorf("user_info_url is required")
	}

	resp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse user info
	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Map to SSOUser
	ssoUser := &SSOUser{
		ProviderID:   p.config.ID,
		ProviderName: p.config.Name,
		Attributes:   make(map[string]string),
	}

	// Convert all attributes to strings
	for k, v := range userInfo {
		if str, ok := v.(string); ok {
			ssoUser.Attributes[k] = str
		} else {
			// Convert to JSON string for complex types
			jsonBytes, _ := json.Marshal(v)
			ssoUser.Attributes[k] = string(jsonBytes)
		}
	}

	// Extract mapped fields
	ssoUser.ExternalID = getStringValue(userInfo, p.config.AttributeMapping.UserID)
	ssoUser.Username = getStringValue(userInfo, p.config.AttributeMapping.Username)
	ssoUser.Email = getStringValue(userInfo, p.config.AttributeMapping.Email)
	ssoUser.FullName = getStringValue(userInfo, p.config.AttributeMapping.FullName)
	ssoUser.FirstName = getStringValue(userInfo, p.config.AttributeMapping.FirstName)
	ssoUser.LastName = getStringValue(userInfo, p.config.AttributeMapping.LastName)

	// Extract groups
	if p.config.AttributeMapping.Groups != "" {
		groups := getArrayValue(userInfo, p.config.AttributeMapping.Groups)
		ssoUser.Groups = groups
	}

	// Use email as fallback for username
	if ssoUser.Username == "" && ssoUser.Email != "" {
		ssoUser.Username = ssoUser.Email
	}

	// Validate required fields
	if ssoUser.ExternalID == "" {
		return nil, fmt.Errorf("missing user ID in OAuth2 response")
	}
	if ssoUser.Email == "" {
		return nil, fmt.Errorf("missing email in OAuth2 response")
	}

	return ssoUser, nil
}

// Logout handles OAuth2 logout (most OAuth2 providers don't support logout)
func (p *OAuth2Provider) Logout(w http.ResponseWriter, r *http.Request, sessionIndex string) error {
	// OAuth2 doesn't have a standard logout flow
	// Just clear local session
	return nil
}

// ValidateConfig validates the OAuth2 configuration
func (p *OAuth2Provider) ValidateConfig() error {
	if p.config.OAuth2Config == nil {
		return fmt.Errorf("OAuth2 config is required")
	}

	cfg := p.config.OAuth2Config

	if cfg.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if cfg.ClientSecret == "" {
		return fmt.Errorf("client_secret is required")
	}
	if cfg.AuthURL == "" {
		return fmt.Errorf("auth_url is required")
	}
	if cfg.TokenURL == "" {
		return fmt.Errorf("token_url is required")
	}
	if cfg.RedirectURL == "" {
		return fmt.Errorf("redirect_url is required")
	}
	if len(cfg.Scopes) == 0 {
		return fmt.Errorf("scopes are required")
	}

	return nil
}

// Helper functions

func getStringValue(data map[string]interface{}, key string) string {
	if key == "" {
		return ""
	}
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getArrayValue(data map[string]interface{}, key string) []string {
	if key == "" {
		return nil
	}
	if val, ok := data[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return nil
}
