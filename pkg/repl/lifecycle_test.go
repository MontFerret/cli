package repl

import (
	"bytes"
	"context"
	"errors"
	"io"
	goruntime "runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MontFerret/cli/v2/internal/testutil"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestWireReplIdleStop(t *testing.T) {
	for _, action := range []string{"cancel", "disconnect", "end Connect"} {
		t.Run(action, func(t *testing.T) {
			lifetime, endConnect := context.WithCancel(t.Context())
			defer endConnect()
			host := wirehost.NewWithConnectContext(lifetime, t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			reading := make(chan struct{})
			input := testutil.ContextReader(func(ctx context.Context, _ []byte) (int, error) {
				close(reading)
				<-ctx.Done()
				return 0, ctx.Err()
			})

			done := make(chan error, 1)
			go func() {
				done <- StartWithIO(ctx, runtime.Options{Type: "wire", Endpoint: host.Endpoint}, nil, input, io.Discard, io.Discard)
			}()
			select {
			case <-reading:
			case <-time.After(10 * time.Second):
				t.Fatal("REPL did not request input")
			}

			switch action {
			case "cancel":
				cancel()
			case "disconnect":
				host.StopTransport()
			case "end Connect":
				endConnect()
			}

			select {
			case err := <-done:
				if err == nil || (action == "cancel" && !errors.Is(err, context.Canceled)) || (action != "cancel" && !strings.Contains(err.Error(), "check the runtime host")) {
					t.Fatalf("idle stop: %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("idle REPL did not exit")
			}
		})
	}
}

func TestWireReplWaitsForBorrowedInputAfterRuntimeClose(t *testing.T) {
	host := wirehost.New(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reading, wake := make(chan struct{}), make(chan struct{})
	wakeReader := sync.OnceFunc(func() { close(wake) })
	defer wakeReader()
	next := strings.NewReader("RETURN 42\nexit\n")
	input := readFunc(func(b []byte) (int, error) {
		select {
		case <-reading:
			return next.Read(b)
		default:
			close(reading)
			<-wake
			return 0, io.EOF
		}
	})

	done := make(chan error, 1)
	go func() {
		done <- StartWithIO(ctx, runtime.Options{Type: "wire", Endpoint: host.Endpoint}, nil, input, io.Discard, io.Discard)
	}()
	select {
	case <-reading:
	case <-time.After(10 * time.Second):
		t.Fatal("REPL did not start reading")
	}

	cancel()
	timeout, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	if err := host.WaitForClientsClosed(timeout); err != nil {
		t.Fatalf("borrowed read held the Wire connection open: %v", err)
	}

	select {
	case err := <-done:
		t.Fatalf("returned with an outstanding borrowed read: %v", err)
	default:
	}

	wakeReader()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("lost cancellation: %v", err)
		}
	case <-timeout.Done():
		t.Fatal("did not return after read settled")
	}

	var out bytes.Buffer
	if err := StartWithIO(t.Context(), runtime.NewDefaultOptions(), nil, input, &out, io.Discard); err != nil || !strings.Contains(out.String(), "\n42\n") {
		t.Fatalf("next REPL lost its input: err=%v transcript=%s", err, &out)
	}
}

func TestWireReplExecutionStops(t *testing.T) {
	for _, transportLoss := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "transport loss"}[transportLoss], func(t *testing.T) {
			host := wirehost.New(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var stdout, stderr bytes.Buffer
			done := make(chan error, 1)
			go func() {
				done <- StartWithIO(ctx, runtime.Options{Type: "wire", Endpoint: host.Endpoint}, nil, strings.NewReader("RETURN DEMO::WAIT()\n"), &stdout, &stderr)
			}()
			select {
			case <-host.WaitStarted:
			case err := <-done:
				t.Fatalf("REPL returned before query started: %v\nstdout=%q\nstderr=%q", err, stdout.String(), stderr.String())
			case <-time.After(10 * time.Second):
				goroutines := make([]byte, 1<<20)
				goroutines = goroutines[:goruntime.Stack(goroutines, true)]
				cancel()

				// The REPL owns these writers until it returns. Do not inspect
				// their buffers if cancellation fails to settle the goroutine.
				select {
				case err := <-done:
					t.Fatalf("query did not start; REPL returned after cancellation: %v\nstdout=%q\nstderr=%q\ngoroutines before cancellation:\n%s", err, stdout.String(), stderr.String(), goroutines)
				case <-time.After(10 * time.Second):
					t.Fatalf("query did not start and REPL did not settle after cancellation\ngoroutines before cancellation:\n%s", goroutines)
				}
			}

			if transportLoss {
				host.StopTransport()
			} else {
				cancel()
			}

			select {
			case err := <-done:
				if err == nil || (!transportLoss && !errors.Is(err, context.Canceled)) || (transportLoss && !strings.Contains(err.Error(), "check the runtime host")) {
					t.Fatalf("REPL stop error = %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("REPL did not settle")
			}

			select {
			case <-host.WaitFinished:
			case <-time.After(10 * time.Second):
				t.Fatal("host work leaked")
			}
		})
	}
}
