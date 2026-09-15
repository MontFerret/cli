package check

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
)

func TestCheckCommandCanceledContext(t *testing.T) {
	input := filepath.Join(t.TempDir(), "query.fql")
	testutil.WriteQuery(t, input, "RETURN 1")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	command := New(nil)
	command.SetContext(ctx)

	stderr, err := testutil.CaptureStderr(t, func() error {
		return command.RunE(command, []string{input})
	})
	if err == nil || err.Error() != "1 of 1 scripts have errors" {
		t.Fatalf("expected compilation failure, got %v", err)
	}

	if !strings.Contains(stderr, context.Canceled.Error()) {
		t.Fatalf("expected cancellation diagnostic, got %q", stderr)
	}
}
