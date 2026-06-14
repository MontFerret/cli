package runtime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestNewDebugSessionIntegration(t *testing.T) {
	ctx := context.Background()
	session, err := NewDebugSession(
		ctx,
		NewDefaultOptions(),
		map[string]any{"limit": 2},
		source.New("debug.fql", "LET x = 1\nVAR y = @limit\n\ny = y + x\nRETURN y"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	nextBreakpoint, err := session.SetBreakpointAt(
		ferret.DebugSourceLocation{Line: 3},
		ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindNextExecutableInFile},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !nextBreakpoint.Bound || nextBreakpoint.Line != 4 {
		t.Fatalf("unexpected next breakpoint: %#v", nextBreakpoint)
	}
	exactBreakpoint, err := session.SetBreakpointAt(
		ferret.DebugSourceLocation{Line: 3},
		ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindExact},
	)
	if err != nil {
		t.Fatal(err)
	}
	if exactBreakpoint.Bound {
		t.Fatalf("expected exact breakpoint to remain unbound: %#v", exactBreakpoint)
	}
	functionBreakpoint, err := session.SetBreakpointAt(
		ferret.DebugSourceLocation{Line: 3},
		ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindNextExecutableInFunction},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !functionBreakpoint.Bound || functionBreakpoint.Line != 4 {
		t.Fatalf("unexpected in-function breakpoint: %#v", functionBreakpoint)
	}

	event, err := session.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if event.Reason != ferret.DebugReasonEntry {
		t.Fatalf("unexpected start event: %#v", event)
	}

	event, err = session.Continue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if event.Reason != ferret.DebugReasonBreakpoint || event.Location.Line != 4 ||
		len(event.HitBreakpointIDs) != 2 ||
		event.HitBreakpointIDs[0] != nextBreakpoint.ID ||
		event.HitBreakpointIDs[1] != functionBreakpoint.ID {
		t.Fatalf("unexpected breakpoint event: %#v", event)
	}

	value, err := session.Evaluate(ctx, "x + y")
	if err != nil {
		t.Fatal(err)
	}
	if value.Display != "3" {
		t.Fatalf("unexpected evaluation: %#v", value)
	}
	if _, err := session.Evaluate(ctx, "LENGTH([1])"); err == nil {
		t.Fatal("expected unsafe expression to be rejected")
	}

	event, err = session.Continue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if event.Reason != ferret.DebugReasonCompleted || event.Output == nil || string(event.Output.Content) != "3" {
		t.Fatalf("unexpected completion event: %#v", event)
	}

	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNewDebugSessionDocumentSurvivesResume(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>debug</body></html>"))
	}))
	defer server.Close()

	ctx := context.Background()
	session, err := NewDebugSession(
		ctx,
		NewDefaultOptions(),
		map[string]any{"url": server.URL},
		source.New("document.fql", "LET doc = DOCUMENT(@url)\nRETURN doc != NONE"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	event, err := session.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if event.Reason != ferret.DebugReasonEntry || event.Location.Line != 1 {
		t.Fatalf("unexpected start event: %#v", event)
	}

	event, err = session.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if event.Reason != ferret.DebugReasonStep || event.Location.Line != 2 {
		t.Fatalf("expected next statement after DOCUMENT, got %#v", event)
	}

	event, err = session.Continue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if event.Reason != ferret.DebugReasonCompleted || event.Output == nil || string(event.Output.Content) != "true" {
		t.Fatalf("unexpected completion event: %#v", event)
	}
}

func TestNewDebugSessionRejectsRemoteRuntime(t *testing.T) {
	_, err := NewDebugSession(
		context.Background(),
		Options{Type: "https://worker.example"},
		nil,
		source.NewAnonymous("RETURN 1"),
	)
	if !errors.Is(err, ErrDebugRequiresBuiltinRuntime) {
		t.Fatalf("expected builtin runtime error, got %v", err)
	}
	var typed *DebugRequiresBuiltinRuntimeError
	if !errors.As(err, &typed) {
		t.Fatalf("expected typed builtin runtime error, got %T", err)
	}
	if got := err.Error(); got != "debug currently supports only the builtin runtime" {
		t.Fatalf("unexpected builtin runtime error: %q", got)
	}
}
