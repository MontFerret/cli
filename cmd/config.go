package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/config"
)

// ConfigCommand command to manipulate with config file
func ConfigCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Ferret configs",
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
		Use:   "get",
		Short: "Get a Ferret config value by key",
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			val, err := store.Get(args[0])

			if err == nil {
				fmt.Println(val)

				return nil
			}

			if err == config.ErrInvalidFlag {
				return fmt.Errorf("%s\n%s", err, config.FlagsStr)
			}

			return err
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set",
		Short: "Set a Ferret config value by key",
		Args:  cobra.MinimumNArgs(2),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			err := store.Set(args[0], args[1])

			if err == config.ErrInvalidFlag {
				return fmt.Errorf("%s\n%s", err, config.FlagsStr)
			}

			return err
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Get a list of Ferret config values",
		Args:    cobra.MaximumNArgs(0),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		Run: func(_ *cobra.Command, _ []string) {
			for _, kv := range store.List() {
				fmt.Printf("%s: %v\n", kv.Key, kv.Value)
			}
		},
	})

	return cmd
}
