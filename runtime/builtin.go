package runtime

import (
	"bytes"
	"context"
	"io"

	"github.com/MontFerret/contrib/modules/html"
	"github.com/MontFerret/contrib/modules/html/drivers/cdp"
	"github.com/MontFerret/contrib/modules/html/drivers/http"
	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/file"
	"github.com/MontFerret/ferret/v2/pkg/runtime"
)

var version = "unknown"

const DefaultRuntime = "builtin"
const DefaultBrowser = "http://127.0.0.1:9222"

type Builtin struct {
	opts   Options
	engine *ferret.Engine
}

func NewBuiltin(opts Options) (Runtime, error) {
	htmlmod, err := html.New(
		html.WithDefaultDriver(http.NewDriver(opts.ToInMemory()...)),
		html.WithDrivers(
			cdp.NewDriver(opts.ToCDP()...),
		),
	)

	engine, err := ferret.New(
		ferret.WithModules(htmlmod),
	)

	if err != nil {
		return nil, err
	}

	return &Builtin{
		opts:   opts,
		engine: engine,
	}, nil
}

func (rt *Builtin) Version(_ context.Context) (string, error) {
	return version, nil
}

func (rt *Builtin) Run(ctx context.Context, query *file.Source, params map[string]any) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	parsedParams, err := runtime.NewParamsFrom(params)

	if err != nil {
		return nil, err
	}

	res, err := rt.engine.Run(ctx, query, ferret.WithSessionParams(parsedParams))

	if err != nil {
		return nil, err
	}

	buf.Write(res.Content)

	return buf, nil
}
