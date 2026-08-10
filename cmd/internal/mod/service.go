package mod

import (
	"context"

	"github.com/MontFerret/cli/v2/pkg/module/discovery"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

// Service isolates the module command group from lifecycle implementation details.
type Service interface {
	Search(context.Context, string) ([]discovery.SearchResult, error)
	Info(context.Context, string) (*discovery.ModuleInfo, error)
	Install(context.Context, install.Options) (*install.Result, error)
	Create(context.Context, scaffold.Options) (*scaffold.Result, error)
	Publish(context.Context, modulepublish.Options) (*modulepublish.Result, error)
}
