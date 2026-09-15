package inspect

import (
	"context"
	"strings"
	"testing"

	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
)

func TestInspectCommandCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	command := New(nil)
	command.SetContext(ctx)

	if err := command.Flags().Set("eval", "RETURN 1"); err != nil {
		t.Fatal(err)
	}

	var stdout string

	stderr, err := testutil.CaptureStderr(t, func() error {
		var runErr error
		stdout, runErr = testutil.CaptureStdout(t, func() error {
			return command.RunE(command, nil)
		})

		return runErr
	})
	if err == nil || err.Error() != "compilation failed" {
		t.Fatalf("expected compilation failure, got %v", err)
	}

	if !strings.Contains(stderr, context.Canceled.Error()) {
		t.Fatalf("expected cancellation diagnostic, got %q", stderr)
	}

	if stdout != "" {
		t.Fatalf("canceled inspection emitted disassembly: %q", stdout)
	}
}
