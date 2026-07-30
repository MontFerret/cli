package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"

	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestConfigCommandPolicyRoundTrip(t *testing.T) {
	home := t.TempDir()
	store := newConfigCommandTestStore(t, home)
	values := []struct {
		key   string
		value string
	}{
		{config.PolicyFSRoot, "./fixtures"},
		{config.PolicyFSReadOnly, "true"},
		{config.PolicyHTTPAllowedSchemes, "http,https"},
		{config.PolicyHTTPAllowedMethods, "GET,POST"},
		{config.PolicyHTTPAllowedHosts, "api.example.com,cdn.example.com"},
		{config.PolicyHTTPBlockedHosts, "blocked.example.com"},
		{config.PolicyHTTPAllowLocalhost, "true"},
		{config.PolicyHTTPAllowPrivateNetworks, "true"},
		{config.PolicyHTTPAllowLinkLocal, "true"},
		{config.PolicyHTTPDefaultHeaders, `{"X-Trace":"config"}`},
		{config.PolicyHTTPBlockedRequestHeaders, "Authorization,X-Secret"},
		{config.PolicyHTTPTimeout, "5s"},
		{config.PolicyHTTPNoTimeout, "false"},
		{config.PolicyHTTPMaxRequestSize, "1024"},
		{config.PolicyHTTPUnlimitedRequestSize, "false"},
		{config.PolicyHTTPMaxResponseSize, "2048"},
		{config.PolicyHTTPUnlimitedResponseSize, "false"},
		{config.PolicyHTTPMaxResponseHeaderSize, "4096"},
		{config.PolicyHTTPFollowRedirects, "false"},
		{config.PolicyHTTPMaxRedirects, "3"},
	}

	for _, value := range values {
		if _, err := executeConfigCommand(store, "set", value.key, value.value); err != nil {
			t.Fatalf("set %s: %v", value.key, err)
		}
	}

	homedir.Reset()
	store = newConfigCommandTestStore(t, home)

	for _, value := range values {
		out, err := executeConfigCommand(store, "get", value.key)
		if err != nil {
			t.Fatalf("get %s: %v", value.key, err)
		}
		if got := strings.TrimSpace(out); got != value.value {
			t.Fatalf("get %s: expected %q, got %q", value.key, value.value, got)
		}
	}

	list, err := executeConfigCommand(store, "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		if !strings.Contains(list, fmt.Sprintf("%s: %s", value.key, value.value)) {
			t.Fatalf("config list does not contain %s: %s\n%s", value.key, value.value, list)
		}
	}

	runCommand := RunCommand(store)
	store.BindFlags(runCommand)
	opts, err := runtimeOptionsFromCommand(runCommand, store)
	if err != nil {
		t.Fatal(err)
	}
	if opts.FSPolicy == nil || opts.FSPolicy.Root != "./fixtures" || !opts.FSPolicy.ReadOnly {
		t.Fatalf("unexpected filesystem policy: %#v", opts.FSPolicy)
	}
	if len(opts.HTTPPolicy) == 0 {
		t.Fatal("expected persisted HTTP policy options")
	}
	if err := cliruntime.ValidateOptions(opts); err != nil {
		t.Fatalf("expected persisted policy to validate, got %v", err)
	}
}

func TestPolicyConfigKeysCoverSupportedPolicyFlags(t *testing.T) {
	for _, key := range config.Flags {
		if strings.HasPrefix(key, "policy-") && !isPolicyConfigKey(key) {
			t.Fatalf("supported policy key %q is missing config validation", key)
		}
	}
}

func TestConfigCommandRejectsInvalidPolicyValuesWithoutWriting(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "blank filesystem root", key: config.PolicyFSRoot, value: " \t "},
		{name: "filesystem boolean", key: config.PolicyFSReadOnly, value: "sometimes"},
		{name: "scheme", key: config.PolicyHTTPAllowedSchemes, value: "not a scheme"},
		{name: "method", key: config.PolicyHTTPAllowedMethods, value: "bad method"},
		{name: "allowed host", key: config.PolicyHTTPAllowedHosts, value: "bad host"},
		{name: "blocked host", key: config.PolicyHTTPBlockedHosts, value: "bad host"},
		{name: "boolean", key: config.PolicyHTTPAllowLocalhost, value: "sometimes"},
		{name: "headers JSON", key: config.PolicyHTTPDefaultHeaders, value: `{"X-Trace":1}`},
		{name: "blocked header", key: config.PolicyHTTPBlockedRequestHeaders, value: "bad header"},
		{name: "duration", key: config.PolicyHTTPTimeout, value: "later"},
		{name: "request size", key: config.PolicyHTTPMaxRequestSize, value: "-1"},
		{name: "response size", key: config.PolicyHTTPMaxResponseSize, value: "-1"},
		{name: "response header size", key: config.PolicyHTTPMaxResponseHeaderSize, value: "-1"},
		{name: "redirect count", key: config.PolicyHTTPMaxRedirects, value: "-1"},
		{name: "list CSV", key: config.PolicyHTTPAllowedHosts, value: `"unterminated`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			store := newConfigCommandTestStore(t, home)
			if err := store.Set(config.ExecRuntime, "builtin"); err != nil {
				t.Fatal(err)
			}
			before := readConfigCommandTestFile(t, home)

			args := []string{"set", tt.key, tt.value}
			if strings.HasPrefix(tt.value, "-") {
				args = []string{"set", tt.key, "--", tt.value}
			}

			_, err := executeConfigCommand(store, args...)
			if err == nil || !strings.Contains(err.Error(), "invalid policy configuration") {
				t.Fatalf("expected policy validation error, got %v", err)
			}

			got, getErr := store.Get(tt.key)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if got != nil {
				t.Fatalf("expected rejected value not to be stored, got %v", got)
			}

			runtimeValue, getErr := store.Get(config.ExecRuntime)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if runtimeValue != "builtin" {
				t.Fatalf("expected existing config to remain unchanged, got %v", runtimeValue)
			}

			after := readConfigCommandTestFile(t, home)
			if !bytes.Equal(before, after) {
				t.Fatalf("expected rejected value not to change config file\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

func TestConfigCommandRejectsPolicyConflictsWithoutWriting(t *testing.T) {
	tests := []struct {
		name           string
		existingKey    string
		existingValue  string
		candidateKey   string
		candidateValue string
	}{
		{
			name:           "timeout and no timeout",
			existingKey:    config.PolicyHTTPTimeout,
			existingValue:  "1s",
			candidateKey:   config.PolicyHTTPNoTimeout,
			candidateValue: "true",
		},
		{
			name:           "no timeout and timeout",
			existingKey:    config.PolicyHTTPNoTimeout,
			existingValue:  "true",
			candidateKey:   config.PolicyHTTPTimeout,
			candidateValue: "1s",
		},
		{
			name:           "request limit and unlimited request",
			existingKey:    config.PolicyHTTPMaxRequestSize,
			existingValue:  "1",
			candidateKey:   config.PolicyHTTPUnlimitedRequestSize,
			candidateValue: "true",
		},
		{
			name:           "unlimited request and request limit",
			existingKey:    config.PolicyHTTPUnlimitedRequestSize,
			existingValue:  "true",
			candidateKey:   config.PolicyHTTPMaxRequestSize,
			candidateValue: "1",
		},
		{
			name:           "response limit and unlimited response",
			existingKey:    config.PolicyHTTPMaxResponseSize,
			existingValue:  "1",
			candidateKey:   config.PolicyHTTPUnlimitedResponseSize,
			candidateValue: "true",
		},
		{
			name:           "unlimited response and response limit",
			existingKey:    config.PolicyHTTPUnlimitedResponseSize,
			existingValue:  "true",
			candidateKey:   config.PolicyHTTPMaxResponseSize,
			candidateValue: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			store := newConfigCommandTestStore(t, home)
			if err := store.Set(tt.existingKey, tt.existingValue); err != nil {
				t.Fatal(err)
			}
			before := readConfigCommandTestFile(t, home)

			_, err := executeConfigCommand(store, "set", tt.candidateKey, tt.candidateValue)
			if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
				t.Fatalf("expected conflict error, got %v", err)
			}

			got, getErr := store.Get(tt.candidateKey)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if got != nil {
				t.Fatalf("expected conflicting value not to be stored, got %v", got)
			}

			existing, getErr := store.Get(tt.existingKey)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if existing != tt.existingValue {
				t.Fatalf("expected existing value %q, got %v", tt.existingValue, existing)
			}

			after := readConfigCommandTestFile(t, home)
			if !bytes.Equal(before, after) {
				t.Fatalf("expected conflict not to change config file\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

func TestConfigCommandUnsetRestoresImplicitPolicyDefaults(t *testing.T) {
	home := t.TempDir()
	store := newConfigCommandTestStore(t, home)

	if _, err := executeConfigCommand(store, "set", config.PolicyFSRoot, "./fixtures"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeConfigCommand(store, "set", config.PolicyHTTPAllowLocalhost, "true"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeConfigCommand(store, "unset", config.PolicyFSRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := executeConfigCommand(store, "unset", config.PolicyHTTPAllowLocalhost); err != nil {
		t.Fatal(err)
	}
	if _, err := executeConfigCommand(store, "unset", config.PolicyHTTPAllowLocalhost); err != nil {
		t.Fatalf("expected repeated unset to succeed, got %v", err)
	}

	homedir.Reset()
	store = newConfigCommandTestStore(t, home)
	runCommand := RunCommand(store)
	store.BindFlags(runCommand)
	opts, err := runtimeOptionsFromCommand(runCommand, store)
	if err != nil {
		t.Fatal(err)
	}
	if opts.FSPolicy != nil {
		t.Fatalf("expected implicit filesystem defaults, got %#v", opts.FSPolicy)
	}
	if len(opts.HTTPPolicy) != 0 {
		t.Fatalf("expected implicit HTTP defaults, got %d options", len(opts.HTTPPolicy))
	}
}

func TestConfigCommandUnsetRejectsUnsupportedKey(t *testing.T) {
	store := newConfigCommandTestStore(t, t.TempDir())

	_, err := executeConfigCommand(store, "unset", "not-a-config-key")
	if err == nil || !strings.Contains(err.Error(), config.ErrInvalidFlag.Error()) {
		t.Fatalf("expected invalid flag error, got %v", err)
	}
}

func TestConfigCommandSetAndUnsetRequireExactArguments(t *testing.T) {
	store := newConfigCommandTestStore(t, t.TempDir())

	for _, args := range [][]string{
		{"set", config.PolicyFSRoot},
		{"set", config.PolicyFSRoot, ".", "extra"},
		{"unset"},
		{"unset", config.PolicyFSRoot, "extra"},
	} {
		if _, err := executeConfigCommand(store, args...); err == nil {
			t.Fatalf("expected argument error for %v", args)
		}
	}
}

func executeConfigCommand(store *config.Store, args ...string) (string, error) {
	command := ConfigCommand(store)
	out := new(bytes.Buffer)
	command.SetOut(out)
	command.SetErr(out)
	command.SetArgs(args)

	err := command.Execute()

	return out.String(), err
}

func readConfigCommandTestFile(t *testing.T, home string) []byte {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join(home, ".ferret", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	return contents
}

func newConfigCommandTestStore(t *testing.T, home string) *config.Store {
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
