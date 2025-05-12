package protobuf

import (
	"errors"
	"io"
	"strings"
)

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
	Syntax   *SyntaxNode
	Package  *PackageNode
	Imports  []*ImportNode
	Options  []*OptionNode
	Messages []*MessageNode
	Enums    []*EnumNode
	Services []*ServiceNode
	Comments []*CommentNode
	Pos      Position
	EndPos   Position
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
	Value     string // proto2 or proto3
	Comments  []*CommentNode
	Pos       Position
	EndPos    Position
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
	Name      string
	Comments  []*CommentNode
	Pos       Position
	EndPos    Position
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
	Path      string
	Public    bool
	Weak      bool
	Comments  []*CommentNode
	Pos       Position
	EndPos    Position
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
	Name      string
	Value     string
	Comments  []*CommentNode
	Pos       Position
	EndPos    Position
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
	Name       string
	Type       string
	Number     int
	Repeated   bool
	Optional   bool
	Required   bool
	Options    []*OptionNode
	Comments   []*CommentNode
	Pos        Position
	EndPos     Position
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
	Name       string
	Fields     []*FieldNode
	Nested     []*MessageNode
	Enums      []*EnumNode
	OneOfs     []*OneOfNode
	Options    []*OptionNode
	Comments   []*CommentNode
	Pos        Position
	EndPos     Position
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
	Name      string
	Number    int
	Options   []*OptionNode
	Comments  []*CommentNode
	Pos       Position
	EndPos    Position
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
	Name      string
	Values    []*EnumValueNode
	Options   []*OptionNode
	Comments  []*CommentNode
	Pos       Position
	EndPos    Position
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
	Name           string
	InputType      string
	OutputType     string
	ClientStreaming bool
	ServerStreaming bool
	Options        []*OptionNode
	Comments       []*CommentNode
	Pos            Position
	EndPos         Position
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
	Name      string
	RPCs      []*RPCNode
	Options   []*OptionNode
	Comments  []*CommentNode
	Pos       Position
	EndPos    Position
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

// OneOfNode represents a oneof field in a message
type OneOfNode struct {
	Name      string
	Fields    []*FieldNode
	Comments  []*CommentNode
	Pos       Position
	EndPos    Position
}

// Type returns the node type
func (n *OneOfNode) Type() NodeType {
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

// Parser represents a protobuf parser
type Parser struct {
	scanner *Scanner
	current Token
	next    Token
}

// NewParser creates a new Parser
func NewParser(r io.Reader) *Parser {
	scanner := NewScanner(r)
	return &Parser{
		scanner: scanner,
	}
}

// NewParser creates a new Parser
func NewStringParser(content string) *Parser {
	scanner := NewScanner(strings.NewReader(content))
	return &Parser{
		scanner: scanner,
	}
}

// Parse parses the protobuf file and returns the AST
func (p *Parser) Parse() (*RootNode, error) {
	// Initialize by reading the first two tokens
	p.advance()
	p.advance()

	root := &RootNode{
		Imports:  make([]*ImportNode, 0),
		Options:  make([]*OptionNode, 0),
		Messages: make([]*MessageNode, 0),
		Enums:    make([]*EnumNode, 0),
		Services: make([]*ServiceNode, 0),
		Comments: make([]*CommentNode, 0),
		Pos:      p.current.Pos,
	}

	// Parse top-level statements
	for p.current.Type != TokenEOF {
		if p.current.Type == TokenComment {
			comment := p.parseComment()
			root.Comments = append(root.Comments, comment)
			continue
		}

		switch p.current.Text {
		case "syntax":
			if root.Syntax != nil {
				return nil, errors.New("multiple syntax statements")
			}
			syntax, err := p.parseSyntax()
			if err != nil {
				return nil, err
			}
			root.Syntax = syntax
		case "package":
			if root.Package != nil {
				return nil, errors.New("multiple package statements")
			}
			pkg, err := p.parsePackage()
			if err != nil {
				return nil, err
			}
			root.Package = pkg
		case "import":
			imp, err := p.parseImport()
			if err != nil {
				return nil, err
			}
			root.Imports = append(root.Imports, imp)
		case "option":
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			root.Options = append(root.Options, opt)
		case "message":
			msg, err := p.parseMessage()
			if err != nil {
				return nil, err
			}
			root.Messages = append(root.Messages, msg)
		case "enum":
			enum, err := p.parseEnum()
			if err != nil {
				return nil, err
			}
			root.Enums = append(root.Enums, enum)
		case "service":
			service, err := p.parseService()
			if err != nil {
				return nil, err
			}
			root.Services = append(root.Services, service)
		default:
			return nil, errors.New("unexpected token: " + p.current.Text)
		}
	}

	root.EndPos = p.current.Pos
	return root, nil
}

// advance moves to the next token
func (p *Parser) advance() {
	p.current = p.next
	p.next, _ = p.scanner.Scan()
}

// expect checks if the current token is the expected type and advances
func (p *Parser) expect(tokenType TokenType, text string) error {
	if p.current.Type != tokenType {
		return errors.New("expected " + string(tokenType) + " but got " + string(p.current.Type))
	}
	if text != "" && p.current.Text != text {
		return errors.New("expected '" + text + "' but got '" + p.current.Text + "'")
	}
	p.advance()
	return nil
}

// parseComment parses a comment
func (p *Parser) parseComment() *CommentNode {
	comment := &CommentNode{
		Text:     strings.TrimPrefix(strings.TrimPrefix(p.current.Text, "//"), " "),
		Leading:  true, // We'll adjust this later if needed
		Trailing: false,
		Pos:      p.current.Pos,
		EndPos:   Position{p.current.Pos.Line, p.current.Pos.Column + len(p.current.Text), p.current.Pos.Offset + len(p.current.Text)},
	}
	p.advance()
	return comment
}

// parseSyntax parses a syntax statement
func (p *Parser) parseSyntax() (*SyntaxNode, error) {
	pos := p.current.Pos
	p.advance() // consume "syntax"
	
	if err := p.expect(TokenPunctuation, "="); err != nil {
		return nil, err
	}
	
	if p.current.Type != TokenString {
		return nil, errors.New("expected string but got " + string(p.current.Type))
	}
	
	value := strings.Trim(p.current.Text, "\"'")
	p.advance() // consume the string
	
	if err := p.expect(TokenPunctuation, ";"); err != nil {
		return nil, err
	}
	
	return &SyntaxNode{
		Value:  value,
		Pos:    pos,
		EndPos: p.current.Pos,
	}, nil
}

// parsePackage parses a package statement
func (p *Parser) parsePackage() (*PackageNode, error) {
	pos := p.current.Pos
	p.advance() // consume "package"
	
	if p.current.Type != TokenIdentifier {
		return nil, errors.New("expected identifier but got " + string(p.current.Type))
	}
	
	name := p.current.Text
	p.advance() // consume the package name
	
	if err := p.expect(TokenPunctuation, ";"); err != nil {
		return nil, err
	}
	
	return &PackageNode{
		Name:   name,
		Pos:    pos,
		EndPos: p.current.Pos,
	}, nil
}

// parseImport parses an import statement
func (p *Parser) parseImport() (*ImportNode, error) {
	pos := p.current.Pos
	p.advance() // consume "import"
	
	imp := &ImportNode{
		Public: false,
		Weak:   false,
		Pos:    pos,
	}
	
	// Check for "public" or "weak"
	if p.current.Type == TokenIdentifier {
		if p.current.Text == "public" {
			imp.Public = true
			p.advance()
		} else if p.current.Text == "weak" {
			imp.Weak = true
			p.advance()
		}
	}
	
	if p.current.Type != TokenString {
		return nil, errors.New("expected string but got " + string(p.current.Type))
	}
	
	imp.Path = strings.Trim(p.current.Text, "\"'")
	p.advance() // consume the string
	
	if err := p.expect(TokenPunctuation, ";"); err != nil {
		return nil, err
	}
	
	imp.EndPos = p.current.Pos
	return imp, nil
}

// parseOption parses an option statement
func (p *Parser) parseOption() (*OptionNode, error) {
	pos := p.current.Pos
	p.advance() // consume "option"
	
	if p.current.Type != TokenIdentifier {
		return nil, errors.New("expected identifier but got " + string(p.current.Type))
	}
	
	name := p.current.Text
	p.advance() // consume the option name
	
	if err := p.expect(TokenPunctuation, "="); err != nil {
		return nil, err
	}
	
	// Option value can be a string, identifier, or number
	if p.current.Type != TokenString && p.current.Type != TokenIdentifier && p.current.Type != TokenNumber {
		return nil, errors.New("expected option value but got " + string(p.current.Type))
	}
	
	value := p.current.Text
	p.advance() // consume the value
	
	if err := p.expect(TokenPunctuation, ";"); err != nil {
		return nil, err
	}
	
	return &OptionNode{
		Name:   name,
		Value:  value,
		Pos:    pos,
		EndPos: p.current.Pos,
	}, nil
}

// parseMessage parses a message definition
func (p *Parser) parseMessage() (*MessageNode, error) {
	// This is a simplified implementation, the full parser would need to handle nested messages,
	// fields, oneofs, etc.
	pos := p.current.Pos
	p.advance() // consume "message"
	
	if p.current.Type != TokenIdentifier {
		return nil, errors.New("expected identifier but got " + string(p.current.Type))
	}
	
	name := p.current.Text
	p.advance() // consume the message name
	
	if err := p.expect(TokenPunctuation, "{"); err != nil {
		return nil, err
	}
	
	// For now, just consume everything until the closing brace
	// This is a simplified version and would need to be expanded
	braceCount := 1
	for braceCount > 0 && p.current.Type != TokenEOF {
		if p.current.Type == TokenPunctuation {
			if p.current.Text == "{" {
				braceCount++
			} else if p.current.Text == "}" {
				braceCount--
			}
		}
		p.advance()
	}
	
	return &MessageNode{
		Name:   name,
		Pos:    pos,
		EndPos: p.current.Pos,
	}, nil
}

// parseEnum parses an enum definition
func (p *Parser) parseEnum() (*EnumNode, error) {
	// This is a simplified implementation
	pos := p.current.Pos
	p.advance() // consume "enum"
	
	if p.current.Type != TokenIdentifier {
		return nil, errors.New("expected identifier but got " + string(p.current.Type))
	}
	
	name := p.current.Text
	p.advance() // consume the enum name
	
	if err := p.expect(TokenPunctuation, "{"); err != nil {
		return nil, err
	}
	
	// For now, just consume everything until the closing brace
	braceCount := 1
	for braceCount > 0 && p.current.Type != TokenEOF {
		if p.current.Type == TokenPunctuation {
			if p.current.Text == "{" {
				braceCount++
			} else if p.current.Text == "}" {
				braceCount--
			}
		}
		p.advance()
	}
	
	return &EnumNode{
		Name:   name,
		Pos:    pos,
		EndPos: p.current.Pos,
	}, nil
}

// parseService parses a service definition
func (p *Parser) parseService() (*ServiceNode, error) {
	// This is a simplified implementation
	pos := p.current.Pos
	p.advance() // consume "service"
	
	if p.current.Type != TokenIdentifier {
		return nil, errors.New("expected identifier but got " + string(p.current.Type))
	}
	
	name := p.current.Text
	p.advance() // consume the service name
	
	if err := p.expect(TokenPunctuation, "{"); err != nil {
		return nil, err
	}
	
	// For now, just consume everything until the closing brace
	braceCount := 1
	for braceCount > 0 && p.current.Type != TokenEOF {
		if p.current.Type == TokenPunctuation {
			if p.current.Text == "{" {
				braceCount++
			} else if p.current.Text == "}" {
				braceCount--
			}
		}
		p.advance()
	}
	
	return &ServiceNode{
		Name:   name,
		Pos:    pos,
		EndPos: p.current.Pos,
	}, nil
} 