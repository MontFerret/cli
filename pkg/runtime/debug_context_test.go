package runtime

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestDebugSessionCanceledOperationsPreserveBreakpoints(t *testing.T) {
	ctx := t.Context()

	session, err := NewDebugSession(ctx, NewDefaultOptions(), nil, source.New("debug.fql", "LET value = 1\nRETURN value"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})

	location := ferret.DebugSourceLocation{Position: ferret.Position{Line: 2}}
	options := ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindExact}

	breakpoint, err := session.SetBreakpointAt(ctx, location, options)
	if err != nil {
		t.Fatal(err)
	}

	if !breakpoint.Bound {
		t.Fatal("expected a bound breakpoint")
	}

	if _, err := session.Start(ctx); err != nil {
		t.Fatal(err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()

	for _, test := range []struct {
		name string
		run  func(context.Context) error
	}{
		{name: "pause", run: session.Pause},
		{name: "add", run: func(ctx context.Context) error {
			_, err := session.SetBreakpointAt(ctx, location, options)
			return err
		}},
		{name: "delete", run: func(ctx context.Context) error {
			return session.DeleteBreakpoint(ctx, breakpoint.ID)
		}},
		{name: "list", run: func(ctx context.Context) error {
			_, err := session.Breakpoints(ctx)
			return err
		}},
		{name: "frames", run: func(ctx context.Context) error {
			_, err := session.Frames(ctx)
			return err
		}},
		{name: "locals", run: func(ctx context.Context) error {
			_, err := session.Locals(ctx)
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(canceled); !errors.Is(err, context.Canceled) {
				t.Fatalf("expected cancellation, got %v", err)
			}

			breakpoints, err := session.Breakpoints(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if !slices.Equal(breakpoints, []ferret.DebugBreakpoint{breakpoint}) {
				t.Fatalf("canceled operation changed breakpoints: %#v", breakpoints)
			}
		})
	}
}
