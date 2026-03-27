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
		targets, err := planTargets(inputs, siblingArtifactPath)

		if err != nil {
			return Plan{}, err
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

	targets, err := planTargets(inputs, func(input string) string {
		return filepath.Join(output, artifactFileName(input))
	})

	if err != nil {
		return Plan{}, err
	}

	return Plan{
		OutputDir: output,
		Targets:   targets,
	}, nil
}

func planTargets(inputs []string, outputPath func(string) string) ([]Target, error) {
	targets := make([]Target, 0, len(inputs))
	seen := make(map[string]string, len(inputs))

	for _, input := range inputs {
		path := outputPath(input)
		key, err := canonicalPath(path)

		if err != nil {
			return nil, err
		}

		if prev, exists := seen[key]; exists {
			return nil, fmt.Errorf("output collision: %s and %s both map to %s", prev, input, path)
		}

		seen[key] = input
		targets = append(targets, Target{
			SourcePath: input,
			OutputPath: path,
		})
	}

	return targets, nil
}
