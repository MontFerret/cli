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
		Location:         ferret.DebugLocation{File: "demo.fql", Line: 2, Column: 1, Span: source.Span{Start: 10, End: 16}},
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
	renderer := NewRenderer(&out, nil)

	renderer.Breakpoints([]ferret.DebugBreakpoint{
		{ID: 1, File: "demo.fql", RequestedLine: 4, RequestedColumn: 3, Line: 7, Column: 5, BindingMode: ferret.DebugBreakpointBindNextExecutableInFile, Bound: true},
		{ID: 2, File: "other.fql", RequestedLine: 9, BindingMode: ferret.DebugBreakpointBindExact},
	})
	renderer.Frames([]ferret.DebugFrame{{Name: "normalize", Location: ferret.DebugLocation{File: "demo.fql", Line: 7, Column: 3}}})
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
	renderer := NewRenderer(&out, nil)

	renderer.Event(&ferret.DebugEvent{
		Reason:           ferret.DebugReasonBreakpoint,
		Location:         ferret.DebugLocation{File: "demo.fql", Line: 12, Column: 4},
		HitBreakpointIDs: []ferret.DebugBreakpointID{3, 7},
	})
	renderer.Event(&ferret.DebugEvent{
		Reason:   ferret.DebugReasonPause,
		Location: ferret.DebugLocation{File: "demo.fql", Line: 13, Column: 1},
	})
	renderer.Event(&ferret.DebugEvent{
		Reason:   ferret.DebugReasonStep,
		Location: ferret.DebugLocation{File: "demo.fql", Line: 14, Column: 2},
	})
	renderer.BreakpointSet(ferret.DebugBreakpoint{
		ID:              8,
		File:            "demo.fql",
		RequestedLine:   10,
		RequestedColumn: 2,
		Line:            12,
		Column:          4,
		BindingMode:     ferret.DebugBreakpointBindNextExecutableInFunction,
		Bound:           true,
	})
	renderer.BreakpointSet(ferret.DebugBreakpoint{
		ID:            9,
		File:          "demo.fql",
		RequestedLine: 20,
		BindingMode:   ferret.DebugBreakpointBindExact,
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
	renderer := NewRenderer(&out, nil)

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
