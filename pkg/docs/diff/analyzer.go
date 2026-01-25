package diff

import (
	"fmt"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api"
)

// Analyzer analyzes differences between proto file versions
type Analyzer struct{}

// NewAnalyzer creates a new diff analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// Compare compares two versions and returns the differences
func (a *Analyzer) Compare(fromVersion, toVersion *api.Version) (*DiffResult, error) {
	result := &DiffResult{
		FromVersion: fromVersion.Version,
		ToVersion:   toVersion.Version,
		Changes:     []Change{},
	}

	// Build maps for quick lookup
	oldMessages := a.buildMessageMap(fromVersion.Files)
	newMessages := a.buildMessageMap(toVersion.Files)

	oldEnums := a.buildEnumMap(fromVersion.Files)
	newEnums := a.buildEnumMap(toVersion.Files)

	oldServices := a.buildServiceMap(fromVersion.Files)
	newServices := a.buildServiceMap(toVersion.Files)

	// Compare messages
	result.Changes = append(result.Changes, a.compareMessages(oldMessages, newMessages)...)

	// Compare enums
	result.Changes = append(result.Changes, a.compareEnums(oldEnums, newEnums)...)

	// Compare services
	result.Changes = append(result.Changes, a.compareServices(oldServices, newServices)...)

	return result, nil
}

// buildMessageMap builds a map of message name to message
func (a *Analyzer) buildMessageMap(files []api.File) map[string]*messageInfo {
	messages := make(map[string]*messageInfo)

	for range files {
		// Parse file content to extract messages
		// For now, return empty map - full implementation would parse proto files
		// This is a placeholder for the actual parser
	}

	return messages
}

type messageInfo struct {
	Name   string
	Fields map[string]*fieldInfo
	File   string
}

type fieldInfo struct {
	Name   string
	Type   string
	Number int
	Label  string
}

type enumInfo struct {
	Name   string
	Values map[string]int
	File   string
}

type serviceInfo struct {
	Name    string
	Methods map[string]*methodInfo
	File    string
}

type methodInfo struct {
	Name       string
	InputType  string
	OutputType string
}

// buildEnumMap builds a map of enum name to enum
func (a *Analyzer) buildEnumMap(files []api.File) map[string]*enumInfo {
	enums := make(map[string]*enumInfo)
	// Placeholder - actual implementation would parse proto files
	return enums
}

// buildServiceMap builds a map of service name to service
func (a *Analyzer) buildServiceMap(files []api.File) map[string]*serviceInfo {
	services := make(map[string]*serviceInfo)
	// Placeholder - actual implementation would parse proto files
	return services
}

// compareMessages compares messages between versions
func (a *Analyzer) compareMessages(oldMessages, newMessages map[string]*messageInfo) []Change {
	changes := []Change{}

	// Check for removed messages
	for name, oldMsg := range oldMessages {
		if _, exists := newMessages[name]; !exists {
			changes = append(changes, Change{
				Type:         MessageRemoved,
				Severity:     Breaking,
				Location:     fmt.Sprintf("%s:message %s", oldMsg.File, name),
				OldValue:     name,
				Description:  fmt.Sprintf("Message '%s' was removed", name),
				MigrationTip: GetMigrationTip(MessageRemoved, name),
			})
		}
	}

	// Check for added messages
	for name, newMsg := range newMessages {
		if _, exists := oldMessages[name]; !exists {
			changes = append(changes, Change{
				Type:         MessageAdded,
				Severity:     NonBreaking,
				Location:     fmt.Sprintf("%s:message %s", newMsg.File, name),
				NewValue:     name,
				Description:  fmt.Sprintf("Message '%s' was added", name),
			})
		}
	}

	// Compare fields in existing messages
	for name := range oldMessages {
		if newMsg, exists := newMessages[name]; exists {
			oldMsg := oldMessages[name]
			changes = append(changes, a.compareFields(oldMsg, newMsg)...)
		}
	}

	return changes
}

// compareFields compares fields within a message
func (a *Analyzer) compareFields(oldMsg, newMsg *messageInfo) []Change {
	changes := []Change{}

	// Check for removed fields
	for fieldName, oldField := range oldMsg.Fields {
		if _, exists := newMsg.Fields[fieldName]; !exists {
			changes = append(changes, Change{
				Type:         FieldRemoved,
				Severity:     Breaking,
				Location:     fmt.Sprintf("%s:message %s:field %s", oldMsg.File, oldMsg.Name, fieldName),
				OldValue:     fmt.Sprintf("%s %s = %d", oldField.Type, fieldName, oldField.Number),
				Description:  fmt.Sprintf("Field '%s' was removed from message '%s'", fieldName, oldMsg.Name),
				MigrationTip: GetMigrationTip(FieldRemoved, fieldName),
			})
		}
	}

	// Check for added fields
	for fieldName, newField := range newMsg.Fields {
		if _, exists := oldMsg.Fields[fieldName]; !exists {
			changes = append(changes, Change{
				Type:         FieldAdded,
				Severity:     NonBreaking,
				Location:     fmt.Sprintf("%s:message %s:field %s", newMsg.File, newMsg.Name, fieldName),
				NewValue:     fmt.Sprintf("%s %s = %d", newField.Type, fieldName, newField.Number),
				Description:  fmt.Sprintf("Field '%s' was added to message '%s'", fieldName, newMsg.Name),
			})
		}
	}

	// Compare existing fields
	for fieldName := range oldMsg.Fields {
		if newField, exists := newMsg.Fields[fieldName]; exists {
			oldField := oldMsg.Fields[fieldName]

			// Check type changes
			if oldField.Type != newField.Type {
				changes = append(changes, Change{
					Type:         TypeChanged,
					Severity:     Breaking,
					Location:     fmt.Sprintf("%s:message %s:field %s", oldMsg.File, oldMsg.Name, fieldName),
					OldValue:     oldField.Type,
					NewValue:     newField.Type,
					Description:  fmt.Sprintf("Field '%s' type changed from '%s' to '%s'", fieldName, oldField.Type, newField.Type),
					MigrationTip: GetMigrationTip(TypeChanged, fieldName),
				})
			}

			// Check field number changes (critical breaking change)
			if oldField.Number != newField.Number {
				changes = append(changes, Change{
					Type:         FieldNumberChanged,
					Severity:     Breaking,
					Location:     fmt.Sprintf("%s:message %s:field %s", oldMsg.File, oldMsg.Name, fieldName),
					OldValue:     fmt.Sprintf("%d", oldField.Number),
					NewValue:     fmt.Sprintf("%d", newField.Number),
					Description:  fmt.Sprintf("Field '%s' number changed from %d to %d (CRITICAL)", fieldName, oldField.Number, newField.Number),
					MigrationTip: GetMigrationTip(FieldNumberChanged, fieldName),
				})
			}

			// Check label changes (optional vs repeated)
			if oldField.Label != newField.Label {
				changes = append(changes, Change{
					Type:         LabelChanged,
					Severity:     Breaking,
					Location:     fmt.Sprintf("%s:message %s:field %s", oldMsg.File, oldMsg.Name, fieldName),
					OldValue:     oldField.Label,
					NewValue:     newField.Label,
					Description:  fmt.Sprintf("Field '%s' label changed from '%s' to '%s'", fieldName, oldField.Label, newField.Label),
					MigrationTip: GetMigrationTip(LabelChanged, fieldName),
				})
			}
		}
	}

	return changes
}

// compareEnums compares enums between versions
func (a *Analyzer) compareEnums(oldEnums, newEnums map[string]*enumInfo) []Change {
	changes := []Change{}

	// Check for removed enums
	for name, oldEnum := range oldEnums {
		if _, exists := newEnums[name]; !exists {
			changes = append(changes, Change{
				Type:         EnumRemoved,
				Severity:     Breaking,
				Location:     fmt.Sprintf("%s:enum %s", oldEnum.File, name),
				OldValue:     name,
				Description:  fmt.Sprintf("Enum '%s' was removed", name),
				MigrationTip: GetMigrationTip(EnumRemoved, name),
			})
		}
	}

	// Check for added enums
	for name, newEnum := range newEnums {
		if _, exists := oldEnums[name]; !exists {
			changes = append(changes, Change{
				Type:        EnumAdded,
				Severity:    NonBreaking,
				Location:    fmt.Sprintf("%s:enum %s", newEnum.File, name),
				NewValue:    name,
				Description: fmt.Sprintf("Enum '%s' was added", name),
			})
		}
	}

	// Compare enum values
	for name := range oldEnums {
		if newEnum, exists := newEnums[name]; exists {
			oldEnum := oldEnums[name]
			changes = append(changes, a.compareEnumValues(oldEnum, newEnum)...)
		}
	}

	return changes
}

// compareEnumValues compares enum values
func (a *Analyzer) compareEnumValues(oldEnum, newEnum *enumInfo) []Change {
	changes := []Change{}

	// Check for removed enum values
	for valueName := range oldEnum.Values {
		if _, exists := newEnum.Values[valueName]; !exists {
			changes = append(changes, Change{
				Type:         EnumValueRemoved,
				Severity:     Breaking,
				Location:     fmt.Sprintf("%s:enum %s:value %s", oldEnum.File, oldEnum.Name, valueName),
				OldValue:     valueName,
				Description:  fmt.Sprintf("Enum value '%s' was removed from enum '%s'", valueName, oldEnum.Name),
				MigrationTip: GetMigrationTip(EnumValueRemoved, valueName),
			})
		}
	}

	// Check for added enum values
	for valueName := range newEnum.Values {
		if _, exists := oldEnum.Values[valueName]; !exists {
			changes = append(changes, Change{
				Type:        EnumValueAdded,
				Severity:    NonBreaking,
				Location:    fmt.Sprintf("%s:enum %s:value %s", newEnum.File, newEnum.Name, valueName),
				NewValue:    valueName,
				Description: fmt.Sprintf("Enum value '%s' was added to enum '%s'", valueName, newEnum.Name),
			})
		}
	}

	return changes
}

// compareServices compares services between versions
func (a *Analyzer) compareServices(oldServices, newServices map[string]*serviceInfo) []Change {
	changes := []Change{}

	// Check for removed services
	for name, oldSvc := range oldServices {
		if _, exists := newServices[name]; !exists {
			changes = append(changes, Change{
				Type:         ServiceRemoved,
				Severity:     Breaking,
				Location:     fmt.Sprintf("%s:service %s", oldSvc.File, name),
				OldValue:     name,
				Description:  fmt.Sprintf("Service '%s' was removed", name),
				MigrationTip: GetMigrationTip(ServiceRemoved, name),
			})
		}
	}

	// Check for added services
	for name, newSvc := range newServices {
		if _, exists := oldServices[name]; !exists {
			changes = append(changes, Change{
				Type:        ServiceAdded,
				Severity:    NonBreaking,
				Location:    fmt.Sprintf("%s:service %s", newSvc.File, name),
				NewValue:    name,
				Description: fmt.Sprintf("Service '%s' was added", name),
			})
		}
	}

	// Compare methods
	for name := range oldServices {
		if newSvc, exists := newServices[name]; exists {
			oldSvc := oldServices[name]
			changes = append(changes, a.compareMethods(oldSvc, newSvc)...)
		}
	}

	return changes
}

// compareMethods compares methods within a service
func (a *Analyzer) compareMethods(oldSvc, newSvc *serviceInfo) []Change {
	changes := []Change{}

	// Check for removed methods
	for methodName := range oldSvc.Methods {
		if _, exists := newSvc.Methods[methodName]; !exists {
			changes = append(changes, Change{
				Type:         MethodRemoved,
				Severity:     Breaking,
				Location:     fmt.Sprintf("%s:service %s:method %s", oldSvc.File, oldSvc.Name, methodName),
				OldValue:     methodName,
				Description:  fmt.Sprintf("Method '%s' was removed from service '%s'", methodName, oldSvc.Name),
				MigrationTip: GetMigrationTip(MethodRemoved, methodName),
			})
		}
	}

	// Check for added methods
	for methodName := range newSvc.Methods {
		if _, exists := oldSvc.Methods[methodName]; !exists {
			changes = append(changes, Change{
				Type:        MethodAdded,
				Severity:    NonBreaking,
				Location:    fmt.Sprintf("%s:service %s:method %s", newSvc.File, newSvc.Name, methodName),
				NewValue:    methodName,
				Description: fmt.Sprintf("Method '%s' was added to service '%s'", methodName, newSvc.Name),
			})
		}
	}

	return changes
}

// Helper to check if a change is breaking
func IsBreaking(change Change) bool {
	return change.Severity == Breaking
}

// CountByType counts changes by type
func CountByType(changes []Change, changeType ChangeType) int {
	count := 0
	for _, change := range changes {
		if change.Type == changeType {
			count++
		}
	}
	return count
}

// FilterBySeverity filters changes by severity
func FilterBySeverity(changes []Change, severity Severity) []Change {
	filtered := []Change{}
	for _, change := range changes {
		if change.Severity == severity {
			filtered = append(filtered, change)
		}
	}
	return filtered
}

// FormatLocation formats a location string for display
func FormatLocation(location string) string {
	// Split by colons and capitalize first letter of each part
	parts := strings.Split(location, ":")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " → ")
}
