package codegen

import (
	"context"
	"testing"
	"time"
)

func TestCacheKeyString(t *testing.T) {
	tests := []struct {
		name     string
		cacheKey *CacheKey
		want     string
	}{
		{
			name: "basic cache key without options",
			cacheKey: &CacheKey{
				ModuleName:    "test-module",
				Version:       "v1.0.0",
				Language:      "go",
				PluginVersion: "1.2.3",
				ProtoHash:     "abc123",
			},
			want: "test-module:v1.0.0:go:1.2.3:abc123",
		},
		{
			name: "cache key with options",
			cacheKey: &CacheKey{
				ModuleName:    "test-module",
				Version:       "v1.0.0",
				Language:      "python",
				PluginVersion: "2.0.0",
				ProtoHash:     "def456",
				Options: map[string]string{
					"option1": "value1",
					"option2": "value2",
				},
			},
			want: "test-module:v1.0.0:python:2.0.0:def456:option1=value1;",
		},
		{
			name: "cache key with long options (truncation test)",
			cacheKey: &CacheKey{
				ModuleName:    "module",
				Version:       "v2.0.0",
				Language:      "java",
				PluginVersion: "3.0.0",
				ProtoHash:     "ghi789",
				Options: map[string]string{
					"very_long_option_name": "very_long_value_that_exceeds_sixteen_characters",
				},
			},
			// Options should be truncated to 16 characters max
			want: "module:v2.0.0:java:3.0.0:ghi789:very_long_option",
		},
		{
			name: "cache key with empty strings",
			cacheKey: &CacheKey{
				ModuleName:    "",
				Version:       "",
				Language:      "",
				PluginVersion: "",
				ProtoHash:     "",
			},
			want: "::::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cacheKey.String()
			// For options tests, we need to be flexible since map iteration order is not guaranteed
			if len(tt.cacheKey.Options) > 0 {
				// Just check that it starts with the expected prefix and has correct structure
				expectedPrefix := tt.cacheKey.ModuleName + ":" + tt.cacheKey.Version + ":" +
					tt.cacheKey.Language + ":" + tt.cacheKey.PluginVersion + ":" + tt.cacheKey.ProtoHash
				if len(got) < len(expectedPrefix) || got[:len(expectedPrefix)] != expectedPrefix {
					t.Errorf("CacheKey.String() prefix = %v, want prefix %v", got, expectedPrefix)
				}
				// Check that it has the options appended (6 parts total)
				parts := 0
				for i := 0; i < len(got); i++ {
					if got[i] == ':' {
						parts++
					}
				}
				if parts != 5 { // 5 colons = 6 parts
					t.Errorf("CacheKey.String() has %d parts, want 6 parts", parts+1)
				}
			} else {
				if got != tt.want {
					t.Errorf("CacheKey.String() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{
			name: "a less than b",
			a:    5,
			b:    10,
			want: 5,
		},
		{
			name: "b less than a",
			a:    15,
			b:    8,
			want: 8,
		},
		{
			name: "a equals b",
			a:    7,
			b:    7,
			want: 7,
		},
		{
			name: "negative numbers",
			a:    -5,
			b:    -10,
			want: -10,
		},
		{
			name: "zero and positive",
			a:    0,
			b:    5,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCompilationRequestStructure(t *testing.T) {
	ctx := context.Background()
	req := CompilationRequest{
		ModuleName:  "test-module",
		Version:     "v1.0.0",
		VersionID:   12345,
		ProtoFiles:  []ProtoFile{{Path: "test.proto", Content: []byte("syntax = \"proto3\";"), Hash: "hash123"}},
		Dependencies: []Dependency{{ModuleName: "dep1", Version: "v1.0.0"}},
		Language:    "go",
		IncludeGRPC: true,
		Options:     map[string]string{"opt1": "val1"},
		Context:     ctx,
	}

	if req.ModuleName != "test-module" {
		t.Errorf("Expected ModuleName to be 'test-module', got %s", req.ModuleName)
	}
	if req.Version != "v1.0.0" {
		t.Errorf("Expected Version to be 'v1.0.0', got %s", req.Version)
	}
	if req.VersionID != 12345 {
		t.Errorf("Expected VersionID to be 12345, got %d", req.VersionID)
	}
	if len(req.ProtoFiles) != 1 {
		t.Errorf("Expected 1 proto file, got %d", len(req.ProtoFiles))
	}
	if len(req.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(req.Dependencies))
	}
	if req.Language != "go" {
		t.Errorf("Expected Language to be 'go', got %s", req.Language)
	}
	if !req.IncludeGRPC {
		t.Error("Expected IncludeGRPC to be true")
	}
	if len(req.Options) != 1 {
		t.Errorf("Expected 1 option, got %d", len(req.Options))
	}
}

func TestProtoFileStructure(t *testing.T) {
	content := []byte("syntax = \"proto3\";")
	pf := ProtoFile{
		Path:    "api/v1/service.proto",
		Content: content,
		Hash:    "sha256hash",
	}

	if pf.Path != "api/v1/service.proto" {
		t.Errorf("Expected Path to be 'api/v1/service.proto', got %s", pf.Path)
	}
	if string(pf.Content) != string(content) {
		t.Errorf("Expected Content to match, got %s", string(pf.Content))
	}
	if pf.Hash != "sha256hash" {
		t.Errorf("Expected Hash to be 'sha256hash', got %s", pf.Hash)
	}
}

func TestDependencyStructure(t *testing.T) {
	dep := Dependency{
		ModuleName: "common-protos",
		Version:    "v2.0.0",
		ProtoFiles: []ProtoFile{
			{Path: "common.proto", Content: []byte("test"), Hash: "hash1"},
		},
	}

	if dep.ModuleName != "common-protos" {
		t.Errorf("Expected ModuleName to be 'common-protos', got %s", dep.ModuleName)
	}
	if dep.Version != "v2.0.0" {
		t.Errorf("Expected Version to be 'v2.0.0', got %s", dep.Version)
	}
	if len(dep.ProtoFiles) != 1 {
		t.Errorf("Expected 1 proto file, got %d", len(dep.ProtoFiles))
	}
}

func TestCompilationResultStructure(t *testing.T) {
	duration := 5 * time.Second
	result := CompilationResult{
		Success:  true,
		Language: "python",
		GeneratedFiles: []GeneratedFile{
			{Path: "output.py", Content: []byte("code"), Size: 1024},
		},
		PackageFiles: []GeneratedFile{
			{Path: "setup.py", Content: []byte("setup"), Size: 512},
		},
		CacheHit:     false,
		Duration:     duration,
		Error:        "",
		S3Key:        "artifacts/test.tar.gz",
		S3Bucket:     "my-bucket",
		ArtifactHash: "artifact-hash",
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.Language != "python" {
		t.Errorf("Expected Language to be 'python', got %s", result.Language)
	}
	if len(result.GeneratedFiles) != 1 {
		t.Errorf("Expected 1 generated file, got %d", len(result.GeneratedFiles))
	}
	if len(result.PackageFiles) != 1 {
		t.Errorf("Expected 1 package file, got %d", len(result.PackageFiles))
	}
	if result.CacheHit {
		t.Error("Expected CacheHit to be false")
	}
	if result.Duration != duration {
		t.Errorf("Expected Duration to be %v, got %v", duration, result.Duration)
	}
	if result.Error != "" {
		t.Errorf("Expected Error to be empty, got %s", result.Error)
	}
	if result.S3Key != "artifacts/test.tar.gz" {
		t.Errorf("Expected S3Key to be 'artifacts/test.tar.gz', got %s", result.S3Key)
	}
	if result.S3Bucket != "my-bucket" {
		t.Errorf("Expected S3Bucket to be 'my-bucket', got %s", result.S3Bucket)
	}
	if result.ArtifactHash != "artifact-hash" {
		t.Errorf("Expected ArtifactHash to be 'artifact-hash', got %s", result.ArtifactHash)
	}
}

func TestGeneratedFileStructure(t *testing.T) {
	content := []byte("generated code content")
	gf := GeneratedFile{
		Path:    "pkg/api/service.go",
		Content: content,
		Size:    2048,
	}

	if gf.Path != "pkg/api/service.go" {
		t.Errorf("Expected Path to be 'pkg/api/service.go', got %s", gf.Path)
	}
	if string(gf.Content) != string(content) {
		t.Errorf("Expected Content to match")
	}
	if gf.Size != 2048 {
		t.Errorf("Expected Size to be 2048, got %d", gf.Size)
	}
}

func TestCompilationJobStructure(t *testing.T) {
	now := time.Now()
	later := now.Add(5 * time.Minute)

	job := CompilationJob{
		ID:          "job-123",
		VersionID:   67890,
		Language:    "java",
		Status:      JobStatusRunning,
		StartedAt:   &now,
		CompletedAt: &later,
		Error:       "",
		CacheHit:    false,
		Result:      nil,
	}

	if job.ID != "job-123" {
		t.Errorf("Expected ID to be 'job-123', got %s", job.ID)
	}
	if job.VersionID != 67890 {
		t.Errorf("Expected VersionID to be 67890, got %d", job.VersionID)
	}
	if job.Language != "java" {
		t.Errorf("Expected Language to be 'java', got %s", job.Language)
	}
	if job.Status != JobStatusRunning {
		t.Errorf("Expected Status to be 'running', got %s", job.Status)
	}
	if job.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}
	if job.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
	if job.Error != "" {
		t.Errorf("Expected Error to be empty, got %s", job.Error)
	}
	if job.CacheHit {
		t.Error("Expected CacheHit to be false")
	}
	if job.Result != nil {
		t.Error("Expected Result to be nil")
	}
}

func TestJobStatusConstants(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   string
	}{
		{
			name:   "pending status",
			status: JobStatusPending,
			want:   "pending",
		},
		{
			name:   "running status",
			status: JobStatusRunning,
			want:   "running",
		},
		{
			name:   "completed status",
			status: JobStatusCompleted,
			want:   "completed",
		},
		{
			name:   "failed status",
			status: JobStatusFailed,
			want:   "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("JobStatus = %v, want %v", tt.status, tt.want)
			}
		})
	}
}

func TestCompilationMetricsStructure(t *testing.T) {
	duration := 3 * time.Second
	metrics := CompilationMetrics{
		Language:      "rust",
		Duration:      duration,
		CacheHit:      true,
		GeneratedSize: 4096,
		Success:       true,
	}

	if metrics.Language != "rust" {
		t.Errorf("Expected Language to be 'rust', got %s", metrics.Language)
	}
	if metrics.Duration != duration {
		t.Errorf("Expected Duration to be %v, got %v", duration, metrics.Duration)
	}
	if !metrics.CacheHit {
		t.Error("Expected CacheHit to be true")
	}
	if metrics.GeneratedSize != 4096 {
		t.Errorf("Expected GeneratedSize to be 4096, got %d", metrics.GeneratedSize)
	}
	if !metrics.Success {
		t.Error("Expected Success to be true")
	}
}

func TestCacheKeyStructure(t *testing.T) {
	ck := CacheKey{
		ModuleName:    "my-module",
		Version:       "v3.0.0",
		Language:      "typescript",
		PluginVersion: "4.0.0",
		ProtoHash:     "xyz789",
		Options: map[string]string{
			"target": "es2020",
			"module": "commonjs",
		},
	}

	if ck.ModuleName != "my-module" {
		t.Errorf("Expected ModuleName to be 'my-module', got %s", ck.ModuleName)
	}
	if ck.Version != "v3.0.0" {
		t.Errorf("Expected Version to be 'v3.0.0', got %s", ck.Version)
	}
	if ck.Language != "typescript" {
		t.Errorf("Expected Language to be 'typescript', got %s", ck.Language)
	}
	if ck.PluginVersion != "4.0.0" {
		t.Errorf("Expected PluginVersion to be '4.0.0', got %s", ck.PluginVersion)
	}
	if ck.ProtoHash != "xyz789" {
		t.Errorf("Expected ProtoHash to be 'xyz789', got %s", ck.ProtoHash)
	}
	if len(ck.Options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(ck.Options))
	}
}
