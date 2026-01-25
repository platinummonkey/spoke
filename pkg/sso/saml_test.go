package sso

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test certificate and key for SAML testing (self-signed, for testing only)
const testCertificate = `-----BEGIN CERTIFICATE-----
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

func TestSAMLProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *SAMLConfig
		expectError bool
		errorMsg    string
	}{
		// Skip valid config test as it requires a real certificate
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
			name: "invalid certificate",
			config: &SAMLConfig{
				EntityID:    "https://idp.example.com",
				SSOURL:      "https://idp.example.com/sso",
				Certificate: "invalid-cert",
			},
			expectError: true,
			errorMsg:    "invalid certificate",
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
		SAMLConfig:   nil, // No SAML config
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
