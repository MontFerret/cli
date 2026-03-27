package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MontFerret/ferret/v2/pkg/bytecode/artifact"
	"github.com/MontFerret/ferret/v2/pkg/compiler"
	"github.com/MontFerret/ferret/v2/pkg/file"

	"github.com/MontFerret/cli/pkg/config"
	"github.com/MontFerret/cli/pkg/source"
)

const artifactFileExtension = ".fqlc"

type (
	buildTarget struct {
		OutputPath string
		SourcePath string
	}

	buildPlan struct {
		OutputDir string
		Targets   []buildTarget
	}
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

	cmd.Flags().StringP("output", "o", "", "Output file path (single input) or directory (multiple inputs)")

	return cmd
}

func runBuild(args []string, output string) error {
	plan, err := planBuild(args, output)

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
		if err := buildSource(c, src, plan.Targets[i].OutputPath); err != nil {
			printError(err)
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d scripts failed to build", failed, len(sources))
	}

	return nil
}

func planBuild(inputs []string, output string) (buildPlan, error) {
	if len(inputs) == 0 {
		return buildPlan{}, fmt.Errorf("build requires at least one input file")
	}

	if output == "" {
		targets := make([]buildTarget, 0, len(inputs))

		for _, input := range inputs {
			targets = append(targets, buildTarget{
				SourcePath: input,
				OutputPath: siblingArtifactPath(input),
			})
		}

		return buildPlan{Targets: targets}, nil
	}

	if len(inputs) == 1 {
		return planSingleBuild(inputs[0], output)
	}

	return planMultiBuild(inputs, output)
}

func planSingleBuild(input, output string) (buildPlan, error) {
	info, err := os.Stat(output)

	switch {
	case err == nil && info.IsDir():
		return buildPlan{
			OutputDir: output,
			Targets: []buildTarget{
				{
					SourcePath: input,
					OutputPath: filepath.Join(output, artifactFileName(input)),
				},
			},
		}, nil
	case err == nil:
		return buildPlan{
			Targets: []buildTarget{
				{
					SourcePath: input,
					OutputPath: output,
				},
			},
		}, nil
	case errors.Is(err, os.ErrNotExist):
		return buildPlan{
			Targets: []buildTarget{
				{
					SourcePath: input,
					OutputPath: output,
				},
			},
		}, nil
	default:
		return buildPlan{}, fmt.Errorf("inspect output %s: %w", output, err)
	}
}

func planMultiBuild(inputs []string, output string) (buildPlan, error) {
	info, err := os.Stat(output)

	switch {
	case err == nil && !info.IsDir():
		return buildPlan{}, fmt.Errorf("--output must be a directory when building multiple files")
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return buildPlan{}, fmt.Errorf("inspect output %s: %w", output, err)
	}

	targets := make([]buildTarget, 0, len(inputs))
	seen := make(map[string]string, len(inputs))

	for _, input := range inputs {
		outputPath := filepath.Join(output, artifactFileName(input))
		key, err := canonicalPath(outputPath)

		if err != nil {
			return buildPlan{}, err
		}

		if prev, exists := seen[key]; exists {
			return buildPlan{}, fmt.Errorf("output collision: %s and %s both map to %s", prev, input, outputPath)
		}

		seen[key] = input
		targets = append(targets, buildTarget{
			SourcePath: input,
			OutputPath: outputPath,
		})
	}

	return buildPlan{
		OutputDir: output,
		Targets:   targets,
	}, nil
}

func buildSource(c *compiler.Compiler, src *file.Source, outputPath string) error {
	same, err := samePath(src.Name(), outputPath)

	if err != nil {
		return err
	}

	if same {
		return fmt.Errorf("output path %s would overwrite source file %s", outputPath, src.Name())
	}

	program, err := c.Compile(src)

	if err != nil {
		return err
	}

	data, err := artifact.Marshal(program, artifact.Options{})

	if err != nil {
		return fmt.Errorf("serialize %s: %w", src.Name(), err)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	return nil
}

func artifactFileName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)

	if ext == "" {
		return base + artifactFileExtension
	}

	return strings.TrimSuffix(base, ext) + artifactFileExtension
}

func siblingArtifactPath(path string) string {
	return filepath.Join(filepath.Dir(path), artifactFileName(path))
}

func samePath(left, right string) (bool, error) {
	leftPath, err := canonicalPath(left)

	if err != nil {
		return false, err
	}

	rightPath, err := canonicalPath(right)

	if err != nil {
		return false, err
	}

	if leftPath == rightPath {
		return true, nil
	}

	leftInfo, err := os.Stat(leftPath)
	if err != nil {
		return false, fmt.Errorf("inspect %s: %w", left, err)
	}

	rightInfo, err := os.Stat(rightPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("inspect %s: %w", right, err)
	}

	return os.SameFile(leftInfo, rightInfo), nil
}

func canonicalPath(path string) (string, error) {
	resolved, err := filepath.Abs(path)

	if err != nil {
		return "", fmt.Errorf("resolve path %s: %w", path, err)
	}

	return filepath.Clean(resolved), nil
}
