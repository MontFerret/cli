package mod

import (
	"errors"
	"io"

	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

var errPromptCanceled = errors.New("prompt canceled")

type (
	terminalDetector func() bool

	prompt interface {
		Readline(string) (string, error)
		Close() error
	}

	promptFactory func(io.Reader, io.Writer) (prompt, error)

	initInput struct {
		options      scaffold.Options
		nameSet      bool
		goModuleSet  bool
		directorySet bool
		namespaceSet bool
	}
)
