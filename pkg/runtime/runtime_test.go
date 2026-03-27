package runtime

import (
	"context"
	"strings"
	"testing"
)

func TestRunArtifact_RemoteRuntimeRejected(t *testing.T) {
	_, err := RunArtifact(context.Background(), Options{Type: "https://worker.example"}, []byte("FBC2"), nil)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "compiled artifacts require the builtin runtime") {
		t.Fatalf("unexpected error: %v", err)
	}
}
