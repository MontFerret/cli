package execution_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	debugcmd "github.com/MontFerret/cli/v2/cmd/internal/debug"
	"github.com/MontFerret/cli/v2/cmd/internal/execution"
	replcmd "github.com/MontFerret/cli/v2/cmd/internal/repl"
	runcmd "github.com/MontFerret/cli/v2/cmd/internal/run"
	versioncmd "github.com/MontFerret/cli/v2/cmd/internal/version"
	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
	ferrethttp "github.com/MontFerret/ferret/v2/pkg/net/http"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

var httpPolicyFlagNames = []string{
	config.PolicyHTTPAllowedSchemes,
	config.PolicyHTTPAllowedMethods,
	config.PolicyHTTPAllowedHosts,
	config.PolicyHTTPBlockedHosts,
	config.PolicyHTTPAllowLocalhost,
	config.PolicyHTTPAllowPrivateNetworks,
	config.PolicyHTTPAllowLinkLocal,
	config.PolicyHTTPDefaultHeaders,
	config.PolicyHTTPBlockedRequestHeaders,
	config.PolicyHTTPTimeout,
	config.PolicyHTTPNoTimeout,
	config.PolicyHTTPMaxRequestSize,
	config.PolicyHTTPUnlimitedRequestSize,
	config.PolicyHTTPMaxResponseSize,
	config.PolicyHTTPUnlimitedResponseSize,
	config.PolicyHTTPMaxResponseHeaderSize,
	config.PolicyHTTPFollowRedirects,
	config.PolicyHTTPMaxRedirects,
}

func TestHTTPPolicyFlagsAppearOnExecutionCommands(t *testing.T) {
	store := new(config.Store)
	commands := []*cobra.Command{
		runcmd.New(store),
		replcmd.New(store),
		debugcmd.New(store),
	}

	for _, command := range commands {
		t.Run(command.Name(), func(t *testing.T) {
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&output)

			if err := command.Help(); err != nil {
				t.Fatal(err)
			}

			for _, name := range httpPolicyFlagNames {
				if command.Flags().Lookup(name) == nil {
					t.Fatalf("expected --%s to be registered", name)
				}
				if !strings.Contains(output.String(), "--"+name) {
					t.Fatalf("expected help to list --%s", name)
				}
			}
		})
	}
}

func TestHTTPPolicyFlagsDoNotAppearOnVersion(t *testing.T) {
	command := versioncmd.New(new(config.Store))

	for _, name := range httpPolicyFlagNames {
		if command.Flags().Lookup(name) != nil {
			t.Fatalf("expected --%s to be absent", name)
		}
	}
}

func TestHTTPPolicyFlagDefaultsDoNotOverrideFerretDefaults(t *testing.T) {
	command := &cobra.Command{Use: "policy-test"}
	execution.AddHTTPPolicyFlags(command)

	options, err := execution.HTTPPolicyOptionsFromCommand(command)
	if err != nil {
		t.Fatal(err)
	}
	if len(options) != 0 {
		t.Fatalf("expected no explicit HTTP policy options, got %d", len(options))
	}
}

func TestHTTPPolicyFlagsRejectInvalidFerretPolicy(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		wantTarget string
		wantReason string
	}{
		{name: "allowed scheme", arg: "--policy-http-allowed-schemes=not a scheme", wantTarget: "allowed schemes", wantReason: "must be a valid URL scheme"},
		{name: "allowed method", arg: "--policy-http-allowed-methods=bad method", wantTarget: "allowed methods", wantReason: "must be a non-empty HTTP method token"},
		{name: "allowed host", arg: "--policy-http-allowed-hosts=bad host", wantTarget: "allowed hosts", wantReason: "must be a valid DNS name"},
		{name: "blocked host", arg: "--policy-http-blocked-hosts=bad host", wantTarget: "blocked hosts", wantReason: "must be a valid DNS name"},
		{name: "default header", arg: `--policy-http-default-headers={"Host":"example.test"}`, wantTarget: "default headers", wantReason: "request header is reserved for the transport"},
		{name: "blocked header", arg: "--policy-http-blocked-request-headers=bad header", wantTarget: "blocked request headers", wantReason: "name is not a valid HTTP field-name token"},
		{name: "timeout", arg: "--policy-http-timeout=-1s", wantTarget: "timeout", wantReason: "must be non-negative"},
		{name: "request size", arg: "--policy-http-max-request-size=-1", wantTarget: "max request size", wantReason: "must be non-negative"},
		{name: "response size", arg: "--policy-http-max-response-size=-1", wantTarget: "max response size", wantReason: "must be non-negative"},
		{name: "response header size", arg: "--policy-http-max-response-header-size=-1", wantTarget: "max response header size", wantReason: "must be non-negative"},
		{name: "redirect count", arg: "--policy-http-max-redirects=-1", wantTarget: "max redirects", wantReason: "must be non-negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPPolicyArguments(t, tt.arg)
			if !errors.Is(err, ferrethttp.ErrInvalidPolicyConfiguration) ||
				!strings.Contains(err.Error(), tt.wantTarget) ||
				!strings.Contains(err.Error(), tt.wantReason) {
				t.Fatalf("expected %s validation error, got %v", tt.wantTarget, err)
			}
		})
	}
}

func TestHTTPPolicyFlagsRejectConflictingLimits(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "timeout",
			args: []string{"--policy-http-timeout=1s", "--policy-http-no-timeout"},
			want: "--policy-http-no-timeout cannot be combined with --policy-http-timeout",
		},
		{
			name: "request size",
			args: []string{"--policy-http-max-request-size=1", "--policy-http-unlimited-request-size"},
			want: "--policy-http-unlimited-request-size cannot be combined with --policy-http-max-request-size",
		},
		{
			name: "response size",
			args: []string{"--policy-http-max-response-size=1", "--policy-http-unlimited-response-size"},
			want: "--policy-http-unlimited-response-size cannot be combined with --policy-http-max-response-size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPPolicyArguments(t, tt.args...)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestHTTPPolicyFlagsRejectInvalidDefaultHeadersJSON(t *testing.T) {
	err := validateHTTPPolicyArguments(t, `--policy-http-default-headers={"X-Trace":1}`)
	if err == nil || !strings.Contains(err.Error(), "expected a JSON object of string values") {
		t.Fatalf("expected default-header JSON error, got %v", err)
	}
}

func TestHTTPPolicyEnvironmentValuesReachBuiltinRuntime(t *testing.T) {
	t.Setenv("FERRET_POLICY_HTTP_ALLOW_LOCALHOST", "true")
	t.Setenv("FERRET_POLICY_HTTP_DEFAULT_HEADERS", `{"X-Ferret-Policy":"environment"}`)

	store := newHTTPPolicyTestStore(t, t.TempDir())
	command := runcmd.New(store)
	store.BindFlags(command)

	opts, err := execution.OptionsFromCommand(command, store)
	if err != nil {
		t.Fatal(err)
	}

	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Get("X-Ferret-Policy")
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	out, err := cliruntime.Run(
		context.Background(),
		opts,
		source.NewAnonymous(fmt.Sprintf("RETURN TO_STRING(IO::NET::HTTP::GET(%q))", server.URL)),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	if got := <-requests; got != "environment" {
		t.Fatalf("expected environment default header, got %q", got)
	}
}

func TestHTTPPolicyFlagValuesReachBuiltinRuntime(t *testing.T) {
	command := &cobra.Command{Use: "policy-test"}
	execution.AddHTTPPolicyFlags(command)
	if err := command.Flags().Parse([]string{
		"--policy-http-allow-localhost",
		`--policy-http-default-headers={"X-Ferret-Policy":"flag"}`,
	}); err != nil {
		t.Fatal(err)
	}

	policy, err := execution.HTTPPolicyOptionsFromCommand(command)
	if err != nil {
		t.Fatal(err)
	}

	opts := cliruntime.NewDefaultOptions()
	opts.HTTPPolicy = policy

	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Get("X-Ferret-Policy")
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	out, err := cliruntime.Run(
		context.Background(),
		opts,
		source.NewAnonymous(fmt.Sprintf("RETURN TO_STRING(IO::NET::HTTP::GET(%q))", server.URL)),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	if got := <-requests; got != "flag" {
		t.Fatalf("expected flag default header, got %q", got)
	}
}

func TestHTTPPolicyConfigValuesReachRuntimeOptions(t *testing.T) {
	home := t.TempDir()
	store := newHTTPPolicyTestStore(t, home)
	if err := store.Set(config.PolicyHTTPAllowLocalhost, "true"); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(config.PolicyHTTPAllowedMethods, "GET,POST"); err != nil {
		t.Fatal(err)
	}

	homedir.Reset()
	store = newHTTPPolicyTestStore(t, home)
	command := runcmd.New(store)
	store.BindFlags(command)

	opts, err := execution.OptionsFromCommand(command, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.HTTPPolicy) != 2 {
		t.Fatalf("expected two configured HTTP policy options, got %d", len(opts.HTTPPolicy))
	}
	if err := cliruntime.ValidateOptions(opts); err != nil {
		t.Fatalf("expected configured policy to validate, got %v", err)
	}
}

func validateHTTPPolicyArguments(t *testing.T, args ...string) error {
	t.Helper()

	command := &cobra.Command{Use: "policy-test"}
	execution.AddHTTPPolicyFlags(command)

	if err := command.Flags().Parse(args); err != nil {
		return err
	}

	options, err := execution.HTTPPolicyOptionsFromCommand(command)
	if err != nil {
		return err
	}

	runtimeOptions := cliruntime.NewDefaultOptions()
	runtimeOptions.HTTPPolicy = options

	return cliruntime.ValidateOptions(runtimeOptions)
}

func newHTTPPolicyTestStore(t *testing.T, home string) *config.Store {
	t.Helper()

	t.Setenv("HOME", home)
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	store, err := config.NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	return store
}
