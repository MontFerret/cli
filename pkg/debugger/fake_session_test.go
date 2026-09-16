package debugger

import (
	"context"

	"github.com/MontFerret/ferret/v2"
	ferruntime "github.com/MontFerret/ferret/v2/pkg/runtime"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

type fakeSession struct {
	startEvent         *ferret.DebugEvent
	continueEvent      *ferret.DebugEvent
	locals             []ferret.DebugVariable
	frames             []ferret.DebugFrame
	breakpoints        []ferret.DebugBreakpoint
	evaluation         ferret.DebugValue
	continueErr        error
	evaluateErr        error
	breakpointsErr     error
	contexts           map[CommandName]context.Context
	expression         string
	closeErr           error
	breakpointLocation ferret.DebugSourceLocation
	breakpointOptions  ferret.DebugBreakpointOptions
	startCalls         int
	continueCalls      int
	stepCalls          int
	nextCalls          int
	outCalls           int
	pauseCalls         int
	setBreakpointCalls int
	deleteCalls        int
	breakpointsCalls   int
	framesCalls        int
	localsCalls        int
	evaluateCalls      int
	closeCalls         int
}

func (f *fakeSession) Start(context.Context) (*ferret.DebugEvent, error) {
	f.startCalls++
	return f.startEvent, nil
}

func (f *fakeSession) Continue(context.Context) (*ferret.DebugEvent, error) {
	f.continueCalls++
	return f.continueEvent, f.continueErr
}

func (f *fakeSession) StepIn(context.Context) (*ferret.DebugEvent, error) {
	f.stepCalls++
	return f.continueEvent, nil
}

func (f *fakeSession) StepOver(context.Context) (*ferret.DebugEvent, error) {
	f.nextCalls++
	return f.continueEvent, nil
}

func (f *fakeSession) StepOut(context.Context) (*ferret.DebugEvent, error) {
	f.outCalls++
	return f.continueEvent, nil
}

func (f *fakeSession) Pause(ctx context.Context) error {
	f.pauseCalls++
	f.recordContext(ctx, CommandPause)

	return nil
}

func (f *fakeSession) SetBreakpointAt(ctx context.Context, location ferret.DebugSourceLocation, options ferret.DebugBreakpointOptions) (ferret.DebugBreakpoint, error) {
	f.setBreakpointCalls++
	f.recordContext(ctx, CommandBreak)
	f.breakpointLocation = location
	f.breakpointOptions = options
	breakpoint := ferret.DebugBreakpoint{
		ID:                ferret.DebugBreakpointID(len(f.breakpoints) + 1),
		RequestedLocation: location,
		Location:          debugLocation(location.SourceName, location.Line, location.Column, source.Span{}),
		BindingMode:       options.BindingMode,
		Bound:             true,
	}
	f.breakpoints = append(f.breakpoints, breakpoint)
	return breakpoint, nil
}

func (f *fakeSession) DeleteBreakpoint(ctx context.Context, id ferret.DebugBreakpointID) error {
	f.deleteCalls++
	f.recordContext(ctx, CommandDelete)

	for i, breakpoint := range f.breakpoints {
		if breakpoint.ID == id {
			f.breakpoints = append(f.breakpoints[:i], f.breakpoints[i+1:]...)
			return nil
		}
	}

	return ferruntime.Errorf(ferruntime.ErrNotFound, "breakpoint %d", id)
}

func (f *fakeSession) Breakpoints(ctx context.Context) ([]ferret.DebugBreakpoint, error) {
	f.breakpointsCalls++
	f.recordContext(ctx, CommandBreakpoints)

	return f.breakpoints, f.breakpointsErr
}

func (f *fakeSession) Frames(ctx context.Context) ([]ferret.DebugFrame, error) {
	f.framesCalls++
	f.recordContext(ctx, CommandWhere)

	return f.frames, nil
}

func (f *fakeSession) Locals(ctx context.Context) ([]ferret.DebugVariable, error) {
	f.localsCalls++
	f.recordContext(ctx, CommandLocals)

	return f.locals, nil
}

func (f *fakeSession) Evaluate(_ context.Context, expression string) (ferret.DebugValue, error) {
	f.evaluateCalls++
	f.expression = expression
	return f.evaluation, f.evaluateErr
}

func (f *fakeSession) Close() error {
	f.closeCalls++
	return f.closeErr
}

func (f *fakeSession) recordContext(ctx context.Context, command CommandName) {
	if f.contexts == nil {
		f.contexts = make(map[CommandName]context.Context)
	}

	f.contexts[command] = ctx
}
