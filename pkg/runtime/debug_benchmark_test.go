package runtime

import (
	"testing"

	"github.com/MontFerret/ferret/v2/pkg/source"
)

func BenchmarkBuiltinDebugLifecycle(b *testing.B) {
	src := source.New("debug.fql", "RETURN @value")
	params := map[string]any{"value": 42}
	opts := NewDefaultOptions()
	b.ReportAllocs()

	for b.Loop() {
		s, err := NewDebugSession(b.Context(), opts, params, src)
		if err != nil {
			b.Fatal(err)
		}

		if _, err := s.Start(b.Context()); err != nil {
			b.Fatal(err)
		}

		if _, err := s.Continue(b.Context()); err != nil {
			b.Fatal(err)
		}

		if err := s.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
