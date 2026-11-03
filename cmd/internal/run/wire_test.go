package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/MontFerret/api"
	portable "github.com/MontFerret/api/diagnostics"
	"github.com/MontFerret/wire/client"
	"github.com/MontFerret/wire/pkg/failure"

	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestWireRunCommands(t *testing.T) {
	host := wirehost.New(t)
	for _, mode := range []string{"inline", "file", "exec", "stdin"} {
		t.Run(mode, func(t *testing.T) {
			query := "RETURN DEMO::IDENTITY([@n, @b, @arr, @obj, @str, @nothing])"
			args := []string{"run", "--runtime", "wire", "--runtime-endpoint", host.Endpoint,
				"--param", "n=42", "--param", "b=true", "--param", "arr=[1,2]", "--param", `obj={"value":false}`, "--param", `str="123"`, "--param", "nothing=null"}

			switch mode {
			case "exec":
				args[0] = "exec"
				args = append(args, "--eval", query)
			case "inline":
				args = append(args, "--eval", query)
			case "file":
				path := filepath.Join(t.TempDir(), "query.fql")
				testutil.WriteQuery(t, path, query)
				args = append(args, path)
			}

			invoke := func() {
				stdout, stderr, err := runWireCommand(t, args)
				if err != nil || stderr != "" || stdout != `[42,true,[1,2],{"value":false},"123",null]` {
					t.Fatalf("stdout=%q stderr=%q error=%v", stdout, stderr, err)
				}
			}

			if mode == "stdin" {
				testutil.WithStdinBytes(t, []byte(query), invoke)
			} else {
				invoke()
			}
		})
	}
}

func TestWireRunRenderedDiagnostics(t *testing.T) {
	host := wirehost.New(t)
	for _, test := range []struct {
		query, rendered string
		span            api.Span
	}{
		{"let arr = []\n\nreturn", "SyntaxError: Expected expression after 'return'\n --> %s:3:7\n  |\n2 | \n3 | return\n  |       ^ missing return value\nHint: Did you forget to provide a value to return?\n", api.Span{Start: 20, End: 20}},
		{"RETURN )", "SyntaxError: Expected expression after 'RETURN'\n --> %s:1:8\n  |\n1 | RETURN )\n  |        ^ missing return value\nHint: Did you forget to provide a value to return?\n", api.Span{Start: 7, End: 8}},
		{"let arr = [1, 2, 3", "SyntaxError: Unclosed array literal\n --> %s:1:19\n  |\n1 | let arr = [1, 2, 3\n  |                   ^ expected ']'\nHint: Add ']' to close the array.\n", api.Span{Start: 18, End: 18}},
		{"return [1, 2 +", "SyntaxError: Expected right-hand expression after '+'\n --> %s:1:15\n  |\n1 | return [1, 2 +\n  |               ^ missing expression\nHint: Provide an expression after the operator.\n", api.Span{Start: 14, End: 14}},
	} {
		t.Run(test.query, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invalid.fql")
			testutil.WriteQuery(t, path, test.query)
			stdout, stderr, err := runWireCommand(t, []string{"run", "--runtime", "wire", "--runtime-endpoint", host.Endpoint, path})
			if err == nil || stdout != "" {
				t.Fatalf("stdout=%q error=%v", stdout, err)
			}

			if want := fmt.Sprintf(test.rendered, path); stderr != want {
				t.Errorf("stderr = %q, want %q", stderr, want)
			}

			var values portable.Diagnostics
			var rpc *client.Error
			var terminal *failure.Failure
			if errors.As(err, &rpc) {
				values = rpc.Diagnostics
			} else if errors.As(err, &terminal) {
				values = terminal.Diagnostics
			}

			if len(values) != 1 || len(values[0].Annotations) != 1 || values[0].Annotations[0].Range.Span != test.span || !values[0].Annotations[0].Primary {
				t.Fatalf("incorrect host diagnostic range: %+v", values)
			}
		})
	}
}

func TestWireArtifactRejectedBeforeConnecting(t *testing.T) {
	host := wirehost.New(t)
	path := filepath.Join(t.TempDir(), "query.fqlc")
	if err := os.WriteFile(path, []byte("FBC2"), 0600); err != nil {
		t.Fatal(err)
	}

	_, _, err := runWireCommand(t, []string{"run", "--runtime", "wire", "--runtime-endpoint", host.Endpoint, path})
	if !errors.Is(err, cliruntime.ErrArtifactRequiresBuiltinRuntime) || host.Connections() != 0 {
		t.Fatalf("error=%v connections=%d", err, host.Connections())
	}
}

func runWireCommand(t *testing.T, args []string) (string, string, error) {
	t.Helper()
	store := testutil.NewStore(t)
	root := &cobra.Command{SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(New(store))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	err := root.ExecuteContext(config.With(ctx, store))

	return stdout.String(), stderr.String(), err
}
