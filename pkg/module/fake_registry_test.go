package module

import (
	"context"

	"github.com/MontFerret/cli/v2/pkg/registryclient"
)

type fakeRegistry struct {
	catalog  *RegistryCatalog
	modules  map[string]*RegistryModule
	versions map[string]*RegistryVersion
	err      error
}

func (f *fakeRegistry) Catalog(context.Context) (*RegistryCatalog, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.catalog, nil
}

func (f *fakeRegistry) Module(_ context.Context, href string) (*RegistryModule, error) {
	item, exists := f.modules[href]
	if !exists {
		return nil, registryclient.ErrNotFound
	}

	return item, nil
}

func (f *fakeRegistry) Version(_ context.Context, href string) (*RegistryVersion, error) {
	item, exists := f.versions[href]
	if !exists {
		return nil, registryclient.ErrNotFound
	}

	return item, nil
}
