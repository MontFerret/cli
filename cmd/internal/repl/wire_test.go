package repl

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/cli/v2/pkg/config"
)

func TestWireReplReusesRuntimeAfterSyntaxError(t *testing.T) {
	host := wirehost.New(t)
	store := testutil.NewStore(t)
	cmd := New(store)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--runtime", "wire", "--runtime-endpoint", host.Endpoint})
	cmd.SetIn(strings.NewReader("RETURN DEMO::IDENTITY(41)\nRETURN )\nRETURN DEMO::IDENTITY(42)\nexit\n"))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	if err := cmd.ExecuteContext(config.With(ctx, store)); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"Wire runtime: " + host.Endpoint + "; version: test-core-version", "41\n", "42\n"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout missing %q: %q", want, stdout.String())
		}
	}

	for _, want := range []string{"anonymous:1:8", "1 | RETURN )", "  |        ^"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr missing %q: %q", want, stderr.String())
		}
	}

	if host.Connections() != 1 {
		t.Fatalf("connections=%d, want 1", host.Connections())
	}
}
