package diagnostics

import (
	"fmt"
	"os"

	"github.com/MontFerret/ferret/v2/pkg/diagnostics"
)

// PrintError preserves the CLI's direct stderr rendering for Ferret diagnostics.
func PrintError(err error) {
	fmt.Fprintln(os.Stderr, diagnostics.Format(err))
}
