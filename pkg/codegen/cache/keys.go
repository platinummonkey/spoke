package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/platinummonkey/spoke/pkg/codegen"
)

// GenerateCacheKey generates a cache key from compilation parameters
func GenerateCacheKey(moduleName, version, language, pluginVersion string, protoFiles []codegen.ProtoFile, dependencies []codegen.Dependency, options map[string]string) *codegen.CacheKey {
	return &codegen.CacheKey{
		ModuleName:    moduleName,
		Version:       version,
		Language:      language,
		PluginVersion: pluginVersion,
		ProtoHash:     generateProtoHash(protoFiles, dependencies),
		Options:       options,
	}
}

// generateProtoHash generates a SHA256 hash of all proto files and dependencies
func generateProtoHash(protoFiles []codegen.ProtoFile, dependencies []codegen.Dependency) string {
	hasher := sha256.New()

	// Hash proto files in sorted order for consistency
	sortedFiles := make([]codegen.ProtoFile, len(protoFiles))
	copy(sortedFiles, protoFiles)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Path < sortedFiles[j].Path
	})

	for _, file := range sortedFiles {
		hasher.Write([]byte(file.Path))
		hasher.Write([]byte{0}) // Separator
		hasher.Write(file.Content)
		hasher.Write([]byte{0}) // Separator
	}

	// Hash dependencies in sorted order
	sortedDeps := make([]codegen.Dependency, len(dependencies))
	copy(sortedDeps, dependencies)
	sort.Slice(sortedDeps, func(i, j int) bool {
		return sortedDeps[i].ModuleName < sortedDeps[j].ModuleName ||
			(sortedDeps[i].ModuleName == sortedDeps[j].ModuleName && sortedDeps[i].Version < sortedDeps[j].Version)
	})

	for _, dep := range sortedDeps {
		hasher.Write([]byte(dep.ModuleName))
		hasher.Write([]byte{0})
		hasher.Write([]byte(dep.Version))
		hasher.Write([]byte{0})

		// Hash dependency proto files
		depFiles := make([]codegen.ProtoFile, len(dep.ProtoFiles))
		copy(depFiles, dep.ProtoFiles)
		sort.Slice(depFiles, func(i, j int) bool {
			return depFiles[i].Path < depFiles[j].Path
		})

		for _, file := range depFiles {
			hasher.Write([]byte(file.Path))
			hasher.Write([]byte{0})
			hasher.Write(file.Content)
			hasher.Write([]byte{0})
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// FormatCacheKey formats a cache key as a string for storage
func FormatCacheKey(key *codegen.CacheKey) string {
	// Format: {moduleName}:{version}:{language}:{pluginVersion}:{protoHash}:{optionsHash}
	parts := []string{
		key.ModuleName,
		key.Version,
		key.Language,
		key.PluginVersion,
		key.ProtoHash,
	}

	// Add options hash if present
	if len(key.Options) > 0 {
		parts = append(parts, hashOptions(key.Options))
	}

	return strings.Join(parts, ":")
}

// hashOptions generates a stable hash of options map
func hashOptions(options map[string]string) string {
	if len(options) == 0 {
		return ""
	}

	// Sort keys for consistent hashing
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	hasher := sha256.New()
	for _, k := range keys {
		hasher.Write([]byte(k))
		hasher.Write([]byte{0})
		hasher.Write([]byte(options[k]))
		hasher.Write([]byte{0})
	}

	hash := hasher.Sum(nil)
	// Return first 16 characters of hex for brevity
	return hex.EncodeToString(hash)[:16]
}

// GetKeyString returns the cache key as a string (helper for codegen.CacheKey.String())
func GetKeyString(key *codegen.CacheKey) string {
	return FormatCacheKey(key)
}

// ValidateCacheKey validates a cache key
func ValidateCacheKey(key *codegen.CacheKey) error {
	if key == nil {
		return fmt.Errorf("cache key is nil")
	}
	if key.ModuleName == "" {
		return fmt.Errorf("module name is required")
	}
	if key.Version == "" {
		return fmt.Errorf("version is required")
	}
	if key.Language == "" {
		return fmt.Errorf("language is required")
	}
	if key.ProtoHash == "" {
		return fmt.Errorf("proto hash is required")
	}
	return nil
}
