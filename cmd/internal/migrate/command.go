package migrate

import (
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
)

// New creates the Ferret migration command group.
func New(store *config.Store, service Service) *cobra.Command {
	command := &cobra.Command{
		Use:   "migrate",
		Short: "Check or run Ferret source migrations",
		Long: "Check Ferret source compatibility or run documented mechanical migrations.\n\n" +
			"Use `ferret migrate check` to inspect FQL without changing files, or " +
			"`ferret migrate run` to apply supported Ferret v1-to-v2 changes.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}

	command.AddCommand(
		newCompatibilityCheckCommand(service),
		newRunCommand(store, service),
	)

	return command
}
