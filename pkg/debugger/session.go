package debugger

import (
	"context"

	apidebug "github.com/MontFerret/api/debugger"
	"github.com/MontFerret/api/source"
)

type (
	// Session is the portable debugger contract consumed by the CLI. Run closes the
	// session when the interactive loop exits. Operations receive the caller's
	// context, which must be non-nil, and preserve cancellation behavior.
	Session interface {
		Start(context.Context) (*apidebug.Event, error)
		Continue(context.Context) (*apidebug.Event, error)
		StepIn(context.Context) (*apidebug.Event, error)
		StepOver(context.Context) (*apidebug.Event, error)
		StepOut(context.Context) (*apidebug.Event, error)
		Pause(context.Context) error
		SetBreakpointAt(context.Context, source.Location, apidebug.BreakpointOptions) (apidebug.Breakpoint, error)
		DeleteBreakpoint(context.Context, apidebug.BreakpointID) error
		Breakpoints(context.Context) ([]apidebug.Breakpoint, error)
		Frames(context.Context) ([]apidebug.Frame, error)
		Locals(context.Context) ([]apidebug.Variable, error)
		Evaluate(context.Context, string) (apidebug.Value, error)
		Close() error
	}

	LineReader interface {
		Readline() (string, error)
	}
)
