package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MontFerret/api"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestRunArtifact_RemoteRuntimeRejected(t *testing.T) {
	_, err := RunArtifact(context.Background(), Options{Type: "https://worker.example"}, []byte("FBC2"), nil)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrArtifactRequiresBuiltinRuntime) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRejectsCanceledConstruction(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	for _, kind := range []string{DefaultRuntime, "wire", "https://worker.example"} {
		t.Run(kind, func(t *testing.T) {
			opts := Options{Type: kind}
			if kind == "wire" {
				opts.Endpoint = "tcp://127.0.0.1:123"
			}

			resources, err := New(ctx, opts)
			if resources != nil || !errors.Is(err, context.Canceled) {
				t.Fatalf("constructed resources after cancellation: %v, %v", resources, err)
			}
		})
	}
}

func TestResourcesOwnsBuiltinEngine(t *testing.T) {
	resources, err := New(t.Context(), NewDefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = resources.Close() })

	// The borrowed UAPI adapter's Close leaves the native engine usable.
	// Resources.Close must still release that engine afterward.
	if err := resources.Runtime.Close(); err != nil {
		t.Fatal(err)
	}

	output, err := resources.Runtime.Run(t.Context(), api.NewSource("owned.fql", "RETURN 42"))
	if err != nil || output == nil || string(output.Content) != "42" {
		t.Fatalf("borrowed adapter became unusable: output=%v error=%v", output, err)
	}

	nativePlan, err := resources.builtin.engine.Compile(t.Context(), source.New("survivor.fql", "RETURN 42"))
	if err != nil {
		t.Fatalf("adapter closed the borrowed engine: %v", err)
	}

	if err := nativePlan.Close(); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := resources.Close(); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := resources.builtin.engine.Compile(t.Context(), source.New("closed.fql", "RETURN 42")); err == nil || !strings.Contains(err.Error(), "engine is closed") {
		t.Fatalf("resource owner left its native engine usable: %v", err)
	}
}
