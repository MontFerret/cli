package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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

			rt, err := runtime.New(store.GetRuntimeOptions())

			if err != nil {
				return err
			}

			stat, _ := os.Stdin.Stat()

			if (stat.Mode() & os.ModeCharDevice) == 0 {
				// check whether the app is getting a query via standard input
				std := bufio.NewReader(os.Stdin)

				content, err := ioutil.ReadAll(std)

				if err != nil {
					return err
				}

				return execScript(cmd, rt, params, string(content))
			}

			if len(args) == 0 {
				return startRepl(cmd, rt, params)
			}

			content, err := os.ReadFile(args[0])

			if err != nil {
				return err
			}

			return execScript(cmd, rt, params, string(content))
		},
	}

	cmd.Flags().StringArrayP(ExecParamFlag, "p", []string{}, "Query bind parameter (--param=foo:\"bar\", --param=id:1)")
	cmd.Flags().StringP(config.RuntimeType, "r", runtime.DefaultRuntime, "Ferret runtime type (\"builtin\"|$url)")
	cmd.Flags().StringP(config.RuntimeCDPAddress, "b", runtime.DefaultBrowser, "Browser debugger address")
	cmd.Flags().String(config.RuntimeProxy, "", "Proxy server address")
	cmd.Flags().String(config.RuntimeUserAgent, "", "User agent header")
	cmd.Flags().Bool(config.RuntimeKeepCookies, false, "Keep cookies between queries")

	return cmd
}

func startRepl(cmd *cobra.Command, rt runtime.Runtime, params map[string]interface{}) error {
	return repl.Start(cmd.Context(), rt, params)
}

func execScript(cmd *cobra.Command, rt runtime.Runtime, params map[string]interface{}, query string) error {
	out, err := rt.Run(cmd.Context(), query, params)

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
