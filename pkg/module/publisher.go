package module

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	modulemanifest "github.com/MontFerret/specs/pkg/module"
	registryspec "github.com/MontFerret/specs/pkg/registry"
)

// Publisher prepares immutable Barn registry records from local Git state.
type Publisher struct {
	git GitRunner
}

// NewPublisher constructs a publication preparer.
func NewPublisher(git GitRunner) *Publisher {
	if git == nil {
		git = NewGit()
	}

	return &Publisher{git: git}
}

// Prepare validates the current module release and builds Barn registration records.
func (p *Publisher) Prepare(ctx context.Context, options PublishOptions) (*Publication, error) {
	directory := options.Directory
	if directory == "" {
		directory = "."
	}

	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve module directory: %w", err)
	}

	manifestPath := filepath.Join(directory, modulemanifest.ManifestFilename)
	manifest, err := ValidateManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	if err := validatePublicationMetadata(manifest); err != nil {
		return nil, err
	}

	for _, filename := range []string{"README.md", "go.mod"} {
		filePath := filepath.Join(directory, filename)
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("required publication file %q: %w", filePath, err)
		}

		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("required publication file %q is not a regular file", filePath)
		}
	}

	repositoryRoot, err := p.git.Run(ctx, directory, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("directory %q is not a Git repository: %w", directory, err)
	}

	repositoryRoot, err = filepath.Abs(repositoryRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve Git repository root: %w", err)
	}

	sourcePath, err := filepath.Rel(repositoryRoot, directory)
	if err != nil || sourcePath == ".." || strings.HasPrefix(sourcePath, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("module directory %q is outside Git repository %q", directory, repositoryRoot)
	}

	if sourcePath == "." {
		sourcePath = ""
	} else {
		sourcePath = filepath.ToSlash(sourcePath)
	}

	statusPath := sourcePath
	if statusPath == "" {
		statusPath = "."
	}

	status, err := p.git.Run(ctx, repositoryRoot, "status", "--porcelain", "--untracked-files=all", "--", statusPath)
	if err != nil {
		return nil, fmt.Errorf("inspect module Git status: %w", err)
	}

	if status != "" {
		return nil, fmt.Errorf("module directory has uncommitted changes; commit or discard them before publishing")
	}

	remote, err := p.git.Run(ctx, repositoryRoot, "config", "--local", "--get", "remote.origin.url")
	if err != nil || remote == "" {
		return nil, fmt.Errorf("Git repository has no usable origin remote")
	}

	repositoryURL, err := normalizeGitRemote(remote)
	if err != nil {
		return nil, fmt.Errorf("normalize origin remote %q: %w", remote, err)
	}

	if err := validateManifestRepository(manifest, repositoryURL, sourcePath); err != nil {
		return nil, err
	}

	tag := options.Tag
	if tag == "" {
		tag = "v" + manifest.Version
		if sourcePath != "" {
			tag = path.Join(sourcePath, tag)
		}
	}

	headCommit, err := p.git.Run(ctx, repositoryRoot, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return nil, fmt.Errorf("resolve HEAD commit: %w", err)
	}

	tagObject := "refs/tags/" + tag + "^{commit}"
	tagCommit, err := p.git.Run(ctx, repositoryRoot, "rev-parse", "--verify", tagObject)
	if err != nil {
		return nil, fmt.Errorf("resolve release tag %q: %w", tag, err)
	}

	if tagCommit != headCommit {
		return nil, fmt.Errorf("release tag %q resolves to %s, not HEAD %s", tag, tagCommit, headCommit)
	}

	for _, filename := range []string{modulemanifest.ManifestFilename, "README.md", "go.mod"} {
		trackedPath := path.Join(sourcePath, filename)
		if _, err := p.git.Run(ctx, repositoryRoot, "cat-file", "-e", tagObject+":"+trackedPath); err != nil {
			return nil, fmt.Errorf("release tag %q does not contain required file %s", tag, trackedPath)
		}
	}

	taggedManifestData, err := p.git.Run(ctx, repositoryRoot, "show", tagObject+":"+path.Join(sourcePath, modulemanifest.ManifestFilename))
	if err != nil {
		return nil, fmt.Errorf("read tagged module manifest: %w", err)
	}

	taggedManifest, err := modulemanifest.Parse([]byte(taggedManifestData))
	if err != nil {
		return nil, fmt.Errorf("validate tagged module manifest: %w", err)
	}

	if !reflect.DeepEqual(manifest, taggedManifest) {
		return nil, fmt.Errorf("working module manifest does not match release tag %q", tag)
	}

	owner, name, _ := strings.Cut(manifest.Name, "/")
	registryManifest := &registryspec.ModuleManifest{
		Schema: registryspec.ModuleManifestSchemaV1,
		Owner:  owner,
		Name:   name,
		Source: registryspec.Source{Repository: repositoryURL, Path: sourcePath},
	}
	versionRecord := &registryspec.VersionRecord{
		Schema:  registryspec.VersionRecordSchemaV1,
		Version: manifest.Version,
		Tag:     tag,
		Commit:  tagCommit,
	}

	if err := registryspec.ValidateModuleManifest(registryManifest); err != nil {
		return nil, fmt.Errorf("validate registry module record: %w", err)
	}

	if err := registryspec.ValidateVersionRecord(versionRecord); err != nil {
		return nil, fmt.Errorf("validate registry version record: %w", err)
	}

	registryManifestJSON, err := marshalRegistryRecord(registryManifest)
	if err != nil {
		return nil, err
	}

	versionRecordJSON, err := marshalRegistryRecord(versionRecord)
	if err != nil {
		return nil, err
	}

	moduleRoot := path.Join("registry/modules", owner, name)
	return &Publication{
		Name:               manifest.Name,
		Repository:         repositoryURL,
		SourcePath:         sourcePath,
		Version:            manifest.Version,
		Tag:                tag,
		Commit:             tagCommit,
		ModuleManifestPath: path.Join(moduleRoot, "manifest.json"),
		ModuleManifestJSON: registryManifestJSON,
		VersionRecordPath:  path.Join(moduleRoot, "versions", "v"+manifest.Version+".json"),
		VersionRecordJSON:  versionRecordJSON,
	}, nil
}
