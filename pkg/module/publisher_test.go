package module

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	registryspec "github.com/MontFerret/specs/pkg/registry"
)

func TestPublisherPreparesValidatedRecords(t *testing.T) {
	repository := newPublicationRepository(t, publicationFixtureOptions{})
	publication, err := NewPublisher(nil).Prepare(context.Background(), PublishOptions{Directory: repository})
	if err != nil {
		t.Fatal(err)
	}

	if publication.Name != "acme/widget" || publication.Repository != "https://github.com/acme/widget" || publication.Tag != "v1.2.3" {
		t.Fatalf("unexpected publication: %#v", publication)
	}
	if publication.ModuleManifestPath != "registry/modules/acme/widget/manifest.json" || publication.VersionRecordPath != "registry/modules/acme/widget/versions/v1.2.3.json" {
		t.Fatalf("unexpected publication paths: %#v", publication)
	}
	if _, err := registryspec.ParseModuleManifest(publication.ModuleManifestJSON); err != nil {
		t.Fatalf("invalid module record: %v", err)
	}
	record, err := registryspec.ParseVersionRecord(publication.VersionRecordJSON)
	if err != nil {
		t.Fatalf("invalid version record: %v", err)
	}
	if record.Commit != publication.Commit {
		t.Fatalf("unexpected commit: %#v", record)
	}
}

func TestPublisherSupportsMonorepoAndCustomTag(t *testing.T) {
	repository := newPublicationRepository(t, publicationFixtureOptions{SourcePath: "modules/widget", Tag: "release/widget-1.2.3"})
	moduleDirectory := filepath.Join(repository, "modules", "widget")

	publication, err := NewPublisher(nil).Prepare(context.Background(), PublishOptions{
		Directory: moduleDirectory,
		Tag:       "release/widget-1.2.3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if publication.SourcePath != "modules/widget" || !strings.Contains(string(publication.ModuleManifestJSON), `"path": "modules/widget"`) {
		t.Fatalf("unexpected monorepo publication: %#v", publication)
	}
}

func TestPublisherRejectsInvalidLocalState(t *testing.T) {
	t.Run("missing manifest", func(t *testing.T) {
		_, err := NewPublisher(nil).Prepare(context.Background(), PublishOptions{Directory: t.TempDir()})
		if err == nil || !strings.Contains(err.Error(), "ferret.yaml") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not a repository", func(t *testing.T) {
		directory := t.TempDir()
		writePublicationFiles(t, directory, "")
		_, err := NewPublisher(nil).Prepare(context.Background(), PublishOptions{Directory: directory})
		if err == nil || !strings.Contains(err.Error(), "not a Git repository") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("placeholder metadata", func(t *testing.T) {
		directory := t.TempDir()
		if err := os.WriteFile(filepath.Join(directory, "ferret.yaml"), []byte(`$schema: https://schemas.ferretlang.org/module/v1.json
name: acme/widget
namespace: Widget
version: 1.2.3
description: "TODO: describe widget"
license: LicenseRef-TODO
documentation: https://example.invalid/TODO
`), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := NewPublisher(nil).Prepare(context.Background(), PublishOptions{Directory: directory})
		if err == nil || !strings.Contains(err.Error(), "TODO placeholder") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("dirty module", func(t *testing.T) {
		repository := newPublicationRepository(t, publicationFixtureOptions{})
		if err := os.WriteFile(filepath.Join(repository, "README.md"), []byte("changed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := NewPublisher(nil).Prepare(context.Background(), PublishOptions{Directory: repository})
		if err == nil || !strings.Contains(err.Error(), "uncommitted changes") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing origin", func(t *testing.T) {
		repository := newPublicationRepository(t, publicationFixtureOptions{NoRemote: true})
		_, err := NewPublisher(nil).Prepare(context.Background(), PublishOptions{Directory: repository})
		if err == nil || !strings.Contains(err.Error(), "origin remote") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing tag", func(t *testing.T) {
		repository := newPublicationRepository(t, publicationFixtureOptions{})
		runTestGit(t, repository, "tag", "-d", "v1.2.3")
		_, err := NewPublisher(nil).Prepare(context.Background(), PublishOptions{Directory: repository})
		if err == nil || !strings.Contains(err.Error(), "resolve release tag") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("repository mismatch", func(t *testing.T) {
		repository := newPublicationRepository(t, publicationFixtureOptions{})
		runTestGit(t, repository, "remote", "set-url", "origin", "https://example.com/acme/other.git")
		_, err := NewPublisher(nil).Prepare(context.Background(), PublishOptions{Directory: repository})
		if err == nil || !strings.Contains(err.Error(), "does not match Git origin") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("tag does not point to HEAD", func(t *testing.T) {
		repository := newPublicationRepository(t, publicationFixtureOptions{})
		if err := os.WriteFile(filepath.Join(repository, "other.txt"), []byte("next\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		runTestGit(t, repository, "add", "other.txt")
		runTestGit(t, repository, "commit", "-m", "next")
		_, err := NewPublisher(nil).Prepare(context.Background(), PublishOptions{Directory: repository})
		if err == nil || !strings.Contains(err.Error(), "not HEAD") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNormalizeGitRemote(t *testing.T) {
	for input, want := range map[string]string{
		"https://github.com/acme/widget.git":         "https://github.com/acme/widget",
		"git@github.com:acme/widget.git":             "https://github.com/acme/widget",
		"ssh://git@git.example:2222/acme/widget.git": "https://git.example:2222/acme/widget",
	} {
		got, err := normalizeGitRemote(input)
		if err != nil {
			t.Fatalf("normalize %q: %v", input, err)
		}
		if got != want {
			t.Fatalf("normalize %q: want %q, got %q", input, want, got)
		}
	}
}

type publicationFixtureOptions struct {
	SourcePath string
	Tag        string
	NoRemote   bool
}

func newPublicationRepository(t *testing.T, options publicationFixtureOptions) string {
	t.Helper()

	repository := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	runTestGit(t, repository, "init")
	runTestGit(t, repository, "config", "user.name", "Ferret Test")
	runTestGit(t, repository, "config", "user.email", "ferret@example.invalid")
	if !options.NoRemote {
		runTestGit(t, repository, "remote", "add", "origin", "git@github.com:acme/widget.git")
	}

	moduleDirectory := repository
	if options.SourcePath != "" {
		moduleDirectory = filepath.Join(repository, filepath.FromSlash(options.SourcePath))
		if err := os.MkdirAll(moduleDirectory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writePublicationFiles(t, moduleDirectory, options.SourcePath)
	runTestGit(t, repository, "add", ".")
	runTestGit(t, repository, "commit", "-m", "release")

	tag := options.Tag
	if tag == "" {
		tag = "v1.2.3"
		if options.SourcePath != "" {
			tag = options.SourcePath + "/" + tag
		}
	}
	runTestGit(t, repository, "tag", tag)

	return repository
}

func writePublicationFiles(t *testing.T, directory, sourcePath string) {
	t.Helper()

	repository := ""
	if sourcePath != "" {
		repository = fmt.Sprintf("repository:\n  url: https://github.com/acme/widget\n  directory: %s\n", sourcePath)
	} else {
		repository = "repository:\n  url: https://github.com/acme/widget\n"
	}
	manifest := fmt.Sprintf(`$schema: https://schemas.ferretlang.org/module/v1.json
name: acme/widget
namespace: Widget
version: 1.2.3
description: Widget integration for Ferret.
license: MIT
documentation: https://example.com/acme/widget
%s`, repository)

	for name, data := range map[string]string{
		"ferret.yaml": manifest,
		"README.md":   "# Widget\n",
		"go.mod":      "module github.com/acme/widget\n\ngo 1.25.0\n",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func runTestGit(t *testing.T, directory string, args ...string) string {
	t.Helper()

	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}

	return strings.TrimSpace(string(output))
}
