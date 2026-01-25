package artifacts

import (
	"context"
	"io"

	"github.com/platinummonkey/spoke/pkg/codegen"
)

// Manager handles storage and retrieval of compiled artifacts
type Manager interface {
	// Store uploads compiled artifacts to S3
	Store(ctx context.Context, req *StoreRequest) (*StoreResult, error)

	// Retrieve downloads compiled artifacts from S3
	Retrieve(ctx context.Context, req *RetrieveRequest) (*RetrieveResult, error)

	// Delete removes compiled artifacts from S3
	Delete(ctx context.Context, moduleName, version, language string) error

	// Exists checks if artifacts exist
	Exists(ctx context.Context, moduleName, version, language string) (bool, error)

	// GetURL returns a presigned URL for downloading artifacts
	GetURL(ctx context.Context, moduleName, version, language string, ttl int) (string, error)

	// Close releases resources
	Close() error
}

// StoreRequest represents a request to store compiled artifacts
type StoreRequest struct {
	ModuleName     string
	Version        string
	Language       string
	Files          []codegen.GeneratedFile
	Metadata       map[string]string
	CompressionFormat string // "zip", "tar.gz", or "none"
}

// StoreResult represents the result of storing artifacts
type StoreResult struct {
	S3Key          string
	S3Bucket       string
	Hash           string
	Size           int64
	CompressedSize int64
}

// RetrieveRequest represents a request to retrieve compiled artifacts
type RetrieveRequest struct {
	ModuleName     string
	Version        string
	Language       string
	ExtractTo      string // Directory to extract files to
}

// RetrieveResult represents the result of retrieving artifacts
type RetrieveResult struct {
	Files          []codegen.GeneratedFile
	Metadata       map[string]string
	Hash           string
	Size           int64
}

// Config holds artifact manager configuration
type Config struct {
	S3Bucket           string
	S3Prefix           string
	S3Region           string
	CompressionFormat  string // Default compression format
	EnableChecksum     bool   // Verify checksums on download
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		S3Prefix:          "compiled/",
		CompressionFormat: "tar.gz",
		EnableChecksum:    true,
	}
}

// Writer writes generated files to a storage backend
type Writer interface {
	io.Writer
	Close() error
}

// Reader reads generated files from a storage backend
type Reader interface {
	io.Reader
	Close() error
}
