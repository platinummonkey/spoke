//go:build integration

package postgres

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupMinIO creates a MinIO testcontainer and returns an S3Client configured to use it
func setupMinIO(t *testing.T) (*S3Client, func()) {
	t.Helper()

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000/tcp"),
	}

	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start MinIO container")

	// Get the host and port
	host, err := minioContainer.Host(ctx)
	require.NoError(t, err)

	port, err := minioContainer.MappedPort(ctx, "9000")
	require.NoError(t, err)

	endpoint := "http://" + host + ":" + port.Port()

	// Configure S3 client for MinIO
	cfg := storage.Config{
		S3Endpoint:     endpoint,
		S3AccessKey:    "minioadmin",
		S3SecretKey:    "minioadmin",
		S3Bucket:       "test-bucket",
		S3Region:       "us-east-1",
		S3UsePathStyle: true,
	}

	client, err := NewS3Client(cfg)
	require.NoError(t, err, "Failed to create S3 client")

	cleanup := func() {
		// S3Client doesn't have a Close method - AWS SDK handles cleanup
		err := minioContainer.Terminate(ctx)
		if err != nil {
			t.Logf("Warning: Failed to terminate MinIO container: %v", err)
		}
	}

	return client, cleanup
}

// TestS3Client_PutObject_Integration tests PutObject with MinIO
func TestS3Client_PutObject_Integration(t *testing.T) {
	client, cleanup := setupMinIO(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name        string
		key         string
		content     string
		contentType string
		wantErr     bool
	}{
		{
			name:        "simple text file",
			key:         "test.txt",
			content:     "Hello, World!",
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name:        "empty file",
			key:         "empty.txt",
			content:     "",
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name:        "binary content",
			key:         "binary.bin",
			content:     string([]byte{0x00, 0x01, 0x02, 0xFF}),
			contentType: "application/octet-stream",
			wantErr:     false,
		},
		{
			name:        "large file",
			key:         "large.txt",
			content:     strings.Repeat("a", 1024*1024), // 1MB
			contentType: "text/plain",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.PutObject(ctx, tt.key, strings.NewReader(tt.content), tt.contentType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestS3Client_GetObject_Integration tests GetObject with MinIO
func TestS3Client_GetObject_Integration(t *testing.T) {
	client, cleanup := setupMinIO(t)
	defer cleanup()

	ctx := context.Background()

	// Put an object first
	testContent := "Test content for retrieval"
	err := client.PutObject(ctx, "test-get.txt", strings.NewReader(testContent), "text/plain")
	require.NoError(t, err)

	t.Run("get existing object", func(t *testing.T) {
		reader, err := client.GetObject(ctx, "test-get.txt")
		require.NoError(t, err)
		defer reader.Close()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, testContent, string(data))
	})

	t.Run("get non-existent object", func(t *testing.T) {
		_, err := client.GetObject(ctx, "does-not-exist.txt")
		assert.Error(t, err)
	})
}

// TestS3Client_PutObjectWithHash_Integration tests deduplication
func TestS3Client_PutObjectWithHash_Integration(t *testing.T) {
	client, cleanup := setupMinIO(t)
	defer cleanup()

	ctx := context.Background()

	content := []byte("Test content for hashing")

	// Upload the object (first time)
	hash, err := client.PutObjectWithHash(ctx, content, "text/plain")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Upload the same content again (should deduplicate)
	hash2, err := client.PutObjectWithHash(ctx, content, "text/plain")
	require.NoError(t, err)
	assert.Equal(t, hash, hash2, "Hash should be the same for identical content")

	// Verify the object exists in S3 with the content-addressable key
	key := "proto-files/sha256/" + hash[:2] + "/" + hash[2:]
	exists, err := client.ObjectExists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists, "Object should exist at content-addressable path")
}

// TestS3Client_ObjectExists_Integration tests existence checks
func TestS3Client_ObjectExists_Integration(t *testing.T) {
	client, cleanup := setupMinIO(t)
	defer cleanup()

	ctx := context.Background()

	// Put an object
	err := client.PutObject(ctx, "exists-test.txt", strings.NewReader("content"), "text/plain")
	require.NoError(t, err)

	t.Run("existing object", func(t *testing.T) {
		exists, err := client.ObjectExists(ctx, "exists-test.txt")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existent object", func(t *testing.T) {
		exists, err := client.ObjectExists(ctx, "does-not-exist.txt")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

// TestS3Client_DeleteObject_Integration tests deletion
func TestS3Client_DeleteObject_Integration(t *testing.T) {
	client, cleanup := setupMinIO(t)
	defer cleanup()

	ctx := context.Background()

	// Put an object
	err := client.PutObject(ctx, "delete-test.txt", strings.NewReader("content"), "text/plain")
	require.NoError(t, err)

	t.Run("delete existing object", func(t *testing.T) {
		err := client.DeleteObject(ctx, "delete-test.txt")
		assert.NoError(t, err)

		// Verify it's gone
		exists, err := client.ObjectExists(ctx, "delete-test.txt")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("delete non-existent object (idempotent)", func(t *testing.T) {
		err := client.DeleteObject(ctx, "does-not-exist.txt")
		assert.NoError(t, err, "Deleting non-existent object should not error")
	})
}

// TestS3Client_HealthCheck_Integration tests health checks
func TestS3Client_HealthCheck_Integration(t *testing.T) {
	client, cleanup := setupMinIO(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.HealthCheck(ctx)
	assert.NoError(t, err, "Health check should pass with healthy MinIO")
}

// Note: createBucketIfNotExists is tested implicitly via NewS3Client in setupMinIO
// The function is private and called during client creation, so it's exercised by all tests
