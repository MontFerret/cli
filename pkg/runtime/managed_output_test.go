package runtime

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MontFerret/api"
	"github.com/MontFerret/cli/v2/internal/testutil/encoded"
)

func TestOwnedOutputCleanupAndAdmission(t *testing.T) {
	executionErr, outputErr, parentErr := errors.New("execution"), errors.New("output cleanup"), errors.New("parent cleanup")
	for _, abandon := range []bool{false, true} {
		inner := &encoded.Output{Chunks: [][]byte{[]byte("available")}, Err: executionErr, CloseErr: outputErr}
		closed := 0
		output := newOwnedOutput(t.Context(), inner, closeFunc(func() error {
			if inner.Closed != 1 {
				t.Error("parent closed before output")
			}

			closed++
			return parentErr
		}))
		if closed != 0 || inner.Closed != 0 {
			t.Fatal("resources released before consumption")
		}

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		for _, attempt := range []func() error{
			//lint:ignore SA1012 Exercise the required nil-context rejection without claiming output.
			func() error { return output.Consume(nil, func(context.Context, []byte) error { return nil }) },
			func() error { return output.Consume(t.Context(), nil) },
			func() error { return output.Consume(ctx, func(context.Context, []byte) error { return nil }) },
		} {
			if attempt() == nil || closed != 0 {
				t.Fatal("invalid consumption claimed output")
			}
		}

		if !abandon {
			content, err := output.Collect(t.Context())
			if content == nil || string(content.Data) != "available" {
				t.Fatalf("content=%v", content)
			}

			for _, cause := range []error{executionErr, outputErr, parentErr} {
				if !errors.Is(err, cause) {
					t.Fatalf("lost %v: %v", cause, err)
				}
			}
		}

		for range 2 {
			err := output.Close()
			if !errors.Is(err, outputErr) || !errors.Is(err, parentErr) || errors.Is(err, executionErr) {
				t.Fatalf("close=%v", err)
			}
		}

		if closed != 1 || inner.Closed != 1 {
			t.Fatalf("parent=%d output=%d", closed, inner.Closed)
		}

		if err := output.Consume(t.Context(), func(context.Context, []byte) error { return nil }); !errors.Is(err, api.ErrOutputClosed) {
			t.Fatalf("reuse=%v", err)
		}
	}
}

func TestManagedOutputConcurrentCloseWaitsForCallback(t *testing.T) {
	entered, release, stopping := make(chan struct{}), make(chan struct{}), make(chan struct{})
	releaseCallback := sync.OnceFunc(func() { close(release) })
	defer releaseCallback()
	var cleaned atomic.Int32
	output := newManagedOutput(t.Context(), api.Metadata{}, func(ctx context.Context, consumer api.Consumer) error {
		return consumer(ctx, []byte("borrowed"))
	}, nil, func() error { cleaned.Add(1); return nil })
	t.Cleanup(func() { _ = output.Close() })
	done := make(chan error, 1)
	go func() {
		done <- output.Consume(t.Context(), func(ctx context.Context, data []byte) error {
			close(entered)
			<-ctx.Done()
			close(stopping)
			<-release
			if string(data) != "borrowed" {
				t.Error("borrowed bytes invalidated")
			}

			return nil
		})
	}()
	waitSignal(t, entered)
	if err := output.Consume(t.Context(), func(context.Context, []byte) error { return nil }); !errors.Is(err, api.ErrOutputInUse) {
		t.Fatalf("competition=%v", err)
	}

	closed := make(chan error, 1)
	go func() { closed <- output.Close() }()
	waitSignal(t, stopping)
	if cleaned.Load() != 0 {
		t.Fatal("cleanup ran inside active callback")
	}

	releaseCallback()
	if err := <-done; !errors.Is(err, api.ErrOutputClosed) {
		t.Fatalf("consume=%v", err)
	}

	if err := <-closed; err != nil || cleaned.Load() != 1 {
		t.Fatalf("close=%v cleaned=%d", err, cleaned.Load())
	}
}

func TestManagedOutputPanicAndContextCancellation(t *testing.T) {
	for _, panicCallback := range []bool{true, false} {
		ctx, cancel := context.WithCancel(t.Context())
		cleaned := make(chan struct{})
		output := newManagedOutput(ctx, api.Metadata{}, func(ctx context.Context, consumer api.Consumer) error {
			return consumer(ctx, []byte("value"))
		}, nil, func() error { close(cleaned); return nil })
		if panicCallback {
			func() {
				defer func() {
					if recover() != "callback panic" {
						t.Error("panic swallowed")
					}
				}()
				_ = output.Consume(t.Context(), func(context.Context, []byte) error { panic("callback panic") })
			}()
		} else {
			cancel()
		}

		waitSignal(t, cleaned)
		cancel()
		if err := output.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHTTPOutputDetachedContentAndLength(t *testing.T) {
	for _, size := range []int64{-1, 3, 4} {
		var closed atomic.Int32
		body := &struct {
			io.Reader
			io.Closer
		}{strings.NewReader("abc"), closeFunc(func() error { closed.Add(1); return nil })}
		output := newHTTPOutput(t.Context(), func() {}, &http.Response{Body: body, ContentLength: size, Header: http.Header{"Content-Type": {"application/opaque"}}})
		metadata := output.Metadata()
		content, err := output.Collect(t.Context())
		if content == nil || !bytes.Equal(content.Data, []byte("abc")) || content.Metadata != metadata {
			t.Fatalf("content=%v error=%v", content, err)
		}

		if (err != nil) != (size == 4) {
			t.Fatalf("length=%d error=%v", size, err)
		}

		if err := output.Close(); err != nil || closed.Load() != 1 {
			t.Fatalf("close=%v count=%d", err, closed.Load())
		}

		if output.Metadata() != metadata || string(content.Data) != "abc" {
			t.Fatal("detached data or metadata changed")
		}
	}
}

func TestHTTPOutputConsumptionCancellationUnblocksRead(t *testing.T) {
	invocation, cancelInvocation := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelInvocation()
	reader, writer := io.Pipe()
	defer writer.Close()
	closed := make(chan struct{})
	body := &struct {
		io.Reader
		io.Closer
	}{reader, closeFunc(func() error { defer close(closed); return reader.Close() })}
	output := newHTTPOutput(invocation, func() {}, &http.Response{Body: body, ContentLength: -1})
	ctx, cancel := context.WithCancel(invocation)
	done := make(chan error, 1)
	entered := make(chan struct{})
	go func() {
		done <- output.Consume(ctx, func(context.Context, []byte) error { close(entered); return nil })
	}()
	if _, err := writer.Write([]byte("prefix")); err != nil {
		t.Fatal(err)
	}
	waitSignal(t, entered)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}

	case <-time.After(5 * time.Second):
		t.Fatal("read did not stop")
	}

	if err := output.Close(); err != nil {
		t.Fatal(err)
	}

	waitSignal(t, closed)
}

func TestManagedOutputEffectiveDeadline(t *testing.T) {
	for _, invocationEarlier := range []bool{false, true} {
		early := time.Now().Add(time.Hour)
		late := early.Add(time.Hour)
		invocationDeadline, consumptionDeadline := late, early
		if invocationEarlier {
			invocationDeadline, consumptionDeadline = early, late
		}

		invocation, cancelInvocation := context.WithDeadline(t.Context(), invocationDeadline)
		consumption, cancelConsumption := context.WithDeadline(t.Context(), consumptionDeadline)
		output := newManagedOutput(invocation, api.Metadata{}, func(ctx context.Context, consumer api.Consumer) error {
			return consumer(ctx, []byte("value"))
		}, nil, func() error { return nil })
		err := output.Consume(consumption, func(ctx context.Context, _ []byte) error {
			deadline, ok := ctx.Deadline()
			if !ok || !deadline.Equal(early) {
				t.Errorf("deadline=%v known=%v, want %v", deadline, ok, early)
			}

			return nil
		})
		cancelInvocation()
		cancelConsumption()
		if err != nil {
			t.Fatal(err)
		}

		if err := output.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
