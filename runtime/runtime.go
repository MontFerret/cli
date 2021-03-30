package runtime

import (
	"context"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type Runtime interface {
	Version(ctx context.Context) (string, error)

	Run(ctx context.Context, query string, params map[string]interface{}) ([]byte, error)
}

func New(opts Options) (Runtime, error) {
	name := strings.ReplaceAll(strings.ToLower(opts.Type), " ", "")

	if name == DefaultRuntime {
		return NewBuiltin(opts), nil
	}

	u, err := url.Parse(name)

	if err != nil {
		return nil, errors.Wrap(err, "parse url")
	}

	return NewRemote(*u, opts), nil
}
