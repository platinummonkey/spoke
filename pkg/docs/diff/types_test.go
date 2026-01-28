package diff

import (
	"encoding/json"
	"testing"
)

func TestGetSeverity(t *testing.T) {
	tests := []struct {
		name       string
		changeType ChangeType
		want       Severity
	}{
		// Breaking changes
		{
			name:       "FieldRemoved is breaking",
			changeType: FieldRemoved,
			want:       Breaking,
		},
		{
			name:       "FieldRenamed is breaking",
			changeType: FieldRenamed,
			want:       Breaking,
		},
		{
			name:       "TypeChanged is breaking",
			changeType: TypeChanged,
			want:       Breaking,
		},
		{
			name:       "MessageRemoved is breaking",
			changeType: MessageRemoved,
			want:       Breaking,
		},
		{
			name:       "EnumRemoved is breaking",
			changeType: EnumRemoved,
			want:       Breaking,
		},
		{
			name:       "EnumValueRemoved is breaking",
			changeType: EnumValueRemoved,
			want:       Breaking,
		},
		{
			name:       "ServiceRemoved is breaking",
			changeType: ServiceRemoved,
			want:       Breaking,
		},
		{
			name:       "MethodRemoved is breaking",
			changeType: MethodRemoved,
			want:       Breaking,
		},
		{
			name:       "FieldNumberChanged is breaking",
			changeType: FieldNumberChanged,
			want:       Breaking,
		},
		{
			name:       "LabelChanged is breaking",
			changeType: LabelChanged,
			want:       Breaking,
		},
		// Non-breaking changes
		{
			name:       "FieldAdded is non-breaking",
			changeType: FieldAdded,
			want:       NonBreaking,
		},
		{
			name:       "MessageAdded is non-breaking",
			changeType: MessageAdded,
			want:       NonBreaking,
		},
		{
			name:       "EnumAdded is non-breaking",
			changeType: EnumAdded,
			want:       NonBreaking,
		},
		{
			name:       "EnumValueAdded is non-breaking",
			changeType: EnumValueAdded,
			want:       NonBreaking,
		},
		{
			name:       "ServiceAdded is non-breaking",
			changeType: ServiceAdded,
			want:       NonBreaking,
		},
		{
			name:       "MethodAdded is non-breaking",
			changeType: MethodAdded,
			want:       NonBreaking,
		},
		// Unknown change type
		{
			name:       "Unknown change type returns warning",
			changeType: ChangeType("unknown"),
			want:       Warning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSeverity(tt.changeType)
			if got != tt.want {
				t.Errorf("GetSeverity(%v) = %v, want %v", tt.changeType, got, tt.want)
			}
		})
	}
}

func TestGetMigrationTip(t *testing.T) {
	tests := []struct {
		name       string
		changeType ChangeType
		location   string
		want       string
	}{
		{
			name:       "FieldRemoved tip",
			changeType: FieldRemoved,
			location:   "Message.field",
			want:       "Remove all references to this field in your code",
		},
		{
			name:       "FieldRenamed tip",
			changeType: FieldRenamed,
			location:   "Message.field",
			want:       "Update all field references to use the new name",
		},
		{
			name:       "TypeChanged tip",
			changeType: TypeChanged,
			location:   "Message.field",
			want:       "Update code to handle the new field type",
		},
		{
			name:       "MessageRemoved tip",
			changeType: MessageRemoved,
			location:   "Message",
			want:       "Remove all usages of this message type",
		},
		{
			name:       "EnumRemoved tip",
			changeType: EnumRemoved,
			location:   "Enum",
			want:       "Replace enum usages with alternative type",
		},
		{
			name:       "EnumValueRemoved tip",
			changeType: EnumValueRemoved,
			location:   "Enum.VALUE",
			want:       "Update code that uses this enum value",
		},
		{
			name:       "ServiceRemoved tip",
			changeType: ServiceRemoved,
			location:   "Service",
			want:       "Remove all service client implementations",
		},
		{
			name:       "MethodRemoved tip",
			changeType: MethodRemoved,
			location:   "Service.Method",
			want:       "Remove all calls to this method",
		},
		{
			name:       "FieldNumberChanged tip",
			changeType: FieldNumberChanged,
			location:   "Message.field",
			want:       "This is a critical breaking change - regenerate all code and redeploy all services",
		},
		{
			name:       "LabelChanged tip",
			changeType: LabelChanged,
			location:   "Message.field",
			want:       "Update code to handle the new field cardinality (single vs repeated)",
		},
		{
			name:       "FieldAdded returns empty tip",
			changeType: FieldAdded,
			location:   "Message.field",
			want:       "",
		},
		{
			name:       "MessageAdded returns empty tip",
			changeType: MessageAdded,
			location:   "Message",
			want:       "",
		},
		{
			name:       "Unknown change type returns empty tip",
			changeType: ChangeType("unknown"),
			location:   "somewhere",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMigrationTip(tt.changeType, tt.location)
			if got != tt.want {
				t.Errorf("GetMigrationTip(%v, %v) = %v, want %v", tt.changeType, tt.location, got, tt.want)
			}
		})
	}
}

func TestChangeJSON(t *testing.T) {
	tests := []struct {
		name   string
		change Change
	}{
		{
			name: "Complete Change struct",
			change: Change{
				Type:         FieldRemoved,
				Severity:     Breaking,
				Location:     "Message.field",
				OldValue:     "string",
				NewValue:     "",
				Description:  "Field was removed",
				MigrationTip: "Remove all references",
			},
		},
		{
			name: "Minimal Change struct",
			change: Change{
				Type:        FieldAdded,
				Severity:    NonBreaking,
				Location:    "Message.newField",
				Description: "New field added",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.change)
			if err != nil {
				t.Fatalf("Failed to marshal Change: %v", err)
			}

			// Unmarshal back
			var unmarshaled Change
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal Change: %v", err)
			}

			// Verify core fields
			if unmarshaled.Type != tt.change.Type {
				t.Errorf("Type mismatch: got %v, want %v", unmarshaled.Type, tt.change.Type)
			}
			if unmarshaled.Severity != tt.change.Severity {
				t.Errorf("Severity mismatch: got %v, want %v", unmarshaled.Severity, tt.change.Severity)
			}
			if unmarshaled.Location != tt.change.Location {
				t.Errorf("Location mismatch: got %v, want %v", unmarshaled.Location, tt.change.Location)
			}
			if unmarshaled.Description != tt.change.Description {
				t.Errorf("Description mismatch: got %v, want %v", unmarshaled.Description, tt.change.Description)
			}
		})
	}
}

func TestDiffResultJSON(t *testing.T) {
	tests := []struct {
		name   string
		result DiffResult
	}{
		{
			name: "DiffResult with changes",
			result: DiffResult{
				FromVersion: "v1.0.0",
				ToVersion:   "v2.0.0",
				Changes: []Change{
					{
						Type:        FieldRemoved,
						Severity:    Breaking,
						Location:    "Message.oldField",
						Description: "Field removed",
					},
					{
						Type:        FieldAdded,
						Severity:    NonBreaking,
						Location:    "Message.newField",
						Description: "Field added",
					},
				},
			},
		},
		{
			name: "DiffResult with no changes",
			result: DiffResult{
				FromVersion: "v1.0.0",
				ToVersion:   "v1.0.1",
				Changes:     []Change{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("Failed to marshal DiffResult: %v", err)
			}

			// Unmarshal back
			var unmarshaled DiffResult
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal DiffResult: %v", err)
			}

			// Verify fields
			if unmarshaled.FromVersion != tt.result.FromVersion {
				t.Errorf("FromVersion mismatch: got %v, want %v", unmarshaled.FromVersion, tt.result.FromVersion)
			}
			if unmarshaled.ToVersion != tt.result.ToVersion {
				t.Errorf("ToVersion mismatch: got %v, want %v", unmarshaled.ToVersion, tt.result.ToVersion)
			}
			if len(unmarshaled.Changes) != len(tt.result.Changes) {
				t.Errorf("Changes length mismatch: got %v, want %v", len(unmarshaled.Changes), len(tt.result.Changes))
			}
		})
	}
}

func TestChangeTypeConstants(t *testing.T) {
	// Verify all ChangeType constants are defined
	changeTypes := []ChangeType{
		FieldAdded,
		FieldRemoved,
		FieldRenamed,
		TypeChanged,
		MessageAdded,
		MessageRemoved,
		EnumAdded,
		EnumRemoved,
		EnumValueAdded,
		EnumValueRemoved,
		ServiceAdded,
		ServiceRemoved,
		MethodAdded,
		MethodRemoved,
		FieldNumberChanged,
		LabelChanged,
	}

	// Ensure they're all non-empty
	for _, ct := range changeTypes {
		if ct == "" {
			t.Errorf("ChangeType constant is empty")
		}
	}
}

func TestSeverityConstants(t *testing.T) {
	// Verify all Severity constants are defined
	severities := []Severity{
		Breaking,
		NonBreaking,
		Warning,
	}

	// Ensure they're all non-empty
	for _, s := range severities {
		if s == "" {
			t.Errorf("Severity constant is empty")
		}
	}
}
