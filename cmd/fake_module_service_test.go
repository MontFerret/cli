package cmd

import (
	"context"

	modulelifecycle "github.com/MontFerret/cli/v2/pkg/module"
)

type fakeModuleService struct {
	searchResults  []modulelifecycle.SearchResult
	info           *modulelifecycle.ModuleInfo
	create         *modulelifecycle.CreateResult
	publication    *modulelifecycle.Publication
	createOptions  modulelifecycle.CreateOptions
	publishOptions modulelifecycle.PublishOptions
	err            error
}

func (f *fakeModuleService) Search(context.Context, string) ([]modulelifecycle.SearchResult, error) {
	return f.searchResults, f.err
}

func (f *fakeModuleService) Info(context.Context, string) (*modulelifecycle.ModuleInfo, error) {
	return f.info, f.err
}

func (f *fakeModuleService) Create(_ context.Context, options modulelifecycle.CreateOptions) (*modulelifecycle.CreateResult, error) {
	f.createOptions = options
	return f.create, f.err
}

func (f *fakeModuleService) Publish(_ context.Context, options modulelifecycle.PublishOptions) (*modulelifecycle.Publication, error) {
	f.publishOptions = options
	return f.publication, f.err
}
