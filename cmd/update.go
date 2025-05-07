package cmd

import (
	"fmt"
	"runtime"

	"github.com/MontFerret/cli/config"
	"github.com/MontFerret/cli/internal/selfupdate"

	"github.com/spf13/cobra"
)

func SelfUpdateCommand(store *config.Store) *cobra.Command {
	cmd := cobra.Command{
		Use:  "update",
		Args: cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return fmt.Errorf("unknown command %q", args[0])
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "self",
		Short: "Update Ferret CLI",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			updater, err := selfupdate.NewUpdater(
				store.RepoOwner(),
				store.Repo(),
				runtime.GOOS,
				runtime.GOARCH,
				store.AppVersion(),
			)
			if err != nil {
				return err
			}
			return updater.Update()
		},
	})

	return &cmd
}
