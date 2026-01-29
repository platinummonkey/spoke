package codegen

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// hashProtoFiles generates a SHA256 hash of proto files and dependencies
func hashProtoFiles(protoFiles []ProtoFile, dependencies []Dependency) string {
	hasher := sha256.New()

	// Hash proto files in sorted order
	sortedFiles := make([]ProtoFile, len(protoFiles))
	copy(sortedFiles, protoFiles)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Path < sortedFiles[j].Path
	})

	for _, file := range sortedFiles {
		hasher.Write([]byte(file.Path))
		hasher.Write([]byte{0})
		hasher.Write(file.Content)
		hasher.Write([]byte{0})
	}

	// Hash dependencies in sorted order
	sortedDeps := make([]Dependency, len(dependencies))
	copy(sortedDeps, dependencies)
	sort.Slice(sortedDeps, func(i, j int) bool {
		if sortedDeps[i].ModuleName != sortedDeps[j].ModuleName {
			return sortedDeps[i].ModuleName < sortedDeps[j].ModuleName
		}
		return sortedDeps[i].Version < sortedDeps[j].Version
	})

	for _, dep := range sortedDeps {
		hasher.Write([]byte(dep.ModuleName))
		hasher.Write([]byte{0})
		hasher.Write([]byte(dep.Version))
		hasher.Write([]byte{0})

		depFiles := make([]ProtoFile, len(dep.ProtoFiles))
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

// ClearCache clears the global memory cache
func ClearCache() {
	globalCache.Range(func(key, value interface{}) bool {
		globalCache.Delete(key)
		return true
	})
}
