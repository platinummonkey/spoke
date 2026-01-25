package languages

import (
	"testing"
)

func TestGetDefaultLanguages(t *testing.T) {
	langs := GetDefaultLanguages()

	if len(langs) != 3 {
		t.Errorf("expected 3 default languages, got %d", len(langs))
	}

	// Check that all default languages are valid
	for _, lang := range langs {
		if err := lang.Validate(); err != nil {
			t.Errorf("default language %s is invalid: %v", lang.ID, err)
		}
	}

	// Check specific languages
	ids := make(map[string]bool)
	for _, lang := range langs {
		ids[lang.ID] = true
	}

	expectedIDs := []string{LanguageGo, LanguagePython, LanguageJava}
	for _, id := range expectedIDs {
		if !ids[id] {
			t.Errorf("expected default language %s not found", id)
		}
	}
}

func TestGetGoLanguageSpec(t *testing.T) {
	spec := getGoLanguageSpec()

	if spec.ID != LanguageGo {
		t.Errorf("expected ID=%s, got %s", LanguageGo, spec.ID)
	}

	if spec.Name != "Go" {
		t.Errorf("expected Name=Go, got %s", spec.Name)
	}

	if spec.ProtocPlugin != "protoc-gen-go" {
		t.Errorf("expected ProtocPlugin=protoc-gen-go, got %s", spec.ProtocPlugin)
	}

	if !spec.SupportsGRPC {
		t.Error("expected Go to support gRPC")
	}

	if spec.GRPCPlugin != "protoc-gen-go-grpc" {
		t.Errorf("expected GRPCPlugin=protoc-gen-go-grpc, got %s", spec.GRPCPlugin)
	}

	if spec.PackageManager == nil {
		t.Fatal("expected PackageManager to be non-nil")
	}

	if spec.PackageManager.Name != "go-modules" {
		t.Errorf("expected PackageManager.Name=go-modules, got %s", spec.PackageManager.Name)
	}

	if !spec.Enabled {
		t.Error("expected Go to be enabled")
	}

	if !spec.Stable {
		t.Error("expected Go to be stable")
	}

	if err := spec.Validate(); err != nil {
		t.Errorf("Go spec validation failed: %v", err)
	}
}

func TestGetPythonLanguageSpec(t *testing.T) {
	spec := getPythonLanguageSpec()

	if spec.ID != LanguagePython {
		t.Errorf("expected ID=%s, got %s", LanguagePython, spec.ID)
	}

	if spec.Name != "Python" {
		t.Errorf("expected Name=Python, got %s", spec.Name)
	}

	if spec.ProtocPlugin != "protoc-gen-python" {
		t.Errorf("expected ProtocPlugin=protoc-gen-python, got %s", spec.ProtocPlugin)
	}

	if !spec.SupportsGRPC {
		t.Error("expected Python to support gRPC")
	}

	if spec.PackageManager == nil {
		t.Fatal("expected PackageManager to be non-nil")
	}

	if spec.PackageManager.Name != "pip" {
		t.Errorf("expected PackageManager.Name=pip, got %s", spec.PackageManager.Name)
	}

	if !spec.Enabled {
		t.Error("expected Python to be enabled")
	}

	if !spec.Stable {
		t.Error("expected Python to be stable")
	}

	if err := spec.Validate(); err != nil {
		t.Errorf("Python spec validation failed: %v", err)
	}
}

func TestGetJavaLanguageSpec(t *testing.T) {
	spec := getJavaLanguageSpec()

	if spec.ID != LanguageJava {
		t.Errorf("expected ID=%s, got %s", LanguageJava, spec.ID)
	}

	if spec.Name != "Java" {
		t.Errorf("expected Name=Java, got %s", spec.Name)
	}

	if spec.ProtocPlugin != "protoc-gen-java" {
		t.Errorf("expected ProtocPlugin=protoc-gen-java, got %s", spec.ProtocPlugin)
	}

	if !spec.SupportsGRPC {
		t.Error("expected Java to support gRPC")
	}

	if spec.PackageManager == nil {
		t.Fatal("expected PackageManager to be non-nil")
	}

	if spec.PackageManager.Name != "maven" {
		t.Errorf("expected PackageManager.Name=maven, got %s", spec.PackageManager.Name)
	}

	if !spec.Enabled {
		t.Error("expected Java to be enabled")
	}

	if !spec.Stable {
		t.Error("expected Java to be stable")
	}

	if err := spec.Validate(); err != nil {
		t.Errorf("Java spec validation failed: %v", err)
	}
}

func TestInitializeDefaultRegistry(t *testing.T) {
	registry := InitializeDefaultRegistry()

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	if registry.Count() != 3 {
		t.Errorf("expected 3 languages in registry, got %d", registry.Count())
	}

	// Verify each default language is registered
	for _, id := range []string{LanguageGo, LanguagePython, LanguageJava} {
		spec, err := registry.Get(id)
		if err != nil {
			t.Errorf("expected language %s to be registered: %v", id, err)
		}
		if spec == nil {
			t.Errorf("expected non-nil spec for %s", id)
		}
	}

	// Verify all are enabled
	enabled := registry.ListEnabled()
	if len(enabled) != 3 {
		t.Errorf("expected 3 enabled languages, got %d", len(enabled))
	}
}
