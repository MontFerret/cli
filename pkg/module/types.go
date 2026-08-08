package module

import (
	"context"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

type (
	// Registry provides the public registry operations used by the module service.
	Registry interface {
		Search(context.Context, barnregistry.SearchOptions) ([]barnregistry.ModuleSummary, error)
		Module(context.Context, string) (*barnregistry.Module, error)
		Version(context.Context, string, string) (*barnregistry.Version, error)
	}

	// PublicationPreparer prepares Barn source records for a tagged module release.
	PublicationPreparer interface {
		Prepare(context.Context, PublishOptions) (*barnpublish.Result, error)
	}

	// GoRunner executes Go toolchain commands in a project directory.
	GoRunner interface {
		Run(context.Context, string, ...string) ([]byte, error)
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

	// CreateOptions controls module project scaffolding.
	CreateOptions struct {
		Name      string
		GoModule  string
		Directory string
		Namespace string
	}

	// CreateResult describes a completed scaffold.
	CreateResult struct {
		Directory string
		Namespace string
	}

	// PublishOptions controls local publication preparation.
	PublishOptions struct {
		Directory string
		Tag       string
	}

	// InstallOptions controls installation into an existing Go application.
	InstallOptions struct {
		Reference string
		Directory string
	}

	// InstallResult describes a resolved and validated project installation.
	InstallResult struct {
		ID                  string
		Version             string
		PackagePath         string
		FerretConstraint    string
		ProjectFerret       string
		EditedFile          string
		Changed             bool
		SourceChanged       bool
		DependenciesChanged bool
	}

	// ScaffoldEnvironment pins toolchain and Ferret versions in generated projects.
	ScaffoldEnvironment struct {
		GoVersion     string
		FerretVersion string
	}

	// EnvironmentProvider resolves scaffold dependency versions lazily.
	EnvironmentProvider func() (ScaffoldEnvironment, error)
)
