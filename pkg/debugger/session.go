package debugger

import (
	"context"

	"github.com/MontFerret/ferret/v2"
)

type (
	// Session is the core debugger contract consumed by the CLI. Run closes the
	// session when the interactive loop exits.
	Session interface {
		Start(context.Context) (*ferret.DebugEvent, error)
		Continue(context.Context) (*ferret.DebugEvent, error)
		StepIn(context.Context) (*ferret.DebugEvent, error)
		StepOver(context.Context) (*ferret.DebugEvent, error)
		StepOut(context.Context) (*ferret.DebugEvent, error)
		Pause() error
		SetBreakpointAt(ferret.DebugSourceLocation, ferret.DebugBreakpointOptions) (ferret.DebugBreakpoint, error)
		DeleteBreakpoint(ferret.DebugBreakpointID) error
		Breakpoints() []ferret.DebugBreakpoint
		Frames() ([]ferret.DebugFrame, error)
		Locals() ([]ferret.DebugVariable, error)
		Evaluate(context.Context, string) (ferret.DebugValue, error)
		Close() error
	}

	LineReader interface {
		Readline() (string, error)
	}
)
