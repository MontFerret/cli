package diagnostics

import (
	"io"
	"os"

	"github.com/MontFerret/cli/v2/internal/diagnostics"
)

// PrintError preserves the CLI's direct stderr rendering for Ferret diagnostics.
func PrintError(err error) {
	diagnostics.Print(os.Stderr, err)
}

// PrintErrorTo renders execution diagnostics to the command's selected stderr.
func PrintErrorTo(out io.Writer, err error) {
	diagnostics.Print(out, err)
}
