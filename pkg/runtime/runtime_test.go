package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunArtifact_RemoteRuntimeRejected(t *testing.T) {
	_, err := RunArtifact(context.Background(), Options{Type: "https://worker.example"}, []byte("FBC2"), nil)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrArtifactRequiresBuiltinRuntime) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewWireRequiresContextAwareFactory(t *testing.T) {
	rt, err := New(Options{Type: "wire", Endpoint: "tcp://127.0.0.1:123"})
	if rt != nil || err == nil || !strings.Contains(err.Error(), "context-aware OpenSource") {
		t.Fatalf("legacy facade constructed a Wire runtime: %v, %v", rt, err)
	}
}
