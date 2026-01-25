package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	// TokenPrefix identifies Spoke tokens
	TokenPrefix = "spoke_"
	// TokenLength is the total length of random bytes (32 bytes = 256 bits)
	TokenLength = 32
)

// TokenGenerator generates and validates API tokens
type TokenGenerator struct{}

// NewTokenGenerator creates a new token generator
func NewTokenGenerator() *TokenGenerator {
	return &TokenGenerator{}
}

// GenerateToken creates a new API token
// Format: spoke_<base64url(32 random bytes)>
// Example: spoke_abc123def456...
func (tg *TokenGenerator) GenerateToken() (token string, tokenHash string, tokenPrefix string, err error) {
	// Generate random bytes
	randomBytes := make([]byte, TokenLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base64url (URL-safe, no padding)
	encodedToken := base64.RawURLEncoding.EncodeToString(randomBytes)

	// Construct full token
	fullToken := TokenPrefix + encodedToken

	// Calculate SHA256 hash for storage
	hash := sha256.Sum256([]byte(fullToken))
	hashStr := hex.EncodeToString(hash[:])

	// Extract prefix (first 8 chars after "spoke_") for identification
	prefix := TokenPrefix
	if len(encodedToken) >= 8 {
		prefix = TokenPrefix + encodedToken[:8]
	}

	return fullToken, hashStr, prefix, nil
}

// HashToken computes the SHA256 hash of a token for lookup
func (tg *TokenGenerator) HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// ValidateTokenFormat checks if a token has the correct format
func (tg *TokenGenerator) ValidateTokenFormat(token string) error {
	if !strings.HasPrefix(token, TokenPrefix) {
		return fmt.Errorf("token must start with %q", TokenPrefix)
	}

	encodedPart := strings.TrimPrefix(token, TokenPrefix)
	if len(encodedPart) == 0 {
		return fmt.Errorf("token is too short")
	}

	// Decode to verify it's valid base64url
	_, err := base64.RawURLEncoding.DecodeString(encodedPart)
	if err != nil {
		return fmt.Errorf("invalid token encoding: %w", err)
	}

	return nil
}

// ExtractPrefix extracts the prefix from a token for display
func (tg *TokenGenerator) ExtractPrefix(token string) string {
	if !strings.HasPrefix(token, TokenPrefix) {
		return ""
	}

	encodedPart := strings.TrimPrefix(token, TokenPrefix)
	if len(encodedPart) >= 8 {
		return TokenPrefix + encodedPart[:8]
	}

	return token
}

// TokenManager manages API token lifecycle
type TokenManager struct {
	generator *TokenGenerator
	// TODO: Add database connection for storage
}

// NewTokenManager creates a new token manager
func NewTokenManager() *TokenManager {
	return &TokenManager{
		generator: NewTokenGenerator(),
	}
}

// CreateToken creates a new API token
func (tm *TokenManager) CreateToken(userID int64, name, description string, scopes []Scope, expiresAt *time.Time) (*APIToken, string, error) {
	// Generate token
	token, tokenHash, tokenPrefix, err := tm.generator.GenerateToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Create token record
	apiToken := &APIToken{
		UserID:      userID,
		TokenHash:   tokenHash,
		TokenPrefix: tokenPrefix,
		Name:        name,
		Description: description,
		Scopes:      scopes,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}

	// TODO: Store in database

	// Return the token ONCE (never stored in plaintext)
	return apiToken, token, nil
}

// ValidateToken validates a token and returns the associated user
func (tm *TokenManager) ValidateToken(token string) (*APIToken, error) {
	// Validate format
	if err := tm.generator.ValidateTokenFormat(token); err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	// Hash the token for lookup
	tokenHash := tm.generator.HashToken(token)

	// TODO: Look up in database by tokenHash
	// - Check if token exists
	// - Check if not revoked (revoked_at IS NULL)
	// - Check if not expired (expires_at IS NULL OR expires_at > NOW())
	// - Update last_used_at

	_ = tokenHash
	return nil, fmt.Errorf("not implemented: token lookup")
}

// RevokeToken revokes a token
func (tm *TokenManager) RevokeToken(tokenID int64, revokedBy int64, reason string) error {
	// TODO: Update database
	// - Set revoked_at = NOW()
	// - Set revoked_by = revokedBy
	// - Set revoke_reason = reason

	_ = tokenID
	_ = revokedBy
	_ = reason
	return fmt.Errorf("not implemented: token revocation")
}

// ListUserTokens lists all tokens for a user
func (tm *TokenManager) ListUserTokens(userID int64) ([]*APIToken, error) {
	// TODO: Query database for user's tokens
	// - Include revoked tokens with revoked_at
	// - Order by created_at DESC

	_ = userID
	return nil, fmt.Errorf("not implemented: token listing")
}

// CleanupExpiredTokens removes expired tokens
func (tm *TokenManager) CleanupExpiredTokens() (int, error) {
	// TODO: Delete or mark as revoked
	// - WHERE expires_at < NOW() AND revoked_at IS NULL

	return 0, fmt.Errorf("not implemented: token cleanup")
}
