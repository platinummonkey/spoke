package artifacts

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.S3Prefix != "compiled/" {
		t.Errorf("expected S3Prefix='compiled/', got %s", cfg.S3Prefix)
	}

	if cfg.CompressionFormat != "tar.gz" {
		t.Errorf("expected CompressionFormat='tar.gz', got %s", cfg.CompressionFormat)
	}

	if !cfg.EnableChecksum {
		t.Error("expected EnableChecksum=true, got false")
	}

	// Ensure S3Bucket is empty by default (must be configured)
	if cfg.S3Bucket != "" {
		t.Errorf("expected S3Bucket='', got %s", cfg.S3Bucket)
	}

	// Ensure S3Region is empty by default (must be configured)
	if cfg.S3Region != "" {
		t.Errorf("expected S3Region='', got %s", cfg.S3Region)
	}
}

func TestStoreRequest(t *testing.T) {
	req := &StoreRequest{
		ModuleName:        "test-module",
		Version:           "1.0.0",
		Language:          "go",
		CompressionFormat: "tar.gz",
	}

	if req.ModuleName != "test-module" {
		t.Errorf("expected ModuleName='test-module', got %s", req.ModuleName)
	}

	if req.Version != "1.0.0" {
		t.Errorf("expected Version='1.0.0', got %s", req.Version)
	}

	if req.Language != "go" {
		t.Errorf("expected Language='go', got %s", req.Language)
	}

	if req.CompressionFormat != "tar.gz" {
		t.Errorf("expected CompressionFormat='tar.gz', got %s", req.CompressionFormat)
	}
}

func TestStoreResult(t *testing.T) {
	result := &StoreResult{
		S3Key:          "compiled/test-module/1.0.0/go/artifacts.tar.gz",
		S3Bucket:       "my-bucket",
		Hash:           "abc123",
		Size:           1024,
		CompressedSize: 512,
	}

	if result.S3Key != "compiled/test-module/1.0.0/go/artifacts.tar.gz" {
		t.Errorf("unexpected S3Key: %s", result.S3Key)
	}

	if result.Size != 1024 {
		t.Errorf("expected Size=1024, got %d", result.Size)
	}

	if result.CompressedSize != 512 {
		t.Errorf("expected CompressedSize=512, got %d", result.CompressedSize)
	}
}

func TestRetrieveRequest(t *testing.T) {
	req := &RetrieveRequest{
		ModuleName: "test-module",
		Version:    "1.0.0",
		Language:   "go",
		ExtractTo:  "/tmp/artifacts",
	}

	if req.ModuleName != "test-module" {
		t.Errorf("expected ModuleName='test-module', got %s", req.ModuleName)
	}

	if req.ExtractTo != "/tmp/artifacts" {
		t.Errorf("expected ExtractTo='/tmp/artifacts', got %s", req.ExtractTo)
	}
}

func TestRetrieveResult(t *testing.T) {
	result := &RetrieveResult{
		Hash: "abc123",
		Size: 1024,
	}

	if result.Hash != "abc123" {
		t.Errorf("expected Hash='abc123', got %s", result.Hash)
	}

	if result.Size != 1024 {
		t.Errorf("expected Size=1024, got %d", result.Size)
	}
}

func TestConfig(t *testing.T) {
	cfg := &Config{
		S3Bucket:          "my-bucket",
		S3Prefix:          "artifacts/",
		S3Region:          "us-west-2",
		CompressionFormat: "zip",
		EnableChecksum:    false,
	}

	if cfg.S3Bucket != "my-bucket" {
		t.Errorf("expected S3Bucket='my-bucket', got %s", cfg.S3Bucket)
	}

	if cfg.S3Prefix != "artifacts/" {
		t.Errorf("expected S3Prefix='artifacts/', got %s", cfg.S3Prefix)
	}

	if cfg.S3Region != "us-west-2" {
		t.Errorf("expected S3Region='us-west-2', got %s", cfg.S3Region)
	}

	if cfg.CompressionFormat != "zip" {
		t.Errorf("expected CompressionFormat='zip', got %s", cfg.CompressionFormat)
	}

	if cfg.EnableChecksum {
		t.Error("expected EnableChecksum=false, got true")
	}
}
