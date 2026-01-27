package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// listLanguages returns a list of all supported languages
func (s *Server) listLanguages(w http.ResponseWriter, r *http.Request) {
	// Check if orchestrator is available
	if s.orchestrator == nil {
		http.Error(w, "Code generation orchestrator not available", http.StatusServiceUnavailable)
		return
	}

	// Get language registry from orchestrator (we need to add a method for this)
	// For now, return hardcoded list based on our language constants
	languages := []LanguageInfo{
		{
			ID:               string(LanguageGo),
			Name:             "Go",
			DisplayName:      "Go (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".pb.go"},
			Enabled:          true,
			Stable:           true,
			Description:      "Go language support with protoc-gen-go",
			DocumentationURL: "https://protobuf.dev/reference/go/go-generated/",
			PluginVersion:    "v1.31.0",
			PackageManager:   &PackageManagerInfo{Name: "go-modules", ConfigFiles: []string{"go.mod"}},
		},
		{
			ID:               string(LanguagePython),
			Name:             "Python",
			DisplayName:      "Python (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{"_pb2.py", "_pb2_grpc.py"},
			Enabled:          true,
			Stable:           true,
			Description:      "Python language support with protobuf and grpcio",
			DocumentationURL: "https://protobuf.dev/reference/python/python-generated/",
			PluginVersion:    "4.24.0",
			PackageManager:   &PackageManagerInfo{Name: "pip", ConfigFiles: []string{"setup.py", "pyproject.toml"}},
		},
		{
			ID:               string(LanguageJava),
			Name:             "Java",
			DisplayName:      "Java (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".java"},
			Enabled:          true,
			Stable:           true,
			Description:      "Java language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/java/java-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "maven", ConfigFiles: []string{"pom.xml"}},
		},
		{
			ID:               string(LanguageCPP),
			Name:             "C++",
			DisplayName:      "C++ (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".pb.h", ".pb.cc"},
			Enabled:          true,
			Stable:           true,
			Description:      "C++ language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/cpp/cpp-generated/",
			PluginVersion:    "3.21.0",
		},
		{
			ID:               string(LanguageCSharp),
			Name:             "C#",
			DisplayName:      "C# (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".cs"},
			Enabled:          true,
			Stable:           true,
			Description:      "C# language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/csharp/csharp-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "nuget", ConfigFiles: []string{"Package.csproj"}},
		},
		{
			ID:               string(LanguageRust),
			Name:             "Rust",
			DisplayName:      "Rust (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".rs"},
			Enabled:          true,
			Stable:           true,
			Description:      "Rust language support with prost and tonic",
			DocumentationURL: "https://github.com/tokio-rs/prost",
			PluginVersion:    "3.2.0",
			PackageManager:   &PackageManagerInfo{Name: "cargo", ConfigFiles: []string{"Cargo.toml"}},
		},
		{
			ID:               string(LanguageTypeScript),
			Name:             "TypeScript",
			DisplayName:      "TypeScript (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".ts", "_pb.ts"},
			Enabled:          true,
			Stable:           true,
			Description:      "TypeScript language support with ts-proto",
			DocumentationURL: "https://github.com/stephenh/ts-proto",
			PluginVersion:    "5.0.1",
			PackageManager:   &PackageManagerInfo{Name: "npm", ConfigFiles: []string{"package.json", "tsconfig.json"}},
		},
		{
			ID:               string(LanguageJavaScript),
			Name:             "JavaScript",
			DisplayName:      "JavaScript (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{"_pb.js"},
			Enabled:          true,
			Stable:           true,
			Description:      "JavaScript language support with protobufjs",
			DocumentationURL: "https://protobuf.dev/reference/javascript/javascript-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "npm", ConfigFiles: []string{"package.json"}},
		},
		{
			ID:               string(LanguageDart),
			Name:             "Dart",
			DisplayName:      "Dart (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".pb.dart", ".pbgrpc.dart"},
			Enabled:          true,
			Stable:           true,
			Description:      "Dart language support with protobuf and gRPC",
			DocumentationURL: "https://pub.dev/packages/protobuf",
			PluginVersion:    "3.1.0",
			PackageManager:   &PackageManagerInfo{Name: "pub", ConfigFiles: []string{"pubspec.yaml"}},
		},
		{
			ID:               string(LanguageSwift),
			Name:             "Swift",
			DisplayName:      "Swift (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".pb.swift", ".grpc.swift"},
			Enabled:          true,
			Stable:           true,
			Description:      "Swift language support with SwiftProtobuf and gRPC-Swift",
			DocumentationURL: "https://github.com/apple/swift-protobuf",
			PluginVersion:    "1.25.0",
			PackageManager:   &PackageManagerInfo{Name: "swift-package", ConfigFiles: []string{"Package.swift"}},
		},
		{
			ID:               string(LanguageKotlin),
			Name:             "Kotlin",
			DisplayName:      "Kotlin (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".kt"},
			Enabled:          true,
			Stable:           true,
			Description:      "Kotlin language support with protobuf-kotlin and gRPC-Kotlin",
			DocumentationURL: "https://github.com/grpc/grpc-kotlin",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "gradle", ConfigFiles: []string{"build.gradle.kts"}},
		},
		{
			ID:               string(LanguageObjectiveC),
			Name:             "Objective-C",
			DisplayName:      "Objective-C (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".pbobjc.h", ".pbobjc.m"},
			Enabled:          true,
			Stable:           true,
			Description:      "Objective-C language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/objective-c/objective-c-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "cocoapods", ConfigFiles: []string{"Podspec"}},
		},
		{
			ID:               string(LanguageRuby),
			Name:             "Ruby",
			DisplayName:      "Ruby (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{"_pb.rb"},
			Enabled:          true,
			Stable:           true,
			Description:      "Ruby language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/ruby/ruby-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "gem", ConfigFiles: []string{"gemspec"}},
		},
		{
			ID:               string(LanguagePHP),
			Name:             "PHP",
			DisplayName:      "PHP (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".php"},
			Enabled:          true,
			Stable:           true,
			Description:      "PHP language support with protobuf and gRPC",
			DocumentationURL: "https://protobuf.dev/reference/php/php-generated/",
			PluginVersion:    "3.21.0",
			PackageManager:   &PackageManagerInfo{Name: "composer", ConfigFiles: []string{"composer.json"}},
		},
		{
			ID:               string(LanguageScala),
			Name:             "Scala",
			DisplayName:      "Scala (Protocol Buffers)",
			SupportsGRPC:     true,
			FileExtensions:   []string{".scala"},
			Enabled:          true,
			Stable:           true,
			Description:      "Scala language support with ScalaPB",
			DocumentationURL: "https://scalapb.github.io/",
			PluginVersion:    "0.11.13",
			PackageManager:   &PackageManagerInfo{Name: "sbt", ConfigFiles: []string{"build.sbt"}},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(languages)
}

// getLanguage returns details for a specific language
func (s *Server) getLanguage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	languageID := vars["id"]

	// Check if orchestrator is available
	if s.orchestrator == nil {
		http.Error(w, "Code generation orchestrator not available", http.StatusServiceUnavailable)
		return
	}

	// Get all languages and find the requested one
	// In a real implementation, we would query the language registry directly
	var targetLang *LanguageInfo

	// Call listLanguages to get all languages (reuse logic)
	allLanguages := []LanguageInfo{} // We'd populate this from registry
	// For now, find in hardcoded list
	for _, lang := range allLanguages {
		if lang.ID == languageID {
			targetLang = &lang
			break
		}
	}

	if targetLang == nil {
		http.Error(w, fmt.Sprintf("Language %s not found", languageID), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targetLang)
}
