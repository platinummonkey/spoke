package compatibility

import (
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// SchemaGraph represents a complete parsed protobuf schema with enhanced metadata
type SchemaGraph struct {
	Package      string
	Syntax       string // "proto2" or "proto3"
	Imports      []Import
	Messages     map[string]*Message  // Fully qualified name -> Message
	Enums        map[string]*Enum     // Fully qualified name -> Enum
	Services     map[string]*Service  // Fully qualified name -> Service
	Dependencies map[string]*SchemaGraph // Import path -> dependency graph
}

// Import represents an import statement
type Import struct {
	Path   string
	Public bool
	Weak   bool
}

// Message represents a protobuf message with all fields
type Message struct {
	Name         string
	FullName     string // package.Message or package.Outer.Inner
	Fields       map[int]*Field  // Field number -> Field (for fast lookup)
	FieldsByName map[string]*Field
	Reserved     *Reserved
	Nested       map[string]*Message
	NestedEnums  map[string]*Enum
	OneOfs       map[string]*OneOf
	Options      map[string]string
}

// Field represents a message field with complete metadata
type Field struct {
	Name         string
	Number       int
	Type         FieldType
	Label        FieldLabel // optional, required, repeated
	TypeName     string     // For message/enum types
	IsMap        bool
	MapKeyType   string
	MapValueType string
	InOneOf      string     // OneOf name if part of oneof
	Deprecated   bool
	Packed       *bool
	DefaultValue string
}

// FieldType represents the protobuf field type
type FieldType int

const (
	FieldTypeUnknown FieldType = iota
	FieldTypeDouble
	FieldTypeFloat
	FieldTypeInt32
	FieldTypeInt64
	FieldTypeUint32
	FieldTypeUint64
	FieldTypeSint32
	FieldTypeSint64
	FieldTypeFixed32
	FieldTypeFixed64
	FieldTypeSfixed32
	FieldTypeSfixed64
	FieldTypeBool
	FieldTypeString
	FieldTypeBytes
	FieldTypeMessage
	FieldTypeEnum
)

func (ft FieldType) String() string {
	return []string{
		"unknown", "double", "float", "int32", "int64", "uint32", "uint64",
		"sint32", "sint64", "fixed32", "fixed64", "sfixed32", "sfixed64",
		"bool", "string", "bytes", "message", "enum",
	}[ft]
}

// FieldLabel represents field cardinality
type FieldLabel int

const (
	FieldLabelOptional FieldLabel = iota
	FieldLabelRequired
	FieldLabelRepeated
)

func (fl FieldLabel) String() string {
	return []string{"optional", "required", "repeated"}[fl]
}

// Enum represents an enum with values
type Enum struct {
	Name         string
	FullName     string
	Values       map[int]*EnumValue  // Number -> Value
	ValuesByName map[string]*EnumValue
	Reserved     *Reserved
	Options      map[string]string
}

// EnumValue represents an enum value
type EnumValue struct {
	Name       string
	Number     int
	Deprecated bool
}

// Service represents a gRPC service
type Service struct {
	Name     string
	FullName string
	Methods  map[string]*Method
}

// Method represents an RPC method
type Method struct {
	Name            string
	InputType       string
	OutputType      string
	ClientStreaming bool
	ServerStreaming bool
	Deprecated      bool
}

// Reserved tracks reserved fields
type Reserved struct {
	Numbers []int
	Ranges  [][2]int
	Names   []string
}

// OneOf represents a oneof group
type OneOf struct {
	Name   string
	Fields []int // Field numbers in this oneof
}

// SchemaGraphBuilder converts protobuf AST to SchemaGraph
type SchemaGraphBuilder struct {
	currentPackage string
	imports        map[string]*SchemaGraph
}

// NewSchemaGraphBuilder creates a new builder
func NewSchemaGraphBuilder() *SchemaGraphBuilder {
	return &SchemaGraphBuilder{
		imports: make(map[string]*SchemaGraph),
	}
}

// BuildFromAST converts a protobuf AST to a SchemaGraph
func (b *SchemaGraphBuilder) BuildFromAST(ast *protobuf.RootNode) (*SchemaGraph, error) {
	// TODO: Implement AST traversal and conversion
	// This will walk the existing protobuf.RootNode and build the enhanced SchemaGraph
	graph := &SchemaGraph{
		Package:  b.extractPackage(ast),
		Syntax:   b.extractSyntax(ast),
		Imports:  b.extractImports(ast),
		Messages: make(map[string]*Message),
		Enums:    make(map[string]*Enum),
		Services: make(map[string]*Service),
	}

	return graph, nil
}

func (b *SchemaGraphBuilder) extractPackage(ast *protobuf.RootNode) string {
	if ast.Package != nil {
		return ast.Package.Name
	}
	return ""
}

func (b *SchemaGraphBuilder) extractSyntax(ast *protobuf.RootNode) string {
	if ast.Syntax != nil {
		return ast.Syntax.Syntax
	}
	return "proto2" // default
}

func (b *SchemaGraphBuilder) extractImports(ast *protobuf.RootNode) []Import {
	var imports []Import
	for _, imp := range ast.Imports {
		imports = append(imports, Import{
			Path:   imp.Path,
			Public: imp.Public,
			Weak:   imp.Weak,
		})
	}
	return imports
}
