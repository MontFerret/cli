package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MontFerret/ferret/v2/pkg/compiler"

	"github.com/MontFerret/cli/v2/pkg/build"
	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/source"
)

func BuildCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [files...]",
		Short: "Compile FQL scripts into bytecode artifacts",
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := cmd.Flags().GetString("output")

			if err != nil {
				return err
			}

			return runBuild(args, output)
		},
	}

	cmd.Flags().StringP("output", "o", "", "Output path: file (for single input) or directory (for single or multiple inputs)")

	return cmd
}

func runBuild(args []string, output string) error {
	plan, err := build.PlanOutputs(args, output)

	if err != nil {
		return err
	}

	sources, err := source.Resolve(source.Input{Args: args})

	if err != nil {
		return err
	}

	if plan.OutputDir != "" {
		if err := os.MkdirAll(plan.OutputDir, 0o755); err != nil {
			return fmt.Errorf("create output directory %s: %w", plan.OutputDir, err)
		}
	}

	c := compiler.New()
	failed := 0

	for i, src := range sources {
		if err := build.WriteArtifact(c, src, plan.Targets[i].OutputPath); err != nil {
			printError(err)
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d scripts failed to build", failed, len(sources))
	}

	return nil
}
