package module

import (
	"context"
	"errors"
	"testing"

	"github.com/MontFerret/cli/v2/pkg/registryclient"
)

func TestServiceSearchFiltersAndSorts(t *testing.T) {
	registry := &fakeRegistry{
		catalog: &RegistryCatalog{Modules: []registryclient.ModuleReference{
			{ID: "zeta/http", Href: "/zeta"},
			{ID: "acme/sqlite", Href: "/sqlite"},
		}},
		modules: map[string]*RegistryModule{
			"/zeta": {
				ID: "zeta/http", Description: "HTTP tools", Latest: "1.0.0",
				Versions: []registryclient.ModuleVersionReference{{Version: "1.0.0", Href: "/zeta/1.0.0"}},
			},
			"/sqlite": {
				ID: "acme/sqlite", Description: "Embedded database",
				Versions: []registryclient.ModuleVersionReference{{Version: "2.0.0-rc.1", Href: "/sqlite/2.0.0-rc.1"}},
			},
		},
	}
	service := NewService(registry, nil, nil)

	results, err := service.Search(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Name != "acme/sqlite" || results[0].Version != "2.0.0-rc.1" || results[1].Name != "zeta/http" {
		t.Fatalf("unexpected results: %#v", results)
	}

	results, err = service.Search(context.Background(), "DATABASE")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "acme/sqlite" {
		t.Fatalf("unexpected filtered results: %#v", results)
	}
}

func TestServiceInfoSelectsLatestOrNewest(t *testing.T) {
	registry := &fakeRegistry{
		catalog: &RegistryCatalog{Modules: []registryclient.ModuleReference{{ID: "acme/sqlite", Href: "/sqlite"}}},
		modules: map[string]*RegistryModule{
			"/sqlite": {
				ID: "acme/sqlite", Description: "SQLite", Latest: "1.0.0",
				Versions: []registryclient.ModuleVersionReference{
					{Version: "2.0.0-rc.1", Href: "/sqlite/2.0.0-rc.1"},
					{Version: "1.0.0", Href: "/sqlite/1.0.0"},
				},
			},
		},
		versions: map[string]*RegistryVersion{
			"/sqlite/1.0.0": {
				ID: "acme/sqlite", Version: "1.0.0", Namespace: "DB::SQLITE",
				Source:        registryclient.Source{Repository: "https://example.com/acme/sqlite", Commit: "abc"},
				Documentation: "https://registry.example/docs.md",
			},
		},
	}
	service := NewService(registry, nil, nil)

	info, err := service.Info(context.Background(), "acme/sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if info.SelectedVersion != "1.0.0" || info.Newest != "2.0.0-rc.1" || info.Namespace != "DB::SQLITE" {
		t.Fatalf("unexpected info: %#v", info)
	}

	_, err = service.Info(context.Background(), "missing/module")
	if !errors.Is(err, registryclient.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestServicePropagatesRegistryFailure(t *testing.T) {
	service := NewService(&fakeRegistry{err: registryclient.ErrUnavailable}, nil, nil)
	if _, err := service.Search(context.Background(), ""); !errors.Is(err, registryclient.ErrUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}
