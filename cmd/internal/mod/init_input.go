package mod

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

func readInitInput(command *cobra.Command, args []string) (initInput, error) {
	input := initInput{nameSet: len(args) == 1}
	if input.nameSet {
		input.options.Name = args[0]
	}

	var err error
	input.options.GoModule, err = command.Flags().GetString(moduleGoModuleFlag)
	if err != nil {
		return initInput{}, err
	}
	input.goModuleSet = command.Flags().Changed(moduleGoModuleFlag)

	input.options.Directory, err = command.Flags().GetString(moduleDirFlag)
	if err != nil {
		return initInput{}, err
	}
	input.directorySet = command.Flags().Changed(moduleDirFlag)

	input.options.Namespace, err = command.Flags().GetString(moduleNamespaceFlag)
	if err != nil {
		return initInput{}, err
	}
	input.namespaceSet = command.Flags().Changed(moduleNamespaceFlag)

	return input, nil
}

func resolveInitOptions(
	ctx context.Context,
	input initInput,
	terminal terminalDetector,
	prompts promptFactory,
	reader io.Reader,
	output io.Writer,
) (scaffold.Options, bool, error) {
	if !terminal() {
		if err := validateNonInteractiveInit(input); err != nil {
			return scaffold.Options{}, false, err
		}

		options, err := resolveInitDefaults(input)
		return options, err == nil, err
	}

	if !needsInitPrompt(input) {
		options, err := resolveInitDefaults(input)
		return options, err == nil, err
	}

	prompt, err := prompts(reader, output)
	if err != nil {
		return scaffold.Options{}, false, fmt.Errorf("start module initialization prompt: %w", err)
	}
	defer prompt.Close()

	wizard := newInitWizard(prompt, output)
	return wizard.Resolve(ctx, input)
}

func validateNonInteractiveInit(input initInput) error {
	missing := make([]string, 0, 2)
	if !input.nameSet {
		missing = append(missing, "<name>")
	}
	if !input.goModuleSet {
		missing = append(missing, "--go-module")
	}
	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf(
		"module init cannot prompt because stdin is not a terminal; provide %s",
		strings.Join(missing, " and "),
	)
}

func needsInitPrompt(input initInput) bool {
	return !input.nameSet || !input.goModuleSet || !input.directorySet || !input.namespaceSet
}

func resolveInitDefaults(input initInput) (scaffold.Options, error) {
	defaults, err := scaffold.DefaultOptions(input.options.Name)
	if err != nil {
		return scaffold.Options{}, err
	}

	return applyInitInput(defaults, input)
}

func applyInitInput(defaults scaffold.Options, input initInput) (scaffold.Options, error) {
	if input.goModuleSet {
		defaults.GoModule = input.options.GoModule
	}
	if input.directorySet {
		defaults.Directory = input.options.Directory
	}
	if input.namespaceSet {
		defaults.Namespace = input.options.Namespace
	}

	if err := defaults.Validate(); err != nil {
		return scaffold.Options{}, err
	}

	return defaults, nil
}
