package runtime

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MontFerret/api"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestWireLargeOutput(t *testing.T) {
	host := wirehost.New(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	output, err := Run(ctx, Options{Type: "wire", Endpoint: host.Endpoint}, source.NewAnonymous("RETURN DEMO::LARGE()"), nil)
	if err != nil {
		t.Fatal(err)
	}

	defer output.Close()
	var total, chunks int
	err = output.Consume(ctx, func(_ context.Context, data []byte) error {
		for i, value := range data {
			position := total + i
			want := byte('x')
			if position == 0 || position == (8<<20)+1 {
				want = '"'
			}

			if value != want {
				t.Fatalf("byte %d = %q, want %q", position, value, want)
			}
		}

		total += len(data)
		chunks++
		return nil
	})
	if err != nil || total != (8<<20)+2 || chunks < 2 {
		t.Fatalf("bytes=%d chunks=%d error=%v", total, chunks, err)
	}

	if err := output.Close(); err != nil {
		t.Fatal(err)
	}

	if err := host.WaitForClientsClosed(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPStreamsBeforeResponseCompletes(t *testing.T) {
	prefix := make(chan struct{})
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = io.WriteString(w, "prefix")
		w.(http.Flusher).Flush()
		select {
		case <-prefix:
			_, _ = io.WriteString(w, "suffix")
		case <-r.Context().Done():
		}
	}))
	defer worker.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	output, err := Run(ctx, Options{Type: worker.URL}, source.NewAnonymous("RETURN 42"), nil)
	if err != nil {
		t.Fatal(err)
	}

	defer output.Close()
	if output.Metadata().LengthKnown {
		t.Fatal("chunked HTTP response length should be unknown")
	}

	var value strings.Builder
	signaled := false
	err = output.Consume(ctx, func(_ context.Context, data []byte) error {
		value.Write(data)
		if !signaled && strings.Contains(value.String(), "prefix") {
			close(prefix)
			signaled = true
		}

		return nil
	})
	if err != nil || value.String() != "prefixsuffix" {
		t.Fatalf("value=%q error=%v", value.String(), err)
	}
}

func TestWireStreamsBeforeHostedDeliveryCompletes(t *testing.T) {
	prefix := make(chan struct{})
	host := wirehost.NewWithRun(t, func(ctx context.Context, _ api.Source, _ ...api.SessionOption) (api.Output, error) {
		return newManagedOutput(ctx, api.Metadata{}, func(ctx context.Context, consumer api.Consumer) error {
			if err := consumer(ctx, []byte("prefix")); err != nil {
				return err
			}

			select {
			case <-prefix:
			case <-ctx.Done():
				return ctx.Err()
			}

			return consumer(ctx, []byte("suffix"))
		}, nil, func() error { return nil }), nil
	})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	output, err := Run(ctx, Options{Type: "wire", Endpoint: host.Endpoint}, source.NewAnonymous("RETURN 42"), nil)
	if err != nil {
		t.Fatal(err)
	}

	defer output.Close()
	var value strings.Builder
	signaled := false
	err = output.Consume(ctx, func(_ context.Context, data []byte) error {
		value.Write(data)
		if !signaled && strings.Contains(value.String(), "prefix") {
			close(prefix)
			signaled = true
		}

		return nil
	})
	if err != nil || value.String() != "prefixsuffix" {
		t.Fatalf("value=%q error=%v", value.String(), err)
	}
}

func TestHTTPAbandonedOutputCancelsRequest(t *testing.T) {
	finished := make(chan struct{})
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(finished)
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer worker.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	output, err := Run(ctx, Options{Type: worker.URL}, source.NewAnonymous("RETURN 42"), nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := output.Close(); err != nil {
		t.Fatal(err)
	}

	waitSignal(t, finished)
	if err := output.Consume(ctx, func(context.Context, []byte) error { return nil }); !errors.Is(err, api.ErrOutputClosed) {
		t.Fatalf("abandoned output consumption: %v", err)
	}
}
