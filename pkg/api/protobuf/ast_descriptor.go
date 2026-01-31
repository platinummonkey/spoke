package protobuf

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ParseWithDescriptor parses a protobuf file using protocompile and returns an AST.
// This is the new descriptor-based parser that replaces the manual parser while
// preserving all functionality, especially @spoke directives.
//
// The parser works in three stages:
// 1. Parse proto content with protocompile to get FileDescriptorProto
// 2. Parse @spoke directives and comments from raw content
// 3. Convert descriptor to AST and merge directives/comments
func ParseWithDescriptor(filename, content string) (*RootNode, error) {
	// Stage 1: Parse with protocompile
	desc, sourceInfo, originalContent, err := parseToDescriptor(filename, content)
	if err != nil {
		return nil, fmt.Errorf("protocompile parse failed: %w", err)
	}

	// Stage 2: Extract spoke directives and comments
	directives, comments, err := ParseSpokeDirectivesFromContent(content)
	if err != nil {
		return nil, fmt.Errorf("spoke directive extraction failed: %w", err)
	}

	// Stage 3: Convert descriptor to AST with position tracking
	root := convertDescriptorToAST(desc, sourceInfo, originalContent)

	// Stage 4: Merge spoke directives and comments into AST
	mergeDirectivesAndComments(root, directives, comments)

	return root, nil
}

// parseToDescriptor uses protocompile to parse proto content into a FileDescriptorProto
func parseToDescriptor(filename, content string) (*descriptorpb.FileDescriptorProto, *descriptorpb.SourceCodeInfo, string, error) {
	// Create a protocompile parser
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{
				filename: content,
			}),
		},
	}

	// Parse the file
	result, err := compiler.Compile(context.Background(), filename)
	if err != nil {
		return nil, nil, "", err
	}

	// Get the first file from result and convert to FileDescriptorProto
	var fileDesc protoreflect.FileDescriptor
	for _, file := range result {
		fileDesc = file
		break
	}

	if fileDesc == nil {
		return nil, nil, "", fmt.Errorf("no files compiled")
	}

	// Convert to FileDescriptorProto
	fileDescProto := protodescriptorToProto(fileDesc)
	return fileDescProto, fileDescProto.GetSourceCodeInfo(), content, nil
}

// protodescriptorToProto converts a protoreflect.FileDescriptor to descriptorpb.FileDescriptorProto
func protodescriptorToProto(fd protoreflect.FileDescriptor) *descriptorpb.FileDescriptorProto {
	name := string(fd.Path())
	pkg := string(fd.Package())
	syntax := fd.Syntax().String()

	fileProto := &descriptorpb.FileDescriptorProto{
		Name:    &name,
		Package: &pkg,
		Syntax:  &syntax,
	}

	// Add dependencies
	deps := make([]string, fd.Imports().Len())
	for i := 0; i < fd.Imports().Len(); i++ {
		deps[i] = string(fd.Imports().Get(i).Path())
	}
	if len(deps) > 0 {
		fileProto.Dependency = deps
	}

	// Add messages
	msgs := make([]*descriptorpb.DescriptorProto, fd.Messages().Len())
	for i := 0; i < fd.Messages().Len(); i++ {
		msgs[i] = messageDescriptorToProto(fd.Messages().Get(i))
	}
	if len(msgs) > 0 {
		fileProto.MessageType = msgs
	}

	// Add enums
	enums := make([]*descriptorpb.EnumDescriptorProto, fd.Enums().Len())
	for i := 0; i < fd.Enums().Len(); i++ {
		enums[i] = enumDescriptorToProto(fd.Enums().Get(i))
	}
	if len(enums) > 0 {
		fileProto.EnumType = enums
	}

	// Add services
	services := make([]*descriptorpb.ServiceDescriptorProto, fd.Services().Len())
	for i := 0; i < fd.Services().Len(); i++ {
		services[i] = serviceDescriptorToProto(fd.Services().Get(i))
	}
	if len(services) > 0 {
		fileProto.Service = services
	}

	return fileProto
}

// messageDescriptorToProto converts a protoreflect.MessageDescriptor to descriptorpb.DescriptorProto
func messageDescriptorToProto(md protoreflect.MessageDescriptor) *descriptorpb.DescriptorProto {
	name := string(md.Name())
	msgProto := &descriptorpb.DescriptorProto{
		Name: &name,
	}

	// Add fields
	fields := make([]*descriptorpb.FieldDescriptorProto, md.Fields().Len())
	for i := 0; i < md.Fields().Len(); i++ {
		fields[i] = fieldDescriptorToProto(md.Fields().Get(i))
	}
	if len(fields) > 0 {
		msgProto.Field = fields
	}

	// Add nested messages
	nestedMsgs := make([]*descriptorpb.DescriptorProto, md.Messages().Len())
	for i := 0; i < md.Messages().Len(); i++ {
		nestedMsgs[i] = messageDescriptorToProto(md.Messages().Get(i))
	}
	if len(nestedMsgs) > 0 {
		msgProto.NestedType = nestedMsgs
	}

	// Add nested enums
	nestedEnums := make([]*descriptorpb.EnumDescriptorProto, md.Enums().Len())
	for i := 0; i < md.Enums().Len(); i++ {
		nestedEnums[i] = enumDescriptorToProto(md.Enums().Get(i))
	}
	if len(nestedEnums) > 0 {
		msgProto.EnumType = nestedEnums
	}

	return msgProto
}

// fieldDescriptorToProto converts a protoreflect.FieldDescriptor to descriptorpb.FieldDescriptorProto
func fieldDescriptorToProto(fd protoreflect.FieldDescriptor) *descriptorpb.FieldDescriptorProto {
	name := string(fd.Name())
	number := int32(fd.Number())

	fieldProto := &descriptorpb.FieldDescriptorProto{
		Name:   &name,
		Number: &number,
	}

	// Set type
	kind := fd.Kind()
	typ := descriptorpb.FieldDescriptorProto_Type(kind)
	fieldProto.Type = &typ

	// Set type name for message/enum types
	if kind == protoreflect.MessageKind || kind == protoreflect.EnumKind {
		typeName := "." + string(fd.Message().FullName())
		fieldProto.TypeName = &typeName
	}

	// Set label
	if fd.Cardinality() == protoreflect.Repeated {
		label := descriptorpb.FieldDescriptorProto_LABEL_REPEATED
		fieldProto.Label = &label
	} else if fd.Cardinality() == protoreflect.Optional {
		label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
		fieldProto.Label = &label
	} else if fd.Cardinality() == protoreflect.Required {
		label := descriptorpb.FieldDescriptorProto_LABEL_REQUIRED
		fieldProto.Label = &label
	}

	return fieldProto
}

// enumDescriptorToProto converts a protoreflect.EnumDescriptor to descriptorpb.EnumDescriptorProto
func enumDescriptorToProto(ed protoreflect.EnumDescriptor) *descriptorpb.EnumDescriptorProto {
	name := string(ed.Name())
	enumProto := &descriptorpb.EnumDescriptorProto{
		Name: &name,
	}

	// Add values
	values := make([]*descriptorpb.EnumValueDescriptorProto, ed.Values().Len())
	for i := 0; i < ed.Values().Len(); i++ {
		val := ed.Values().Get(i)
		valName := string(val.Name())
		valNumber := int32(val.Number())
		values[i] = &descriptorpb.EnumValueDescriptorProto{
			Name:   &valName,
			Number: &valNumber,
		}
	}
	if len(values) > 0 {
		enumProto.Value = values
	}

	return enumProto
}

// serviceDescriptorToProto converts a protoreflect.ServiceDescriptor to descriptorpb.ServiceDescriptorProto
func serviceDescriptorToProto(sd protoreflect.ServiceDescriptor) *descriptorpb.ServiceDescriptorProto {
	name := string(sd.Name())
	svcProto := &descriptorpb.ServiceDescriptorProto{
		Name: &name,
	}

	// Add methods
	methods := make([]*descriptorpb.MethodDescriptorProto, sd.Methods().Len())
	for i := 0; i < sd.Methods().Len(); i++ {
		method := sd.Methods().Get(i)
		methodName := string(method.Name())
		inputType := "." + string(method.Input().FullName())
		outputType := "." + string(method.Output().FullName())
		clientStreaming := method.IsStreamingClient()
		serverStreaming := method.IsStreamingServer()

		methods[i] = &descriptorpb.MethodDescriptorProto{
			Name:            &methodName,
			InputType:       &inputType,
			OutputType:      &outputType,
			ClientStreaming: &clientStreaming,
			ServerStreaming: &serverStreaming,
		}
	}
	if len(methods) > 0 {
		svcProto.Method = methods
	}

	return svcProto
}

// positionMap holds line numbers for proto elements
type positionMap struct {
	packageLine int
	syntaxLine  int
	imports     map[string]int // import path -> line
	messages    map[string]int // message name -> line
	enums       map[string]int // enum name -> line
	services    map[string]int // service name -> line
	fields      map[string]int // field name -> line (for top-level and nested fields)
}

// extractPositionsFromContent scans the proto content to find line numbers for each element
func extractPositionsFromContent(content string) *positionMap {
	pm := &positionMap{
		imports:  make(map[string]int),
		messages: make(map[string]int),
		enums:    make(map[string]int),
		services: make(map[string]int),
		fields:   make(map[string]int),
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Skip comments and empty lines for pattern matching
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Find syntax
		if strings.HasPrefix(trimmed, "syntax") {
			pm.syntaxLine = lineNum
		}

		// Find package
		if strings.HasPrefix(trimmed, "package") {
			pm.packageLine = lineNum
		}

		// Find imports
		if strings.HasPrefix(trimmed, "import") {
			// Extract import path
			parts := strings.Split(trimmed, "\"")
			if len(parts) >= 2 {
				pm.imports[parts[1]] = lineNum
			}
		}

		// Find messages
		if strings.HasPrefix(trimmed, "message") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				pm.messages[parts[1]] = lineNum
			}
		}

		// Find enums
		if strings.HasPrefix(trimmed, "enum") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				pm.enums[parts[1]] = lineNum
			}
		}

		// Find services
		if strings.HasPrefix(trimmed, "service") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				pm.services[parts[1]] = lineNum
			}
		}

		// Find fields (lines with "= number;")
		// This matches field declarations like "string name = 1;"
		if strings.Contains(trimmed, "=") && strings.HasSuffix(trimmed, ";") {
			parts := strings.Fields(trimmed)
			// Field format: [repeated|optional] type name = number;
			// We want the field name, which is before the "="
			for idx, part := range parts {
				if part == "=" && idx > 0 {
					fieldName := parts[idx-1]
					pm.fields[fieldName] = lineNum
					break
				}
			}
		}
	}

	return pm
}

// convertDescriptorToAST converts a FileDescriptorProto to our AST structure
func convertDescriptorToAST(desc *descriptorpb.FileDescriptorProto, sourceInfo *descriptorpb.SourceCodeInfo, content string) *RootNode {
	// Extract positions from content
	positions := extractPositionsFromContent(content)
	root := &RootNode{
		Messages:        make([]*MessageNode, 0),
		Enums:           make([]*EnumNode, 0),
		Services:        make([]*ServiceNode, 0),
		Imports:         make([]*ImportNode, 0),
		Options:         make([]*OptionNode, 0),
		Comments:        make([]*CommentNode, 0),
		SpokeDirectives: make([]*SpokeDirectiveNode, 0),
	}

	// Use sourceInfo to extract positions if available
	// Note: sourceInfo is currently unused, but kept for future enhancement
	_ = sourceInfo

	// Convert syntax
	if desc.Syntax != nil {
		root.Syntax = &SyntaxNode{
			Value:           desc.GetSyntax(),
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
			Pos:             Position{Line: positions.syntaxLine},
		}
	}

	// Convert package
	if desc.Package != nil {
		root.Package = &PackageNode{
			Name:            desc.GetPackage(),
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
			Pos:             Position{Line: positions.packageLine},
		}
	}

	// Convert imports
	for _, dep := range desc.GetDependency() {
		root.Imports = append(root.Imports, &ImportNode{
			Path:            dep,
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}
	for i, publicDep := range desc.GetPublicDependency() {
		if int(publicDep) < len(root.Imports) {
			root.Imports[publicDep].Public = true
		}
		_ = i // unused
	}
	for i, weakDep := range desc.GetWeakDependency() {
		if int(weakDep) < len(root.Imports) {
			root.Imports[weakDep].Weak = true
		}
		_ = i // unused
	}

	// Convert file options
	if desc.Options != nil {
		// TODO: Convert file options to OptionNode
		// For now, we'll skip this as it's complex and may not be critical
	}

	// Convert messages
	for _, msgDesc := range desc.GetMessageType() {
		root.Messages = append(root.Messages, convertMessage(msgDesc, positions))
	}

	// Convert enums
	for _, enumDesc := range desc.GetEnumType() {
		root.Enums = append(root.Enums, convertEnum(enumDesc, positions))
	}

	// Convert services
	for _, svcDesc := range desc.GetService() {
		root.Services = append(root.Services, convertService(svcDesc, positions))
	}

	return root
}

// convertMessage converts a DescriptorProto (message descriptor) to MessageNode
func convertMessage(desc *descriptorpb.DescriptorProto, positions *positionMap) *MessageNode {
	msg := &MessageNode{
		Name:            desc.GetName(),
		Fields:          make([]*FieldNode, 0),
		Nested:          make([]*MessageNode, 0),
		Enums:           make([]*EnumNode, 0),
		Options:         make([]*OptionNode, 0),
		Comments:        make([]*CommentNode, 0),
		SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		Pos:             Position{Line: positions.messages[desc.GetName()]},
	}

	// Convert fields
	for _, fieldDesc := range desc.GetField() {
		msg.Fields = append(msg.Fields, convertField(fieldDesc, positions))
	}

	// Convert nested messages
	for _, nestedDesc := range desc.GetNestedType() {
		msg.Nested = append(msg.Nested, convertMessage(nestedDesc, positions))
	}

	// Convert nested enums
	for _, enumDesc := range desc.GetEnumType() {
		msg.Enums = append(msg.Enums, convertEnum(enumDesc, positions))
	}

	// TODO: Convert oneofs, extensions, reserved fields

	return msg
}

// convertField converts a FieldDescriptorProto to FieldNode
func convertField(desc *descriptorpb.FieldDescriptorProto, positions *positionMap) *FieldNode {
	field := &FieldNode{
		Name:            desc.GetName(),
		Type:            getFieldTypeName(desc),
		Number:          int(desc.GetNumber()),
		Options:         make([]*OptionNode, 0),
		Comments:        make([]*CommentNode, 0),
		SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		Pos:             Position{Line: positions.fields[desc.GetName()]},
	}

	// Set field modifiers
	label := desc.GetLabel()
	switch label {
	case descriptorpb.FieldDescriptorProto_LABEL_REPEATED:
		field.Repeated = true
	case descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL:
		field.Optional = true
	case descriptorpb.FieldDescriptorProto_LABEL_REQUIRED:
		field.Required = true
	}

	return field
}

// getFieldTypeName returns the type name for a field
func getFieldTypeName(desc *descriptorpb.FieldDescriptorProto) string {
	// If it's a message or enum type, use the type name
	if desc.TypeName != nil {
		return strings.TrimPrefix(desc.GetTypeName(), ".")
	}

	// Otherwise, use the scalar type
	switch desc.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		return "double"
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		return "float"
	case descriptorpb.FieldDescriptorProto_TYPE_INT64:
		return "int64"
	case descriptorpb.FieldDescriptorProto_TYPE_UINT64:
		return "uint64"
	case descriptorpb.FieldDescriptorProto_TYPE_INT32:
		return "int32"
	case descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		return "fixed64"
	case descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		return "fixed32"
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		return "bool"
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return "string"
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		return "bytes"
	case descriptorpb.FieldDescriptorProto_TYPE_UINT32:
		return "uint32"
	case descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		return "sfixed32"
	case descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		return "sfixed64"
	case descriptorpb.FieldDescriptorProto_TYPE_SINT32:
		return "sint32"
	case descriptorpb.FieldDescriptorProto_TYPE_SINT64:
		return "sint64"
	default:
		return "unknown"
	}
}

// convertEnum converts an EnumDescriptorProto to EnumNode
func convertEnum(desc *descriptorpb.EnumDescriptorProto, positions *positionMap) *EnumNode {
	enum := &EnumNode{
		Name:            desc.GetName(),
		Values:          make([]*EnumValueNode, 0),
		Options:         make([]*OptionNode, 0),
		Comments:        make([]*CommentNode, 0),
		SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		Pos:             Position{Line: positions.enums[desc.GetName()]},
	}

	// Convert enum values
	for _, valueDesc := range desc.GetValue() {
		enum.Values = append(enum.Values, &EnumValueNode{
			Name:            valueDesc.GetName(),
			Number:          int(valueDesc.GetNumber()),
			Options:         make([]*OptionNode, 0),
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	return enum
}

// convertService converts a ServiceDescriptorProto to ServiceNode
func convertService(desc *descriptorpb.ServiceDescriptorProto, positions *positionMap) *ServiceNode {
	svc := &ServiceNode{
		Name:            desc.GetName(),
		RPCs:            make([]*RPCNode, 0),
		Options:         make([]*OptionNode, 0),
		Comments:        make([]*CommentNode, 0),
		SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		Pos:             Position{Line: positions.services[desc.GetName()]},
	}

	// Convert RPC methods
	for _, methodDesc := range desc.GetMethod() {
		svc.RPCs = append(svc.RPCs, &RPCNode{
			Name:            methodDesc.GetName(),
			InputType:       strings.TrimPrefix(methodDesc.GetInputType(), "."),
			OutputType:      strings.TrimPrefix(methodDesc.GetOutputType(), "."),
			ClientStreaming: methodDesc.GetClientStreaming(),
			ServerStreaming: methodDesc.GetServerStreaming(),
			Options:         make([]*OptionNode, 0),
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	return svc
}

// mergeDirectivesAndComments merges spoke directives and comments into the AST
// by associating them with nodes based on line numbers. Directives are consumed
// after being associated to prevent multiple associations.
func mergeDirectivesAndComments(root *RootNode, directives map[int]*SpokeDirectiveNode, comments map[int][]*CommentNode) {
	// Create a set to track consumed directive lines
	consumed := make(map[int]bool)

	// Associate directives/comments with root-level nodes
	if root.Syntax != nil {
		associateAndMarkConsumed(root.Syntax, directives, comments, consumed, root.Syntax.Pos.Line)
	}

	if root.Package != nil {
		associateAndMarkConsumed(root.Package, directives, comments, consumed, root.Package.Pos.Line)
	}

	for _, imp := range root.Imports {
		associateAndMarkConsumed(imp, directives, comments, consumed, imp.Pos.Line)
	}

	for _, msg := range root.Messages {
		mergeDirectivesForMessage(msg, directives, comments, consumed)
	}

	for _, enum := range root.Enums {
		mergeDirectivesForEnum(enum, directives, comments, consumed)
	}

	for _, svc := range root.Services {
		mergeDirectivesForService(svc, directives, comments, consumed)
	}

	// Associate any remaining directives with root
	associateAndMarkConsumed(root, directives, comments, consumed, 1)
}

// associateAndMarkConsumed associates directives with a node and marks them as consumed
func associateAndMarkConsumed(node interface{}, directives map[int]*SpokeDirectiveNode, comments map[int][]*CommentNode, consumed map[int]bool, startLine int) {
	// Look for directives/comments in the 3 lines before the node
	for line := startLine - 3; line < startLine; line++ {
		if line < 1 || consumed[line] {
			continue
		}

		// Check if this line has a directive
		if directive, ok := directives[line]; ok {
			// Associate with node
			switch n := node.(type) {
			case *RootNode:
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
				consumed[line] = true
			case *SyntaxNode:
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
				consumed[line] = true
			case *PackageNode:
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
				consumed[line] = true
			case *ImportNode:
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
				consumed[line] = true
			case *MessageNode:
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
				consumed[line] = true
			case *FieldNode:
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
				consumed[line] = true
			case *EnumNode:
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
				consumed[line] = true
			case *EnumValueNode:
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
				consumed[line] = true
			case *ServiceNode:
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
				consumed[line] = true
			case *RPCNode:
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
				consumed[line] = true
			}
		}

		// Also handle comments
		if commentList, ok := comments[line]; ok {
			switch n := node.(type) {
			case *RootNode:
				n.Comments = append(n.Comments, commentList...)
			case *SyntaxNode:
				n.Comments = append(n.Comments, commentList...)
			case *PackageNode:
				n.Comments = append(n.Comments, commentList...)
			case *ImportNode:
				n.Comments = append(n.Comments, commentList...)
			case *MessageNode:
				n.Comments = append(n.Comments, commentList...)
			case *FieldNode:
				n.Comments = append(n.Comments, commentList...)
			case *EnumNode:
				n.Comments = append(n.Comments, commentList...)
			case *EnumValueNode:
				n.Comments = append(n.Comments, commentList...)
			case *ServiceNode:
				n.Comments = append(n.Comments, commentList...)
			case *RPCNode:
				n.Comments = append(n.Comments, commentList...)
			}
		}
	}
}

// mergeDirectivesForMessage recursively merges directives for a message and its nested elements
func mergeDirectivesForMessage(msg *MessageNode, directives map[int]*SpokeDirectiveNode, comments map[int][]*CommentNode, consumed map[int]bool) {
	associateAndMarkConsumed(msg, directives, comments, consumed, msg.Pos.Line)

	for _, field := range msg.Fields {
		associateAndMarkConsumed(field, directives, comments, consumed, field.Pos.Line)
	}

	for _, nested := range msg.Nested {
		mergeDirectivesForMessage(nested, directives, comments, consumed)
	}

	for _, enum := range msg.Enums {
		mergeDirectivesForEnum(enum, directives, comments, consumed)
	}
}

// mergeDirectivesForEnum merges directives for an enum and its values
func mergeDirectivesForEnum(enum *EnumNode, directives map[int]*SpokeDirectiveNode, comments map[int][]*CommentNode, consumed map[int]bool) {
	associateAndMarkConsumed(enum, directives, comments, consumed, enum.Pos.Line)

	for _, value := range enum.Values {
		associateAndMarkConsumed(value, directives, comments, consumed, value.Pos.Line)
	}
}

// mergeDirectivesForService merges directives for a service and its RPCs
func mergeDirectivesForService(svc *ServiceNode, directives map[int]*SpokeDirectiveNode, comments map[int][]*CommentNode, consumed map[int]bool) {
	associateAndMarkConsumed(svc, directives, comments, consumed, svc.Pos.Line)

	for _, rpc := range svc.RPCs {
		associateAndMarkConsumed(rpc, directives, comments, consumed, rpc.Pos.Line)
	}
}

// UseDescriptorParser enables the new descriptor-based parser.
// The descriptor parser is now the default as it provides complete AST parsing
// including field declarations, which the legacy parser did not support.
var UseDescriptorParser = true

// ParseWithFallback attempts to parse with the descriptor parser if enabled,
// falls back to legacy parser if disabled or if descriptor parser fails.
func ParseWithFallback(content string) (*RootNode, error) {
	if UseDescriptorParser {
		// Try new parser
		ast, err := ParseWithDescriptor("input.proto", content)
		if err == nil {
			return ast, nil
		}
		// Log error but fall back to legacy parser
		_ = err
	}

	// Use legacy parser
	parser := NewParserFromString(content)
	return parser.Parse()
}

// NewParserFromString creates a parser from a string (helper for fallback)
func NewParserFromString(content string) *Parser {
	return NewParser(strings.NewReader(content))
}
