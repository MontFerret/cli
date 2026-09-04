package debugger

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/encoding"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestRendererEventPauseAndCompletion(t *testing.T) {
	src := source.New("demo.fql", "LET x = 1\nRETURN x")
	var out bytes.Buffer
	renderer := NewRenderer(&out, src)

	renderer.Event(&ferret.DebugEvent{
		Reason:           ferret.DebugReasonBreakpoint,
		Location:         debugLocation("demo.fql", 2, 1, source.Span{Start: 10, End: 16}),
		HitBreakpointIDs: []ferret.DebugBreakpointID{3},
	})

	got := out.String()
	for _, expected := range []string{"Paused on breakpoint 3 at demo.fql:2:1", "2 | RETURN x", "~"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected %q in %q", expected, got)
		}
	}

	out.Reset()
	renderer.Event(&ferret.DebugEvent{
		Reason: ferret.DebugReasonCompleted,
		Output: &encoding.Output{Content: []byte(`["Ada"]`)},
	})
	if got := out.String(); !strings.Contains(got, "Program completed.\nResult:\n[\"Ada\"]") {
		t.Fatalf("unexpected completion output: %q", got)
	}
}

func TestRendererCollectionsAndErrors(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, source.Source{})

	renderer.Breakpoints([]ferret.DebugBreakpoint{
		{
			ID:                1,
			RequestedLocation: debugSourceLocation("demo.fql", 4, 3),
			Location:          debugLocation("demo.fql", 7, 5, source.Span{}),
			BindingMode:       ferret.DebugBreakpointBindNextExecutableInFile,
			Bound:             true,
		},
		{
			ID:                2,
			RequestedLocation: debugSourceLocation("other.fql", 9, 0),
			BindingMode:       ferret.DebugBreakpointBindExact,
		},
	})
	renderer.Frames([]ferret.DebugFrame{{Name: "normalize", Location: debugSourceLocation("demo.fql", 7, 3)}})
	renderer.Locals([]ferret.DebugVariable{
		{Name: "user", Value: ferret.DebugValue{Display: `{"name": "Ada"}`}},
		{Name: "@limit", Param: true, Value: ferret.DebugValue{Display: "10"}},
	})
	renderer.Error("Evaluation error", errors.New("expected expression"))
	renderer.Event(&ferret.DebugEvent{Reason: ferret.DebugReasonRuntimeError, Error: errors.New("division by zero")})

	got := out.String()
	for _, expected := range []string{
		"Requested", "Bound", "Mode", "State",
		"demo.fql:4:3", "demo.fql:7:5", "other.fql:9", "next-file", "exact", "bound", "unbound",
		"#0 normalize at demo.fql:7:3",
		"Locals:", `user = {"name": "Ada"}`, "Params:", "@limit = 10",
		"Evaluation error: expected expression",
		"Paused on runtime error.", "division by zero",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected %q in %q", expected, got)
		}
	}
}

func TestRendererBreakpointHitsAndSetMessages(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, source.Source{})

	renderer.Event(&ferret.DebugEvent{
		Reason:           ferret.DebugReasonBreakpoint,
		Location:         debugLocation("demo.fql", 12, 4, source.Span{}),
		HitBreakpointIDs: []ferret.DebugBreakpointID{3, 7},
	})
	renderer.Event(&ferret.DebugEvent{
		Reason:   ferret.DebugReasonPause,
		Location: debugLocation("demo.fql", 13, 1, source.Span{}),
	})
	renderer.Event(&ferret.DebugEvent{
		Reason:   ferret.DebugReasonStep,
		Location: debugLocation("demo.fql", 14, 2, source.Span{}),
	})
	renderer.BreakpointSet(ferret.DebugBreakpoint{
		ID:                8,
		RequestedLocation: debugSourceLocation("demo.fql", 10, 2),
		Location:          debugLocation("demo.fql", 12, 4, source.Span{}),
		BindingMode:       ferret.DebugBreakpointBindNextExecutableInFunction,
		Bound:             true,
	})
	renderer.BreakpointSet(ferret.DebugBreakpoint{
		ID:                9,
		RequestedLocation: debugSourceLocation("demo.fql", 20, 0),
		BindingMode:       ferret.DebugBreakpointBindExact,
	})

	got := out.String()
	for _, expected := range []string{
		"Paused on breakpoints 3, 7 at demo.fql:12:4",
		"Paused on pause request at demo.fql:13:1",
		"Paused after step at demo.fql:14:2",
		"Breakpoint 8 set at demo.fql:12:4 (requested demo.fql:10:2, in-function).",
		"Breakpoint 9 could not be bound at demo.fql:20 (exact).",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected %q in %q", expected, got)
		}
	}
}

func TestRendererEmptyCollections(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, source.Source{})

	renderer.Breakpoints(nil)
	renderer.Frames(nil)
	renderer.Locals(nil)

	got := out.String()
	for _, expected := range []string{"No breakpoints.", "No stack frames available.", "No local variables available."} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected %q in %q", expected, got)
		}
	}
}

func TestRendererHelpIncludesAliasesAndPauseBehavior(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, source.Source{})

	renderer.Help()

	got := out.String()
	for _, expected := range []string{
		"break, b <location>",
		"delete, d <id>",
		"breakpoints, bp, bl",
		"where, w, bt",
		"locals, l",
		"print, p, eval, e <expr>",
		"Request pause after the next resume",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected %q in %q", expected, got)
		}
	}
}

func debugSourceLocation(file string, line, column int) ferret.DebugSourceLocation {
	return ferret.DebugSourceLocation{
		File: file,
		Position: ferret.Position{
			Line:   line,
			Column: column,
		},
	}
}

func debugLocation(file string, line, column int, span source.Span) ferret.DebugLocation {
	return ferret.DebugLocation{
		Location: debugSourceLocation(file, line, column),
		Span:     span,
	}
}
