package examples

// ExampleData contains all the data needed to generate a code example
type ExampleData struct {
	Language       string
	ModuleName     string
	Version        string
	PackagePath    string
	ServiceName    string
	Methods        []MethodExample
	Messages       []MessageExample
	Imports        []string
	PackageManager PackageManagerInfo
}

// MethodExample represents a single RPC method
type MethodExample struct {
	Name             string
	RequestType      string
	ResponseType     string
	SampleFields     []SampleField
	IsClientStream   bool
	IsServerStream   bool
	IsBidirectional  bool
}

// MessageExample represents a protobuf message
type MessageExample struct {
	Name   string
	Fields []SampleField
}

// SampleField represents a field with a sample value
type SampleField struct {
	Name        string
	Type        string
	SampleValue string
	IsRepeated  bool
}

// PackageManagerInfo contains package manager specific information
type PackageManagerInfo struct {
	Command        string // e.g., "go get", "pip install", "npm install"
	PackageName    string
	InstallExample string
}
