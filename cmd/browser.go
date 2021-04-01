package cmd

import (
	"fmt"
	"github.com/MontFerret/cli/config"
	"github.com/spf13/cobra"
)

func BrowserCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browser",
		Short: "Manage Ferret browsers",
		Long:  "",
		Args:  cobra.MaximumNArgs(0),
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return fmt.Errorf("unknown command %q", args[0])
		},
	}

	return cmd
}
