package module

import (
	"context"
	"strings"
	"testing"

	"github.com/MontFerret/cli/v2/pkg/module/discovery"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

type fakeLifecycle struct {
	query          string
	name           string
	installOptions install.Options
	createOptions  scaffold.Options
	publishOptions modulepublish.Options
	searchResults  []discovery.SearchResult
	info           *discovery.ModuleInfo
	installResult  *install.Result
	createResult   *scaffold.Result
	publishResult  *modulepublish.Result
}

func (f *fakeLifecycle) Search(_ context.Context, query string) ([]discovery.SearchResult, error) {
	f.query = query
	return f.searchResults, nil
}

func (f *fakeLifecycle) Info(_ context.Context, name string) (*discovery.ModuleInfo, error) {
	f.name = name
	return f.info, nil
}

func (f *fakeLifecycle) Install(_ context.Context, options install.Options) (*install.Result, error) {
	f.installOptions = options
	return f.installResult, nil
}

func (f *fakeLifecycle) Create(_ context.Context, options scaffold.Options) (*scaffold.Result, error) {
	f.createOptions = options
	return f.createResult, nil
}

func (f *fakeLifecycle) Publish(_ context.Context, options modulepublish.Options) (*modulepublish.Result, error) {
	f.publishOptions = options
	return f.publishResult, nil
}

func TestServiceDelegatesToWorkflowComponents(t *testing.T) {
	lifecycle := &fakeLifecycle{
		searchResults: []discovery.SearchResult{{Name: "acme/widget"}},
		info:          &discovery.ModuleInfo{Name: "acme/widget"},
		installResult: &install.Result{ID: "acme/widget"},
		createResult:  &scaffold.Result{Directory: "widget"},
		publishResult: &modulepublish.Result{Status: modulepublish.StatusSubmitted},
	}
	service := NewService(lifecycle, lifecycle, lifecycle, lifecycle)
	ctx := context.Background()

	searchResults, err := service.Search(ctx, "widget")
	if err != nil || len(searchResults) != 1 || searchResults[0].Name != "acme/widget" || lifecycle.query != "widget" {
		t.Fatalf("unexpected search delegation: results=%#v query=%q err=%v", searchResults, lifecycle.query, err)
	}

	info, err := service.Info(ctx, "acme/widget")
	if err != nil || info != lifecycle.info || lifecycle.name != "acme/widget" {
		t.Fatalf("unexpected info delegation: info=%#v name=%q err=%v", info, lifecycle.name, err)
	}

	installOptions := install.Options{Reference: "acme/widget@1.0.0", Directory: "."}
	installed, err := service.Install(ctx, installOptions)
	if err != nil || installed != lifecycle.installResult || lifecycle.installOptions != installOptions {
		t.Fatalf("unexpected install delegation: result=%#v options=%#v err=%v", installed, lifecycle.installOptions, err)
	}

	createOptions := scaffold.Options{Name: "acme/widget", GoModule: "example.com/widget"}
	created, err := service.Create(ctx, createOptions)
	if err != nil || created != lifecycle.createResult || lifecycle.createOptions != createOptions {
		t.Fatalf("unexpected create delegation: result=%#v options=%#v err=%v", created, lifecycle.createOptions, err)
	}

	publishOptions := modulepublish.Options{Directory: ".", Tag: "v1.0.0"}
	published, err := service.Publish(ctx, publishOptions)
	if err != nil || published != lifecycle.publishResult || lifecycle.publishOptions != publishOptions {
		t.Fatalf("unexpected publish delegation: result=%#v options=%#v err=%v", published, lifecycle.publishOptions, err)
	}
}

func TestServiceRejectsUnconfiguredComponents(t *testing.T) {
	service := NewService(nil, nil, nil, nil)
	ctx := context.Background()

	if _, err := service.Search(ctx, ""); err == nil || !strings.Contains(err.Error(), "discovery") {
		t.Fatalf("expected discovery configuration error, got %v", err)
	}
	if _, err := service.Info(ctx, "acme/widget"); err == nil || !strings.Contains(err.Error(), "discovery") {
		t.Fatalf("expected discovery configuration error, got %v", err)
	}
	if _, err := service.Install(ctx, install.Options{}); err == nil || !strings.Contains(err.Error(), "installer") {
		t.Fatalf("expected installer configuration error, got %v", err)
	}
	if _, err := service.Create(ctx, scaffold.Options{}); err == nil || !strings.Contains(err.Error(), "scaffolder") {
		t.Fatalf("expected scaffolder configuration error, got %v", err)
	}
	if _, err := service.Publish(ctx, modulepublish.Options{}); err == nil || !strings.Contains(err.Error(), "publisher") {
		t.Fatalf("expected publisher configuration error, got %v", err)
	}
}
