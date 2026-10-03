package runtime

import (
	"context"
	"io"
	"testing"

	"github.com/MontFerret/cli/v2/pkg/logger"
	"github.com/MontFerret/ferret/v2/pkg/bytecode/artifact"
	"github.com/MontFerret/ferret/v2/pkg/compiler"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func BenchmarkBuiltinLifecycle(b *testing.B) {
	opts := NewDefaultOptions()
	opts.Logger.LogOutput = logger.OutputNone

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rt, err := New(context.Background(), opts)
		if err != nil {
			b.Fatal(err)
		}

		if err := rt.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuiltinArtifactLifecycle(b *testing.B) {
	c, err := compiler.New()
	if err != nil {
		b.Fatal(err)
	}

	program, err := c.Compile(b.Context(), source.New("benchmark.fql", "RETURN @value"))
	if err != nil {
		b.Fatal(err)
	}

	data, err := artifact.Marshal(program)
	if err != nil {
		b.Fatal(err)
	}

	opts := NewDefaultOptions()
	opts.Logger.LogOutput = logger.OutputNone
	params := map[string]any{"value": 42}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		out, err := RunArtifact(b.Context(), opts, data, params)
		if err != nil {
			b.Fatal(err)
		}

		if _, err := io.Copy(io.Discard, out); err != nil {
			b.Fatal(err)
		}

		if err := out.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuiltinSource(b *testing.B) {
	opts := NewDefaultOptions()
	opts.Logger.LogOutput = logger.OutputNone

	rt, err := New(context.Background(), opts)
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = rt.Close() })
	src := source.New("benchmark.fql", "RETURN @value")
	params := map[string]any{"value": 42}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		out, err := RunSource(context.Background(), rt, src, params)
		if err != nil {
			b.Fatal(err)
		}

		if _, err := io.Copy(io.Discard, out); err != nil {
			b.Fatal(err)
		}

		if err := out.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
