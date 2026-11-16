package debugger

import (
	"context"

	apidebug "github.com/MontFerret/api/debugger"
	"github.com/MontFerret/api/source"
	ferruntime "github.com/MontFerret/ferret/v2/pkg/runtime"
)

type fakeSession struct {
	startErr           error
	startEvent         *apidebug.Event
	continueEvent      *apidebug.Event
	locals             []apidebug.Variable
	frames             []apidebug.Frame
	breakpoints        []apidebug.Breakpoint
	evaluation         apidebug.Value
	continueErr        error
	evaluateErr        error
	breakpointsErr     error
	contexts           map[CommandName]context.Context
	expression         string
	closeErr           error
	breakpointLocation source.Location
	breakpointOptions  apidebug.BreakpointOptions
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

func (f *fakeSession) Start(context.Context) (*apidebug.Event, error) {
	f.startCalls++
	return f.startEvent, f.startErr
}

func (f *fakeSession) Continue(context.Context) (*apidebug.Event, error) {
	f.continueCalls++
	return f.continueEvent, f.continueErr
}

func (f *fakeSession) StepIn(context.Context) (*apidebug.Event, error) {
	f.stepCalls++
	return f.continueEvent, f.continueErr
}

func (f *fakeSession) StepOver(context.Context) (*apidebug.Event, error) {
	f.nextCalls++
	return f.continueEvent, f.continueErr
}

func (f *fakeSession) StepOut(context.Context) (*apidebug.Event, error) {
	f.outCalls++
	return f.continueEvent, f.continueErr
}

func (f *fakeSession) Pause(ctx context.Context) error {
	f.pauseCalls++
	f.recordContext(ctx, CommandPause)

	return nil
}

func (f *fakeSession) SetBreakpointAt(ctx context.Context, location source.Location, options apidebug.BreakpointOptions) (apidebug.Breakpoint, error) {
	f.setBreakpointCalls++
	f.recordContext(ctx, CommandBreak)
	f.breakpointLocation = location
	f.breakpointOptions = options
	breakpoint := apidebug.Breakpoint{
		ID:                apidebug.BreakpointID(len(f.breakpoints) + 1),
		RequestedLocation: location,
		Location:          debugLocation(location.SourceName, location.Line, location.Column, source.Span{}),
		BindingMode:       options.BindingMode,
		Bound:             true,
	}
	f.breakpoints = append(f.breakpoints, breakpoint)
	return breakpoint, nil
}

func (f *fakeSession) DeleteBreakpoint(ctx context.Context, id apidebug.BreakpointID) error {
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

func (f *fakeSession) Breakpoints(ctx context.Context) ([]apidebug.Breakpoint, error) {
	f.breakpointsCalls++
	f.recordContext(ctx, CommandBreakpoints)

	return f.breakpoints, f.breakpointsErr
}

func (f *fakeSession) Frames(ctx context.Context) ([]apidebug.Frame, error) {
	f.framesCalls++
	f.recordContext(ctx, CommandWhere)

	return f.frames, nil
}

func (f *fakeSession) Locals(ctx context.Context) ([]apidebug.Variable, error) {
	f.localsCalls++
	f.recordContext(ctx, CommandLocals)

	return f.locals, nil
}

func (f *fakeSession) Evaluate(_ context.Context, expression string) (apidebug.Value, error) {
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
