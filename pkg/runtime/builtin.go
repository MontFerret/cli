package runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/MontFerret/cli/v2/pkg/logger"
	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/logging"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

var version = "unknown"

const DefaultRuntime = "builtin"
const DefaultBrowser = "http://127.0.0.1:9222"

type Builtin struct {
	opts   Options
	engine *ferret.Engine
	logger *logger.Logger
}

func NewBuiltin(opts Options) (Runtime, error) {
	mods, err := newModules(opts)

	if err != nil {
		return nil, fmt.Errorf("initialize modules: %w", err)
	}

	log, err := logger.New(opts.Logger)

	if err != nil {
		return nil, fmt.Errorf("initialize logger: %w", err)
	}

	engineOpts := []ferret.Option{
		ferret.WithModules(mods...),
	}

	if log.Output() != nil {
		engineOpts = append(
			engineOpts,
			ferret.WithLog(log.Output()),
			ferret.WithLogLevel(logging.LogLevel(opts.Logger.Level)),
		)
	}

	engine, err := ferret.New(engineOpts...)

	if err != nil {
		_ = log.Close()
		return nil, fmt.Errorf("initialize engine: %w", err)
	}

	return &Builtin{
		opts:   opts,
		engine: engine,
		logger: log,
	}, nil
}

func (rt *Builtin) Version(_ context.Context) (string, error) {
	return version, nil
}

func (rt *Builtin) Run(ctx context.Context, query *source.Source, params map[string]any) (io.ReadCloser, error) {
	res, err := rt.engine.Run(ctx, query, ferret.WithSessionParams(params))

	if err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewBuffer(res.Content)), nil
}

func (rt *Builtin) RunArtifact(ctx context.Context, data []byte, params map[string]any) (io.ReadCloser, error) {
	plan, err := rt.engine.Load(data)

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = plan.Close()
	}()

	session, err := plan.NewSession(ctx, ferret.WithSessionParams(params))

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

func (rt *Builtin) Close() error {
	if rt == nil {
		return nil
	}

	var err error

	if rt.engine != nil {
		err = errors.Join(err, rt.engine.Close())
	}

	if rt.logger != nil {
		err = errors.Join(err, rt.logger.Close())
	}

	return err
}
