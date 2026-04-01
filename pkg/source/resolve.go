package source

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/MontFerret/ferret/v2/pkg/source"
)

type Input struct {
	Eval string
	Args []string
}

// Resolve returns file sources from eval, stdin, or file paths.
// Returns nil, nil when no input is available (caller should show help).
func Resolve(input Input) ([]*source.Source, error) {
	if input.Eval != "" {
		return []*source.Source{source.New("<eval>", input.Eval)}, nil
	}

	if len(input.Args) == 0 {
		stat, _ := os.Stdin.Stat()

		if (stat.Mode() & os.ModeCharDevice) == 0 {
			content, err := io.ReadAll(bufio.NewReader(os.Stdin))

			if err != nil {
				return nil, err
			}

			return []*source.Source{source.New("stdin", string(content))}, nil
		}

		return nil, nil
	}

	sources := make([]*source.Source, 0, len(input.Args))

	for _, path := range input.Args {
		content, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		sources = append(sources, source.New(path, string(content)))
	}

	return sources, nil
}
