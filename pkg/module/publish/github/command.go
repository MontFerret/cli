package github

import (
	"context"
)

type commandRunnerFunc func(context.Context, string, ...string) ([]byte, error)

func (f commandRunnerFunc) Run(ctx context.Context, name string, arguments ...string) ([]byte, error) {
	return f(ctx, name, arguments...)
}
