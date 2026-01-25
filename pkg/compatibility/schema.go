package compatibility

import (
	"strings"

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
	FieldTypeMap
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
	graph := &SchemaGraph{
		Package:  b.extractPackage(ast),
		Syntax:   b.extractSyntax(ast),
		Imports:  b.extractImports(ast),
		Messages: make(map[string]*Message),
		Enums:    make(map[string]*Enum),
		Services: make(map[string]*Service),
	}

	b.currentPackage = graph.Package

	// Extract top-level messages
	for _, msgNode := range ast.Messages {
		msg := b.buildMessage(msgNode, "")
		graph.Messages[msg.Name] = msg
	}

	// Extract top-level enums
	for _, enumNode := range ast.Enums {
		enum := b.buildEnum(enumNode, "")
		graph.Enums[enum.Name] = enum
	}

	// Extract services
	for _, svcNode := range ast.Services {
		svc := b.buildService(svcNode)
		graph.Services[svc.Name] = svc
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
		return ast.Syntax.Value
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

func (b *SchemaGraphBuilder) buildMessage(msgNode *protobuf.MessageNode, parentPrefix string) *Message {
	fullName := b.makeFullName(parentPrefix, msgNode.Name)

	msg := &Message{
		Name:         msgNode.Name,
		FullName:     fullName,
		Fields:       make(map[int]*Field),
		FieldsByName: make(map[string]*Field),
		Nested:       make(map[string]*Message),
		NestedEnums:  make(map[string]*Enum),
		OneOfs:       make(map[string]*OneOf),
		Options:      make(map[string]string),
	}

	// Extract fields
	for _, fieldNode := range msgNode.Fields {
		field := b.buildField(fieldNode, "")
		msg.Fields[field.Number] = field
		msg.FieldsByName[field.Name] = field
	}

	// Extract oneofs
	for _, oneofNode := range msgNode.OneOfs {
		oneof := &OneOf{
			Name:   oneofNode.Name,
			Fields: make([]int, 0),
		}
		for _, fieldNode := range oneofNode.Fields {
			field := b.buildField(fieldNode, oneofNode.Name)
			msg.Fields[field.Number] = field
			msg.FieldsByName[field.Name] = field
			oneof.Fields = append(oneof.Fields, field.Number)
		}
		msg.OneOfs[oneof.Name] = oneof
	}

	// Extract nested messages
	for _, nestedNode := range msgNode.Nested {
		nested := b.buildMessage(nestedNode, fullName)
		msg.Nested[nested.Name] = nested
	}

	// Extract nested enums
	for _, enumNode := range msgNode.Enums {
		enum := b.buildEnum(enumNode, fullName)
		msg.NestedEnums[enum.Name] = enum
	}

	return msg
}

func (b *SchemaGraphBuilder) buildField(fieldNode *protobuf.FieldNode, oneofName string) *Field {
	field := &Field{
		Name:       fieldNode.Name,
		Number:     fieldNode.Number,
		Type:       b.parseFieldType(fieldNode.Type),
		TypeName:   fieldNode.Type,
		Label:      b.parseFieldLabel(fieldNode),
		InOneOf:    oneofName,
		Deprecated: b.hasDeprecatedOption(fieldNode.Options),
	}

	// Check if it's a map field (proto3: map<key, value>)
	if strings.HasPrefix(fieldNode.Type, "map<") {
		field.IsMap = true
	}

	return field
}

func (b *SchemaGraphBuilder) buildEnum(enumNode *protobuf.EnumNode, parentPrefix string) *Enum {
	fullName := b.makeFullName(parentPrefix, enumNode.Name)

	enum := &Enum{
		Name:         enumNode.Name,
		FullName:     fullName,
		Values:       make(map[int]*EnumValue),
		ValuesByName: make(map[string]*EnumValue),
		Options:      make(map[string]string),
	}

	for _, valNode := range enumNode.Values {
		val := &EnumValue{
			Name:       valNode.Name,
			Number:     valNode.Number,
			Deprecated: b.hasDeprecatedOption(valNode.Options),
		}
		enum.Values[val.Number] = val
		enum.ValuesByName[val.Name] = val
	}

	return enum
}

func (b *SchemaGraphBuilder) buildService(svcNode *protobuf.ServiceNode) *Service {
	fullName := b.makeFullName(b.currentPackage, svcNode.Name)

	svc := &Service{
		Name:     svcNode.Name,
		FullName: fullName,
		Methods:  make(map[string]*Method),
	}

	for _, rpcNode := range svcNode.RPCs {
		method := &Method{
			Name:            rpcNode.Name,
			InputType:       rpcNode.InputType,
			OutputType:      rpcNode.OutputType,
			ClientStreaming: rpcNode.ClientStreaming,
			ServerStreaming: rpcNode.ServerStreaming,
			Deprecated:      b.hasDeprecatedOption(rpcNode.Options),
		}
		svc.Methods[method.Name] = method
	}

	return svc
}

func (b *SchemaGraphBuilder) makeFullName(prefix, name string) string {
	if prefix == "" {
		if b.currentPackage == "" {
			return name
		}
		return b.currentPackage + "." + name
	}
	return prefix + "." + name
}

func (b *SchemaGraphBuilder) parseFieldType(typeName string) FieldType {
	switch typeName {
	case "double":
		return FieldTypeDouble
	case "float":
		return FieldTypeFloat
	case "int32":
		return FieldTypeInt32
	case "int64":
		return FieldTypeInt64
	case "uint32":
		return FieldTypeUint32
	case "uint64":
		return FieldTypeUint64
	case "sint32":
		return FieldTypeSint32
	case "sint64":
		return FieldTypeSint64
	case "fixed32":
		return FieldTypeFixed32
	case "fixed64":
		return FieldTypeFixed64
	case "sfixed32":
		return FieldTypeSfixed32
	case "sfixed64":
		return FieldTypeSfixed64
	case "bool":
		return FieldTypeBool
	case "string":
		return FieldTypeString
	case "bytes":
		return FieldTypeBytes
	default:
		if strings.HasPrefix(typeName, "map<") {
			return FieldTypeMap
		}
		// Custom message or enum type
		return FieldTypeMessage
	}
}

func (b *SchemaGraphBuilder) parseFieldLabel(fieldNode *protobuf.FieldNode) FieldLabel {
	if fieldNode.Repeated {
		return FieldLabelRepeated
	}
	if fieldNode.Optional {
		return FieldLabelOptional
	}
	if fieldNode.Required {
		return FieldLabelRequired
	}
	return FieldLabelOptional // proto3 default
}

func (b *SchemaGraphBuilder) hasDeprecatedOption(options []*protobuf.OptionNode) bool {
	for _, opt := range options {
		if opt.Name == "deprecated" && opt.Value == "true" {
			return true
		}
	}
	return false
}
