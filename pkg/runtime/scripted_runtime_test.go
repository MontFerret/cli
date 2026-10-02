package runtime

import (
	"context"

	"github.com/MontFerret/api"
)

type scriptedRuntime struct {
	api.Runtime
	output   *api.Output
	err      error
	closeErr error
	closed   int
}

func (r *scriptedRuntime) Version(ctx context.Context) (api.Version, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	return "scripted-runtime", nil
}

func (r *scriptedRuntime) Run(context.Context, api.Source, ...api.SessionOption) (*api.Output, error) {
	return r.output, r.err
}

func (r *scriptedRuntime) Close() error {
	r.closed++

	return r.closeErr
}
