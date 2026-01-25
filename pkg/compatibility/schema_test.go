package compatibility

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

func TestSchemaGraphBuilder_BuildFromAST(t *testing.T) {
	tests := []struct {
		name     string
		ast      *protobuf.RootNode
		wantPkg  string
		wantMsgs int
		wantEnums int
		wantSvcs int
	}{
		{
			name: "empty schema",
			ast: &protobuf.RootNode{
				Package: &protobuf.PackageNode{Name: "test.empty"},
				Syntax:  &protobuf.SyntaxNode{Value: "proto3"},
			},
			wantPkg:  "test.empty",
			wantMsgs: 0,
			wantEnums: 0,
			wantSvcs: 0,
		},
		{
			name: "simple message",
			ast: &protobuf.RootNode{
				Package: &protobuf.PackageNode{Name: "test.simple"},
				Syntax:  &protobuf.SyntaxNode{Value: "proto3"},
				Messages: []*protobuf.MessageNode{
					{
						Name: "User",
						Fields: []*protobuf.FieldNode{
							{Name: "id", Number: 1, Type: "string"},
							{Name: "name", Number: 2, Type: "string"},
							{Name: "age", Number: 3, Type: "int32"},
						},
					},
				},
			},
			wantPkg:  "test.simple",
			wantMsgs: 1,
			wantEnums: 0,
			wantSvcs: 0,
		},
		{
			name: "message with nested message",
			ast: &protobuf.RootNode{
				Package: &protobuf.PackageNode{Name: "test.nested"},
				Syntax:  &protobuf.SyntaxNode{Value: "proto3"},
				Messages: []*protobuf.MessageNode{
					{
						Name: "User",
						Fields: []*protobuf.FieldNode{
							{Name: "id", Number: 1, Type: "string"},
						},
						Nested: []*protobuf.MessageNode{
							{
								Name: "Address",
								Fields: []*protobuf.FieldNode{
									{Name: "street", Number: 1, Type: "string"},
								},
							},
						},
					},
				},
			},
			wantPkg:  "test.nested",
			wantMsgs: 1,
			wantEnums: 0,
			wantSvcs: 0,
		},
		{
			name: "enum",
			ast: &protobuf.RootNode{
				Package: &protobuf.PackageNode{Name: "test.enum"},
				Syntax:  &protobuf.SyntaxNode{Value: "proto3"},
				Enums: []*protobuf.EnumNode{
					{
						Name: "Status",
						Values: []*protobuf.EnumValueNode{
							{Name: "STATUS_UNSPECIFIED", Number: 0},
							{Name: "STATUS_ACTIVE", Number: 1},
						},
					},
				},
			},
			wantPkg:  "test.enum",
			wantMsgs: 0,
			wantEnums: 1,
			wantSvcs: 0,
		},
		{
			name: "service",
			ast: &protobuf.RootNode{
				Package: &protobuf.PackageNode{Name: "test.service"},
				Syntax:  &protobuf.SyntaxNode{Value: "proto3"},
				Services: []*protobuf.ServiceNode{
					{
						Name: "UserService",
						RPCs: []*protobuf.RPCNode{
							{
								Name:       "GetUser",
								InputType:  "GetUserRequest",
								OutputType: "User",
							},
						},
					},
				},
			},
			wantPkg:  "test.service",
			wantMsgs: 0,
			wantEnums: 0,
			wantSvcs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewSchemaGraphBuilder()
			graph, err := builder.BuildFromAST(tt.ast)
			if err != nil {
				t.Fatalf("BuildFromAST() error = %v", err)
			}

			if graph.Package != tt.wantPkg {
				t.Errorf("Package = %q, want %q", graph.Package, tt.wantPkg)
			}
			if len(graph.Messages) != tt.wantMsgs {
				t.Errorf("Messages count = %d, want %d", len(graph.Messages), tt.wantMsgs)
			}
			if len(graph.Enums) != tt.wantEnums {
				t.Errorf("Enums count = %d, want %d", len(graph.Enums), tt.wantEnums)
			}
			if len(graph.Services) != tt.wantSvcs {
				t.Errorf("Services count = %d, want %d", len(graph.Services), tt.wantSvcs)
			}
		})
	}
}

func TestSchemaGraphBuilder_ParseFieldType(t *testing.T) {
	builder := NewSchemaGraphBuilder()

	tests := []struct {
		typeName string
		want     FieldType
	}{
		{"string", FieldTypeString},
		{"int32", FieldTypeInt32},
		{"int64", FieldTypeInt64},
		{"uint32", FieldTypeUint32},
		{"uint64", FieldTypeUint64},
		{"bool", FieldTypeBool},
		{"bytes", FieldTypeBytes},
		{"double", FieldTypeDouble},
		{"float", FieldTypeFloat},
		{"map<string,int32>", FieldTypeMap},
		{"CustomMessage", FieldTypeMessage},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			got := builder.parseFieldType(tt.typeName)
			if got != tt.want {
				t.Errorf("parseFieldType(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestMessage_FieldLookup(t *testing.T) {
	msg := &Message{
		Name:     "User",
		FullName: "test.User",
		Fields: map[int]*Field{
			1: {Name: "id", Number: 1, Type: FieldTypeString},
			2: {Name: "name", Number: 2, Type: FieldTypeString},
		},
		FieldsByName: map[string]*Field{
			"id":   {Name: "id", Number: 1, Type: FieldTypeString},
			"name": {Name: "name", Number: 2, Type: FieldTypeString},
		},
	}

	// Test lookup by number
	field := msg.Fields[1]
	if field.Name != "id" {
		t.Errorf("Fields[1].Name = %q, want %q", field.Name, "id")
	}

	// Test lookup by name
	field = msg.FieldsByName["name"]
	if field.Number != 2 {
		t.Errorf("FieldsByName[name].Number = %d, want %d", field.Number, 2)
	}
}

func TestEnum_ValueLookup(t *testing.T) {
	enum := &Enum{
		Name:     "Status",
		FullName: "test.Status",
		Values: map[int]*EnumValue{
			0: {Name: "UNSPECIFIED", Number: 0},
			1: {Name: "ACTIVE", Number: 1},
		},
		ValuesByName: map[string]*EnumValue{
			"UNSPECIFIED": {Name: "UNSPECIFIED", Number: 0},
			"ACTIVE":      {Name: "ACTIVE", Number: 1},
		},
	}

	// Test lookup by number
	val := enum.Values[1]
	if val.Name != "ACTIVE" {
		t.Errorf("Values[1].Name = %q, want %q", val.Name, "ACTIVE")
	}

	// Test lookup by name
	val = enum.ValuesByName["ACTIVE"]
	if val.Number != 1 {
		t.Errorf("ValuesByName[ACTIVE].Number = %d, want %d", val.Number, 1)
	}
}
