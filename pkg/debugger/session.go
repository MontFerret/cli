package debugger

import (
	"context"

	"github.com/MontFerret/ferret/v2"
)

type (
	// Session is the core debugger contract consumed by the CLI. Run closes the
	// session when the interactive loop exits. Operations receive the caller's
	// context, which must be non-nil, and preserve core cancellation behavior.
	Session interface {
		Start(context.Context) (*ferret.DebugEvent, error)
		Continue(context.Context) (*ferret.DebugEvent, error)
		StepIn(context.Context) (*ferret.DebugEvent, error)
		StepOver(context.Context) (*ferret.DebugEvent, error)
		StepOut(context.Context) (*ferret.DebugEvent, error)
		Pause(context.Context) error
		SetBreakpointAt(context.Context, ferret.DebugSourceLocation, ferret.DebugBreakpointOptions) (ferret.DebugBreakpoint, error)
		DeleteBreakpoint(context.Context, ferret.DebugBreakpointID) error
		Breakpoints(context.Context) ([]ferret.DebugBreakpoint, error)
		Frames(context.Context) ([]ferret.DebugFrame, error)
		Locals(context.Context) ([]ferret.DebugVariable, error)
		Evaluate(context.Context, string) (ferret.DebugValue, error)
		Close() error
	}

	LineReader interface {
		Readline() (string, error)
	}
)
