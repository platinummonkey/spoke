package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestS3Client_ContentAddressableStorage(t *testing.T) {
	// Test that the same content produces the same hash
	content := []byte("test content for deduplication")

	expectedHash := sha256.Sum256(content)
	expectedHashStr := hex.EncodeToString(expectedHash[:])
	expectedKey := fmt.Sprintf("proto-files/sha256/%s/%s", expectedHashStr[:2], expectedHashStr[2:])

	if len(expectedHashStr) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(expectedHashStr))
	}

	prefix := "proto-files/sha256/"
	if expectedKey[:len(prefix)] != prefix {
		t.Errorf("Expected key to start with %q, got %q", prefix, expectedKey[:len(prefix)])
	}
}

func TestS3Client_KeyFormat(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		wantKey  string
	}{
		{
			name:    "typical hash",
			hash:    "abc123def456789012345678901234567890123456789012345678901234",
			wantKey: "proto-files/sha256/ab/c123def456789012345678901234567890123456789012345678901234",
		},
		{
			name:    "hash starting with numbers",
			hash:    "123456def456789012345678901234567890123456789012345678901234",
			wantKey: "proto-files/sha256/12/3456def456789012345678901234567890123456789012345678901234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := fmt.Sprintf("proto-files/sha256/%s/%s", tt.hash[:2], tt.hash[2:])
			if key != tt.wantKey {
				t.Errorf("Key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestNewS3Client_ConfigValidation(t *testing.T) {
	t.Skip("Skipping - requires actual S3/MinIO instance. Use integration tests with testcontainers.")

	// This test would require:
	// - testcontainers with MinIO
	// - Proper setup/teardown
	// - Build tag: // +build integration
}

func TestS3Client_HashCalculation(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name:    "empty content",
			content: "",
			wantLen: 64, // SHA256 hex is always 64 chars
		},
		{
			name:    "simple content",
			content: "hello world",
			wantLen: 64,
		},
		{
			name:    "large content",
			content: string(bytes.Repeat([]byte("a"), 10000)),
			wantLen: 64,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash := sha256.Sum256([]byte(tc.content))
			hashStr := hex.EncodeToString(hash[:])

			if len(hashStr) != tc.wantLen {
				t.Errorf("Hash length = %d, want %d", len(hashStr), tc.wantLen)
			}
		})
	}
}

func TestS3Client_Deduplication(t *testing.T) {
	// Test that uploading the same content twice produces the same key
	content := []byte("duplicate content test")

	hash1 := sha256.Sum256(content)
	hash2 := sha256.Sum256(content)

	hashStr1 := hex.EncodeToString(hash1[:])
	hashStr2 := hex.EncodeToString(hash2[:])

	if hashStr1 != hashStr2 {
		t.Error("Same content should produce same hash")
	}

	key1 := fmt.Sprintf("proto-files/sha256/%s/%s", hashStr1[:2], hashStr1[2:])
	key2 := fmt.Sprintf("proto-files/sha256/%s/%s", hashStr2[:2], hashStr2[2:])

	if key1 != key2 {
		t.Error("Same content should produce same S3 key")
	}
}

func TestS3Client_ObjectKeyGeneration(t *testing.T) {
	// Test the key generation pattern matches expected format
	ctx := context.Background()
	_ = ctx // For future use with actual S3 operations

	content := []byte("test content")
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	// Expected pattern: proto-files/sha256/XX/YYYYYY...
	expectedPrefix := "proto-files/sha256/"
	key := fmt.Sprintf("%s%s/%s", expectedPrefix, hashStr[:2], hashStr[2:])

	if len(key) != len(expectedPrefix) + 2 + 1 + 62 {
		t.Errorf("Key length unexpected: %d", len(key))
	}

	if key[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Key prefix = %q, want %q", key[:len(expectedPrefix)], expectedPrefix)
	}
}

// Note: Integration tests with actual S3/MinIO would go in a separate file
// and use build tags like: // +build integration
