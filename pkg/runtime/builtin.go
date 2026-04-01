package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/runtime"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

var version = "unknown"

const DefaultRuntime = "builtin"
const DefaultBrowser = "http://127.0.0.1:9222"

type Builtin struct {
	opts   Options
	engine *ferret.Engine
}

func NewBuiltin(opts Options) (Runtime, error) {
	mods, err := newModules(opts)

	if err != nil {
		return nil, fmt.Errorf("initialize modules: %w", err)
	}

	engine, err := ferret.New(ferret.WithModules(mods...))

	if err != nil {
		return nil, fmt.Errorf("initialize engine: %w", err)
	}

	return &Builtin{
		opts:   opts,
		engine: engine,
	}, nil
}

func (rt *Builtin) Version(_ context.Context) (string, error) {
	return version, nil
}

func (rt *Builtin) Run(ctx context.Context, query *source.Source, params map[string]any) (io.ReadCloser, error) {
	parsedParams, err := runtime.NewParamsFrom(params)

	if err != nil {
		return nil, err
	}

	res, err := rt.engine.Run(ctx, query, ferret.WithSessionParams(parsedParams))

	if err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewBuffer(res.Content)), nil
}

func (rt *Builtin) RunArtifact(ctx context.Context, data []byte, params map[string]any) (io.ReadCloser, error) {
	parsedParams, err := runtime.NewParamsFrom(params)

	if err != nil {
		return nil, err
	}

	plan, err := rt.engine.Load(data)

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = plan.Close()
	}()

	session, err := plan.NewSession(ctx, ferret.WithSessionParams(parsedParams))

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = session.Close()
	}()

	res, err := session.Run(ctx)

	if err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewBuffer(res.Content)), nil
}
