package marketplace

import (
	"context"
	"io"
)

// Storage defines the interface for storing plugin artifacts
type Storage interface {
	// StorePluginArchive stores a plugin archive and returns the download URL
	StorePluginArchive(ctx context.Context, pluginID, version string, data io.Reader) (string, error)

	// GetPluginArchive retrieves a plugin archive
	GetPluginArchive(ctx context.Context, pluginID, version string) (io.ReadCloser, error)

	// DeletePluginArchive deletes a plugin archive
	DeletePluginArchive(ctx context.Context, pluginID, version string) error

	// StorePluginManifest stores a plugin manifest and returns the manifest URL
	StorePluginManifest(ctx context.Context, pluginID, version string, data []byte) (string, error)

	// GetPluginManifest retrieves a plugin manifest
	GetPluginManifest(ctx context.Context, pluginID, version string) ([]byte, error)

	// ListPluginVersions lists all versions of a plugin
	ListPluginVersions(ctx context.Context, pluginID string) ([]string, error)

	// GetArchiveChecksum calculates SHA-256 checksum of an archive
	GetArchiveChecksum(data io.Reader) (string, error)
}

// FileSystemStorage implements Storage using the local filesystem
type FileSystemStorage struct {
	baseDir    string
	baseURL    string // Base URL for download links
}

// NewFileSystemStorage creates a new filesystem-based storage
func NewFileSystemStorage(baseDir, baseURL string) (*FileSystemStorage, error) {
	return &FileSystemStorage{
		baseDir: baseDir,
		baseURL: baseURL,
	}, nil
}

// S3Storage implements Storage using AWS S3
type S3Storage struct {
	bucket string
	region string
	prefix string
}

// NewS3Storage creates a new S3-based storage
func NewS3Storage(bucket, region, prefix string) (*S3Storage, error) {
	return &S3Storage{
		bucket: bucket,
		region: region,
		prefix: prefix,
	}, nil
}
