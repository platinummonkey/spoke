package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/platinummonkey/spoke/pkg/sso"
)

// This example demonstrates how to integrate SSO into your Spoke server

func main() {
	// 1. Connect to database
	db, err := sql.Open("postgres", "postgres://user:pass@localhost/spoke?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 2. Create router
	router := mux.NewRouter()

	// 3. Initialize SSO handlers
	baseURL := "https://spoke.example.com"
	ssoHandlers := sso.NewHandlers(db, baseURL)

	// 4. Register SSO routes
	ssoHandlers.RegisterRoutes(router)

	// 5. Add a protected endpoint that requires SSO authentication
	router.HandleFunc("/api/protected", func(w http.ResponseWriter, r *http.Request) {
		// Get authenticated user from SSO session
		authCtx, err := ssoHandlers.GetAuthContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message": "Hello, %s!", "user_id": %d}`, authCtx.User.Username, authCtx.User.ID)
	}).Methods("GET")

	// 6. Example: Configure Azure AD provider programmatically
	configureAzureAD(db)

	// 7. Example: Configure Okta provider
	configureOkta(db)

	// 8. Example: Configure generic SAML provider
	configureSAML(db)

	// 9. Start server
	log.Println("SSO-enabled Spoke server starting on :8080")
	log.Println("SSO login URLs:")
	log.Println("  - Azure AD: https://spoke.example.com/auth/sso/azuread/login")
	log.Println("  - Okta:     https://spoke.example.com/auth/sso/okta/login")
	log.Println("  - SAML:     https://spoke.example.com/auth/sso/saml-idp/login")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// configureAzureAD sets up Azure AD (Microsoft Entra ID) SSO
func configureAzureAD(db *sql.DB) {
	storage := sso.NewStorage(db)

	// Check if provider already exists
	exists, _ := storage.ProviderExists("azuread")
	if exists {
		log.Println("Azure AD provider already configured")
		return
	}

	// Get preset configuration
	config, err := sso.GetPresetConfig(sso.ProviderAzureAD)
	if err != nil {
		log.Printf("Failed to get Azure AD preset: %v", err)
		return
	}

	// Customize configuration
	config.Name = "azuread"
	config.Enabled = true
	config.AutoProvision = true
	config.DefaultRole = "developer"

	// Set Azure AD credentials (these should come from environment variables)
	config.OIDCConfig.ClientID = "your-azure-client-id"
	config.OIDCConfig.ClientSecret = "your-azure-client-secret"
	config.OIDCConfig.IssuerURL = "https://login.microsoftonline.com/your-tenant-id/v2.0"
	config.OIDCConfig.RedirectURL = "https://spoke.example.com/auth/sso/azuread/callback"

	// Configure group mappings
	config.GroupMapping = []sso.GroupMap{
		{SSOGroup: "Spoke-Admins", SpokeRole: "admin"},
		{SSOGroup: "Spoke-Developers", SpokeRole: "developer"},
		{SSOGroup: "Spoke-Viewers", SpokeRole: "viewer"},
	}

	// Save configuration
	if err := storage.CreateProvider(config); err != nil {
		log.Printf("Failed to create Azure AD provider: %v", err)
		return
	}

	log.Println("Azure AD provider configured successfully")
}

// configureOkta sets up Okta SSO
func configureOkta(db *sql.DB) {
	storage := sso.NewStorage(db)

	// Check if provider already exists
	exists, _ := storage.ProviderExists("okta")
	if exists {
		log.Println("Okta provider already configured")
		return
	}

	// Get preset configuration
	config, err := sso.GetPresetConfig(sso.ProviderOkta)
	if err != nil {
		log.Printf("Failed to get Okta preset: %v", err)
		return
	}

	// Customize configuration
	config.Name = "okta"
	config.Enabled = true
	config.AutoProvision = true
	config.DefaultRole = "developer"

	// Set Okta credentials
	config.OIDCConfig.ClientID = "your-okta-client-id"
	config.OIDCConfig.ClientSecret = "your-okta-client-secret"
	config.OIDCConfig.IssuerURL = "https://your-org.okta.com"
	config.OIDCConfig.RedirectURL = "https://spoke.example.com/auth/sso/okta/callback"

	// Configure group mappings
	config.GroupMapping = []sso.GroupMap{
		{SSOGroup: "Spoke Admins", SpokeRole: "admin"},
		{SSOGroup: "Spoke Users", SpokeRole: "developer"},
	}

	// Save configuration
	if err := storage.CreateProvider(config); err != nil {
		log.Printf("Failed to create Okta provider: %v", err)
		return
	}

	log.Println("Okta provider configured successfully")
}

// configureSAML sets up a generic SAML 2.0 provider
func configureSAML(db *sql.DB) {
	storage := sso.NewStorage(db)

	// Check if provider already exists
	exists, _ := storage.ProviderExists("saml-idp")
	if exists {
		log.Println("SAML provider already configured")
		return
	}

	// Create SAML configuration
	config := &sso.ProviderConfig{
		Name:          "saml-idp",
		ProviderType:  sso.ProviderTypeSAML,
		ProviderName:  sso.ProviderGenericSAML,
		Enabled:       true,
		AutoProvision: true,
		DefaultRole:   "developer",
		SAMLConfig: &sso.SAMLConfig{
			EntityID:           "https://idp.example.com/metadata",
			SSOURL:             "https://idp.example.com/sso/saml",
			SLOUrl:             "https://idp.example.com/slo/saml",
			Certificate:        loadSAMLCertificate(), // Load IdP certificate
			SignRequests:       true,
			ForceAuthn:         false,
			AllowIDPInitiated:  true,
			NameIDFormat:       "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			DefaultRedirectURL: "https://spoke.example.com",
		},
		AttributeMapping: sso.AttributeMap{
			UserID:    "urn:oid:0.9.2342.19200300.100.1.1",
			Username:  "urn:oid:0.9.2342.19200300.100.1.1",
			Email:     "urn:oid:0.9.2342.19200300.100.1.3",
			FullName:  "urn:oid:2.5.4.3",
			FirstName: "urn:oid:2.5.4.42",
			LastName:  "urn:oid:2.5.4.4",
			Groups:    "urn:oid:1.3.6.1.4.1.5923.1.5.1.1",
		},
		GroupMapping: []sso.GroupMap{
			{SSOGroup: "cn=admins,ou=groups,dc=example,dc=com", SpokeRole: "admin"},
			{SSOGroup: "cn=developers,ou=groups,dc=example,dc=com", SpokeRole: "developer"},
		},
	}

	// Save configuration
	if err := storage.CreateProvider(config); err != nil {
		log.Printf("Failed to create SAML provider: %v", err)
		return
	}

	log.Println("SAML provider configured successfully")
}

// loadSAMLCertificate loads the IdP certificate
// In production, this should load from a secure location
func loadSAMLCertificate() string {
	// Example: Load from file or environment variable
	return `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAJC1HiIAZAiIMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMjQwMTI0MDAwMDAwWhcNMjUwMTI0MDAwMDAwWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEAuGbXWiK3dQTyCbX5xdE4yCuYp0AF2d15Qq1JSXT/lx8CEcXb9RbDddl8
jGDv+spi5qPa8qEHiK7bulIaZxwtFST6wijF9uJhL9nNj5VQ8ZVHJmHN8/DYmwTW
rqjG+MaGrXmxJT4fNYYjDKh0v56aLFpLQJ5M7l9xNKRO0tVJQnYvW3I38SBBcnK/
J9q3f6fG4lJ2n7n0gTuZxcOsQH1MQ8B9jC9sCzJtCDkJBQBTNOjPbFQZkMtClPzG
BmB6gT8n1aV1LqLlWC7Nf3cZRkNkFvWXHrQJr5K+3p2yw3w+mW0wQQ4yJsP+6KvD
WVBGJlrQPKL0TjMQXiLqLqLDKLqA3wIDAQABo1AwTjAdBgNVHQ4EFgQUo7h7hXmq
UqtJR8cw0K8vxl3yNpowHwYDVR0jBBgwFoAUo7h7hXmqUqtJR8cw0K8vxl3yNpow
DAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEApdN2qBDwG3Gs8QdQW+B5
LLn3ILzLILCQmM8VW4r5gHEqMnE5Zk6q9kK7X3cZW6sLFN7eNGgZqJE+wPZGJLz1
NdRLMqfHcHCqOzDGPYGpLKrGKMKtVvLlvYPLmkZF0r5R9K8nDJwZQ7kMJ0hWLLqE
HqQ9tLxSLkr4L4vQFDGVJQqOBsJPVGqmQ8F8nYJqLqLnQXRkJdQlPZzLKJQqLmYx
G5GvKh6xLhYFHQQpQKqLnL9FLqLKqJLKqLqLQqLJqLqKLqKqLqKqLqKqLqKqLqKq
LqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKq
LqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLqKqLg==
-----END CERTIFICATE-----`
}

// Example middleware for SSO authentication
func SSOAuthMiddleware(ssoHandlers *sso.Handlers) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get authenticated user from SSO session
			authCtx, err := ssoHandlers.GetAuthContext(r)
			if err != nil {
				// No valid SSO session, redirect to login
				http.Redirect(w, r, "/auth/sso/azuread/login", http.StatusFound)
				return
			}

			// Check if user is active
			if !authCtx.User.IsActive {
				http.Error(w, "User account is inactive", http.StatusForbidden)
				return
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// Example: Protect specific routes with SSO
func protectedRoutes(router *mux.Router, ssoHandlers *sso.Handlers) {
	// Create a subrouter for protected routes
	protected := router.PathPrefix("/api").Subrouter()

	// Apply SSO authentication middleware
	protected.Use(SSOAuthMiddleware(ssoHandlers))

	// Add protected endpoints
	protected.HandleFunc("/modules", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"modules": ["module1", "module2"]}`)
	}).Methods("GET")

	protected.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		authCtx, _ := ssoHandlers.GetAuthContext(r)
		fmt.Fprintf(w, `{"username": "%s", "email": "%s"}`,
			authCtx.User.Username, authCtx.User.Email)
	}).Methods("GET")
}
