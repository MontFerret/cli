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

	var missingDependency *install.MissingDependencyError
	var missingComposition *install.MissingCompositionError
	hasMissingDependency := errors.As(err, &missingDependency)
	hasMissingComposition := errors.As(err, &missingComposition)

	if (!hasMissingDependency && !hasMissingComposition) ||
		(hasMissingDependency && options.InstallMissingDependencies) ||
		(hasMissingComposition && options.ScaffoldMissingComposition) {
		return nil, false, err
	}

	if !terminal() {
		manual := make([]string, 0, 2)

		if hasMissingDependency {
			manual = append(manual, fmt.Sprintf("run go get %s@%s", missingDependency.Path, missingDependency.Version))
		}

		if hasMissingComposition {
			manual = append(manual, fmt.Sprintf("create %s with NewFerret in package %s", missingComposition.File, missingComposition.Package))
		}

		return nil, false, fmt.Errorf(
			"module install cannot prompt because stdin is not a terminal; rerun with --yes or %s: %w",
			strings.Join(manual, " and "),
			err,
		)
	}

	linePrompt, err := prompts(reader, output)
	if err != nil {
		return nil, false, fmt.Errorf("start project setup prompt: %w", err)
	}

	defer linePrompt.Close()

	approved, err := confirmInstallPrerequisites(ctx, linePrompt, output, missingDependency, missingComposition)
	if err != nil || !approved {
		return nil, false, err
	}

	if hasMissingDependency {
		options.InstallMissingDependencies = true
	}

	if hasMissingComposition {
		options.ScaffoldMissingComposition = true
	}

	result, err = service.Install(ctx, options)
	if err != nil {
		return nil, false, err
	}

	return result, true, nil
}

func confirmInstallPrerequisites(
	ctx context.Context,
	linePrompt prompt,
	output io.Writer,
	dependency *install.MissingDependencyError,
	composition *install.MissingCompositionError,
) (bool, error) {
	fmt.Fprintln(output, "\nProject setup required")

	if dependency != nil {
		fmt.Fprintln(output, "  Ferret modules are compiled into the application and require Ferret v2.")
		fmt.Fprintf(output, "  Add dependency: %s@%s\n", dependency.Path, dependency.Version)
	}

	if composition != nil {
		fmt.Fprintln(output, "  A composition helper owns the Ferret engine options for this package.")
		fmt.Fprintf(output, "  Create composition helper: %s (package %s)\n", composition.File, composition.Package)
	}

	fmt.Fprintln(output, "  All project changes will be validated and committed together.")

	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		answer, err := linePrompt.Readline("Apply project setup? [Y/n]: ")
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
