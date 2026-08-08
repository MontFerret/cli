package repl

import (
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/cmd/internal/execution"
	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/config"
	clirepl "github.com/MontFerret/cli/v2/pkg/repl"
)

func New(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repl",
		Short: "Launch interactive FQL shell",
		Args:  cobra.NoArgs,
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			paramFlag, err := cmd.Flags().GetStringArray(execution.ParamFlag)

			if err != nil {
				return err
			}

			params, err := execution.ParseParams(paramFlag)

			if err != nil {
				return err
			}

			store := config.From(cmd.Context())
			rtOpts, err := execution.OptionsFromCommand(cmd, store)
			if err != nil {
				return err
			}

			cleanup, err := browser.EnsureBrowser(cmd.Context(), rtOpts, store.GetBrowserOptions())

			if err != nil {
				return err
			}

			defer cleanup()

			return clirepl.Start(cmd.Context(), rtOpts, params)
		},
	}

	execution.AddParamFlags(cmd)
	execution.AddRuntimeFlags(cmd)

	return cmd
}
