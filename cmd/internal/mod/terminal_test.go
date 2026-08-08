package mod

import (
	"testing"

	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
)

func TestDefaultTerminalRejectsNullInput(t *testing.T) {
	testutil.WithDevNullStdin(t, func() {
		if defaultTerminal() {
			t.Fatal("null input was treated as an interactive terminal")
		}
	})
}
