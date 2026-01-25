package sso

import "time"

// ProviderType represents the SSO provider type
type ProviderType string

const (
	ProviderTypeSAML   ProviderType = "saml"
	ProviderTypeOAuth2 ProviderType = "oauth2"
	ProviderTypeOIDC   ProviderType = "oidc"
)

// ProviderName represents the SSO provider name
type ProviderName string

const (
	ProviderAzureAD       ProviderName = "azuread"
	ProviderOkta          ProviderName = "okta"
	ProviderGoogle        ProviderName = "google"
	ProviderGenericSAML   ProviderName = "generic_saml"
	ProviderGenericOAuth2 ProviderName = "generic_oauth2"
	ProviderGenericOIDC   ProviderName = "generic_oidc"
)

// ProviderConfig represents SSO provider configuration
type ProviderConfig struct {
	ID                   int64        `json:"id"`
	Name                 string       `json:"name"` // Unique name for this provider instance
	ProviderType         ProviderType `json:"provider_type"`
	ProviderName         ProviderName `json:"provider_name"`
	Enabled              bool         `json:"enabled"`
	AutoProvision        bool         `json:"auto_provision"` // JIT user provisioning
	DefaultRole          string       `json:"default_role"`   // Default role for new users
	GroupMapping         []GroupMap   `json:"group_mapping,omitempty"`
	SAMLConfig           *SAMLConfig  `json:"saml_config,omitempty"`
	OAuth2Config         *OAuth2Config `json:"oauth2_config,omitempty"`
	OIDCConfig           *OIDCConfig  `json:"oidc_config,omitempty"`
	AttributeMapping     AttributeMap `json:"attribute_mapping"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// SAMLConfig holds SAML 2.0 configuration
type SAMLConfig struct {
	EntityID              string   `json:"entity_id"`
	SSOURL                string   `json:"sso_url"`
	SLOUrl                string   `json:"slo_url,omitempty"` // Single Logout URL
	Certificate           string   `json:"certificate"` // PEM encoded certificate
	PrivateKey            string   `json:"-"` // Never expose private key in JSON
	MetadataURL           string   `json:"metadata_url,omitempty"`
	SignRequests          bool     `json:"sign_requests"`
	ForceAuthn            bool     `json:"force_authn"`
	AllowIDPInitiated     bool     `json:"allow_idp_initiated"`
	NameIDFormat          string   `json:"name_id_format,omitempty"`
	DefaultRedirectURL    string   `json:"default_redirect_url"`
	AudienceRestriction   []string `json:"audience_restriction,omitempty"`
}

// OAuth2Config holds OAuth2 configuration
type OAuth2Config struct {
	ClientID             string   `json:"client_id"`
	ClientSecret         string   `json:"-"` // Never expose secret in JSON
	AuthURL              string   `json:"auth_url"`
	TokenURL             string   `json:"token_url"`
	UserInfoURL          string   `json:"user_info_url,omitempty"`
	Scopes               []string `json:"scopes"`
	RedirectURL          string   `json:"redirect_url"`
	UserinfoEndpoint     string   `json:"userinfo_endpoint,omitempty"`
}

// OIDCConfig holds OpenID Connect configuration
type OIDCConfig struct {
	ClientID             string   `json:"client_id"`
	ClientSecret         string   `json:"-"` // Never expose secret in JSON
	IssuerURL            string   `json:"issuer_url"` // Discovery endpoint
	RedirectURL          string   `json:"redirect_url"`
	Scopes               []string `json:"scopes"`
	SkipIssuerCheck      bool     `json:"skip_issuer_check,omitempty"`
	UserinfoEndpoint     string   `json:"userinfo_endpoint,omitempty"`
}

// AttributeMap defines how SSO attributes map to user fields
type AttributeMap struct {
	UserID    string `json:"user_id"`    // Unique user identifier
	Username  string `json:"username"`
	Email     string `json:"email"`
	FullName  string `json:"full_name,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Groups    string `json:"groups,omitempty"` // Attribute containing group memberships
}

// GroupMap maps SSO groups to Spoke roles
type GroupMap struct {
	SSOGroup  string `json:"sso_group"`  // Group name from SSO provider
	SpokeRole string `json:"spoke_role"` // Spoke role (admin, developer, viewer)
}

// SSOUser represents user information from SSO provider
type SSOUser struct {
	ExternalID  string            `json:"external_id"` // Unique ID from provider
	Username    string            `json:"username"`
	Email       string            `json:"email"`
	FullName    string            `json:"full_name,omitempty"`
	FirstName   string            `json:"first_name,omitempty"`
	LastName    string            `json:"last_name,omitempty"`
	Groups      []string          `json:"groups,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"` // Raw attributes
	ProviderID  int64             `json:"provider_id"`
	ProviderName string           `json:"provider_name"`
}

// SSOUserMapping represents mapping between SSO user and Spoke user
type SSOUserMapping struct {
	ID              int64     `json:"id"`
	ProviderID      int64     `json:"provider_id"`
	ExternalUserID  string    `json:"external_user_id"` // User ID from SSO provider
	InternalUserID  int64     `json:"internal_user_id"` // Spoke user ID
	LastLoginAt     time.Time `json:"last_login_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SSOSession represents an SSO session
type SSOSession struct {
	ID              string    `json:"id"`
	ProviderID      int64     `json:"provider_id"`
	UserID          int64     `json:"user_id"`
	ExternalUserID  string    `json:"external_user_id"`
	SAMLSessionIndex string   `json:"saml_session_index,omitempty"` // For SAML logout
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}
