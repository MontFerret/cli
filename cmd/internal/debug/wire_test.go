package debug

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/MontFerret/api"
	"github.com/MontFerret/api/diagnostics"
	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
	inputtest "github.com/MontFerret/cli/v2/internal/testutil"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/ferret/v2"
)

func scriptCommand(t *testing.T, query string, args ...string) (*cobra.Command, context.Context) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.fql")
	testutil.WriteQuery(t, path, query)
	store := testutil.NewStore(t)
	cmd := New(store)
	cmd.SilenceUsage = true
	cmd.SetArgs(append(args, path))

	return cmd, config.With(t.Context(), store)
}

func waitForDisconnect(t *testing.T, host *wirehost.Host) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := host.WaitForClientsClosed(ctx); err != nil || host.Connections() != 1 {
		t.Fatalf("expected one released connection: count=%d err=%v", host.Connections(), err)
	}
}

func TestDebugInspectionAndStepping(t *testing.T) {
	const query = "FUNC increment(value) {\n  LET copy = value\n  RETURN copy + 1\n}\nLET number = @limit\nLET answer = increment(number)\nLET nextValue = answer + 1\nRETURN nextValue"

	for _, backend := range []string{"builtin", "wire"} {
		t.Run(backend, func(t *testing.T) {
			args := []string{"--param=limit=10", "--runtime=" + backend}
			if backend == "wire" {
				host := wirehost.New(t)
				args = append(args, "--runtime-endpoint="+host.Endpoint)
				defer waitForDisconnect(t, host)
			}

			cmd, ctx := scriptCommand(t, query, args...)
			cmd.SetIn(strings.NewReader("b --exact 6\nbl\nc\nd 1\nd 99\ns\nbt\nl\ne value + 1\nout\nn\nc\nq\n"))
			var out bytes.Buffer
			cmd.SetOut(&out)

			if err := cmd.ExecuteContext(ctx); err != nil {
				t.Fatal(err)
			}

			transcript := out.String()
			for _, want := range []string{", exact).", "Requested", "Paused on breakpoint 1", "Breakpoint 1 deleted.",
				"Unknown breakpoint: 99", "#0 increment", "#1 <main>", "value = 10", "\n11\n", "Result:\n12\n"} {
				if !strings.Contains(transcript, want) {
					t.Fatalf("missing %q in transcript:\n%s", want, transcript)
				}
			}

			if strings.Count(transcript, "Paused after step") != 3 {
				t.Fatalf("step, next, and out did not all stop:\n%s", transcript)
			}
		})
	}
}

func TestDebugPauseAtStop(t *testing.T) {
	for _, backend := range []string{"builtin", "wire"} {
		t.Run(backend, func(t *testing.T) {
			args := []string{"--runtime=" + backend}
			want := "Paused on pause request"
			if backend == "wire" {
				host := wirehost.New(t)
				args = append(args, "--runtime-endpoint="+host.Endpoint)
				defer waitForDisconnect(t, host)
				want = "Pause error:"
			}

			cmd, ctx := scriptCommand(t, "LET x = 1\nRETURN x + 1", args...)
			cmd.SetIn(strings.NewReader("pause\nc\nc\nq\n"))
			var out bytes.Buffer
			cmd.SetOut(&out)

			if err := cmd.ExecuteContext(ctx); err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(out.String(), want) || !strings.Contains(out.String(), "Result:\n2\n") {
				t.Fatalf("pause and resume:\n%s", &out)
			}
		})
	}
}

func TestWireDebugTerminalErrorKeepsOutput(t *testing.T) {
	host := wirehost.New(t, ferret.WithAfterRunHook(func(context.Context, error) error {
		return errors.New("private host cleanup failure")
	}))
	cmd, ctx := scriptCommand(t, "RETURN 42", "--runtime=wire", "--runtime-endpoint="+host.Endpoint)
	cmd.SetIn(strings.NewReader("c\nhelp\n"))
	// Capture both streams in order, including Cobra's rendering of the returned error.
	var transcript bytes.Buffer
	cmd.SetOut(&transcript)
	cmd.SetErr(&transcript)

	err := cmd.ExecuteContext(ctx)
	if err == nil {
		t.Fatal("terminal failure reported success")
	}

	got := transcript.String()
	content, diagnostic := strings.Index(got, "Result:\n42\n"), strings.Index(got, "runtime operation failed")
	if content < 0 || diagnostic <= content || strings.Count(got, "runtime operation failed") != 1 || strings.Contains(got, "Commands:") {
		t.Fatalf("terminal result and error:\n%s", got)
	}

	waitForDisconnect(t, host)
}

func TestWireDebugUndeliveredCompletion(t *testing.T) {
	host := wirehost.New(t, ferret.WithAfterRunHook(func(context.Context, error) error {
		// The runtime completes but Wire cannot deliver this command result.
		return diagnostics.Diagnostics{{Message: strings.Repeat("x", 5<<20)}}
	}))
	cmd, ctx := scriptCommand(t, "RETURN 42", "--runtime=wire", "--runtime-endpoint="+host.Endpoint)
	cmd.SetIn(strings.NewReader("c\nhelp\nwhere\nc\nq\n"))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	err := cmd.ExecuteContext(ctx)
	if err == nil || !strings.Contains(err.Error(), "outbound message limit exceeded") {
		t.Fatalf("expected command delivery failure, got %v", err)
	}

	if strings.Contains(out.String(), "Commands:") || strings.Contains(out.String(), "Debug session terminated.") {
		t.Fatalf("accepted input after losing the execution result: %s", &out)
	}

	waitForDisconnect(t, host)
}

func TestTerminalBreakpoints(t *testing.T) {
	for _, backend := range []string{"builtin", "wire"} {
		t.Run(backend, func(t *testing.T) {
			flags := []string{"--runtime=" + backend}
			if backend == "wire" {
				host := wirehost.New(t)
				flags = append(flags, "--runtime-endpoint="+host.Endpoint)
				defer waitForDisconnect(t, host)
			}

			cmd, ctx := scriptCommand(t, "RETURN 42", flags...)
			cmd.SetIn(strings.NewReader("b 1\nc\nbl\nb 1\nd 1\nq\n"))
			var out bytes.Buffer
			cmd.SetOut(&out)

			if err := cmd.ExecuteContext(ctx); err != nil {
				t.Fatal(err)
			}

			_, after, ok := strings.Cut(out.String(), "Result:\n42\n")
			if !ok || !strings.Contains(after, "Requested") || !strings.Contains(after, "next-file") {
				t.Fatalf("missing retained breakpoint: %s", &out)
			}

			want := []string{"Breakpoint 2 set", "Breakpoint 1 deleted."}
			if backend == "wire" {
				want = []string{"Breakpoint error: debug session is terminal", "Delete breakpoint error: debug session is terminal"}
			}

			for _, message := range want {
				if !strings.Contains(after, message) {
					t.Fatalf("expected %q after completion: %s", message, after)
				}
			}
		})
	}
}

func TestWireDebugPreparationFailures(t *testing.T) {
	for _, mode := range []string{"compile", "handshake", "browser", "filesystem", "http"} {
		t.Run(mode, func(t *testing.T) {
			var host *wirehost.Host
			query := "RETURN 42"
			want := ""
			var flags []string

			switch mode {
			case "compile":
				query, want = "RETURN )", "script.fql:1:"
			case "handshake":
				host = wirehost.NewWithVersionFunc(t, func(context.Context) (api.Version, error) {
					return "", errors.New("metadata unavailable")
				})
				want = "connect Wire runtime"
			case "browser":
				flags, want = []string{"--browser-open"}, "browser configuration"
			case "filesystem":
				flags, want = []string{"--policy-fs-read-only"}, "filesystem"
			case "http":
				flags, want = []string{"--policy-http-follow-redirects=true"}, "HTTP"
			}

			if host == nil {
				host = wirehost.New(t)
			}

			args := append([]string{"--runtime=wire", "--runtime-endpoint=" + host.Endpoint}, flags...)
			cmd, ctx := scriptCommand(t, query, args...)
			cmd.SetIn(strings.NewReader("c\n"))
			var out, stderr bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)

			err := cmd.ExecuteContext(ctx)
			if err == nil || !strings.Contains(stderr.String(), want) || out.Len() != 0 {
				t.Fatalf("error=%v stdout=%q stderr=%q", err, out.String(), stderr.String())
			}

			if mode == "compile" && (strings.Count(stderr.String(), "1 | RETURN )") != 1 || !strings.Contains(stderr.String(), "^")) {
				t.Fatalf("lost or repeated source diagnostic: %s", &stderr)
			}

			if len(flags) == 0 {
				waitForDisconnect(t, host)
			} else if host.Connections() != 0 {
				t.Fatalf("invalid policy opened %d connections", host.Connections())
			}
		})
	}
}

func TestWireDebugEarlyExit(t *testing.T) {
	for _, line := range []string{"q\n", ""} {
		host := wirehost.New(t)
		cmd, ctx := scriptCommand(t, "RETURN DEMO::WAIT()", "--runtime=wire", "--runtime-endpoint="+host.Endpoint)
		cmd.SetIn(strings.NewReader(line))
		cmd.SetOut(io.Discard)

		if err := cmd.ExecuteContext(ctx); err != nil {
			t.Fatal(err)
		}

		waitForDisconnect(t, host)
		select {
		case <-host.WaitStarted:
			t.Fatal("early exit resumed execution")
		default:
		}
	}
}

func TestWireDebugInterrupted(t *testing.T) {
	for _, state := range []string{"idle", "executing"} {
		for _, action := range []string{"cancel", "disconnect", "end Connect"} {
			t.Run(state+"/"+action, func(t *testing.T) {
				lifetime, endConnect := context.WithCancel(t.Context())
				defer endConnect()
				host := wirehost.NewWithConnectContext(lifetime, t)
				cmd, parent := scriptCommand(t, "RETURN DEMO::WAIT()", "--runtime=wire", "--runtime-endpoint="+host.Endpoint)
				ctx, cancel := context.WithCancel(parent)
				defer cancel()

				var input io.Reader = strings.NewReader("c\n")
				ready := host.WaitStarted
				if state == "idle" {
					reading := make(chan struct{})
					ready = reading
					next := strings.NewReader("c\nq\n")
					input = inputtest.ContextReader(func(ctx context.Context, b []byte) (int, error) {
						select {
						case <-reading:
							return next.Read(b)
						default:
							close(reading)
							<-ctx.Done()
							return 0, ctx.Err()
						}
					})
				}

				cmd.SetIn(input)
				cmd.SetOut(io.Discard)
				cmd.SetErr(io.Discard)
				done := make(chan error, 1)
				go func() { done <- cmd.ExecuteContext(ctx) }()

				select {
				case <-ready:
				case err := <-done:
					t.Fatalf("exited before %s: %v", state, err)
				case <-time.After(10 * time.Second):
					t.Fatalf("debugger never reached %s", state)
				}

				switch action {
				case "cancel":
					cancel()
				case "end Connect":
					endConnect()
				default:
					host.StopTransport()
				}

				select {
				case err := <-done:
					if err == nil || (action == "cancel" && !errors.Is(err, context.Canceled)) {
						t.Fatalf("unexpected %s result: %v", action, err)
					}
				case <-time.After(10 * time.Second):
					t.Fatal("debugger did not exit")
				}

				if state == "executing" {
					select {
					case <-host.WaitFinished:
					case <-time.After(5 * time.Second):
						t.Fatal("host function did not settle")
					}
				}

				waitForDisconnect(t, host)

				if state == "idle" {
					next, ctx := scriptCommand(t, "RETURN 42")
					var out bytes.Buffer
					next.SetIn(input)
					next.SetOut(&out)
					if err := next.ExecuteContext(ctx); err != nil || !strings.Contains(out.String(), "Result:\n42\n") {
						t.Fatalf("next prompt lost its input: err=%v transcript=%s", err, &out)
					}
				}
			})
		}
	}
}
