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
	// TODO: Implement message comparison
	// - Check for removed messages
	// - Check for field changes
	// - Check for reserved field changes
}

func (c *Comparator) compareEnums() {
	// TODO: Implement enum comparison
	// - Check for removed enum values
	// - Check for changed enum value numbers
}

func (c *Comparator) compareServices() {
	// TODO: Implement service comparison
	// - Check for removed RPCs
	// - Check for changed input/output types
	// - Check for streaming changes
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
