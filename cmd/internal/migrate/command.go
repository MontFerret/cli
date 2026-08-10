package migrate

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/internal/migration"
	"github.com/MontFerret/cli/v2/pkg/config"
)

const (
	dryRunFlag = "dry-run"
	printFlag  = "print"
)

// New creates the v1-to-v2 compatibility migration command.
func New(store *config.Store, service Service) *cobra.Command {
	command := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate a Ferret v1 Go application to the v2 compatibility API",
		Long: "Migrate the containing Go module from Ferret v1 imports to the Ferret v2 compatibility API.\n\n" +
			"The command performs only documented mechanical import and dependency changes. " +
			"It does not convert application logic to the native Ferret v2 API.",
		Args: cobra.NoArgs,
		PreRun: func(command *cobra.Command, _ []string) {
			store.BindFlags(command)
		},
		RunE: func(command *cobra.Command, _ []string) error {
			dryRun, err := command.Flags().GetBool(dryRunFlag)
			if err != nil {
				return err
			}

			printChanges, err := command.Flags().GetBool(printFlag)
			if err != nil {
				return err
			}

			if dryRun && printChanges {
				return fmt.Errorf("--dry-run and --print cannot be used together")
			}

			if service == nil {
				return fmt.Errorf("migration service is not configured")
			}

			mode := migration.ModeApply
			if dryRun {
				mode = migration.ModeDryRun
			} else if printChanges {
				mode = migration.ModePrint
			}

			result, err := service.Migrate(command.Context(), migration.Options{
				Directory: ".",
				Mode:      mode,
			})
			if err != nil {
				return err
			}

			if printChanges {
				if err := renderMigrationDiff(command.OutOrStdout(), result); err != nil {
					return err
				}

				renderMigrationStatus(command.ErrOrStderr(), result, mode)
			} else {
				renderMigrationStatus(command.OutOrStdout(), result, mode)
			}

			renderMigrationWarnings(command.ErrOrStderr(), result)

			return nil
		},
	}

	command.Flags().Bool(dryRunFlag, false, "Show which files would change without modifying the project")
	command.Flags().Bool(printFlag, false, "Print a unified diff without modifying the project")

	return command
}
