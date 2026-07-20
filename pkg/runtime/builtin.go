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
	ferretnet "github.com/MontFerret/ferret/v2/pkg/net"
	ferrethttp "github.com/MontFerret/ferret/v2/pkg/net/http"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

var version = "unknown"

const DefaultRuntime = "builtin"
const DefaultBrowser = "http://127.0.0.1:9222"

type Builtin struct {
	opts    Options
	engine  *ferret.Engine
	logger  *logger.Logger
	network ferretnet.Network
}

func NewBuiltin(opts Options) (Runtime, error) {
	return newBuiltin(opts)
}

func newBuiltin(opts Options) (*Builtin, error) {
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

	var network ferretnet.Network

	if len(opts.HTTPPolicy) > 0 {
		client, err := ferrethttp.New(opts.HTTPPolicy...)
		if err != nil {
			_ = log.Close()
			return nil, fmt.Errorf("initialize HTTP policy: %w", err)
		}

		network, err = ferretnet.New(ferretnet.WithHTTPClient(client))
		if err != nil {
			if closer, ok := client.(ferrethttp.IdleConnectionCloser); ok {
				closer.CloseIdleConnections()
			}

			_ = log.Close()
			return nil, fmt.Errorf("initialize network: %w", err)
		}

		engineOpts = append(engineOpts, ferret.WithNetwork(network))
	}

	engine, err := ferret.New(engineOpts...)

	if err != nil {
		if network != nil {
			ferretnet.CloseIdleNetworkConnections(network)
		}

		_ = log.Close()
		return nil, fmt.Errorf("initialize engine: %w", err)
	}

	return &Builtin{
		opts:    opts,
		engine:  engine,
		logger:  log,
		network: network,
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

	if rt.network != nil {
		ferretnet.CloseIdleNetworkConnections(rt.network)
	}

	if rt.logger != nil {
		err = errors.Join(err, rt.logger.Close())
	}

	return err
}
