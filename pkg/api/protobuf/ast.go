package protobuf

// NodeType represents the type of AST node
type NodeType int

const (
	NodeTypeUnknown NodeType = iota
	NodeTypeSyntax
	NodeTypePackage
	NodeTypeImport
	NodeTypeOption
	NodeTypeMessage
	NodeTypeEnum
	NodeTypeService
	NodeTypeField
	NodeTypeEnumValue
	NodeTypeRPC
	NodeTypeComment
	NodeTypeOneOf
	NodeTypeExtend
	NodeTypeSpoke
)

// Position represents the position in the source code
type Position struct {
	Line   int
	Column int
	Offset int
}

// Node represents a node in the protobuf AST
type Node interface {
	NodeType() NodeType
	Position() Position
	End() Position
}

// RootNode represents the root of the protobuf AST
type RootNode struct {
	Syntax          *SyntaxNode
	Package         *PackageNode
	Imports         []*ImportNode
	Options         []*OptionNode
	Messages        []*MessageNode
	Enums           []*EnumNode
	Services        []*ServiceNode
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *RootNode) NodeType() NodeType {
	return NodeTypeUnknown
}

// Position returns the start position
func (n *RootNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *RootNode) End() Position {
	return n.EndPos
}

// SyntaxNode represents a syntax statement in protobuf
type SyntaxNode struct {
	Value           string // proto2 or proto3
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *SyntaxNode) NodeType() NodeType {
	return NodeTypeSyntax
}

// Position returns the start position
func (n *SyntaxNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *SyntaxNode) End() Position {
	return n.EndPos
}

// PackageNode represents a package statement in protobuf
type PackageNode struct {
	Name            string
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *PackageNode) NodeType() NodeType {
	return NodeTypePackage
}

// Position returns the start position
func (n *PackageNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *PackageNode) End() Position {
	return n.EndPos
}

// ImportNode represents an import statement in protobuf
type ImportNode struct {
	Path            string
	Public          bool
	Weak            bool
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *ImportNode) NodeType() NodeType {
	return NodeTypeImport
}

// Position returns the start position
func (n *ImportNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *ImportNode) End() Position {
	return n.EndPos
}

// OptionNode represents an option statement in protobuf
type OptionNode struct {
	Name            string
	Value           string
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *OptionNode) NodeType() NodeType {
	return NodeTypeOption
}

// Position returns the start position
func (n *OptionNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *OptionNode) End() Position {
	return n.EndPos
}

// FieldNode represents a field in a message or service
type FieldNode struct {
	Name            string
	Type            string
	Number          int
	Repeated        bool
	Optional        bool
	Required        bool
	Options         []*OptionNode
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *FieldNode) NodeType() NodeType {
	return NodeTypeField
}

// Position returns the start position
func (n *FieldNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *FieldNode) End() Position {
	return n.EndPos
}

// MessageNode represents a message definition in protobuf
type MessageNode struct {
	Name            string
	Fields          []*FieldNode
	Nested          []*MessageNode
	Enums           []*EnumNode
	OneOfs          []*OneOfNode
	Options         []*OptionNode
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *MessageNode) NodeType() NodeType {
	return NodeTypeMessage
}

// Position returns the start position
func (n *MessageNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *MessageNode) End() Position {
	return n.EndPos
}

// EnumValueNode represents an enum value
type EnumValueNode struct {
	Name            string
	Number          int
	Options         []*OptionNode
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *EnumValueNode) NodeType() NodeType {
	return NodeTypeEnumValue
}

// Position returns the start position
func (n *EnumValueNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *EnumValueNode) End() Position {
	return n.EndPos
}

// EnumNode represents an enum definition in protobuf
type EnumNode struct {
	Name            string
	Values          []*EnumValueNode
	Options         []*OptionNode
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *EnumNode) NodeType() NodeType {
	return NodeTypeEnum
}

// Position returns the start position
func (n *EnumNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *EnumNode) End() Position {
	return n.EndPos
}

// RPCNode represents an RPC method in a service
type RPCNode struct {
	Name            string
	InputType       string
	OutputType      string
	ClientStreaming bool
	ServerStreaming bool
	Options         []*OptionNode
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *RPCNode) NodeType() NodeType {
	return NodeTypeRPC
}

// Position returns the start position
func (n *RPCNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *RPCNode) End() Position {
	return n.EndPos
}

// ServiceNode represents a service definition in protobuf
type ServiceNode struct {
	Name            string
	RPCs            []*RPCNode
	Options         []*OptionNode
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *ServiceNode) NodeType() NodeType {
	return NodeTypeService
}

// Position returns the start position
func (n *ServiceNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *ServiceNode) End() Position {
	return n.EndPos
}

// CommentNode represents a comment in protobuf
type CommentNode struct {
	Text      string
	Leading   bool // Leading comment (before a statement)
	Trailing  bool // Trailing comment (after a statement on the same line)
	Pos       Position
	EndPos    Position
}

// NodeType returns the node type
func (n *CommentNode) NodeType() NodeType {
	return NodeTypeComment
}

// Position returns the start position
func (n *CommentNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *CommentNode) End() Position {
	return n.EndPos
}

// SpokeDirectiveNode represents a @spoke directive in protobuf comments
type SpokeDirectiveNode struct {
	Option    string // The option type (e.g., "domain", "option")
	Value     string // The value after the second colon
	Comments  []*CommentNode
	Pos       Position
	EndPos    Position
}

// NodeType returns the node type
func (n *SpokeDirectiveNode) NodeType() NodeType {
	return NodeTypeSpoke
}

// Position returns the start position
func (n *SpokeDirectiveNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *SpokeDirectiveNode) End() Position {
	return n.EndPos
}

// OneOfNode represents a oneof field in a message
type OneOfNode struct {
	Name            string
	Fields          []*FieldNode
	Comments        []*CommentNode
	SpokeDirectives []*SpokeDirectiveNode
	Pos             Position
	EndPos          Position
}

// NodeType returns the node type
func (n *OneOfNode) NodeType() NodeType {
	return NodeTypeOneOf
}

// Position returns the start position
func (n *OneOfNode) Position() Position {
	return n.Pos
}

// End returns the end position
func (n *OneOfNode) End() Position {
	return n.EndPos
}

