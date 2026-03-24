package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MontFerret/ferret/v2/pkg/formatter"

	"github.com/MontFerret/cli/pkg/config"
	"github.com/MontFerret/cli/pkg/source"
)

func FormatCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fmt [files...]",
		Short: "Format FQL scripts",
		Args:  cobra.MinimumNArgs(0),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := buildFormatterOptions(cmd)

			if err != nil {
				return err
			}

			f := formatter.New(opts...)

			dryRun, err := cmd.Flags().GetBool("dry-run")

			if err != nil {
				return err
			}

			sources, err := source.Resolve(source.Input{Args: args})

			if err != nil {
				return err
			}

			if sources == nil {
				return cmd.Help()
			}

			for i, src := range sources {
				if dryRun {
					if len(sources) > 1 {
						fmt.Fprintf(os.Stdout, "==> %s <==\n", src.Name())
					}

					if err := f.Format(os.Stdout, src); err != nil {
						return err
					}

					if len(sources) > 1 {
						fmt.Fprintln(os.Stdout)
					}
				} else {
					var buf bytes.Buffer

					if err := f.Format(&buf, src); err != nil {
						return err
					}

					if err := os.WriteFile(args[i], buf.Bytes(), 0o644); err != nil {
						return fmt.Errorf("writing %s: %w", args[i], err)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().Bool("dry-run", false, "Do not overwrite files and print the output to stdout")
	cmd.Flags().Uint64("print-width", 80, "Maximum line length")
	cmd.Flags().Uint64("tab-width", 4, "Indentation size")
	cmd.Flags().Bool("single-quote", false, "Use single quotes instead of double quotes")
	cmd.Flags().Bool("bracket-spacing", true, "Add spaces inside brackets")
	cmd.Flags().String("case-mode", "upper", "Keyword case mode: upper, lower, ignore")

	return cmd
}

func buildFormatterOptions(cmd *cobra.Command) ([]formatter.Option, error) {
	var opts []formatter.Option

	if cmd.Flags().Changed("print-width") {
		v, err := cmd.Flags().GetUint64("print-width")

		if err != nil {
			return nil, err
		}

		opts = append(opts, formatter.WithPrintWidth(v))
	}

	if cmd.Flags().Changed("tab-width") {
		v, err := cmd.Flags().GetUint64("tab-width")

		if err != nil {
			return nil, err
		}

		opts = append(opts, formatter.WithTabWidth(v))
	}

	if cmd.Flags().Changed("single-quote") {
		v, err := cmd.Flags().GetBool("single-quote")

		if err != nil {
			return nil, err
		}

		opts = append(opts, formatter.WithSingleQuote(v))
	}

	if cmd.Flags().Changed("bracket-spacing") {
		v, err := cmd.Flags().GetBool("bracket-spacing")

		if err != nil {
			return nil, err
		}

		opts = append(opts, formatter.WithBracketSpacing(v))
	}

	if cmd.Flags().Changed("case-mode") {
		v, err := cmd.Flags().GetString("case-mode")

		if err != nil {
			return nil, err
		}

		mode, err := parseCaseMode(v)

		if err != nil {
			return nil, err
		}

		opts = append(opts, formatter.WithCaseMode(mode))
	}

	return opts, nil
}

func parseCaseMode(value string) (formatter.CaseMode, error) {
	switch strings.ToLower(value) {
	case "upper":
		return formatter.CaseModeUpper, nil
	case "lower":
		return formatter.CaseModeLower, nil
	case "ignore":
		return formatter.CaseModeIgnore, nil
	default:
		return 0, fmt.Errorf("unknown case mode %q: expected upper, lower, or ignore", value)
	}
}
