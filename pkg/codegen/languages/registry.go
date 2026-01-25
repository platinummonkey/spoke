package languages

import (
	"sync"
)

// Registry manages available language configurations
type Registry struct {
	mu        sync.RWMutex
	languages map[string]*LanguageSpec
}

// NewRegistry creates a new language registry
func NewRegistry() *Registry {
	return &Registry{
		languages: make(map[string]*LanguageSpec),
	}
}

// Register adds a language to the registry
func (r *Registry) Register(spec *LanguageSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.languages[spec.ID]; exists {
		return ErrLanguageAlreadyExists
	}

	r.languages[spec.ID] = spec
	return nil
}

// Get retrieves a language by ID
func (r *Registry) Get(id string) (*LanguageSpec, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	spec, exists := r.languages[id]
	if !exists {
		return nil, ErrLanguageNotFound
	}

	return spec, nil
}

// List returns all registered languages
func (r *Registry) List() []*LanguageSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()

	specs := make([]*LanguageSpec, 0, len(r.languages))
	for _, spec := range r.languages {
		specs = append(specs, spec)
	}

	return specs
}

// ListEnabled returns all enabled languages
func (r *Registry) ListEnabled() []*LanguageSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()

	specs := make([]*LanguageSpec, 0, len(r.languages))
	for _, spec := range r.languages {
		if spec.Enabled {
			specs = append(specs, spec)
		}
	}

	return specs
}

// Update updates an existing language configuration
func (r *Registry) Update(spec *LanguageSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.languages[spec.ID]; !exists {
		return ErrLanguageNotFound
	}

	r.languages[spec.ID] = spec
	return nil
}

// Delete removes a language from the registry
func (r *Registry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.languages[id]; !exists {
		return ErrLanguageNotFound
	}

	delete(r.languages, id)
	return nil
}

// IsEnabled checks if a language is enabled
func (r *Registry) IsEnabled(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	spec, exists := r.languages[id]
	if !exists {
		return false
	}

	return spec.Enabled
}

// Count returns the number of registered languages
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.languages)
}
