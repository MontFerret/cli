package cmd

import (
	"context"

	barnpublish "github.com/MontFerret/barn/pkg/publish"

	modulelifecycle "github.com/MontFerret/cli/v2/pkg/module"
)

type facadeModuleService struct{}

func (f *facadeModuleService) Search(context.Context, string) ([]modulelifecycle.SearchResult, error) {
	return nil, nil
}

func (f *facadeModuleService) Info(context.Context, string) (*modulelifecycle.ModuleInfo, error) {
	return nil, nil
}

func (f *facadeModuleService) Install(context.Context, modulelifecycle.InstallOptions) (*modulelifecycle.InstallResult, error) {
	return nil, nil
}

func (f *facadeModuleService) Create(context.Context, modulelifecycle.CreateOptions) (*modulelifecycle.CreateResult, error) {
	return nil, nil
}

func (f *facadeModuleService) Publish(context.Context, modulelifecycle.PublishOptions) (*barnpublish.Result, error) {
	return nil, nil
}
