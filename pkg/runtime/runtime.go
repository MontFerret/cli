package runtime

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/MontFerret/ferret/v2/pkg/file"
)

type Runtime interface {
	Version(ctx context.Context) (string, error)

	Run(ctx context.Context, query *file.Source, params map[string]any) (io.ReadCloser, error)
}

func New(opts Options) (Runtime, error) {
	name := strings.ReplaceAll(strings.ToLower(opts.Type), " ", "")

	if name == DefaultRuntime {
		return NewBuiltin(opts)
	}

	u, err := url.Parse(name)

	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	return NewRemote(*u, opts), nil
}

func Run(ctx context.Context, opts Options, query *file.Source, params map[string]any) (io.ReadCloser, error) {
	rt, err := New(opts)

	if err != nil {
		return nil, err
	}

	return rt.Run(ctx, query, params)
}
