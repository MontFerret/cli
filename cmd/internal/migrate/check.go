package migrate

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/internal/migration"
)

const compatibilityFromFlag = "from"

func newCompatibilityCheckCommand(service Service) *cobra.Command {
	command := &cobra.Command{
		Use:   "check [path]",
		Short: "Check FQL source for Ferret version compatibility",
		Long: "Check a standalone FQL file or recursively inspect a directory for supported behavior changes " +
			"between Ferret versions. The check does not require a Go module and never modifies source files.\n\n" +
			"Directory scans include testdata, hidden and underscore-prefixed directories, and nested Go modules. " +
			"They skip .git, .hg, .svn, vendor, and node_modules and do not follow directory symlinks.\n\n" +
			"Compatibility findings and malformed FQL make the command fail after all readable files are checked.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			from, err := command.Flags().GetString(compatibilityFromFlag)
			if err != nil {
				return err
			}

			if from != migration.CompatibilityVersionV1 {
				return fmt.Errorf("unsupported compatibility source version %q: expected v1", from)
			}

			checker, ok := service.(compatibilityService)
			if !ok {
				return fmt.Errorf("compatibility checker is not configured")
			}

			target := "."
			if len(args) == 1 {
				target = args[0]
			}

			result, err := checker.CheckCompatibility(command.Context(), migration.CompatibilityOptions{
				Path: target,
				From: from,
			})
			if err != nil {
				return err
			}

			renderCompatibilityDiagnostics(command.ErrOrStderr(), result)

			if err := newCompatibilityCheckError(result); err != nil {
				return err
			}

			renderCompatibilitySuccess(command.OutOrStdout(), target, result)

			return nil
		},
	}

	command.Flags().String(
		compatibilityFromFlag,
		migration.CompatibilityVersionV1,
		"Ferret source version to check (currently only v1)",
	)

	return command
}
