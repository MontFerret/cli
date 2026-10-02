package runtime

import (
	"context"
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
		resources, err := OpenSource(context.Background(), opts)
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
	resources, err := OpenSource(context.Background(), Options{Type: "wire", Endpoint: host.Endpoint})
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = resources.Close() })
	src := api.NewSource("benchmark.fql", "RETURN DEMO::IDENTITY(@value)")
	params := api.WithParams(map[string]any{"value": 42})
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := resources.Runtime.Run(context.Background(), src, params); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWireVersion(b *testing.B) {
	host := wirehost.New(b)
	resources, err := OpenSource(context.Background(), Options{Type: "wire", Endpoint: host.Endpoint})
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
