package migration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

func planDependencyChanges(
	ctx context.Context,
	runner Runner,
	versionProvider ferretVersionProvider,
	project *migrationProject,
	sources *goSourcePlan,
) ([]plannedChange, bool, error) {
	// Existing compat imports alone are not a migration and must not trigger dependency cleanup or upgrades.
	if sources.UpdatedImports == 0 {
		return nil, false, nil
	}

	if versionProvider == nil {
		return nil, false, fmt.Errorf("ferret version provider is not configured")
	}

	floorVersion, err := versionProvider()
	if err != nil {
		return nil, false, fmt.Errorf("resolve Ferret v2 dependency floor: %w", err)
	}

	floor, err := parseFerretVersion(floorVersion)
	if err != nil {
		return nil, false, fmt.Errorf("CLI Ferret dependency %q is invalid: %w", floorVersion, err)
	}

	existingVersion, existingIndirect := migrationRequirement(project.GoModFile, v2ModulePath)
	targetVersion := floorVersion
	needsVersionChange := existingVersion == ""

	if existingVersion != "" {
		existing, err := parseFerretVersion(existingVersion)
		if err != nil {
			return nil, false, fmt.Errorf("project Ferret v2 dependency %q is invalid: %w", existingVersion, err)
		}

		if !existing.LessThan(floor) {
			targetVersion = existingVersion
		}

		needsVersionChange = existing.LessThan(floor)
	}

	stage, err := newDependencyStage(project)
	if err != nil {
		return nil, false, err
	}
	defer os.RemoveAll(stage.directory)

	resolveDependencies := len(sources.Changes) > 0 || needsVersionChange || existingIndirect
	if resolveDependencies {
		query := v2ModulePath + "@" + targetVersion
		if _, err := runner.Run(ctx, project.Root, "get", "-modfile="+stage.modPath, query); err != nil {
			return nil, false, fmt.Errorf("resolve %s through the Go module toolchain: %w", query, err)
		}
	}

	stagedMod, err := os.ReadFile(stage.modPath)
	if err != nil {
		return nil, false, fmt.Errorf("read staged go.mod: %w", err)
	}

	parsed, err := modfile.Parse(stage.modPath, stagedMod, nil)
	if err != nil {
		return nil, false, fmt.Errorf("parse staged go.mod: %w", err)
	}

	resolvedVersion, _ := migrationRequirement(parsed, v2ModulePath)
	if resolvedVersion != "" {
		resolved, err := parseFerretVersion(resolvedVersion)
		if err != nil {
			return nil, false, fmt.Errorf("resolved Ferret v2 dependency %q is invalid: %w", resolvedVersion, err)
		}

		target, err := parseFerretVersion(targetVersion)
		if err != nil {
			return nil, false, err
		}

		if resolved.GreaterThan(target) {
			targetVersion = resolvedVersion
		}
	}

	// A subtree scan cannot prove that Ferret v1 imports are absent elsewhere in the module.
	dropV1 := project.ModuleWide && !sources.RemainingV1
	setMigrationRequirements(parsed, targetVersion, dropV1)
	parsed.Cleanup()

	updatedMod, err := parsed.Format()
	if err != nil {
		return nil, false, fmt.Errorf("format staged go.mod: %w", err)
	}

	updatedSum, sumExists, err := readOptionalMigrationFile(stage.sumPath)
	if err != nil {
		return nil, false, err
	}

	changes := make([]plannedChange, 0, 2)
	if !bytes.Equal(project.GoMod.Data, updatedMod) {
		changes = append(changes, plannedChange{
			change: Change{
				Path:         "go.mod",
				Before:       project.GoMod.Data,
				After:        updatedMod,
				BeforeExists: true,
			},
			before: project.GoMod,
			mode:   project.GoMod.Mode,
		})
	}

	if project.GoSum.Exists != sumExists || !bytes.Equal(project.GoSum.Data, updatedSum) {
		changes = append(changes, plannedChange{
			change: Change{
				Path:         "go.sum",
				Before:       project.GoSum.Data,
				After:        updatedSum,
				BeforeExists: project.GoSum.Exists,
			},
			before: project.GoSum,
			mode:   migrationFileMode(project.GoSum),
		})
	}

	return changes, len(changes) > 0, nil
}

func parseFerretVersion(version string) (*semver.Version, error) {
	normalized := strings.TrimPrefix(strings.TrimSpace(version), "v")
	parsed, err := semver.StrictNewVersion(normalized)
	if err != nil {
		return nil, err
	}

	return parsed, nil
}

func migrationRequirement(file *modfile.File, path string) (string, bool) {
	for _, requirement := range file.Require {
		if requirement.Mod.Path == path {
			return requirement.Mod.Version, requirement.Indirect
		}
	}

	return "", false
}

func setMigrationRequirements(file *modfile.File, targetVersion string, dropV1 bool) {
	requirements := make([]*modfile.Require, 0, len(file.Require)+1)
	foundV2 := false

	for _, requirement := range file.Require {
		if dropV1 && requirement.Mod.Path == v1ModulePath {
			continue
		}

		copyRequirement := &modfile.Require{
			Mod:      requirement.Mod,
			Indirect: requirement.Indirect,
		}

		if copyRequirement.Mod.Path == v2ModulePath {
			copyRequirement.Mod.Version = targetVersion
			copyRequirement.Indirect = false
			foundV2 = true
		}

		requirements = append(requirements, copyRequirement)
	}

	if !foundV2 {
		requirements = append(requirements, &modfile.Require{
			Mod: module.Version{Path: v2ModulePath, Version: targetVersion},
		})
	}

	file.SetRequire(requirements)
}

type dependencyStage struct {
	directory string
	modPath   string
	sumPath   string
}

func newDependencyStage(project *migrationProject) (*dependencyStage, error) {
	directory, err := os.MkdirTemp("", "ferret-migrate-*")
	if err != nil {
		return nil, fmt.Errorf("create migration staging directory: %w", err)
	}

	stage := &dependencyStage{
		directory: directory,
		modPath:   filepath.Join(directory, "project.mod"),
		sumPath:   filepath.Join(directory, "project.sum"),
	}

	if err := os.WriteFile(stage.modPath, project.GoMod.Data, project.GoMod.Mode.Perm()); err != nil {
		os.RemoveAll(directory)
		return nil, fmt.Errorf("stage go.mod: %w", err)
	}

	if project.GoSum.Exists {
		if err := os.WriteFile(stage.sumPath, project.GoSum.Data, project.GoSum.Mode.Perm()); err != nil {
			os.RemoveAll(directory)
			return nil, fmt.Errorf("stage go.sum: %w", err)
		}
	}

	return stage, nil
}
