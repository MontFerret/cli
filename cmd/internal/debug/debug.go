package debug

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/api/source"
	"github.com/MontFerret/cli/v2/cmd/internal/execution"
	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/debugger"
	clirun "github.com/MontFerret/cli/v2/pkg/run"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func New(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug <script.fql>",
		Short: "Debug a FQL script interactively",
		Long: `Debug a local FQL source file using the builtin runtime or an external Wire runtime.

Prompt commands: help, break, delete, breakpoints, continue, step, next, out,
pause, where, locals, print, and quit.

The builtin runtime can queue a pause while stopped. Wire alpha.4 only accepts
pause while running, so a pause request at the prompt is rejected.
Compiled artifacts, stdin, inline evaluation, and conditional breakpoints are unsupported.`,
		Example: `  ferret debug script.fql
  ferret debug --runtime wire --runtime-endpoint tcp://127.0.0.1:54321 --param limit=10 script.fql`,
		Args: cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			paramFlags, err := cmd.Flags().GetStringArray(execution.ParamFlag)
			if err != nil {
				return err
			}

			params, err := execution.ParseParams(paramFlags)
			if err != nil {
				return err
			}

			store := config.From(cmd.Context())
			rtOpts, err := execution.OptionsFromCommand(cmd, store)
			if err != nil {
				return err
			}

			if err := execute(cmd, rtOpts, store.GetBrowserOptions(), params, args); err != nil {
				return &diagnosticError{cause: err}
			}

			return nil
		},
	}

	execution.AddParamFlags(cmd)
	execution.AddRuntimeFlags(cmd)

	return cmd
}

func execute(cmd *cobra.Command, rtOpts cliruntime.Options, brOpts browser.Options, params map[string]any, args []string) error {
	input, err := clirun.ResolveInput("", args)
	if err != nil {
		return err
	}

	if input != nil && len(input.Artifact) > 0 {
		return fmt.Errorf("debugging compiled artifacts is not supported yet; run debug with the original .fql source file")
	}

	if input == nil || input.Source.Empty() {
		return fmt.Errorf("debug requires a source script file")
	}

	if err := cliruntime.ValidateOptions(rtOpts); err != nil {
		return err
	}

	session, err := cliruntime.NewDebugSession(cmd.Context(), rtOpts, params, input.Source)
	if err != nil {
		return err
	}

	cleanup, err := browser.EnsureBrowser(session.Context(), rtOpts, brOpts)
	if err != nil {
		return errors.Join(err, session.Close())
	}

	defer cleanup()

	src := source.New(input.Source.Name(), input.Source.Content())

	return debugger.StartWithIO(session.Context(), session, src, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
}
