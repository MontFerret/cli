package run

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/MontFerret/ferret/v2/pkg/bytecode/artifact"
	"github.com/MontFerret/ferret/v2/pkg/file"
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

	return resolveStdin()
}

func resolveFile(path string) (*Input, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return resolveData(path, data), nil
}

func resolveStdin() (*Input, error) {
	stat, err := os.Stdin.Stat()

	if err != nil {
		return nil, fmt.Errorf("stat stdin: %w", err)
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, nil
	}

	data, err := io.ReadAll(bufio.NewReader(os.Stdin))

	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}

	return resolveData("stdin", data), nil
}

func resolveData(name string, data []byte) *Input {
	if artifact.HasMagic(data) {
		return &Input{
			Artifact: data,
		}
	}

	return &Input{
		Source: file.NewSource(name, string(data)),
	}
}
