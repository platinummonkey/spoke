package examples

import (
	"testing"
)

func TestExampleData(t *testing.T) {
	tests := []struct {
		name string
		data ExampleData
	}{
		{
			name: "empty ExampleData",
			data: ExampleData{},
		},
		{
			name: "ExampleData with all fields",
			data: ExampleData{
				Language:    "go",
				ModuleName:  "example.com/module",
				Version:     "v1.0.0",
				PackagePath: "example/path",
				ServiceName: "ExampleService",
				Methods: []MethodExample{
					{
						Name:            "GetExample",
						RequestType:     "GetRequest",
						ResponseType:    "GetResponse",
						IsClientStream:  false,
						IsServerStream:  false,
						IsBidirectional: false,
					},
				},
				Messages: []MessageExample{
					{
						Name: "ExampleMessage",
						Fields: []SampleField{
							{
								Name:        "field1",
								Type:        "string",
								SampleValue: "value1",
								IsRepeated:  false,
							},
						},
					},
				},
				Imports: []string{"fmt", "context"},
				PackageManager: PackageManagerInfo{
					Command:        "go get",
					PackageName:    "example.com/module",
					InstallExample: "go get example.com/module",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify fields can be accessed
			_ = tt.data.Language
			_ = tt.data.ModuleName
			_ = tt.data.Version
			_ = tt.data.PackagePath
			_ = tt.data.ServiceName
			_ = tt.data.Methods
			_ = tt.data.Messages
			_ = tt.data.Imports
			_ = tt.data.PackageManager
		})
	}
}

func TestMethodExample(t *testing.T) {
	tests := []struct {
		name   string
		method MethodExample
	}{
		{
			name:   "empty MethodExample",
			method: MethodExample{},
		},
		{
			name: "unary method",
			method: MethodExample{
				Name:            "GetUser",
				RequestType:     "GetUserRequest",
				ResponseType:    "GetUserResponse",
				IsClientStream:  false,
				IsServerStream:  false,
				IsBidirectional: false,
			},
		},
		{
			name: "client streaming method",
			method: MethodExample{
				Name:            "StreamData",
				RequestType:     "StreamRequest",
				ResponseType:    "StreamResponse",
				SampleFields:    []SampleField{{Name: "data", Type: "string", SampleValue: "test"}},
				IsClientStream:  true,
				IsServerStream:  false,
				IsBidirectional: false,
			},
		},
		{
			name: "server streaming method",
			method: MethodExample{
				Name:            "WatchEvents",
				RequestType:     "WatchRequest",
				ResponseType:    "WatchResponse",
				IsClientStream:  false,
				IsServerStream:  true,
				IsBidirectional: false,
			},
		},
		{
			name: "bidirectional streaming method",
			method: MethodExample{
				Name:            "Chat",
				RequestType:     "ChatMessage",
				ResponseType:    "ChatMessage",
				IsClientStream:  true,
				IsServerStream:  true,
				IsBidirectional: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.method.Name
			_ = tt.method.RequestType
			_ = tt.method.ResponseType
			_ = tt.method.SampleFields
			_ = tt.method.IsClientStream
			_ = tt.method.IsServerStream
			_ = tt.method.IsBidirectional
		})
	}
}

func TestMessageExample(t *testing.T) {
	tests := []struct {
		name    string
		message MessageExample
	}{
		{
			name:    "empty MessageExample",
			message: MessageExample{},
		},
		{
			name: "MessageExample with fields",
			message: MessageExample{
				Name: "User",
				Fields: []SampleField{
					{Name: "id", Type: "int64", SampleValue: "123"},
					{Name: "name", Type: "string", SampleValue: "John Doe"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.message.Name
			_ = tt.message.Fields
		})
	}
}

func TestSampleField(t *testing.T) {
	tests := []struct {
		name  string
		field SampleField
	}{
		{
			name:  "empty SampleField",
			field: SampleField{},
		},
		{
			name: "simple field",
			field: SampleField{
				Name:        "username",
				Type:        "string",
				SampleValue: "johndoe",
				IsRepeated:  false,
			},
		},
		{
			name: "repeated field",
			field: SampleField{
				Name:        "tags",
				Type:        "string",
				SampleValue: "tag1",
				IsRepeated:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.field.Name
			_ = tt.field.Type
			_ = tt.field.SampleValue
			_ = tt.field.IsRepeated
		})
	}
}

func TestPackageManagerInfo(t *testing.T) {
	tests := []struct {
		name string
		pm   PackageManagerInfo
	}{
		{
			name: "empty PackageManagerInfo",
			pm:   PackageManagerInfo{},
		},
		{
			name: "go package manager",
			pm: PackageManagerInfo{
				Command:        "go get",
				PackageName:    "github.com/example/pkg",
				InstallExample: "go get github.com/example/pkg",
			},
		},
		{
			name: "npm package manager",
			pm: PackageManagerInfo{
				Command:        "npm install",
				PackageName:    "@example/package",
				InstallExample: "npm install @example/package",
			},
		},
		{
			name: "pip package manager",
			pm: PackageManagerInfo{
				Command:        "pip install",
				PackageName:    "example-package",
				InstallExample: "pip install example-package",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.pm.Command
			_ = tt.pm.PackageName
			_ = tt.pm.InstallExample
		})
	}
}
