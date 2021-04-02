package browser

import (
	"context"

	"github.com/ziflex/waitfor/pkg/runner"
)

type Browser interface {
	Open(ctx context.Context) error

	Close(ctx context.Context) error
}

func Wait(ctx context.Context, opts Options) error {
	return runner.Test(ctx, []string{
		opts.ToURL(),
	}, runner.WithAttempts(10))
}
