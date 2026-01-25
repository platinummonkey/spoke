package rules

// Registry interface for registering rules
type Registry interface {
	Register(rule Rule)
}

// RegisterDefaultRules registers all built-in lint rules
func RegisterDefaultRules(registry Registry) {
	// Naming rules
	registry.Register(NewMessageNamingRule())
	registry.Register(NewFieldNamingRule())
	registry.Register(NewServiceNamingRule())
	registry.Register(NewEnumNamingRule())
	registry.Register(NewEnumValueNamingRule())

	// TODO: Add more built-in rules:
	// - Package naming
	// - Comment requirements
	// - Documentation coverage
	// - Deprecation tracking
	// - Structure rules
}
