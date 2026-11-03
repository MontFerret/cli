package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/MontFerret/api"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
)

func BenchmarkWireConnection(b *testing.B) {
	host := wirehost.New(b)
	opts := Options{Type: "wire", Endpoint: host.Endpoint}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		resources, err := New(context.Background(), opts)
		if err != nil {
			b.Fatal(err)
		}

		if err := resources.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWireSource(b *testing.B) {
	host := wirehost.New(b)
	resources, err := New(context.Background(), Options{Type: "wire", Endpoint: host.Endpoint})
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = resources.Close() })
	src := api.NewSource("benchmark.fql", "RETURN DEMO::IDENTITY(@value)")
	params := api.WithParams(map[string]any{"value": 42})
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		out, err := resources.Runtime.Run(context.Background(), src, params)
		if err != nil {
			b.Fatal(err)
		}
		if err := out.Consume(context.Background(), func(context.Context, []byte) error { return nil }); err != nil {
			b.Fatal(err)
		}
		if err := out.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWireVersion(b *testing.B) {
	host := wirehost.New(b)
	resources, err := New(context.Background(), Options{Type: "wire", Endpoint: host.Endpoint})
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = resources.Close() })
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := resources.Runtime.Version(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWireSourceLarge(b *testing.B) {
	host := wirehost.New(b)
	resources, err := New(b.Context(), Options{Type: "wire", Endpoint: host.Endpoint})
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = resources.Close() })
	src := api.NewSource("benchmark.fql", "RETURN DEMO::IDENTITY(@value)")
	params := api.WithParam("value", strings.Repeat("x", 1<<20))
	b.SetBytes(1 << 20)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		out, err := resources.Runtime.Run(b.Context(), src, params)
		if err != nil {
			b.Fatal(err)
		}
		if err := out.Consume(b.Context(), func(context.Context, []byte) error { return nil }); err != nil {
			b.Fatal(err)
		}
		if err := out.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
