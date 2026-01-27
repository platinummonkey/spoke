package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database connection
// Uses in-memory SQLite for testing (would use PostgreSQL in real tests)
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// For now, skip tests that require database
	// In production, this would connect to a test PostgreSQL database
	t.Skip("Database tests require PostgreSQL test instance")
	return nil
}

// TestNewAuthHandlers verifies handler initialization
func TestNewAuthHandlers(t *testing.T) {
	db := &sql.DB{} // Mock DB
	handlers := NewAuthHandlers(db)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.db)
	assert.NotNil(t, handlers.tokenGenerator)
	assert.NotNil(t, handlers.tokenManager)
}

// TestRegisterRoutes verifies all routes are registered
func TestRegisterRoutes(t *testing.T) {
	db := &sql.DB{}
	handlers := NewAuthHandlers(db)
	router := mux.NewRouter()

	handlers.RegisterRoutes(router)

	// Test that routes are registered by checking if they match
	tests := []struct {
		method string
		path   string
	}{
		{"POST", "/auth/users"},
		{"GET", "/auth/users/123"},
		{"PUT", "/auth/users/123"},
		{"DELETE", "/auth/users/123"},
		{"POST", "/auth/tokens"},
		{"GET", "/auth/tokens"},
		{"GET", "/auth/tokens/123"},
		{"DELETE", "/auth/tokens/123"},
		{"POST", "/auth/organizations"},
		{"GET", "/auth/organizations/123"},
		{"GET", "/auth/organizations/123/members"},
		{"POST", "/auth/organizations/123/members"},
		{"DELETE", "/auth/organizations/123/members/456"},
		{"GET", "/auth/modules/mymodule/permissions"},
		{"POST", "/auth/modules/mymodule/permissions"},
		{"DELETE", "/auth/modules/mymodule/permissions/123"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			var match mux.RouteMatch
			matched := router.Match(req, &match)
			assert.True(t, matched, "Route %s %s should be registered", tt.method, tt.path)
		})
	}
}

// TestCreateUser_Success tests successful user creation
func TestCreateUser_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	// This would use a real test database
	t.Skip("Requires PostgreSQL test database")
}

// TestCreateUser_Validation tests user creation validation
func TestCreateUser_Validation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		skipWithMockDB bool
	}{
		{
			name:           "missing username",
			requestBody:    map[string]interface{}{"email": "test@example.com"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "username is required",
		},
		{
			name:           "missing email for non-bot user",
			requestBody:    map[string]interface{}{"username": "testuser", "is_bot": false},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email is required for non-bot users",
		},
		{
			name:           "bot user without email is valid",
			requestBody:    map[string]interface{}{"username": "bot-user", "is_bot": true},
			expectedStatus: http.StatusInternalServerError, // Would succeed with real DB
			skipWithMockDB: true,                           // Skip - requires real database
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipWithMockDB {
				t.Skip("Requires real database")
			}

			// Create mock database that returns error
			db := &sql.DB{}
			handlers := NewAuthHandlers(db)

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/auth/users", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handlers.createUser(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

// TestCreateUser_InvalidJSON tests invalid JSON handling
func TestCreateUser_InvalidJSON(t *testing.T) {
	db := &sql.DB{}
	handlers := NewAuthHandlers(db)

	req := httptest.NewRequest("POST", "/auth/users", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlers.createUser(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid character")
}

// TestCreateToken_Validation tests token creation validation
func TestCreateToken_Validation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing user_id",
			requestBody:    map[string]interface{}{"name": "my-token"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "user_id is required",
		},
		{
			name:           "missing name",
			requestBody:    map[string]interface{}{"user_id": 123},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name is required",
		},
		{
			name: "zero user_id",
			requestBody: map[string]interface{}{
				"user_id": 0,
				"name":    "my-token",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "user_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &sql.DB{}
			handlers := NewAuthHandlers(db)

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/auth/tokens", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handlers.createToken(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedError)
		})
	}
}

// TestCreateToken_InvalidJSON tests invalid JSON handling
func TestCreateToken_InvalidJSON(t *testing.T) {
	db := &sql.DB{}
	handlers := NewAuthHandlers(db)

	req := httptest.NewRequest("POST", "/auth/tokens", bytes.NewReader([]byte("{invalid")))
	w := httptest.NewRecorder()

	handlers.createToken(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestTokenGeneration tests the token generation flow
func TestTokenGeneration(t *testing.T) {
	generator := auth.NewTokenGenerator()

	token, hash, prefix, err := generator.GenerateToken()
	require.NoError(t, err)

	// Verify token format
	assert.True(t, len(token) > 40, "Token should be long enough")
	assert.Contains(t, token, "spoke_", "Token should have spoke_ prefix")

	// Verify hash is 64 hex characters (SHA256)
	assert.Len(t, hash, 64, "Hash should be 64 characters")

	// Verify prefix
	assert.Contains(t, prefix, "spoke_", "Prefix should contain spoke_")

	// Verify token can be validated
	err = generator.ValidateTokenFormat(token)
	assert.NoError(t, err, "Generated token should be valid")

	// Verify hash is deterministic
	hash2 := generator.HashToken(token)
	assert.Equal(t, hash, hash2, "Hashing should be deterministic")
}

// TestTokenValidation tests token format validation
func TestTokenValidation(t *testing.T) {
	generator := auth.NewTokenGenerator()

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "valid token",
			token:       "spoke_abcdefghijklmnopqrstuvwxyz123456",
			expectError: false,
		},
		{
			name:        "missing prefix",
			token:       "abcdefghijklmnopqrstuvwxyz123456",
			expectError: true,
		},
		{
			name:        "wrong prefix",
			token:       "github_abcdefghijklmnopqrstuvwxyz123456",
			expectError: true,
		},
		{
			name:        "only prefix",
			token:       "spoke_",
			expectError: true,
		},
		{
			name:        "invalid base64",
			token:       "spoke_!!!invalid!!!",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.ValidateTokenFormat(tt.token)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTokenPrefixExtraction tests prefix extraction
func TestTokenPrefixExtraction(t *testing.T) {
	generator := auth.NewTokenGenerator()

	tests := []struct {
		name           string
		token          string
		expectedPrefix string
	}{
		{
			name:           "normal token",
			token:          "spoke_abcdefghijklmnop",
			expectedPrefix: "spoke_abcdefgh",
		},
		{
			name:           "short token",
			token:          "spoke_abc",
			expectedPrefix: "spoke_abc",
		},
		{
			name:           "no prefix",
			token:          "abcdefgh",
			expectedPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := generator.ExtractPrefix(tt.token)
			assert.Equal(t, tt.expectedPrefix, prefix)
		})
	}
}

// TestUpdateUser_Validation tests update validation
func TestUpdateUser_Validation(t *testing.T) {
	db := &sql.DB{}
	handlers := NewAuthHandlers(db)

	// Test invalid JSON
	req := httptest.NewRequest("PUT", "/auth/users/123", bytes.NewReader([]byte("{invalid")))
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.updateUser(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdateUser_EmptyUpdate tests empty update request
func TestUpdateUser_EmptyUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
}

// TestDeleteUser_SoftDelete verifies soft delete behavior
func TestDeleteUser_SoftDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
	// Would verify that user is marked inactive, not actually deleted
}

// TestGetUser_NotFound tests user not found error
func TestGetUser_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
}

// TestRevokeToken tests token revocation
func TestRevokeToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
}

// TestListTokens tests token listing
func TestListTokens(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
}

// TestCreateOrganization_Validation tests organization creation validation
func TestCreateOrganization_Validation(t *testing.T) {
	db := &sql.DB{}
	handlers := NewAuthHandlers(db)

	// Test invalid JSON
	req := httptest.NewRequest("POST", "/auth/organizations", bytes.NewReader([]byte("bad json")))
	w := httptest.NewRecorder()

	handlers.createOrganization(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAddOrganizationMember_Validation tests member addition validation
func TestAddOrganizationMember_Validation(t *testing.T) {
	db := &sql.DB{}
	handlers := NewAuthHandlers(db)

	// Test invalid JSON
	req := httptest.NewRequest("POST", "/auth/organizations/123/members", bytes.NewReader([]byte("{")))
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.addOrganizationMember(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGrantModulePermission_Validation tests permission grant validation
func TestGrantModulePermission_Validation(t *testing.T) {
	db := &sql.DB{}
	handlers := NewAuthHandlers(db)

	// Test invalid JSON
	req := httptest.NewRequest("POST", "/auth/modules/test/permissions", bytes.NewReader([]byte("invalid")))
	req = mux.SetURLVars(req, map[string]string{"module_name": "test"})
	w := httptest.NewRecorder()

	handlers.grantModulePermission(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestTokenExpiration tests token expiration handling
func TestTokenExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
	// Would test that expired tokens are rejected
}

// TestTokenScopes tests token scope handling
func TestTokenScopes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
	// Would test that token scopes are correctly stored and retrieved
}

// TestConcurrentTokenGeneration tests token generation under concurrency
func TestConcurrentTokenGeneration(t *testing.T) {
	generator := auth.NewTokenGenerator()
	numTokens := 100

	tokens := make(chan string, numTokens)
	hashes := make(chan string, numTokens)

	// Generate tokens concurrently
	for i := 0; i < numTokens; i++ {
		go func() {
			token, hash, _, err := generator.GenerateToken()
			if err == nil {
				tokens <- token
				hashes <- hash
			}
		}()
	}

	// Collect tokens
	tokenSet := make(map[string]bool)
	hashSet := make(map[string]bool)

	for i := 0; i < numTokens; i++ {
		token := <-tokens
		hash := <-hashes
		tokenSet[token] = true
		hashSet[hash] = true
	}

	// Verify uniqueness
	assert.Len(t, tokenSet, numTokens, "All tokens should be unique")
	assert.Len(t, hashSet, numTokens, "All hashes should be unique")
}

// TestTokenHashConsistency tests that hashing is consistent
func TestTokenHashConsistency(t *testing.T) {
	generator := auth.NewTokenGenerator()
	token := "spoke_testtoken12345678901234567890"

	// Hash multiple times
	hash1 := generator.HashToken(token)
	hash2 := generator.HashToken(token)
	hash3 := generator.HashToken(token)

	assert.Equal(t, hash1, hash2)
	assert.Equal(t, hash2, hash3)
	assert.Len(t, hash1, 64) // SHA256 produces 64 hex characters
}

// TestTokenHashDifferentiation tests that different tokens have different hashes
func TestTokenHashDifferentiation(t *testing.T) {
	generator := auth.NewTokenGenerator()

	token1 := "spoke_token1"
	token2 := "spoke_token2"

	hash1 := generator.HashToken(token1)
	hash2 := generator.HashToken(token2)

	assert.NotEqual(t, hash1, hash2, "Different tokens should have different hashes")
}

// Benchmark token generation performance
func BenchmarkTokenGeneration(b *testing.B) {
	generator := auth.NewTokenGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, err := generator.GenerateToken()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark token hashing performance
func BenchmarkTokenHashing(b *testing.B) {
	generator := auth.NewTokenGenerator()
	token := "spoke_abcdefghijklmnopqrstuvwxyz123456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.HashToken(token)
	}
}

// Benchmark token validation performance
func BenchmarkTokenValidation(b *testing.B) {
	generator := auth.NewTokenGenerator()
	token := "spoke_abcdefghijklmnopqrstuvwxyz123456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.ValidateTokenFormat(token)
	}
}
