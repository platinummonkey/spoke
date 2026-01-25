# SSO (Single Sign-On) Package

This package provides comprehensive SSO integration for Spoke, supporting SAML 2.0, OAuth2, and OpenID Connect (OIDC) authentication protocols.

## Features

- **Multiple Protocol Support**:
  - SAML 2.0 with signature verification
  - OAuth2 with standard flows
  - OpenID Connect with ID token validation

- **Major Identity Provider Support**:
  - Azure AD (Microsoft Entra ID)
  - Okta
  - Google Workspace
  - Generic SAML/OAuth2/OIDC providers

- **Just-In-Time (JIT) User Provisioning**:
  - Automatically create users on first login
  - Update user information on subsequent logins
  - Configurable auto-provisioning per provider

- **Group/Role Mapping**:
  - Map SSO groups to Spoke roles (admin, developer, viewer)
  - Support for multi-group memberships
  - Automatic role assignment based on group membership

- **Session Management**:
  - Secure session creation and validation
  - Session expiration handling
  - SAML logout support

## Architecture

### Components

1. **Provider Interface**: Common interface for all SSO providers
2. **SAML Provider**: Implements SAML 2.0 protocol using `gosaml2`
3. **OAuth2 Provider**: Implements OAuth2 using `golang.org/x/oauth2`
4. **OIDC Provider**: Implements OpenID Connect using `coreos/go-oidc`
5. **Storage**: Database-backed provider configuration storage
6. **Provisioner**: Handles JIT user creation and updates
7. **Handlers**: HTTP handlers for SSO endpoints

### Database Tables

- `sso_providers`: Provider configurations
- `sso_user_mappings`: External user ID to internal user ID mappings
- `sso_sessions`: Active SSO sessions

## Usage

### 1. Configure an SSO Provider

#### Azure AD (OIDC)

```go
config := &sso.ProviderConfig{
    Name:          "azuread",
    ProviderType:  sso.ProviderTypeOIDC,
    ProviderName:  sso.ProviderAzureAD,
    Enabled:       true,
    AutoProvision: true,
    DefaultRole:   "developer",
    OIDCConfig: &sso.OIDCConfig{
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        IssuerURL:    "https://login.microsoftonline.com/tenant-id/v2.0",
        RedirectURL:  "https://spoke.example.com/auth/sso/azuread/callback",
        Scopes:       []string{"openid", "profile", "email"},
    },
    AttributeMapping: sso.AttributeMap{
        UserID:   "oid",
        Username: "preferred_username",
        Email:    "email",
        Groups:   "groups",
    },
    GroupMapping: []sso.GroupMap{
        {SSOGroup: "Spoke-Admins", SpokeRole: "admin"},
        {SSOGroup: "Spoke-Developers", SpokeRole: "developer"},
    },
}

storage := sso.NewStorage(db)
err := storage.CreateProvider(config)
```

#### Okta (OIDC)

```go
config := &sso.ProviderConfig{
    Name:          "okta",
    ProviderType:  sso.ProviderTypeOIDC,
    ProviderName:  sso.ProviderOkta,
    Enabled:       true,
    AutoProvision: true,
    OIDCConfig: &sso.OIDCConfig{
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        IssuerURL:    "https://your-org.okta.com",
        RedirectURL:  "https://spoke.example.com/auth/sso/okta/callback",
        Scopes:       []string{"openid", "profile", "email", "groups"},
    },
    AttributeMapping: sso.AttributeMap{
        UserID:   "sub",
        Username: "preferred_username",
        Email:    "email",
        Groups:   "groups",
    },
}
```

#### Generic SAML Provider

```go
config := &sso.ProviderConfig{
    Name:          "saml-provider",
    ProviderType:  sso.ProviderTypeSAML,
    ProviderName:  sso.ProviderGenericSAML,
    Enabled:       true,
    AutoProvision: true,
    SAMLConfig: &sso.SAMLConfig{
        EntityID:    "https://idp.example.com",
        SSOURL:      "https://idp.example.com/sso",
        SLOUrl:      "https://idp.example.com/slo",
        Certificate: pemEncodedCert,
        SignRequests: true,
    },
    AttributeMapping: sso.AttributeMap{
        UserID:   "uid",
        Username: "uid",
        Email:    "email",
        FullName: "displayName",
        Groups:   "groups",
    },
}
```

### 2. Register HTTP Handlers

```go
import (
    "github.com/gorilla/mux"
    "github.com/platinummonkey/spoke/pkg/sso"
)

router := mux.NewRouter()
handlers := sso.NewHandlers(db, "https://spoke.example.com")
handlers.RegisterRoutes(router)
```

### 3. API Endpoints

#### Provider Management

- `GET /sso/providers` - List all providers
- `POST /sso/providers` - Create a new provider
- `GET /sso/providers/{name}` - Get provider details
- `PUT /sso/providers/{name}` - Update provider
- `DELETE /sso/providers/{name}` - Delete provider

#### Authentication

- `GET /auth/sso/{provider}/login` - Initiate SSO login
- `GET|POST /auth/sso/{provider}/callback` - Handle SSO callback
- `GET|POST /auth/sso/logout` - SSO logout

#### Metadata (SAML only)

- `GET /sso/metadata/{provider}` - Get SAML SP metadata

### 4. SSO Login Flow

1. User navigates to `/auth/sso/{provider}/login?return_url=/dashboard`
2. System generates state token and redirects to IdP
3. User authenticates at IdP
4. IdP redirects back to `/auth/sso/{provider}/callback`
5. System validates response and provisions user if needed
6. System creates session and redirects to return URL

### 5. Using Preset Configurations

```go
// Get preset config for Azure AD
preset, _ := sso.GetPresetConfig(sso.ProviderAzureAD)
preset.Name = "my-azuread"
preset.OIDCConfig.ClientID = "your-client-id"
preset.OIDCConfig.ClientSecret = "your-client-secret"
preset.OIDCConfig.IssuerURL = "https://login.microsoftonline.com/tenant-id/v2.0"
preset.OIDCConfig.RedirectURL = "https://spoke.example.com/callback"

storage.CreateProvider(preset)
```

## Group/Role Mapping

Map SSO groups to Spoke roles:

```go
GroupMapping: []sso.GroupMap{
    {SSOGroup: "engineering-admins", SpokeRole: "admin"},
    {SSOGroup: "engineering-team", SpokeRole: "developer"},
    {SSOGroup: "product-team", SpokeRole: "viewer"},
}
```

Role precedence (highest to lowest):
1. `admin` - Full access
2. `developer` - Can push and manage schemas
3. `viewer` - Read-only access

## Attribute Mapping

Configure how SSO attributes map to user fields:

```go
AttributeMapping: sso.AttributeMap{
    UserID:    "oid",           // Unique user identifier
    Username:  "preferred_username",
    Email:     "email",
    FullName:  "name",
    FirstName: "given_name",
    LastName:  "family_name",
    Groups:    "groups",        // Attribute containing group memberships
}
```

## JIT User Provisioning

When `AutoProvision` is enabled:

1. User authenticates via SSO
2. System checks if user mapping exists
3. If not, creates new Spoke user with SSO attributes
4. Maps SSO groups to Spoke roles
5. Creates user-to-SSO mapping
6. On subsequent logins, updates user information

## Security Considerations

- Secrets (client_secret, private_key) are never exposed in API responses
- SAML assertions are cryptographically verified
- OIDC ID tokens are validated using provider's public keys
- State tokens prevent CSRF attacks
- Sessions have configurable expiration
- All SSO traffic should use HTTPS

## Testing

Run SSO tests:

```bash
go test ./pkg/sso/...
```

## Example: Complete Azure AD Setup

```go
// 1. Create provider configuration
config := &sso.ProviderConfig{
    Name:          "azuread",
    ProviderType:  sso.ProviderTypeOIDC,
    ProviderName:  sso.ProviderAzureAD,
    Enabled:       true,
    AutoProvision: true,
    DefaultRole:   "viewer",
    OIDCConfig: &sso.OIDCConfig{
        ClientID:     "app-client-id",
        ClientSecret: "app-client-secret",
        IssuerURL:    "https://login.microsoftonline.com/tenant-id/v2.0",
        RedirectURL:  "https://spoke.example.com/auth/sso/azuread/callback",
        Scopes:       []string{"openid", "profile", "email"},
    },
    AttributeMapping: sso.AttributeMap{
        UserID:   "oid",
        Username: "preferred_username",
        Email:    "email",
        FullName: "name",
        Groups:   "groups",
    },
    GroupMapping: []sso.GroupMap{
        {SSOGroup: "Spoke-Admins", SpokeRole: "admin"},
        {SSOGroup: "Spoke-Users", SpokeRole: "developer"},
    },
}

// 2. Save to database
storage := sso.NewStorage(db)
err := storage.CreateProvider(config)

// 3. Users can now login at:
// https://spoke.example.com/auth/sso/azuread/login
```

## Troubleshooting

### SAML Issues

- Verify IdP certificate is valid PEM format
- Check EntityID matches IdP configuration
- Ensure ACS URL is registered with IdP
- Review SAML assertions for attribute names

### OIDC Issues

- Verify issuer URL is correct (with version, e.g., /v2.0)
- Check client ID and secret
- Ensure redirect URL is registered
- Verify required scopes are requested

### OAuth2 Issues

- Check auth and token URLs
- Verify client credentials
- Ensure user_info_url is correct
- Review OAuth2 response for attribute names

## Dependencies

- `github.com/russellhaering/gosaml2` - SAML 2.0 implementation
- `github.com/russellhaering/goxmldsig` - XML digital signatures
- `github.com/coreos/go-oidc/v3` - OpenID Connect
- `golang.org/x/oauth2` - OAuth2 client

## Future Enhancements

- [ ] Multi-factor authentication enforcement
- [ ] Provider-specific logout flows
- [ ] SAML attribute encryption
- [ ] Just-in-time organization provisioning
- [ ] Custom attribute transformations
- [ ] Provider health checks
- [ ] SSO analytics and audit logs
