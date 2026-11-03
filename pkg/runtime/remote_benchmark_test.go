package runtime

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MontFerret/ferret/v2/pkg/source"
)

func BenchmarkHTTPSource(b *testing.B) {
	for _, size := range []int{2, 1 << 20} {
		name := map[int]string{2: "small", 1 << 20: "1MiB"}[size]
		b.Run(name, func(b *testing.B) {
			content := bytes.Repeat([]byte("x"), size)
			worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(content)
			}))
			defer worker.Close()

			resources, err := New(b.Context(), Options{Type: worker.URL})
			if err != nil {
				b.Fatal(err)
			}

			b.Cleanup(func() { _ = resources.Close() })
			src := source.New("benchmark.fql", "RETURN @value")
			params := map[string]any{"value": 42}
			b.SetBytes(int64(size))
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				out, err := RunSource(b.Context(), resources, src, params)
				if err != nil {
					b.Fatal(err)
				}

				copyErr := out.Consume(b.Context(), func(context.Context, []byte) error { return nil })
				closeErr := out.Close()
				if copyErr != nil || closeErr != nil {
					b.Fatalf("copy=%v close=%v", copyErr, closeErr)
				}
			}
		})
	}
}
