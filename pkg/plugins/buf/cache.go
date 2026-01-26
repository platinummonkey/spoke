package buf

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheEntry represents a cached plugin entry
type CacheEntry struct {
	PluginRef    string    `json:"plugin_ref"`
	Version      string    `json:"version"`
	BinaryPath   string    `json:"binary_path"`
	DownloadedAt time.Time `json:"downloaded_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
	Checksum     string    `json:"checksum"`
	Size         int64     `json:"size"`
}

// Cache manages the plugin cache
type Cache struct {
	cacheDir    string
	metadataDir string
}

// NewCache creates a new cache manager
func NewCache() *Cache {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".buf", "plugins")
	metadataDir := filepath.Join(homeDir, ".buf", "cache-metadata")

	return &Cache{
		cacheDir:    cacheDir,
		metadataDir: metadataDir,
	}
}

// GetEntry retrieves a cache entry for a plugin
func (c *Cache) GetEntry(pluginRef, version string) (*CacheEntry, error) {
	metadataPath := c.getMetadataPath(pluginRef, version)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache entry not found")
		}
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to parse cache entry: %w", err)
	}

	return &entry, nil
}

// SaveEntry saves a cache entry for a plugin
func (c *Cache) SaveEntry(entry *CacheEntry) error {
	if err := os.MkdirAll(c.metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	metadataPath := c.getMetadataPath(entry.PluginRef, entry.Version)

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache entry: %w", err)
	}

	return nil
}

// UpdateLastUsed updates the last used timestamp for a cached plugin
func (c *Cache) UpdateLastUsed(pluginRef, version string) error {
	entry, err := c.GetEntry(pluginRef, version)
	if err != nil {
		return err
	}

	entry.LastUsedAt = time.Now()
	return c.SaveEntry(entry)
}

// ListEntries returns all cache entries
func (c *Cache) ListEntries() ([]*CacheEntry, error) {
	if _, err := os.Stat(c.metadataDir); os.IsNotExist(err) {
		return []*CacheEntry{}, nil
	}

	var entries []*CacheEntry

	err := filepath.Walk(c.metadataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		var entry CacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil // Skip invalid entries
		}

		entries = append(entries, &entry)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return entries, nil
}

// RemoveEntry removes a cache entry and its associated binary
func (c *Cache) RemoveEntry(pluginRef, version string) error {
	// Get entry to find binary path
	entry, err := c.GetEntry(pluginRef, version)
	if err == nil {
		// Remove binary
		if entry.BinaryPath != "" {
			os.Remove(entry.BinaryPath)
			// Try to remove parent directory if empty
			os.Remove(filepath.Dir(entry.BinaryPath))
		}
	}

	// Remove metadata
	metadataPath := c.getMetadataPath(pluginRef, version)
	return os.Remove(metadataPath)
}

// Clear removes all cache entries and binaries
func (c *Cache) Clear() error {
	// Remove cache directory
	if err := os.RemoveAll(c.cacheDir); err != nil {
		return err
	}

	// Remove metadata directory
	return os.RemoveAll(c.metadataDir)
}

// GetTotalSize returns the total size of all cached binaries
func (c *Cache) GetTotalSize() (int64, error) {
	var totalSize int64

	entries, err := c.ListEntries()
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		totalSize += entry.Size
	}

	return totalSize, nil
}

// PruneOldEntries removes entries that haven't been used in the specified duration
func (c *Cache) PruneOldEntries(maxAge time.Duration) (int, error) {
	entries, err := c.ListEntries()
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-maxAge)
	pruned := 0

	for _, entry := range entries {
		if entry.LastUsedAt.Before(cutoff) {
			if err := c.RemoveEntry(entry.PluginRef, entry.Version); err == nil {
				pruned++
			}
		}
	}

	return pruned, nil
}

// VerifyIntegrity verifies that cached binaries match their checksums
func (c *Cache) VerifyIntegrity() ([]string, error) {
	var corrupted []string

	entries, err := c.ListEntries()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		// Check if binary exists
		if _, err := os.Stat(entry.BinaryPath); os.IsNotExist(err) {
			corrupted = append(corrupted, fmt.Sprintf("%s@%s (missing binary)", entry.PluginRef, entry.Version))
			continue
		}

		// Verify checksum
		actualChecksum, err := c.calculateChecksum(entry.BinaryPath)
		if err != nil {
			corrupted = append(corrupted, fmt.Sprintf("%s@%s (checksum error)", entry.PluginRef, entry.Version))
			continue
		}

		if actualChecksum != entry.Checksum {
			corrupted = append(corrupted, fmt.Sprintf("%s@%s (checksum mismatch)", entry.PluginRef, entry.Version))
		}
	}

	return corrupted, nil
}

// getMetadataPath returns the path to the metadata file for a plugin
func (c *Cache) getMetadataPath(pluginRef, version string) string {
	// Create a safe filename from plugin ref and version
	hash := sha256.Sum256([]byte(pluginRef + "@" + version))
	filename := hex.EncodeToString(hash[:]) + ".json"
	return filepath.Join(c.metadataDir, filename)
}

// calculateChecksum calculates the SHA-256 checksum of a file
func (c *Cache) calculateChecksum(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// CreateEntry creates a new cache entry from a downloaded binary
func (c *Cache) CreateEntry(pluginRef, version, binaryPath string) (*CacheEntry, error) {
	// Get file info
	info, err := os.Stat(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat binary: %w", err)
	}

	// Calculate checksum
	checksum, err := c.calculateChecksum(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	entry := &CacheEntry{
		PluginRef:    pluginRef,
		Version:      version,
		BinaryPath:   binaryPath,
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
		Checksum:     checksum,
		Size:         info.Size(),
	}

	if err := c.SaveEntry(entry); err != nil {
		return nil, err
	}

	return entry, nil
}
