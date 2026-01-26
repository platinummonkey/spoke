package buf

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Downloader downloads Buf plugins from the Buf registry
type Downloader struct {
	registryURL string
	cacheDir    string
	httpClient  *http.Client
}

// NewDownloader creates a new Buf plugin downloader
func NewDownloader() *Downloader {
	homeDir, _ := os.UserHomeDir()
	return &Downloader{
		registryURL: "https://buf.build",
		cacheDir:    filepath.Join(homeDir, ".buf", "plugins"),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Download downloads a Buf plugin binary from the registry
// Returns the path to the downloaded binary
func (d *Downloader) Download(pluginRef, version string) (string, error) {
	// Parse plugin reference
	// Example: buf.build/library/connect-go
	parts := strings.Split(pluginRef, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid plugin reference: %s (expected format: buf.build/owner/name)", pluginRef)
	}

	owner := parts[len(parts)-2]
	pluginName := parts[len(parts)-1]

	// Determine platform
	platform := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	// Build download URL
	// Buf registry URL format: https://buf.build/owner/name/plugins/version/platform.zip
	downloadURL := fmt.Sprintf("%s/%s/%s/plugins/%s/%s.zip",
		d.registryURL, owner, pluginName, version, platform)

	// Create cache directory for this plugin
	cacheDir := filepath.Join(d.cacheDir, pluginName, version)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Expected binary path
	binaryName := "protoc-gen-" + pluginName
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(cacheDir, binaryName)

	// If already downloaded, return cached path
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	// Download the plugin
	resp, err := d.httpClient.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %d for %s", resp.StatusCode, downloadURL)
	}

	// Save to temporary file
	tempFile, err := os.CreateTemp("", "buf-plugin-*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy response to temp file
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return "", fmt.Errorf("failed to save plugin: %w", err)
	}

	// Close temp file before extracting
	tempFile.Close()

	// Extract the archive
	if err := d.extractArchive(tempFile.Name(), cacheDir, binaryName); err != nil {
		return "", fmt.Errorf("failed to extract plugin: %w", err)
	}

	// Verify binary was extracted
	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("plugin binary not found after extraction: %s", binaryPath)
	}

	// Make binary executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	return binaryPath, nil
}

// extractArchive extracts a plugin archive to the destination directory
func (d *Downloader) extractArchive(archivePath, destDir, binaryName string) error {
	// Determine archive type from file extension
	if strings.HasSuffix(archivePath, ".zip") {
		return d.extractZip(archivePath, destDir, binaryName)
	} else if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		return d.extractTarGz(archivePath, destDir, binaryName)
	}

	// Try zip first (Buf uses zip format)
	if err := d.extractZip(archivePath, destDir, binaryName); err == nil {
		return nil
	}

	// Try tar.gz
	return d.extractTarGz(archivePath, destDir, binaryName)
}

// extractZip extracts a zip archive
func (d *Downloader) extractZip(archivePath, destDir, binaryName string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Look for the binary file
		if filepath.Base(f.Name) == binaryName || strings.HasSuffix(f.Name, binaryName) {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			destPath := filepath.Join(destDir, binaryName)
			outFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, rc); err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("binary %s not found in archive", binaryName)
}

// extractTarGz extracts a tar.gz archive
func (d *Downloader) extractTarGz(archivePath, destDir, binaryName string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for the binary file
		if filepath.Base(header.Name) == binaryName || strings.HasSuffix(header.Name, binaryName) {
			destPath := filepath.Join(destDir, binaryName)
			outFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("binary %s not found in archive", binaryName)
}

// ClearCache removes all cached plugins
func (d *Downloader) ClearCache() error {
	return os.RemoveAll(d.cacheDir)
}

// GetCacheSize returns the total size of cached plugins in bytes
func (d *Downloader) GetCacheSize() (int64, error) {
	var size int64

	err := filepath.Walk(d.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// ListCachedPlugins returns a list of cached plugin references and versions
func (d *Downloader) ListCachedPlugins() ([]string, error) {
	var plugins []string

	if _, err := os.Stat(d.cacheDir); os.IsNotExist(err) {
		return plugins, nil
	}

	entries, err := os.ReadDir(d.cacheDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginName := entry.Name()
		versionsDir := filepath.Join(d.cacheDir, pluginName)

		versionEntries, err := os.ReadDir(versionsDir)
		if err != nil {
			continue
		}

		for _, versionEntry := range versionEntries {
			if versionEntry.IsDir() {
				plugins = append(plugins, fmt.Sprintf("%s@%s", pluginName, versionEntry.Name()))
			}
		}
	}

	return plugins, nil
}
