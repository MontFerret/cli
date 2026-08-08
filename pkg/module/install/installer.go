package install

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"

	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

// Installer resolves registry releases and installs them into existing Go applications.
type Installer struct {
	registry      Registry
	runner        Runner
	ferretVersion ferretVersionProvider
}

// New constructs a project-local module installer.
func New(registry Registry, runner Runner) *Installer {
	if runner == nil {
		runner = execGoRunner{}
	}

	return &Installer{registry: registry, runner: runner, ferretVersion: currentFerretVersion}
}

// Install resolves, validates, stages, and commits one module installation.
func (installer *Installer) Install(ctx context.Context, options Options) (*Result, error) {
	if installer.registry == nil {
		return nil, fmt.Errorf("module registry is not configured")
	}

	id, requestedVersion, err := parseInstallReference(options.Reference)
	if err != nil {
		return nil, err
	}

	project, err := discoverInstallProject(ctx, installer.runner, options.Directory)
	if err != nil {
		return nil, err
	}

	stage, ferretDependencyAdded, err := installer.prepareProjectFerret(ctx, project, options.InstallMissingDependencies)
	if err != nil {
		return nil, err
	}

	if stage != nil {
		defer os.RemoveAll(stage.directory)
	}

	projectVersion, err := parseProjectFerretVersion(project.FerretVersion)
	if err != nil {
		return nil, err
	}

	release, err := installer.resolveRelease(ctx, id, requestedVersion, projectVersion)
	if err != nil {
		return nil, err
	}

	target, err := discoverComposition(ctx, installer.runner, project)
	if err != nil {
		return nil, err
	}

	rewrite, err := rewriteComposition(target, id, release.Version.Package.Path, release.HistoricalPackages)
	if err != nil {
		return nil, err
	}

	result := &Result{
		ID:                    id,
		Version:               release.Version.Version,
		PackagePath:           release.Version.Package.Path,
		FerretConstraint:      release.Version.Ferret,
		ProjectFerret:         project.FerretVersion,
		EditedFile:            relativeInstallPath(project.Root, target.Filename),
		FerretDependencyAdded: ferretDependencyAdded,
	}

	if !ferretDependencyAdded {
		exactDependency, err := installer.hasExactDependency(ctx, project.Root, release.Version.Package.Path, release.Version.Version)
		if err != nil {
			return nil, err
		}

		if rewrite.Registered && exactDependency {
			if _, statErr := os.Stat(project.GoSumPath); statErr == nil {
				return result, nil
			} else if !os.IsNotExist(statErr) {
				return nil, fmt.Errorf("stat project go.sum: %w", statErr)
			}
		}
	}

	if stage == nil {
		stage, err = newInstallStage(project)
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(stage.directory)
	}

	sourceSnapshot, err := stageInstallComposition(stage, target, rewrite.Source)
	if err != nil {
		return nil, err
	}

	query := release.Version.Package.Path + "@v" + release.Version.Version

	if _, err := installer.runner.Run(ctx, project.Root, "get", "-modfile="+stage.modPath, query); err != nil {
		return nil, fmt.Errorf("resolve %s through the Go module toolchain: %w", query, err)
	}

	if err := installer.validateResolvedModule(ctx, project, stage.modPath, release.Version); err != nil {
		return nil, err
	}

	packageTarget, err := installPackageTarget(project.Root, target.Directory)
	if err != nil {
		return nil, err
	}

	if _, err := installer.runner.Run(
		ctx,
		project.Root,
		"build",
		"-mod=mod",
		"-modfile="+stage.modPath,
		"-overlay="+stage.overlayPath,
		"-o="+stage.buildOutput,
		packageTarget,
	); err != nil {
		return nil, fmt.Errorf("build package %s with %s: %w", target.Package, query, err)
	}

	updatedMod, err := os.ReadFile(stage.modPath)
	if err != nil {
		return nil, fmt.Errorf("read staged go.mod: %w", err)
	}

	updatedSum, sumExists, err := readOptionalInstallFile(stage.sumPath)
	if err != nil {
		return nil, err
	}

	changes := []fileChange{
		{Before: sourceSnapshot, After: rewrite.Source, Mode: sourceSnapshot.Mode},
		{Before: stage.goModSnapshot, After: updatedMod, Mode: stage.goModSnapshot.Mode},
	}

	if stage.goSumSnapshot.Exists || sumExists {
		changes = append(changes, fileChange{Before: stage.goSumSnapshot, After: updatedSum, Mode: installFileMode(stage.goSumSnapshot)})
	}

	if err := commitInstallChanges(changes); err != nil {
		return nil, fmt.Errorf("commit module installation: %w", err)
	}

	result.SourceChanged = !bytes.Equal(sourceSnapshot.Data, rewrite.Source)
	result.DependenciesChanged = !bytes.Equal(stage.goModSnapshot.Data, updatedMod) || stage.goSumSnapshot.Exists != sumExists || !bytes.Equal(stage.goSumSnapshot.Data, updatedSum)
	result.Changed = result.SourceChanged || result.DependenciesChanged

	return result, nil
}

func (installer *Installer) prepareProjectFerret(ctx context.Context, project *projectInfo, installMissing bool) (*installStage, bool, error) {
	if project.FerretVersion != "" {
		return nil, false, nil
	}

	if installer.ferretVersion == nil {
		return nil, false, fmt.Errorf("ferret dependency version provider is not configured")
	}

	version, err := installer.ferretVersion()
	if err != nil {
		return nil, false, fmt.Errorf("resolve Ferret dependency version: %w", err)
	}

	if _, err := parseProjectFerretVersion(version); err != nil {
		return nil, false, fmt.Errorf("resolve Ferret dependency version: %w", err)
	}

	if !installMissing {
		return nil, false, &MissingDependencyError{Path: ferretCoreModulePath, Version: version}
	}

	stage, err := newInstallStage(project)
	if err != nil {
		return nil, false, err
	}

	query := ferretCoreModulePath + "@" + version
	if _, err := installer.runner.Run(ctx, project.Root, "get", "-modfile="+stage.modPath, query); err != nil {
		os.RemoveAll(stage.directory)
		return nil, false, fmt.Errorf("resolve %s through the Go module toolchain: %w", query, err)
	}

	selected, err := installer.selectedFerretVersion(ctx, project.Root, stage.modPath)
	if err != nil {
		os.RemoveAll(stage.directory)
		return nil, false, err
	}

	if selected != version {
		os.RemoveAll(stage.directory)
		return nil, false, fmt.Errorf("go selected %s@%s instead of approved dependency %s@%s", ferretCoreModulePath, selected, ferretCoreModulePath, version)
	}

	project.FerretVersion = selected

	return stage, true, nil
}

func (installer *Installer) selectedFerretVersion(ctx context.Context, directory, modFile string) (string, error) {
	output, err := installer.runner.Run(ctx, directory, "list", "-mod=mod", "-modfile="+modFile, "-m", "-json", ferretCoreModulePath)
	if err != nil {
		return "", fmt.Errorf("inspect resolved Ferret dependency: %w", err)
	}

	var selected goModuleInfo
	if err := json.Unmarshal(output, &selected); err != nil {
		return "", fmt.Errorf("decode resolved Ferret dependency: %w", err)
	}

	if selected.Path != ferretCoreModulePath || selected.Version == "" {
		return "", fmt.Errorf("go did not select a released %s version", ferretCoreModulePath)
	}

	return selected.Version, nil
}

func (installer *Installer) resolveRelease(ctx context.Context, id, requestedVersion string, projectVersion *semver.Version) (*installRelease, error) {
	item, err := installer.registry.Module(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load registry module %q: %w", id, err)
	}

	if len(item.Versions) == 0 {
		return nil, fmt.Errorf("%w: module %q has no versions", barnregistry.ErrMalformedArtifact, id)
	}

	loaded := make(map[string]*barnregistry.Version, len(item.Versions))
	load := func(version string) (*barnregistry.Version, error) {
		if existing := loaded[version]; existing != nil {
			return existing, nil
		}

		record, err := installer.registry.Version(ctx, id, version)
		if err != nil {
			return nil, fmt.Errorf("load registry module %s@%s: %w", id, version, err)
		}

		loaded[version] = record

		return record, nil
	}

	var selected *barnregistry.Version

	if requestedVersion != "" {
		selected, err = load(requestedVersion)
		if err != nil {
			return nil, err
		}

		compatible, err := releaseSupportsFerret(selected.Ferret, projectVersion)
		if err != nil {
			return nil, fmt.Errorf("module %s@%s has unusable compatibility metadata: %w", id, selected.Version, err)
		}

		if !compatible {
			return nil, fmt.Errorf(
				"module %s@%s requires Ferret %s; project selects Ferret %s",
				id,
				selected.Version,
				selected.Ferret,
				projectVersion.Original(),
			)
		}
	} else {
		for _, available := range item.Versions {
			record, loadErr := load(available.Version)
			if loadErr != nil {
				return nil, loadErr
			}

			compatible, compatibilityErr := releaseSupportsFerret(record.Ferret, projectVersion)
			if compatibilityErr != nil {
				return nil, fmt.Errorf("module %s@%s has unusable compatibility metadata: %w", id, record.Version, compatibilityErr)
			}

			if compatible {
				selected = record
				break
			}
		}

		if selected == nil {
			return nil, fmt.Errorf("module %s has no release compatible with project Ferret %s", id, projectVersion.Original())
		}
	}

	historicalPackages := make(map[string]struct{}, len(item.Versions))
	for _, available := range item.Versions {
		record, loadErr := load(available.Version)
		if loadErr != nil {
			return nil, loadErr
		}

		if record.Package.Path == "" {
			return nil, fmt.Errorf("module %s@%s does not declare package.path", id, record.Version)
		}

		historicalPackages[record.Package.Path] = struct{}{}
	}

	return &installRelease{Version: selected, HistoricalPackages: historicalPackages}, nil
}

func (installer *Installer) hasExactDependency(ctx context.Context, directory, packagePath, version string) (bool, error) {
	output, err := installer.runner.Run(ctx, directory, "list", "-m", "-json", packagePath)
	if err != nil {
		return false, nil
	}

	var selected goModuleInfo
	if err := json.Unmarshal(output, &selected); err != nil {
		return false, fmt.Errorf("decode selected module %s: %w", packagePath, err)
	}

	return selected.Path == packagePath && selected.Version == "v"+version && selected.Replace == nil, nil
}

func (installer *Installer) validateResolvedModule(ctx context.Context, project *projectInfo, modFile string, release *barnregistry.Version) error {
	moduleOutput, err := installer.runner.Run(ctx, project.Root, "list", "-mod=mod", "-modfile="+modFile, "-m", "-json", release.Package.Path)
	if err != nil {
		return fmt.Errorf("inspect resolved module %s: %w", release.Package.Path, err)
	}

	var selected goModuleInfo
	if err := json.Unmarshal(moduleOutput, &selected); err != nil {
		return fmt.Errorf("decode resolved module %s: %w", release.Package.Path, err)
	}

	wantedVersion := "v" + release.Version
	if selected.Path != release.Package.Path || selected.Version != wantedVersion {
		return fmt.Errorf("go resolved %s@%s instead of registry release %s@%s", selected.Path, selected.Version, release.Package.Path, wantedVersion)
	}

	if selected.Replace != nil {
		return fmt.Errorf("project replaces registry package %s; remove the replace directive before installing %s@%s", release.Package.Path, release.ID, release.Version)
	}

	ferretOutput, err := installer.runner.Run(ctx, project.Root, "list", "-mod=mod", "-modfile="+modFile, "-m", "-json", ferretCoreModulePath)
	if err != nil {
		return fmt.Errorf("inspect resolved Ferret dependency: %w", err)
	}

	var ferretModule goModuleInfo
	if err := json.Unmarshal(ferretOutput, &ferretModule); err != nil {
		return fmt.Errorf("decode resolved Ferret dependency: %w", err)
	}

	if ferretModule.Version != project.FerretVersion {
		return fmt.Errorf(
			"installing %s@%s would change project Ferret from %s to %s",
			release.ID,
			release.Version,
			project.FerretVersion,
			ferretModule.Version,
		)
	}

	downloadOutput, err := installer.runner.Run(ctx, project.Root, "mod", "download", "-json", "-modfile="+modFile, release.Package.Path+"@"+wantedVersion)
	if err != nil {
		return fmt.Errorf("download exact module release %s@%s: %w", release.Package.Path, wantedVersion, err)
	}

	var download goDownloadInfo
	if err := json.Unmarshal(downloadOutput, &download); err != nil {
		return fmt.Errorf("decode downloaded module metadata: %w", err)
	}

	if download.Error != "" {
		return fmt.Errorf("download %s@%s: %s", release.Package.Path, wantedVersion, download.Error)
	}

	if download.Path != release.Package.Path || download.Version != wantedVersion {
		return fmt.Errorf("go downloaded %s@%s instead of %s@%s", download.Path, download.Version, release.Package.Path, wantedVersion)
	}

	if download.Origin != nil && download.Origin.Hash != "" && !strings.EqualFold(download.Origin.Hash, release.Source.Commit) {
		return fmt.Errorf(
			"registry commit %s for %s@%s does not match Go module origin %s",
			release.Source.Commit,
			release.ID,
			release.Version,
			download.Origin.Hash,
		)
	}

	return nil
}
