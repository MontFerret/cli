package cmd

import (
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/MontFerret/ferret/v2/pkg/runtime"

	"github.com/MontFerret/cli/browser"
	"github.com/MontFerret/cli/config"
	"github.com/MontFerret/cli/repl"
	cliruntime "github.com/MontFerret/cli/runtime"
)

func ReplCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repl",
		Short: "Launch interactive FQL shell",
		Args:  cobra.NoArgs,
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			paramFlag, err := cmd.Flags().GetStringArray(ParamFlag)

			if err != nil {
				return err
			}

			params, err := parseParams(paramFlag)

			if err != nil {
				return err
			}

			store := config.From(cmd.Context())

			rtOpts := store.GetRuntimeOptions()

			if rtOpts.WithBrowser {
				brOpts := store.GetBrowserOptions()
				brOpts.Detach = true
				brOpts.Headless = rtOpts.WithHeadlessBrowser

				if rtOpts.BrowserAddress != "" {
					u, err := url.Parse(rtOpts.BrowserAddress)

					if err != nil {
						return runtime.Error(err, "invalid browser address")
					}

					if u.Port() != "" {
						p, err := strconv.ParseUint(u.Port(), 10, 64)

						if err != nil {
							return err
						}

						brOpts.Port = p
					}
				}

				pid, err := browser.Open(cmd.Context(), brOpts)

				if err != nil {
					return err
				}

				defer browser.Close(cmd.Context(), brOpts, pid)
			}

			return repl.Start(cmd.Context(), rtOpts, params)
		},
	}

	cmd.Flags().StringArrayP(ParamFlag, "p", []string{}, "Query bind parameter (--param=foo:\"bar\", --param=id:1)")
	cmd.Flags().StringP(config.ExecRuntime, "r", cliruntime.DefaultRuntime, "Ferret runtime type (\"builtin\"|$url)")
	cmd.Flags().String(config.ExecProxy, "x", "Proxy server address")
	cmd.Flags().String(config.ExecUserAgent, "a", "User agent header")
	cmd.Flags().StringP(config.ExecBrowserAddress, "d", cliruntime.DefaultBrowser, "Browser debugger address")
	cmd.Flags().BoolP(config.ExecWithBrowser, "B", false, "Open browser for script execution")
	cmd.Flags().BoolP(config.ExecWithBrowserHeadless, "b", false, "Open browser for script execution in headless mode")
	cmd.Flags().BoolP(config.ExecKeepCookies, "c", false, "Keep cookies between queries")

	return cmd
}
