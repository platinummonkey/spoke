package validation

import (
	"fmt"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// Serializer converts a normalized AST back to protobuf source
type Serializer struct {
	config *NormalizationConfig
	indent string
}

// NewSerializer creates a new serializer
func NewSerializer(config *NormalizationConfig) *Serializer {
	return &Serializer{
		config: config,
		indent: "  ", // 2 spaces
	}
}

// Serialize converts an AST to protobuf source code
func (s *Serializer) Serialize(ast *protobuf.RootNode) (string, error) {
	var builder strings.Builder

	// Syntax
	if ast.Syntax != nil {
		builder.WriteString(fmt.Sprintf("syntax = \"%s\";\n\n", ast.Syntax.Value))
	}

	// Package
	if ast.Package != nil {
		builder.WriteString(fmt.Sprintf("package %s;\n\n", ast.Package.Name))
	}

	// Imports
	if len(ast.Imports) > 0 {
		for _, imp := range ast.Imports {
			s.writeImport(&builder, imp)
		}
		builder.WriteString("\n")
	}

	// Options
	for _, opt := range ast.Options {
		builder.WriteString(fmt.Sprintf("option %s = %s;\n", opt.Name, opt.Value))
	}
	if len(ast.Options) > 0 {
		builder.WriteString("\n")
	}

	// Messages
	for _, msg := range ast.Messages {
		s.writeMessage(&builder, msg, 0)
		builder.WriteString("\n")
	}

	// Enums
	for _, enum := range ast.Enums {
		s.writeEnum(&builder, enum, 0)
		builder.WriteString("\n")
	}

	// Services
	for _, svc := range ast.Services {
		s.writeService(&builder, svc)
		builder.WriteString("\n")
	}

	result := builder.String()

	// Remove trailing whitespace if configured
	if s.config.RemoveTrailingWhitespace {
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			lines[i] = strings.TrimRight(line, " \t")
		}
		result = strings.Join(lines, "\n")
	}

	return result, nil
}

func (s *Serializer) writeImport(builder *strings.Builder, imp *protobuf.ImportNode) {
	modifier := ""
	if imp.Public {
		modifier = "public "
	} else if imp.Weak {
		modifier = "weak "
	}
	builder.WriteString(fmt.Sprintf("import %s\"%s\";\n", modifier, imp.Path))
}

func (s *Serializer) writeMessage(builder *strings.Builder, msg *protobuf.MessageNode, depth int) {
	indent := s.indentation(depth)

	// Message declaration
	builder.WriteString(fmt.Sprintf("%smessage %s {\n", indent, msg.Name))

	// Nested enums
	for _, enum := range msg.Enums {
		s.writeEnum(builder, enum, depth+1)
	}

	// Nested messages
	for _, nested := range msg.Nested {
		s.writeMessage(builder, nested, depth+1)
	}

	// OneOfs
	for _, oneof := range msg.OneOfs {
		s.writeOneOf(builder, oneof, depth+1)
	}

	// Fields
	for _, field := range msg.Fields {
		s.writeField(builder, field, depth+1)
	}

	// Options
	for _, opt := range msg.Options {
		builder.WriteString(fmt.Sprintf("%s%soption %s = %s;\n", indent, s.indent, opt.Name, opt.Value))
	}

	builder.WriteString(fmt.Sprintf("%s}\n", indent))
}

func (s *Serializer) writeField(builder *strings.Builder, field *protobuf.FieldNode, depth int) {
	indent := s.indentation(depth)

	// Field label
	label := ""
	if field.Repeated {
		label = "repeated "
	} else if field.Optional {
		label = "optional "
	} else if field.Required {
		label = "required "
	}

	builder.WriteString(fmt.Sprintf("%s%s%s %s = %d;\n",
		indent, label, field.Type, field.Name, field.Number))
}

func (s *Serializer) writeOneOf(builder *strings.Builder, oneof *protobuf.OneOfNode, depth int) {
	indent := s.indentation(depth)

	builder.WriteString(fmt.Sprintf("%soneof %s {\n", indent, oneof.Name))

	for _, field := range oneof.Fields {
		s.writeField(builder, field, depth+1)
	}

	builder.WriteString(fmt.Sprintf("%s}\n", indent))
}

func (s *Serializer) writeEnum(builder *strings.Builder, enum *protobuf.EnumNode, depth int) {
	indent := s.indentation(depth)

	builder.WriteString(fmt.Sprintf("%senum %s {\n", indent, enum.Name))

	for _, value := range enum.Values {
		builder.WriteString(fmt.Sprintf("%s%s%s = %d;\n",
			indent, s.indent, value.Name, value.Number))
	}

	// Options
	for _, opt := range enum.Options {
		builder.WriteString(fmt.Sprintf("%s%soption %s = %s;\n", indent, s.indent, opt.Name, opt.Value))
	}

	builder.WriteString(fmt.Sprintf("%s}\n", indent))
}

func (s *Serializer) writeService(builder *strings.Builder, svc *protobuf.ServiceNode) {
	builder.WriteString(fmt.Sprintf("service %s {\n", svc.Name))

	for _, rpc := range svc.RPCs {
		s.writeRPC(builder, rpc)
	}

	// Options
	for _, opt := range svc.Options {
		builder.WriteString(fmt.Sprintf("%soption %s = %s;\n", s.indent, opt.Name, opt.Value))
	}

	builder.WriteString("}\n")
}

func (s *Serializer) writeRPC(builder *strings.Builder, rpc *protobuf.RPCNode) {
	inputType := rpc.InputType
	if rpc.ClientStreaming {
		inputType = "stream " + inputType
	}

	outputType := rpc.OutputType
	if rpc.ServerStreaming {
		outputType = "stream " + outputType
	}

	builder.WriteString(fmt.Sprintf("%srpc %s(%s) returns (%s);\n",
		s.indent, rpc.Name, inputType, outputType))
}

func (s *Serializer) indentation(depth int) string {
	if depth == 0 {
		return ""
	}
	return strings.Repeat(s.indent, depth)
}
