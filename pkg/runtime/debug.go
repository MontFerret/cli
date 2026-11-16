package runtime

import (
	"context"
	"errors"
	"sync"

	"github.com/MontFerret/api"
	"github.com/MontFerret/api/debugger"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

// DebugSession owns a portable debugger session, its plan, and the selected runtime.
type DebugSession struct {
	debugger.Session

	plan      api.Plan
	runtime   *Resources
	ctx       context.Context
	stop      context.CancelFunc
	closeOnce sync.Once
	closeErr  error
}

// NewDebugSession compiles the local source on the selected runtime and binds the
// supplied parameters to a new session. The caller owns the returned session.
func NewDebugSession(ctx context.Context, opts Options, params map[string]any, src source.Source) (*DebugSession, error) {
	rt, err := New(ctx, opts)
	if err != nil {
		return nil, err
	}

	return newDebugSession(ctx, rt, opts.Endpoint, params, api.NewSource(src.Name(), src.Content()))
}

func newDebugSession(ctx context.Context, rt *Resources, endpoint string, params map[string]any, src api.Source) (*DebugSession, error) {
	ctx, stop := rt.CommandContext(ctx, endpoint)
	s := &DebugSession{ctx: ctx, stop: stop, runtime: rt}

	var err error
	s.plan, err = rt.Runtime.CompileDebug(ctx, src)
	if err != nil {
		return nil, errors.Join(err, s.Close())
	}

	s.Session, err = s.plan.NewDebugSession(ctx, api.WithParams(params))
	if err != nil {
		return nil, errors.Join(err, s.Close())
	}

	return s, nil
}

// Context bounds interactive input and execution by the caller's context and
// the lifetime of the selected transport. Pass it to Start and resume commands.
func (s *DebugSession) Context() context.Context {
	return s.ctx
}

// Close settles the session before releasing its plan and runtime. Repeated
// calls retain the same cleanup result.
func (s *DebugSession) Close() error {
	if s == nil {
		return nil
	}

	s.closeOnce.Do(func() {
		if s.Session != nil {
			s.closeErr = s.Session.Close()
		}

		if s.plan != nil {
			s.closeErr = errors.Join(s.closeErr, s.plan.Close())
		}

		if s.stop != nil {
			s.stop()
		}

		s.closeErr = errors.Join(s.closeErr, s.runtime.Close())
	})

	return s.closeErr
}
