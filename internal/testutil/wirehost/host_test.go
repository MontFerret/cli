package wirehost

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/MontFerret/wire/server"
)

func TestHostCloseBeforeServe(t *testing.T) {
	var closes atomic.Int32
	runtime := &closeRuntime{close: func() error {
		closes.Add(1)

		return nil
	}}

	wire, err := server.New(runtime)
	if err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = ln.Close() })
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	// Force teardown to win startup without relying on goroutine scheduling.
	if err := wire.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}

	host := &Host{Runtime: runtime, server: wire, listener: &listener{Listener: ln}, served: make(chan error, 1)}
	serveErr := wire.Serve(ctx, host.listener)
	if !errors.Is(serveErr, grpc.ErrServerStopped) {
		t.Fatalf("Serve after shutdown = %v, want ErrServerStopped", serveErr)
	}

	host.served <- serveErr
	for range 2 {
		if err := host.Close(); err != nil {
			t.Fatalf("Close = %v", err)
		}
	}

	if closes.Load() != 1 {
		t.Fatalf("runtime closes = %d, want 1", closes.Load())
	}

	if _, err := ln.Accept(); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("listener Accept = %v, want ErrClosed", err)
	}
}

func TestHostCloseConcurrentRetainsFailures(t *testing.T) {
	serveErr := errors.New("serve failed")
	cleanupErr := errors.New("runtime close failed")
	closing := make(chan struct{})
	release := make(chan struct{})
	var closes atomic.Int32
	host := &Host{
		served: make(chan error, 1),
		Runtime: &closeRuntime{close: func() error {
			closes.Add(1)
			close(closing)
			<-release

			return cleanupErr
		}},
	}
	host.served <- errors.Join(grpc.ErrServerStopped, serveErr)
	start := make(chan struct{})
	results := make(chan error, 8)
	for range cap(results) {
		go func() {
			<-start
			results <- host.Close()
		}()
	}

	close(start)
	<-closing
	close(release)
	for range cap(results) {
		err := <-results
		if !errors.Is(err, serveErr) || !errors.Is(err, cleanupErr) {
			t.Errorf("Close = %v, want serving and cleanup failures", err)
		}

		if again := host.Close(); again != err {
			t.Errorf("Close changed retained result: %v, want %v", again, err)
		}
	}

	if closes.Load() != 1 {
		t.Fatalf("runtime closes = %d, want 1", closes.Load())
	}
}
