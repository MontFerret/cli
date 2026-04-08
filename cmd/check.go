package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/ferret/v2/pkg/compiler"

	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/source"
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
			sources, err := source.Resolve(source.Input{Args: args})

			if err != nil {
				return err
			}

			if sources == nil {
				return cmd.Help()
			}

			c := compiler.New()
			failed := 0

			for _, src := range sources {
				_, err := c.Compile(src)

				if err != nil {
					printError(err)
					failed++
				}
			}

			if failed > 0 {
				return fmt.Errorf("%d of %d scripts have errors", failed, len(sources))
			}

			return nil
		},
	}

	return cmd
}
