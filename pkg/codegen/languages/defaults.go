package languages

// GetDefaultLanguages returns the default language configurations
func GetDefaultLanguages() []*LanguageSpec {
	return []*LanguageSpec{
		getGoLanguageSpec(),
		getPythonLanguageSpec(),
		getJavaLanguageSpec(),
	}
}

func getGoLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageGo,
		Name:            "Go",
		DisplayName:     "Go (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-go",
		PluginVersion:   "v1.31.0",
		ProtocFlags: []string{
			"--go_opt=paths=source_relative",
		},
		DockerImage:    "spoke/compiler-go",
		DockerTag:      "1.31.0",
		SupportsGRPC:   true,
		GRPCPlugin:     "protoc-gen-go-grpc",
		GRPCPluginVersion: "v1.3.0",
		GRPCFlags: []string{
			"--go-grpc_opt=paths=source_relative",
		},
		PackageManager: &PackageManagerSpec{
			Name:        "go-modules",
			ConfigFiles: []string{"go.mod", "README.md"},
			DefaultVersions: map[string]string{
				"google.golang.org/protobuf": "v1.31.0",
				"google.golang.org/grpc":     "v1.60.0",
			},
		},
		FileExtensions: []string{".pb.go"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "Go language support with protoc-gen-go",
		DocumentationURL: "https://protobuf.dev/reference/go/go-generated/",
	}
}

func getPythonLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguagePython,
		Name:            "Python",
		DisplayName:     "Python (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-python",
		PluginVersion:   "4.24.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-python",
		DockerTag:       "4.24.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "grpc_python_plugin",
		GRPCPluginVersion: "1.59.0",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "pip",
			ConfigFiles: []string{"setup.py", "pyproject.toml", "README.md"},
			DefaultVersions: map[string]string{
				"protobuf": "4.24.0",
				"grpcio":   "1.59.0",
				"grpcio-tools": "1.59.0",
			},
		},
		FileExtensions: []string{"_pb2.py", "_pb2_grpc.py"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "Python language support with protobuf and grpcio",
		DocumentationURL: "https://protobuf.dev/reference/python/python-generated/",
	}
}

func getJavaLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageJava,
		Name:            "Java",
		DisplayName:     "Java (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-java",
		PluginVersion:   "3.21.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-java",
		DockerTag:       "3.21.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "protoc-gen-grpc-java",
		GRPCPluginVersion: "1.59.0",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "maven",
			ConfigFiles: []string{"pom.xml", "README.md"},
			DefaultVersions: map[string]string{
				"com.google.protobuf:protobuf-java": "3.21.0",
				"io.grpc:grpc-protobuf":             "1.59.0",
				"io.grpc:grpc-stub":                 "1.59.0",
			},
		},
		FileExtensions: []string{".java"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "Java language support with protobuf and gRPC",
		DocumentationURL: "https://protobuf.dev/reference/java/java-generated/",
	}
}

// InitializeDefaultRegistry creates and populates a registry with default languages
func InitializeDefaultRegistry() *Registry {
	registry := NewRegistry()

	for _, spec := range GetDefaultLanguages() {
		if err := registry.Register(spec); err != nil {
			// This should never happen with valid default specs
			panic("failed to register default language: " + err.Error())
		}
	}

	return registry
}
