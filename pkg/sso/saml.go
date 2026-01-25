package sso

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"time"

	saml2 "github.com/russellhaering/gosaml2"
	dsig "github.com/russellhaering/goxmldsig"
)

// SAMLProvider implements SAML 2.0 SSO
type SAMLProvider struct {
	config     *ProviderConfig
	sp         *saml2.SAMLServiceProvider
	baseURL    string
}

// NewSAMLProvider creates a new SAML provider
func NewSAMLProvider(config *ProviderConfig, baseURL string) (*SAMLProvider, error) {
	if config.SAMLConfig == nil {
		return nil, fmt.Errorf("SAML config is required")
	}

	// Parse certificate
	certBlock, _ := pem.Decode([]byte(config.SAMLConfig.Certificate))
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	certStore := dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{cert},
	}

	// Parse private key if provided
	var keyStore dsig.X509KeyStore
	if config.SAMLConfig.PrivateKey != "" {
		keyBlock, _ := pem.Decode([]byte(config.SAMLConfig.PrivateKey))
		if keyBlock == nil {
			return nil, fmt.Errorf("failed to decode private key PEM")
		}

		privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err != nil {
			// Try PKCS8 format
			pkcs8Key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
			var ok bool
			privateKey, ok = pkcs8Key.(*rsa.PrivateKey)
			if !ok {
				return nil, fmt.Errorf("private key is not RSA")
			}
		}

		keyStore = &dsig.TLSCertKeyStore{
			PrivateKey:  privateKey,
			Certificate: [][]byte{[]byte(config.SAMLConfig.Certificate)},
		}
	}

	// Create SAML service provider
	sp := &saml2.SAMLServiceProvider{
		IdentityProviderSSOURL:      config.SAMLConfig.SSOURL,
		IdentityProviderIssuer:      config.SAMLConfig.EntityID,
		ServiceProviderIssuer:       baseURL + "/sso/metadata",
		AssertionConsumerServiceURL: baseURL + fmt.Sprintf("/auth/sso/%s/callback", config.Name),
		SignAuthnRequests:           config.SAMLConfig.SignRequests,
		AudienceURI:                 baseURL,
		IDPCertificateStore:         &certStore,
		SPKeyStore:                  keyStore,
	}

	// Set NameID format
	if config.SAMLConfig.NameIDFormat != "" {
		sp.NameIdFormat = config.SAMLConfig.NameIDFormat
	}

	return &SAMLProvider{
		config:  config,
		sp:      sp,
		baseURL: baseURL,
	}, nil
}

// GetType returns the provider type
func (p *SAMLProvider) GetType() ProviderType {
	return ProviderTypeSAML
}

// GetName returns the provider name
func (p *SAMLProvider) GetName() ProviderName {
	return p.config.ProviderName
}

// InitiateLogin redirects to IdP for authentication
func (p *SAMLProvider) InitiateLogin(w http.ResponseWriter, r *http.Request, state string) error {
	// Build AuthnRequest
	authnRequest, err := p.sp.BuildAuthRequest()
	if err != nil {
		return fmt.Errorf("failed to build auth request: %w", err)
	}

	// Encode and redirect
	authURL, err := p.sp.BuildAuthURL(state)
	if err != nil {
		return fmt.Errorf("failed to build auth URL: %w", err)
	}

	// Store RelayState in session/cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "saml_relay_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	_ = authnRequest // Used by BuildAuthURL internally

	http.Redirect(w, r, authURL, http.StatusFound)
	return nil
}

// HandleCallback processes SAML assertion
func (p *SAMLProvider) HandleCallback(w http.ResponseWriter, r *http.Request) (*SSOUser, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	// Get SAMLResponse
	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		return nil, fmt.Errorf("missing SAMLResponse parameter")
	}

	// Decode base64
	assertionBytes, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SAMLResponse: %w", err)
	}

	// Parse and validate assertion
	assertionInfo, err := p.sp.RetrieveAssertionInfo(string(assertionBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to validate assertion: %w", err)
	}

	// Validate WarningInfo
	if assertionInfo.WarningInfo != nil {
		if assertionInfo.WarningInfo.InvalidTime {
			return nil, fmt.Errorf("assertion has invalid time")
		}
		if assertionInfo.WarningInfo.NotInAudience {
			return nil, fmt.Errorf("assertion not in expected audience")
		}
	}

	// Extract user information
	ssoUser := &SSOUser{
		ProviderID:   p.config.ID,
		ProviderName: p.config.Name,
		Attributes:   make(map[string]string),
	}

	// Map attributes
	for _, attr := range assertionInfo.Values {
		ssoUser.Attributes[attr.Name] = attr.Values[0].Value

		switch attr.Name {
		case p.config.AttributeMapping.UserID:
			ssoUser.ExternalID = attr.Values[0].Value
		case p.config.AttributeMapping.Username:
			ssoUser.Username = attr.Values[0].Value
		case p.config.AttributeMapping.Email:
			ssoUser.Email = attr.Values[0].Value
		case p.config.AttributeMapping.FullName:
			ssoUser.FullName = attr.Values[0].Value
		case p.config.AttributeMapping.FirstName:
			ssoUser.FirstName = attr.Values[0].Value
		case p.config.AttributeMapping.LastName:
			ssoUser.LastName = attr.Values[0].Value
		case p.config.AttributeMapping.Groups:
			// Groups may be multi-valued
			for _, v := range attr.Values {
				ssoUser.Groups = append(ssoUser.Groups, v.Value)
			}
		}
	}

	// Use NameID as fallback for user ID
	if ssoUser.ExternalID == "" {
		ssoUser.ExternalID = assertionInfo.NameID
	}

	// Use email as fallback for username
	if ssoUser.Username == "" && ssoUser.Email != "" {
		ssoUser.Username = ssoUser.Email
	}

	// Validate required fields
	if ssoUser.ExternalID == "" {
		return nil, fmt.Errorf("missing user ID in SAML assertion")
	}
	if ssoUser.Email == "" {
		return nil, fmt.Errorf("missing email in SAML assertion")
	}

	return ssoUser, nil
}

// Logout handles SAML logout
func (p *SAMLProvider) Logout(w http.ResponseWriter, r *http.Request, sessionIndex string) error {
	if p.config.SAMLConfig.SLOUrl == "" {
		// No SLO configured, just clear local session
		return nil
	}

	// Build basic LogoutRequest XML
	logoutRequestXML := fmt.Sprintf(`<?xml version="1.0"?>
<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
                     xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
                     ID="_%s"
                     Version="2.0"
                     IssueInstant="%s"
                     Destination="%s">
  <saml:Issuer>%s</saml:Issuer>
  <saml:NameID Format="urn:oasis:names:tc:SAML:2.0:nameid-format:transient"></saml:NameID>
  <samlp:SessionIndex>%s</samlp:SessionIndex>
</samlp:LogoutRequest>`,
		generateID(),
		time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		p.config.SAMLConfig.SLOUrl,
		p.sp.ServiceProviderIssuer,
		sessionIndex)

	// Encode and build logout URL
	encodedRequest := base64.StdEncoding.EncodeToString([]byte(logoutRequestXML))
	logoutURL, err := url.Parse(p.config.SAMLConfig.SLOUrl)
	if err != nil {
		return fmt.Errorf("invalid SLO URL: %w", err)
	}

	query := logoutURL.Query()
	query.Set("SAMLRequest", encodedRequest)
	logoutURL.RawQuery = query.Encode()

	http.Redirect(w, r, logoutURL.String(), http.StatusFound)
	return nil
}

// generateID generates a random ID for SAML requests
func generateID() string {
	b := make([]byte, 20)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ValidateConfig validates the SAML configuration
func (p *SAMLProvider) ValidateConfig() error {
	if p.config.SAMLConfig == nil {
		return fmt.Errorf("SAML config is required")
	}

	cfg := p.config.SAMLConfig

	if cfg.EntityID == "" {
		return fmt.Errorf("entity_id is required")
	}
	if cfg.SSOURL == "" {
		return fmt.Errorf("sso_url is required")
	}
	if cfg.Certificate == "" {
		return fmt.Errorf("certificate is required")
	}

	// Validate certificate format
	block, _ := pem.Decode([]byte(cfg.Certificate))
	if block == nil {
		return fmt.Errorf("invalid certificate PEM format")
	}

	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("invalid certificate: %w", err)
	}

	// Validate private key if present
	if cfg.PrivateKey != "" {
		keyBlock, _ := pem.Decode([]byte(cfg.PrivateKey))
		if keyBlock == nil {
			return fmt.Errorf("invalid private key PEM format")
		}
	}

	return nil
}

// GetMetadata returns the service provider metadata
func (p *SAMLProvider) GetMetadata() ([]byte, error) {
	metadata, err := p.sp.Metadata()
	if err != nil {
		return nil, fmt.Errorf("failed to generate metadata: %w", err)
	}

	// Marshal to XML
	metadataXML := fmt.Sprintf(`<?xml version="1.0"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"
                     entityID="%s">
  <md:SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                                 Location="%s"
                                 index="1"/>
  </md:SPSSODescriptor>
</md:EntityDescriptor>`,
		p.sp.ServiceProviderIssuer,
		p.sp.AssertionConsumerServiceURL)

	_ = metadata // Use the metadata object if needed in future
	return []byte(metadataXML), nil
}
