package cmd

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
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

	openCmd := &cobra.Command{
		Use:   "open",
		Short: "Open browser",
		Args:  cobra.MaximumNArgs(0),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			pid, err := browser.Open(cmd.Context(), store.GetBrowserOptions())

			if err != nil {
				return err
			}

			fmt.Println(pid)

			return nil
		},
	}

	openCmd.Flags().BoolP(config.BrowserDetach, "d", false, "Start browser in background and print process ID")
	openCmd.Flags().Bool(config.BrowserHeadless, false, "Start browser in headless mode")
	openCmd.Flags().Uint64P(config.BrowserPort, "p", 9222, "Browser remote debugging port")
	openCmd.Flags().String(config.BrowserUserDir, "", "Browser user directory")

	closeCmd := &cobra.Command{
		Use:   "close",
		Short: "Close browser",
		Args:  cobra.MaximumNArgs(1),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var pid uint64

			if len(args) > 0 {
				p, err := strconv.ParseUint(args[0], 10, 64)

				if err != nil {
					return errors.Wrap(err, "invalid pid number")
				}

				pid = p
			}

			return browser.Close(cmd.Context(), store.GetBrowserOptions(), pid)
		},
	}

	cmd.AddCommand(openCmd)
	cmd.AddCommand(closeCmd)

	return cmd
}
