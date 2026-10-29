package output

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MontFerret/api"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/wire/client"
)

// BenchmarkWireOutputDelivery measures the real command writer over loopback
// Wire. Native evaluation/encoding, runtime selection and startup are excluded.
func BenchmarkWireOutputDelivery(b *testing.B) {
	for _, test := range []struct {
		name                string
		size, callbackBytes int
	}{
		{"64B/whole", 64, 64},
		{"1MiB/whole", 1 << 20, 1 << 20},
		{"8MiB/whole", 8 << 20, 8 << 20},
		{"1MiB/256B", 1 << 20, 256},
	} {
		b.Run(test.name, func(b *testing.B) {
			payload := wireDeliveryPayload(test.size)
			remote := newWireDeliveryRuntime(b, payload, test.callbackBytes)
			writer := &wireDeliveryWriter{}
			b.SetBytes(int64(test.size))
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				writer.bytes = 0

				out, err := remote.Run(b.Context(), api.Source{})
				if err != nil {
					b.Fatal(err)
				}

				resultErr, failure := Write(b.Context(), writer, out, false)
				if resultErr != nil || failure != nil || writer.bytes != test.size {
					b.Fatalf("received=%d want=%d result=%v failure=%v", writer.bytes, test.size, resultErr, failure)
				}
			}
		})
	}
}

func wireDeliveryPayload(size int) []byte {
	payload := make([]byte, size)
	for index := range payload {
		payload[index] = byte(index % 251)
	}

	return payload
}

func newWireDeliveryRuntime(t testing.TB, payload []byte, callbackBytes int) api.Runtime {
	t.Helper()
	host := wirehost.NewWithRun(t, func(context.Context, api.Source, ...api.SessionOption) (api.Output, error) {
		return &wireDeliveryFixture{payload: payload, callbackBytes: callbackBytes}, nil
	})
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	remote, err := client.New(ctx, strings.TrimPrefix(host.Endpoint, "tcp://"), client.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := remote.Close(); err != nil {
			t.Errorf("close Wire runtime: %v", err)
		}
	})

	return remote
}

func TestWireDeliveryWriter(t *testing.T) {
	payload := wireDeliveryPayload(65549)
	remote := newWireDeliveryRuntime(t, payload, 1009)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	writer := &wireDeliveryWriter{expected: payload}
	for range 3 {
		writer.bytes = 0

		out, err := remote.Run(ctx, api.Source{})
		if err != nil {
			t.Fatal(err)
		}

		resultErr, failure := Write(ctx, writer, out, false)
		if resultErr != nil || failure != nil || writer.bytes != len(payload) {
			t.Fatalf("received=%d result=%v failure=%v", writer.bytes, resultErr, failure)
		}
	}
}
