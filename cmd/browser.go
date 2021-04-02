package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/browser"
	"github.com/MontFerret/cli/config"
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

	cmd.AddCommand(&cobra.Command{
		Use:   "open",
		Short: "Open browser",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := store.GetBrowserOptions()

			b := browser.New(opts)

			return b.Open(cmd.Context())
		},
	})

	cmd.PersistentFlags().Uint64(config.BrowserPort, 9222, "Browser remote debugging port")
	cmd.PersistentFlags().String(config.BrowserUserDir, "", "Browser user directory")
	cmd.PersistentFlags().Bool(config.BrowserHeadless, false, "Start browser in headless mode")

	return cmd
}
