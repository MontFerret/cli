package install

import (
	"context"

	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

type fakeRegistry struct {
	modules  map[string]*barnregistry.Module
	versions map[string]*barnregistry.Version
}

func (registry *fakeRegistry) Module(_ context.Context, id string) (*barnregistry.Module, error) {
	item, exists := registry.modules[id]
	if !exists {
		return nil, barnregistry.ErrModuleNotFound
	}

	return item, nil
}

func (registry *fakeRegistry) Version(_ context.Context, id, version string) (*barnregistry.Version, error) {
	item, exists := registry.versions[id+"@"+version]
	if !exists {
		return nil, barnregistry.ErrVersionNotFound
	}

	return item, nil
}
