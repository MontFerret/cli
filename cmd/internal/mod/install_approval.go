package mod

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/MontFerret/cli/v2/pkg/module/install"
)

func installWithApproval(
	ctx context.Context,
	service Service,
	options install.Options,
	terminal terminalDetector,
	prompts promptFactory,
	reader io.Reader,
	output io.Writer,
) (*install.Result, bool, error) {
	result, err := service.Install(ctx, options)
	if err == nil {
		return result, true, nil
	}

	var missing *install.MissingDependencyError
	if !errors.As(err, &missing) || options.InstallMissingDependencies {
		return nil, false, err
	}

	if !terminal() {
		return nil, false, fmt.Errorf(
			"module install cannot prompt because stdin is not a terminal; rerun with --yes or run go get %s@%s: %w",
			missing.Path,
			missing.Version,
			err,
		)
	}

	linePrompt, err := prompts(reader, output)
	if err != nil {
		return nil, false, fmt.Errorf("start dependency approval prompt: %w", err)
	}
	defer linePrompt.Close()

	approved, err := confirmInstallDependency(ctx, linePrompt, output, missing)
	if err != nil || !approved {
		return nil, false, err
	}

	options.InstallMissingDependencies = true
	result, err = service.Install(ctx, options)
	if err != nil {
		return nil, false, err
	}

	return result, true, nil
}

func confirmInstallDependency(ctx context.Context, linePrompt prompt, output io.Writer, dependency *install.MissingDependencyError) (bool, error) {
	fmt.Fprintf(
		output,
		"\nProject dependency required\n  Ferret modules are compiled into the application and require Ferret v2.\n  Dependency: %s@%s\n",
		dependency.Path,
		dependency.Version,
	)

	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		answer, err := linePrompt.Readline(fmt.Sprintf("Install %s@%s? [Y/n]: ", dependency.Path, dependency.Version))
		if errors.Is(err, errPromptCanceled) {
			fmt.Fprintln(output, "\nModule installation canceled.")
			return false, nil
		}
		if err != nil {
			return false, err
		}

		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "", "y", "yes":
			return true, nil
		case "n", "no":
			fmt.Fprintln(output, "\nModule installation canceled.")
			return false, nil
		default:
			fmt.Fprintln(output, "  Please answer yes or no.")
		}
	}
}
