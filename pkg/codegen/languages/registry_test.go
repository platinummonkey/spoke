package languages

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.Count() != 0 {
		t.Errorf("expected empty registry, got count=%d", r.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	spec := &LanguageSpec{
		ID:           "test",
		Name:         "Test",
		DockerImage:  "test/image",
		ProtocPlugin: "protoc-gen-test",
		Enabled:      true,
	}

	// Test successful registration
	err := r.Register(spec)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if r.Count() != 1 {
		t.Errorf("expected count=1, got %d", r.Count())
	}

	// Test duplicate registration
	err = r.Register(spec)
	if err != ErrLanguageAlreadyExists {
		t.Errorf("expected ErrLanguageAlreadyExists, got: %v", err)
	}
}

func TestRegistry_Register_Invalid(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name string
		spec *LanguageSpec
		err  error
	}{
		{
			name: "missing ID",
			spec: &LanguageSpec{
				Name:         "Test",
				DockerImage:  "test/image",
				ProtocPlugin: "protoc-gen-test",
			},
			err: ErrInvalidLanguageID,
		},
		{
			name: "missing Name",
			spec: &LanguageSpec{
				ID:           "test",
				DockerImage:  "test/image",
				ProtocPlugin: "protoc-gen-test",
			},
			err: ErrInvalidLanguageName,
		},
		{
			name: "missing DockerImage",
			spec: &LanguageSpec{
				ID:           "test",
				Name:         "Test",
				ProtocPlugin: "protoc-gen-test",
			},
			err: ErrInvalidDockerImage,
		},
		{
			name: "missing ProtocPlugin",
			spec: &LanguageSpec{
				ID:          "test",
				Name:        "Test",
				DockerImage: "test/image",
			},
			err: ErrInvalidProtocPlugin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Register(tt.spec)
			if err != tt.err {
				t.Errorf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	spec := &LanguageSpec{
		ID:           "test",
		Name:         "Test",
		DockerImage:  "test/image",
		ProtocPlugin: "protoc-gen-test",
		Enabled:      true,
	}

	r.Register(spec)

	// Test successful get
	retrieved, err := r.Get("test")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if retrieved.ID != "test" {
		t.Errorf("expected ID=test, got %s", retrieved.ID)
	}

	// Test not found
	_, err = r.Get("nonexistent")
	if err != ErrLanguageNotFound {
		t.Errorf("expected ErrLanguageNotFound, got: %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	specs := []*LanguageSpec{
		{ID: "go", Name: "Go", DockerImage: "spoke/go", ProtocPlugin: "protoc-gen-go", Enabled: true},
		{ID: "python", Name: "Python", DockerImage: "spoke/python", ProtocPlugin: "protoc-gen-python", Enabled: true},
		{ID: "java", Name: "Java", DockerImage: "spoke/java", ProtocPlugin: "protoc-gen-java", Enabled: false},
	}

	for _, spec := range specs {
		r.Register(spec)
	}

	all := r.List()
	if len(all) != 3 {
		t.Errorf("expected 3 languages, got %d", len(all))
	}
}

func TestRegistry_ListEnabled(t *testing.T) {
	r := NewRegistry()

	specs := []*LanguageSpec{
		{ID: "go", Name: "Go", DockerImage: "spoke/go", ProtocPlugin: "protoc-gen-go", Enabled: true},
		{ID: "python", Name: "Python", DockerImage: "spoke/python", ProtocPlugin: "protoc-gen-python", Enabled: true},
		{ID: "java", Name: "Java", DockerImage: "spoke/java", ProtocPlugin: "protoc-gen-java", Enabled: false},
	}

	for _, spec := range specs {
		r.Register(spec)
	}

	enabled := r.ListEnabled()
	if len(enabled) != 2 {
		t.Errorf("expected 2 enabled languages, got %d", len(enabled))
	}

	for _, spec := range enabled {
		if !spec.Enabled {
			t.Errorf("expected only enabled languages, got disabled: %s", spec.ID)
		}
	}
}

func TestRegistry_Update(t *testing.T) {
	r := NewRegistry()

	spec := &LanguageSpec{
		ID:           "test",
		Name:         "Test",
		DockerImage:  "test/image",
		ProtocPlugin: "protoc-gen-test",
		Enabled:      true,
	}

	r.Register(spec)

	// Update the spec
	updated := &LanguageSpec{
		ID:           "test",
		Name:         "Test Updated",
		DockerImage:  "test/image:v2",
		ProtocPlugin: "protoc-gen-test",
		Enabled:      false,
	}

	err := r.Update(updated)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify update
	retrieved, _ := r.Get("test")
	if retrieved.Name != "Test Updated" {
		t.Errorf("expected Name='Test Updated', got %s", retrieved.Name)
	}
	if retrieved.DockerImage != "test/image:v2" {
		t.Errorf("expected DockerImage='test/image:v2', got %s", retrieved.DockerImage)
	}
	if retrieved.Enabled {
		t.Error("expected Enabled=false")
	}

	// Test update nonexistent
	err = r.Update(&LanguageSpec{
		ID:           "nonexistent",
		Name:         "Nonexistent",
		DockerImage:  "test/image",
		ProtocPlugin: "protoc-gen-test",
	})
	if err != ErrLanguageNotFound {
		t.Errorf("expected ErrLanguageNotFound, got: %v", err)
	}
}

func TestRegistry_Delete(t *testing.T) {
	r := NewRegistry()

	spec := &LanguageSpec{
		ID:           "test",
		Name:         "Test",
		DockerImage:  "test/image",
		ProtocPlugin: "protoc-gen-test",
		Enabled:      true,
	}

	r.Register(spec)

	// Test successful delete
	err := r.Delete("test")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if r.Count() != 0 {
		t.Errorf("expected count=0, got %d", r.Count())
	}

	// Test delete nonexistent
	err = r.Delete("nonexistent")
	if err != ErrLanguageNotFound {
		t.Errorf("expected ErrLanguageNotFound, got: %v", err)
	}
}

func TestRegistry_IsEnabled(t *testing.T) {
	r := NewRegistry()

	specs := []*LanguageSpec{
		{ID: "enabled", Name: "Enabled", DockerImage: "test/enabled", ProtocPlugin: "protoc-gen-enabled", Enabled: true},
		{ID: "disabled", Name: "Disabled", DockerImage: "test/disabled", ProtocPlugin: "protoc-gen-disabled", Enabled: false},
	}

	for _, spec := range specs {
		r.Register(spec)
	}

	if !r.IsEnabled("enabled") {
		t.Error("expected 'enabled' to be enabled")
	}

	if r.IsEnabled("disabled") {
		t.Error("expected 'disabled' to be disabled")
	}

	if r.IsEnabled("nonexistent") {
		t.Error("expected 'nonexistent' to be disabled")
	}
}

func TestRegistry_Count(t *testing.T) {
	r := NewRegistry()

	if r.Count() != 0 {
		t.Errorf("expected count=0, got %d", r.Count())
	}

	specs := []*LanguageSpec{
		{ID: "go", Name: "Go", DockerImage: "spoke/go", ProtocPlugin: "protoc-gen-go", Enabled: true},
		{ID: "python", Name: "Python", DockerImage: "spoke/python", ProtocPlugin: "protoc-gen-python", Enabled: true},
		{ID: "java", Name: "Java", DockerImage: "spoke/java", ProtocPlugin: "protoc-gen-java", Enabled: false},
	}

	for i, spec := range specs {
		r.Register(spec)
		if r.Count() != i+1 {
			t.Errorf("expected count=%d, got %d", i+1, r.Count())
		}
	}
}

func TestRegistry_Concurrent(t *testing.T) {
	r := NewRegistry()

	// Test concurrent access
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 10; i++ {
			spec := &LanguageSpec{
				ID:           "test",
				Name:         "Test",
				DockerImage:  "test/image",
				ProtocPlugin: "protoc-gen-test",
				Enabled:      true,
			}
			r.Register(spec)
			r.Update(spec)
			r.Delete("test")
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 10; i++ {
			r.Get("test")
			r.List()
			r.ListEnabled()
			r.IsEnabled("test")
			r.Count()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}
