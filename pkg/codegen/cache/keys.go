// Package cache provides cache key generation and formatting for compilation results
//
// CRITICAL INVARIANT: SORTED ORDER REQUIREMENT
// All cache keys MUST be generated with inputs sorted in a consistent order.
// This ensures that identical compilations produce identical cache keys regardless
// of input order.
//
// Sorted components:
//   - Proto files: sorted by Path (string comparison)
//   - Dependencies: sorted by ModuleName, then Version
//   - Dependency proto files: sorted by Path
//   - Options map: keys sorted alphabetically before hashing
//
// CHANGING THESE SORT ORDERS WILL INVALIDATE THE ENTIRE CACHE
//
// Cache Key Format Version: v1
// Format: {moduleName}:{version}:{language}:{pluginVersion}:{protoHash}:{optionsHash}
//
// DO NOT modify generateProtoHash(), hashOptions(), or FormatCacheKey() without:
// 1. Incrementing cache format version
// 2. Clearing all existing cached compilations
// 3. Updating cache migration documentation
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
//
// IMPORTANT: This function ensures deterministic key generation by sorting all inputs.
// DO NOT call with pre-sorted inputs assuming you're helping - the sorting is intentional
// and must happen in this function to guarantee consistency.
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
//
// CRITICAL INVARIANT: All inputs are sorted before hashing to ensure deterministic results.
// Hash format version: v1
//
// Algorithm:
// 1. Sort proto files by Path
// 2. Hash each file: path + \0 + content + \0
// 3. Sort dependencies by (ModuleName, Version)
// 4. For each dependency:
//    - Hash: moduleName + \0 + version + \0
//    - Sort dependency proto files by Path
//    - Hash each file: path + \0 + content + \0
//
// CHANGING THIS ALGORITHM INVALIDATES ALL CACHED COMPILATIONS
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
//
// Cache Key Format Version: v1
// Format: {moduleName}:{version}:{language}:{pluginVersion}:{protoHash}:{optionsHash}
//
// CRITICAL: This is the single source of truth for cache key string format.
// The codegen.CacheKey.String() method delegates to this function.
//
// CHANGING THIS FORMAT INVALIDATES ALL CACHED COMPILATIONS
// If you must change it:
// 1. Increment version number in package documentation
// 2. Clear all cached data (or implement migration)
// 3. Update cache documentation
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
//
// CRITICAL INVARIANT: Map keys are sorted alphabetically before hashing.
// This ensures identical options produce identical hashes regardless of iteration order.
//
// Algorithm:
// 1. Extract and sort all keys alphabetically
// 2. For each key in sorted order: hash key + \0 + value + \0
// 3. Return first 16 hex characters of SHA256 hash
//
// CHANGING THIS ALGORITHM INVALIDATES ALL CACHED COMPILATIONS WITH OPTIONS
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
