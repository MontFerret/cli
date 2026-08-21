package migrate

import (
	"fmt"
	"io"

	"github.com/MontFerret/cli/v2/internal/migration"
)

func renderCompatibilityDiagnostics(output io.Writer, result *migration.CompatibilityResult) {
	for _, diagnostic := range result.Diagnostics {
		fmt.Fprintf(
			output,
			"%s:%d:%d: %s\n",
			diagnostic.Path,
			diagnostic.Line,
			diagnostic.Column,
			diagnostic.Message,
		)
		fmt.Fprintf(output, "  help: %s\n\n", diagnostic.Help)
	}
}

func renderCompatibilitySuccess(output io.Writer, target string, result *migration.CompatibilityResult) {
	if result.ScannedFiles == 0 {
		fmt.Fprintf(output, "✓ No FQL files found at %s.\n", target)

		return
	}

	fmt.Fprintf(
		output,
		"✓ No v1 compatibility issues found in %d FQL %s.\n",
		result.ScannedFiles,
		migrationNoun(result.ScannedFiles, "file", "files"),
	)
}
