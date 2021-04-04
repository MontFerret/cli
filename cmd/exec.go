package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/MontFerret/cli/browser"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/MontFerret/ferret/pkg/runtime/core"
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/config"
	"github.com/MontFerret/cli/repl"
	"github.com/MontFerret/cli/runtime"
)

const (
	ExecParamFlag = "param"
)

// RumCommand command to execute FQL scripts
func ExecCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Executes FQL script",
		Args:  cobra.MinimumNArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			paramFlag, err := cmd.Flags().GetStringArray(ExecParamFlag)

			if err != nil {
				return err
			}

			params, err := parseExecParams(paramFlag)

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
						return errors.Wrap(err, "invalid browser address")
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

			stat, _ := os.Stdin.Stat()

			if (stat.Mode() & os.ModeCharDevice) == 0 {
				// check whether the app is getting a query via standard input
				std := bufio.NewReader(os.Stdin)

				content, err := ioutil.ReadAll(std)

				if err != nil {
					return err
				}

				return execScript(cmd, rtOpts, params, string(content))
			}

			if len(args) == 0 {
				return startRepl(cmd, rtOpts, params)
			}

			content, err := os.ReadFile(args[0])

			if err != nil {
				return err
			}

			return execScript(cmd, rtOpts, params, string(content))
		},
	}

	cmd.Flags().StringArrayP(ExecParamFlag, "p", []string{}, "Query bind parameter (--param=foo:\"bar\", --param=id:1)")
	cmd.Flags().StringP(config.ExecRuntime, "r", runtime.DefaultRuntime, "Ferret runtime type (\"builtin\"|$url)")
	cmd.Flags().String(config.ExecProxy, "x", "Proxy server address")
	cmd.Flags().String(config.ExecUserAgent, "a", "User agent header")
	cmd.Flags().StringP(config.ExecBrowserAddress, "d", runtime.DefaultBrowser, "Browser debugger address")
	cmd.Flags().BoolP(config.ExecWithBrowser, "B", false, "Open browser for script execution")
	cmd.Flags().BoolP(config.ExecWithBrowserHeadless, "b", false, "Open browser for script execution in headless mode")
	cmd.Flags().BoolP(config.ExecKeepCookies, "c", false, "Keep cookies between queries")

	return cmd
}

func startRepl(cmd *cobra.Command, opts runtime.Options, params map[string]interface{}) error {
	return repl.Start(cmd.Context(), opts, params)
}

func execScript(cmd *cobra.Command, opts runtime.Options, params map[string]interface{}, query string) error {
	out, err := runtime.Run(cmd.Context(), opts, query, params)

	if err != nil {
		return err
	}

	fmt.Println(string(out))

	return err
}

func parseExecParams(flags []string) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	for _, entry := range flags {
		pair := strings.SplitN(entry, ":", 2)

		if len(pair) < 2 {
			return nil, core.Error(core.ErrInvalidArgument, entry)
		}

		var value interface{}
		key := pair[0]

		err := json.Unmarshal([]byte(pair[1]), &value)

		if err != nil {
			fmt.Println(pair[1])
			return nil, err
		}

		res[key] = value
	}

	return res, nil
}
