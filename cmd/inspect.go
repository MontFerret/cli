package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/ferret/v2/pkg/asm"
	"github.com/MontFerret/ferret/v2/pkg/compiler"

	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/source"
)

func InspectCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [script]",
		Short: "Compile and disassemble a FQL script",
		Args:  cobra.MinimumNArgs(0),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			eval, err := cmd.Flags().GetString("eval")

			if err != nil {
				return err
			}

			if eval != "" && len(args) > 0 {
				return fmt.Errorf("cannot use --eval with file arguments")
			}

			sources, err := source.Resolve(source.Input{Eval: eval, Args: args})

			if err != nil {
				return err
			}

			if sources == nil {
				return cmd.Help()
			}

			c := compiler.New()

			program, err := c.Compile(sources[0])

			if err != nil {
				printError(err)
				return fmt.Errorf("compilation failed")
			}

			showBytecode, _ := cmd.Flags().GetBool("bytecode")
			showConstants, _ := cmd.Flags().GetBool("constants")
			showFunctions, _ := cmd.Flags().GetBool("functions")
			showSummary, _ := cmd.Flags().GetBool("summary")
			showSpans, _ := cmd.Flags().GetBool("spans")

			hasFilter := showBytecode || showConstants || showFunctions || showSummary || showSpans

			if !hasFilter {
				out, err := asm.Disassemble(program)

				if err != nil {
					return err
				}

				fmt.Print(out)

				return nil
			}

			sections := 0

			if showBytecode {
				sections++
			}

			if showConstants {
				sections++
			}

			if showFunctions {
				sections++
			}

			if showSummary {
				sections++
			}

			if showSpans {
				sections++
			}

			needHeader := sections > 1

			if showSummary {
				if needHeader {
					fmt.Println("==> Summary <==")
				}

				printSummary(program)

				if needHeader {
					fmt.Println()
				}
			}

			if showBytecode {
				if needHeader {
					fmt.Println("==> Bytecode <==")
				}

				printBytecode(program)

				if needHeader {
					fmt.Println()
				}
			}

			if showConstants {
				if needHeader {
					fmt.Println("==> Constants <==")
				}

				printConstants(program)

				if needHeader {
					fmt.Println()
				}
			}

			if showFunctions {
				if needHeader {
					fmt.Println("==> Functions <==")
				}

				printFunctions(program)

				if needHeader {
					fmt.Println()
				}
			}

			if showSpans {
				if needHeader {
					fmt.Println("==> Spans <==")
				}

				printSpans(program)
			}

			return nil
		},
	}

	addEvalFlag(cmd)
	cmd.Flags().Bool("bytecode", false, "Show only bytecode instructions")
	cmd.Flags().Bool("constants", false, "Show only the constant pool")
	cmd.Flags().Bool("functions", false, "Show only function definitions")
	cmd.Flags().Bool("summary", false, "Show a high-level program summary")
	cmd.Flags().Bool("spans", false, "Show debug source spans")

	return cmd
}
