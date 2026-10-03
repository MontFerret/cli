package run

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/cmd/internal/diagnostics"
	"github.com/MontFerret/cli/v2/cmd/internal/execution"
	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/config"
	clirun "github.com/MontFerret/cli/v2/pkg/run"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func New(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run [script]",
		Aliases: []string{"exec"},
		Short:   "Run a FQL script or compiled artifact",
		Args:    cobra.MaximumNArgs(1),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			paramFlag, err := cmd.Flags().GetStringArray(execution.ParamFlag)

			if err != nil {
				return err
			}

			params, err := execution.ParseParams(paramFlag)

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
			rtOpts, err := execution.OptionsFromCommand(cmd, store)
			if err != nil {
				return err
			}

			return execute(cmd, rtOpts, store.GetBrowserOptions(), params, eval, args)
		},
	}

	execution.AddEvalFlag(cmd)
	execution.AddParamFlags(cmd)
	execution.AddRuntimeFlags(cmd)

	return cmd
}

func execute(cmd *cobra.Command, rtOpts cliruntime.Options, brOpts browser.Options, params map[string]interface{}, eval string, args []string) error {
	input, err := clirun.ResolveInput(eval, args)

	if err != nil {
		return err
	}

	if input == nil {
		return cmd.Help()
	}

	if err := cliruntime.ValidateOptions(rtOpts); err != nil {
		return err
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

	return writeResult(cmd, out, err)
}

func writeResult(cmd *cobra.Command, out io.ReadCloser, err error) error {
	if out != nil {
		_, copyErr := io.Copy(cmd.OutOrStdout(), out)
		err = errors.Join(err, copyErr, out.Close())
	}

	if err != nil {
		diagnostics.PrintErrorTo(cmd.ErrOrStderr(), err)
	}

	return err
}
