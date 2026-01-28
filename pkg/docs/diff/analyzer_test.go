package diff

import (
	"testing"

	"github.com/platinummonkey/spoke/pkg/api"
)

func TestNewAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer()
	if analyzer == nil {
		t.Fatal("NewAnalyzer() returned nil")
	}
}

func TestCompare(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name        string
		fromVersion *api.Version
		toVersion   *api.Version
		wantErr     bool
	}{
		{
			name: "empty versions",
			fromVersion: &api.Version{
				Version: "v1.0.0",
				Files:   []api.File{},
			},
			toVersion: &api.Version{
				Version: "v2.0.0",
				Files:   []api.File{},
			},
			wantErr: false,
		},
		{
			name: "versions with files",
			fromVersion: &api.Version{
				Version: "v1.0.0",
				Files: []api.File{
					{Path: "test.proto", Content: "syntax = \"proto3\";"},
				},
			},
			toVersion: &api.Version{
				Version: "v2.0.0",
				Files: []api.File{
					{Path: "test.proto", Content: "syntax = \"proto3\";"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.Compare(tt.fromVersion, tt.toVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result == nil {
				t.Fatal("Compare() returned nil result")
			}
			if result.FromVersion != tt.fromVersion.Version {
				t.Errorf("FromVersion = %v, want %v", result.FromVersion, tt.fromVersion.Version)
			}
			if result.ToVersion != tt.toVersion.Version {
				t.Errorf("ToVersion = %v, want %v", result.ToVersion, tt.toVersion.Version)
			}
			if result.Changes == nil {
				t.Error("Changes slice is nil, expected empty slice")
			}
		})
	}
}

func TestBuildMessageMap(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name  string
		files []api.File
	}{
		{
			name:  "empty files",
			files: []api.File{},
		},
		{
			name: "single file",
			files: []api.File{
				{Path: "test.proto", Content: "syntax = \"proto3\";"},
			},
		},
		{
			name: "multiple files",
			files: []api.File{
				{Path: "test1.proto", Content: "syntax = \"proto3\";"},
				{Path: "test2.proto", Content: "syntax = \"proto3\";"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.buildMessageMap(tt.files)
			if result == nil {
				t.Fatal("buildMessageMap() returned nil")
			}
			// Currently returns empty map as it's a placeholder
			if len(result) != 0 {
				t.Errorf("buildMessageMap() returned %d messages, expected 0 (placeholder)", len(result))
			}
		})
	}
}

func TestBuildEnumMap(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name  string
		files []api.File
	}{
		{
			name:  "empty files",
			files: []api.File{},
		},
		{
			name: "single file",
			files: []api.File{
				{Path: "test.proto", Content: "syntax = \"proto3\";"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.buildEnumMap(tt.files)
			if result == nil {
				t.Fatal("buildEnumMap() returned nil")
			}
			if len(result) != 0 {
				t.Errorf("buildEnumMap() returned %d enums, expected 0 (placeholder)", len(result))
			}
		})
	}
}

func TestBuildServiceMap(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name  string
		files []api.File
	}{
		{
			name:  "empty files",
			files: []api.File{},
		},
		{
			name: "single file",
			files: []api.File{
				{Path: "test.proto", Content: "syntax = \"proto3\";"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.buildServiceMap(tt.files)
			if result == nil {
				t.Fatal("buildServiceMap() returned nil")
			}
			if len(result) != 0 {
				t.Errorf("buildServiceMap() returned %d services, expected 0 (placeholder)", len(result))
			}
		})
	}
}

func TestCompareMessages(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name        string
		oldMessages map[string]*messageInfo
		newMessages map[string]*messageInfo
		wantChanges int
	}{
		{
			name:        "both empty",
			oldMessages: map[string]*messageInfo{},
			newMessages: map[string]*messageInfo{},
			wantChanges: 0,
		},
		{
			name: "message removed",
			oldMessages: map[string]*messageInfo{
				"User": {
					Name:   "User",
					Fields: map[string]*fieldInfo{},
					File:   "user.proto",
				},
			},
			newMessages: map[string]*messageInfo{},
			wantChanges: 1,
		},
		{
			name:        "message added",
			oldMessages: map[string]*messageInfo{},
			newMessages: map[string]*messageInfo{
				"User": {
					Name:   "User",
					Fields: map[string]*fieldInfo{},
					File:   "user.proto",
				},
			},
			wantChanges: 1,
		},
		{
			name: "message unchanged",
			oldMessages: map[string]*messageInfo{
				"User": {
					Name:   "User",
					Fields: map[string]*fieldInfo{},
					File:   "user.proto",
				},
			},
			newMessages: map[string]*messageInfo{
				"User": {
					Name:   "User",
					Fields: map[string]*fieldInfo{},
					File:   "user.proto",
				},
			},
			wantChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := analyzer.compareMessages(tt.oldMessages, tt.newMessages)
			if changes == nil {
				t.Fatal("compareMessages() returned nil")
			}
			if len(changes) != tt.wantChanges {
				t.Errorf("compareMessages() returned %d changes, want %d", len(changes), tt.wantChanges)
			}

			// Verify change properties for message removed
			if tt.name == "message removed" && len(changes) > 0 {
				change := changes[0]
				if change.Type != MessageRemoved {
					t.Errorf("Change type = %v, want %v", change.Type, MessageRemoved)
				}
				if change.Severity != Breaking {
					t.Errorf("Change severity = %v, want %v", change.Severity, Breaking)
				}
				if change.OldValue != "User" {
					t.Errorf("Change OldValue = %v, want User", change.OldValue)
				}
				if change.MigrationTip == "" {
					t.Error("MigrationTip should not be empty for breaking changes")
				}
			}

			// Verify change properties for message added
			if tt.name == "message added" && len(changes) > 0 {
				change := changes[0]
				if change.Type != MessageAdded {
					t.Errorf("Change type = %v, want %v", change.Type, MessageAdded)
				}
				if change.Severity != NonBreaking {
					t.Errorf("Change severity = %v, want %v", change.Severity, NonBreaking)
				}
				if change.NewValue != "User" {
					t.Errorf("Change NewValue = %v, want User", change.NewValue)
				}
			}
		})
	}
}

func TestCompareFields(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name        string
		oldMsg      *messageInfo
		newMsg      *messageInfo
		wantChanges int
		checkChange func(t *testing.T, changes []Change)
	}{
		{
			name: "field removed",
			oldMsg: &messageInfo{
				Name: "User",
				File: "user.proto",
				Fields: map[string]*fieldInfo{
					"name": {Name: "name", Type: "string", Number: 1},
				},
			},
			newMsg: &messageInfo{
				Name:   "User",
				File:   "user.proto",
				Fields: map[string]*fieldInfo{},
			},
			wantChanges: 1,
			checkChange: func(t *testing.T, changes []Change) {
				if changes[0].Type != FieldRemoved {
					t.Errorf("Type = %v, want %v", changes[0].Type, FieldRemoved)
				}
				if changes[0].Severity != Breaking {
					t.Errorf("Severity = %v, want %v", changes[0].Severity, Breaking)
				}
			},
		},
		{
			name: "field added",
			oldMsg: &messageInfo{
				Name:   "User",
				File:   "user.proto",
				Fields: map[string]*fieldInfo{},
			},
			newMsg: &messageInfo{
				Name: "User",
				File: "user.proto",
				Fields: map[string]*fieldInfo{
					"email": {Name: "email", Type: "string", Number: 2},
				},
			},
			wantChanges: 1,
			checkChange: func(t *testing.T, changes []Change) {
				if changes[0].Type != FieldAdded {
					t.Errorf("Type = %v, want %v", changes[0].Type, FieldAdded)
				}
				if changes[0].Severity != NonBreaking {
					t.Errorf("Severity = %v, want %v", changes[0].Severity, NonBreaking)
				}
			},
		},
		{
			name: "type changed",
			oldMsg: &messageInfo{
				Name: "User",
				File: "user.proto",
				Fields: map[string]*fieldInfo{
					"age": {Name: "age", Type: "int32", Number: 1},
				},
			},
			newMsg: &messageInfo{
				Name: "User",
				File: "user.proto",
				Fields: map[string]*fieldInfo{
					"age": {Name: "age", Type: "string", Number: 1},
				},
			},
			wantChanges: 1,
			checkChange: func(t *testing.T, changes []Change) {
				if changes[0].Type != TypeChanged {
					t.Errorf("Type = %v, want %v", changes[0].Type, TypeChanged)
				}
				if changes[0].OldValue != "int32" {
					t.Errorf("OldValue = %v, want int32", changes[0].OldValue)
				}
				if changes[0].NewValue != "string" {
					t.Errorf("NewValue = %v, want string", changes[0].NewValue)
				}
			},
		},
		{
			name: "field number changed",
			oldMsg: &messageInfo{
				Name: "User",
				File: "user.proto",
				Fields: map[string]*fieldInfo{
					"id": {Name: "id", Type: "int32", Number: 1},
				},
			},
			newMsg: &messageInfo{
				Name: "User",
				File: "user.proto",
				Fields: map[string]*fieldInfo{
					"id": {Name: "id", Type: "int32", Number: 2},
				},
			},
			wantChanges: 1,
			checkChange: func(t *testing.T, changes []Change) {
				if changes[0].Type != FieldNumberChanged {
					t.Errorf("Type = %v, want %v", changes[0].Type, FieldNumberChanged)
				}
				if changes[0].OldValue != "1" {
					t.Errorf("OldValue = %v, want 1", changes[0].OldValue)
				}
				if changes[0].NewValue != "2" {
					t.Errorf("NewValue = %v, want 2", changes[0].NewValue)
				}
			},
		},
		{
			name: "label changed",
			oldMsg: &messageInfo{
				Name: "User",
				File: "user.proto",
				Fields: map[string]*fieldInfo{
					"tags": {Name: "tags", Type: "string", Number: 1, Label: "optional"},
				},
			},
			newMsg: &messageInfo{
				Name: "User",
				File: "user.proto",
				Fields: map[string]*fieldInfo{
					"tags": {Name: "tags", Type: "string", Number: 1, Label: "repeated"},
				},
			},
			wantChanges: 1,
			checkChange: func(t *testing.T, changes []Change) {
				if changes[0].Type != LabelChanged {
					t.Errorf("Type = %v, want %v", changes[0].Type, LabelChanged)
				}
				if changes[0].OldValue != "optional" {
					t.Errorf("OldValue = %v, want optional", changes[0].OldValue)
				}
				if changes[0].NewValue != "repeated" {
					t.Errorf("NewValue = %v, want repeated", changes[0].NewValue)
				}
			},
		},
		{
			name: "no changes",
			oldMsg: &messageInfo{
				Name: "User",
				File: "user.proto",
				Fields: map[string]*fieldInfo{
					"name": {Name: "name", Type: "string", Number: 1, Label: "optional"},
				},
			},
			newMsg: &messageInfo{
				Name: "User",
				File: "user.proto",
				Fields: map[string]*fieldInfo{
					"name": {Name: "name", Type: "string", Number: 1, Label: "optional"},
				},
			},
			wantChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := analyzer.compareFields(tt.oldMsg, tt.newMsg)
			if changes == nil {
				t.Fatal("compareFields() returned nil")
			}
			if len(changes) != tt.wantChanges {
				t.Errorf("compareFields() returned %d changes, want %d", len(changes), tt.wantChanges)
			}
			if tt.checkChange != nil && len(changes) > 0 {
				tt.checkChange(t, changes)
			}
		})
	}
}

func TestCompareEnums(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name        string
		oldEnums    map[string]*enumInfo
		newEnums    map[string]*enumInfo
		wantChanges int
	}{
		{
			name:        "both empty",
			oldEnums:    map[string]*enumInfo{},
			newEnums:    map[string]*enumInfo{},
			wantChanges: 0,
		},
		{
			name: "enum removed",
			oldEnums: map[string]*enumInfo{
				"Status": {
					Name:   "Status",
					Values: map[string]int{},
					File:   "status.proto",
				},
			},
			newEnums:    map[string]*enumInfo{},
			wantChanges: 1,
		},
		{
			name:     "enum added",
			oldEnums: map[string]*enumInfo{},
			newEnums: map[string]*enumInfo{
				"Status": {
					Name:   "Status",
					Values: map[string]int{},
					File:   "status.proto",
				},
			},
			wantChanges: 1,
		},
		{
			name: "enum unchanged",
			oldEnums: map[string]*enumInfo{
				"Status": {
					Name:   "Status",
					Values: map[string]int{},
					File:   "status.proto",
				},
			},
			newEnums: map[string]*enumInfo{
				"Status": {
					Name:   "Status",
					Values: map[string]int{},
					File:   "status.proto",
				},
			},
			wantChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := analyzer.compareEnums(tt.oldEnums, tt.newEnums)
			if changes == nil {
				t.Fatal("compareEnums() returned nil")
			}
			if len(changes) != tt.wantChanges {
				t.Errorf("compareEnums() returned %d changes, want %d", len(changes), tt.wantChanges)
			}

			// Verify change properties for enum removed
			if tt.name == "enum removed" && len(changes) > 0 {
				change := changes[0]
				if change.Type != EnumRemoved {
					t.Errorf("Change type = %v, want %v", change.Type, EnumRemoved)
				}
				if change.Severity != Breaking {
					t.Errorf("Change severity = %v, want %v", change.Severity, Breaking)
				}
			}

			// Verify change properties for enum added
			if tt.name == "enum added" && len(changes) > 0 {
				change := changes[0]
				if change.Type != EnumAdded {
					t.Errorf("Change type = %v, want %v", change.Type, EnumAdded)
				}
				if change.Severity != NonBreaking {
					t.Errorf("Change severity = %v, want %v", change.Severity, NonBreaking)
				}
			}
		})
	}
}

func TestCompareEnumValues(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name        string
		oldEnum     *enumInfo
		newEnum     *enumInfo
		wantChanges int
	}{
		{
			name: "enum value removed",
			oldEnum: &enumInfo{
				Name:   "Status",
				File:   "status.proto",
				Values: map[string]int{"ACTIVE": 0, "INACTIVE": 1},
			},
			newEnum: &enumInfo{
				Name:   "Status",
				File:   "status.proto",
				Values: map[string]int{"ACTIVE": 0},
			},
			wantChanges: 1,
		},
		{
			name: "enum value added",
			oldEnum: &enumInfo{
				Name:   "Status",
				File:   "status.proto",
				Values: map[string]int{"ACTIVE": 0},
			},
			newEnum: &enumInfo{
				Name:   "Status",
				File:   "status.proto",
				Values: map[string]int{"ACTIVE": 0, "PENDING": 2},
			},
			wantChanges: 1,
		},
		{
			name: "no changes",
			oldEnum: &enumInfo{
				Name:   "Status",
				File:   "status.proto",
				Values: map[string]int{"ACTIVE": 0, "INACTIVE": 1},
			},
			newEnum: &enumInfo{
				Name:   "Status",
				File:   "status.proto",
				Values: map[string]int{"ACTIVE": 0, "INACTIVE": 1},
			},
			wantChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := analyzer.compareEnumValues(tt.oldEnum, tt.newEnum)
			if changes == nil {
				t.Fatal("compareEnumValues() returned nil")
			}
			if len(changes) != tt.wantChanges {
				t.Errorf("compareEnumValues() returned %d changes, want %d", len(changes), tt.wantChanges)
			}

			if tt.name == "enum value removed" && len(changes) > 0 {
				change := changes[0]
				if change.Type != EnumValueRemoved {
					t.Errorf("Change type = %v, want %v", change.Type, EnumValueRemoved)
				}
				if change.Severity != Breaking {
					t.Errorf("Change severity = %v, want %v", change.Severity, Breaking)
				}
			}

			if tt.name == "enum value added" && len(changes) > 0 {
				change := changes[0]
				if change.Type != EnumValueAdded {
					t.Errorf("Change type = %v, want %v", change.Type, EnumValueAdded)
				}
				if change.Severity != NonBreaking {
					t.Errorf("Change severity = %v, want %v", change.Severity, NonBreaking)
				}
			}
		})
	}
}

func TestCompareServices(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name        string
		oldServices map[string]*serviceInfo
		newServices map[string]*serviceInfo
		wantChanges int
	}{
		{
			name:        "both empty",
			oldServices: map[string]*serviceInfo{},
			newServices: map[string]*serviceInfo{},
			wantChanges: 0,
		},
		{
			name: "service removed",
			oldServices: map[string]*serviceInfo{
				"UserService": {
					Name:    "UserService",
					Methods: map[string]*methodInfo{},
					File:    "user.proto",
				},
			},
			newServices: map[string]*serviceInfo{},
			wantChanges: 1,
		},
		{
			name:        "service added",
			oldServices: map[string]*serviceInfo{},
			newServices: map[string]*serviceInfo{
				"UserService": {
					Name:    "UserService",
					Methods: map[string]*methodInfo{},
					File:    "user.proto",
				},
			},
			wantChanges: 1,
		},
		{
			name: "service unchanged",
			oldServices: map[string]*serviceInfo{
				"UserService": {
					Name:    "UserService",
					Methods: map[string]*methodInfo{},
					File:    "user.proto",
				},
			},
			newServices: map[string]*serviceInfo{
				"UserService": {
					Name:    "UserService",
					Methods: map[string]*methodInfo{},
					File:    "user.proto",
				},
			},
			wantChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := analyzer.compareServices(tt.oldServices, tt.newServices)
			if changes == nil {
				t.Fatal("compareServices() returned nil")
			}
			if len(changes) != tt.wantChanges {
				t.Errorf("compareServices() returned %d changes, want %d", len(changes), tt.wantChanges)
			}

			if tt.name == "service removed" && len(changes) > 0 {
				change := changes[0]
				if change.Type != ServiceRemoved {
					t.Errorf("Change type = %v, want %v", change.Type, ServiceRemoved)
				}
				if change.Severity != Breaking {
					t.Errorf("Change severity = %v, want %v", change.Severity, Breaking)
				}
			}

			if tt.name == "service added" && len(changes) > 0 {
				change := changes[0]
				if change.Type != ServiceAdded {
					t.Errorf("Change type = %v, want %v", change.Type, ServiceAdded)
				}
				if change.Severity != NonBreaking {
					t.Errorf("Change severity = %v, want %v", change.Severity, NonBreaking)
				}
			}
		})
	}
}

func TestCompareMethods(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name        string
		oldSvc      *serviceInfo
		newSvc      *serviceInfo
		wantChanges int
	}{
		{
			name: "method removed",
			oldSvc: &serviceInfo{
				Name: "UserService",
				File: "user.proto",
				Methods: map[string]*methodInfo{
					"GetUser": {Name: "GetUser", InputType: "GetUserRequest", OutputType: "User"},
				},
			},
			newSvc: &serviceInfo{
				Name:    "UserService",
				File:    "user.proto",
				Methods: map[string]*methodInfo{},
			},
			wantChanges: 1,
		},
		{
			name: "method added",
			oldSvc: &serviceInfo{
				Name:    "UserService",
				File:    "user.proto",
				Methods: map[string]*methodInfo{},
			},
			newSvc: &serviceInfo{
				Name: "UserService",
				File: "user.proto",
				Methods: map[string]*methodInfo{
					"CreateUser": {Name: "CreateUser", InputType: "CreateUserRequest", OutputType: "User"},
				},
			},
			wantChanges: 1,
		},
		{
			name: "no changes",
			oldSvc: &serviceInfo{
				Name: "UserService",
				File: "user.proto",
				Methods: map[string]*methodInfo{
					"GetUser": {Name: "GetUser", InputType: "GetUserRequest", OutputType: "User"},
				},
			},
			newSvc: &serviceInfo{
				Name: "UserService",
				File: "user.proto",
				Methods: map[string]*methodInfo{
					"GetUser": {Name: "GetUser", InputType: "GetUserRequest", OutputType: "User"},
				},
			},
			wantChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := analyzer.compareMethods(tt.oldSvc, tt.newSvc)
			if changes == nil {
				t.Fatal("compareMethods() returned nil")
			}
			if len(changes) != tt.wantChanges {
				t.Errorf("compareMethods() returned %d changes, want %d", len(changes), tt.wantChanges)
			}

			if tt.name == "method removed" && len(changes) > 0 {
				change := changes[0]
				if change.Type != MethodRemoved {
					t.Errorf("Change type = %v, want %v", change.Type, MethodRemoved)
				}
				if change.Severity != Breaking {
					t.Errorf("Change severity = %v, want %v", change.Severity, Breaking)
				}
			}

			if tt.name == "method added" && len(changes) > 0 {
				change := changes[0]
				if change.Type != MethodAdded {
					t.Errorf("Change type = %v, want %v", change.Type, MethodAdded)
				}
				if change.Severity != NonBreaking {
					t.Errorf("Change severity = %v, want %v", change.Severity, NonBreaking)
				}
			}
		})
	}
}

func TestIsBreaking(t *testing.T) {
	tests := []struct {
		name   string
		change Change
		want   bool
	}{
		{
			name: "breaking change",
			change: Change{
				Type:     FieldRemoved,
				Severity: Breaking,
			},
			want: true,
		},
		{
			name: "non-breaking change",
			change: Change{
				Type:     FieldAdded,
				Severity: NonBreaking,
			},
			want: false,
		},
		{
			name: "warning",
			change: Change{
				Type:     ChangeType("unknown"),
				Severity: Warning,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBreaking(tt.change)
			if got != tt.want {
				t.Errorf("IsBreaking() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountByType(t *testing.T) {
	changes := []Change{
		{Type: FieldAdded},
		{Type: FieldRemoved},
		{Type: FieldAdded},
		{Type: MessageAdded},
	}

	tests := []struct {
		name       string
		changeType ChangeType
		want       int
	}{
		{
			name:       "count field added",
			changeType: FieldAdded,
			want:       2,
		},
		{
			name:       "count field removed",
			changeType: FieldRemoved,
			want:       1,
		},
		{
			name:       "count message added",
			changeType: MessageAdded,
			want:       1,
		},
		{
			name:       "count non-existent type",
			changeType: ServiceAdded,
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountByType(changes, tt.changeType)
			if got != tt.want {
				t.Errorf("CountByType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterBySeverity(t *testing.T) {
	changes := []Change{
		{Type: FieldAdded, Severity: NonBreaking},
		{Type: FieldRemoved, Severity: Breaking},
		{Type: MessageAdded, Severity: NonBreaking},
		{Type: TypeChanged, Severity: Breaking},
	}

	tests := []struct {
		name     string
		severity Severity
		want     int
	}{
		{
			name:     "filter breaking",
			severity: Breaking,
			want:     2,
		},
		{
			name:     "filter non-breaking",
			severity: NonBreaking,
			want:     2,
		},
		{
			name:     "filter warning",
			severity: Warning,
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterBySeverity(changes, tt.severity)
			if len(got) != tt.want {
				t.Errorf("FilterBySeverity() returned %d changes, want %d", len(got), tt.want)
			}
			// Verify all returned changes have the correct severity
			for _, change := range got {
				if change.Severity != tt.severity {
					t.Errorf("FilterBySeverity() returned change with severity %v, want %v", change.Severity, tt.severity)
				}
			}
		})
	}
}

func TestFormatLocation(t *testing.T) {
	tests := []struct {
		name     string
		location string
		want     string
	}{
		{
			name:     "simple location",
			location: "file.proto:message User",
			want:     "File.proto → Message User",
		},
		{
			name:     "nested location",
			location: "file.proto:message User:field name",
			want:     "File.proto → Message User → Field name",
		},
		{
			name:     "single part",
			location: "file.proto",
			want:     "File.proto",
		},
		{
			name:     "empty string",
			location: "",
			want:     "",
		},
		{
			name:     "enum location",
			location: "status.proto:enum Status:value ACTIVE",
			want:     "Status.proto → Enum Status → Value ACTIVE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatLocation(tt.location)
			if got != tt.want {
				t.Errorf("FormatLocation() = %v, want %v", got, tt.want)
			}
		})
	}
}
