package mod

import (
	"context"

	barnpublish "github.com/MontFerret/barn/pkg/publish"

	modulelifecycle "github.com/MontFerret/cli/v2/pkg/module"
)

// Service isolates the module command group from lifecycle implementation details.
type Service interface {
	Search(context.Context, string) ([]modulelifecycle.SearchResult, error)
	Info(context.Context, string) (*modulelifecycle.ModuleInfo, error)
	Create(context.Context, modulelifecycle.CreateOptions) (*modulelifecycle.CreateResult, error)
	Publish(context.Context, modulelifecycle.PublishOptions) (*barnpublish.Result, error)
}
