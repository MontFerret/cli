package wirehost

import (
	"context"

	"github.com/MontFerret/api"
)

type runRuntime struct {
	api.Runtime
	run func(context.Context, api.Source, ...api.SessionOption) (api.Output, error)
}

func (r *runRuntime) Run(ctx context.Context, src api.Source, options ...api.SessionOption) (api.Output, error) {
	return r.run(ctx, src, options...)
}
