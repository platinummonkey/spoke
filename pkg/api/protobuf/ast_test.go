package protobuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	endPos := Position{Line: 10, Column: 1, Offset: 100}

	node := &RootNode{
		Pos:    pos,
		EndPos: endPos,
	}

	assert.Equal(t, NodeTypeUnknown, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}

func TestSyntaxNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	endPos := Position{Line: 1, Column: 20, Offset: 20}

	node := &SyntaxNode{
		Value:  "proto3",
		Pos:    pos,
		EndPos: endPos,
	}

	assert.Equal(t, NodeTypeSyntax, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}

func TestPackageNode(t *testing.T) {
	pos := Position{Line: 2, Column: 1, Offset: 21}
	endPos := Position{Line: 2, Column: 15, Offset: 35}

	node := &PackageNode{
		Name:   "test.package",
		Pos:    pos,
		EndPos: endPos,
	}

	assert.Equal(t, NodeTypePackage, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}

func TestImportNode(t *testing.T) {
	pos := Position{Line: 3, Column: 1, Offset: 36}
	endPos := Position{Line: 3, Column: 25, Offset: 60}

	node := &ImportNode{
		Path:   "test.proto",
		Public: true,
		Pos:    pos,
		EndPos: endPos,
	}

	assert.Equal(t, NodeTypeImport, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}

func TestOptionNode(t *testing.T) {
	pos := Position{Line: 4, Column: 1, Offset: 61}
	endPos := Position{Line: 4, Column: 30, Offset: 90}

	node := &OptionNode{
		Name:   "java_package",
		Value:  "com.test",
		Pos:    pos,
		EndPos: endPos,
	}

	assert.Equal(t, NodeTypeOption, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}

func TestFieldNode(t *testing.T) {
	pos := Position{Line: 5, Column: 3, Offset: 93}
	endPos := Position{Line: 5, Column: 25, Offset: 115}

	node := &FieldNode{
		Name:     "test_field",
		Type:     "string",
		Number:   1,
		Repeated: false,
		Pos:      pos,
		EndPos:   endPos,
	}

	assert.Equal(t, NodeTypeField, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}

func TestMessageNode(t *testing.T) {
	pos := Position{Line: 6, Column: 1, Offset: 116}
	endPos := Position{Line: 10, Column: 1, Offset: 200}

	node := &MessageNode{
		Name:   "TestMessage",
		Fields: []*FieldNode{},
		Pos:    pos,
		EndPos: endPos,
	}

	assert.Equal(t, NodeTypeMessage, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}

func TestEnumNode(t *testing.T) {
	pos := Position{Line: 11, Column: 1, Offset: 201}
	endPos := Position{Line: 15, Column: 1, Offset: 250}

	node := &EnumNode{
		Name:   "TestEnum",
		Values: []*EnumValueNode{},
		Pos:    pos,
		EndPos: endPos,
	}

	assert.Equal(t, NodeTypeEnum, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}

func TestEnumValueNode(t *testing.T) {
	pos := Position{Line: 12, Column: 3, Offset: 205}
	endPos := Position{Line: 12, Column: 20, Offset: 222}

	node := &EnumValueNode{
		Name:   "VALUE_UNKNOWN",
		Number: 0,
		Pos:    pos,
		EndPos: endPos,
	}

	assert.Equal(t, NodeTypeEnumValue, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}

func TestServiceNode(t *testing.T) {
	pos := Position{Line: 16, Column: 1, Offset: 251}
	endPos := Position{Line: 20, Column: 1, Offset: 300}

	node := &ServiceNode{
		Name: "TestService",
		RPCs: []*RPCNode{},
		Pos:  pos,
		EndPos: endPos,
	}

	assert.Equal(t, NodeTypeService, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}

func TestRPCNode(t *testing.T) {
	pos := Position{Line: 17, Column: 3, Offset: 255}
	endPos := Position{Line: 17, Column: 50, Offset: 302}

	node := &RPCNode{
		Name:       "TestRPC",
		InputType:  "TestRequest",
		OutputType: "TestResponse",
		Pos:        pos,
		EndPos:     endPos,
	}

	assert.Equal(t, NodeTypeRPC, node.NodeType())
	assert.Equal(t, pos, node.Position())
	assert.Equal(t, endPos, node.End())
}
