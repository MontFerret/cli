// Package wirehost provides a configured native runtime over real loopback TCP
// for CLI tests and benchmarks. It uses only public Ferret and Wire APIs.
package wirehost

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/MontFerret/api"
	"github.com/MontFerret/ferret/v2"
	native "github.com/MontFerret/ferret/v2/pkg/runtime"
	"github.com/MontFerret/ferret/v2/uapi"
	"github.com/MontFerret/wire/server"
)

// Host owns its configured engine, server, and listener independently of clients.
type Host struct {
	Endpoint     string
	Runtime      api.Runtime
	WaitStarted  chan struct{}
	WaitFinished chan struct{}
	engine       *ferret.Engine
	server       *server.Server
	listener     *listener
	served       chan error
	once         sync.Once
	err          error
}

// New starts a bounded, test-owned host with DEMO::IDENTITY and DEMO::WAIT.
func New(t testing.TB) *Host {
	return NewWithVersion(t, "test-core-version")
}

// NewWithVersion hosts an opaque runtime version, including an empty value.
func NewWithVersion(t testing.TB, version api.Version) *Host {
	return newHost(t, version, nil)
}

// NewWithVersionFunc controls hosted metadata retrieval for handshake failure tests.
func NewWithVersionFunc(t testing.TB, version func(context.Context) (api.Version, error)) *Host {
	return newHost(t, "", version)
}

func newHost(t testing.TB, version api.Version, retrieve func(context.Context) (api.Version, error)) *Host {
	t.Helper()
	h := &Host{WaitStarted: make(chan struct{}), WaitFinished: make(chan struct{})}
	var started, finished sync.Once
	var err error
	h.engine, err = ferret.New(ferret.WithFSRoot(t.TempDir()), ferret.WithFunctionsRegistrar(func(ns native.Namespace) {
		demo := ns.Namespace("DEMO")
		demo.Function().A1().Add("IDENTITY", func(_ context.Context, value native.Value) (native.Value, error) {
			return value, nil
		})
		demo.Function().A0().Add("WAIT", func(ctx context.Context) (native.Value, error) {
			started.Do(func() { close(h.WaitStarted) })
			defer finished.Do(func() { close(h.WaitFinished) })
			<-ctx.Done()

			return nil, ctx.Err()
		})
	}))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := h.Close(); err != nil {
			t.Errorf("close Wire fixture: %v", err)
		}
	})
	h.Runtime = uapi.Wrap(h.engine, version)
	if retrieve != nil {
		h.Runtime = &versionRuntime{Runtime: h.Runtime, retrieve: retrieve}
	}

	h.server, err = server.NewServer(h.Runtime)
	if err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	h.listener = &listener{Listener: ln}
	h.Endpoint = "tcp://" + ln.Addr().String()
	h.served = make(chan error, 1)
	go func() { h.served <- h.server.Serve(context.Background(), h.listener) }()

	return h
}

// Connections counts physical connections accepted by the fixture.
func (h *Host) Connections() int64 {
	return h.listener.accepted.Load()
}

// WaitForClientsClosed observes transport release without polling or closing clients.
func (h *Host) WaitForClientsClosed(ctx context.Context) error {
	for {
		h.listener.mu.Lock()
		if len(h.listener.connections) == 0 {
			h.listener.mu.Unlock()

			return nil
		}

		if h.listener.changed == nil {
			h.listener.changed = make(chan struct{})
		}

		changed := h.listener.changed
		h.listener.mu.Unlock()

		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// StopTransport simulates a broken transport without closing the host engine.
func (h *Host) StopTransport() {
	h.listener.mu.Lock()
	connections := make([]net.Conn, 0, len(h.listener.connections))
	for connection := range h.listener.connections {
		connections = append(connections, connection)
	}
	h.listener.mu.Unlock()

	for _, connection := range connections {
		_ = connection.Close()
	}
}

// Close stops the server before releasing the host-owned engine.
func (h *Host) Close() error {
	h.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if h.server != nil {
			h.err = errors.Join(h.err, h.server.Shutdown(ctx))
		}

		if h.listener != nil {
			if err := h.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				h.err = errors.Join(h.err, err)
			}
		}

		if h.served != nil {
			select {
			case err := <-h.served:
				h.err = errors.Join(h.err, err)
			case <-ctx.Done():
				h.err = errors.Join(h.err, ctx.Err())
			}
		}

		if h.Runtime != nil {
			h.err = errors.Join(h.err, h.Runtime.Close())
		}

		if h.engine != nil {
			h.err = errors.Join(h.err, h.engine.Close())
		}
	})

	return h.err
}
