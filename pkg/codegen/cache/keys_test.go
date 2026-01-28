package cache

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/codegen"
)

func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		name          string
		moduleName    string
		version       string
		language      string
		pluginVersion string
		protoFiles    []codegen.ProtoFile
		dependencies  []codegen.Dependency
		options       map[string]string
	}{
		{
			name:          "basic cache key",
			moduleName:    "test-module",
			version:       "v1.0.0",
			language:      "go",
			pluginVersion: "v1.2.3",
			protoFiles: []codegen.ProtoFile{
				{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
			},
			dependencies: nil,
			options:      nil,
		},
		{
			name:          "with multiple proto files",
			moduleName:    "multi-module",
			version:       "v2.0.0",
			language:      "python",
			pluginVersion: "v2.0.0",
			protoFiles: []codegen.ProtoFile{
				{Path: "a.proto", Content: []byte("message A {}")},
				{Path: "b.proto", Content: []byte("message B {}")},
			},
			dependencies: nil,
			options:      nil,
		},
		{
			name:          "with dependencies",
			moduleName:    "dep-module",
			version:       "v1.5.0",
			language:      "java",
			pluginVersion: "v1.0.0",
			protoFiles: []codegen.ProtoFile{
				{Path: "main.proto", Content: []byte("syntax = \"proto3\";")},
			},
			dependencies: []codegen.Dependency{
				{
					ModuleName: "dep1",
					Version:    "v1.0.0",
					ProtoFiles: []codegen.ProtoFile{
						{Path: "dep.proto", Content: []byte("message Dep {}")},
					},
				},
			},
			options: nil,
		},
		{
			name:          "with options",
			moduleName:    "opts-module",
			version:       "v1.0.0",
			language:      "typescript",
			pluginVersion: "v3.0.0",
			protoFiles: []codegen.ProtoFile{
				{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
			},
			dependencies: nil,
			options: map[string]string{
				"optimize_for": "SPEED",
				"go_package":   "github.com/test/pkg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenerateCacheKey(
				tt.moduleName,
				tt.version,
				tt.language,
				tt.pluginVersion,
				tt.protoFiles,
				tt.dependencies,
				tt.options,
			)

			if key == nil {
				t.Fatal("expected non-nil cache key")
			}
			if key.ModuleName != tt.moduleName {
				t.Errorf("expected ModuleName %q, got %q", tt.moduleName, key.ModuleName)
			}
			if key.Version != tt.version {
				t.Errorf("expected Version %q, got %q", tt.version, key.Version)
			}
			if key.Language != tt.language {
				t.Errorf("expected Language %q, got %q", tt.language, key.Language)
			}
			if key.PluginVersion != tt.pluginVersion {
				t.Errorf("expected PluginVersion %q, got %q", tt.pluginVersion, key.PluginVersion)
			}
			if key.ProtoHash == "" {
				t.Error("expected non-empty ProtoHash")
			}
			if tt.options != nil && key.Options == nil {
				t.Error("expected non-nil Options")
			}
		})
	}
}

func TestGenerateProtoHash(t *testing.T) {
	tests := []struct {
		name         string
		protoFiles   []codegen.ProtoFile
		dependencies []codegen.Dependency
		expectSame   bool
		compareWith  *struct {
			protoFiles   []codegen.ProtoFile
			dependencies []codegen.Dependency
		}
	}{
		{
			name: "empty inputs",
			protoFiles: []codegen.ProtoFile{},
			dependencies: []codegen.Dependency{},
		},
		{
			name: "single proto file",
			protoFiles: []codegen.ProtoFile{
				{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
			},
			dependencies: nil,
		},
		{
			name: "multiple proto files - order matters",
			protoFiles: []codegen.ProtoFile{
				{Path: "b.proto", Content: []byte("message B {}")},
				{Path: "a.proto", Content: []byte("message A {}")},
			},
			dependencies: nil,
			expectSame: true,
			compareWith: &struct {
				protoFiles   []codegen.ProtoFile
				dependencies []codegen.Dependency
			}{
				protoFiles: []codegen.ProtoFile{
					{Path: "a.proto", Content: []byte("message A {}")},
					{Path: "b.proto", Content: []byte("message B {}")},
				},
				dependencies: nil,
			},
		},
		{
			name: "with dependencies",
			protoFiles: []codegen.ProtoFile{
				{Path: "main.proto", Content: []byte("syntax = \"proto3\";")},
			},
			dependencies: []codegen.Dependency{
				{
					ModuleName: "dep1",
					Version:    "v1.0.0",
					ProtoFiles: []codegen.ProtoFile{
						{Path: "dep.proto", Content: []byte("message Dep {}")},
					},
				},
			},
		},
		{
			name: "with dependency having multiple proto files",
			protoFiles: []codegen.ProtoFile{
				{Path: "main.proto", Content: []byte("syntax = \"proto3\";")},
			},
			dependencies: []codegen.Dependency{
				{
					ModuleName: "dep1",
					Version:    "v1.0.0",
					ProtoFiles: []codegen.ProtoFile{
						{Path: "z.proto", Content: []byte("message Z {}")},
						{Path: "a.proto", Content: []byte("message A {}")},
					},
				},
			},
		},
		{
			name: "multiple dependencies - order consistency",
			protoFiles: []codegen.ProtoFile{
				{Path: "main.proto", Content: []byte("syntax = \"proto3\";")},
			},
			dependencies: []codegen.Dependency{
				{
					ModuleName: "dep2",
					Version:    "v2.0.0",
					ProtoFiles: []codegen.ProtoFile{
						{Path: "dep2.proto", Content: []byte("message Dep2 {}")},
					},
				},
				{
					ModuleName: "dep1",
					Version:    "v1.0.0",
					ProtoFiles: []codegen.ProtoFile{
						{Path: "dep1.proto", Content: []byte("message Dep1 {}")},
					},
				},
			},
			expectSame: true,
			compareWith: &struct {
				protoFiles   []codegen.ProtoFile
				dependencies []codegen.Dependency
			}{
				protoFiles: []codegen.ProtoFile{
					{Path: "main.proto", Content: []byte("syntax = \"proto3\";")},
				},
				dependencies: []codegen.Dependency{
					{
						ModuleName: "dep1",
						Version:    "v1.0.0",
						ProtoFiles: []codegen.ProtoFile{
							{Path: "dep1.proto", Content: []byte("message Dep1 {}")},
						},
					},
					{
						ModuleName: "dep2",
						Version:    "v2.0.0",
						ProtoFiles: []codegen.ProtoFile{
							{Path: "dep2.proto", Content: []byte("message Dep2 {}")},
						},
					},
				},
			},
		},
		{
			name: "dependency with multiple proto files - order consistency",
			protoFiles: []codegen.ProtoFile{
				{Path: "main.proto", Content: []byte("syntax = \"proto3\";")},
			},
			dependencies: []codegen.Dependency{
				{
					ModuleName: "dep1",
					Version:    "v1.0.0",
					ProtoFiles: []codegen.ProtoFile{
						{Path: "z.proto", Content: []byte("message Z {}")},
						{Path: "a.proto", Content: []byte("message A {}")},
					},
				},
			},
			expectSame: true,
			compareWith: &struct {
				protoFiles   []codegen.ProtoFile
				dependencies []codegen.Dependency
			}{
				protoFiles: []codegen.ProtoFile{
					{Path: "main.proto", Content: []byte("syntax = \"proto3\";")},
				},
				dependencies: []codegen.Dependency{
					{
						ModuleName: "dep1",
						Version:    "v1.0.0",
						ProtoFiles: []codegen.ProtoFile{
							{Path: "a.proto", Content: []byte("message A {}")},
							{Path: "z.proto", Content: []byte("message Z {}")},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := generateProtoHash(tt.protoFiles, tt.dependencies)
			if hash == "" {
				t.Error("expected non-empty hash")
			}
			if len(hash) != 64 { // SHA256 hex string length
				t.Errorf("expected hash length 64, got %d", len(hash))
			}

			if tt.compareWith != nil {
				compareHash := generateProtoHash(tt.compareWith.protoFiles, tt.compareWith.dependencies)
				if tt.expectSame && hash != compareHash {
					t.Errorf("expected same hash for reordered inputs, got %s vs %s", hash, compareHash)
				} else if !tt.expectSame && hash == compareHash {
					t.Error("expected different hashes")
				}
			}
		})
	}
}

func TestFormatCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		key      *codegen.CacheKey
		expected string
	}{
		{
			name: "basic key without options",
			key: &codegen.CacheKey{
				ModuleName:    "test-module",
				Version:       "v1.0.0",
				Language:      "go",
				PluginVersion: "v1.2.3",
				ProtoHash:     "abcd1234",
				Options:       nil,
			},
			expected: "test-module:v1.0.0:go:v1.2.3:abcd1234",
		},
		{
			name: "key with empty options",
			key: &codegen.CacheKey{
				ModuleName:    "test-module",
				Version:       "v1.0.0",
				Language:      "go",
				PluginVersion: "v1.2.3",
				ProtoHash:     "abcd1234",
				Options:       map[string]string{},
			},
			expected: "test-module:v1.0.0:go:v1.2.3:abcd1234",
		},
		{
			name: "key with options",
			key: &codegen.CacheKey{
				ModuleName:    "test-module",
				Version:       "v1.0.0",
				Language:      "go",
				PluginVersion: "v1.2.3",
				ProtoHash:     "abcd1234",
				Options: map[string]string{
					"optimize_for": "SPEED",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCacheKey(tt.key)
			if result == "" {
				t.Error("expected non-empty formatted key")
			}
			if tt.expected != "" && result != tt.expected {
				if len(tt.key.Options) == 0 {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestHashOptions(t *testing.T) {
	tests := []struct {
		name       string
		options    map[string]string
		expectSame bool
		compareWith map[string]string
	}{
		{
			name:    "empty options",
			options: map[string]string{},
		},
		{
			name:    "nil options",
			options: nil,
		},
		{
			name: "single option",
			options: map[string]string{
				"key1": "value1",
			},
		},
		{
			name: "multiple options - order independence",
			options: map[string]string{
				"key2": "value2",
				"key1": "value1",
			},
			expectSame: true,
			compareWith: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "different values",
			options: map[string]string{
				"key1": "value1",
			},
			expectSame: false,
			compareWith: map[string]string{
				"key1": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashOptions(tt.options)
			if len(tt.options) == 0 && hash != "" {
				t.Errorf("expected empty hash for empty options, got %q", hash)
			}
			if len(tt.options) > 0 {
				if hash == "" {
					t.Error("expected non-empty hash for non-empty options")
				}
				if len(hash) != 16 {
					t.Errorf("expected hash length 16, got %d", len(hash))
				}
			}

			if tt.compareWith != nil {
				compareHash := hashOptions(tt.compareWith)
				if tt.expectSame && hash != compareHash {
					t.Errorf("expected same hash, got %s vs %s", hash, compareHash)
				} else if !tt.expectSame && hash == compareHash {
					t.Error("expected different hashes")
				}
			}
		})
	}
}

func TestGetKeyString(t *testing.T) {
	key := &codegen.CacheKey{
		ModuleName:    "test-module",
		Version:       "v1.0.0",
		Language:      "go",
		PluginVersion: "v1.2.3",
		ProtoHash:     "abcd1234",
		Options:       nil,
	}

	result := GetKeyString(key)
	expected := FormatCacheKey(key)

	if result != expected {
		t.Errorf("expected GetKeyString to match FormatCacheKey, got %q vs %q", result, expected)
	}
}

func TestValidateCacheKey(t *testing.T) {
	tests := []struct {
		name      string
		key       *codegen.CacheKey
		expectErr bool
		errMsg    string
	}{
		{
			name:      "nil key",
			key:       nil,
			expectErr: true,
			errMsg:    "cache key is nil",
		},
		{
			name: "missing module name",
			key: &codegen.CacheKey{
				ModuleName:    "",
				Version:       "v1.0.0",
				Language:      "go",
				PluginVersion: "v1.2.3",
				ProtoHash:     "abcd1234",
			},
			expectErr: true,
			errMsg:    "module name is required",
		},
		{
			name: "missing version",
			key: &codegen.CacheKey{
				ModuleName:    "test-module",
				Version:       "",
				Language:      "go",
				PluginVersion: "v1.2.3",
				ProtoHash:     "abcd1234",
			},
			expectErr: true,
			errMsg:    "version is required",
		},
		{
			name: "missing language",
			key: &codegen.CacheKey{
				ModuleName:    "test-module",
				Version:       "v1.0.0",
				Language:      "",
				PluginVersion: "v1.2.3",
				ProtoHash:     "abcd1234",
			},
			expectErr: true,
			errMsg:    "language is required",
		},
		{
			name: "missing proto hash",
			key: &codegen.CacheKey{
				ModuleName:    "test-module",
				Version:       "v1.0.0",
				Language:      "go",
				PluginVersion: "v1.2.3",
				ProtoHash:     "",
			},
			expectErr: true,
			errMsg:    "proto hash is required",
		},
		{
			name: "valid key without plugin version",
			key: &codegen.CacheKey{
				ModuleName:    "test-module",
				Version:       "v1.0.0",
				Language:      "go",
				PluginVersion: "",
				ProtoHash:     "abcd1234",
			},
			expectErr: false,
		},
		{
			name: "valid key with all fields",
			key: &codegen.CacheKey{
				ModuleName:    "test-module",
				Version:       "v1.0.0",
				Language:      "go",
				PluginVersion: "v1.2.3",
				ProtoHash:     "abcd1234",
				Options: map[string]string{
					"key": "value",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCacheKey(tt.key)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestHashConsistency(t *testing.T) {
	// Test that hash generation is consistent across multiple calls
	protoFiles := []codegen.ProtoFile{
		{Path: "test.proto", Content: []byte("syntax = \"proto3\";")},
	}
	dependencies := []codegen.Dependency{
		{
			ModuleName: "dep1",
			Version:    "v1.0.0",
			ProtoFiles: []codegen.ProtoFile{
				{Path: "dep.proto", Content: []byte("message Dep {}")},
			},
		},
	}

	hash1 := generateProtoHash(protoFiles, dependencies)
	hash2 := generateProtoHash(protoFiles, dependencies)

	if hash1 != hash2 {
		t.Errorf("expected consistent hashes, got %s vs %s", hash1, hash2)
	}
}

func TestCacheKeyIntegration(t *testing.T) {
	// Integration test: generate, format, and validate a cache key
	protoFiles := []codegen.ProtoFile{
		{Path: "api.proto", Content: []byte("syntax = \"proto3\";")},
		{Path: "types.proto", Content: []byte("message Type {}")},
	}
	dependencies := []codegen.Dependency{
		{
			ModuleName: "common",
			Version:    "v1.0.0",
			ProtoFiles: []codegen.ProtoFile{
				{Path: "common.proto", Content: []byte("message Common {}")},
			},
		},
	}
	options := map[string]string{
		"go_package": "github.com/test/api",
	}

	key := GenerateCacheKey(
		"test-api",
		"v2.0.0",
		"go",
		"v1.5.0",
		protoFiles,
		dependencies,
		options,
	)

	if err := ValidateCacheKey(key); err != nil {
		t.Errorf("expected valid cache key, got error: %v", err)
	}

	formatted := FormatCacheKey(key)
	if formatted == "" {
		t.Error("expected non-empty formatted key")
	}

	keyString := GetKeyString(key)
	if keyString != formatted {
		t.Error("expected GetKeyString to match FormatCacheKey")
	}
}
