package debugger

import (
	"context"

	"github.com/MontFerret/ferret/v2"
)

type (
	Session interface {
		Start(context.Context) (*ferret.DebugEvent, error)
		Continue(context.Context) (*ferret.DebugEvent, error)
		Step(context.Context) (*ferret.DebugEvent, error)
		Next(context.Context) (*ferret.DebugEvent, error)
		Out(context.Context) (*ferret.DebugEvent, error)
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
