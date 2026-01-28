package buf

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDownloader(t *testing.T) {
	d := NewDownloader()

	assert.NotNil(t, d)
	assert.Equal(t, "https://buf.build", d.registryURL)
	assert.NotEmpty(t, d.cacheDir)
	assert.Contains(t, d.cacheDir, ".buf/plugins")
	assert.NotNil(t, d.httpClient)
	assert.Equal(t, 5*time.Minute, d.httpClient.Timeout)
}

func TestDownload_InvalidPluginReference(t *testing.T) {
	d := NewDownloader()

	tests := []struct {
		name      string
		pluginRef string
	}{
		{
			name:      "empty string",
			pluginRef: "",
		},
		{
			name:      "single part",
			pluginRef: "test",
		},
		{
			name:      "two parts",
			pluginRef: "buf.build/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := d.Download(tt.pluginRef, "v1.0.0")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid plugin reference")
		})
	}
}

func TestDownload_ValidPluginReference(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL format
		assert.Contains(t, r.URL.Path, "/library/connect-go/plugins/v1.0.0/")

		// Create a simple zip file with a mock binary
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)

		zipWriter := zip.NewWriter(w)
		binaryName := "protoc-gen-connect-go"
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}

		writer, err := zipWriter.Create(binaryName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = writer.Write([]byte("fake binary content"))
		zipWriter.Close()
	}))
	defer server.Close()

	// Create temporary cache directory
	tmpDir := t.TempDir()

	d := &Downloader{
		registryURL: server.URL,
		cacheDir:    tmpDir,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	path, err := d.Download("buf.build/library/connect-go", "v1.0.0")
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "protoc-gen-connect-go")

	// Verify file exists
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestDownload_CachedPlugin(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create the cached binary
	pluginDir := filepath.Join(tmpDir, "connect-go", "v1.0.0")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	binaryName := "protoc-gen-connect-go"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(pluginDir, binaryName)
	err = os.WriteFile(binaryPath, []byte("cached binary"), 0755)
	require.NoError(t, err)

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    tmpDir,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	// Should return cached path without downloading
	path, err := d.Download("buf.build/library/connect-go", "v1.0.0")
	require.NoError(t, err)
	assert.Equal(t, binaryPath, path)
}

func TestDownload_HTTPError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	d := &Downloader{
		registryURL: server.URL,
		cacheDir:    tmpDir,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	_, err := d.Download("buf.build/library/connect-go", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download failed: HTTP 404")
}

func TestDownload_CreateCacheDirError(t *testing.T) {
	// Create a file where we expect a directory
	tmpDir := t.TempDir()
	badPath := filepath.Join(tmpDir, "blocked")
	err := os.WriteFile(badPath, []byte("blocking file"), 0644)
	require.NoError(t, err)

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    filepath.Join(badPath, "subdir"), // This will fail
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	_, err = d.Download("buf.build/library/connect-go", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create cache directory")
}

func TestDownload_BinaryNotFoundAfterExtraction(t *testing.T) {
	// Create test server that returns a zip without the expected binary
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)

		zipWriter := zip.NewWriter(w)
		// Create a file with the wrong name
		writer, _ := zipWriter.Create("wrong-name.txt")
		_, _ = writer.Write([]byte("not the binary"))
		zipWriter.Close()
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	d := &Downloader{
		registryURL: server.URL,
		cacheDir:    tmpDir,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	_, err := d.Download("buf.build/library/test", "v1.0.0")
	assert.Error(t, err)
	// The error will be from extractArchive about binary not found
	assert.True(t, strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "extract"))
}

func TestExtractZip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test zip file
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)

	zipWriter := zip.NewWriter(zipFile)
	binaryName := "protoc-gen-test"
	writer, err := zipWriter.Create(binaryName)
	require.NoError(t, err)
	_, err = writer.Write([]byte("test binary content"))
	require.NoError(t, err)
	err = zipWriter.Close()
	require.NoError(t, err)
	zipFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractZip(zipPath, destDir, binaryName)
	assert.NoError(t, err)

	// Verify extracted file
	extractedPath := filepath.Join(destDir, binaryName)
	content, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, "test binary content", string(content))
}

func TestExtractZip_BinaryNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test zip file without the expected binary
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)

	zipWriter := zip.NewWriter(zipFile)
	writer, err := zipWriter.Create("other-file.txt")
	require.NoError(t, err)
	_, err = writer.Write([]byte("not the binary"))
	require.NoError(t, err)
	zipWriter.Close()
	zipFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractZip(zipPath, destDir, "protoc-gen-test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "binary protoc-gen-test not found in archive")
}

func TestExtractZip_InvalidZip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid zip file
	zipPath := filepath.Join(tmpDir, "invalid.zip")
	err := os.WriteFile(zipPath, []byte("not a zip file"), 0644)
	require.NoError(t, err)

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractZip(zipPath, destDir, "protoc-gen-test")
	assert.Error(t, err)
}

func TestExtractZip_WithSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test zip file with binary in subdirectory
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)

	zipWriter := zip.NewWriter(zipFile)
	binaryName := "protoc-gen-test"
	// Binary in subdirectory
	writer, err := zipWriter.Create("bin/" + binaryName)
	require.NoError(t, err)
	_, err = writer.Write([]byte("test binary content"))
	require.NoError(t, err)
	zipWriter.Close()
	zipFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractZip(zipPath, destDir, binaryName)
	assert.NoError(t, err)

	// Verify extracted file
	extractedPath := filepath.Join(destDir, binaryName)
	content, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, "test binary content", string(content))
}

func TestExtractTarGz(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test tar.gz file
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")
	tarGzFile, err := os.Create(tarGzPath)
	require.NoError(t, err)

	gzWriter := gzip.NewWriter(tarGzFile)
	tarWriter := tar.NewWriter(gzWriter)

	binaryName := "protoc-gen-test"
	content := []byte("test binary content")
	header := &tar.Header{
		Name: binaryName,
		Mode: 0755,
		Size: int64(len(content)),
	}
	err = tarWriter.WriteHeader(header)
	require.NoError(t, err)
	_, err = tarWriter.Write(content)
	require.NoError(t, err)

	tarWriter.Close()
	gzWriter.Close()
	tarGzFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractTarGz(tarGzPath, destDir, binaryName)
	assert.NoError(t, err)

	// Verify extracted file
	extractedPath := filepath.Join(destDir, binaryName)
	extractedContent, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, "test binary content", string(extractedContent))
}

func TestExtractTarGz_BinaryNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test tar.gz file without the expected binary
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")
	tarGzFile, err := os.Create(tarGzPath)
	require.NoError(t, err)

	gzWriter := gzip.NewWriter(tarGzFile)
	tarWriter := tar.NewWriter(gzWriter)

	content := []byte("not the binary")
	header := &tar.Header{
		Name: "other-file.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}
	err = tarWriter.WriteHeader(header)
	require.NoError(t, err)
	_, err = tarWriter.Write(content)
	require.NoError(t, err)

	tarWriter.Close()
	gzWriter.Close()
	tarGzFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractTarGz(tarGzPath, destDir, "protoc-gen-test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "binary protoc-gen-test not found in archive")
}

func TestExtractTarGz_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid tar.gz file
	tarGzPath := filepath.Join(tmpDir, "invalid.tar.gz")
	err := os.WriteFile(tarGzPath, []byte("not a tar.gz file"), 0644)
	require.NoError(t, err)

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractTarGz(tarGzPath, destDir, "protoc-gen-test")
	assert.Error(t, err)
}

func TestExtractTarGz_WithSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test tar.gz file with binary in subdirectory
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")
	tarGzFile, err := os.Create(tarGzPath)
	require.NoError(t, err)

	gzWriter := gzip.NewWriter(tarGzFile)
	tarWriter := tar.NewWriter(gzWriter)

	binaryName := "protoc-gen-test"
	content := []byte("test binary content")
	header := &tar.Header{
		Name: "bin/" + binaryName,
		Mode: 0755,
		Size: int64(len(content)),
	}
	err = tarWriter.WriteHeader(header)
	require.NoError(t, err)
	_, err = tarWriter.Write(content)
	require.NoError(t, err)

	tarWriter.Close()
	gzWriter.Close()
	tarGzFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractTarGz(tarGzPath, destDir, binaryName)
	assert.NoError(t, err)

	// Verify extracted file
	extractedPath := filepath.Join(destDir, binaryName)
	extractedContent, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, "test binary content", string(extractedContent))
}

func TestExtractArchive_Zip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test zip file
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)

	zipWriter := zip.NewWriter(zipFile)
	binaryName := "protoc-gen-test"
	writer, err := zipWriter.Create(binaryName)
	require.NoError(t, err)
	_, err = writer.Write([]byte("test binary"))
	require.NoError(t, err)
	zipWriter.Close()
	zipFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractArchive(zipPath, destDir, binaryName)
	assert.NoError(t, err)
}

func TestExtractArchive_TarGz(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test tar.gz file
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")
	tarGzFile, err := os.Create(tarGzPath)
	require.NoError(t, err)

	gzWriter := gzip.NewWriter(tarGzFile)
	tarWriter := tar.NewWriter(gzWriter)

	binaryName := "protoc-gen-test"
	content := []byte("test binary")
	header := &tar.Header{
		Name: binaryName,
		Mode: 0755,
		Size: int64(len(content)),
	}
	err = tarWriter.WriteHeader(header)
	require.NoError(t, err)
	_, err = tarWriter.Write(content)
	require.NoError(t, err)

	tarWriter.Close()
	gzWriter.Close()
	tarGzFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractArchive(tarGzPath, destDir, binaryName)
	assert.NoError(t, err)
}

func TestExtractArchive_Tgz(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test .tgz file
	tgzPath := filepath.Join(tmpDir, "test.tgz")
	tgzFile, err := os.Create(tgzPath)
	require.NoError(t, err)

	gzWriter := gzip.NewWriter(tgzFile)
	tarWriter := tar.NewWriter(gzWriter)

	binaryName := "protoc-gen-test"
	content := []byte("test binary")
	header := &tar.Header{
		Name: binaryName,
		Mode: 0755,
		Size: int64(len(content)),
	}
	err = tarWriter.WriteHeader(header)
	require.NoError(t, err)
	_, err = tarWriter.Write(content)
	require.NoError(t, err)

	tarWriter.Close()
	gzWriter.Close()
	tgzFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractArchive(tgzPath, destDir, binaryName)
	assert.NoError(t, err)
}

func TestExtractArchive_UnknownFormat_FallbackToZip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a zip file with no extension
	unknownPath := filepath.Join(tmpDir, "test.unknown")
	unknownFile, err := os.Create(unknownPath)
	require.NoError(t, err)

	zipWriter := zip.NewWriter(unknownFile)
	binaryName := "protoc-gen-test"
	writer, err := zipWriter.Create(binaryName)
	require.NoError(t, err)
	_, err = writer.Write([]byte("test binary"))
	require.NoError(t, err)
	zipWriter.Close()
	unknownFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractArchive(unknownPath, destDir, binaryName)
	assert.NoError(t, err)
}

func TestExtractArchive_UnknownFormat_FallbackToTarGz(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a tar.gz file with no extension
	unknownPath := filepath.Join(tmpDir, "test.unknown")
	unknownFile, err := os.Create(unknownPath)
	require.NoError(t, err)

	gzWriter := gzip.NewWriter(unknownFile)
	tarWriter := tar.NewWriter(gzWriter)

	binaryName := "protoc-gen-test"
	content := []byte("test binary")
	header := &tar.Header{
		Name: binaryName,
		Mode: 0755,
		Size: int64(len(content)),
	}
	err = tarWriter.WriteHeader(header)
	require.NoError(t, err)
	_, err = tarWriter.Write(content)
	require.NoError(t, err)

	tarWriter.Close()
	gzWriter.Close()
	unknownFile.Close()

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractArchive(unknownPath, destDir, binaryName)
	assert.NoError(t, err)
}

func TestClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	// Create some cache files
	err := os.MkdirAll(filepath.Join(cacheDir, "plugin1", "v1.0.0"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(cacheDir, "plugin1", "v1.0.0", "binary"), []byte("test"), 0755)
	require.NoError(t, err)

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	err = d.ClearCache()
	assert.NoError(t, err)

	// Verify cache directory is removed
	_, err = os.Stat(cacheDir)
	assert.True(t, os.IsNotExist(err))
}

func TestClearCache_NonexistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "nonexistent")

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	err := d.ClearCache()
	// os.RemoveAll returns nil for nonexistent directories
	assert.NoError(t, err)
}

func TestGetCacheSize(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	// Create some cache files
	err := os.MkdirAll(filepath.Join(cacheDir, "plugin1", "v1.0.0"), 0755)
	require.NoError(t, err)

	file1Content := []byte("test content 1")
	err = os.WriteFile(filepath.Join(cacheDir, "plugin1", "v1.0.0", "binary1"), file1Content, 0755)
	require.NoError(t, err)

	file2Content := []byte("test content 2 longer")
	err = os.WriteFile(filepath.Join(cacheDir, "plugin1", "v1.0.0", "binary2"), file2Content, 0755)
	require.NoError(t, err)

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	size, err := d.GetCacheSize()
	assert.NoError(t, err)
	expectedSize := int64(len(file1Content) + len(file2Content))
	assert.Equal(t, expectedSize, size)
}

func TestGetCacheSize_EmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	// Create empty cache directory
	err := os.MkdirAll(cacheDir, 0755)
	require.NoError(t, err)

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	size, err := d.GetCacheSize()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), size)
}

func TestGetCacheSize_NonexistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "nonexistent")

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	size, err := d.GetCacheSize()
	assert.Error(t, err)
	assert.Equal(t, int64(0), size)
}

func TestListCachedPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	// Create some plugin directories
	err := os.MkdirAll(filepath.Join(cacheDir, "connect-go", "v1.0.0"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(cacheDir, "connect-go", "v1.1.0"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(cacheDir, "grpc-go", "v2.0.0"), 0755)
	require.NoError(t, err)

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	plugins, err := d.ListCachedPlugins()
	assert.NoError(t, err)
	assert.Len(t, plugins, 3)

	// Check that all expected plugins are listed
	pluginMap := make(map[string]bool)
	for _, p := range plugins {
		pluginMap[p] = true
	}
	assert.True(t, pluginMap["connect-go@v1.0.0"])
	assert.True(t, pluginMap["connect-go@v1.1.0"])
	assert.True(t, pluginMap["grpc-go@v2.0.0"])
}

func TestListCachedPlugins_EmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	// Create empty cache directory
	err := os.MkdirAll(cacheDir, 0755)
	require.NoError(t, err)

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	plugins, err := d.ListCachedPlugins()
	assert.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestListCachedPlugins_NonexistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "nonexistent")

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	plugins, err := d.ListCachedPlugins()
	assert.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestListCachedPlugins_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	// Create plugin directories
	err := os.MkdirAll(filepath.Join(cacheDir, "connect-go", "v1.0.0"), 0755)
	require.NoError(t, err)

	// Create a file (not directory) in cache root - should be skipped
	err = os.WriteFile(filepath.Join(cacheDir, "readme.txt"), []byte("info"), 0644)
	require.NoError(t, err)

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	plugins, err := d.ListCachedPlugins()
	assert.NoError(t, err)
	assert.Len(t, plugins, 1)
	assert.Equal(t, "connect-go@v1.0.0", plugins[0])
}

func TestListCachedPlugins_WithFilesInVersionDir(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	// Create plugin directory structure
	err := os.MkdirAll(filepath.Join(cacheDir, "connect-go", "v1.0.0"), 0755)
	require.NoError(t, err)

	// Create a file (not directory) in version directory - should be skipped
	err = os.WriteFile(filepath.Join(cacheDir, "connect-go", "config.txt"), []byte("config"), 0644)
	require.NoError(t, err)

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	plugins, err := d.ListCachedPlugins()
	assert.NoError(t, err)
	assert.Len(t, plugins, 1)
	assert.Equal(t, "connect-go@v1.0.0", plugins[0])
}

func TestDownload_WindowsBinaryExtension(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)

		zipWriter := zip.NewWriter(w)
		// Windows binary should have .exe extension
		writer, err := zipWriter.Create("protoc-gen-test.exe")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = writer.Write([]byte("windows binary"))
		zipWriter.Close()
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	d := &Downloader{
		registryURL: server.URL,
		cacheDir:    tmpDir,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	path, err := d.Download("buf.build/library/test", "v1.0.0")
	require.NoError(t, err)
	assert.Contains(t, path, ".exe")
}

func TestDownload_PlatformInURL(t *testing.T) {
	expectedPlatform := runtime.GOOS + "_" + runtime.GOARCH

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify platform is in the URL
		assert.Contains(t, r.URL.Path, expectedPlatform)

		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)

		zipWriter := zip.NewWriter(w)
		binaryName := "protoc-gen-test"
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}
		writer, _ := zipWriter.Create(binaryName)
		_, _ = writer.Write([]byte("binary"))
		zipWriter.Close()
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	d := &Downloader{
		registryURL: server.URL,
		cacheDir:    tmpDir,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	_, err := d.Download("buf.build/library/test", "v1.0.0")
	require.NoError(t, err)
}

func TestExtractZip_ErrorOpeningFile(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "extracted")
	err := os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	// Try to extract from a directory instead of a file
	invalidPath := filepath.Join(tmpDir, "not-a-zip-dir")
	err = os.MkdirAll(invalidPath, 0755)
	require.NoError(t, err)

	d := NewDownloader()
	err = d.extractZip(invalidPath, destDir, "test-binary")
	assert.Error(t, err)
}

func TestExtractZip_ErrorCreatingOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid zip file
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)

	zipWriter := zip.NewWriter(zipFile)
	binaryName := "protoc-gen-test"
	writer, err := zipWriter.Create(binaryName)
	require.NoError(t, err)
	_, err = writer.Write([]byte("test binary"))
	require.NoError(t, err)
	zipWriter.Close()
	zipFile.Close()

	d := NewDownloader()

	// Use a file as destination instead of directory
	badDestPath := filepath.Join(tmpDir, "bad-dest")
	err = os.WriteFile(badDestPath, []byte("blocking"), 0644)
	require.NoError(t, err)

	err = d.extractZip(zipPath, badDestPath, binaryName)
	assert.Error(t, err)
}

func TestExtractTarGz_ErrorOpeningFile(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "extracted")
	err := os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	d := NewDownloader()
	err = d.extractTarGz("/nonexistent/file.tar.gz", destDir, "test-binary")
	assert.Error(t, err)
}

func TestExtractTarGz_ErrorCreatingOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid tar.gz file
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")
	tarGzFile, err := os.Create(tarGzPath)
	require.NoError(t, err)

	gzWriter := gzip.NewWriter(tarGzFile)
	tarWriter := tar.NewWriter(gzWriter)

	binaryName := "protoc-gen-test"
	content := []byte("test binary")
	header := &tar.Header{
		Name: binaryName,
		Mode: 0755,
		Size: int64(len(content)),
	}
	err = tarWriter.WriteHeader(header)
	require.NoError(t, err)
	_, err = tarWriter.Write(content)
	require.NoError(t, err)

	tarWriter.Close()
	gzWriter.Close()
	tarGzFile.Close()

	d := NewDownloader()

	// Use a file as destination instead of directory
	badDestPath := filepath.Join(tmpDir, "bad-dest")
	err = os.WriteFile(badDestPath, []byte("blocking"), 0644)
	require.NoError(t, err)

	err = d.extractTarGz(tarGzPath, badDestPath, binaryName)
	assert.Error(t, err)
}

func TestExtractArchive_BothFormatsInvalid(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid archive file
	invalidPath := filepath.Join(tmpDir, "test.unknown")
	err := os.WriteFile(invalidPath, []byte("not a valid archive format"), 0644)
	require.NoError(t, err)

	d := NewDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	err = d.extractArchive(invalidPath, destDir, "protoc-gen-test")
	assert.Error(t, err)
}

func TestGetCacheSize_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	// Create nested directory structure with files
	err := os.MkdirAll(filepath.Join(cacheDir, "plugin1", "v1.0.0", "subdir"), 0755)
	require.NoError(t, err)

	file1 := []byte("content1")
	file2 := []byte("content2 is longer")
	file3 := []byte("deep")

	err = os.WriteFile(filepath.Join(cacheDir, "plugin1", "v1.0.0", "binary1"), file1, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(cacheDir, "plugin1", "v1.0.0", "binary2"), file2, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(cacheDir, "plugin1", "v1.0.0", "subdir", "file3"), file3, 0644)
	require.NoError(t, err)

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	size, err := d.GetCacheSize()
	assert.NoError(t, err)
	expectedSize := int64(len(file1) + len(file2) + len(file3))
	assert.Equal(t, expectedSize, size)
}

func TestListCachedPlugins_MultiplePlugins(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	// Create multiple plugins with multiple versions
	plugins := map[string][]string{
		"connect-go": {"v1.0.0", "v1.1.0", "v1.2.0"},
		"grpc-go":    {"v2.0.0", "v2.1.0"},
		"validate":   {"v0.6.0"},
	}

	for plugin, versions := range plugins {
		for _, version := range versions {
			err := os.MkdirAll(filepath.Join(cacheDir, plugin, version), 0755)
			require.NoError(t, err)
		}
	}

	d := &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    cacheDir,
		httpClient:  &http.Client{},
	}

	cached, err := d.ListCachedPlugins()
	assert.NoError(t, err)
	assert.Len(t, cached, 6)

	// Verify all expected plugins are present
	expectedPlugins := []string{
		"connect-go@v1.0.0", "connect-go@v1.1.0", "connect-go@v1.2.0",
		"grpc-go@v2.0.0", "grpc-go@v2.1.0",
		"validate@v0.6.0",
	}

	pluginMap := make(map[string]bool)
	for _, p := range cached {
		pluginMap[p] = true
	}

	for _, expected := range expectedPlugins {
		assert.True(t, pluginMap[expected], "Expected plugin %s not found", expected)
	}
}
