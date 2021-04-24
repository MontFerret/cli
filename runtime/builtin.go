package runtime

import (
	"context"

	"github.com/MontFerret/ferret"
	"github.com/MontFerret/ferret/pkg/drivers"
	"github.com/MontFerret/ferret/pkg/drivers/cdp"
	"github.com/MontFerret/ferret/pkg/drivers/http"
	"github.com/MontFerret/ferret/pkg/runtime"
	"github.com/pkg/errors"
)

var version = "unknown"

const DefaultRuntime = "builtin"
const DefaultBrowser = "http://127.0.0.1:9222"

type Builtin struct {
	version  string
	opts     Options
	compiler *ferret.Instance
}

func NewBuiltin(opts Options) Runtime {
	rt := new(Builtin)
	rt.opts = opts
	rt.compiler = ferret.New()

	return rt
}

func (rt *Builtin) Version(_ context.Context) (string, error) {
	return version, nil
}

func (rt *Builtin) Run(ctx context.Context, query string, params map[string]interface{}) ([]byte, error) {
	program, err := rt.compiler.Compile(query)

	if err != nil {
		return nil, errors.Wrap(err, "compile query")
	}

	ctx = drivers.WithContext(
		ctx,
		http.NewDriver(rt.opts.ToInMemory()...),
		drivers.AsDefault(),
	)

	ctx = drivers.WithContext(
		ctx,
		cdp.NewDriver(rt.opts.ToCDP()...),
	)

	return program.Run(ctx, runtime.WithParams(params))
}
