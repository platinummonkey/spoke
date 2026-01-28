// Package sso provides enterprise single sign-on (SSO) integration for the Spoke registry.
//
// # Overview
//
// This package enables authentication via SAML 2.0, OAuth2, and OpenID Connect with
// just-in-time (JIT) user provisioning and automatic role assignment based on SSO groups.
//
// # Supported Protocols
//
// SAML 2.0: Enterprise identity providers (Azure AD, Okta, OneLogin)
// OAuth2: Standard OAuth2 flows
// OpenID Connect: Modern authentication layer on top of OAuth2
//
// # Usage Example
//
// Configure SSO provider:
//
//	config := &sso.ProviderConfig{
//		Type:         sso.ProviderTypeSAML,
//		EntityID:     "https://spoke.example.com",
//		SSOURL:       "https://idp.example.com/sso/saml",
//		Certificate:  pemCert,
//		AttributeMap: sso.AttributeMap{
//			Email:    "email",
//			FullName: "displayName",
//			Groups:   "memberOf",
//		},
//		GroupMap: sso.GroupMap{
//			"Engineering": sso.RoleOrgDeveloper,
//			"Admins":      sso.RoleOrgAdmin,
//		},
//	}
//
// Provider factory with presets:
//
//	// Azure AD
//	provider := sso.NewAzureADProvider(tenantID, clientID, clientSecret)
//
//	// Okta
//	provider := sso.NewOktaProvider(domain, clientID, clientSecret)
//
//	// Google Workspace
//	provider := sso.NewGoogleProvider(clientID, clientSecret)
//
// # JIT User Provisioning
//
// When a user logs in via SSO for the first time, the system:
//   1. Validates authentication with IdP
//   2. Extracts user attributes (email, name, groups)
//   3. Creates user account in Spoke
//   4. Maps SSO groups to organization roles
//   5. Creates organization membership
//   6. Issues API token
//
// # Related Packages
//
//   - pkg/auth: User creation and token generation
//   - pkg/orgs: Organization membership and roles
package sso
