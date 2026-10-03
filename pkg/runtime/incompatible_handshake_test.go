package runtime

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	wirev1 "github.com/MontFerret/wire/gen/ferret/wire/v1"
)

func TestWireRejectsHandshakeWithoutRuntimeVersion(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	defer listener.Close()
	server := grpc.NewServer()
	wirev1.RegisterRuntimeServiceServer(server, &unversionedServer{})
	defer server.Stop()
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	resources, err := New(ctx, Options{Type: "wire", Endpoint: "tcp://" + listener.Addr().String()})
	if resources != nil || err == nil || !strings.Contains(err.Error(), "invalid Connect handshake") {
		t.Fatalf("unversioned handshake: resources=%v error=%v", resources, err)
	}

	server.Stop()
	select {
	case <-served:
	case <-ctx.Done():
		t.Fatal("incompatible server did not stop")
	}
}
