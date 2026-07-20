package runtime

import (
	"testing"

	"github.com/MontFerret/cli/v2/pkg/logger"
)

func BenchmarkBuiltinLifecycle(b *testing.B) {
	opts := NewDefaultOptions()
	opts.Logger.LogOutput = logger.OutputNone

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rt, err := newBuiltin(opts)
		if err != nil {
			b.Fatal(err)
		}

		if err := rt.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
