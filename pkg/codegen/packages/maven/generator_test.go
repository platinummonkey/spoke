package maven

import (
	"strings"
	"testing"

	"github.com/platinummonkey/spoke/pkg/codegen/packages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_GetName(t *testing.T) {
	gen := NewGenerator()
	assert.Equal(t, "maven", gen.GetName())
}

func TestGenerator_GetConfigFiles(t *testing.T) {
	gen := NewGenerator()
	files := gen.GetConfigFiles()
	assert.ElementsMatch(t, []string{"pom.xml", "README.md"}, files)
}

func TestGenerator_Generate(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:   "test-service",
		Version:      "1.0.0",
		Language:     "java",
		IncludeGRPC:  true,
		Dependencies: []packages.Dependency{
			{Name: "common-protos", Version: "1.5.0", ImportPath: "com.spoke.generated:common-protos"},
		},
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)
	assert.Len(t, files, 2)

	// Find pom.xml file
	var pomXML *string
	var readme *string
	for _, f := range files {
		content := string(f.Content)
		if f.Path == "pom.xml" {
			pomXML = &content
		} else if f.Path == "README.md" {
			readme = &content
		}
	}

	require.NotNil(t, pomXML)
	require.NotNil(t, readme)

	// Verify pom.xml content
	assert.Contains(t, *pomXML, "<groupId>com.spoke.generated</groupId>")
	assert.Contains(t, *pomXML, "<artifactId>test-service</artifactId>")
	assert.Contains(t, *pomXML, "<version>1.0.0</version>")
	assert.Contains(t, *pomXML, "protobuf-java")
	assert.Contains(t, *pomXML, "grpc-protobuf")
	assert.Contains(t, *pomXML, "grpc-stub")

	// Verify README content
	assert.Contains(t, *readme, "# test-service")
	assert.Contains(t, *readme, "Protocol Buffer generated code for Java")
	assert.Contains(t, *readme, "<artifactId>test-service</artifactId>")
	assert.Contains(t, *readme, "<version>1.0.0</version>")
}

func TestGenerator_Generate_NoGRPC(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "test-service",
		Version:     "1.0.0",
		Language:    "java",
		IncludeGRPC: false,
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)

	// Find pom.xml file
	var pomXML *string
	for _, f := range files {
		if f.Path == "pom.xml" {
			content := string(f.Content)
			pomXML = &content
			break
		}
	}

	require.NotNil(t, pomXML)
	assert.NotContains(t, *pomXML, "grpc-protobuf")
	assert.NotContains(t, *pomXML, "grpc-stub")
	assert.Contains(t, *pomXML, "protobuf-java")
}

func TestGenerator_Generate_WithDependencies(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "test-service",
		Version:     "1.0.0",
		Language:    "java",
		IncludeGRPC: false,
		Dependencies: []packages.Dependency{
			{Name: "com.example:common-lib", Version: "2.0.0"},
			{Name: "simple-dep", Version: "1.5.0"},
		},
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)

	// Find pom.xml file
	var pomXML *string
	for _, f := range files {
		if f.Path == "pom.xml" {
			content := string(f.Content)
			pomXML = &content
			break
		}
	}

	require.NotNil(t, pomXML)

	// Verify dependencies are included
	assert.Contains(t, *pomXML, "<groupId>com.example</groupId>")
	assert.Contains(t, *pomXML, "<artifactId>common-lib</artifactId>")
	assert.Contains(t, *pomXML, "<version>2.0.0</version>")

	// Simple dependency should use default groupId
	assert.Contains(t, *pomXML, "<groupId>com.spoke.generated</groupId>")
	assert.Contains(t, *pomXML, "<artifactId>simple-dep</artifactId>")
	assert.Contains(t, *pomXML, "<version>1.5.0</version>")
}

func TestConvertToArtifactId(t *testing.T) {
	tests := []struct {
		name       string
		moduleName string
		expected   string
	}{
		{
			name:       "simple name",
			moduleName: "UserService",
			expected:   "userservice",
		},
		{
			name:       "with hyphens",
			moduleName: "user-service",
			expected:   "user-service",
		},
		{
			name:       "with underscores",
			moduleName: "user_service",
			expected:   "user-service",
		},
		{
			name:       "with spaces",
			moduleName: "user service",
			expected:   "user-service",
		},
		{
			name:       "mixed case and separators",
			moduleName: "User_Service API",
			expected:   "user-service-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToArtifactId(tt.moduleName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertDependencies(t *testing.T) {
	tests := []struct {
		name     string
		deps     []packages.Dependency
		expected []map[string]string
	}{
		{
			name: "empty dependencies",
			deps: []packages.Dependency{},
			expected: []map[string]string{},
		},
		{
			name: "simple dependency name",
			deps: []packages.Dependency{
				{Name: "common-protos", Version: "1.0.0"},
			},
			expected: []map[string]string{
				{
					"GroupId":    "com.spoke.generated",
					"ArtifactId": "common-protos",
					"Version":    "1.0.0",
				},
			},
		},
		{
			name: "dependency with groupId:artifactId",
			deps: []packages.Dependency{
				{Name: "com.example:my-lib", Version: "2.5.0"},
			},
			expected: []map[string]string{
				{
					"GroupId":    "com.example",
					"ArtifactId": "my-lib",
					"Version":    "2.5.0",
				},
			},
		},
		{
			name: "multiple dependencies mixed format",
			deps: []packages.Dependency{
				{Name: "simple-dep", Version: "1.0.0"},
				{Name: "com.foo:bar-lib", Version: "3.0.0"},
			},
			expected: []map[string]string{
				{
					"GroupId":    "com.spoke.generated",
					"ArtifactId": "simple-dep",
					"Version":    "1.0.0",
				},
				{
					"GroupId":    "com.foo",
					"ArtifactId": "bar-lib",
					"Version":    "3.0.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertDependencies(tt.deps)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerator_NewGenerator(t *testing.T) {
	gen := NewGenerator()
	assert.NotNil(t, gen)
	assert.IsType(t, &Generator{}, gen)
}

func TestGenerator_GeneratePomXML(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName:  "my-service",
		Version:     "2.0.0-SNAPSHOT",
		IncludeGRPC: true,
		Dependencies: []packages.Dependency{
			{Name: "org.example:dependency", Version: "1.2.3"},
		},
	}

	file, err := gen.generatePomXML(req)
	require.NoError(t, err)

	assert.Equal(t, "pom.xml", file.Path)
	assert.Greater(t, file.Size, int64(0))
	assert.Equal(t, int64(len(file.Content)), file.Size)

	content := string(file.Content)

	// Verify XML structure
	assert.True(t, strings.HasPrefix(content, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"))
	assert.Contains(t, content, "<project xmlns=\"http://maven.apache.org/POM/4.0.0\"")
	assert.Contains(t, content, "<modelVersion>4.0.0</modelVersion>")

	// Verify module info
	assert.Contains(t, content, "<artifactId>my-service</artifactId>")
	assert.Contains(t, content, "<version>2.0.0-SNAPSHOT</version>")
	assert.Contains(t, content, "<name>my-service</name>")

	// Verify protobuf and gRPC dependencies
	assert.Contains(t, content, "protobuf.version")
	assert.Contains(t, content, "grpc.version")

	// Verify custom dependency
	assert.Contains(t, content, "<groupId>org.example</groupId>")
	assert.Contains(t, content, "<artifactId>dependency</artifactId>")
	assert.Contains(t, content, "<version>1.2.3</version>")
}

func TestGenerator_GenerateReadme(t *testing.T) {
	gen := NewGenerator()

	req := &packages.GenerateRequest{
		ModuleName: "test-api",
		Version:    "3.1.4",
	}

	files, err := gen.Generate(req)
	require.NoError(t, err)

	var readme *string
	for _, f := range files {
		if f.Path == "README.md" {
			content := string(f.Content)
			readme = &content
			assert.Equal(t, int64(len(f.Content)), f.Size)
			break
		}
	}

	require.NotNil(t, readme)
	assert.Contains(t, *readme, "# test-api")
	assert.Contains(t, *readme, "Protocol Buffer generated code for Java")
	assert.Contains(t, *readme, "## Installation")
	assert.Contains(t, *readme, "<dependency>")
	assert.Contains(t, *readme, "<artifactId>test-api</artifactId>")
	assert.Contains(t, *readme, "<version>3.1.4</version>")
}
