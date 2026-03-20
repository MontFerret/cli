package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/MontFerret/ferret/v2/pkg/compiler"
	"github.com/MontFerret/ferret/v2/pkg/diagnostics"
	"github.com/MontFerret/ferret/v2/pkg/file"

	"github.com/MontFerret/cli/config"
)

func CheckCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [files...]",
		Short: "Check FQL scripts for syntax and semantic errors",
		Args:  cobra.MinimumNArgs(0),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c := compiler.New()

			// If no args, try reading from stdin
			if len(args) == 0 {
				stat, _ := os.Stdin.Stat()

				if (stat.Mode() & os.ModeCharDevice) == 0 {
					content, err := io.ReadAll(bufio.NewReader(os.Stdin))

					if err != nil {
						return err
					}

					_, err = c.Compile(file.NewSource("stdin", string(content)))

					if err != nil {
						fmt.Fprintln(os.Stderr, diagnostics.Format(err))
						return fmt.Errorf("stdin has errors")
					}

					return nil
				}

				return cmd.Help()
			}

			failed := 0

			for _, path := range args {
				content, err := os.ReadFile(path)

				if err != nil {
					fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
					failed++

					continue
				}

				_, err = c.Compile(file.NewSource(path, string(content)))

				if err != nil {
					fmt.Fprintln(os.Stderr, diagnostics.Format(err))
					failed++
				}
			}

			if failed > 0 {
				return fmt.Errorf("%d of %d scripts have errors", failed, len(args))
			}

			return nil
		},
	}

	return cmd
}
