package repl

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestWireReplIdleCancellation(t *testing.T) {
	testWireReplIdleStop(t, false)
}

func TestWireReplIdleTransportLoss(t *testing.T) {
	testWireReplIdleStop(t, true)
}

func testWireReplIdleStop(t *testing.T, transportLoss bool) {
	t.Helper()
	host := wirehost.New(t)
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	reading := make(chan struct{})
	input := readFunc(func(data []byte) (int, error) {
		close(reading)

		return reader.Read(data)
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var stdout, stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- StartWithIO(ctx, runtime.Options{Type: "wire", Endpoint: host.Endpoint}, nil, input, &stdout, &stderr)
	}()
	select {
	case <-reading:
	case <-time.After(10 * time.Second):
		t.Fatal("REPL did not request input")
	}

	if transportLoss {
		host.StopTransport()
	} else {
		cancel()
	}

	select {
	case err := <-done:
		if err == nil || (!transportLoss && !errors.Is(err, context.Canceled)) || (transportLoss && !strings.Contains(err.Error(), "check the runtime host")) {
			t.Fatalf("idle stop: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("idle REPL did not exit")
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
			case <-time.After(10 * time.Second):
				t.Fatal("query did not start")
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
