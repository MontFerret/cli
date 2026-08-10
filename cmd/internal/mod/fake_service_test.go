package mod

import (
	"context"

	"github.com/MontFerret/cli/v2/pkg/module/discovery"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

type fakeModuleService struct {
	searchResults   []discovery.SearchResult
	info            *discovery.ModuleInfo
	install         *install.Result
	create          *scaffold.Result
	publication     *modulepublish.Result
	createOptions   scaffold.Options
	installOptions  install.Options
	publishOptions  modulepublish.Options
	publishCalls    int
	createCalls     int
	installCalls    int
	installHistory  []install.Options
	installSequence []fakeInstallResponse
	err             error
}

type fakeInstallResponse struct {
	result *install.Result
	err    error
}

func (f *fakeModuleService) Search(context.Context, string) ([]discovery.SearchResult, error) {
	return f.searchResults, f.err
}

func (f *fakeModuleService) Info(context.Context, string) (*discovery.ModuleInfo, error) {
	return f.info, f.err
}

func (f *fakeModuleService) Install(_ context.Context, options install.Options) (*install.Result, error) {
	f.installOptions = options
	f.installHistory = append(f.installHistory, options)
	call := f.installCalls
	f.installCalls++
	if call < len(f.installSequence) {
		response := f.installSequence[call]
		return response.result, response.err
	}
	return f.install, f.err
}

func (f *fakeModuleService) Create(_ context.Context, options scaffold.Options) (*scaffold.Result, error) {
	f.createOptions = options
	f.createCalls++
	return f.create, f.err
}

func (f *fakeModuleService) Publish(_ context.Context, options modulepublish.Options) (*modulepublish.Result, error) {
	f.publishOptions = options
	f.publishCalls++
	return f.publication, f.err
}
