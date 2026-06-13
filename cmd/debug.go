package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/internal/debugcli"
	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/config"
	clirun "github.com/MontFerret/cli/v2/pkg/run"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func DebugCommand(store *config.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug <script.fql>",
		Short: "Debug a FQL script interactively",
		Long: `Debug a local FQL source script using the interactive Ferret debugger.

Prompt commands: help, break, delete, breakpoints, continue, step, next, out,
pause, where, locals, print, and quit.

Debugging currently requires the builtin runtime and does not support compiled
artifacts, stdin, inline evaluation, remote runtimes, or conditional breakpoints.`,
		Args: cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			paramFlags, err := cmd.Flags().GetStringArray(paramFlag)
			if err != nil {
				return err
			}

			params, err := parseParams(paramFlags)
			if err != nil {
				return err
			}

			store := config.From(cmd.Context())
			return executeDebug(cmd, store.GetRuntimeOptions(), store.GetBrowserOptions(), params, args)
		},
	}

	addParamFlags(cmd)
	addRuntimeFlags(cmd)

	return cmd
}

func executeDebug(cmd *cobra.Command, rtOpts cliruntime.Options, brOpts browser.Options, params map[string]any, args []string) error {
	input, err := clirun.ResolveInput("", args)
	if err != nil {
		return err
	}

	if input != nil && len(input.Artifact) > 0 {
		return fmt.Errorf("debugging compiled artifacts is not supported")
	}

	if input == nil || input.Source == nil {
		return fmt.Errorf("debug requires a source script file")
	}

	if err := cliruntime.ValidateOptions(rtOpts); err != nil {
		return err
	}

	if !cliruntime.IsBuiltinType(rtOpts.Type) {
		return cliruntime.ErrDebugRequiresBuiltinRuntime
	}

	cleanup, err := browser.EnsureBrowser(cmd.Context(), rtOpts, brOpts)
	if err != nil {
		return err
	}
	defer cleanup()

	session, err := cliruntime.NewDebugSession(cmd.Context(), rtOpts, params, input.Source)
	if err != nil {
		printError(err)
		return err
	}

	return debugcli.Start(cmd.Context(), session, input.Source)
}
