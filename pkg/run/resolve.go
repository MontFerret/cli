package run

import (
	"fmt"
	"os"

	"github.com/MontFerret/ferret/v2/pkg/bytecode/artifact"
	"github.com/MontFerret/ferret/v2/pkg/file"

	"github.com/MontFerret/cli/pkg/source"
)

func ResolveInput(eval string, args []string) (*Input, error) {
	if eval != "" {
		return &Input{
			Source: file.NewSource("<eval>", eval),
		}, nil
	}

	if len(args) == 1 {
		return resolveFile(args[0])
	}

	sources, err := source.Resolve(source.Input{})

	if err != nil {
		return nil, err
	}

	if sources == nil {
		return nil, nil
	}

	return &Input{
		Source: sources[0],
	}, nil
}

func resolveFile(path string) (*Input, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if artifact.HasMagic(data) {
		return &Input{
			Artifact: data,
		}, nil
	}

	return &Input{
		Source: file.NewSource(path, string(data)),
	}, nil
}
