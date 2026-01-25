package compatibility

import (
	"fmt"
)

// CompatibilityMode defines the type of compatibility checking
type CompatibilityMode int

const (
	CompatibilityModeNone CompatibilityMode = iota
	CompatibilityModeBackward
	CompatibilityModeForward
	CompatibilityModeFull
	CompatibilityModeBackwardTransitive
	CompatibilityModeForwardTransitive
	CompatibilityModeFullTransitive
)

func (m CompatibilityMode) String() string {
	return []string{
		"NONE", "BACKWARD", "FORWARD", "FULL",
		"BACKWARD_TRANSITIVE", "FORWARD_TRANSITIVE", "FULL_TRANSITIVE",
	}[m]
}

// Comparator compares two schema versions for compatibility
type Comparator struct {
	mode       CompatibilityMode
	oldSchema  *SchemaGraph
	newSchema  *SchemaGraph
	violations []Violation
}

// NewComparator creates a new comparator
func NewComparator(mode CompatibilityMode, oldSchema, newSchema *SchemaGraph) *Comparator {
	return &Comparator{
		mode:       mode,
		oldSchema:  oldSchema,
		newSchema:  newSchema,
		violations: make([]Violation, 0),
	}
}

// Violation represents a compatibility violation
type Violation struct {
	Rule           string
	Level          ViolationLevel
	Category       ViolationCategory
	Message        string
	Location       string
	OldValue       string
	NewValue       string
	WireBreaking   bool
	SourceBreaking bool
	Suggestion     string
}

// ViolationLevel indicates the severity
type ViolationLevel int

const (
	ViolationLevelInfo ViolationLevel = iota
	ViolationLevelWarning
	ViolationLevelError
)

func (vl ViolationLevel) String() string {
	return []string{"INFO", "WARNING", "ERROR"}[vl]
}

// ViolationCategory groups related violations
type ViolationCategory int

const (
	CategoryFieldChange ViolationCategory = iota
	CategoryTypeChange
	CategoryEnumChange
	CategoryServiceChange
	CategoryReservedChange
	CategoryPackageChange
	CategoryImportChange
)

func (vc ViolationCategory) String() string {
	return []string{
		"field_change", "type_change", "enum_change", "service_change",
		"reserved_change", "package_change", "import_change",
	}[vc]
}

// CheckResult contains the results of a compatibility check
type CheckResult struct {
	Compatible bool
	Mode       string
	Violations []Violation
	Summary    Summary
}

// Summary provides an overview of violations
type Summary struct {
	TotalViolations int
	Errors          int
	Warnings        int
	Infos           int
	WireBreaking    int
	SourceBreaking  int
}

// Compare runs the compatibility check
func (c *Comparator) Compare() (*CheckResult, error) {
	// Reset violations
	c.violations = make([]Violation, 0)

	// Skip if mode is NONE
	if c.mode == CompatibilityModeNone {
		return &CheckResult{
			Compatible: true,
			Mode:       c.mode.String(),
			Violations: c.violations,
		}, nil
	}

	// Run comparison checks
	c.comparePackages()
	c.compareImports()
	c.compareMessages()
	c.compareEnums()
	c.compareServices()

	// Determine if compatible
	compatible := true
	for _, v := range c.violations {
		if v.Level == ViolationLevelError {
			compatible = false
			break
		}
	}

	// Generate summary
	summary := c.generateSummary()

	return &CheckResult{
		Compatible: compatible,
		Mode:       c.mode.String(),
		Violations: c.violations,
		Summary:    summary,
	}, nil
}

func (c *Comparator) comparePackages() {
	if c.oldSchema.Package != c.newSchema.Package {
		c.addViolation(Violation{
			Rule:           "PACKAGE_CHANGED",
			Level:          ViolationLevelError,
			Category:       CategoryPackageChange,
			Location:       "package",
			OldValue:       c.oldSchema.Package,
			NewValue:       c.newSchema.Package,
			Message:        "Package name changed - breaks all imports",
			WireBreaking:   false,
			SourceBreaking: true,
			Suggestion:     "Create a new package instead of renaming. Consider backward compatibility aliases.",
		})
	}
}

func (c *Comparator) compareImports() {
	// TODO: Implement import comparison
}

func (c *Comparator) compareMessages() {
	// Check for removed messages
	for msgName, oldMsg := range c.oldSchema.Messages {
		if _, exists := c.newSchema.Messages[msgName]; !exists {
			c.addViolation(Violation{
				Rule:           "MESSAGE_REMOVED",
				Level:          ViolationLevelError,
				Category:       CategoryTypeChange,
				Location:       msgName,
				Message:        fmt.Sprintf("Message %s was removed", msgName),
				WireBreaking:   true,
				SourceBreaking: true,
				Suggestion:     "Do not remove messages. Mark as deprecated instead.",
			})
			continue
		}

		newMsg := c.newSchema.Messages[msgName]
		c.compareMessageFields(oldMsg, newMsg)
		c.compareNestedMessages(oldMsg, newMsg)
	}

	// Check for added messages (info only)
	for msgName := range c.newSchema.Messages {
		if _, exists := c.oldSchema.Messages[msgName]; !exists {
			c.addViolation(Violation{
				Rule:           "MESSAGE_ADDED",
				Level:          ViolationLevelInfo,
				Category:       CategoryTypeChange,
				Location:       msgName,
				Message:        fmt.Sprintf("Message %s was added", msgName),
				WireBreaking:   false,
				SourceBreaking: false,
			})
		}
	}
}

func (c *Comparator) compareMessageFields(oldMsg, newMsg *Message) {
	// Check for removed fields
	for fieldNum, oldField := range oldMsg.Fields {
		newField, exists := newMsg.Fields[fieldNum]
		if !exists {
			level := ViolationLevelError
			if c.mode == CompatibilityModeForward || c.mode == CompatibilityModeForwardTransitive {
				level = ViolationLevelWarning
			}
			c.addViolation(Violation{
				Rule:           "FIELD_REMOVED",
				Level:          level,
				Category:       CategoryFieldChange,
				Location:       fmt.Sprintf("%s.%s", oldMsg.FullName, oldField.Name),
				Message:        fmt.Sprintf("Field %d (%s) was removed", fieldNum, oldField.Name),
				WireBreaking:   false,
				SourceBreaking: true,
				Suggestion:     "Mark field as reserved instead of removing it.",
			})
			continue
		}

		c.compareField(oldMsg.FullName, oldField, newField)
	}

	// Check for added fields (info, but check for required fields)
	for fieldNum, newField := range newMsg.Fields {
		if _, exists := oldMsg.Fields[fieldNum]; !exists {
			if newField.Label == FieldLabelRequired {
				c.addViolation(Violation{
					Rule:           "REQUIRED_FIELD_ADDED",
					Level:          ViolationLevelError,
					Category:       CategoryFieldChange,
					Location:       fmt.Sprintf("%s.%s", newMsg.FullName, newField.Name),
					Message:        fmt.Sprintf("Required field %d (%s) was added", fieldNum, newField.Name),
					WireBreaking:   true,
					SourceBreaking: true,
					Suggestion:     "New fields must be optional or repeated.",
				})
			} else {
				c.addViolation(Violation{
					Rule:           "FIELD_ADDED",
					Level:          ViolationLevelInfo,
					Category:       CategoryFieldChange,
					Location:       fmt.Sprintf("%s.%s", newMsg.FullName, newField.Name),
					Message:        fmt.Sprintf("Field %d (%s) was added", fieldNum, newField.Name),
					WireBreaking:   false,
					SourceBreaking: false,
				})
			}
		}
	}
}

func (c *Comparator) compareField(msgName string, oldField, newField *Field) {
	location := fmt.Sprintf("%s.%s", msgName, oldField.Name)

	// Check field name change
	if oldField.Name != newField.Name {
		c.addViolation(Violation{
			Rule:           "FIELD_NAME_CHANGED",
			Level:          ViolationLevelWarning,
			Category:       CategoryFieldChange,
			Location:       location,
			OldValue:       oldField.Name,
			NewValue:       newField.Name,
			Message:        fmt.Sprintf("Field %d name changed from %s to %s", oldField.Number, oldField.Name, newField.Name),
			WireBreaking:   false,
			SourceBreaking: true,
			Suggestion:     "Field name changes break source code compatibility.",
		})
	}

	// Check field type change
	if !c.isTypeCompatible(oldField.Type, newField.Type) {
		c.addViolation(Violation{
			Rule:           "FIELD_TYPE_CHANGED",
			Level:          ViolationLevelError,
			Category:       CategoryTypeChange,
			Location:       location,
			OldValue:       oldField.Type.String(),
			NewValue:       newField.Type.String(),
			Message:        fmt.Sprintf("Field %s type changed from %s to %s (incompatible)", oldField.Name, oldField.Type, newField.Type),
			WireBreaking:   true,
			SourceBreaking: true,
			Suggestion:     "Type changes must be wire-compatible (e.g., int32 ↔ int64, sint32 ↔ sint64).",
		})
	}

	// Check label change (optional/required/repeated)
	if oldField.Label != newField.Label {
		level := ViolationLevelError
		wireBreaking := true

		// Allow optional → repeated in proto3
		if oldField.Label == FieldLabelOptional && newField.Label == FieldLabelRepeated {
			level = ViolationLevelWarning
			wireBreaking = false
		}

		c.addViolation(Violation{
			Rule:           "FIELD_LABEL_CHANGED",
			Level:          level,
			Category:       CategoryFieldChange,
			Location:       location,
			OldValue:       oldField.Label.String(),
			NewValue:       newField.Label.String(),
			Message:        fmt.Sprintf("Field %s label changed from %s to %s", oldField.Name, oldField.Label, newField.Label),
			WireBreaking:   wireBreaking,
			SourceBreaking: true,
			Suggestion:     "Avoid changing field labels. Use new field numbers instead.",
		})
	}

	// Check oneof membership change
	if oldField.InOneOf != newField.InOneOf {
		c.addViolation(Violation{
			Rule:           "FIELD_ONEOF_CHANGED",
			Level:          ViolationLevelError,
			Category:       CategoryFieldChange,
			Location:       location,
			OldValue:       oldField.InOneOf,
			NewValue:       newField.InOneOf,
			Message:        fmt.Sprintf("Field %s oneof membership changed", oldField.Name),
			WireBreaking:   true,
			SourceBreaking: true,
			Suggestion:     "Do not move fields in/out of oneofs.",
		})
	}
}

func (c *Comparator) compareNestedMessages(oldMsg, newMsg *Message) {
	// Recursively compare nested messages
	for nestedName, oldNested := range oldMsg.Nested {
		if newNested, exists := newMsg.Nested[nestedName]; exists {
			c.compareMessageFields(oldNested, newNested)
			c.compareNestedMessages(oldNested, newNested)
		}
	}
}

// isTypeCompatible checks if two field types are wire-compatible
func (c *Comparator) isTypeCompatible(oldType, newType FieldType) bool {
	if oldType == newType {
		return true
	}

	// Wire-compatible type pairs
	compatiblePairs := map[FieldType][]FieldType{
		FieldTypeInt32:    {FieldTypeInt64, FieldTypeUint32, FieldTypeUint64},
		FieldTypeInt64:    {FieldTypeInt32, FieldTypeUint32, FieldTypeUint64},
		FieldTypeUint32:   {FieldTypeInt32, FieldTypeInt64, FieldTypeUint64},
		FieldTypeUint64:   {FieldTypeInt32, FieldTypeInt64, FieldTypeUint32},
		FieldTypeSint32:   {FieldTypeSint64},
		FieldTypeSint64:   {FieldTypeSint32},
		FieldTypeFixed32:  {FieldTypeFixed64, FieldTypeSfixed32},
		FieldTypeFixed64:  {FieldTypeFixed32, FieldTypeSfixed64},
		FieldTypeSfixed32: {FieldTypeSfixed64, FieldTypeFixed32},
		FieldTypeSfixed64: {FieldTypeSfixed32, FieldTypeFixed64},
		FieldTypeString:   {FieldTypeBytes},
		FieldTypeBytes:    {FieldTypeString},
	}

	if compatible, ok := compatiblePairs[oldType]; ok {
		for _, t := range compatible {
			if t == newType {
				return true
			}
		}
	}

	return false
}

func (c *Comparator) compareEnums() {
	// Check for removed enums
	for enumName, oldEnum := range c.oldSchema.Enums {
		if _, exists := c.newSchema.Enums[enumName]; !exists {
			c.addViolation(Violation{
				Rule:           "ENUM_REMOVED",
				Level:          ViolationLevelError,
				Category:       CategoryEnumChange,
				Location:       enumName,
				Message:        fmt.Sprintf("Enum %s was removed", enumName),
				WireBreaking:   true,
				SourceBreaking: true,
				Suggestion:     "Do not remove enums. Mark as deprecated instead.",
			})
			continue
		}

		newEnum := c.newSchema.Enums[enumName]
		c.compareEnumValues(oldEnum, newEnum)
	}

	// Check for added enums (info only)
	for enumName := range c.newSchema.Enums {
		if _, exists := c.oldSchema.Enums[enumName]; !exists {
			c.addViolation(Violation{
				Rule:           "ENUM_ADDED",
				Level:          ViolationLevelInfo,
				Category:       CategoryEnumChange,
				Location:       enumName,
				Message:        fmt.Sprintf("Enum %s was added", enumName),
				WireBreaking:   false,
				SourceBreaking: false,
			})
		}
	}
}

func (c *Comparator) compareEnumValues(oldEnum, newEnum *Enum) {
	// Check for removed enum values
	for valueNum, oldValue := range oldEnum.Values {
		if _, exists := newEnum.Values[valueNum]; !exists {
			level := ViolationLevelError
			if c.mode == CompatibilityModeForward || c.mode == CompatibilityModeForwardTransitive {
				level = ViolationLevelWarning
			}
			c.addViolation(Violation{
				Rule:           "ENUM_VALUE_REMOVED",
				Level:          level,
				Category:       CategoryEnumChange,
				Location:       fmt.Sprintf("%s.%s", oldEnum.FullName, oldValue.Name),
				Message:        fmt.Sprintf("Enum value %d (%s) was removed", valueNum, oldValue.Name),
				WireBreaking:   false,
				SourceBreaking: true,
				Suggestion:     "Do not remove enum values. Mark as deprecated or reserve the number.",
			})
		}
	}

	// Check for changed enum value numbers (by name)
	for valueName, oldValue := range oldEnum.ValuesByName {
		if newValue, exists := newEnum.ValuesByName[valueName]; exists {
			if oldValue.Number != newValue.Number {
				c.addViolation(Violation{
					Rule:           "ENUM_VALUE_NUMBER_CHANGED",
					Level:          ViolationLevelError,
					Category:       CategoryEnumChange,
					Location:       fmt.Sprintf("%s.%s", oldEnum.FullName, valueName),
					OldValue:       fmt.Sprintf("%d", oldValue.Number),
					NewValue:       fmt.Sprintf("%d", newValue.Number),
					Message:        fmt.Sprintf("Enum value %s number changed from %d to %d", valueName, oldValue.Number, newValue.Number),
					WireBreaking:   true,
					SourceBreaking: true,
					Suggestion:     "Never change enum value numbers.",
				})
			}
		}
	}

	// Check for added enum values (info)
	for valueNum, newValue := range newEnum.Values {
		if _, exists := oldEnum.Values[valueNum]; !exists {
			c.addViolation(Violation{
				Rule:           "ENUM_VALUE_ADDED",
				Level:          ViolationLevelInfo,
				Category:       CategoryEnumChange,
				Location:       fmt.Sprintf("%s.%s", newEnum.FullName, newValue.Name),
				Message:        fmt.Sprintf("Enum value %d (%s) was added", valueNum, newValue.Name),
				WireBreaking:   false,
				SourceBreaking: false,
			})
		}
	}
}

func (c *Comparator) compareServices() {
	// Check for removed services
	for svcName, oldSvc := range c.oldSchema.Services {
		if _, exists := c.newSchema.Services[svcName]; !exists {
			c.addViolation(Violation{
				Rule:           "SERVICE_REMOVED",
				Level:          ViolationLevelError,
				Category:       CategoryServiceChange,
				Location:       svcName,
				Message:        fmt.Sprintf("Service %s was removed", svcName),
				WireBreaking:   true,
				SourceBreaking: true,
				Suggestion:     "Do not remove services. Mark as deprecated instead.",
			})
			continue
		}

		newSvc := c.newSchema.Services[svcName]
		c.compareServiceMethods(oldSvc, newSvc)
	}

	// Check for added services (info)
	for svcName := range c.newSchema.Services {
		if _, exists := c.oldSchema.Services[svcName]; !exists {
			c.addViolation(Violation{
				Rule:           "SERVICE_ADDED",
				Level:          ViolationLevelInfo,
				Category:       CategoryServiceChange,
				Location:       svcName,
				Message:        fmt.Sprintf("Service %s was added", svcName),
				WireBreaking:   false,
				SourceBreaking: false,
			})
		}
	}
}

func (c *Comparator) compareServiceMethods(oldSvc, newSvc *Service) {
	// Check for removed methods
	for methodName, oldMethod := range oldSvc.Methods {
		if _, exists := newSvc.Methods[methodName]; !exists {
			c.addViolation(Violation{
				Rule:           "RPC_REMOVED",
				Level:          ViolationLevelError,
				Category:       CategoryServiceChange,
				Location:       fmt.Sprintf("%s.%s", oldSvc.FullName, methodName),
				Message:        fmt.Sprintf("RPC method %s was removed", methodName),
				WireBreaking:   true,
				SourceBreaking: true,
				Suggestion:     "Do not remove RPC methods. Mark as deprecated instead.",
			})
			continue
		}

		newMethod := newSvc.Methods[methodName]
		c.compareMethod(oldSvc.FullName, oldMethod, newMethod)
	}

	// Check for added methods (info)
	for methodName := range newSvc.Methods {
		if _, exists := oldSvc.Methods[methodName]; !exists {
			c.addViolation(Violation{
				Rule:           "RPC_ADDED",
				Level:          ViolationLevelInfo,
				Category:       CategoryServiceChange,
				Location:       fmt.Sprintf("%s.%s", newSvc.FullName, methodName),
				Message:        fmt.Sprintf("RPC method %s was added", methodName),
				WireBreaking:   false,
				SourceBreaking: false,
			})
		}
	}
}

func (c *Comparator) compareMethod(svcName string, oldMethod, newMethod *Method) {
	location := fmt.Sprintf("%s.%s", svcName, oldMethod.Name)

	// Check input type change
	if oldMethod.InputType != newMethod.InputType {
		c.addViolation(Violation{
			Rule:           "RPC_INPUT_TYPE_CHANGED",
			Level:          ViolationLevelError,
			Category:       CategoryServiceChange,
			Location:       location,
			OldValue:       oldMethod.InputType,
			NewValue:       newMethod.InputType,
			Message:        fmt.Sprintf("RPC %s input type changed from %s to %s", oldMethod.Name, oldMethod.InputType, newMethod.InputType),
			WireBreaking:   true,
			SourceBreaking: true,
			Suggestion:     "Create a new RPC method instead of changing types.",
		})
	}

	// Check output type change
	if oldMethod.OutputType != newMethod.OutputType {
		c.addViolation(Violation{
			Rule:           "RPC_OUTPUT_TYPE_CHANGED",
			Level:          ViolationLevelError,
			Category:       CategoryServiceChange,
			Location:       location,
			OldValue:       oldMethod.OutputType,
			NewValue:       newMethod.OutputType,
			Message:        fmt.Sprintf("RPC %s output type changed from %s to %s", oldMethod.Name, oldMethod.OutputType, newMethod.OutputType),
			WireBreaking:   true,
			SourceBreaking: true,
			Suggestion:     "Create a new RPC method instead of changing types.",
		})
	}

	// Check streaming changes
	if oldMethod.ClientStreaming != newMethod.ClientStreaming {
		c.addViolation(Violation{
			Rule:           "RPC_CLIENT_STREAMING_CHANGED",
			Level:          ViolationLevelError,
			Category:       CategoryServiceChange,
			Location:       location,
			OldValue:       fmt.Sprintf("%v", oldMethod.ClientStreaming),
			NewValue:       fmt.Sprintf("%v", newMethod.ClientStreaming),
			Message:        fmt.Sprintf("RPC %s client streaming changed", oldMethod.Name),
			WireBreaking:   true,
			SourceBreaking: true,
			Suggestion:     "Cannot change streaming behavior. Create a new RPC method.",
		})
	}

	if oldMethod.ServerStreaming != newMethod.ServerStreaming {
		c.addViolation(Violation{
			Rule:           "RPC_SERVER_STREAMING_CHANGED",
			Level:          ViolationLevelError,
			Category:       CategoryServiceChange,
			Location:       location,
			OldValue:       fmt.Sprintf("%v", oldMethod.ServerStreaming),
			NewValue:       fmt.Sprintf("%v", newMethod.ServerStreaming),
			Message:        fmt.Sprintf("RPC %s server streaming changed", oldMethod.Name),
			WireBreaking:   true,
			SourceBreaking: true,
			Suggestion:     "Cannot change streaming behavior. Create a new RPC method.",
		})
	}
}

func (c *Comparator) addViolation(v Violation) {
	c.violations = append(c.violations, v)
}

func (c *Comparator) generateSummary() Summary {
	summary := Summary{
		TotalViolations: len(c.violations),
	}

	for _, v := range c.violations {
		switch v.Level {
		case ViolationLevelError:
			summary.Errors++
		case ViolationLevelWarning:
			summary.Warnings++
		case ViolationLevelInfo:
			summary.Infos++
		}

		if v.WireBreaking {
			summary.WireBreaking++
		}
		if v.SourceBreaking {
			summary.SourceBreaking++
		}
	}

	return summary
}

// ViolationBuilder helps construct violations fluently
type ViolationBuilder struct {
	violation Violation
}

// NewViolationBuilder creates a new violation builder
func NewViolationBuilder(rule string) *ViolationBuilder {
	return &ViolationBuilder{
		violation: Violation{
			Rule: rule,
		},
	}
}

func (b *ViolationBuilder) WithLevel(level ViolationLevel) *ViolationBuilder {
	b.violation.Level = level
	return b
}

func (b *ViolationBuilder) WithCategory(category ViolationCategory) *ViolationBuilder {
	b.violation.Category = category
	return b
}

func (b *ViolationBuilder) WithLocation(location string) *ViolationBuilder {
	b.violation.Location = location
	return b
}

func (b *ViolationBuilder) WithMessage(message string) *ViolationBuilder {
	b.violation.Message = message
	return b
}

func (b *ViolationBuilder) WithChange(oldValue, newValue string) *ViolationBuilder {
	b.violation.OldValue = oldValue
	b.violation.NewValue = newValue
	return b
}

func (b *ViolationBuilder) WithWireBreaking(breaking bool) *ViolationBuilder {
	b.violation.WireBreaking = breaking
	return b
}

func (b *ViolationBuilder) WithSourceBreaking(breaking bool) *ViolationBuilder {
	b.violation.SourceBreaking = breaking
	return b
}

func (b *ViolationBuilder) WithSuggestion(suggestion string) *ViolationBuilder {
	b.violation.Suggestion = suggestion
	return b
}

func (b *ViolationBuilder) Build() Violation {
	return b.violation
}

// Helper function to check if schemas are compatible
func CheckCompatibility(oldSchema, newSchema *SchemaGraph, mode CompatibilityMode) (*CheckResult, error) {
	comparator := NewComparator(mode, oldSchema, newSchema)
	return comparator.Compare()
}

// ParseCompatibilityMode converts a string to CompatibilityMode
func ParseCompatibilityMode(s string) (CompatibilityMode, error) {
	modes := map[string]CompatibilityMode{
		"NONE":                  CompatibilityModeNone,
		"BACKWARD":              CompatibilityModeBackward,
		"FORWARD":               CompatibilityModeForward,
		"FULL":                  CompatibilityModeFull,
		"BACKWARD_TRANSITIVE":   CompatibilityModeBackwardTransitive,
		"FORWARD_TRANSITIVE":    CompatibilityModeForwardTransitive,
		"FULL_TRANSITIVE":       CompatibilityModeFullTransitive,
	}

	if mode, ok := modes[s]; ok {
		return mode, nil
	}
	return CompatibilityModeNone, fmt.Errorf("unknown compatibility mode: %s", s)
}
