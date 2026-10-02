package version

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/cmd/internal/execution"
	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/runtime"
)

func New(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the CLI and selected runtime version information",
		Args:  cobra.MaximumNArgs(0),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runVersion(cmd, store)
		},
	}

	execution.AddRuntimeSelectionFlags(cmd)

	return cmd
}

func runVersion(cmd *cobra.Command, store *config.Store) (err error) {
	resources, err := runtime.OpenSource(cmd.Context(), store.GetRuntimeOptions())
	if err != nil {
		return err
	}

	defer func() { err = errors.Join(err, resources.Close()) }()

	var version string
	if resources.Legacy != nil {
		version, err = resources.Legacy.Version(cmd.Context())
	} else {
		metadata, versionErr := resources.Runtime.Version(cmd.Context())
		version, err = metadata.String(), versionErr
	}

	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Version:\n  Self: %s\n  Runtime: %s\n", store.AppVersion(), version)

	return err
}
