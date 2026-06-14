package debugger

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/chzyer/readline"

	"github.com/MontFerret/ferret/v2"
	ferruntime "github.com/MontFerret/ferret/v2/pkg/runtime"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestRunDispatchesCommandsAndClosesSession(t *testing.T) {
	src := source.New("demo.fql", "LET x = 1\nRETURN x")
	session := &fakeSession{
		startEvent:    debugEvent(ferret.DebugReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 3}),
		continueEvent: debugEvent(ferret.DebugReasonPause, "demo.fql", 2, source.Span{Start: 10, End: 16}),
		locals:        []ferret.DebugVariable{{Name: "x", Value: ferret.DebugValue{Display: "1"}}},
		frames:        []ferret.DebugFrame{{Name: "<main>", Location: ferret.DebugLocation{File: "demo.fql", Line: 1}}},
		evaluation:    ferret.DebugValue{Display: "2"},
	}
	input := &fakeLineReader{results: []lineResult{
		{line: ""},
		{err: readline.ErrInterrupt},
		{line: "break --exact 2:1"},
		{line: "breakpoints"},
		{line: "pause"},
		{line: "locals"},
		{line: "print x + 1"},
		{line: "where"},
		{line: "continue"},
		{line: "step"},
		{line: "next"},
		{line: "out"},
		{line: "help"},
		{line: "delete 1"},
		{line: "q"},
	}}
	var out bytes.Buffer

	if err := Run(context.Background(), session, src, input, &out); err != nil {
		t.Fatal(err)
	}

	if session.startCalls != 1 || session.pauseCalls != 1 || session.continueCalls != 1 || session.stepCalls != 1 ||
		session.nextCalls != 1 || session.outCalls != 1 || session.closeCalls != 1 {
		t.Fatalf("unexpected session calls: %#v", session)
	}
	if session.breakpointLocation != (ferret.DebugSourceLocation{File: "demo.fql", Line: 2, Column: 1}) {
		t.Fatalf("unexpected breakpoint location: %#v", session.breakpointLocation)
	}
	if session.breakpointOptions.BindingMode != ferret.DebugBreakpointBindExact {
		t.Fatalf("unexpected breakpoint options: %#v", session.breakpointOptions)
	}
	if session.expression != "x + 1" {
		t.Fatalf("unexpected expression: %q", session.expression)
	}

	got := out.String()
	for _, expected := range []string{
		"Ferret debugger started.",
		"Paused at demo.fql:1:1",
		"Breakpoint 1 set at demo.fql:2:1 (exact).",
		"Pause requested.",
		"Locals:",
		"x = 1",
		"\n2\n",
		"#0 <main> at demo.fql:1",
		"Paused on pause request at demo.fql:2:1",
		"Commands:",
		"Breakpoint 1 deleted.",
		"Debug session terminated.",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected %q in %q", expected, got)
		}
	}
}

func TestRunReportsCommandErrorsAndContinues(t *testing.T) {
	session := &fakeSession{
		startEvent:  debugEvent(ferret.DebugReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		continueErr: errors.New("cannot resume while debug session is completed"),
		evaluateErr: errors.New("expression is not supported by the safe debugger evaluator"),
	}
	var out bytes.Buffer

	err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
		results: []lineResult{
			{line: "wat"},
			{line: "delete 99"},
			{line: "print LENGTH([1])"},
			{line: "continue"},
			{line: "q"},
		},
	}, &out)
	if err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, expected := range []string{
		"unknown command: wat",
		"Unknown breakpoint: 99",
		"Evaluation error: expression is not supported by the safe debugger evaluator",
		"Debugger error: cannot resume while debug session is completed",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected %q in %q", expected, got)
		}
	}
}

func TestRunEOFAndCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	session := &fakeSession{
		startEvent: debugEvent(ferret.DebugReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		closeErr:   closeErr,
	}
	var out bytes.Buffer

	err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
		results: []lineResult{{err: io.EOF}},
	}, &out)
	if !errors.Is(err, closeErr) {
		t.Fatalf("expected close error, got %v", err)
	}
	if !strings.Contains(out.String(), "Debug session terminated.") {
		t.Fatalf("unexpected output: %q", out.String())
	}
	if session.closeCalls != 1 {
		t.Fatalf("expected one close call, got %d", session.closeCalls)
	}
}

func TestRunRepeatsPreviousResumeCommandOnEmptyInput(t *testing.T) {
	tests := []struct {
		name    string
		command string
		calls   func(*fakeSession) int
	}{
		{name: "next", command: "next", calls: func(session *fakeSession) int { return session.nextCalls }},
		{name: "step alias", command: "s", calls: func(session *fakeSession) int { return session.stepCalls }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := &fakeSession{
				startEvent:    debugEvent(ferret.DebugReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
				continueEvent: debugEvent(ferret.DebugReasonStep, "demo.fql", 2, source.Span{Start: 9, End: 10}),
			}

			err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
				results: []lineResult{{line: test.command}, {line: ""}, {line: "q"}},
			}, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if got := test.calls(session); got != 2 {
				t.Fatalf("expected repeated %s call, got %d calls", test.command, got)
			}
		})
	}
}

func TestRunEmptyInputDoesNotRepeatNonRepeatableCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
		check   func(*testing.T, *fakeSession)
	}{
		{
			name:    "where clears previous resume and is not repeated",
			command: "where",
			check: func(t *testing.T, session *fakeSession) {
				t.Helper()
				if session.nextCalls != 1 || session.framesCalls != 1 {
					t.Fatalf("unexpected session calls: %#v", session)
				}
			},
		},
		{
			name:    "delete clears previous resume and is not repeated",
			command: "delete 99",
			check: func(t *testing.T, session *fakeSession) {
				t.Helper()
				if session.nextCalls != 1 || session.deleteCalls != 1 {
					t.Fatalf("unexpected session calls: %#v", session)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := &fakeSession{
				startEvent:    debugEvent(ferret.DebugReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
				continueEvent: debugEvent(ferret.DebugReasonStep, "demo.fql", 1, source.Span{Start: 0, End: 1}),
			}

			err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
				results: []lineResult{{line: "next"}, {line: test.command}, {line: ""}, {line: "q"}},
			}, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			test.check(t, session)
		})
	}
}

func TestRunParseErrorsDoNotClearRepeatCommand(t *testing.T) {
	session := &fakeSession{
		startEvent:    debugEvent(ferret.DebugReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		continueEvent: debugEvent(ferret.DebugReasonStep, "demo.fql", 1, source.Span{Start: 0, End: 1}),
	}

	err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
		results: []lineResult{{line: "next"}, {line: "wat"}, {line: ""}, {line: "q"}},
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if session.nextCalls != 2 {
		t.Fatalf("expected parse error to preserve repeat command, got %d next calls", session.nextCalls)
	}
}

func TestRunBlocksCommandsAfterCompletionAndKeepsSafeCommandsAvailable(t *testing.T) {
	session := &fakeSession{
		startEvent:    debugEvent(ferret.DebugReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		continueEvent: debugEvent(ferret.DebugReasonCompleted, "", 0, source.Span{}),
	}
	var out bytes.Buffer

	err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
		results: []lineResult{
			{line: "continue"},
			{line: ""},
			{line: "step"},
			{line: "next"},
			{line: "out"},
			{line: "pause"},
			{line: "where"},
			{line: "locals"},
			{line: "print 1"},
			{line: "help"},
			{line: "break 1"},
			{line: "breakpoints"},
			{line: "delete 1"},
			{line: "q"},
		},
	}, &out)
	if err != nil {
		t.Fatal(err)
	}

	if session.continueCalls != 1 || session.stepCalls != 0 || session.nextCalls != 0 || session.outCalls != 0 ||
		session.pauseCalls != 0 || session.framesCalls != 0 || session.localsCalls != 0 || session.evaluateCalls != 0 {
		t.Fatalf("terminal commands reached session: %#v", session)
	}
	if session.setBreakpointCalls != 1 || session.breakpointsCalls != 1 || session.deleteCalls != 1 || session.closeCalls != 1 {
		t.Fatalf("safe commands were not dispatched: %#v", session)
	}

	got := out.String()
	for _, expected := range []string{
		"Program completed.",
		"Program has completed. Start a new debug session to continue debugging.",
		"Commands:",
		"Breakpoint 1 set",
		"Breakpoint 1 deleted.",
		"Debug session terminated.",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected %q in %q", expected, got)
		}
	}
}

func TestRunBlocksResumeCommandsAfterTermination(t *testing.T) {
	session := &fakeSession{
		startEvent:    debugEvent(ferret.DebugReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		continueEvent: debugEvent(ferret.DebugReasonTerminated, "", 0, source.Span{}),
	}
	var out bytes.Buffer

	err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
		results: []lineResult{
			{line: "continue"},
			{line: ""},
			{line: "step"},
			{line: "next"},
			{line: "out"},
			{line: "pause"},
			{line: "where"},
			{line: "locals"},
			{line: "print 1"},
			{line: "q"},
		},
	}, &out)
	if err != nil {
		t.Fatal(err)
	}

	if session.continueCalls != 1 || session.stepCalls != 0 || session.nextCalls != 0 || session.outCalls != 0 ||
		session.pauseCalls != 0 || session.framesCalls != 0 || session.localsCalls != 0 || session.evaluateCalls != 0 {
		t.Fatalf("terminal commands reached session: %#v", session)
	}
	if got := out.String(); !strings.Contains(got, "Program has terminated. Start a new debug session to continue debugging.") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunTracksTerminalInitialEvent(t *testing.T) {
	tests := []struct {
		name    string
		reason  ferret.DebugReason
		message string
	}{
		{name: "completed", reason: ferret.DebugReasonCompleted, message: "Program has completed. Start a new debug session to continue debugging."},
		{name: "terminated", reason: ferret.DebugReasonTerminated, message: "Program has terminated. Start a new debug session to continue debugging."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := &fakeSession{startEvent: debugEvent(test.reason, "", 0, source.Span{})}
			var out bytes.Buffer

			err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
				results: []lineResult{{line: "continue"}, {line: "q"}},
			}, &out)
			if err != nil {
				t.Fatal(err)
			}
			if session.continueCalls != 0 {
				t.Fatalf("continue reached terminal session: %#v", session)
			}
			if got := out.String(); !strings.Contains(got, test.message) {
				t.Fatalf("unexpected output: %q", got)
			}
		})
	}
}

func TestRunKeepsRuntimeErrorEventActive(t *testing.T) {
	session := &fakeSession{
		startEvent:    debugEvent(ferret.DebugReasonRuntimeError, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		continueEvent: debugEvent(ferret.DebugReasonStep, "demo.fql", 1, source.Span{Start: 0, End: 1}),
	}

	err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
		results: []lineResult{{line: "next"}, {line: "q"}},
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if session.nextCalls != 1 {
		t.Fatalf("expected next after runtime error, got %d calls", session.nextCalls)
	}
}

func TestRunResumeErrorPreservesState(t *testing.T) {
	session := &fakeSession{
		startEvent:    debugEvent(ferret.DebugReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		continueEvent: debugEvent(ferret.DebugReasonCompleted, "", 0, source.Span{}),
		continueErr:   errors.New("resume failed"),
	}

	err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
		results: []lineResult{{line: "continue"}, {line: "next"}, {line: "q"}},
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if session.continueCalls != 1 || session.nextCalls != 1 {
		t.Fatalf("expected resume error to preserve active state: %#v", session)
	}
}

type lineResult struct {
	line string
	err  error
}

type fakeLineReader struct {
	results []lineResult
	index   int
}

func (f *fakeLineReader) Readline() (string, error) {
	if f.index >= len(f.results) {
		return "", io.EOF
	}
	result := f.results[f.index]
	f.index++
	return result.line, result.err
}

type fakeSession struct {
	startEvent         *ferret.DebugEvent
	continueEvent      *ferret.DebugEvent
	locals             []ferret.DebugVariable
	frames             []ferret.DebugFrame
	breakpoints        []ferret.DebugBreakpoint
	evaluation         ferret.DebugValue
	continueErr        error
	evaluateErr        error
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

func (f *fakeSession) Step(context.Context) (*ferret.DebugEvent, error) {
	f.stepCalls++
	return f.continueEvent, nil
}

func (f *fakeSession) Next(context.Context) (*ferret.DebugEvent, error) {
	f.nextCalls++
	return f.continueEvent, nil
}

func (f *fakeSession) Out(context.Context) (*ferret.DebugEvent, error) {
	f.outCalls++
	return f.continueEvent, nil
}

func (f *fakeSession) Pause() error {
	f.pauseCalls++
	return nil
}

func (f *fakeSession) SetBreakpointAt(location ferret.DebugSourceLocation, options ferret.DebugBreakpointOptions) (ferret.DebugBreakpoint, error) {
	f.setBreakpointCalls++
	f.breakpointLocation = location
	f.breakpointOptions = options
	breakpoint := ferret.DebugBreakpoint{
		ID:              ferret.DebugBreakpointID(len(f.breakpoints) + 1),
		File:            location.File,
		RequestedLine:   location.Line,
		RequestedColumn: location.Column,
		Line:            location.Line,
		Column:          location.Column,
		BindingMode:     options.BindingMode,
		Bound:           true,
	}
	f.breakpoints = append(f.breakpoints, breakpoint)
	return breakpoint, nil
}

func (f *fakeSession) DeleteBreakpoint(id ferret.DebugBreakpointID) error {
	f.deleteCalls++
	for i, breakpoint := range f.breakpoints {
		if breakpoint.ID == id {
			f.breakpoints = append(f.breakpoints[:i], f.breakpoints[i+1:]...)
			return nil
		}
	}
	return ferruntime.Errorf(ferruntime.ErrNotFound, "breakpoint %d", id)
}

func (f *fakeSession) Breakpoints() []ferret.DebugBreakpoint {
	f.breakpointsCalls++
	return f.breakpoints
}

func (f *fakeSession) Frames() ([]ferret.DebugFrame, error) {
	f.framesCalls++
	return f.frames, nil
}

func (f *fakeSession) Locals() ([]ferret.DebugVariable, error) {
	f.localsCalls++
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

func debugEvent(reason ferret.DebugReason, file string, line int, span source.Span) *ferret.DebugEvent {
	return &ferret.DebugEvent{
		Reason:   reason,
		Location: ferret.DebugLocation{File: file, Line: line, Column: 1, Span: span},
	}
}
