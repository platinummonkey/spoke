package buf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	cache := NewCache()

	assert.NotNil(t, cache)
	assert.NotEmpty(t, cache.cacheDir)
	assert.NotEmpty(t, cache.metadataDir)
	assert.Contains(t, cache.cacheDir, ".buf/plugins")
	assert.Contains(t, cache.metadataDir, ".buf/cache-metadata")
}

func TestCacheEntry_Serialization(t *testing.T) {
	entry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   "/path/to/binary",
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     "abc123",
		Size:         12345,
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded CacheEntry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, entry.PluginRef, decoded.PluginRef)
	assert.Equal(t, entry.Version, decoded.Version)
	assert.Equal(t, entry.BinaryPath, decoded.BinaryPath)
	assert.Equal(t, entry.Checksum, decoded.Checksum)
	assert.Equal(t, entry.Size, decoded.Size)
}

func TestSaveEntry_Success(t *testing.T) {
	// Create temporary cache directory
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	entry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   "/path/to/binary",
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     "abc123",
		Size:         12345,
	}

	err := cache.SaveEntry(entry)
	require.NoError(t, err)

	// Verify metadata file was created
	metadataPath := cache.getMetadataPath(entry.PluginRef, entry.Version)
	_, err = os.Stat(metadataPath)
	assert.NoError(t, err)

	// Verify metadata directory was created with correct permissions
	info, err := os.Stat(cache.metadataDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestGetEntry_Success(t *testing.T) {
	// Create temporary cache directory
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	entry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   "/path/to/binary",
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     "abc123",
		Size:         12345,
	}

	// Save entry first
	err := cache.SaveEntry(entry)
	require.NoError(t, err)

	// Retrieve entry
	retrieved, err := cache.GetEntry(entry.PluginRef, entry.Version)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, entry.PluginRef, retrieved.PluginRef)
	assert.Equal(t, entry.Version, retrieved.Version)
	assert.Equal(t, entry.BinaryPath, retrieved.BinaryPath)
	assert.Equal(t, entry.Checksum, retrieved.Checksum)
	assert.Equal(t, entry.Size, retrieved.Size)
}

func TestGetEntry_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	_, err := cache.GetEntry("buf.build/nonexistent/plugin", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache entry not found")
}

func TestGetEntry_CorruptedMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create metadata directory
	err := os.MkdirAll(cache.metadataDir, 0755)
	require.NoError(t, err)

	// Write corrupted metadata
	metadataPath := cache.getMetadataPath("buf.build/test/plugin", "v1.0.0")
	err = os.WriteFile(metadataPath, []byte("invalid json"), 0644)
	require.NoError(t, err)

	_, err = cache.GetEntry("buf.build/test/plugin", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse cache entry")
}

func TestUpdateLastUsed(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	originalTime := time.Now().Add(-1 * time.Hour)
	entry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   "/path/to/binary",
		DownloadedAt: originalTime,
		LastUsedAt:   originalTime,
		Checksum:     "abc123",
		Size:         12345,
	}

	// Save entry
	err := cache.SaveEntry(entry)
	require.NoError(t, err)

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update last used
	err = cache.UpdateLastUsed(entry.PluginRef, entry.Version)
	require.NoError(t, err)

	// Retrieve and verify
	updated, err := cache.GetEntry(entry.PluginRef, entry.Version)
	require.NoError(t, err)
	assert.True(t, updated.LastUsedAt.After(originalTime))
}

func TestUpdateLastUsed_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	err := cache.UpdateLastUsed("buf.build/nonexistent/plugin", "v1.0.0")
	assert.Error(t, err)
}

func TestListEntries_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	entries, err := cache.ListEntries()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestListEntries_Multiple(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create multiple entries
	entries := []*CacheEntry{
		{
			PluginRef:    "buf.build/library/connect-go",
			Version:      "v1.5.0",
			BinaryPath:   "/path/to/binary1",
			DownloadedAt: time.Now(),
			LastUsedAt:   time.Now(),
			Checksum:     "abc123",
			Size:         12345,
		},
		{
			PluginRef:    "buf.build/library/grpc-go",
			Version:      "v1.2.0",
			BinaryPath:   "/path/to/binary2",
			DownloadedAt: time.Now(),
			LastUsedAt:   time.Now(),
			Checksum:     "def456",
			Size:         67890,
		},
	}

	for _, entry := range entries {
		err := cache.SaveEntry(entry)
		require.NoError(t, err)
	}

	// List entries
	listed, err := cache.ListEntries()
	require.NoError(t, err)
	assert.Len(t, listed, 2)

	// Verify entries are present (order may vary)
	pluginRefs := make(map[string]bool)
	for _, entry := range listed {
		pluginRefs[entry.PluginRef] = true
	}
	assert.True(t, pluginRefs["buf.build/library/connect-go"])
	assert.True(t, pluginRefs["buf.build/library/grpc-go"])
}

func TestListEntries_SkipsInvalidEntries(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create metadata directory
	err := os.MkdirAll(cache.metadataDir, 0755)
	require.NoError(t, err)

	// Create one valid entry
	validEntry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   "/path/to/binary",
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     "abc123",
		Size:         12345,
	}
	err = cache.SaveEntry(validEntry)
	require.NoError(t, err)

	// Create an invalid metadata file
	invalidPath := filepath.Join(cache.metadataDir, "invalid.json")
	err = os.WriteFile(invalidPath, []byte("invalid json"), 0644)
	require.NoError(t, err)

	// Create a non-JSON file (should be skipped)
	nonJSONPath := filepath.Join(cache.metadataDir, "notjson.txt")
	err = os.WriteFile(nonJSONPath, []byte("text file"), 0644)
	require.NoError(t, err)

	// List entries - should only return the valid entry
	entries, err := cache.ListEntries()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, validEntry.PluginRef, entries[0].PluginRef)
}

func TestRemoveEntry_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create a binary file
	binaryDir := filepath.Join(tmpDir, "binaries")
	err := os.MkdirAll(binaryDir, 0755)
	require.NoError(t, err)

	binaryPath := filepath.Join(binaryDir, "test-binary")
	err = os.WriteFile(binaryPath, []byte("binary content"), 0755)
	require.NoError(t, err)

	entry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   binaryPath,
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     "abc123",
		Size:         12345,
	}

	// Save entry
	err = cache.SaveEntry(entry)
	require.NoError(t, err)

	// Remove entry
	err = cache.RemoveEntry(entry.PluginRef, entry.Version)
	require.NoError(t, err)

	// Verify metadata is removed
	_, err = cache.GetEntry(entry.PluginRef, entry.Version)
	assert.Error(t, err)

	// Verify binary is removed
	_, err = os.Stat(binaryPath)
	assert.True(t, os.IsNotExist(err))
}

func TestRemoveEntry_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	err := cache.RemoveEntry("buf.build/nonexistent/plugin", "v1.0.0")
	assert.Error(t, err)
}

func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create some entries
	entry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   "/path/to/binary",
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     "abc123",
		Size:         12345,
	}
	err := cache.SaveEntry(entry)
	require.NoError(t, err)

	// Create cache directory with a file
	err = os.MkdirAll(cache.cacheDir, 0755)
	require.NoError(t, err)
	testFile := filepath.Join(cache.cacheDir, "test.bin")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Clear cache
	err = cache.Clear()
	require.NoError(t, err)

	// Verify directories are removed
	_, err = os.Stat(cache.cacheDir)
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(cache.metadataDir)
	assert.True(t, os.IsNotExist(err))
}

func TestGetTotalSize(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create entries with different sizes
	entries := []*CacheEntry{
		{
			PluginRef:    "buf.build/library/connect-go",
			Version:      "v1.5.0",
			BinaryPath:   "/path/to/binary1",
			DownloadedAt: time.Now(),
			LastUsedAt:   time.Now(),
			Checksum:     "abc123",
			Size:         12345,
		},
		{
			PluginRef:    "buf.build/library/grpc-go",
			Version:      "v1.2.0",
			BinaryPath:   "/path/to/binary2",
			DownloadedAt: time.Now(),
			LastUsedAt:   time.Now(),
			Checksum:     "def456",
			Size:         67890,
		},
	}

	for _, entry := range entries {
		err := cache.SaveEntry(entry)
		require.NoError(t, err)
	}

	// Get total size
	totalSize, err := cache.GetTotalSize()
	require.NoError(t, err)
	assert.Equal(t, int64(12345+67890), totalSize)
}

func TestGetTotalSize_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	totalSize, err := cache.GetTotalSize()
	require.NoError(t, err)
	assert.Equal(t, int64(0), totalSize)
}

func TestPruneOldEntries(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create entries with different last used times
	oldTime := time.Now().Add(-48 * time.Hour)
	recentTime := time.Now().Add(-1 * time.Hour)

	entries := []*CacheEntry{
		{
			PluginRef:    "buf.build/library/old-plugin",
			Version:      "v1.0.0",
			BinaryPath:   "/path/to/old",
			DownloadedAt: oldTime,
			LastUsedAt:   oldTime,
			Checksum:     "old123",
			Size:         1000,
		},
		{
			PluginRef:    "buf.build/library/recent-plugin",
			Version:      "v1.0.0",
			BinaryPath:   "/path/to/recent",
			DownloadedAt: recentTime,
			LastUsedAt:   recentTime,
			Checksum:     "recent456",
			Size:         2000,
		},
	}

	for _, entry := range entries {
		err := cache.SaveEntry(entry)
		require.NoError(t, err)
	}

	// Prune entries older than 24 hours
	pruned, err := cache.PruneOldEntries(24 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)

	// Verify old entry is removed
	_, err = cache.GetEntry("buf.build/library/old-plugin", "v1.0.0")
	assert.Error(t, err)

	// Verify recent entry still exists
	_, err = cache.GetEntry("buf.build/library/recent-plugin", "v1.0.0")
	assert.NoError(t, err)
}

func TestPruneOldEntries_NoneToRemove(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	entry := &CacheEntry{
		PluginRef:    "buf.build/library/recent-plugin",
		Version:      "v1.0.0",
		BinaryPath:   "/path/to/recent",
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     "recent456",
		Size:         2000,
	}

	err := cache.SaveEntry(entry)
	require.NoError(t, err)

	// Prune with very long duration
	pruned, err := cache.PruneOldEntries(365 * 24 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 0, pruned)
}

func TestVerifyIntegrity_AllValid(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create a binary file
	binaryDir := filepath.Join(tmpDir, "binaries")
	err := os.MkdirAll(binaryDir, 0755)
	require.NoError(t, err)

	binaryPath := filepath.Join(binaryDir, "test-binary")
	binaryContent := []byte("binary content")
	err = os.WriteFile(binaryPath, binaryContent, 0755)
	require.NoError(t, err)

	// Calculate correct checksum
	checksum, err := cache.calculateChecksum(binaryPath)
	require.NoError(t, err)

	entry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   binaryPath,
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     checksum,
		Size:         int64(len(binaryContent)),
	}

	err = cache.SaveEntry(entry)
	require.NoError(t, err)

	// Verify integrity
	corrupted, err := cache.VerifyIntegrity()
	require.NoError(t, err)
	assert.Empty(t, corrupted)
}

func TestVerifyIntegrity_MissingBinary(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	entry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   "/nonexistent/path/to/binary",
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     "abc123",
		Size:         12345,
	}

	err := cache.SaveEntry(entry)
	require.NoError(t, err)

	// Verify integrity
	corrupted, err := cache.VerifyIntegrity()
	require.NoError(t, err)
	assert.Len(t, corrupted, 1)
	assert.Contains(t, corrupted[0], "missing binary")
	assert.Contains(t, corrupted[0], "buf.build/library/connect-go")
}

func TestVerifyIntegrity_ChecksumMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create a binary file
	binaryDir := filepath.Join(tmpDir, "binaries")
	err := os.MkdirAll(binaryDir, 0755)
	require.NoError(t, err)

	binaryPath := filepath.Join(binaryDir, "test-binary")
	err = os.WriteFile(binaryPath, []byte("binary content"), 0755)
	require.NoError(t, err)

	entry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   binaryPath,
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     "wrong_checksum",
		Size:         12345,
	}

	err = cache.SaveEntry(entry)
	require.NoError(t, err)

	// Verify integrity
	corrupted, err := cache.VerifyIntegrity()
	require.NoError(t, err)
	assert.Len(t, corrupted, 1)
	assert.Contains(t, corrupted[0], "checksum mismatch")
	assert.Contains(t, corrupted[0], "buf.build/library/connect-go")
}

func TestVerifyIntegrity_ChecksumError(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create a binary directory without read permissions to trigger checksum error
	binaryDir := filepath.Join(tmpDir, "binaries")
	err := os.MkdirAll(binaryDir, 0755)
	require.NoError(t, err)

	binaryPath := filepath.Join(binaryDir, "test-binary")
	err = os.WriteFile(binaryPath, []byte("binary content"), 0000) // No permissions
	require.NoError(t, err)

	entry := &CacheEntry{
		PluginRef:    "buf.build/library/connect-go",
		Version:      "v1.5.0",
		BinaryPath:   binaryPath,
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     "abc123",
		Size:         12345,
	}

	err = cache.SaveEntry(entry)
	require.NoError(t, err)

	// Verify integrity
	corrupted, err := cache.VerifyIntegrity()
	require.NoError(t, err)
	assert.Len(t, corrupted, 1)
	assert.Contains(t, corrupted[0], "checksum error")
}

func TestGetMetadataPath(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	path1 := cache.getMetadataPath("buf.build/library/connect-go", "v1.5.0")
	path2 := cache.getMetadataPath("buf.build/library/connect-go", "v1.5.0")
	path3 := cache.getMetadataPath("buf.build/library/grpc-go", "v1.2.0")

	// Same inputs should produce same path
	assert.Equal(t, path1, path2)

	// Different inputs should produce different paths
	assert.NotEqual(t, path1, path3)

	// Path should be in metadata directory
	assert.Contains(t, path1, cache.metadataDir)

	// Path should have .json extension
	assert.True(t, filepath.Ext(path1) == ".json")

	// Filename should be a hex-encoded hash
	filename := filepath.Base(path1)
	assert.Len(t, filename, 64+5) // SHA-256 hash (64 chars) + ".json" (5 chars)
}

func TestCalculateChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    tmpDir,
		metadataDir: tmpDir,
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "test.bin")
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)

	// Calculate checksum
	checksum1, err := cache.calculateChecksum(testFile)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum1)
	assert.Len(t, checksum1, 64) // SHA-256 produces 64 hex characters

	// Calculate again - should be the same
	checksum2, err := cache.calculateChecksum(testFile)
	require.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)

	// Modify file
	err = os.WriteFile(testFile, []byte("different content"), 0644)
	require.NoError(t, err)

	// Checksum should be different
	checksum3, err := cache.calculateChecksum(testFile)
	require.NoError(t, err)
	assert.NotEqual(t, checksum1, checksum3)
}

func TestCalculateChecksum_NonExistentFile(t *testing.T) {
	cache := &Cache{
		cacheDir:    "/tmp",
		metadataDir: "/tmp",
	}

	_, err := cache.calculateChecksum("/nonexistent/file")
	assert.Error(t, err)
}

func TestCreateEntry_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create a binary file
	binaryPath := filepath.Join(tmpDir, "test-plugin")
	binaryContent := []byte("plugin binary content")
	err := os.WriteFile(binaryPath, binaryContent, 0755)
	require.NoError(t, err)

	// Create entry
	entry, err := cache.CreateEntry("buf.build/library/connect-go", "v1.5.0", binaryPath)
	require.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, "buf.build/library/connect-go", entry.PluginRef)
	assert.Equal(t, "v1.5.0", entry.Version)
	assert.Equal(t, binaryPath, entry.BinaryPath)
	assert.Equal(t, int64(len(binaryContent)), entry.Size)
	assert.NotEmpty(t, entry.Checksum)
	assert.False(t, entry.DownloadedAt.IsZero())
	assert.False(t, entry.LastUsedAt.IsZero())

	// Verify entry was saved
	retrieved, err := cache.GetEntry("buf.build/library/connect-go", "v1.5.0")
	require.NoError(t, err)
	assert.Equal(t, entry.PluginRef, retrieved.PluginRef)
	assert.Equal(t, entry.Checksum, retrieved.Checksum)
}

func TestCreateEntry_NonExistentBinary(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	_, err := cache.CreateEntry("buf.build/library/connect-go", "v1.5.0", "/nonexistent/binary")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stat binary")
}

func TestCreateEntry_ChecksumError(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &Cache{
		cacheDir:    filepath.Join(tmpDir, "plugins"),
		metadataDir: filepath.Join(tmpDir, "metadata"),
	}

	// Create a binary file without read permissions
	binaryPath := filepath.Join(tmpDir, "test-plugin")
	err := os.WriteFile(binaryPath, []byte("content"), 0000)
	require.NoError(t, err)

	_, err = cache.CreateEntry("buf.build/library/connect-go", "v1.5.0", binaryPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to calculate checksum")
}
