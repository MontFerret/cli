package discovery

import (
	"context"
	"errors"
	"reflect"
	"testing"

	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

func TestServiceSearchUsesBarnFilteringAndProjectsDetails(t *testing.T) {
	registry := &fakeRegistry{
		summaries: []barnregistry.ModuleSummary{
			{ID: "zeta/http", Latest: "1.0.0"},
			{ID: "acme/sqlite"},
		},
		modules: map[string]*barnregistry.Module{
			"zeta/http": {
				ID: "zeta/http", Description: "HTTP tools",
				Versions: []barnregistry.VersionSummary{{Version: "1.0.0"}},
			},
			"acme/sqlite": {
				ID: "acme/sqlite", Description: "Embedded database",
				Versions: []barnregistry.VersionSummary{{Version: "2.0.0-rc.1"}},
			},
		},
	}
	service := New(registry)

	results, err := service.Search(context.Background(), "SQLITE")
	if err != nil {
		t.Fatal(err)
	}

	if registry.searchOptions != (barnregistry.SearchOptions{Query: "SQLITE"}) {
		t.Fatalf("unexpected search options: %#v", registry.searchOptions)
	}

	want := []SearchResult{
		{Name: "acme/sqlite", Version: "2.0.0-rc.1", Description: "Embedded database"},
		{Name: "zeta/http", Version: "1.0.0", Description: "HTTP tools"},
	}
	if !reflect.DeepEqual(results, want) {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestServiceSearchForwardsDescriptionQuery(t *testing.T) {
	registry := &fakeRegistry{
		search: func(_ context.Context, options barnregistry.SearchOptions) ([]barnregistry.ModuleSummary, error) {
			if options != (barnregistry.SearchOptions{Query: "ai"}) {
				t.Fatalf("unexpected search options: %#v", options)
			}

			return []barnregistry.ModuleSummary{{ID: "montferret/llm"}}, nil
		},
		modules: map[string]*barnregistry.Module{
			"montferret/llm": {
				ID:          "montferret/llm",
				Description: "Text generation under AI::LLM for Ferret.",
				Versions:    []barnregistry.VersionSummary{{Version: "1.0.0-rc.4"}},
			},
		},
	}
	service := New(registry)

	results, err := service.Search(context.Background(), "ai")
	if err != nil {
		t.Fatal(err)
	}

	want := []SearchResult{{
		Name: "montferret/llm", Version: "1.0.0-rc.4", Description: "Text generation under AI::LLM for Ferret.",
	}}
	if !reflect.DeepEqual(results, want) {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestServiceSearchPropagatesBarnAndCancellationErrors(t *testing.T) {
	t.Run("typed Barn error", func(t *testing.T) {
		failure := &barnregistry.HTTPError{URL: "https://registry.example/index.json", StatusCode: 503, Status: "503 Service Unavailable"}
		service := New(&fakeRegistry{searchErr: failure})

		_, err := service.Search(context.Background(), "")
		var httpError *barnregistry.HTTPError
		if !errors.As(err, &httpError) || httpError != failure {
			t.Fatalf("expected Barn HTTP error, got %v", err)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		service := New(&fakeRegistry{search: func(ctx context.Context, _ barnregistry.SearchOptions) ([]barnregistry.ModuleSummary, error) {
			<-ctx.Done()

			return nil, ctx.Err()
		}})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := service.Search(ctx, "")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	})
}

func TestServiceInfoSelectsLatestOrNewest(t *testing.T) {
	registry := &fakeRegistry{
		modules: map[string]*barnregistry.Module{
			"acme/sqlite": {
				ID:          "acme/sqlite",
				Description: "SQLite",
				Latest:      "1.0.0",
				Versions: []barnregistry.VersionSummary{
					{Version: "2.0.0-rc.1"},
					{Version: "1.0.0"},
				},
			},
			"acme/preview": {
				ID:          "acme/preview",
				Description: "Preview",
				Versions:    []barnregistry.VersionSummary{{Version: "1.0.0-rc.1"}},
			},
		},
		versions: map[string]*barnregistry.Version{
			"acme/sqlite@1.0.0": {
				ID: "acme/sqlite", Version: "1.0.0", Namespace: "DB::SQLITE", Ferret: ">=2.0.0 <3.0.0",
				Source:  barnregistry.Source{Repository: "https://example.com/acme/sqlite", Path: "modules/sqlite", Commit: "abc"},
				Content: map[string]string{"documentation": "https://registry.example/docs.md"},
			},
			"acme/preview@1.0.0-rc.1": {
				ID: "acme/preview", Version: "1.0.0-rc.1", Namespace: "PREVIEW",
				Source: barnregistry.Source{Repository: "https://example.com/acme/preview", Commit: "def"},
			},
		},
	}
	service := New(registry)

	info, err := service.Info(context.Background(), "acme/sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if info.SelectedVersion != "1.0.0" || info.Newest != "2.0.0-rc.1" || info.Namespace != "DB::SQLITE" || info.Documentation != "https://registry.example/docs.md" {
		t.Fatalf("unexpected stable info: %#v", info)
	}

	info, err = service.Info(context.Background(), "acme/preview")
	if err != nil {
		t.Fatal(err)
	}
	if info.Latest != "" || info.SelectedVersion != "1.0.0-rc.1" || info.Newest != "1.0.0-rc.1" {
		t.Fatalf("unexpected prerelease info: %#v", info)
	}
}

func TestServiceInfoPropagatesBarnErrors(t *testing.T) {
	service := New(&fakeRegistry{moduleErr: barnregistry.ErrModuleNotFound})
	if _, err := service.Info(context.Background(), "missing/module"); !errors.Is(err, barnregistry.ErrModuleNotFound) {
		t.Fatalf("expected module-not-found error, got %v", err)
	}

	service = New(&fakeRegistry{modules: map[string]*barnregistry.Module{
		"acme/empty": {ID: "acme/empty"},
	}})
	if _, err := service.Info(context.Background(), "acme/empty"); !errors.Is(err, barnregistry.ErrMalformedArtifact) {
		t.Fatalf("expected malformed-artifact error, got %v", err)
	}
}

type fakeRegistry struct {
	summaries     []barnregistry.ModuleSummary
	modules       map[string]*barnregistry.Module
	versions      map[string]*barnregistry.Version
	search        func(context.Context, barnregistry.SearchOptions) ([]barnregistry.ModuleSummary, error)
	searchOptions barnregistry.SearchOptions
	searchErr     error
	moduleErr     error
	versionErr    error
}

func (registry *fakeRegistry) Search(ctx context.Context, options barnregistry.SearchOptions) ([]barnregistry.ModuleSummary, error) {
	registry.searchOptions = options
	if registry.search != nil {
		return registry.search(ctx, options)
	}

	return registry.summaries, registry.searchErr
}

func (registry *fakeRegistry) Module(_ context.Context, id string) (*barnregistry.Module, error) {
	if registry.moduleErr != nil {
		return nil, registry.moduleErr
	}

	item, exists := registry.modules[id]
	if !exists {
		return nil, barnregistry.ErrModuleNotFound
	}

	return item, nil
}

func (registry *fakeRegistry) Version(_ context.Context, id, version string) (*barnregistry.Version, error) {
	if registry.versionErr != nil {
		return nil, registry.versionErr
	}

	item, exists := registry.versions[id+"@"+version]
	if !exists {
		return nil, barnregistry.ErrVersionNotFound
	}

	return item, nil
}
