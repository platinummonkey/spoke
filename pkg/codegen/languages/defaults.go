package languages

// GetDefaultLanguages returns the default language configurations
func GetDefaultLanguages() []*LanguageSpec {
	return []*LanguageSpec{
		getGoLanguageSpec(),
		getPythonLanguageSpec(),
		getJavaLanguageSpec(),
		getCPPLanguageSpec(),
		getCSharpLanguageSpec(),
		getRustLanguageSpec(),
		getTypeScriptLanguageSpec(),
		getJavaScriptLanguageSpec(),
		getDartLanguageSpec(),
		getSwiftLanguageSpec(),
		getKotlinLanguageSpec(),
		getObjectiveCLanguageSpec(),
		getRubyLanguageSpec(),
		getPHPLanguageSpec(),
		getScalaLanguageSpec(),
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

func getCPPLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageCPP,
		Name:            "C++",
		DisplayName:     "C++ (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-cpp",
		PluginVersion:   "3.21.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-cpp",
		DockerTag:       "3.21.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "grpc_cpp_plugin",
		GRPCPluginVersion: "1.59.0",
		GRPCFlags:       []string{},
		PackageManager:  nil, // C++ doesn't have a standard package manager
		FileExtensions:  []string{".pb.h", ".pb.cc"},
		Enabled:         true,
		Stable:          true,
		Experimental:    false,
		Description:     "C++ language support with protobuf and gRPC",
		DocumentationURL: "https://protobuf.dev/reference/cpp/cpp-generated/",
	}
}

func getCSharpLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageCSharp,
		Name:            "C#",
		DisplayName:     "C# (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-csharp",
		PluginVersion:   "3.21.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-csharp",
		DockerTag:       "3.21.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "grpc_csharp_plugin",
		GRPCPluginVersion: "2.59.0",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "nuget",
			ConfigFiles: []string{"Package.csproj", "README.md"},
			DefaultVersions: map[string]string{
				"Google.Protobuf": "3.21.0",
				"Grpc.Core":       "2.59.0",
				"Grpc.Tools":      "2.59.0",
			},
		},
		FileExtensions: []string{".cs"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "C# language support with protobuf and gRPC",
		DocumentationURL: "https://protobuf.dev/reference/csharp/csharp-generated/",
	}
}

func getRustLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageRust,
		Name:            "Rust",
		DisplayName:     "Rust (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-rust",
		PluginVersion:   "3.2.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-rust",
		DockerTag:       "3.2.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "protoc-gen-tonic",
		GRPCPluginVersion: "0.10.0",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "cargo",
			ConfigFiles: []string{"Cargo.toml", "README.md"},
			DefaultVersions: map[string]string{
				"prost":       "0.12.0",
				"prost-types": "0.12.0",
				"tonic":       "0.10.0",
			},
		},
		FileExtensions: []string{".rs"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "Rust language support with prost and tonic",
		DocumentationURL: "https://github.com/tokio-rs/prost",
	}
}

func getTypeScriptLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageTypeScript,
		Name:            "TypeScript",
		DisplayName:     "TypeScript (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-ts",
		PluginVersion:   "5.0.1",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-typescript",
		DockerTag:       "5.0.1",
		SupportsGRPC:    true,
		GRPCPlugin:      "protoc-gen-grpc-web",
		GRPCPluginVersion: "1.4.2",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "npm",
			ConfigFiles: []string{"package.json", "tsconfig.json", "README.md"},
			DefaultVersions: map[string]string{
				"google-protobuf": "^3.21.0",
				"@grpc/grpc-js":   "^1.9.0",
				"@types/google-protobuf": "^3.15.0",
			},
		},
		FileExtensions: []string{".ts", "_pb.ts"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "TypeScript language support with ts-proto",
		DocumentationURL: "https://github.com/stephenh/ts-proto",
	}
}

func getJavaScriptLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageJavaScript,
		Name:            "JavaScript",
		DisplayName:     "JavaScript (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-js",
		PluginVersion:   "3.21.0",
		ProtocFlags:     []string{"--js_opt=import_style=commonjs"},
		DockerImage:     "spoke/compiler-javascript",
		DockerTag:       "3.21.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "protoc-gen-grpc-web",
		GRPCPluginVersion: "1.4.2",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "npm",
			ConfigFiles: []string{"package.json", "README.md"},
			DefaultVersions: map[string]string{
				"google-protobuf": "^3.21.0",
				"@grpc/grpc-js":   "^1.9.0",
			},
		},
		FileExtensions: []string{"_pb.js"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "JavaScript language support with protobufjs",
		DocumentationURL: "https://protobuf.dev/reference/javascript/javascript-generated/",
	}
}

func getDartLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageDart,
		Name:            "Dart",
		DisplayName:     "Dart (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-dart",
		PluginVersion:   "3.1.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-dart",
		DockerTag:       "3.1.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "protoc-gen-dart",
		GRPCPluginVersion: "3.1.0",
		GRPCFlags:       []string{"--dart_opt=grpc"},
		PackageManager: &PackageManagerSpec{
			Name:        "pub",
			ConfigFiles: []string{"pubspec.yaml", "README.md"},
			DefaultVersions: map[string]string{
				"protobuf": "^3.1.0",
				"grpc":     "^3.2.0",
			},
		},
		FileExtensions: []string{".pb.dart", ".pbgrpc.dart"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "Dart language support with protobuf and gRPC",
		DocumentationURL: "https://pub.dev/packages/protobuf",
	}
}

func getSwiftLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageSwift,
		Name:            "Swift",
		DisplayName:     "Swift (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-swift",
		PluginVersion:   "1.25.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-swift",
		DockerTag:       "1.25.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "protoc-gen-grpc-swift",
		GRPCPluginVersion: "1.21.0",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "swift-package",
			ConfigFiles: []string{"Package.swift", "README.md"},
			DefaultVersions: map[string]string{
				"swift-protobuf": "1.25.0",
				"grpc-swift":     "1.21.0",
			},
		},
		FileExtensions: []string{".pb.swift", ".grpc.swift"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "Swift language support with SwiftProtobuf and gRPC-Swift",
		DocumentationURL: "https://github.com/apple/swift-protobuf",
	}
}

func getKotlinLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageKotlin,
		Name:            "Kotlin",
		DisplayName:     "Kotlin (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-kotlin",
		PluginVersion:   "3.21.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-kotlin",
		DockerTag:       "3.21.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "protoc-gen-grpc-kotlin",
		GRPCPluginVersion: "1.4.0",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "gradle",
			ConfigFiles: []string{"build.gradle.kts", "README.md"},
			DefaultVersions: map[string]string{
				"com.google.protobuf:protobuf-kotlin": "3.21.0",
				"io.grpc:grpc-kotlin-stub":            "1.4.0",
				"io.grpc:grpc-protobuf":               "1.59.0",
			},
		},
		FileExtensions: []string{".kt"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "Kotlin language support with protobuf-kotlin and gRPC-Kotlin",
		DocumentationURL: "https://github.com/grpc/grpc-kotlin",
	}
}

func getObjectiveCLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageObjectiveC,
		Name:            "Objective-C",
		DisplayName:     "Objective-C (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-objc",
		PluginVersion:   "3.21.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-objc",
		DockerTag:       "3.21.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "grpc_objective_c_plugin",
		GRPCPluginVersion: "1.59.0",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "cocoapods",
			ConfigFiles: []string{"Podspec", "README.md"},
			DefaultVersions: map[string]string{
				"Protobuf":      "3.21.0",
				"gRPC-ProtoRPC": "1.59.0",
			},
		},
		FileExtensions: []string{".pbobjc.h", ".pbobjc.m"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "Objective-C language support with protobuf and gRPC",
		DocumentationURL: "https://protobuf.dev/reference/objective-c/objective-c-generated/",
	}
}

func getRubyLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageRuby,
		Name:            "Ruby",
		DisplayName:     "Ruby (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-ruby",
		PluginVersion:   "3.21.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-ruby",
		DockerTag:       "3.21.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "grpc_ruby_plugin",
		GRPCPluginVersion: "1.59.0",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "gem",
			ConfigFiles: []string{"gemspec", "README.md"},
			DefaultVersions: map[string]string{
				"google-protobuf": "3.21.0",
				"grpc":            "1.59.0",
			},
		},
		FileExtensions: []string{"_pb.rb"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "Ruby language support with protobuf and gRPC",
		DocumentationURL: "https://protobuf.dev/reference/ruby/ruby-generated/",
	}
}

func getPHPLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguagePHP,
		Name:            "PHP",
		DisplayName:     "PHP (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-php",
		PluginVersion:   "3.21.0",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-php",
		DockerTag:       "3.21.0",
		SupportsGRPC:    true,
		GRPCPlugin:      "grpc_php_plugin",
		GRPCPluginVersion: "1.59.0",
		GRPCFlags:       []string{},
		PackageManager: &PackageManagerSpec{
			Name:        "composer",
			ConfigFiles: []string{"composer.json", "README.md"},
			DefaultVersions: map[string]string{
				"google/protobuf": "^3.21",
				"grpc/grpc":       "^1.59",
			},
		},
		FileExtensions: []string{".php"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "PHP language support with protobuf and gRPC",
		DocumentationURL: "https://protobuf.dev/reference/php/php-generated/",
	}
}

func getScalaLanguageSpec() *LanguageSpec {
	return &LanguageSpec{
		ID:              LanguageScala,
		Name:            "Scala",
		DisplayName:     "Scala (Protocol Buffers)",
		ProtocPlugin:    "protoc-gen-scala",
		PluginVersion:   "0.11.13",
		ProtocFlags:     []string{},
		DockerImage:     "spoke/compiler-scala",
		DockerTag:       "0.11.13",
		SupportsGRPC:    true,
		GRPCPlugin:      "protoc-gen-scala",
		GRPCPluginVersion: "0.11.13",
		GRPCFlags:       []string{"--scala_opt=grpc"},
		PackageManager: &PackageManagerSpec{
			Name:        "sbt",
			ConfigFiles: []string{"build.sbt", "README.md"},
			DefaultVersions: map[string]string{
				"com.thesamet.scalapb::scalapb-runtime": "0.11.13",
				"io.grpc:grpc-netty":                    "1.59.0",
			},
		},
		FileExtensions: []string{".scala"},
		Enabled:        true,
		Stable:         true,
		Experimental:   false,
		Description:    "Scala language support with ScalaPB",
		DocumentationURL: "https://scalapb.github.io/",
	}
}

// InitializeDefaultRegistry creates and populates a registry with default languages
func InitializeDefaultRegistry() (*Registry, error) {
	registry := NewRegistry()

	for _, spec := range GetDefaultLanguages() {
		if err := registry.Register(spec); err != nil {
			return nil, err
		}
	}

	return registry, nil
}
