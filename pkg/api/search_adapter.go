package api

import (
	"github.com/platinummonkey/spoke/pkg/search"
)

// SearchStorageAdapter adapts api.Storage to search.StorageReader
// This breaks the import cycle between pkg/api and pkg/search
type SearchStorageAdapter struct {
	storage Storage
}

// NewSearchStorageAdapter creates a new adapter
func NewSearchStorageAdapter(storage Storage) *SearchStorageAdapter {
	return &SearchStorageAdapter{storage: storage}
}

// GetVersion implements search.StorageReader
func (a *SearchStorageAdapter) GetVersion(moduleName, version string) (*search.Version, error) {
	apiVer, err := a.storage.GetVersion(moduleName, version)
	if err != nil {
		return nil, err
	}

	// Convert api.Version to search.Version
	searchVer := &search.Version{
		Version:      apiVer.Version,
		ModuleName:   apiVer.ModuleName,
		Files:        make([]search.FileInfo, len(apiVer.Files)),
		Dependencies: apiVer.Dependencies,
	}

	for i, file := range apiVer.Files {
		searchVer.Files[i] = search.FileInfo{
			Path:    file.Path,
			Content: file.Content,
		}
	}

	return searchVer, nil
}

// GetFile implements search.StorageReader
func (a *SearchStorageAdapter) GetFile(moduleName, version, path string) (*search.File, error) {
	apiFile, err := a.storage.GetFile(moduleName, version, path)
	if err != nil {
		return nil, err
	}

	return &search.File{
		Path:    apiFile.Path,
		Content: []byte(apiFile.Content),
	}, nil
}

// ListModules implements search.StorageReader
func (a *SearchStorageAdapter) ListModules() ([]*search.Module, error) {
	apiModules, err := a.storage.ListModules()
	if err != nil {
		return nil, err
	}

	searchModules := make([]*search.Module, len(apiModules))
	for i, mod := range apiModules {
		searchModules[i] = &search.Module{
			Name:        mod.Name,
			Description: mod.Description,
		}
	}

	return searchModules, nil
}

// ListVersions implements search.StorageReader
func (a *SearchStorageAdapter) ListVersions(moduleName string) ([]*search.Version, error) {
	apiVersions, err := a.storage.ListVersions(moduleName)
	if err != nil {
		return nil, err
	}

	searchVersions := make([]*search.Version, len(apiVersions))
	for i, ver := range apiVersions {
		searchVersions[i] = &search.Version{
			Version:      ver.Version,
			ModuleName:   ver.ModuleName,
			Files:        make([]search.FileInfo, len(ver.Files)),
			Dependencies: ver.Dependencies,
		}

		for j, file := range ver.Files {
			searchVersions[i].Files[j] = search.FileInfo{
				Path:    file.Path,
				Content: file.Content,
			}
		}
	}

	return searchVersions, nil
}
