package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
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
		RunCommand(store),
		ReplCommand(store),
		DebugCommand(store),
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
	command := VersionCommand(new(config.Store))

	for _, name := range httpPolicyFlagNames {
		if command.Flags().Lookup(name) != nil {
			t.Fatalf("expected --%s to be absent", name)
		}
	}
}

func TestHTTPPolicyFlagDefaultsDoNotOverrideFerretDefaults(t *testing.T) {
	command := &cobra.Command{Use: "policy-test"}
	addHTTPPolicyFlags(command)

	options, err := httpPolicyOptionsFromCommand(command)
	if err != nil {
		t.Fatal(err)
	}
	if len(options) != 0 {
		t.Fatalf("expected no explicit HTTP policy options, got %d", len(options))
	}
}

func TestHTTPPolicyFlagsRejectInvalidFerretPolicy(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{name: "allowed scheme", arg: "--policy-http-allowed-schemes=not a scheme", want: "WithAllowedSchemes"},
		{name: "allowed method", arg: "--policy-http-allowed-methods=bad method", want: "WithAllowedMethods"},
		{name: "allowed host", arg: "--policy-http-allowed-hosts=bad host", want: "WithAllowedHosts"},
		{name: "blocked host", arg: "--policy-http-blocked-hosts=bad host", want: "WithBlockedHosts"},
		{name: "default header", arg: `--policy-http-default-headers={"Host":"example.test"}`, want: "WithDefaultHeaders"},
		{name: "blocked header", arg: "--policy-http-blocked-request-headers=bad header", want: "WithBlockedRequestHeaders"},
		{name: "timeout", arg: "--policy-http-timeout=-1s", want: "WithTimeout"},
		{name: "request size", arg: "--policy-http-max-request-size=-1", want: "WithMaxRequestSize"},
		{name: "response size", arg: "--policy-http-max-response-size=-1", want: "WithMaxResponseSize"},
		{name: "response header size", arg: "--policy-http-max-response-header-size=-1", want: "WithMaxResponseHeaderSize"},
		{name: "redirect count", arg: "--policy-http-max-redirects=-1", want: "WithMaxRedirects"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPPolicyArguments(t, tt.arg)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %s error, got %v", tt.want, err)
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
	command := RunCommand(store)
	store.BindFlags(command)

	opts, err := runtimeOptionsFromCommand(command, store)
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
	addHTTPPolicyFlags(command)
	if err := command.Flags().Parse([]string{
		"--policy-http-allow-localhost",
		`--policy-http-default-headers={"X-Ferret-Policy":"flag"}`,
	}); err != nil {
		t.Fatal(err)
	}

	policy, err := httpPolicyOptionsFromCommand(command)
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
	command := RunCommand(store)
	store.BindFlags(command)

	opts, err := runtimeOptionsFromCommand(command, store)
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
	addHTTPPolicyFlags(command)

	if err := command.Flags().Parse(args); err != nil {
		return err
	}

	options, err := httpPolicyOptionsFromCommand(command)
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
