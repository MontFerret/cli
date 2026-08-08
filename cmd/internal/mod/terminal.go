package mod

import (
	"os"

	"github.com/mattn/go-isatty"
)

func defaultTerminal() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
