package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/MontFerret/ferret/v2/pkg/bytecode/artifact"

	"github.com/MontFerret/ferret/v2/pkg/file"

	"github.com/MontFerret/cli/pkg/browser"
	"github.com/MontFerret/cli/pkg/config"
	cliruntime "github.com/MontFerret/cli/pkg/runtime"
	"github.com/MontFerret/cli/pkg/source"
)

type runInput struct {
	Artifact []byte
	Source   *file.Source
}

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
	input, err := resolveRunInput(eval, args)

	if err != nil {
		return err
	}

	if input == nil {
		return cmd.Help()
	}

	if len(input.Artifact) > 0 && !cliruntime.IsBuiltinType(rtOpts.Type) {
		return fmt.Errorf("compiled artifacts require the builtin runtime")
	}

	cleanup, err := browser.EnsureBrowser(cmd.Context(), rtOpts, brOpts)

	if err != nil {
		return err
	}

	defer cleanup()

	if len(input.Artifact) > 0 {
		return runArtifact(cmd, rtOpts, params, input.Artifact)
	}

	return runScript(cmd, rtOpts, params, input.Source)
}

func runScript(cmd *cobra.Command, opts cliruntime.Options, params map[string]interface{}, query *file.Source) error {
	out, err := cliruntime.Run(cmd.Context(), opts, query, params)

	if err != nil {
		printError(err)
		return err
	}

	defer out.Close()

	_, err = io.Copy(os.Stdout, out)

	return err
}

func runArtifact(cmd *cobra.Command, opts cliruntime.Options, params map[string]interface{}, artifactData []byte) error {
	out, err := cliruntime.RunArtifact(cmd.Context(), opts, artifactData, params)

	if err != nil {
		printError(err)
		return err
	}

	defer out.Close()

	_, err = io.Copy(os.Stdout, out)

	return err
}

func resolveRunInput(eval string, args []string) (*runInput, error) {
	if eval != "" {
		return &runInput{
			Source: file.NewSource("<eval>", eval),
		}, nil
	}

	if len(args) == 1 {
		return resolveRunFile(args[0])
	}

	sources, err := source.Resolve(source.Input{})

	if err != nil {
		return nil, err
	}

	if sources == nil {
		return nil, nil
	}

	return &runInput{
		Source: sources[0],
	}, nil
}

func resolveRunFile(path string) (*runInput, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if isArtifactData(data) {
		return &runInput{
			Artifact: data,
		}, nil
	}

	return &runInput{
		Source: file.NewSource(path, string(data)),
	}, nil
}

func isArtifactData(data []byte) bool {
	return artifact.HasMagic(data)
}
