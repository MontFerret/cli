package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/config"
	"github.com/MontFerret/cli/runtime"
)

// VersionCommand command to display version
func VersionCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the CLI version information",
		Args:  cobra.MaximumNArgs(0),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runVersion(cmd, store)
		},
	}

	cmd.Flags().StringP(config.ExecRuntime, "r", runtime.DefaultRuntime, "Ferret runtime type (\"builtin\"|$url)")

	return cmd
}

func runVersion(cmd *cobra.Command, store *config.Store) error {
	rt, err := runtime.New(store.GetRuntimeOptions())

	if err != nil {
		return err
	}

	ver, err := rt.Version(cmd.Context())

	if err != nil {
		return err
	}

	fmt.Println("Version:")
	fmt.Printf("  Self: %s\n", store.AppVersion())
	fmt.Printf("  Runtime: %s\n", ver)

	return nil
}
