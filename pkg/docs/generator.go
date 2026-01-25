package docs

import (
	"fmt"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// Documentation represents generated documentation for a proto schema
type Documentation struct {
	PackageName string
	Syntax      string
	Description string
	Messages    []*MessageDoc
	Enums       []*EnumDoc
	Services    []*ServiceDoc
	Imports     []string
	Options     map[string]string
}

// MessageDoc represents documentation for a message
type MessageDoc struct {
	Name        string
	FullName    string
	Description string
	Fields      []*FieldDoc
	NestedTypes []*MessageDoc
	Enums       []*EnumDoc
	Deprecated  bool
	Location    string
}

// FieldDoc represents documentation for a field
type FieldDoc struct {
	Name        string
	Number      int
	Type        string
	Label       string
	Description string
	Deprecated  bool
	Required    bool
	Optional    bool
	Repeated    bool
	OneofName   string
}

// EnumDoc represents documentation for an enum
type EnumDoc struct {
	Name        string
	FullName    string
	Description string
	Values      []*EnumValueDoc
	Deprecated  bool
}

// EnumValueDoc represents documentation for an enum value
type EnumValueDoc struct {
	Name        string
	Number      int
	Description string
	Deprecated  bool
}

// ServiceDoc represents documentation for a service
type ServiceDoc struct {
	Name        string
	Description string
	Methods     []*MethodDoc
	Deprecated  bool
}

// MethodDoc represents documentation for a service method
type MethodDoc struct {
	Name              string
	Description       string
	RequestType       string
	ResponseType      string
	ClientStreaming   bool
	ServerStreaming   bool
	Deprecated        bool
	HTTPMethod        string
	HTTPPath          string
}

// Generator generates documentation from proto AST
type Generator struct{}

// NewGenerator creates a new documentation generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates documentation from a proto AST
func (g *Generator) Generate(ast *protobuf.RootNode) (*Documentation, error) {
	doc := &Documentation{
		Messages: make([]*MessageDoc, 0),
		Enums:    make([]*EnumDoc, 0),
		Services: make([]*ServiceDoc, 0),
		Imports:  make([]string, 0),
		Options:  make(map[string]string),
	}

	// Extract syntax
	if ast.Syntax != nil {
		doc.Syntax = ast.Syntax.Value
	}

	// Extract package
	if ast.Package != nil {
		doc.PackageName = ast.Package.Name
	}

	// Extract imports
	for _, imp := range ast.Imports {
		doc.Imports = append(doc.Imports, imp.Path)
	}

	// Extract messages
	for _, msg := range ast.Messages {
		messageDoc := g.generateMessageDoc(msg, doc.PackageName)
		doc.Messages = append(doc.Messages, messageDoc)
	}

	// Extract enums
	for _, enum := range ast.Enums {
		enumDoc := g.generateEnumDoc(enum, doc.PackageName)
		doc.Enums = append(doc.Enums, enumDoc)
	}

	// Extract services
	for _, svc := range ast.Services {
		serviceDoc := g.generateServiceDoc(svc)
		doc.Services = append(doc.Services, serviceDoc)
	}

	return doc, nil
}

// generateMessageDoc generates documentation for a message
func (g *Generator) generateMessageDoc(msg *protobuf.MessageNode, packageName string) *MessageDoc {
	doc := &MessageDoc{
		Name:        msg.Name,
		FullName:    fmt.Sprintf("%s.%s", packageName, msg.Name),
		Description: extractComments(msg.Comments),
		Fields:      make([]*FieldDoc, 0),
		NestedTypes: make([]*MessageDoc, 0),
		Enums:       make([]*EnumDoc, 0),
		Location:    fmt.Sprintf("%s.%s", packageName, msg.Name),
	}

	// Extract fields
	for _, field := range msg.Fields {
		fieldDoc := g.generateFieldDoc(field)
		doc.Fields = append(doc.Fields, fieldDoc)
	}

	// Extract nested messages
	for _, nested := range msg.Nested {
		nestedDoc := g.generateMessageDoc(nested, doc.FullName)
		doc.NestedTypes = append(doc.NestedTypes, nestedDoc)
	}

	// Extract nested enums
	for _, enum := range msg.Enums {
		enumDoc := g.generateEnumDoc(enum, doc.FullName)
		doc.Enums = append(doc.Enums, enumDoc)
	}

	return doc
}

// generateFieldDoc generates documentation for a field
func (g *Generator) generateFieldDoc(field *protobuf.FieldNode) *FieldDoc {
	doc := &FieldDoc{
		Name:        field.Name,
		Number:      field.Number,
		Type:        field.Type,
		Description: extractComments(field.Comments),
	}

	// Determine field label
	if field.Repeated {
		doc.Label = "repeated"
		doc.Repeated = true
	} else if field.Optional {
		doc.Label = "optional"
		doc.Optional = true
	} else if field.Required {
		doc.Label = "required"
		doc.Required = true
	}

	return doc
}

// generateEnumDoc generates documentation for an enum
func (g *Generator) generateEnumDoc(enum *protobuf.EnumNode, packageName string) *EnumDoc {
	doc := &EnumDoc{
		Name:        enum.Name,
		FullName:    fmt.Sprintf("%s.%s", packageName, enum.Name),
		Description: extractComments(enum.Comments),
		Values:      make([]*EnumValueDoc, 0),
	}

	for _, value := range enum.Values {
		valueDoc := &EnumValueDoc{
			Name:        value.Name,
			Number:      value.Number,
			Description: extractComments(value.Comments),
		}
		doc.Values = append(doc.Values, valueDoc)
	}

	return doc
}

// generateServiceDoc generates documentation for a service
func (g *Generator) generateServiceDoc(svc *protobuf.ServiceNode) *ServiceDoc {
	doc := &ServiceDoc{
		Name:        svc.Name,
		Description: extractComments(svc.Comments),
		Methods:     make([]*MethodDoc, 0),
	}

	for _, rpc := range svc.RPCs {
		methodDoc := &MethodDoc{
			Name:            rpc.Name,
			Description:     extractComments(rpc.Comments),
			RequestType:     rpc.InputType,
			ResponseType:    rpc.OutputType,
			ClientStreaming: rpc.ClientStreaming,
			ServerStreaming: rpc.ServerStreaming,
		}

		// Extract HTTP annotations if present
		// This would parse google.api.http options
		// For now, we'll leave these empty

		doc.Methods = append(doc.Methods, methodDoc)
	}

	return doc
}

// extractComments extracts and formats comments
func extractComments(comments []*protobuf.CommentNode) string {
	if len(comments) == 0 {
		return ""
	}

	var lines []string
	for _, comment := range comments {
		text := strings.TrimSpace(comment.Text)
		// Remove leading comment markers
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)
		if text != "" {
			lines = append(lines, text)
		}
	}

	return strings.Join(lines, "\n")
}

// Summary returns a summary of the documentation
func (d *Documentation) Summary() string {
	return fmt.Sprintf("Package: %s, Messages: %d, Enums: %d, Services: %d",
		d.PackageName, len(d.Messages), len(d.Enums), len(d.Services))
}

// FindMessage finds a message by name
func (d *Documentation) FindMessage(name string) *MessageDoc {
	for _, msg := range d.Messages {
		if msg.Name == name || msg.FullName == name {
			return msg
		}
		// Check nested messages
		if nested := findNestedMessage(msg, name); nested != nil {
			return nested
		}
	}
	return nil
}

// findNestedMessage recursively finds a nested message
func findNestedMessage(msg *MessageDoc, name string) *MessageDoc {
	for _, nested := range msg.NestedTypes {
		if nested.Name == name || nested.FullName == name {
			return nested
		}
		if found := findNestedMessage(nested, name); found != nil {
			return found
		}
	}
	return nil
}

// FindEnum finds an enum by name
func (d *Documentation) FindEnum(name string) *EnumDoc {
	for _, enum := range d.Enums {
		if enum.Name == name || enum.FullName == name {
			return enum
		}
	}
	// Check nested enums
	for _, msg := range d.Messages {
		if enum := findNestedEnum(msg, name); enum != nil {
			return enum
		}
	}
	return nil
}

// findNestedEnum recursively finds a nested enum
func findNestedEnum(msg *MessageDoc, name string) *EnumDoc {
	for _, enum := range msg.Enums {
		if enum.Name == name || enum.FullName == name {
			return enum
		}
	}
	for _, nested := range msg.NestedTypes {
		if enum := findNestedEnum(nested, name); enum != nil {
			return enum
		}
	}
	return nil
}

// FindService finds a service by name
func (d *Documentation) FindService(name string) *ServiceDoc {
	for _, svc := range d.Services {
		if svc.Name == name {
			return svc
		}
	}
	return nil
}
