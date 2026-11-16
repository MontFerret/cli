package debugger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/chzyer/readline"

	apidebug "github.com/MontFerret/api/debugger"
	"github.com/MontFerret/api/result"
	"github.com/MontFerret/api/source"
	"github.com/MontFerret/wire/client"
	"github.com/MontFerret/wire/pkg/failure"
)

func TestRunDispatchesCommandsAndClosesSession(t *testing.T) {
	ctx := t.Context()
	src := source.New("demo.fql", "LET x = 1\nRETURN x")
	session := &fakeSession{
		startEvent:    debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 3}),
		continueEvent: debugEvent(apidebug.ReasonPause, "demo.fql", 2, source.Span{Start: 10, End: 16}),
		locals:        []apidebug.Variable{{Name: "x", Value: apidebug.Value{Display: "1"}}},
		frames:        []apidebug.Frame{{Name: "<main>", Location: debugSourceLocation("demo.fql", 1, 0)}},
		evaluation:    apidebug.Value{Display: "2"},
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

	if err := Run(ctx, session, src, input, &out); err != nil {
		t.Fatal(err)
	}

	for _, command := range []CommandName{CommandBreak, CommandDelete, CommandBreakpoints, CommandPause, CommandWhere, CommandLocals} {
		if got := session.contexts[command]; got != ctx {
			t.Fatalf("command %v received context %v, want caller context", command, got)
		}
	}

	if session.startCalls != 1 || session.pauseCalls != 1 || session.continueCalls != 1 || session.stepCalls != 1 ||
		session.nextCalls != 1 || session.outCalls != 1 || session.closeCalls != 1 {
		t.Fatalf("unexpected session calls: %#v", session)
	}
	if session.breakpointLocation != debugSourceLocation("demo.fql", 2, 1) {
		t.Fatalf("unexpected breakpoint location: %#v", session.breakpointLocation)
	}
	if session.breakpointOptions.BindingMode != apidebug.BreakpointBindExact {
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
		startEvent:  debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		evaluateErr: errors.New("expression is not supported by the safe debugger evaluator"),
	}
	var out bytes.Buffer

	err := Run(context.Background(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
		results: []lineResult{
			{line: "wat"},
			{line: "delete 99"},
			{line: "print LENGTH([1])"},
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
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected %q in %q", expected, got)
		}
	}
}

func TestRunReportsBreakpointListErrorsAndContinues(t *testing.T) {
	for _, test := range []struct {
		name        string
		breakpoints []apidebug.Breakpoint
	}{
		{name: "empty result"},
		{name: "partial result", breakpoints: []apidebug.Breakpoint{{ID: 1}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			session := &fakeSession{
				startEvent:     debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
				breakpoints:    test.breakpoints,
				breakpointsErr: context.Canceled,
			}
			var out bytes.Buffer

			err := Run(t.Context(), session, source.New("demo.fql", "RETURN 1"), &fakeLineReader{
				results: []lineResult{{line: "breakpoints"}, {line: "locals"}, {line: "q"}},
			}, &out)
			if err != nil {
				t.Fatal(err)
			}

			if session.breakpointsCalls != 1 || session.localsCalls != 1 || session.closeCalls != 1 {
				t.Fatalf("unexpected session calls: %#v", session)
			}

			got := out.String()
			if !strings.Contains(got, "Breakpoints error: context canceled") {
				t.Fatalf("missing breakpoint-list error in %q", got)
			}

			if strings.Contains(got, "No breakpoints.") || strings.Contains(got, "Requested") {
				t.Fatalf("rendered breakpoint list after failure: %q", got)
			}
		})
	}
}

func TestRunEOFAndCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	session := &fakeSession{
		startEvent: debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
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
				startEvent:    debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
				continueEvent: debugEvent(apidebug.ReasonStep, "demo.fql", 2, source.Span{Start: 9, End: 10}),
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
				startEvent:    debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
				continueEvent: debugEvent(apidebug.ReasonStep, "demo.fql", 1, source.Span{Start: 0, End: 1}),
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
		startEvent:    debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		continueEvent: debugEvent(apidebug.ReasonStep, "demo.fql", 1, source.Span{Start: 0, End: 1}),
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
		startEvent:    debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		continueEvent: debugEvent(apidebug.ReasonCompleted, "", 0, source.Span{}),
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
		startEvent:    debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		continueEvent: debugEvent(apidebug.ReasonTerminated, "", 0, source.Span{}),
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
		reason  apidebug.Reason
		message string
	}{
		{name: "completed", reason: apidebug.ReasonCompleted, message: "Program has completed. Start a new debug session to continue debugging."},
		{name: "terminated", reason: apidebug.ReasonTerminated, message: "Program has terminated. Start a new debug session to continue debugging."},
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
		startEvent:    debugEvent(apidebug.ReasonRuntimeError, "demo.fql", 1, source.Span{Start: 0, End: 1}),
		continueEvent: debugEvent(apidebug.ReasonStep, "demo.fql", 1, source.Span{Start: 0, End: 1}),
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

func TestRunRequiresExecutionEvent(t *testing.T) {
	for _, command := range []string{"start", "continue", "step", "next", "out"} {
		for _, opErr := range []error{nil, errors.New("result unavailable")} {
			t.Run(command+"/"+fmt.Sprint(opErr), func(t *testing.T) {
				session := &fakeSession{
					startEvent:  debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{}),
					continueErr: opErr,
				}
				if command == "start" {
					session.startEvent, session.startErr = nil, opErr
				}

				input := &fakeLineReader{results: []lineResult{{line: command}, {line: "where"}, {line: "q"}}}
				err := Run(t.Context(), session, source.New("demo.fql", "RETURN 1"), input, io.Discard)
				if err == nil || session.framesCalls != 0 || session.closeCalls != 1 {
					t.Fatalf("error=%v frames=%d closes=%d", err, session.framesCalls, session.closeCalls)
				}

				if opErr != nil && !errors.Is(err, opErr) {
					t.Fatalf("lost operation error: %v", err)
				}

				if opErr == nil && !strings.Contains(err.Error(), "no event") {
					t.Fatalf("missing protocol error: %v", err)
				}
			})
		}
	}
}

func TestRunStoppedEventWithSessionLoss(t *testing.T) {
	session := &fakeSession{
		startEvent:    debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{}),
		continueEvent: debugEvent(apidebug.ReasonStep, "demo.fql", 2, source.Span{}),
		continueErr:   client.ErrClosed,
	}
	input := &fakeLineReader{results: []lineResult{{line: "step"}, {line: "where"}}}
	var out bytes.Buffer
	err := Run(t.Context(), session, source.New("demo.fql", "LET x = 1\nRETURN x"), input, &out)
	if !errors.Is(err, client.ErrClosed) || session.framesCalls != 0 || !strings.Contains(out.String(), "Paused after step") {
		t.Fatalf("error=%v frames=%d transcript=%s", err, session.framesCalls, &out)
	}
}

func debugEvent(reason apidebug.Reason, file string, line int, span source.Span) *apidebug.Event {
	return &apidebug.Event{
		Reason:   reason,
		Location: debugLocation(file, line, 1, span),
	}
}

func TestRunTerminalOperationFailure(t *testing.T) {
	opErr, closeErr := errors.New("operation failed"), errors.New("session close failed")
	for _, command := range []string{"start", "continue", "step", "next", "out"} {
		for _, reason := range []apidebug.Reason{apidebug.ReasonCompleted, apidebug.ReasonTerminated} {
			t.Run(command+"/"+string(reason), func(t *testing.T) {
				entry := debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{})
				terminal := &apidebug.Event{Reason: reason, Error: opErr, Output: &result.Content{Data: []byte("42")}}
				want := "Program terminated.\nResult:\n42\n"
				if reason == apidebug.ReasonCompleted {
					want = "Program completed.\nResult:\n42\n"
				}

				session := &fakeSession{startEvent: entry, continueEvent: terminal, continueErr: opErr, closeErr: closeErr}
				if command == "start" {
					session.startEvent, session.startErr = terminal, opErr
				}

				input := &fakeLineReader{results: []lineResult{{line: command}, {line: "help"}}}
				var out bytes.Buffer
				err := Run(t.Context(), session, source.New("demo.fql", "RETURN 42"), input, &out)
				if !errors.Is(err, opErr) || !errors.Is(err, closeErr) || session.closeCalls != 1 {
					t.Fatalf("error=%v closes=%d", err, session.closeCalls)
				}

				if !strings.Contains(out.String(), want) || strings.Contains(out.String(), "Commands:") || strings.Contains(out.String(), opErr.Error()) {
					t.Fatalf("terminal transcript: %s", &out)
				}
			})
		}
	}
}

func TestRunStoppedEventWithErrorStaysInspectable(t *testing.T) {
	session := &fakeSession{
		startEvent:    debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{}),
		continueEvent: debugEvent(apidebug.ReasonStep, "demo.fql", 2, source.Span{}),
		continueErr:   &failure.Failure{Category: failure.CategoryInternalRuntime, Message: "command warning"},
	}
	var out bytes.Buffer
	input := &fakeLineReader{results: []lineResult{{line: "step"}, {line: "where"}, {line: "q"}}}
	err := Run(t.Context(), session, source.New("demo.fql", "LET x = 1\nRETURN x"), input, &out)
	if err != nil || session.framesCalls != 1 || session.closeCalls != 1 {
		t.Fatalf("error=%v session=%+v", err, session)
	}

	stop, diagnostic := strings.Index(out.String(), "Paused after step"), strings.Index(out.String(), "Debugger error: command warning")
	if stop < 0 || diagnostic <= stop {
		t.Fatalf("missing stop or error: %s", &out)
	}
}

func TestRunLostSessionDuringInspection(t *testing.T) {
	lost := &client.Error{Category: failure.CategoryDebugSessionNotFound, Message: "session was lost"}
	session := &fakeSession{
		startEvent:     debugEvent(apidebug.ReasonEntry, "demo.fql", 1, source.Span{}),
		breakpointsErr: lost,
	}
	var out bytes.Buffer
	input := &fakeLineReader{results: []lineResult{{line: "bl"}, {line: "continue"}}}
	err := Run(t.Context(), session, source.New("demo.fql", "RETURN 1"), input, &out)
	if !errors.Is(err, lost) || session.continueCalls != 0 || session.closeCalls != 1 {
		t.Fatalf("err=%v session=%+v", err, session)
	}

	if strings.Contains(out.String(), "session was lost") {
		t.Fatalf("prompt also printed the returned fatal error: %s", &out)
	}
}
