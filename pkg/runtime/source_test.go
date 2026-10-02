package runtime

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/MontFerret/api"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestWireEndpointValidation(t *testing.T) {
	for _, endpoint := range []string{"", "tcp://localhost:123", "tcp://127.0.0.2:123", "tcp://[::1]:123", "http://127.0.0.1:123", "tcp://user@127.0.0.1:123", "tcp://127.0.0.1:0", "tcp://127.0.0.1:65536", "tcp://127.0.0.1:-1", "tcp://127.0.0.1:123/", "tcp://127.0.0.1:123?", "tcp://127.0.0.1:123#", "tcp://127.0.0.1:123/path", " tcp://127.0.0.1:123"} {
		t.Run(endpoint, func(t *testing.T) {
			if _, err := wireAddress(endpoint); err == nil {
				t.Fatal("accepted invalid endpoint")
			}
		})
	}

	for _, endpoint := range []string{"tcp://127.0.0.1:1", "tcp://127.0.0.1:65535"} {
		if _, err := wireAddress(endpoint); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceOutputAndCleanupFailures(t *testing.T) {
	runErr := errors.New("execution failed")
	closeErr := errors.New("logical cleanup failed")
	physicalErr := errors.New("physical cleanup failed")

	for _, output := range []*api.Output{nil, {}, {Content: []byte("available")}} {
		rt := &scriptedRuntime{output: output, err: runErr, closeErr: closeErr}
		physicalClosed := 0
		resources := &SourceResources{Runtime: rt, owners: []io.Closer{rt, closeFunc(func() error {
			if rt.closed != 1 {
				t.Error("transport closed before runtime")
			}
			physicalClosed++

			return physicalErr
		})}}
		reader, err := RunSource(t.Context(), resources, source.New("source.fql", "RETURN 42"), nil)
		if !errors.Is(err, runErr) || (reader == nil) != (output == nil) {
			t.Fatalf("output presence lost: reader=%v error=%v", reader, err)
		}

		if reader != nil {
			data, _ := io.ReadAll(reader)
			_ = reader.Close()
			if string(data) != string(output.Content) {
				t.Fatalf("output changed: %q", data)
			}
		}

		for range 2 {
			err = resources.Close()
			if !errors.Is(err, closeErr) || !errors.Is(err, physicalErr) {
				t.Fatalf("cleanup errors lost: %v", err)
			}
		}

		if rt.closed != 1 || physicalClosed != 1 {
			t.Fatal("cleanup repeated")
		}
	}

	rt := &scriptedRuntime{output: &api.Output{}}
	reader, err := RunSource(t.Context(), &SourceResources{Runtime: rt}, source.New("empty.fql", "RETURN 42"), nil)
	if err != nil || reader == nil {
		t.Fatalf("successful empty output lost: %v", err)
	}

	data, _ := io.ReadAll(reader)
	_ = reader.Close()
	if len(data) != 0 {
		t.Fatalf("manufactured output: %q", data)
	}
}

func TestWireFailedHandshake(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	server := grpc.NewServer()
	defer server.Stop()
	defer listener.Close()
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	resources, err := OpenSource(t.Context(), Options{Type: "wire", Endpoint: "tcp://" + listener.Addr().String()})
	if resources != nil || err == nil || !strings.Contains(err.Error(), "unknown service") {
		t.Fatalf("failed handshake: resources=%v error=%v", resources, err)
	}

	server.Stop()
	select {
	case <-served:
	case <-time.After(10 * time.Second):
		t.Fatal("server did not stop")
	}
}

func TestWireConfiguredRuntimeAndOwnership(t *testing.T) {
	host := wirehost.New(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	resources, err := OpenSource(ctx, Options{Type: "wire", Endpoint: host.Endpoint})
	if err != nil {
		t.Fatal(err)
	}

	for _, value := range []any{nil, true, 42.0, "123", []any{1.0, false}, map[string]any{"value": nil}} {
		output, err := resources.Runtime.Run(ctx, api.NewSource("host.fql", "RETURN DEMO::IDENTITY(@value)"), api.WithParams(map[string]any{"value": value}))
		if err != nil || output == nil {
			t.Fatalf("run %v: %v", value, err)
		}
	}

	if host.Connections() != 1 {
		t.Fatalf("connections = %d, want 1", host.Connections())
	}

	for range 2 {
		if err := resources.Close(); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := host.Runtime.Run(ctx, api.NewSource("still-owned.fql", "RETURN DEMO::IDENTITY(42)")); err != nil {
		t.Fatalf("client closed host runtime: %v", err)
	}
}

func TestWireExecutionCancellation(t *testing.T) {
	host := wirehost.New(t)
	ctx, cancel := context.WithCancel(t.Context())
	resources, err := OpenSource(ctx, Options{Type: "wire", Endpoint: host.Endpoint})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := resources.Runtime.Run(ctx, api.NewSource("wait.fql", "RETURN DEMO::WAIT()"))
		done <- err
	}()
	waitSignal(t, host.WaitStarted)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("execution did not settle")
	}

	waitSignal(t, host.WaitFinished)
	if err := resources.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestWireHandshakeTimeoutAndCancellation(t *testing.T) {
	for _, cancelParent := range []bool{false, true} {
		t.Run(map[bool]string{false: "timeout", true: "parent cancellation"}[cancelParent], func(t *testing.T) {
			listener, err := net.Listen("tcp4", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}

			defer listener.Close()
			accepted := make(chan net.Conn, 1)
			go func() { connection, _ := listener.Accept(); accepted <- connection }()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := OpenSource(ctx, Options{Type: "wire", Endpoint: "tcp://" + listener.Addr().String(), ConnectTimeout: 200 * time.Millisecond})
				done <- err
			}()

			var connection net.Conn
			select {
			case connection = <-accepted:
				if connection == nil {
					t.Fatal("accept failed")
				}
			case <-time.After(10 * time.Second):
				t.Fatal("connection not established")
			}

			defer connection.Close()
			if cancelParent {
				cancel()
			}

			select {
			case err := <-done:
				want := context.DeadlineExceeded
				if cancelParent {
					want = context.Canceled
				}
				if !errors.Is(err, want) || !strings.Contains(err.Error(), listener.Addr().String()) {
					t.Fatalf("handshake error = %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("handshake did not stop")
			}

			_ = connection.SetReadDeadline(time.Now().Add(time.Second))
			if _, err := io.Copy(io.Discard, connection); err != nil {
				t.Fatalf("client transport leaked: %v", err)
			}
		})
	}
}

func TestLegacyHTTPExecution(t *testing.T) {
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"text":"RETURN @value","params":{"value":42}}` {
			t.Errorf("Worker request = %s", body)
		}

		_, _ = io.WriteString(w, "42")
	}))
	defer worker.Close()
	output, err := Run(t.Context(), Options{Type: worker.URL}, source.New("legacy.fql", "RETURN @value"), map[string]any{"value": 42})
	if err != nil {
		t.Fatal(err)
	}

	assertRuntimeOutput(t, output, "42")
}

func waitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(10 * time.Second):
		t.Fatal("fixture did not reach expected state")
	}
}
