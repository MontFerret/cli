package cmd

import (
	"fmt"
	"os"

	"github.com/MontFerret/ferret/v2/pkg/diagnostics"
)

func printError(err error) {
	fmt.Fprintln(os.Stderr, diagnostics.Format(err))
}
