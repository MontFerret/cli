package mod

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

type initWizard struct {
	prompt prompt
	output io.Writer
}

func newInitWizard(prompt prompt, output io.Writer) *initWizard {
	return &initWizard{prompt: prompt, output: output}
}

func (wizard *initWizard) Resolve(ctx context.Context, input initInput) (scaffold.Options, bool, error) {
	if err := ctx.Err(); err != nil {
		return scaffold.Options{}, false, err
	}

	defaults, err := wizard.resolveDefaults(ctx, input)
	if err != nil {
		return wizard.resolveError(err)
	}

	options, err := applyInitInput(defaults, input)
	if err != nil {
		return scaffold.Options{}, false, err
	}

	if !input.goModuleSet {
		options.GoModule, err = wizard.readField(
			ctx,
			"Go module path (--go-module)",
			"Import path written to go.mod and used by Go consumers.",
			"Go module",
			options.GoModule,
			func(value string) error {
				candidate := options
				candidate.GoModule = value
				return candidate.Validate()
			},
		)
		if err != nil {
			return wizard.resolveError(err)
		}
	}

	if !input.directorySet {
		options.Directory, err = wizard.readField(
			ctx,
			"Destination directory (--dir)",
			"Directory where the new module project will be created.",
			"Directory",
			options.Directory,
			func(string) error { return nil },
		)
		if err != nil {
			return wizard.resolveError(err)
		}
	}

	if !input.namespaceSet {
		options.Namespace, err = wizard.readField(
			ctx,
			"Runtime namespace (--namespace)",
			"FQL namespace exposed by the module; independent from its distribution name.",
			"Namespace",
			options.Namespace,
			func(value string) error {
				candidate := options
				candidate.Namespace = value
				return candidate.Validate()
			},
		)
		if err != nil {
			return wizard.resolveError(err)
		}
	}

	confirmed, err := wizard.confirm(ctx, options)
	if err != nil {
		return wizard.resolveError(err)
	}
	if !confirmed {
		wizard.canceled()
		return scaffold.Options{}, false, nil
	}

	return options, true, nil
}

func (wizard *initWizard) resolveDefaults(ctx context.Context, input initInput) (scaffold.Options, error) {
	if input.nameSet {
		return scaffold.DefaultOptions(input.options.Name)
	}

	for {
		value, err := wizard.readField(
			ctx,
			"Ferret module name",
			"Registry identity used for distribution and discovery. Use owner/name.",
			"Name",
			"",
			func(value string) error {
				_, err := scaffold.DefaultOptions(value)
				return err
			},
		)
		if err != nil {
			return scaffold.Options{}, err
		}

		defaults, err := scaffold.DefaultOptions(value)
		if err == nil {
			return defaults, nil
		}
	}
}

func (wizard *initWizard) readField(
	ctx context.Context,
	title string,
	description string,
	label string,
	defaultValue string,
	validate func(string) error,
) (string, error) {
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		fmt.Fprintf(wizard.output, "\n%s\n  %s\n", title, description)
		prompt := label + ": "
		if defaultValue != "" {
			prompt = fmt.Sprintf("%s [%s]: ", label, defaultValue)
		}

		value, err := wizard.prompt.Readline(prompt)
		if err != nil {
			return "", err
		}

		value = strings.TrimSpace(value)
		if value == "" {
			value = defaultValue
		}

		if err := validate(value); err != nil {
			fmt.Fprintf(wizard.output, "  Invalid value: %v\n", err)
			continue
		}

		return value, nil
	}
}

func (wizard *initWizard) confirm(ctx context.Context, options scaffold.Options) (bool, error) {
	fmt.Fprintf(
		wizard.output,
		"\nModule configuration:\n  Name: %s\n  Go module: %s\n  Directory: %s\n  Namespace: %s\n",
		options.Name,
		options.GoModule,
		options.Directory,
		options.Namespace,
	)

	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		answer, err := wizard.prompt.Readline("Create module? [Y/n]: ")
		if err != nil {
			return false, err
		}

		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "", "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Fprintln(wizard.output, "  Please answer yes or no.")
		}
	}
}

func (wizard *initWizard) resolveError(err error) (scaffold.Options, bool, error) {
	if errors.Is(err, errPromptCanceled) {
		wizard.canceled()
		return scaffold.Options{}, false, nil
	}

	return scaffold.Options{}, false, err
}

func (wizard *initWizard) canceled() {
	fmt.Fprintln(wizard.output, "\nModule initialization canceled.")
}
