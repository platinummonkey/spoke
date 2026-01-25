package sso

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderConfigSerialization(t *testing.T) {
	config := &ProviderConfig{
		Name:          "test-provider",
		ProviderType:  ProviderTypeOIDC,
		ProviderName:  ProviderAzureAD,
		Enabled:       true,
		AutoProvision: true,
		DefaultRole:   "developer",
		GroupMapping: []GroupMap{
			{SSOGroup: "admins", SpokeRole: "admin"},
			{SSOGroup: "developers", SpokeRole: "developer"},
		},
		OIDCConfig: &OIDCConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-secret",
			IssuerURL:    "https://login.microsoftonline.com/tenant-id/v2.0",
			RedirectURL:  "https://spoke.example.com/auth/sso/azuread/callback",
			Scopes:       []string{"openid", "profile", "email"},
		},
		AttributeMapping: AttributeMap{
			UserID:   "oid",
			Username: "preferred_username",
			Email:    "email",
			Groups:   "groups",
		},
	}

	// Serialize
	data, err := json.Marshal(config)
	require.NoError(t, err)

	// Deserialize
	var decoded ProviderConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, config.Name, decoded.Name)
	assert.Equal(t, config.ProviderType, decoded.ProviderType)
	assert.Equal(t, config.Enabled, decoded.Enabled)
	assert.Equal(t, config.AutoProvision, decoded.AutoProvision)
	assert.Len(t, decoded.GroupMapping, 2)
	assert.NotNil(t, decoded.OIDCConfig)
	assert.Equal(t, config.OIDCConfig.ClientID, decoded.OIDCConfig.ClientID)
}

func TestSSOUserSerialization(t *testing.T) {
	user := &SSOUser{
		ExternalID: "12345",
		Username:   "john.doe",
		Email:      "john.doe@example.com",
		FullName:   "John Doe",
		Groups:     []string{"developers", "team-a"},
		Attributes: map[string]string{
			"department": "Engineering",
			"location":   "San Francisco",
		},
		ProviderID:   1,
		ProviderName: "azuread",
	}

	// Serialize
	data, err := json.Marshal(user)
	require.NoError(t, err)

	// Deserialize
	var decoded SSOUser
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, user.ExternalID, decoded.ExternalID)
	assert.Equal(t, user.Username, decoded.Username)
	assert.Equal(t, user.Email, decoded.Email)
	assert.Equal(t, user.Groups, decoded.Groups)
	assert.Equal(t, user.Attributes, decoded.Attributes)
}

func TestGroupMapping(t *testing.T) {
	mapping := GroupMap{
		SSOGroup:  "engineering-team",
		SpokeRole: "developer",
	}

	data, err := json.Marshal(mapping)
	require.NoError(t, err)

	var decoded GroupMap
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, mapping.SSOGroup, decoded.SSOGroup)
	assert.Equal(t, mapping.SpokeRole, decoded.SpokeRole)
}

func TestAttributeMapping(t *testing.T) {
	attrMap := AttributeMap{
		UserID:    "sub",
		Username:  "preferred_username",
		Email:     "email",
		FullName:  "name",
		FirstName: "given_name",
		LastName:  "family_name",
		Groups:    "groups",
	}

	data, err := json.Marshal(attrMap)
	require.NoError(t, err)

	var decoded AttributeMap
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, attrMap.UserID, decoded.UserID)
	assert.Equal(t, attrMap.Username, decoded.Username)
	assert.Equal(t, attrMap.Email, decoded.Email)
	assert.Equal(t, attrMap.Groups, decoded.Groups)
}

func TestProviderTypes(t *testing.T) {
	tests := []struct {
		name         string
		providerType ProviderType
		valid        bool
	}{
		{"SAML", ProviderTypeSAML, true},
		{"OAuth2", ProviderTypeOAuth2, true},
		{"OIDC", ProviderTypeOIDC, true},
		{"Invalid", ProviderType("invalid"), false},
	}

	validTypes := map[ProviderType]bool{
		ProviderTypeSAML:   true,
		ProviderTypeOAuth2: true,
		ProviderTypeOIDC:   true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, exists := validTypes[tt.providerType]
			assert.Equal(t, tt.valid, exists)
		})
	}
}

func TestProviderNames(t *testing.T) {
	validProviders := []ProviderName{
		ProviderAzureAD,
		ProviderOkta,
		ProviderGoogle,
		ProviderGenericSAML,
		ProviderGenericOAuth2,
		ProviderGenericOIDC,
	}

	for _, provider := range validProviders {
		assert.NotEmpty(t, string(provider))
	}
}
