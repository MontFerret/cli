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

func newRunCommand(store *config.Store, service Service) *cobra.Command {
	command := &cobra.Command{
		Use:   "run [path]",
		Short: "Run supported Ferret v1-to-v2 migrations",
		Long: "Run supported Ferret v1-to-v2 migrations on a standalone FQL file or project directory. " +
			"The path defaults to the current directory. FQL-only targets do not require a Go module or Go toolchain.\n\n" +
			"The selected directory is the migration boundary. A containing Go module supplies metadata and dependency " +
			"ownership only when the selected directory contains Go source.\n\n" +
			"Directory migration skips descendant vendor, testdata, node_modules, hidden and underscore-prefixed " +
			"directories, and nested Go modules. The selected directory itself is scanned regardless of its name. " +
			"Directory symlinks are not followed.\n\n" +
			"FQL migration returns and canonically formats a structurally recognized final top-level FOR. " +
			"Malformed FQL is left unchanged and reported for manual follow-up.\n\n" +
			"The command performs only documented mechanical import, dependency, and source changes. " +
			"It does not convert application logic or arbitrary APIs to native Ferret v2 equivalents.",
		Args: cobra.MaximumNArgs(1),
		PreRun: func(command *cobra.Command, _ []string) {
			store.BindFlags(command)
		},
		RunE: func(command *cobra.Command, args []string) error {
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

			path := "."
			if len(args) == 1 {
				path = args[0]
			}

			result, err := service.Migrate(command.Context(), migration.Options{
				Path: path,
				Mode: mode,
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

	command.Flags().Bool(dryRunFlag, false, "Show which files would change without modifying the target")
	command.Flags().Bool(printFlag, false, "Print a unified diff without modifying the target")

	return command
}
