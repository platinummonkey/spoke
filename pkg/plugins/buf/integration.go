package buf

import (
	"github.com/platinummonkey/spoke/pkg/plugins"
)

// CreateBufPluginFactory creates a factory function for creating Buf plugins
// This is used to integrate Buf plugin support with the plugin loader
func CreateBufPluginFactory() plugins.BufPluginFactory {
	return func(manifest *plugins.Manifest) (plugins.Plugin, error) {
		adapter, err := NewBufPluginAdapterFromManifest(manifest)
		if err != nil {
			return nil, err
		}
		return adapter, nil
	}
}

// ConfigureLoader configures a plugin loader to support Buf plugins
func ConfigureLoader(loader *plugins.Loader) {
	loader.SetBufPluginFactory(CreateBufPluginFactory())
}
