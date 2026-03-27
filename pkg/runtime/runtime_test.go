package runtime

import (
	"context"
	"errors"
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
