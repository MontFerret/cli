package cmd

import (
	"context"

	modulelifecycle "github.com/MontFerret/cli/v2/pkg/module"
)

type facadeModuleService struct{}

func (f *facadeModuleService) Search(context.Context, string) ([]modulelifecycle.SearchResult, error) {
	return nil, nil
}

func (f *facadeModuleService) Info(context.Context, string) (*modulelifecycle.ModuleInfo, error) {
	return nil, nil
}

func (f *facadeModuleService) Create(context.Context, modulelifecycle.CreateOptions) (*modulelifecycle.CreateResult, error) {
	return nil, nil
}

func (f *facadeModuleService) Publish(context.Context, modulelifecycle.PublishOptions) (*modulelifecycle.Publication, error) {
	return nil, nil
}
