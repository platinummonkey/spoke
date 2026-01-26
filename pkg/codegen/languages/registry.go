package languages

import (
	"context"
	"sync"

	"github.com/platinummonkey/spoke/pkg/plugins"
	"github.com/sirupsen/logrus"
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

// LoadPlugins loads language plugins from a plugin loader
func (r *Registry) LoadPlugins(ctx context.Context, loader *plugins.Loader, log *logrus.Logger) error {
	if log == nil {
		log = logrus.New()
	}

	discoveredPlugins, err := loader.DiscoverPlugins(ctx)
	if err != nil {
		return err
	}

	for _, plugin := range discoveredPlugins {
		if langPlugin, ok := plugin.(plugins.LanguagePlugin); ok {
			pluginSpec := langPlugin.GetLanguageSpec()
			spec := convertPluginLanguageSpec(pluginSpec)

			if err := r.Register(spec); err != nil {
				log.Warnf("Failed to register language plugin %s: %v", plugin.Manifest().ID, err)
				continue
			}

			log.Infof("Registered language plugin: %s (%s)", spec.Name, spec.ID)
		}
	}

	return nil
}

// EnableLanguage enables a language by ID
func (r *Registry) EnableLanguage(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	spec, exists := r.languages[id]
	if !exists {
		return ErrLanguageNotFound
	}

	spec.Enabled = true
	return nil
}

// DisableLanguage disables a language by ID
func (r *Registry) DisableLanguage(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	spec, exists := r.languages[id]
	if !exists {
		return ErrLanguageNotFound
	}

	spec.Enabled = false
	return nil
}

// convertPluginLanguageSpec converts a plugin LanguageSpec to codegen LanguageSpec
func convertPluginLanguageSpec(pluginSpec *plugins.LanguageSpec) *LanguageSpec {
	spec := &LanguageSpec{
		ID:               pluginSpec.ID,
		Name:             pluginSpec.Name,
		DisplayName:      pluginSpec.DisplayName,
		ProtocPlugin:     pluginSpec.ProtocPlugin,
		PluginVersion:    pluginSpec.PluginVersion,
		DockerImage:      pluginSpec.DockerImage,
		SupportsGRPC:     pluginSpec.SupportsGRPC,
		FileExtensions:   pluginSpec.FileExtensions,
		Enabled:          pluginSpec.Enabled,
		Stable:           pluginSpec.Stable,
		Description:      pluginSpec.Description,
		DocumentationURL: pluginSpec.DocumentationURL,
	}

	// Convert package manager if present
	if pluginSpec.PackageManager != nil {
		spec.PackageManager = &PackageManagerSpec{
			Name:            pluginSpec.PackageManager.Name,
			ConfigFiles:     pluginSpec.PackageManager.ConfigFiles,
			DependencyMap:   pluginSpec.PackageManager.DependencyMap,
			DefaultVersions: pluginSpec.PackageManager.DefaultVersions,
		}
	}

	return spec
}
