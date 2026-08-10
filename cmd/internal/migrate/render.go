package migrate

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/MontFerret/cli/v2/internal/migration"
)

func renderMigrationStatus(output io.Writer, result *migration.Result, mode migration.Mode) {
	fmt.Fprintln(output, "Ferret v1 → v2 compatibility migration")
	fmt.Fprintln(output, "✓ Found go.mod")
	fmt.Fprintf(output, "✓ Scanned %d Go %s\n", result.ScannedFiles, migrationNoun(result.ScannedFiles, "file", "files"))

	if len(result.Changes) == 0 {
		if len(result.ManualActions) == 0 {
			fmt.Fprintln(output, "✓ No v1 migration changes required")
		} else {
			fmt.Fprintln(output, "No safe automatic changes available.")
		}

		if mode != migration.ModeApply {
			fmt.Fprintln(output, "No files changed.")
		}

		return
	}

	if mode == migration.ModeDryRun || mode == migration.ModePrint {
		if mode == migration.ModeDryRun {
			fmt.Fprintln(output, "Would update:")
		} else {
			fmt.Fprintf(
				output,
				"Printed a unified diff for %d %s.\n",
				len(result.Changes),
				migrationNoun(len(result.Changes), "file", "files"),
			)
		}

		if mode == migration.ModeDryRun {
			for _, change := range result.Changes {
				fmt.Fprintf(output, "  %s\n", change.Path)
			}
		}

		fmt.Fprintln(output, "No files changed.")
		return
	}

	fmt.Fprintln(output, "Changed:")
	for _, change := range result.Changes {
		fmt.Fprintf(output, "  %s\n", change.Path)
	}

	if result.UpdatedImports > 0 {
		fmt.Fprintf(
			output,
			"✓ Updated %d Ferret %s\n",
			result.UpdatedImports,
			migrationNoun(result.UpdatedImports, "import", "imports"),
		)
	}
	if result.DependenciesChanged {
		fmt.Fprintln(output, "✓ Updated Go module dependencies")
	}
	if result.FormattedFiles > 0 {
		fmt.Fprintf(
			output,
			"✓ Formatted %d Go %s\n",
			result.FormattedFiles,
			migrationNoun(result.FormattedFiles, "file", "files"),
		)
	}

	if len(result.ManualActions) > 0 {
		fmt.Fprintln(output, "Mechanical migration completed. Manual follow-up is still required.")
		return
	}

	fmt.Fprintln(output, "Migration completed.")
	fmt.Fprintln(output, "Your project now uses the Ferret v2 compatibility API.")
	fmt.Fprintln(output, "You can migrate to the native v2 API incrementally.")
}

func renderMigrationWarnings(output io.Writer, result *migration.Result) {
	if len(result.ManualActions) > 0 {
		fmt.Fprintln(output, "Manual follow-up:")
		for _, action := range result.ManualActions {
			fmt.Fprintf(
				output,
				"  %s:%d: %s (%s)\n",
				action.Path,
				action.Line,
				action.ImportPath,
				action.Reason,
			)
		}
	}

	if result.VendorDetected && len(result.Changes) > 0 {
		fmt.Fprintln(output, "Vendor directory was not modified; run go mod vendor after reviewing the migration.")
	}
}

func renderMigrationDiff(output io.Writer, result *migration.Result) error {
	for _, change := range result.Changes {
		fromFile := "a/" + filepath.ToSlash(change.Path)
		if !change.BeforeExists {
			fromFile = "/dev/null"
		}

		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(string(change.Before)),
			B:        difflib.SplitLines(string(change.After)),
			FromFile: fromFile,
			ToFile:   "b/" + filepath.ToSlash(change.Path),
			Context:  3,
		}

		if err := difflib.WriteUnifiedDiff(output, diff); err != nil {
			return fmt.Errorf("render migration diff for %s: %w", change.Path, err)
		}
	}

	return nil
}

func migrationNoun(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}
