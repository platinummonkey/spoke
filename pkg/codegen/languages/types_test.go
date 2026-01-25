package languages

import (
	"testing"
)

func TestLanguageSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    *LanguageSpec
		wantErr error
	}{
		{
			name: "valid spec",
			spec: &LanguageSpec{
				ID:           "test",
				Name:         "Test",
				DockerImage:  "test/image",
				ProtocPlugin: "protoc-gen-test",
			},
			wantErr: nil,
		},
		{
			name: "missing ID",
			spec: &LanguageSpec{
				Name:         "Test",
				DockerImage:  "test/image",
				ProtocPlugin: "protoc-gen-test",
			},
			wantErr: ErrInvalidLanguageID,
		},
		{
			name: "missing Name",
			spec: &LanguageSpec{
				ID:           "test",
				DockerImage:  "test/image",
				ProtocPlugin: "protoc-gen-test",
			},
			wantErr: ErrInvalidLanguageName,
		},
		{
			name: "missing DockerImage",
			spec: &LanguageSpec{
				ID:           "test",
				Name:         "Test",
				ProtocPlugin: "protoc-gen-test",
			},
			wantErr: ErrInvalidDockerImage,
		},
		{
			name: "missing ProtocPlugin",
			spec: &LanguageSpec{
				ID:          "test",
				Name:        "Test",
				DockerImage: "test/image",
			},
			wantErr: ErrInvalidProtocPlugin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLanguageSpec_GetFullDockerImage(t *testing.T) {
	tests := []struct {
		name string
		spec *LanguageSpec
		want string
	}{
		{
			name: "with tag",
			spec: &LanguageSpec{
				DockerImage: "spoke/compiler-go",
				DockerTag:   "1.31.0",
			},
			want: "spoke/compiler-go:1.31.0",
		},
		{
			name: "without tag",
			spec: &LanguageSpec{
				DockerImage: "spoke/compiler-go",
			},
			want: "spoke/compiler-go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.GetFullDockerImage()
			if got != tt.want {
				t.Errorf("GetFullDockerImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPackageManagerSpec_Validate(t *testing.T) {
	spec := &PackageManagerSpec{
		Name:        "test",
		ConfigFiles: []string{"test.conf"},
	}

	if spec.Name != "test" {
		t.Errorf("expected Name='test', got %s", spec.Name)
	}
}
