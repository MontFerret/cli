package browser

import (
	"context"

	"github.com/ziflex/waitfor/pkg/runner"
)

type Browser interface {
	Open(ctx context.Context) error

	Close(ctx context.Context, pid string) error
}

func Open(ctx context.Context, opts Options) error {
	b := New(opts)

	if err := b.Open(ctx); err != nil {
		return err
	}

	if opts.Detach {
		return Wait(ctx, opts)
	}

	return nil
}

func Wait(ctx context.Context, opts Options) error {
	return runner.Test(ctx, []string{
		opts.ToURL(),
	}, runner.WithAttempts(10))
}

func Close(ctx context.Context, opts Options, pid string) error {
	b := New(opts)

	return b.Close(ctx, pid)
}
