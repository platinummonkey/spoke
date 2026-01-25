package diff

// ChangeType represents the type of change detected
type ChangeType string

const (
	FieldAdded        ChangeType = "field_added"
	FieldRemoved      ChangeType = "field_removed"
	FieldRenamed      ChangeType = "field_renamed"
	TypeChanged       ChangeType = "type_changed"
	MessageAdded      ChangeType = "message_added"
	MessageRemoved    ChangeType = "message_removed"
	EnumAdded         ChangeType = "enum_added"
	EnumRemoved       ChangeType = "enum_removed"
	EnumValueAdded    ChangeType = "enum_value_added"
	EnumValueRemoved  ChangeType = "enum_value_removed"
	ServiceAdded      ChangeType = "service_added"
	ServiceRemoved    ChangeType = "service_removed"
	MethodAdded       ChangeType = "method_added"
	MethodRemoved     ChangeType = "method_removed"
	FieldNumberChanged ChangeType = "field_number_changed"
	LabelChanged      ChangeType = "label_changed"
)

// Severity represents the severity level of a change
type Severity string

const (
	Breaking    Severity = "breaking"
	NonBreaking Severity = "non_breaking"
	Warning     Severity = "warning"
)

// Change represents a single change between two versions
type Change struct {
	Type         ChangeType `json:"type"`
	Severity     Severity   `json:"severity"`
	Location     string     `json:"location"`
	OldValue     string     `json:"old_value,omitempty"`
	NewValue     string     `json:"new_value,omitempty"`
	Description  string     `json:"description"`
	MigrationTip string     `json:"migration_tip,omitempty"`
}

// DiffResult contains all changes detected between two versions
type DiffResult struct {
	FromVersion string   `json:"from_version"`
	ToVersion   string   `json:"to_version"`
	Changes     []Change `json:"changes"`
}

// GetSeverity determines the severity of a change based on its type
func GetSeverity(changeType ChangeType) Severity {
	switch changeType {
	case FieldRemoved, FieldRenamed, TypeChanged, MessageRemoved,
		EnumRemoved, EnumValueRemoved, ServiceRemoved, MethodRemoved,
		FieldNumberChanged:
		return Breaking

	case FieldAdded, MessageAdded, EnumAdded, EnumValueAdded,
		ServiceAdded, MethodAdded:
		// Field added could be breaking if it's required, but proto3 has no required
		return NonBreaking

	case LabelChanged:
		// Changing from optional to repeated or vice versa is breaking
		return Breaking

	default:
		return Warning
	}
}

// GetMigrationTip provides a migration tip based on change type
func GetMigrationTip(changeType ChangeType, location string) string {
	switch changeType {
	case FieldRemoved:
		return "Remove all references to this field in your code"
	case FieldRenamed:
		return "Update all field references to use the new name"
	case TypeChanged:
		return "Update code to handle the new field type"
	case MessageRemoved:
		return "Remove all usages of this message type"
	case EnumRemoved:
		return "Replace enum usages with alternative type"
	case EnumValueRemoved:
		return "Update code that uses this enum value"
	case ServiceRemoved:
		return "Remove all service client implementations"
	case MethodRemoved:
		return "Remove all calls to this method"
	case FieldNumberChanged:
		return "This is a critical breaking change - regenerate all code and redeploy all services"
	case LabelChanged:
		return "Update code to handle the new field cardinality (single vs repeated)"
	default:
		return ""
	}
}
