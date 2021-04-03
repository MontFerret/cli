package browser

import (
	"context"

	"github.com/ziflex/waitfor/pkg/runner"
)

type Browser interface {
	Open(ctx context.Context) (uint64, error)

	Close(ctx context.Context, pid uint64) error
}

func Open(ctx context.Context, opts Options) (uint64, error) {
	b := New(opts)

	pid, err := b.Open(ctx)

	if err != nil {
		return 0, err
	}

	if opts.Detach {
		return pid, Wait(ctx, opts)
	}

	return pid, nil
}

func Wait(ctx context.Context, opts Options) error {
	return runner.Test(ctx, []string{
		opts.ToURL(),
	}, runner.WithAttempts(10))
}

func Close(ctx context.Context, opts Options, pid uint64) error {
	b := New(opts)

	return b.Close(ctx, pid)
}
