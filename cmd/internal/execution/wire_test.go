package execution_test

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/cmd/internal/execution"
	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
	"github.com/MontFerret/cli/v2/pkg/config"
)

func TestWireExplicitConfiguration(t *testing.T) {
	for _, extra := range [][]string{
		{}, {"--browser-open=false"}, {"--browser-headless=false"}, {"--browser-cookies=false"},
		{"--browser-address=http://127.0.0.1:9222"}, {"--proxy="}, {"--user-agent="},
		{"--policy-fs-read-only=false"}, {"--policy-http-follow-redirects=true"},
	} {
		t.Run(strings.Join(extra, " "), func(t *testing.T) {
			store := testutil.NewStore(t)
			cmd := &cobra.Command{}
			execution.AddRuntimeFlags(cmd)
			args := append([]string{"--runtime=wire", "--runtime-endpoint=tcp://127.0.0.1:123"}, extra...)
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatal(err)
			}

			store.BindFlags(cmd)
			opts, err := execution.OptionsFromCommand(cmd, store)
			if (err != nil) != (len(extra) != 0) {
				t.Fatalf("explicit options %v: %v", extra, err)
			}

			if len(extra) == 0 && opts.ConnectTimeout != 5*time.Second {
				t.Fatalf("default timeout = %v", opts.ConnectTimeout)
			}
		})
	}
}

func TestWireConfigPrecedenceAndDefaults(t *testing.T) {
	store := testutil.NewStore(t)
	if err := store.Set(config.ExecRuntime, "wire"); err != nil {
		t.Fatal(err)
	}

	if err := store.Set(config.ExecRuntimeEndpoint, "tcp://127.0.0.1:111"); err != nil {
		t.Fatal(err)
	}

	if err := store.Set(config.ExecRuntimeConnectTimeout, "9s"); err != nil {
		t.Fatal(err)
	}

	store, err := config.NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("FERRET_RUNTIME_ENDPOINT", "tcp://127.0.0.1:222")
	t.Setenv("FERRET_RUNTIME_CONNECT_TIMEOUT", "7s")
	cmd := &cobra.Command{}
	execution.AddRuntimeFlags(cmd)
	store.BindFlags(cmd)
	opts, err := execution.OptionsFromCommand(cmd, store)
	if err != nil || opts.Endpoint != "tcp://127.0.0.1:222" || opts.ConnectTimeout != 7*time.Second {
		t.Fatalf("environment precedence: %+v error=%v", opts, err)
	}

	cmd = &cobra.Command{}
	execution.AddRuntimeFlags(cmd)
	if err := cmd.ParseFlags([]string{"--runtime-endpoint=tcp://127.0.0.1:333", "--runtime-connect-timeout=3s"}); err != nil {
		t.Fatal(err)
	}

	store.BindFlags(cmd)
	opts, err = execution.OptionsFromCommand(cmd, store)
	if err != nil || opts.Endpoint != "tcp://127.0.0.1:333" || opts.ConnectTimeout != 3*time.Second {
		t.Fatalf("flag precedence: %+v error=%v", opts, err)
	}
}

func TestWireConfigurationRejections(t *testing.T) {
	for _, args := range [][]string{
		{"--runtime=wire"},
		{"--runtime=builtin", "--runtime-endpoint=tcp://127.0.0.1:1"},
		{"--runtime=https://worker.example", "--runtime-connect-timeout=5s"},
		{"--runtime=wire", "--runtime-endpoint=tcp://127.0.0.1:1", "--runtime-connect-timeout=0s"},
		{"--runtime=wire", "--runtime-endpoint=tcp://127.0.0.1:1", "--runtime-connect-timeout=-1s"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			store := testutil.NewStore(t)
			cmd := &cobra.Command{}
			execution.AddRuntimeFlags(cmd)
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatal(err)
			}

			store.BindFlags(cmd)
			if _, err := execution.OptionsFromCommand(cmd, store); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestWireRejectsConfiguredPolicyAndBrowserDefaults(t *testing.T) {
	for _, test := range []struct{ key, value string }{
		{config.ExecWithBrowser, "false"},
		{config.ExecBrowserAddress, "http://127.0.0.1:9222"},
		{config.PolicyFSReadOnly, "false"},
		{config.PolicyHTTPFollowRedirects, "true"},
	} {
		for _, environment := range []bool{false, true} {
			t.Run(test.key+"/"+map[bool]string{false: "config", true: "environment"}[environment], func(t *testing.T) {
				store := testutil.NewStore(t)
				if environment {
					t.Setenv("FERRET_"+strings.ToUpper(strings.ReplaceAll(test.key, "-", "_")), test.value)
				} else {
					if err := store.Set(test.key, test.value); err != nil {
						t.Fatal(err)
					}
				}

				cmd := &cobra.Command{}
				execution.AddRuntimeFlags(cmd)
				if err := cmd.ParseFlags([]string{"--runtime=wire", "--runtime-endpoint=tcp://127.0.0.1:1"}); err != nil {
					t.Fatal(err)
				}

				store.BindFlags(cmd)
				if _, err := execution.OptionsFromCommand(cmd, store); err == nil {
					t.Fatal("explicit builtin configuration accepted")
				}
			})
		}
	}
}
