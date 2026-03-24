package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/MontFerret/ferret/v2/pkg/file"

	"github.com/MontFerret/cli/pkg/browser"
	"github.com/MontFerret/cli/pkg/config"
	cliruntime "github.com/MontFerret/cli/pkg/runtime"
	"github.com/MontFerret/cli/pkg/source"
)

func RunCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run [script]",
		Aliases: []string{"exec"},
		Short:   "Run a FQL script",
		Args:    cobra.MinimumNArgs(0),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			paramFlag, err := cmd.Flags().GetStringArray(paramFlag)

			if err != nil {
				return err
			}

			params, err := parseParams(paramFlag)

			if err != nil {
				return err
			}

			eval, err := cmd.Flags().GetString("eval")

			if err != nil {
				return err
			}

			if eval != "" && len(args) > 0 {
				return fmt.Errorf("cannot use --eval with file arguments")
			}

			store := config.From(cmd.Context())
			rtOpts := store.GetRuntimeOptions()

			cleanup, err := browser.EnsureBrowser(cmd.Context(), rtOpts, store.GetBrowserOptions())

			if err != nil {
				return err
			}

			defer cleanup()

			sources, err := source.Resolve(source.Input{Eval: eval, Args: args})

			if err != nil {
				return err
			}

			if sources == nil {
				return cmd.Help()
			}

			return runScript(cmd, rtOpts, params, sources[0])
		},
	}

	addEvalFlag(cmd)
	addParamFlags(cmd)
	addRuntimeFlags(cmd)

	return cmd
}

func runScript(cmd *cobra.Command, opts cliruntime.Options, params map[string]interface{}, query *file.Source) error {
	out, err := cliruntime.Run(cmd.Context(), opts, query, params)

	if err != nil {
		printError(err)
		return err
	}

	defer out.Close()

	_, err = io.Copy(os.Stdout, out)

	return err
}
