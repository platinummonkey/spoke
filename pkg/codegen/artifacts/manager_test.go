package artifacts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/platinummonkey/spoke/pkg/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// s3ClientAPI defines the S3 operations we need
type s3ClientAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

// mockS3Client is a mock S3 client for testing
type mockS3Client struct {
	putObjectFunc    func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	getObjectFunc    func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	deleteObjectFunc func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	headObjectFunc   func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getObjectFunc != nil {
		return m.getObjectFunc(ctx, params, optFns...)
	}
	return &s3.GetObjectOutput{}, nil
}

func (m *mockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.deleteObjectFunc != nil {
		return m.deleteObjectFunc(ctx, params, optFns...)
	}
	return &s3.DeleteObjectOutput{}, nil
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headObjectFunc != nil {
		return m.headObjectFunc(ctx, params, optFns...)
	}
	return &s3.HeadObjectOutput{}, nil
}

// testS3Manager extends S3Manager for testing
type testS3Manager struct {
	*S3Manager
	mockClient s3ClientAPI
}

// Helper function to create a test S3Manager with a mock client
func newTestS3Manager(client s3ClientAPI, cfg *Config) *testS3Manager {
	if cfg == nil {
		cfg = DefaultConfig()
		cfg.S3Bucket = "test-bucket"
		cfg.S3Region = "us-west-2"
	}
	return &testS3Manager{
		S3Manager: &S3Manager{
			config: cfg,
		},
		mockClient: client,
	}
}

// Override methods to use mock client
func (m *testS3Manager) Store(ctx context.Context, req *StoreRequest) (*StoreResult, error) {
	if req == nil {
		return nil, errors.New("store request cannot be nil")
	}

	key := m.S3Manager.buildS3Key(req.ModuleName, req.Version, req.Language)
	compressed, hash, size, err := m.S3Manager.compressFiles(req.Files, req.CompressionFormat)
	if err != nil {
		return nil, err
	}

	_, err = m.mockClient.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &m.S3Manager.config.S3Bucket,
		Key:         &key,
		Body:        bytes.NewReader(compressed),
		ContentType: stringPtr(m.S3Manager.getContentType(req.CompressionFormat)),
		Metadata:    m.S3Manager.convertMetadata(req.Metadata),
	})
	if err != nil {
		return nil, ErrUploadFailed
	}

	return &StoreResult{
		S3Key:          key,
		S3Bucket:       m.S3Manager.config.S3Bucket,
		Hash:           hash,
		Size:           size,
		CompressedSize: int64(len(compressed)),
	}, nil
}

func (m *testS3Manager) Retrieve(ctx context.Context, req *RetrieveRequest) (*RetrieveResult, error) {
	if req == nil {
		return nil, errors.New("retrieve request cannot be nil")
	}

	key := m.S3Manager.buildS3Key(req.ModuleName, req.Version, req.Language)

	output, err := m.mockClient.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &m.S3Manager.config.S3Bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, ErrDownloadFailed
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, err
	}

	files, hash, err := m.S3Manager.decompressFiles(data, m.S3Manager.config.CompressionFormat)
	if err != nil {
		return nil, ErrDecompressionFailed
	}

	metadata := m.S3Manager.extractMetadata(output.Metadata)

	return &RetrieveResult{
		Files:    files,
		Metadata: metadata,
		Hash:     hash,
		Size:     int64(len(data)),
	}, nil
}

func (m *testS3Manager) Delete(ctx context.Context, moduleName, version, language string) error {
	key := m.S3Manager.buildS3Key(moduleName, version, language)

	_, err := m.mockClient.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &m.S3Manager.config.S3Bucket,
		Key:    &key,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *testS3Manager) Exists(ctx context.Context, moduleName, version, language string) (bool, error) {
	key := m.S3Manager.buildS3Key(moduleName, version, language)

	_, err := m.mockClient.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &m.S3Manager.config.S3Bucket,
		Key:    &key,
	})
	if err != nil {
		return false, nil
	}

	return true, nil
}

func stringPtr(s string) *string {
	return &s
}

func TestS3Manager_buildS3Key(t *testing.T) {
	manager := newTestS3Manager(&mockS3Client{}, nil)

	tests := []struct {
		name       string
		moduleName string
		version    string
		language   string
		expected   string
	}{
		{
			name:       "basic path",
			moduleName: "testmodule",
			version:    "v1.0.0",
			language:   "go",
			expected:   "compiled/testmodule/v1.0.0/go.tar.gz",
		},
		{
			name:       "with special characters",
			moduleName: "test-module",
			version:    "v2.1.0-beta",
			language:   "python",
			expected:   "compiled/test-module/v2.1.0-beta/python.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.buildS3Key(tt.moduleName, tt.version, tt.language)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestS3Manager_getContentType(t *testing.T) {
	manager := newTestS3Manager(&mockS3Client{}, nil)

	tests := []struct {
		format   string
		expected string
	}{
		{"zip", "application/zip"},
		{"tar.gz", "application/gzip"},
		{"unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := manager.getContentType(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestS3Manager_convertMetadata(t *testing.T) {
	manager := newTestS3Manager(&mockS3Client{}, nil)

	t.Run("nil metadata", func(t *testing.T) {
		result := manager.convertMetadata(nil)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("with metadata", func(t *testing.T) {
		metadata := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		result := manager.convertMetadata(metadata)
		assert.Equal(t, metadata, result)
	})
}

func TestS3Manager_extractMetadata(t *testing.T) {
	manager := newTestS3Manager(&mockS3Client{}, nil)

	t.Run("nil metadata", func(t *testing.T) {
		result := manager.extractMetadata(nil)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("with metadata", func(t *testing.T) {
		metadata := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		result := manager.extractMetadata(metadata)
		assert.Equal(t, metadata, result)
	})
}

func TestS3Manager_compressFiles(t *testing.T) {
	manager := newTestS3Manager(&mockS3Client{}, nil)

	t.Run("compress single file", func(t *testing.T) {
		files := []codegen.GeneratedFile{
			{
				Path:    "test.go",
				Content: []byte("package test\n"),
				Size:    13,
			},
		}

		compressed, hash, size, err := manager.compressFiles(files, "tar.gz")
		require.NoError(t, err)
		assert.NotEmpty(t, compressed)
		assert.NotEmpty(t, hash)
		assert.Equal(t, int64(13), size)
	})

	t.Run("compress multiple files", func(t *testing.T) {
		files := []codegen.GeneratedFile{
			{
				Path:    "file1.go",
				Content: []byte("package test1\n"),
				Size:    14,
			},
			{
				Path:    "file2.go",
				Content: []byte("package test2\n"),
				Size:    14,
			},
		}

		compressed, hash, size, err := manager.compressFiles(files, "tar.gz")
		require.NoError(t, err)
		assert.NotEmpty(t, compressed)
		assert.NotEmpty(t, hash)
		assert.Equal(t, int64(28), size)
	})

	t.Run("compress empty files", func(t *testing.T) {
		files := []codegen.GeneratedFile{}

		compressed, hash, size, err := manager.compressFiles(files, "tar.gz")
		require.NoError(t, err)
		assert.NotEmpty(t, compressed)
		assert.NotEmpty(t, hash)
		assert.Equal(t, int64(0), size)
	})
}

func TestS3Manager_decompressFiles(t *testing.T) {
	manager := newTestS3Manager(&mockS3Client{}, nil)

	t.Run("decompress valid archive", func(t *testing.T) {
		// Create a test archive
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		tarWriter := tar.NewWriter(gzWriter)

		content := []byte("package test\n")
		header := &tar.Header{
			Name: "test.go",
			Mode: 0644,
			Size: int64(len(content)),
		}
		require.NoError(t, tarWriter.WriteHeader(header))
		_, err := tarWriter.Write(content)
		require.NoError(t, err)

		require.NoError(t, tarWriter.Close())
		require.NoError(t, gzWriter.Close())

		// Decompress
		files, hash, err := manager.decompressFiles(buf.Bytes(), "tar.gz")
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, "test.go", files[0].Path)
		assert.Equal(t, content, files[0].Content)
		assert.NotEmpty(t, hash)
	})

	t.Run("decompress invalid data", func(t *testing.T) {
		_, _, err := manager.decompressFiles([]byte("invalid data"), "tar.gz")
		assert.Error(t, err)
	})
}

func TestS3Manager_Store(t *testing.T) {
	t.Run("successful store", func(t *testing.T) {
		mockClient := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				assert.NotNil(t, params.Body)
				assert.NotNil(t, params.Bucket)
				assert.NotNil(t, params.Key)
				return &s3.PutObjectOutput{}, nil
			},
		}
		manager := newTestS3Manager(mockClient, nil)

		req := &StoreRequest{
			ModuleName: "testmodule",
			Version:    "v1.0.0",
			Language:   "go",
			Files: []codegen.GeneratedFile{
				{
					Path:    "test.go",
					Content: []byte("package test\n"),
					Size:    13,
				},
			},
			Metadata: map[string]string{
				"author": "test",
			},
			CompressionFormat: "tar.gz",
		}

		result, err := manager.Store(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.S3Key)
		assert.NotEmpty(t, result.Hash)
		assert.Greater(t, result.Size, int64(0))
		assert.Greater(t, result.CompressedSize, int64(0))
	})

	t.Run("nil request", func(t *testing.T) {
		manager := newTestS3Manager(&mockS3Client{}, nil)
		_, err := manager.Store(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("upload failure", func(t *testing.T) {
		mockClient := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("upload failed")
			},
		}
		manager := newTestS3Manager(mockClient, nil)

		req := &StoreRequest{
			ModuleName: "testmodule",
			Version:    "v1.0.0",
			Language:   "go",
			Files: []codegen.GeneratedFile{
				{
					Path:    "test.go",
					Content: []byte("package test\n"),
					Size:    13,
				},
			},
			CompressionFormat: "tar.gz",
		}

		_, err := manager.Store(context.Background(), req)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUploadFailed)
	})
}

func TestS3Manager_Retrieve(t *testing.T) {
	t.Run("successful retrieve", func(t *testing.T) {
		// Create test archive
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		tarWriter := tar.NewWriter(gzWriter)

		content := []byte("package test\n")
		header := &tar.Header{
			Name: "test.go",
			Mode: 0644,
			Size: int64(len(content)),
		}
		require.NoError(t, tarWriter.WriteHeader(header))
		_, err := tarWriter.Write(content)
		require.NoError(t, err)
		require.NoError(t, tarWriter.Close())
		require.NoError(t, gzWriter.Close())

		mockClient := &mockS3Client{
			getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body: io.NopCloser(bytes.NewReader(buf.Bytes())),
					Metadata: map[string]string{
						"author": "test",
					},
				}, nil
			},
		}
		manager := newTestS3Manager(mockClient, nil)

		req := &RetrieveRequest{
			ModuleName: "testmodule",
			Version:    "v1.0.0",
			Language:   "go",
		}

		result, err := manager.Retrieve(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Files, 1)
		assert.Equal(t, "test.go", result.Files[0].Path)
		assert.Equal(t, content, result.Files[0].Content)
		assert.NotEmpty(t, result.Hash)
	})

	t.Run("nil request", func(t *testing.T) {
		manager := newTestS3Manager(&mockS3Client{}, nil)
		_, err := manager.Retrieve(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("download failure", func(t *testing.T) {
		mockClient := &mockS3Client{
			getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("download failed")
			},
		}
		manager := newTestS3Manager(mockClient, nil)

		req := &RetrieveRequest{
			ModuleName: "testmodule",
			Version:    "v1.0.0",
			Language:   "go",
		}

		_, err := manager.Retrieve(context.Background(), req)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrDownloadFailed)
	})

	t.Run("decompression failure", func(t *testing.T) {
		mockClient := &mockS3Client{
			getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body:     io.NopCloser(bytes.NewReader([]byte("invalid data"))),
					Metadata: map[string]string{},
				}, nil
			},
		}
		manager := newTestS3Manager(mockClient, nil)

		req := &RetrieveRequest{
			ModuleName: "testmodule",
			Version:    "v1.0.0",
			Language:   "go",
		}

		_, err := manager.Retrieve(context.Background(), req)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrDecompressionFailed)
	})
}

func TestS3Manager_Delete(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		mockClient := &mockS3Client{
			deleteObjectFunc: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				assert.NotNil(t, params.Bucket)
				assert.NotNil(t, params.Key)
				return &s3.DeleteObjectOutput{}, nil
			},
		}
		manager := newTestS3Manager(mockClient, nil)

		err := manager.Delete(context.Background(), "testmodule", "v1.0.0", "go")
		assert.NoError(t, err)
	})

	t.Run("delete failure", func(t *testing.T) {
		mockClient := &mockS3Client{
			deleteObjectFunc: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, errors.New("delete failed")
			},
		}
		manager := newTestS3Manager(mockClient, nil)

		err := manager.Delete(context.Background(), "testmodule", "v1.0.0", "go")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})
}

func TestS3Manager_Exists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		mockClient := &mockS3Client{
			headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return &s3.HeadObjectOutput{}, nil
			},
		}
		manager := newTestS3Manager(mockClient, nil)

		exists, err := manager.Exists(context.Background(), "testmodule", "v1.0.0", "go")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("does not exist", func(t *testing.T) {
		mockClient := &mockS3Client{
			headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, errors.New("not found")
			},
		}
		manager := newTestS3Manager(mockClient, nil)

		exists, err := manager.Exists(context.Background(), "testmodule", "v1.0.0", "go")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestS3Manager_Close(t *testing.T) {
	manager := newTestS3Manager(&mockS3Client{}, nil)
	err := manager.Close()
	assert.NoError(t, err)
}

func TestS3Manager_RoundTrip(t *testing.T) {
	// Test full round trip: compress -> store -> retrieve -> decompress
	storage := make(map[string][]byte)

	mockClient := &mockS3Client{
		putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			data, err := io.ReadAll(params.Body)
			if err != nil {
				return nil, err
			}
			storage[*params.Key] = data
			return &s3.PutObjectOutput{}, nil
		},
		getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			data, ok := storage[*params.Key]
			if !ok {
				return nil, errors.New("not found")
			}
			return &s3.GetObjectOutput{
				Body:     io.NopCloser(bytes.NewReader(data)),
				Metadata: map[string]string{},
			}, nil
		},
	}
	manager := newTestS3Manager(mockClient, nil)

	// Original files
	originalFiles := []codegen.GeneratedFile{
		{
			Path:    "file1.go",
			Content: []byte("package file1\n"),
			Size:    14,
		},
		{
			Path:    "dir/file2.go",
			Content: []byte("package file2\n"),
			Size:    14,
		},
	}

	// Store
	storeReq := &StoreRequest{
		ModuleName: "testmodule",
		Version:    "v1.0.0",
		Language:   "go",
		Files:      originalFiles,
		Metadata: map[string]string{
			"test": "value",
		},
		CompressionFormat: "tar.gz",
	}

	storeResult, err := manager.Store(context.Background(), storeReq)
	require.NoError(t, err)
	assert.NotEmpty(t, storeResult.Hash)

	// Retrieve
	retrieveReq := &RetrieveRequest{
		ModuleName: "testmodule",
		Version:    "v1.0.0",
		Language:   "go",
	}

	retrieveResult, err := manager.Retrieve(context.Background(), retrieveReq)
	require.NoError(t, err)
	assert.Len(t, retrieveResult.Files, 2)

	// Verify files match
	for i, file := range retrieveResult.Files {
		assert.Equal(t, originalFiles[i].Path, file.Path)
		assert.Equal(t, originalFiles[i].Content, file.Content)
	}
}
