---
title: "SSO Integration"
weight: 9
---

# Single Sign-On (SSO) Integration

Spoke supports enterprise authentication via SAML 2.0, OAuth2, and OpenID Connect (OIDC).

## Overview

Spoke's SSO integration supports:

- **SAML 2.0**: Enterprise identity providers
- **OAuth2**: Standard OAuth2 authorization flow
- **OpenID Connect (OIDC)**: Modern authentication protocol
- **Just-In-Time (JIT) Provisioning**: Automatic user creation
- **Group/Role Mapping**: Map SSO groups to Spoke roles

## Supported Providers

- Azure AD (Microsoft Entra ID)
- Okta
- Google Workspace
- Generic SAML/OAuth2/OIDC providers

## Configuration

### Enable SSO

```yaml
# config.yaml
sso:
  enabled: true
  default_provider: azure
  jit_provisioning: true

  providers:
    - id: azure
      type: oidc
      name: Azure AD
      client_id: your-client-id
      client_secret: your-client-secret
      issuer_url: https://login.microsoftonline.com/{tenant}/v2.0
      redirect_url: https://spoke.company.com/auth/callback/azure
      scopes:
        - openid
        - profile
        - email
      attribute_mapping:
        user_id: sub
        email: email
        first_name: given_name
        last_name: family_name
        groups: groups
      role_mapping:
        - sso_group: "Spoke-Admins"
          spoke_role: "org:admin"
        - sso_group: "Spoke-Developers"
          spoke_role: "org:developer"
```

## Azure AD Setup

### 1. Register Application in Azure

1. Go to Azure Portal → Azure Active Directory → App registrations
2. Click "New registration"
3. Set redirect URI: `https://spoke.company.com/auth/callback/azure`
4. Note the Application (client) ID

### 2. Configure Authentication

1. Add redirect URI for callback
2. Enable ID tokens
3. Configure optional claims (groups, email)

### 3. Create Client Secret

1. Go to Certificates & secrets
2. Create new client secret
3. Copy the secret value

### 4. Configure Spoke

```yaml
sso:
  providers:
    - id: azure
      type: oidc
      name: Azure AD
      client_id: "abc123-def456-ghi789"
      client_secret: "your-secret-here"
      issuer_url: "https://login.microsoftonline.com/{tenant-id}/v2.0"
      redirect_url: "https://spoke.company.com/auth/callback/azure"
      scopes:
        - openid
        - profile
        - email
        - offline_access
      attribute_mapping:
        user_id: sub
        email: email
        first_name: given_name
        last_name: family_name
        groups: groups
      role_mapping:
        - sso_group: "Spoke Admins"
          spoke_role: "org:admin"
        - sso_group: "Engineering"
          spoke_role: "org:developer"
```

## Okta Setup

### 1. Create Application in Okta

1. Go to Okta Admin → Applications → Create App Integration
2. Select "OIDC - OpenID Connect"
3. Select "Web Application"
4. Set redirect URI: `https://spoke.company.com/auth/callback/okta`

### 2. Configure Spoke

```yaml
sso:
  providers:
    - id: okta
      type: oidc
      name: Okta
      client_id: "your-okta-client-id"
      client_secret: "your-okta-client-secret"
      issuer_url: "https://your-domain.okta.com"
      redirect_url: "https://spoke.company.com/auth/callback/okta"
      scopes:
        - openid
        - profile
        - email
        - groups
      attribute_mapping:
        user_id: sub
        email: email
        first_name: given_name
        last_name: family_name
        groups: groups
      role_mapping:
        - sso_group: "Spoke-Admins"
          spoke_role: "org:admin"
```

## Google Workspace Setup

### 1. Create OAuth2 Credentials

1. Go to Google Cloud Console
2. Create OAuth2 credentials
3. Add authorized redirect URI: `https://spoke.company.com/auth/callback/google`

### 2. Configure Spoke

```yaml
sso:
  providers:
    - id: google
      type: oidc
      name: Google
      client_id: "your-google-client-id.apps.googleusercontent.com"
      client_secret: "your-google-client-secret"
      issuer_url: "https://accounts.google.com"
      redirect_url: "https://spoke.company.com/auth/callback/google"
      scopes:
        - openid
        - profile
        - email
      attribute_mapping:
        user_id: sub
        email: email
        first_name: given_name
        last_name: family_name
      hd_restriction: "company.com"  # Restrict to domain
```

## SAML 2.0 Setup

### Generic SAML Provider

```yaml
sso:
  providers:
    - id: saml-idp
      type: saml
      name: Corporate SAML
      entity_id: "https://spoke.company.com"
      sso_url: "https://idp.company.com/sso"
      certificate: |
        -----BEGIN CERTIFICATE-----
        MIIDXTCCAkWgAwIBAgIJAKL...
        -----END CERTIFICATE-----
      attribute_mapping:
        user_id: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/nameidentifier"
        email: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
        first_name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname"
        last_name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname"
        groups: "http://schemas.xmlsoap.org/claims/Group"
      role_mapping:
        - sso_group: "spoke_admins"
          spoke_role: "org:admin"
```

## Just-In-Time (JIT) Provisioning

When enabled, users are automatically created on first login.

### Configuration

```yaml
sso:
  jit_provisioning: true
  jit_config:
    enabled: true
    default_organization: "default-org"
    default_role: "org:viewer"
    update_on_login: true  # Update user info on each login
```

### User Creation Flow

1. User authenticates via SSO
2. Spoke checks if user exists (by SSO user ID)
3. If not exists:
   - Create new user with mapped attributes
   - Assign to organization
   - Apply role mappings
4. If exists:
   - Update user attributes (if `update_on_login: true`)
   - Sync group/role memberships

## Role Mapping

Map SSO groups to Spoke roles:

```yaml
role_mapping:
  # Exact match
  - sso_group: "Spoke-Admins"
    spoke_role: "org:admin"
    organization: "default-org"

  # Multiple groups → same role
  - sso_group: "Engineering"
    spoke_role: "org:developer"
  - sso_group: "DevOps"
    spoke_role: "org:developer"

  # Module-specific permissions
  - sso_group: "User-Service-Team"
    spoke_role: "module:maintainer"
    resource_id: "module-user"
```

## Login Flow

### 1. Initiate SSO Login

```http
GET /auth/sso/login?provider=azure
```

Redirects to Azure AD login.

### 2. User Authenticates

User logs in at Azure AD.

### 3. Callback

Azure redirects to:

```
https://spoke.company.com/auth/callback/azure?code=...
```

### 4. Token Exchange

Spoke exchanges code for tokens.

### 5. User Creation/Update

JIT provisioning creates or updates user.

### 6. Session Created

User is redirected to Spoke with session cookie.

## API Usage with SSO

### Get SSO Token

After SSO login, get an API token:

```http
POST /auth/token
Authorization: Bearer <session-cookie>

{
  "name": "CLI Token",
  "expires_in": 2592000  // 30 days
}
```

**Response:**

```json
{
  "token": "spoke_xxxxxxxxxxxx",
  "expires_at": "2025-02-23T10:00:00Z"
}
```

### Use Token with CLI

```bash
export SPOKE_TOKEN="spoke_xxxxxxxxxxxx"

spoke-cli push -module user -version v1.0.0 -dir ./proto
```

## Session Management

### Configuration

```yaml
auth:
  session_timeout: 24h
  session_max_age: 168h  # 7 days
  session_cookie_name: "spoke_session"
  session_cookie_secure: true
  session_cookie_http_only: true
```

### Logout

```http
POST /auth/logout
```

Invalidates session and clears cookie.

## Security Best Practices

### 1. Use HTTPS

Always use HTTPS in production:

```yaml
server:
  tls_enabled: true
  tls_cert: /path/to/cert.pem
  tls_key: /path/to/key.pem
```

### 2. Validate Certificates

For SAML, validate IdP certificates:

```yaml
sso:
  providers:
    - id: saml-idp
      type: saml
      validate_signature: true
      certificate: |
        -----BEGIN CERTIFICATE-----
        ...
        -----END CERTIFICATE-----
```

### 3. Restrict Domains (OIDC)

For Google Workspace, restrict to your domain:

```yaml
hd_restriction: "company.com"
```

### 4. Short Session Timeouts

Use reasonable session timeouts:

```yaml
auth:
  session_timeout: 8h
  session_max_age: 24h
```

### 5. Audit SSO Logins

Enable audit logging:

```yaml
audit:
  enabled: true
  log_auth_events: true
```

## Troubleshooting

### Login Fails

**Check logs:**

```bash
tail -f /var/log/spoke/server.log | grep SSO
```

**Common issues:**

- Incorrect client ID/secret
- Wrong redirect URI
- Missing scopes
- Certificate validation failure (SAML)

### User Not Provisioned

**Check JIT config:**

```yaml
sso:
  jit_provisioning: true
  jit_config:
    enabled: true
```

**Check logs:**

```bash
tail -f /var/log/spoke/server.log | grep "JIT"
```

### Group Mapping Not Working

**Verify groups are included in claims:**

For Azure AD:
1. Azure Portal → App Registration → Token configuration
2. Add optional claim: `groups`

For Okta:
1. Add `groups` scope to authorization server
2. Create groups claim

**Check attribute mapping:**

```yaml
attribute_mapping:
  groups: groups  # Must match claim name
```

### Session Expires Too Quickly

**Increase timeout:**

```yaml
auth:
  session_timeout: 24h
  session_max_age: 168h
```

## Examples

### Complete Azure AD Setup

```yaml
sso:
  enabled: true
  default_provider: azure
  jit_provisioning: true
  jit_config:
    enabled: true
    default_organization: "acme-corp"
    default_role: "org:viewer"
    update_on_login: true

  providers:
    - id: azure
      type: oidc
      name: Azure AD
      client_id: "abc123-def456"
      client_secret: "secret-value"
      issuer_url: "https://login.microsoftonline.com/tenant-id/v2.0"
      redirect_url: "https://spoke.acme.com/auth/callback/azure"
      scopes:
        - openid
        - profile
        - email
        - offline_access
      attribute_mapping:
        user_id: sub
        email: email
        first_name: given_name
        last_name: family_name
        groups: groups
      role_mapping:
        - sso_group: "Spoke-Admins"
          spoke_role: "org:admin"
        - sso_group: "Engineering"
          spoke_role: "org:developer"
        - sso_group: "Product"
          spoke_role: "org:viewer"

auth:
  session_timeout: 12h
  session_max_age: 168h

audit:
  enabled: true
  log_auth_events: true
```

## Next Steps

- [RBAC Configuration](/guides/rbac/) - Role-based access control
- [Multi-Tenancy](/guides/multi-tenancy/) - Organization management
- [Audit Logging](/guides/audit/) - Compliance and auditing
