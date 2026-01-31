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

// getDummyProtoContent returns appropriate proto content for an import path
// For well-known Google protobuf types, returns proper definitions
// For other imports, returns minimal valid proto
func getDummyProtoContent(importPath string) string {
	// Handle well-known Google protobuf types
	switch importPath {
	case "google/protobuf/timestamp.proto":
		return `syntax = "proto3";
package google.protobuf;

message Timestamp {
  int64 seconds = 1;
  int32 nanos = 2;
}
`
	case "google/protobuf/duration.proto":
		return `syntax = "proto3";
package google.protobuf;

message Duration {
  int64 seconds = 1;
  int32 nanos = 2;
}
`
	case "google/protobuf/any.proto":
		return `syntax = "proto3";
package google.protobuf;

message Any {
  string type_url = 1;
  bytes value = 2;
}
`
	case "google/protobuf/struct.proto":
		return `syntax = "proto3";
package google.protobuf;

message Struct {
  map<string, Value> fields = 1;
}

message Value {
  oneof kind {
    double number_value = 1;
    string string_value = 2;
    bool bool_value = 3;
    Struct struct_value = 4;
    ListValue list_value = 5;
  }
}

message ListValue {
  repeated Value values = 1;
}
`
	case "google/protobuf/empty.proto":
		return `syntax = "proto3";
package google.protobuf;

message Empty {}
`
	case "google/protobuf/wrappers.proto":
		return `syntax = "proto3";
package google.protobuf;

message DoubleValue { double value = 1; }
message FloatValue { float value = 1; }
message Int64Value { int64 value = 1; }
message UInt64Value { uint64 value = 1; }
message Int32Value { int32 value = 1; }
message UInt32Value { uint32 value = 1; }
message BoolValue { bool value = 1; }
message StringValue { string value = 1; }
message BytesValue { bytes value = 1; }
`
	default:
		// For unknown imports, try to create a reasonable proto file based on the import path
		// Extract package name and potential message names from the path
		return generateDummyProtoFromPath(importPath)
	}
}

// generateDummyProtoFromPath generates a dummy proto file with inferred package and message names
func generateDummyProtoFromPath(importPath string) string {
	// Remove .proto extension
	path := strings.TrimSuffix(importPath, ".proto")

	// Extract package name from path (e.g., "user/user.proto" -> "user")
	parts := strings.Split(path, "/")
	var packageName string
	var messageName string

	if len(parts) > 0 {
		// Use the last directory as package name
		if len(parts) > 1 {
			packageName = parts[len(parts)-2]
		} else {
			packageName = parts[0]
		}

		// Use the file name (last part) as a message name, capitalized
		fileName := parts[len(parts)-1]
		if fileName != "" {
			// Capitalize first letter for message name
			messageName = strings.ToUpper(fileName[:1]) + fileName[1:]
		}
	}

	if packageName == "" {
		packageName = "dummy"
	}
	if messageName == "" {
		messageName = "DummyMessage"
	}

	// Generate a proto file with the inferred package and a dummy message
	return fmt.Sprintf(`syntax = "proto3";
package %s;

// Auto-generated dummy message for import resolution
message %s {
  string id = 1;
}
`, packageName, messageName)
}

// extractImportPaths extracts import paths from proto content using simple text parsing
// This is used to provide dummy files for imports before full parsing
func extractImportPaths(content string) []string {
	var imports []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Match import statements: import "path" or import public "path" or import weak "path"
		if strings.HasPrefix(line, "import ") {
			// Find the quoted string
			start := strings.Index(line, "\"")
			if start == -1 {
				continue
			}
			end := strings.Index(line[start+1:], "\"")
			if end == -1 {
				continue
			}

			importPath := line[start+1 : start+1+end]
			imports = append(imports, importPath)
		}
	}

	return imports
}

// extractImportModifiers extracts public and weak import paths from proto content
// Returns two slices: publicImports and weakImports
func extractImportModifiers(content string) (publicImports []string, weakImports []string) {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Match import statements
		if strings.HasPrefix(line, "import ") {
			// Find the quoted string
			start := strings.Index(line, "\"")
			if start == -1 {
				continue
			}
			end := strings.Index(line[start+1:], "\"")
			if end == -1 {
				continue
			}

			importPath := line[start+1 : start+1+end]

			// Check if import is public or weak
			beforeQuote := line[:start]
			if strings.Contains(beforeQuote, "public") {
				publicImports = append(publicImports, importPath)
			} else if strings.Contains(beforeQuote, "weak") {
				weakImports = append(weakImports, importPath)
			}
		}
	}

	return publicImports, weakImports
}

// parseToDescriptor uses protocompile to parse proto content into a FileDescriptorProto
func parseToDescriptor(filename, content string) (*descriptorpb.FileDescriptorProto, *descriptorpb.SourceCodeInfo, string, error) {
	// Extract imports from content to provide dummy files for unresolvable imports
	imports := extractImportPaths(content)

	// Build a map with the main file and dummy files for all imports
	fileMap := map[string]string{
		filename: content,
	}

	// Add dummy proto files for each import so protocompile doesn't fail on unresolvable imports
	for _, imp := range imports {
		if imp != "" && imp != filename {
			// Create a proto file for the import
			// For well-known Google protobuf types, use proper definitions
			fileMap[imp] = getDummyProtoContent(imp)
		}
	}

	// Create a protocompile parser
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(fileMap),
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

	// Extract public/weak import information from content
	publicImports, weakImports := extractImportModifiers(content)

	// Map import paths to indices
	importMap := make(map[string]int)
	for i, dep := range fileDescProto.Dependency {
		importMap[dep] = i
	}

	// Set public dependency indices
	for _, pubPath := range publicImports {
		if idx, ok := importMap[pubPath]; ok {
			fileDescProto.PublicDependency = append(fileDescProto.PublicDependency, int32(idx))
		}
	}

	// Set weak dependency indices
	for _, weakPath := range weakImports {
		if idx, ok := importMap[weakPath]; ok {
			fileDescProto.WeakDependency = append(fileDescProto.WeakDependency, int32(idx))
		}
	}

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
	var publicDeps []int32
	var weakDeps []int32

	for i := 0; i < fd.Imports().Len(); i++ {
		imp := fd.Imports().Get(i)
		deps[i] = string(imp.Path())

		// Check if import is public or weak based on the descriptor
		// Note: protoreflect doesn't expose IsPublic/IsWeak methods directly
		// We'll extract this from source content in parseToDescriptor
	}
	if len(deps) > 0 {
		fileProto.Dependency = deps
	}
	if len(publicDeps) > 0 {
		fileProto.PublicDependency = publicDeps
	}
	if len(weakDeps) > 0 {
		fileProto.WeakDependency = weakDeps
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

	// Add file options
	if fd.Options() != nil {
		// Convert protoreflect options to FileOptions proto
		fileProto.Options = fd.Options().(*descriptorpb.FileOptions)
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

	// Add oneofs
	oneofs := make([]*descriptorpb.OneofDescriptorProto, md.Oneofs().Len())
	for i := 0; i < md.Oneofs().Len(); i++ {
		od := md.Oneofs().Get(i)
		name := string(od.Name())
		oneofs[i] = &descriptorpb.OneofDescriptorProto{
			Name: &name,
		}
	}
	if len(oneofs) > 0 {
		msgProto.OneofDecl = oneofs
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
	if kind == protoreflect.MessageKind {
		if msg := fd.Message(); msg != nil {
			typeName := "." + string(msg.FullName())
			fieldProto.TypeName = &typeName
		}
	} else if kind == protoreflect.EnumKind {
		if enum := fd.Enum(); enum != nil {
			typeName := "." + string(enum.FullName())
			fieldProto.TypeName = &typeName
		}
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

	// Set oneof index if field belongs to a oneof
	if od := fd.ContainingOneof(); od != nil {
		index := int32(od.Index())
		fieldProto.OneofIndex = &index
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
	oneofs      map[string]int // oneof name -> line
}

// extractPositionsFromContent scans the proto content to find line numbers for each element
func extractPositionsFromContent(content string) *positionMap {
	pm := &positionMap{
		imports:  make(map[string]int),
		messages: make(map[string]int),
		enums:    make(map[string]int),
		services: make(map[string]int),
		fields:   make(map[string]int),
		oneofs:   make(map[string]int),
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

		// Find oneofs
		if strings.HasPrefix(trimmed, "oneof") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				pm.oneofs[parts[1]] = lineNum
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
	// Mark public imports
	for _, publicDepIdx := range desc.GetPublicDependency() {
		if int(publicDepIdx) < len(root.Imports) {
			root.Imports[publicDepIdx].Public = true
		}
	}
	// Mark weak imports
	for _, weakDepIdx := range desc.GetWeakDependency() {
		if int(weakDepIdx) < len(root.Imports) {
			root.Imports[weakDepIdx].Weak = true
		}
	}
	for i, weakDep := range desc.GetWeakDependency() {
		if int(weakDep) < len(root.Imports) {
			root.Imports[weakDep].Weak = true
		}
		_ = i // unused
	}

	// Convert file options
	if desc.Options != nil {
		root.Options = convertFileOptions(desc.Options)
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

// convertFileOptions converts FileOptions to OptionNodes
func convertFileOptions(opts *descriptorpb.FileOptions) []*OptionNode {
	if opts == nil {
		return nil
	}

	options := make([]*OptionNode, 0)

	// Extract commonly used file options
	if opts.GoPackage != nil {
		options = append(options, &OptionNode{
			Name:            "go_package",
			Value:           opts.GetGoPackage(),
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	if opts.JavaPackage != nil {
		options = append(options, &OptionNode{
			Name:            "java_package",
			Value:           opts.GetJavaPackage(),
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	if opts.JavaOuterClassname != nil {
		options = append(options, &OptionNode{
			Name:            "java_outer_classname",
			Value:           opts.GetJavaOuterClassname(),
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	if opts.JavaMultipleFiles != nil {
		value := "false"
		if opts.GetJavaMultipleFiles() {
			value = "true"
		}
		options = append(options, &OptionNode{
			Name:            "java_multiple_files",
			Value:           value,
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	if opts.OptimizeFor != nil {
		value := opts.GetOptimizeFor().String()
		options = append(options, &OptionNode{
			Name:            "optimize_for",
			Value:           value,
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	if opts.CcEnableArenas != nil {
		value := "false"
		if opts.GetCcEnableArenas() {
			value = "true"
		}
		options = append(options, &OptionNode{
			Name:            "cc_enable_arenas",
			Value:           value,
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	if opts.ObjcClassPrefix != nil {
		options = append(options, &OptionNode{
			Name:            "objc_class_prefix",
			Value:           opts.GetObjcClassPrefix(),
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	if opts.CsharpNamespace != nil {
		options = append(options, &OptionNode{
			Name:            "csharp_namespace",
			Value:           opts.GetCsharpNamespace(),
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	if opts.Deprecated != nil {
		value := "false"
		if opts.GetDeprecated() {
			value = "true"
		}
		options = append(options, &OptionNode{
			Name:            "deprecated",
			Value:           value,
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		})
	}

	return options
}

// convertMessage converts a DescriptorProto (message descriptor) to MessageNode
func convertMessage(desc *descriptorpb.DescriptorProto, positions *positionMap) *MessageNode {
	msg := &MessageNode{
		Name:            desc.GetName(),
		Fields:          make([]*FieldNode, 0),
		Nested:          make([]*MessageNode, 0),
		Enums:           make([]*EnumNode, 0),
		OneOfs:          make([]*OneOfNode, 0),
		Options:         make([]*OptionNode, 0),
		Comments:        make([]*CommentNode, 0),
		SpokeDirectives: make([]*SpokeDirectiveNode, 0),
		Pos:             Position{Line: positions.messages[desc.GetName()]},
	}

	// Convert fields first (we'll organize them into oneofs later)
	allFields := make([]*FieldNode, 0)
	for _, fieldDesc := range desc.GetField() {
		allFields = append(allFields, convertField(fieldDesc, positions))
	}

	// Convert oneofs
	for oneofIndex, oneofDesc := range desc.GetOneofDecl() {
		oneof := &OneOfNode{
			Name:            oneofDesc.GetName(),
			Fields:          make([]*FieldNode, 0),
			Comments:        make([]*CommentNode, 0),
			SpokeDirectives: make([]*SpokeDirectiveNode, 0),
			Pos:             Position{Line: positions.oneofs[oneofDesc.GetName()]},
		}

		// Find all fields that belong to this oneof
		for i, fieldDesc := range desc.GetField() {
			if fieldDesc.OneofIndex != nil && int(*fieldDesc.OneofIndex) == oneofIndex {
				oneof.Fields = append(oneof.Fields, allFields[i])
			}
		}

		msg.OneOfs = append(msg.OneOfs, oneof)
	}

	// Add all fields to message (including oneof fields)
	msg.Fields = allFields

	// Convert nested messages
	for _, nestedDesc := range desc.GetNestedType() {
		msg.Nested = append(msg.Nested, convertMessage(nestedDesc, positions))
	}

	// Convert nested enums
	for _, enumDesc := range desc.GetEnumType() {
		msg.Enums = append(msg.Enums, convertEnum(enumDesc, positions))
	}

	// TODO: Convert extensions, reserved fields (low priority - rarely used)

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
			case *OneOfNode:
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
			case *OneOfNode:
				n.Comments = append(n.Comments, commentList...)
			}
		}
	}
}

// mergeDirectivesForMessage recursively merges directives for a message and its nested elements
func mergeDirectivesForMessage(msg *MessageNode, directives map[int]*SpokeDirectiveNode, comments map[int][]*CommentNode, consumed map[int]bool) {
	associateAndMarkConsumed(msg, directives, comments, consumed, msg.Pos.Line)

	// Process oneofs first (before fields) so oneof-level directives are consumed first
	for _, oneof := range msg.OneOfs {
		associateAndMarkConsumed(oneof, directives, comments, consumed, oneof.Pos.Line)
		// Also associate directives with oneof fields
		for _, field := range oneof.Fields {
			associateAndMarkConsumed(field, directives, comments, consumed, field.Pos.Line)
		}
	}

	// Process fields after oneofs
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

// ParseWithFallback parses proto content using the descriptor parser.
// This function is kept for backward compatibility but now directly uses
// the descriptor parser without fallback to the deprecated legacy parser.
//
// Deprecated: Use ParseWithDescriptor directly for clarity.
func ParseWithFallback(content string) (*RootNode, error) {
	return ParseWithDescriptor("input.proto", content)
}
