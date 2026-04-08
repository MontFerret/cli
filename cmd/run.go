package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/config"
	clirun "github.com/MontFerret/cli/v2/pkg/run"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func RunCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run [script]",
		Aliases: []string{"exec"},
		Short:   "Run a FQL script or compiled artifact",
		Args:    cobra.MaximumNArgs(1),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			paramFlag, err := cmd.Flags().GetStringArray(paramFlag)

			if err != nil {
				return err
			}

			params, err := parseParams(paramFlag)

			if err != nil {
				return err
			}

			eval, err := cmd.Flags().GetString("eval")

			if err != nil {
				return err
			}

			if eval != "" && len(args) > 0 {
				return fmt.Errorf("cannot use --eval with file arguments")
			}

			store := config.From(cmd.Context())
			return executeRun(cmd, store.GetRuntimeOptions(), store.GetBrowserOptions(), params, eval, args)
		},
	}

	addEvalFlag(cmd)
	addParamFlags(cmd)
	addRuntimeFlags(cmd)

	return cmd
}

func executeRun(cmd *cobra.Command, rtOpts cliruntime.Options, brOpts browser.Options, params map[string]interface{}, eval string, args []string) error {
	input, err := clirun.ResolveInput(eval, args)

	if err != nil {
		return err
	}

	if input == nil {
		return cmd.Help()
	}

	if len(input.Artifact) > 0 && !cliruntime.IsBuiltinType(rtOpts.Type) {
		return cliruntime.ErrArtifactRequiresBuiltinRuntime
	}

	cleanup, err := browser.EnsureBrowser(cmd.Context(), rtOpts, brOpts)

	if err != nil {
		return err
	}

	defer cleanup()

	out, err := clirun.Execute(cmd.Context(), rtOpts, params, input)

	if err != nil {
		printError(err)
		return err
	}

	defer out.Close()

	_, err = io.Copy(os.Stdout, out)

	return err
}
