package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/platinummonkey/spoke/pkg/storage"
)

// S3Client handles object storage operations
type S3Client struct {
	client *s3.Client
	bucket string
	config storage.Config
}

// NewS3Client creates a new S3 client
func NewS3Client(cfg storage.Config) (*S3Client, error) {
	ctx := context.Background()

	// Configure AWS SDK
	var awsConfig aws.Config
	var err error

	if cfg.S3AccessKey != "" && cfg.S3SecretKey != "" {
		// Use static credentials (for MinIO or AWS with explicit keys)
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.S3Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.S3AccessKey,
				cfg.S3SecretKey,
				"",
			)),
		)
	} else {
		// Use default credential chain (IAM roles, env vars, etc.)
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.S3Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		}
		if cfg.S3UsePathStyle {
			o.UsePathStyle = true
		}
	})

	// Create bucket if it doesn't exist (for local dev with MinIO)
	if err := createBucketIfNotExists(ctx, s3Client, cfg.S3Bucket, cfg.S3Region); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return &S3Client{
		client: s3Client,
		bucket: cfg.S3Bucket,
		config: cfg,
	}, nil
}

// PutObject uploads content to S3
func (c *S3Client) PutObject(ctx context.Context, key string, content io.Reader, contentType string) error {
	// Read content to calculate hash and size
	data, err := io.ReadAll(content)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	// Calculate SHA256 checksum
	hash := sha256.Sum256(data)
	checksum := hex.EncodeToString(hash[:])

	// Upload to S3
	_, err = c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"checksum-sha256": checksum,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upload to s3: %w", err)
	}

	return nil
}

// GetObject retrieves content from S3
func (c *S3Client) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get object from s3: %w", err)
	}

	return result.Body, nil
}

// ObjectExists checks if an object exists
func (c *S3Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if it's a "not found" error
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// DeleteObject deletes an object from S3
func (c *S3Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// HealthCheck verifies S3 connectivity
func (c *S3Client) HealthCheck(ctx context.Context) error {
	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})

	if err != nil {
		return fmt.Errorf("s3 health check failed: %w", err)
	}

	return nil
}

// PutObjectWithHash uploads content with a given hash as key
func (c *S3Client) PutObjectWithHash(ctx context.Context, content []byte, contentType string) (string, error) {
	// Calculate SHA256 hash
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	// Use content-addressable storage: sha256/ab/cd123...
	key := fmt.Sprintf("proto-files/sha256/%s/%s", hashStr[:2], hashStr[2:])

	// Check if already exists (deduplication)
	exists, err := c.ObjectExists(ctx, key)
	if err != nil {
		return "", err
	}

	if !exists {
		// Upload to S3
		if err := c.PutObject(ctx, key, bytes.NewReader(content), contentType); err != nil {
			return "", err
		}
	}

	return hashStr, nil
}

// Helper functions

func createBucketIfNotExists(ctx context.Context, client *s3.Client, bucket, region string) error {
	// Check if bucket exists
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if err == nil {
		// Bucket exists
		return nil
	}

	// Bucket doesn't exist, create it
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		// Ignore error if bucket already exists (race condition)
		if !isBucketAlreadyExistsError(err) {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

func isNotFoundError(err error) bool {
	// Check if error indicates object not found
	// This is a simplified check - in production, should check specific error types
	return err != nil && (containsString(err.Error(), "NotFound") || containsString(err.Error(), "NoSuchKey"))
}

func isBucketAlreadyExistsError(err error) bool {
	return err != nil && (containsString(err.Error(), "BucketAlreadyExists") || containsString(err.Error(), "BucketAlreadyOwnedByYou"))
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
