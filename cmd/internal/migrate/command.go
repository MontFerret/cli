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
		Short: "Migrate supported Ferret v1 Go and FQL source behavior",
		Long: "Migrate the containing Go module from supported Ferret v1 Go imports and FQL source behavior to Ferret v2.\n\n" +
			"FQL migration returns and canonically formats a structurally recognized final top-level FOR. " +
			"Malformed FQL is left unchanged and reported for manual follow-up.\n\n" +
			"The command performs only documented mechanical import, dependency, and source changes. " +
			"It does not convert application logic or arbitrary APIs to native Ferret v2 equivalents.",
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
	command.AddCommand(newCompatibilityCheckCommand(service))

	return command
}
