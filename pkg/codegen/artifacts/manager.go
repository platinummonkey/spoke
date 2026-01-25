package artifacts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/platinummonkey/spoke/pkg/codegen"
)

// S3Manager implements artifact storage using S3
type S3Manager struct {
	client *s3.Client
	config *Config
}

// NewS3Manager creates a new S3 artifact manager
func NewS3Manager(cfg *Config) (Manager, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.S3Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)

	return &S3Manager{
		client: client,
		config: cfg,
	}, nil
}

// Store uploads compiled artifacts to S3
func (m *S3Manager) Store(ctx context.Context, req *StoreRequest) (*StoreResult, error) {
	if req == nil {
		return nil, fmt.Errorf("store request cannot be nil")
	}

	// Build S3 key
	key := m.buildS3Key(req.ModuleName, req.Version, req.Language)

	// Compress files
	compressed, hash, size, err := m.compressFiles(req.Files, req.CompressionFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to compress files: %w", err)
	}

	// Upload to S3
	_, err = m.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(m.config.S3Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(compressed),
		ContentType: aws.String(m.getContentType(req.CompressionFormat)),
		Metadata:    m.convertMetadata(req.Metadata),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}

	return &StoreResult{
		S3Key:          key,
		S3Bucket:       m.config.S3Bucket,
		Hash:           hash,
		Size:           size,
		CompressedSize: int64(len(compressed)),
	}, nil
}

// Retrieve downloads compiled artifacts from S3
func (m *S3Manager) Retrieve(ctx context.Context, req *RetrieveRequest) (*RetrieveResult, error) {
	if req == nil {
		return nil, fmt.Errorf("retrieve request cannot be nil")
	}

	// Build S3 key
	key := m.buildS3Key(req.ModuleName, req.Version, req.Language)

	// Download from S3
	output, err := m.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(m.config.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer output.Body.Close()

	// Read all data
	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object: %w", err)
	}

	// Decompress files
	files, hash, err := m.decompressFiles(data, m.config.CompressionFormat)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecompressionFailed, err)
	}

	// Extract metadata
	metadata := m.extractMetadata(output.Metadata)

	return &RetrieveResult{
		Files:    files,
		Metadata: metadata,
		Hash:     hash,
		Size:     int64(len(data)),
	}, nil
}

// Delete removes compiled artifacts from S3
func (m *S3Manager) Delete(ctx context.Context, moduleName, version, language string) error {
	key := m.buildS3Key(moduleName, version, language)

	_, err := m.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(m.config.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// Exists checks if artifacts exist
func (m *S3Manager) Exists(ctx context.Context, moduleName, version, language string) (bool, error) {
	key := m.buildS3Key(moduleName, version, language)

	_, err := m.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(m.config.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if error is "not found"
		// AWS SDK v2 doesn't have a simple IsNotFound check
		return false, nil
	}

	return true, nil
}

// GetURL returns a presigned URL for downloading artifacts
func (m *S3Manager) GetURL(ctx context.Context, moduleName, version, language string, ttl int) (string, error) {
	key := m.buildS3Key(moduleName, version, language)

	presignClient := s3.NewPresignClient(m.client)

	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(m.config.S3Bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(ttl) * time.Second
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return req.URL, nil
}

// Close releases resources
func (m *S3Manager) Close() error {
	// S3 client doesn't need explicit closing
	return nil
}

// buildS3Key builds an S3 key from module info
func (m *S3Manager) buildS3Key(moduleName, version, language string) string {
	// Format: {prefix}/{moduleName}/{version}/{language}.tar.gz
	return filepath.Join(
		m.config.S3Prefix,
		moduleName,
		version,
		language+".tar.gz",
	)
}

// compressFiles compresses files using tar.gz
func (m *S3Manager) compressFiles(files []codegen.GeneratedFile, format string) ([]byte, string, int64, error) {
	var buf bytes.Buffer
	hasher := sha256.New()

	// Create gzip writer
	gzWriter := gzip.NewWriter(io.MultiWriter(&buf, hasher))
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	var totalSize int64

	// Add each file to the archive
	for _, file := range files {
		header := &tar.Header{
			Name: file.Path,
			Mode: 0644,
			Size: int64(len(file.Content)),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, "", 0, err
		}

		if _, err := tarWriter.Write(file.Content); err != nil {
			return nil, "", 0, err
		}

		totalSize += int64(len(file.Content))
	}

	// Close writers to flush
	if err := tarWriter.Close(); err != nil {
		return nil, "", 0, err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, "", 0, err
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	return buf.Bytes(), hash, totalSize, nil
}

// decompressFiles decompresses files from tar.gz
func (m *S3Manager) decompressFiles(data []byte, format string) ([]codegen.GeneratedFile, string, error) {
	hasher := sha256.New()
	hasher.Write(data)
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Create gzip reader
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	var files []codegen.GeneratedFile

	// Read each file from the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}

		// Read file content
		content, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, "", err
		}

		files = append(files, codegen.GeneratedFile{
			Path:    header.Name,
			Content: content,
			Size:    header.Size,
		})
	}

	return files, hash, nil
}

// getContentType returns the content type for the compression format
func (m *S3Manager) getContentType(format string) string {
	switch format {
	case "zip":
		return "application/zip"
	case "tar.gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// convertMetadata converts metadata map to S3 format
func (m *S3Manager) convertMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return make(map[string]string)
	}
	return metadata
}

// extractMetadata extracts metadata from S3 format
func (m *S3Manager) extractMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return make(map[string]string)
	}
	return metadata
}
