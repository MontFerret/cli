package config

import (
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestRuntimeMetadataOptionsDetectUnboundConfiguration(t *testing.T) {
	for _, test := range []struct{ key, value string }{
		{PolicyFSReadOnly, "false"}, {PolicyHTTPFollowRedirects, "true"}, {ExecWithBrowser, "false"},
	} {
		t.Run(test.key, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			homedir.Reset()
			t.Cleanup(homedir.Reset)
			store, err := NewStore("ferret", "test")
			if err != nil {
				t.Fatal(err)
			}

			cmd := &cobra.Command{}
			cmd.Flags().String(ExecRuntime, runtime.DefaultRuntime, "")
			cmd.Flags().String(ExecRuntimeEndpoint, "", "")
			if err := cmd.ParseFlags([]string{"--runtime=wire", "--runtime-endpoint=tcp://127.0.0.1:1"}); err != nil {
				t.Fatal(err)
			}

			store.BindFlags(cmd)
			if err := runtime.ValidateOptions(store.GetRuntimeOptions()); err != nil {
				t.Fatalf("untouched defaults rejected: %v", err)
			}

			t.Setenv("FERRET_"+strings.ToUpper(strings.ReplaceAll(test.key, "-", "_")), test.value)
			if err := runtime.ValidateOptions(store.GetRuntimeOptions()); err == nil {
				t.Fatal("explicit environmental default was ignored without a registered flag")
			}
		})
	}
}
