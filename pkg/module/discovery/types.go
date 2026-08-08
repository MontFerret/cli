package discovery

import (
	"context"

	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

type (
	// Registry provides the registry operations used for module discovery.
	Registry interface {
		Search(context.Context, barnregistry.SearchOptions) ([]barnregistry.ModuleSummary, error)
		Module(context.Context, string) (*barnregistry.Module, error)
		Version(context.Context, string, string) (*barnregistry.Version, error)
	}

	// SearchResult is one row in module discovery output.
	SearchResult struct {
		Name        string
		Version     string
		Description string
	}

	// ModuleInfo is the detailed registry view of a module.
	ModuleInfo struct {
		Name            string
		Description     string
		Latest          string
		Newest          string
		SelectedVersion string
		Versions        []string
		Namespace       string
		Ferret          string
		Repository      string
		SourcePath      string
		Commit          string
		Documentation   string
	}
)
