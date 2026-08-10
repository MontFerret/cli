package module

import (
	"context"

	"github.com/MontFerret/cli/v2/pkg/module/discovery"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

type (
	// Discovery provides module search and information workflows.
	Discovery interface {
		Search(context.Context, string) ([]discovery.SearchResult, error)
		Info(context.Context, string) (*discovery.ModuleInfo, error)
	}

	// Installer installs a registered module into a Go application.
	Installer interface {
		Install(context.Context, install.Options) (*install.Result, error)
	}

	// Scaffolder creates a new module project.
	Scaffolder interface {
		Create(context.Context, scaffold.Options) (*scaffold.Result, error)
	}

	// Publisher prepares and optionally submits a module release for publication.
	Publisher interface {
		Publish(context.Context, modulepublish.Options) (*modulepublish.Result, error)
	}
)
