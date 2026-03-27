package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func PlanOutputs(inputs []string, output string) (Plan, error) {
	if len(inputs) == 0 {
		return Plan{}, fmt.Errorf("build requires at least one input file")
	}

	if output == "" {
		targets := make([]Target, 0, len(inputs))

		for _, input := range inputs {
			targets = append(targets, Target{
				SourcePath: input,
				OutputPath: siblingArtifactPath(input),
			})
		}

		return Plan{Targets: targets}, nil
	}

	if len(inputs) == 1 {
		return planSingleOutput(inputs[0], output)
	}

	return planMultiOutput(inputs, output)
}

func planSingleOutput(input, output string) (Plan, error) {
	info, err := os.Stat(output)

	switch {
	case err == nil && info.IsDir():
		return Plan{
			OutputDir: output,
			Targets: []Target{
				{
					SourcePath: input,
					OutputPath: filepath.Join(output, artifactFileName(input)),
				},
			},
		}, nil
	case err == nil:
		return Plan{
			Targets: []Target{
				{
					SourcePath: input,
					OutputPath: output,
				},
			},
		}, nil
	case errors.Is(err, os.ErrNotExist):
		return Plan{
			Targets: []Target{
				{
					SourcePath: input,
					OutputPath: output,
				},
			},
		}, nil
	default:
		return Plan{}, fmt.Errorf("inspect output %s: %w", output, err)
	}
}

func planMultiOutput(inputs []string, output string) (Plan, error) {
	info, err := os.Stat(output)

	switch {
	case err == nil && !info.IsDir():
		return Plan{}, fmt.Errorf("--output must be a directory when building multiple files")
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return Plan{}, fmt.Errorf("inspect output %s: %w", output, err)
	}

	targets := make([]Target, 0, len(inputs))
	seen := make(map[string]string, len(inputs))

	for _, input := range inputs {
		outputPath := filepath.Join(output, artifactFileName(input))
		key, err := canonicalPath(outputPath)

		if err != nil {
			return Plan{}, err
		}

		if prev, exists := seen[key]; exists {
			return Plan{}, fmt.Errorf("output collision: %s and %s both map to %s", prev, input, outputPath)
		}

		seen[key] = input
		targets = append(targets, Target{
			SourcePath: input,
			OutputPath: outputPath,
		})
	}

	return Plan{
		OutputDir: output,
		Targets:   targets,
	}, nil
}
