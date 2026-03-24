package cmd

import (
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/pkg/browser"
	"github.com/MontFerret/cli/pkg/config"
	"github.com/MontFerret/cli/pkg/repl"
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
			paramFlag, err := cmd.Flags().GetStringArray(paramFlag)

			if err != nil {
				return err
			}

			params, err := parseParams(paramFlag)

			if err != nil {
				return err
			}

			store := config.From(cmd.Context())
			rtOpts := store.GetRuntimeOptions()

			cleanup, err := browser.EnsureBrowser(cmd.Context(), rtOpts, store.GetBrowserOptions())

			if err != nil {
				return err
			}

			defer cleanup()

			return repl.Start(cmd.Context(), rtOpts, params)
		},
	}

	addParamFlags(cmd)
	addRuntimeFlags(cmd)

	return cmd
}
