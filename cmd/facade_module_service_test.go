package cmd

import (
	"context"

	barnpublish "github.com/MontFerret/barn/pkg/publish"

	"github.com/MontFerret/cli/v2/pkg/module/discovery"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

type facadeModuleService struct{}

func (f *facadeModuleService) Search(context.Context, string) ([]discovery.SearchResult, error) {
	return nil, nil
}

func (f *facadeModuleService) Info(context.Context, string) (*discovery.ModuleInfo, error) {
	return nil, nil
}

func (f *facadeModuleService) Install(context.Context, install.Options) (*install.Result, error) {
	return nil, nil
}

func (f *facadeModuleService) Create(context.Context, scaffold.Options) (*scaffold.Result, error) {
	return nil, nil
}

func (f *facadeModuleService) Publish(context.Context, modulepublish.Options) (*barnpublish.Result, error) {
	return nil, nil
}
