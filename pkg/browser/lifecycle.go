package browser

import (
	"context"
	"net/url"
	"strconv"

	"github.com/MontFerret/ferret/v2/pkg/runtime"

	cliruntime "github.com/MontFerret/cli/pkg/runtime"
)

// EnsureBrowser opens a browser if runtime options require it.
// Returns a cleanup function that must be deferred.
func EnsureBrowser(ctx context.Context, rtOpts cliruntime.Options, brOpts Options) (func(), error) {
	noop := func() {}

	if !rtOpts.WithBrowser {
		return noop, nil
	}

	brOpts.Detach = true
	brOpts.Headless = rtOpts.WithHeadlessBrowser

	if rtOpts.BrowserAddress != "" {
		u, err := url.Parse(rtOpts.BrowserAddress)

		if err != nil {
			return noop, runtime.Error(err, "invalid browser address")
		}

		if u.Port() != "" {
			p, err := strconv.ParseUint(u.Port(), 10, 64)

			if err != nil {
				return noop, err
			}

			brOpts.Port = p
		}
	}

	pid, err := Open(ctx, brOpts)

	if err != nil {
		return noop, err
	}

	return func() {
		Close(ctx, brOpts, pid)
	}, nil
}
