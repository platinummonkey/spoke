package compatibility

import (
	"testing"
)

func TestComparator_CompareMessages_FieldRemoved(t *testing.T) {
	oldSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
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
			},
		},
	}

	newSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
				Name:     "User",
				FullName: "test.User",
				Fields: map[int]*Field{
					1: {Name: "id", Number: 1, Type: FieldTypeString},
					// name field removed
				},
				FieldsByName: map[string]*Field{
					"id": {Name: "id", Number: 1, Type: FieldTypeString},
				},
			},
		},
	}

	comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
	result, err := comparator.Compare()
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.Compatible {
		t.Error("Expected incompatible result for removed field")
	}

	// Check for FIELD_REMOVED violation
	found := false
	for _, v := range result.Violations {
		if v.Rule == "FIELD_REMOVED" && v.Level == ViolationLevelError {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected FIELD_REMOVED violation with ERROR level")
	}
}

func TestComparator_CompareMessages_FieldAdded(t *testing.T) {
	oldSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
				Name:     "User",
				FullName: "test.User",
				Fields: map[int]*Field{
					1: {Name: "id", Number: 1, Type: FieldTypeString},
				},
				FieldsByName: map[string]*Field{
					"id": {Name: "id", Number: 1, Type: FieldTypeString},
				},
			},
		},
	}

	newSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
				Name:     "User",
				FullName: "test.User",
				Fields: map[int]*Field{
					1: {Name: "id", Number: 1, Type: FieldTypeString},
					2: {Name: "email", Number: 2, Type: FieldTypeString, Label: FieldLabelOptional},
				},
				FieldsByName: map[string]*Field{
					"id":    {Name: "id", Number: 1, Type: FieldTypeString},
					"email": {Name: "email", Number: 2, Type: FieldTypeString, Label: FieldLabelOptional},
				},
			},
		},
	}

	comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
	result, err := comparator.Compare()
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if !result.Compatible {
		t.Error("Expected compatible result for added optional field")
	}

	// Check for FIELD_ADDED info
	found := false
	for _, v := range result.Violations {
		if v.Rule == "FIELD_ADDED" && v.Level == ViolationLevelInfo {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected FIELD_ADDED violation with INFO level")
	}
}

func TestComparator_CompareMessages_RequiredFieldAdded(t *testing.T) {
	oldSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
				Name:     "User",
				FullName: "test.User",
				Fields: map[int]*Field{
					1: {Name: "id", Number: 1, Type: FieldTypeString},
				},
				FieldsByName: map[string]*Field{
					"id": {Name: "id", Number: 1, Type: FieldTypeString},
				},
			},
		},
	}

	newSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
				Name:     "User",
				FullName: "test.User",
				Fields: map[int]*Field{
					1: {Name: "id", Number: 1, Type: FieldTypeString},
					2: {Name: "email", Number: 2, Type: FieldTypeString, Label: FieldLabelRequired},
				},
				FieldsByName: map[string]*Field{
					"id":    {Name: "id", Number: 1, Type: FieldTypeString},
					"email": {Name: "email", Number: 2, Type: FieldTypeString, Label: FieldLabelRequired},
				},
			},
		},
	}

	comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
	result, err := comparator.Compare()
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.Compatible {
		t.Error("Expected incompatible result for added required field")
	}

	// Check for REQUIRED_FIELD_ADDED violation
	found := false
	for _, v := range result.Violations {
		if v.Rule == "REQUIRED_FIELD_ADDED" && v.Level == ViolationLevelError {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected REQUIRED_FIELD_ADDED violation with ERROR level")
	}
}

func TestComparator_CompareMessages_FieldTypeChanged_Incompatible(t *testing.T) {
	oldSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
				Name:     "User",
				FullName: "test.User",
				Fields: map[int]*Field{
					1: {Name: "age", Number: 1, Type: FieldTypeInt32},
				},
				FieldsByName: map[string]*Field{
					"age": {Name: "age", Number: 1, Type: FieldTypeInt32},
				},
			},
		},
	}

	newSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
				Name:     "User",
				FullName: "test.User",
				Fields: map[int]*Field{
					1: {Name: "age", Number: 1, Type: FieldTypeString},
				},
				FieldsByName: map[string]*Field{
					"age": {Name: "age", Number: 1, Type: FieldTypeString},
				},
			},
		},
	}

	comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
	result, err := comparator.Compare()
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.Compatible {
		t.Error("Expected incompatible result for incompatible type change")
	}

	// Check for FIELD_TYPE_CHANGED violation
	found := false
	for _, v := range result.Violations {
		if v.Rule == "FIELD_TYPE_CHANGED" && v.Level == ViolationLevelError && v.WireBreaking {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected FIELD_TYPE_CHANGED violation with ERROR level and WireBreaking flag")
	}
}

func TestComparator_CompareMessages_FieldTypeChanged_Compatible(t *testing.T) {
	oldSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
				Name:     "User",
				FullName: "test.User",
				Fields: map[int]*Field{
					1: {Name: "age", Number: 1, Type: FieldTypeInt32},
				},
				FieldsByName: map[string]*Field{
					"age": {Name: "age", Number: 1, Type: FieldTypeInt32},
				},
			},
		},
	}

	newSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
				Name:     "User",
				FullName: "test.User",
				Fields: map[int]*Field{
					1: {Name: "age", Number: 1, Type: FieldTypeInt64},
				},
				FieldsByName: map[string]*Field{
					"age": {Name: "age", Number: 1, Type: FieldTypeInt64},
				},
			},
		},
	}

	comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
	result, err := comparator.Compare()
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// int32 -> int64 is wire-compatible
	if !result.Compatible {
		t.Error("Expected compatible result for wire-compatible type change (int32 -> int64)")
	}
}

func TestComparator_CompareEnums_ValueRemoved(t *testing.T) {
	oldSchema := &SchemaGraph{
		Package: "test",
		Enums: map[string]*Enum{
			"test.Status": {
				Name:     "Status",
				FullName: "test.Status",
				Values: map[int]*EnumValue{
					0: {Name: "UNSPECIFIED", Number: 0},
					1: {Name: "ACTIVE", Number: 1},
					2: {Name: "INACTIVE", Number: 2},
				},
				ValuesByName: map[string]*EnumValue{
					"UNSPECIFIED": {Name: "UNSPECIFIED", Number: 0},
					"ACTIVE":      {Name: "ACTIVE", Number: 1},
					"INACTIVE":    {Name: "INACTIVE", Number: 2},
				},
			},
		},
	}

	newSchema := &SchemaGraph{
		Package: "test",
		Enums: map[string]*Enum{
			"test.Status": {
				Name:     "Status",
				FullName: "test.Status",
				Values: map[int]*EnumValue{
					0: {Name: "UNSPECIFIED", Number: 0},
					1: {Name: "ACTIVE", Number: 1},
					// INACTIVE removed
				},
				ValuesByName: map[string]*EnumValue{
					"UNSPECIFIED": {Name: "UNSPECIFIED", Number: 0},
					"ACTIVE":      {Name: "ACTIVE", Number: 1},
				},
			},
		},
	}

	comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
	result, err := comparator.Compare()
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.Compatible {
		t.Error("Expected incompatible result for removed enum value")
	}

	// Check for ENUM_VALUE_REMOVED violation
	found := false
	for _, v := range result.Violations {
		if v.Rule == "ENUM_VALUE_REMOVED" && v.Level == ViolationLevelError {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected ENUM_VALUE_REMOVED violation with ERROR level")
	}
}

func TestComparator_CompareEnums_ValueNumberChanged(t *testing.T) {
	oldSchema := &SchemaGraph{
		Package: "test",
		Enums: map[string]*Enum{
			"test.Status": {
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
			},
		},
	}

	newSchema := &SchemaGraph{
		Package: "test",
		Enums: map[string]*Enum{
			"test.Status": {
				Name:     "Status",
				FullName: "test.Status",
				Values: map[int]*EnumValue{
					0: {Name: "UNSPECIFIED", Number: 0},
					2: {Name: "ACTIVE", Number: 2}, // Number changed from 1 to 2
				},
				ValuesByName: map[string]*EnumValue{
					"UNSPECIFIED": {Name: "UNSPECIFIED", Number: 0},
					"ACTIVE":      {Name: "ACTIVE", Number: 2},
				},
			},
		},
	}

	comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
	result, err := comparator.Compare()
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.Compatible {
		t.Error("Expected incompatible result for enum value number change")
	}

	// Check for ENUM_VALUE_NUMBER_CHANGED violation
	found := false
	for _, v := range result.Violations {
		if v.Rule == "ENUM_VALUE_NUMBER_CHANGED" && v.Level == ViolationLevelError && v.WireBreaking {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected ENUM_VALUE_NUMBER_CHANGED violation with ERROR level and WireBreaking flag")
	}
}

func TestComparator_CompareServices_RPCRemoved(t *testing.T) {
	oldSchema := &SchemaGraph{
		Package: "test",
		Services: map[string]*Service{
			"test.UserService": {
				Name:     "UserService",
				FullName: "test.UserService",
				Methods: map[string]*Method{
					"GetUser":    {Name: "GetUser", InputType: "GetUserRequest", OutputType: "User"},
					"CreateUser": {Name: "CreateUser", InputType: "CreateUserRequest", OutputType: "User"},
				},
			},
		},
	}

	newSchema := &SchemaGraph{
		Package: "test",
		Services: map[string]*Service{
			"test.UserService": {
				Name:     "UserService",
				FullName: "test.UserService",
				Methods: map[string]*Method{
					"GetUser": {Name: "GetUser", InputType: "GetUserRequest", OutputType: "User"},
					// CreateUser removed
				},
			},
		},
	}

	comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
	result, err := comparator.Compare()
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.Compatible {
		t.Error("Expected incompatible result for removed RPC")
	}

	// Check for RPC_REMOVED violation
	found := false
	for _, v := range result.Violations {
		if v.Rule == "RPC_REMOVED" && v.Level == ViolationLevelError {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected RPC_REMOVED violation with ERROR level")
	}
}

func TestComparator_CompareServices_RPCInputTypeChanged(t *testing.T) {
	oldSchema := &SchemaGraph{
		Package: "test",
		Services: map[string]*Service{
			"test.UserService": {
				Name:     "UserService",
				FullName: "test.UserService",
				Methods: map[string]*Method{
					"GetUser": {Name: "GetUser", InputType: "GetUserRequest", OutputType: "User"},
				},
			},
		},
	}

	newSchema := &SchemaGraph{
		Package: "test",
		Services: map[string]*Service{
			"test.UserService": {
				Name:     "UserService",
				FullName: "test.UserService",
				Methods: map[string]*Method{
					"GetUser": {Name: "GetUser", InputType: "GetUserRequestV2", OutputType: "User"}, // Input type changed
				},
			},
		},
	}

	comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
	result, err := comparator.Compare()
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.Compatible {
		t.Error("Expected incompatible result for RPC input type change")
	}

	// Check for RPC_INPUT_TYPE_CHANGED violation
	found := false
	for _, v := range result.Violations {
		if v.Rule == "RPC_INPUT_TYPE_CHANGED" && v.Level == ViolationLevelError && v.WireBreaking {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected RPC_INPUT_TYPE_CHANGED violation with ERROR level and WireBreaking flag")
	}
}

func TestComparator_Summary(t *testing.T) {
	oldSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
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
			},
		},
	}

	newSchema := &SchemaGraph{
		Package: "test",
		Messages: map[string]*Message{
			"test.User": {
				Name:     "User",
				FullName: "test.User",
				Fields: map[int]*Field{
					1: {Name: "id", Number: 1, Type: FieldTypeString},
					// name field removed (ERROR)
					3: {Name: "email", Number: 3, Type: FieldTypeString, Label: FieldLabelOptional}, // added (INFO)
				},
				FieldsByName: map[string]*Field{
					"id":    {Name: "id", Number: 1, Type: FieldTypeString},
					"email": {Name: "email", Number: 3, Type: FieldTypeString, Label: FieldLabelOptional},
				},
			},
		},
	}

	comparator := NewComparator(CompatibilityModeBackward, oldSchema, newSchema)
	result, err := comparator.Compare()
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	summary := result.Summary

	if summary.TotalViolations != 2 {
		t.Errorf("TotalViolations = %d, want %d", summary.TotalViolations, 2)
	}
	if summary.Errors != 1 {
		t.Errorf("Errors = %d, want %d", summary.Errors, 1)
	}
	if summary.Infos != 1 {
		t.Errorf("Infos = %d, want %d", summary.Infos, 1)
	}
}

func TestCompatibilityMode_String(t *testing.T) {
	tests := []struct {
		mode CompatibilityMode
		want string
	}{
		{CompatibilityModeNone, "NONE"},
		{CompatibilityModeBackward, "BACKWARD"},
		{CompatibilityModeForward, "FORWARD"},
		{CompatibilityModeFull, "FULL"},
		{CompatibilityModeBackwardTransitive, "BACKWARD_TRANSITIVE"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.mode.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseCompatibilityMode(t *testing.T) {
	tests := []struct {
		input   string
		want    CompatibilityMode
		wantErr bool
	}{
		{"BACKWARD", CompatibilityModeBackward, false},
		{"FORWARD", CompatibilityModeForward, false},
		{"FULL", CompatibilityModeFull, false},
		{"NONE", CompatibilityModeNone, false},
		{"INVALID", CompatibilityModeNone, true},
		{"backward", CompatibilityModeNone, true}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCompatibilityMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCompatibilityMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseCompatibilityMode() = %v, want %v", got, tt.want)
			}
		})
	}
}
