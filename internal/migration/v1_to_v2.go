package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type v1ToV2Planner struct {
	runner        Runner
	ferretVersion ferretVersionProvider
}

func newV1ToV2Planner(runner Runner, version ferretVersionProvider) *v1ToV2Planner {
	return &v1ToV2Planner{runner: runner, ferretVersion: version}
}

func (planner *v1ToV2Planner) Plan(ctx context.Context, options Options) (*migrationPlan, error) {
	directory := options.Directory
	if directory == "" {
		directory = defaultDirectory
	}

	project, err := discoverMigrationProject(ctx, planner.runner, directory)
	if err != nil {
		return nil, err
	}

	sources, err := planSourceChanges(ctx, project)
	if err != nil {
		return nil, err
	}

	dependencies, dependencyChanged, err := planDependencyChanges(
		ctx,
		planner.runner,
		planner.ferretVersion,
		project,
		sources,
	)
	if err != nil {
		return nil, err
	}

	changes := append(append([]plannedChange{}, sources.Changes...), dependencies...)
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].change.Path < changes[j].change.Path
	})

	resultChanges := make([]Change, len(changes))
	for index, change := range changes {
		resultChanges[index] = change.change
	}

	vendorDetected := false
	if info, statErr := os.Stat(filepath.Join(project.Root, "vendor")); statErr == nil {
		vendorDetected = info.IsDir()
	} else if !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("stat project vendor directory: %w", statErr)
	}

	return &migrationPlan{
		result: Result{
			Root:                project.Root,
			GoModPath:           project.GoModPath,
			Changes:             resultChanges,
			ManualActions:       sources.ManualActions,
			ScannedFiles:        sources.ScannedFiles,
			UpdatedImports:      sources.UpdatedImports,
			FormattedFiles:      sources.FormattedFiles,
			DependenciesChanged: dependencyChanged,
			VendorDetected:      vendorDetected,
		},
		changes: changes,
	}, nil
}
