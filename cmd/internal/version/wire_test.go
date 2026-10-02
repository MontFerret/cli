package version

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MontFerret/api"
	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/runtime"
	"github.com/MontFerret/wire/client"
	"github.com/MontFerret/wire/pkg/failure"
)

func TestWireVersionCommand(t *testing.T) {
	for _, value := range []api.Version{"v2.0.0-alpha.57", " runtime+opaque \n", "版本-α", "runtime\x00\xff", ""} {
		t.Run(string(value), func(t *testing.T) {
			host := wirehost.NewWithVersion(t, value)
			store := testutil.NewStore(t)
			stdout, stderr, err := executeVersion(t.Context(), store, []string{"--runtime", "wire", "--runtime-endpoint", host.Endpoint})
			want := fmt.Sprintf("Version:\n  Self: test\n  Runtime: %s\n", value)
			if err != nil || stdout != want || stderr != "" || host.Connections() != 1 {
				t.Fatalf("stdout=%q stderr=%q error=%v connections=%d; want %q", stdout, stderr, err, host.Connections(), want)
			}

			waitForVersionCleanup(t, host)

			if _, err := host.Runtime.Run(t.Context(), api.NewSource("survivor.fql", "RETURN DEMO::IDENTITY(42)")); err != nil {
				t.Fatalf("version command closed host engine: %v", err)
			}
		})
	}
}

func TestVersionBuiltinAndLegacyOutput(t *testing.T) {
	for _, mode := range []string{"builtin", "legacy"} {
		t.Run(mode, func(t *testing.T) {
			selected, expected := "builtin", runtime.EmbeddedVersion()
			if mode == "legacy" {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/info" {
						t.Errorf("version request path=%q", r.URL.Path)
					}

					_, _ = fmt.Fprint(w, `{"version":{"ferret":"legacy-version"}}`)
				}))
				defer server.Close()
				selected, expected = server.URL, "legacy-version"
			}

			stdout, stderr, err := executeVersion(t.Context(), testutil.NewStore(t), []string{"--runtime", selected})
			want := fmt.Sprintf("Version:\n  Self: test\n  Runtime: %s\n", expected)
			if err != nil || stdout != want || stderr != "" {
				t.Fatalf("stdout=%q stderr=%q error=%v; want %q", stdout, stderr, err, want)
			}
		})
	}
}

func TestWireVersionConfigurationPrecedence(t *testing.T) {
	host := wirehost.NewWithVersion(t, "selected-host")
	store := testutil.NewStore(t)
	for key, value := range map[string]string{config.ExecRuntime: "wire", config.ExecRuntimeEndpoint: "tcp://127.0.0.1:1", config.ExecRuntimeConnectTimeout: "0s"} {
		if err := store.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}

	store, err := config.NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("FERRET_RUNTIME_ENDPOINT", host.Endpoint)
	t.Setenv("FERRET_RUNTIME_CONNECT_TIMEOUT", "5s")
	stdout, _, err := executeVersion(t.Context(), store, nil)
	if err != nil || !strings.Contains(stdout, "Runtime: selected-host\n") {
		t.Fatalf("environment precedence: stdout=%q error=%v", stdout, err)
	}

	store, err = config.NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("FERRET_RUNTIME_ENDPOINT", "tcp://127.0.0.1:1")
	t.Setenv("FERRET_RUNTIME_CONNECT_TIMEOUT", "0s")
	stdout, _, err = executeVersion(t.Context(), store, []string{"--runtime-endpoint", host.Endpoint, "--runtime-connect-timeout", "5s"})
	if err != nil || !strings.Contains(stdout, "Runtime: selected-host\n") || host.Connections() != 2 {
		t.Fatalf("flag precedence: stdout=%q error=%v connections=%d", stdout, err, host.Connections())
	}
}

func TestWireVersionValidatesBeforeConnecting(t *testing.T) {
	host := wirehost.New(t)
	for _, args := range [][]string{
		{"--runtime", "wire"},
		{"--runtime", "wire", "--runtime-endpoint", "tcp://localhost:1"},
		{"--runtime", "wire", "--runtime-endpoint", host.Endpoint, "--runtime-connect-timeout=0s"},
		{"--runtime", "builtin", "--runtime-endpoint", host.Endpoint},
		{"--runtime", "http://worker.example", "--runtime-connect-timeout=5s"},
	} {
		stdout, _, err := executeVersion(t.Context(), testutil.NewStore(t), args)
		if err == nil || stdout != "" || host.Connections() != 0 {
			t.Fatalf("accepted options %v: stdout=%q error=%v connections=%d", args, stdout, err, host.Connections())
		}
	}

	for _, setting := range []struct{ key, value string }{
		{config.ExecWithBrowser, "false"}, {config.ExecBrowserAddress, runtime.DefaultBrowser},
		{config.PolicyFSReadOnly, "false"}, {config.PolicyHTTPFollowRedirects, "true"},
	} {
		for _, environment := range []bool{false, true} {
			t.Run(setting.key+"/"+fmt.Sprint(environment), func(t *testing.T) {
				store := testutil.NewStore(t)
				if environment {
					t.Setenv("FERRET_"+strings.ToUpper(strings.ReplaceAll(setting.key, "-", "_")), setting.value)
				} else if err := store.Set(setting.key, setting.value); err != nil {
					t.Fatal(err)
				}

				stdout, _, err := executeVersion(t.Context(), store, []string{"--runtime", "wire", "--runtime-endpoint", host.Endpoint})
				if err == nil || stdout != "" || host.Connections() != 0 {
					t.Fatalf("accepted explicit configuration: stdout=%q error=%v connections=%d", stdout, err, host.Connections())
				}
			})
		}
	}
}

func TestWireVersionHandshakeFailure(t *testing.T) {
	host := wirehost.NewWithVersionFunc(t, func(context.Context) (api.Version, error) {
		return "", errors.New("private hosted metadata error")
	})
	stdout, stderr, err := executeVersion(t.Context(), testutil.NewStore(t), []string{"--runtime", "wire", "--runtime-endpoint", host.Endpoint})
	var wireErr *client.Error
	if !errors.As(err, &wireErr) || wireErr.Category != failure.CategoryInternalRuntime || stdout != "" || stderr != "" || strings.Contains(err.Error(), "private") {
		t.Fatalf("failed metadata: stdout=%q stderr=%q error=%v", stdout, stderr, err)
	}

	waitForVersionCleanup(t, host)

	if !strings.Contains(err.Error(), host.Endpoint) || host.Connections() != 1 {
		t.Fatalf("missing endpoint context or unexpected retry: error=%v connections=%d", err, host.Connections())
	}
}

func TestWireVersionOutputFailureReleasesClient(t *testing.T) {
	host := wirehost.New(t)
	cmd := New(testutil.NewStore(t))
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	cmd.SetArgs([]string{"--runtime", "wire", "--runtime-endpoint", host.Endpoint})
	writeErr := errors.New("output is closed")
	cmd.SetOut(failingWriter{err: writeErr})
	if err := cmd.ExecuteContext(t.Context()); !errors.Is(err, writeErr) {
		t.Fatalf("lost output error: %v", err)
	}

	waitForVersionCleanup(t, host)

	if _, err := host.Runtime.Run(t.Context(), api.NewSource("survivor.fql", "RETURN DEMO::IDENTITY(42)")); err != nil {
		t.Fatalf("output failure closed hosted runtime: %v", err)
	}
}

func waitForVersionCleanup(t *testing.T, host *wirehost.Host) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	if err := host.WaitForClientsClosed(ctx); err != nil {
		t.Fatalf("client transport was not released: %v", err)
	}
}

func executeVersion(ctx context.Context, store *config.Store, args []string) (string, string, error) {
	cmd := New(store)
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	cmd.SetArgs(args)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := cmd.ExecuteContext(ctx)

	return stdout.String(), stderr.String(), err
}

func TestWireVersionCommandCancellation(t *testing.T) {
	started, finished := make(chan struct{}), make(chan struct{})
	host := wirehost.NewWithVersionFunc(t, func(ctx context.Context) (api.Version, error) {
		close(started)
		defer close(finished)
		<-ctx.Done()

		return "", ctx.Err()
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	store := testutil.NewStore(t)
	done := make(chan error, 1)
	go func() {
		stdout, _, err := executeVersion(ctx, store, []string{"--runtime", "wire", "--runtime-endpoint", host.Endpoint})
		if stdout != "" {
			err = errors.Join(err, fmt.Errorf("canceled metadata produced output: %q", stdout))
		}

		done <- err
	}()
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("host did not receive metadata request")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("lost command cancellation: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("version command did not stop")
	}

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("hosted metadata request did not settle")
	}

	waitForVersionCleanup(t, host)
}

func TestVersionRegistersOnlyRuntimeSelectionFlags(t *testing.T) {
	cmd := New(testutil.NewStore(t))
	for _, name := range []string{config.ExecRuntime, config.ExecRuntimeEndpoint, config.ExecRuntimeConnectTimeout} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("missing selection flag --%s", name)
		}
	}

	for _, name := range []string{config.ExecWithBrowser, config.ExecBrowserAddress, config.ExecProxy, config.ExecUserAgent, config.PolicyFSRoot, config.PolicyHTTPFollowRedirects} {
		if cmd.Flags().Lookup(name) != nil {
			t.Errorf("unexpected execution flag --%s", name)
		}
	}
}
