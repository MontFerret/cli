package runtime

import (
	"context"
	"errors"
	"sync"

	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

// DebugSession owns a core debugger session and all runtime resources created
// for it.
type DebugSession struct {
	*ferret.DebugSession

	runtime   *Builtin
	plan      *ferret.Plan
	closeErr  error
	closeOnce sync.Once
}

// NewDebugSession compiles source for debugging and creates a retained-state
// debugger session through the builtin Ferret runtime.
func NewDebugSession(ctx context.Context, opts Options, params map[string]any, src *source.Source) (*DebugSession, error) {
	opts = NormalizeOptions(opts)

	if err := ValidateOptions(opts); err != nil {
		return nil, err
	}

	if !IsBuiltinType(opts.Type) {
		return nil, ErrDebugRequiresBuiltinRuntime
	}

	rt, err := newBuiltin(opts)
	if err != nil {
		return nil, err
	}

	plan, err := rt.engine.CompileDebug(ctx, src)
	if err != nil {
		return nil, errors.Join(err, rt.Close())
	}

	session, err := plan.NewDebugSession(ctx, ferret.WithSessionParams(params))
	if err != nil {
		return nil, errors.Join(err, plan.Close(), rt.Close())
	}

	return &DebugSession{
		DebugSession: session,
		runtime:      rt,
		plan:         plan,
	}, nil
}

// Close releases the debugger session, plan, engine, and logger.
func (s *DebugSession) Close() error {
	if s == nil {
		return nil
	}

	s.closeOnce.Do(func() {
		if s.DebugSession != nil {
			s.closeErr = errors.Join(s.closeErr, s.DebugSession.Close())
		}
		if s.plan != nil {
			s.closeErr = errors.Join(s.closeErr, s.plan.Close())
		}
		if s.runtime != nil {
			s.closeErr = errors.Join(s.closeErr, s.runtime.Close())
		}
	})

	return s.closeErr
}
