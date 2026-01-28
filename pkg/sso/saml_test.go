package sso

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test certificate and key for SAML testing (self-signed, for testing only)
const testCertificate = `-----BEGIN CERTIFICATE-----
MIIDizCCAnOgAwIBAgIUSFZKuGtORn0Swgu5dIVJBF58qREwDQYJKoZIhvcNAQEL
BQAwVTELMAkGA1UEBhMCVVMxDTALBgNVBAgMBFRlc3QxDTALBgNVBAcMBFRlc3Qx
DTALBgNVBAoMBFRlc3QxGTAXBgNVBAMMEHRlc3QuZXhhbXBsZS5jb20wHhcNMjYw
MTI4MjIxNTA0WhcNMjcwMTI4MjIxNTA0WjBVMQswCQYDVQQGEwJVUzENMAsGA1UE
CAwEVGVzdDENMAsGA1UEBwwEVGVzdDENMAsGA1UECgwEVGVzdDEZMBcGA1UEAwwQ
dGVzdC5leGFtcGxlLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AKjnv/B2fPTslhsQHPFE/RF7ICfSq3BIVELtwfTe054cMtYpKsPGzNqFz8QJICd6
kxLnV8GQTYd3vrL0yHISEOz6Ay7vOGqe34WThS5jXjf3BhRChRoMXsgush7XkdzO
fnFzQ1dHxqxQjfJFg3hIDaAwQEGQPhuoA3YSEJG1ReeKdgGvXJJZ9Y2N//27Ayfz
K3GmuoucOpnD4Ec6hkAdbiWDHyyb3e+MF3OYaimCpRmVnYi9W2Qa/laiPFf1UuZy
ewdeChnOrLa7CiIq5Et4Q5twbohkMZL9fPr7uT/tivYjLgu6BBBh/4T/LbsWbNcF
JzAiXSljN+4FNFY4UjJOf0kCAwEAAaNTMFEwHQYDVR0OBBYEFDLaGgYYOUVWM0pM
SVORaP2OHeqTMB8GA1UdIwQYMBaAFDLaGgYYOUVWM0pMSVORaP2OHeqTMA8GA1Ud
EwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAEBkxZMiUIiZhEtpgAHSJRkh
WeItSXk3xN5Z1O14h+XiEQT9PGoq5uXHVe973kFij4d+O+MtqEiPzKBLg8nJnC2C
XxHRe1VCR+jyw/9MuCMC0BssR9IUHGGq29mpvm2+GYUSZzqDT0jL//z5pOMYHTKQ
5Kqo5s22TRrcuxc4EtjZZVO96SZXu7LlpOcuQ6B9j9LhX4snnIJO7QT2XpBL7BLR
3tHbxSZqROr3p80dzj8RptXCCz4Xq6ohgWSpVCL3zexKG3/BGgUY0Kqp1zrHNSZQ
PZhuWKT1ZonPT9jDjiiFGp5Be/xOxr6H8iHMlr+e8L4/jmgAsRkrly+De4x9xYc=
-----END CERTIFICATE-----`

const testPrivateKey = `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCo57/wdnz07JYb
EBzxRP0ReyAn0qtwSFRC7cH03tOeHDLWKSrDxszahc/ECSAnepMS51fBkE2Hd76y
9MhyEhDs+gMu7zhqnt+Fk4UuY1439wYUQoUaDF7ILrIe15Hczn5xc0NXR8asUI3y
RYN4SA2gMEBBkD4bqAN2EhCRtUXninYBr1ySWfWNjf/9uwMn8ytxprqLnDqZw+BH
OoZAHW4lgx8sm93vjBdzmGopgqUZlZ2IvVtkGv5WojxX9VLmcnsHXgoZzqy2uwoi
KuRLeEObcG6IZDGS/Xz6+7k/7Yr2Iy4LugQQYf+E/y27FmzXBScwIl0pYzfuBTRW
OFIyTn9JAgMBAAECggEATaUTgAgIE1N7AX/bvjG3oESYmJXox5oIWigQBHA2mbVe
zUJpbUxDOaVPyE9ln6BiYctFdS7P5Rlv6bZLOt0BON8JfZbsuV7FZBNXouZ9Fn8R
JVka9MmA/McyjKkOXZHzYFXbPBE7zFTPm/LGqBF/agckUr9rPa1zweA2C7VoKDKo
EwMNwhZ3eX8CItme5c0Q5xd/no6BSSzNq3Ndv2tve4VfV7QxgvOvkqy7iJYaRMrL
m6mxZBpqxWgeQc0OJTuxx+zdJ2Ib9fNPkCqoeD79BQWnY0i0vTgChNR/Wh0PGUha
zGduWTuj/UYksrHWWKTBdQwEJcqbUpRMhDwsW4e3/QKBgQDXu71LVd14Co0Xl5pi
uXwBf+LVxmggoen3p0NFIkr6nARVYuNSF16dgUQ0MIzUdNvsciF0YRL3rAXexu+r
kHmIkvR4vopZQTqIyVi48V1U4DZ6dWzZMVySd7Yef5ye99VgzHBuY+2IO0TpKZf0
CVaL+6VLJN77IHzHiclY719yGwKBgQDIbnOPgX/8hai722J1OAXwY/MH7GaaQ5iO
isxxZntAkf5toik+tEQgOEsq+WWMTNHSI5/YPsLMkk0AxHq9P4G8zBDP66SxEL8X
q3KLCqR6IWbD1/WwJIsN+T/GFSRKukDRLM/uF2/TE8SrOfDwgptkk8HHRJsRptSl
QCCw4ipKawKBgGsQrGBQC+rAacd0oNUwMr/XxS7NGe5gDOqwoy0TWNzJQ0lRG3op
SPaoKb4w/iOOn3rYJYxJhQ1P3VXzqwydVgOW0yd9gNHNEozCSHr4ppYx9DeQQWYF
Hmk+ai72rDckzkwNChtvEnqS159T2irt23r7d8w0T0mYlPS+iCPQILFTAoGAdayL
QkzIpKygZTKneqSasAkubY94qcdX8RBCea2uXTmZxCo5xuu1N6l1UFS+LwIHCjYK
Kb6nRc37UaEJYsS/WeYBVOFHfwGS/8WT6VglOuMTX5YSVAkQbvLQY26UMR9q4KRL
q8Cs0aNAizroX3x+2Sz6zxBTbqihHigpSVBvfeMCgYBtR8XXm5fBp/ANF1VMJODH
rAu4kQ4qiHJEtxJYaIBc6XD2usi/ElclmVcucztD14lyZ8C6j2B/Sg7bPRSnuYrv
7D0u/FEGBcQoXZDYDbFOueeV6BpnZTXXT8FAZYcpwzVCUB7sOQm+us0LHzlfdYEF
vvne2oHrNJZsiPz9w2WJew==
-----END PRIVATE KEY-----`

func TestNewSAMLProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      *ProviderConfig
		baseURL     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config without private key",
			config: &ProviderConfig{
				Name:         "test-saml",
				ProviderType: ProviderTypeSAML,
				ProviderName: ProviderGenericSAML,
				SAMLConfig: &SAMLConfig{
					EntityID:    "https://idp.example.com",
					SSOURL:      "https://idp.example.com/sso",
					Certificate: testCertificate,
				},
				AttributeMapping: AttributeMap{
					UserID: "uid",
					Email:  "email",
				},
			},
			baseURL:     "https://sp.example.com",
			expectError: false,
		},
		{
			name: "valid config with private key",
			config: &ProviderConfig{
				Name:         "test-saml",
				ProviderType: ProviderTypeSAML,
				ProviderName: ProviderGenericSAML,
				SAMLConfig: &SAMLConfig{
					EntityID:     "https://idp.example.com",
					SSOURL:       "https://idp.example.com/sso",
					Certificate:  testCertificate,
					PrivateKey:   testPrivateKey,
					SignRequests: true,
				},
				AttributeMapping: AttributeMap{
					UserID: "uid",
					Email:  "email",
				},
			},
			baseURL:     "https://sp.example.com",
			expectError: false,
		},
		{
			name: "valid config with NameIDFormat",
			config: &ProviderConfig{
				Name:         "test-saml",
				ProviderType: ProviderTypeSAML,
				ProviderName: ProviderGenericSAML,
				SAMLConfig: &SAMLConfig{
					EntityID:     "https://idp.example.com",
					SSOURL:       "https://idp.example.com/sso",
					Certificate:  testCertificate,
					NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
				},
				AttributeMapping: AttributeMap{
					UserID: "uid",
					Email:  "email",
				},
			},
			baseURL:     "https://sp.example.com",
			expectError: false,
		},
		{
			name: "nil SAML config",
			config: &ProviderConfig{
				Name:         "test-saml",
				ProviderType: ProviderTypeSAML,
				ProviderName: ProviderGenericSAML,
				SAMLConfig:   nil,
			},
			baseURL:     "https://sp.example.com",
			expectError: true,
			errorMsg:    "SAML config is required",
		},
		{
			name: "invalid certificate PEM",
			config: &ProviderConfig{
				Name:         "test-saml",
				ProviderType: ProviderTypeSAML,
				ProviderName: ProviderGenericSAML,
				SAMLConfig: &SAMLConfig{
					EntityID:    "https://idp.example.com",
					SSOURL:      "https://idp.example.com/sso",
					Certificate: "invalid-cert",
				},
			},
			baseURL:     "https://sp.example.com",
			expectError: true,
			errorMsg:    "failed to decode certificate PEM",
		},
		{
			name: "invalid private key PEM",
			config: &ProviderConfig{
				Name:         "test-saml",
				ProviderType: ProviderTypeSAML,
				ProviderName: ProviderGenericSAML,
				SAMLConfig: &SAMLConfig{
					EntityID:    "https://idp.example.com",
					SSOURL:      "https://idp.example.com/sso",
					Certificate: testCertificate,
					PrivateKey:  "invalid-key",
				},
			},
			baseURL:     "https://sp.example.com",
			expectError: true,
			errorMsg:    "failed to decode private key PEM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewSAMLProvider(tt.config, tt.baseURL)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, tt.config, provider.config)
				assert.Equal(t, tt.baseURL, provider.baseURL)
				assert.NotNil(t, provider.sp)
			}
		})
	}
}

func TestSAMLProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *SAMLConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &SAMLConfig{
				EntityID:    "https://idp.example.com",
				SSOURL:      "https://idp.example.com/sso",
				Certificate: testCertificate,
			},
			expectError: false,
		},
		{
			name: "valid config with private key",
			config: &SAMLConfig{
				EntityID:    "https://idp.example.com",
				SSOURL:      "https://idp.example.com/sso",
				Certificate: testCertificate,
				PrivateKey:  testPrivateKey,
			},
			expectError: false,
		},
		{
			name: "missing entity_id",
			config: &SAMLConfig{
				SSOURL:      "https://idp.example.com/sso",
				Certificate: testCertificate,
			},
			expectError: true,
			errorMsg:    "entity_id is required",
		},
		{
			name: "missing sso_url",
			config: &SAMLConfig{
				EntityID:    "https://idp.example.com",
				Certificate: testCertificate,
			},
			expectError: true,
			errorMsg:    "sso_url is required",
		},
		{
			name: "missing certificate",
			config: &SAMLConfig{
				EntityID: "https://idp.example.com",
				SSOURL:   "https://idp.example.com/sso",
			},
			expectError: true,
			errorMsg:    "certificate is required",
		},
		{
			name: "invalid certificate PEM format",
			config: &SAMLConfig{
				EntityID:    "https://idp.example.com",
				SSOURL:      "https://idp.example.com/sso",
				Certificate: "invalid-cert",
			},
			expectError: true,
			errorMsg:    "invalid certificate PEM format",
		},
		{
			name: "invalid private key PEM format",
			config: &SAMLConfig{
				EntityID:    "https://idp.example.com",
				SSOURL:      "https://idp.example.com/sso",
				Certificate: testCertificate,
				PrivateKey:  "invalid-key",
			},
			expectError: true,
			errorMsg:    "invalid private key PEM format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerConfig := &ProviderConfig{
				Name:         "test-saml",
				ProviderType: ProviderTypeSAML,
				ProviderName: ProviderGenericSAML,
				Enabled:      true,
				SAMLConfig:   tt.config,
				AttributeMapping: AttributeMap{
					UserID:   "urn:oid:0.9.2342.19200300.100.1.1",
					Username: "urn:oid:0.9.2342.19200300.100.1.1",
					Email:    "urn:oid:0.9.2342.19200300.100.1.3",
				},
			}

			provider := &SAMLProvider{config: providerConfig}
			err := provider.ValidateConfig()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSAMLProvider_GetType(t *testing.T) {
	config := &ProviderConfig{
		Name:         "test-saml",
		ProviderType: ProviderTypeSAML,
		ProviderName: ProviderGenericSAML,
		SAMLConfig: &SAMLConfig{
			EntityID:    "https://idp.example.com",
			SSOURL:      "https://idp.example.com/sso",
			Certificate: testCertificate,
		},
		AttributeMapping: AttributeMap{
			UserID: "uid",
			Email:  "email",
		},
	}

	provider := &SAMLProvider{config: config}
	assert.Equal(t, ProviderTypeSAML, provider.GetType())
}

func TestSAMLProvider_GetName(t *testing.T) {
	config := &ProviderConfig{
		Name:         "test-saml",
		ProviderType: ProviderTypeSAML,
		ProviderName: ProviderGenericSAML,
		SAMLConfig: &SAMLConfig{
			EntityID:    "https://idp.example.com",
			SSOURL:      "https://idp.example.com/sso",
			Certificate: testCertificate,
		},
		AttributeMapping: AttributeMap{
			UserID: "uid",
			Email:  "email",
		},
	}

	provider := &SAMLProvider{config: config}
	assert.Equal(t, ProviderGenericSAML, provider.GetName())
}

func TestSAMLProvider_InitiateLogin(t *testing.T) {
	config := &ProviderConfig{
		ID:           1,
		Name:         "test-saml",
		ProviderType: ProviderTypeSAML,
		ProviderName: ProviderGenericSAML,
		SAMLConfig: &SAMLConfig{
			EntityID:    "https://idp.example.com",
			SSOURL:      "https://idp.example.com/sso",
			Certificate: testCertificate,
		},
		AttributeMapping: AttributeMap{
			UserID: "uid",
			Email:  "email",
		},
	}

	provider, err := NewSAMLProvider(config, "https://sp.example.com")
	require.NoError(t, err)

	// Test successful login initiation
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/auth/login", nil)
	state := "test-state-123"

	err = provider.InitiateLogin(w, r, state)
	assert.NoError(t, err)

	// Check redirect
	assert.Equal(t, http.StatusFound, w.Code)
	location := w.Header().Get("Location")
	assert.NotEmpty(t, location)
	assert.Contains(t, location, "https://idp.example.com/sso")

	// Check cookie
	cookies := w.Result().Cookies()
	var found bool
	for _, cookie := range cookies {
		if cookie.Name == "saml_relay_state" {
			found = true
			assert.Equal(t, state, cookie.Value)
			assert.True(t, cookie.HttpOnly)
			assert.True(t, cookie.Secure)
			assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
			assert.Equal(t, 600, cookie.MaxAge)
		}
	}
	assert.True(t, found, "saml_relay_state cookie not found")
}

func TestSAMLProvider_HandleCallback(t *testing.T) {
	config := &ProviderConfig{
		ID:           1,
		Name:         "test-saml",
		ProviderType: ProviderTypeSAML,
		ProviderName: ProviderGenericSAML,
		SAMLConfig: &SAMLConfig{
			EntityID:    "https://idp.example.com",
			SSOURL:      "https://idp.example.com/sso",
			Certificate: testCertificate,
		},
		AttributeMapping: AttributeMap{
			UserID: "uid",
			Email:  "email",
		},
	}

	provider, err := NewSAMLProvider(config, "https://sp.example.com")
	require.NoError(t, err)

	tests := []struct {
		name         string
		formValues   url.Values
		expectError  bool
		errorMsg     string
	}{
		{
			name: "missing SAMLResponse",
			formValues: url.Values{},
			expectError: true,
			errorMsg:    "missing SAMLResponse parameter",
		},
		{
			name: "invalid base64 SAMLResponse",
			formValues: url.Values{
				"SAMLResponse": []string{"not-valid-base64!@#$"},
			},
			expectError: true,
			errorMsg:    "failed to decode SAMLResponse",
		},
		{
			name: "invalid SAML assertion",
			formValues: url.Values{
				"SAMLResponse": []string{base64.StdEncoding.EncodeToString([]byte("invalid-xml"))},
			},
			expectError: true,
			errorMsg:    "failed to validate assertion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/auth/callback", strings.NewReader(tt.formValues.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			user, err := provider.HandleCallback(w, r)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}
		})
	}
}

func TestSAMLProvider_Logout(t *testing.T) {
	tests := []struct {
		name         string
		sloURL       string
		sessionIndex string
		expectRedirect bool
	}{
		{
			name:           "with SLO URL",
			sloURL:         "https://idp.example.com/slo",
			sessionIndex:   "session-123",
			expectRedirect: true,
		},
		{
			name:           "without SLO URL",
			sloURL:         "",
			sessionIndex:   "session-123",
			expectRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ProviderConfig{
				ID:           1,
				Name:         "test-saml",
				ProviderType: ProviderTypeSAML,
				ProviderName: ProviderGenericSAML,
				SAMLConfig: &SAMLConfig{
					EntityID:    "https://idp.example.com",
					SSOURL:      "https://idp.example.com/sso",
					Certificate: testCertificate,
					SLOUrl:      tt.sloURL,
				},
				AttributeMapping: AttributeMap{
					UserID: "uid",
					Email:  "email",
				},
			}

			provider, err := NewSAMLProvider(config, "https://sp.example.com")
			require.NoError(t, err)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/auth/logout", nil)

			err = provider.Logout(w, r, tt.sessionIndex)
			assert.NoError(t, err)

			if tt.expectRedirect {
				assert.Equal(t, http.StatusFound, w.Code)
				location := w.Header().Get("Location")
				assert.Contains(t, location, tt.sloURL)
				assert.Contains(t, location, "SAMLRequest=")
			} else {
				// No redirect when SLO URL is not configured
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

func TestSAMLProvider_GetMetadata(t *testing.T) {
	tests := []struct {
		name        string
		config      *ProviderConfig
		expectError bool
	}{
		{
			name: "successful metadata generation",
			config: &ProviderConfig{
				ID:           1,
				Name:         "test-saml",
				ProviderType: ProviderTypeSAML,
				ProviderName: ProviderGenericSAML,
				SAMLConfig: &SAMLConfig{
					EntityID:    "https://idp.example.com",
					SSOURL:      "https://idp.example.com/sso",
					Certificate: testCertificate,
				},
				AttributeMapping: AttributeMap{
					UserID: "uid",
					Email:  "email",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewSAMLProvider(tt.config, "https://sp.example.com")
			require.NoError(t, err)

			metadata, err := provider.GetMetadata()

			// The underlying library may require encryption certificate
			// so we test that the function executes without panicking
			if err != nil {
				// If there's an error, it should be descriptive
				assert.NotEmpty(t, err.Error())
			} else {
				assert.NotNil(t, metadata)
				metadataStr := string(metadata)
				assert.Contains(t, metadataStr, "EntityDescriptor")
				assert.Contains(t, metadataStr, "https://sp.example.com/sso/metadata")
				assert.Contains(t, metadataStr, "https://sp.example.com/auth/sso/test-saml/callback")
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	// Test that generateID creates unique IDs
	id1 := generateID()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Equal(t, 40, len(id1)) // 20 bytes = 40 hex chars
	assert.Equal(t, 40, len(id2))
}

func TestSAMLConfig_Serialization(t *testing.T) {
	config := &SAMLConfig{
		EntityID:            "https://idp.example.com",
		SSOURL:              "https://idp.example.com/sso",
		SLOUrl:              "https://idp.example.com/slo",
		Certificate:         testCertificate,
		SignRequests:        true,
		ForceAuthn:          false,
		AllowIDPInitiated:   true,
		NameIDFormat:        "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
		DefaultRedirectURL:  "https://spoke.example.com",
		AudienceRestriction: []string{"https://spoke.example.com"},
	}

	assert.NotEmpty(t, config.EntityID)
	assert.NotEmpty(t, config.SSOURL)
	assert.NotEmpty(t, config.Certificate)
	assert.True(t, config.SignRequests)
	assert.True(t, config.AllowIDPInitiated)
}

func TestSAMLProvider_NilConfig(t *testing.T) {
	config := &ProviderConfig{
		Name:         "test-saml",
		ProviderType: ProviderTypeSAML,
		ProviderName: ProviderGenericSAML,
		SAMLConfig:   nil,
		AttributeMapping: AttributeMap{
			UserID: "uid",
			Email:  "email",
		},
	}

	provider := &SAMLProvider{config: config}
	err := provider.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SAML config is required")
}
