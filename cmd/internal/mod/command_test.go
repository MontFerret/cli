package mod

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
	registryspec "github.com/MontFerret/specs/pkg/registry"

	"github.com/MontFerret/cli/v2/pkg/config"
	modulelifecycle "github.com/MontFerret/cli/v2/pkg/module"
)

func TestModCommandSearchRendersTableAndEmptyResult(t *testing.T) {
	service := &fakeModuleService{searchResults: []modulelifecycle.SearchResult{{
		Name: "acme/sqlite", Version: "1.2.3", Description: "SQLite integration",
	}}}

	output, err := executeModCommand(t, service, "search", "sqlite")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output, "NAME") || !strings.Contains(output, "acme/sqlite") || !strings.Contains(output, "SQLite integration") {
		t.Fatalf("unexpected output:\n%s", output)
	}

	service.searchResults = nil
	output, err = executeModCommand(t, service, "search")
	if err != nil {
		t.Fatal(err)
	}

	if output != "No modules found.\n" {
		t.Fatalf("unexpected empty output: %q", output)
	}
}

func TestModCommandInfoRendersOptionalFields(t *testing.T) {
	service := &fakeModuleService{info: &modulelifecycle.ModuleInfo{
		Name: "acme/sqlite", Description: "SQLite", Newest: "1.0.0-rc.1", SelectedVersion: "1.0.0-rc.1",
		Versions: []string{"1.0.0-rc.1"}, Namespace: "DB::SQLITE", Repository: "https://example.com/acme/sqlite",
		Commit: "abc", Documentation: "https://registry.example/docs.md",
	}}
	output, err := executeModCommand(t, service, "info", "acme/sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Latest stable: (none)") || !strings.Contains(output, "Namespace: DB::SQLITE") || strings.Contains(output, "License:") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestModCommandInitRequiresAndPassesFlags(t *testing.T) {
	service := &fakeModuleService{create: &modulelifecycle.CreateResult{Directory: "/tmp/sqlite", Namespace: "DB::SQLITE"}}
	if _, err := executeModCommand(t, service, "init", "db/sqlite"); err == nil || !strings.Contains(err.Error(), "--go-module is required") {
		t.Fatalf("unexpected missing flag error: %v", err)
	}

	output, err := executeModCommand(t, service,
		"init", "db/sqlite",
		"--go-module", "example.com/sqlite",
		"--dir", "custom",
		"--namespace", "DB::SQLITE",
	)
	if err != nil {
		t.Fatal(err)
	}
	if service.createOptions.Name != "db/sqlite" || service.createOptions.GoModule != "example.com/sqlite" || service.createOptions.Directory != "custom" || service.createOptions.Namespace != "DB::SQLITE" {
		t.Fatalf("unexpected create options: %#v", service.createOptions)
	}
	if !strings.Contains(output, "Created Ferret module") || !strings.Contains(output, "go mod tidy") {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestModCommandPublishRendersRecords(t *testing.T) {
	service := &fakeModuleService{publication: &barnpublish.Result{
		Kind: barnpublish.NewModule,
		Module: &registryspec.ModuleManifest{
			Source: registryspec.Source{Repository: "https://example.com/acme/widget", Path: "modules/widget"},
		},
		Version: &registryspec.VersionRecord{Version: "1.2.3", Tag: "release-1.2.3", Commit: "abc"},
		Files: []barnpublish.File{
			{Path: "registry/modules/acme/widget/manifest.json", Content: []byte("{}\n")},
			{Path: "registry/modules/acme/widget/versions/v1.2.3.json", Content: []byte("{}\n")},
		},
	}}
	output, err := executeModCommand(t, service, "publish", "--tag", "release-1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if service.publishOptions.Tag != "release-1.2.3" || !strings.Contains(output, "Manifest: valid") || !strings.Contains(output, service.publication.Files[1].Path) || !strings.Contains(output, "Add both records") {
		t.Fatalf("unexpected publication output:\n%s", output)
	}
}

func TestModCommandPublishRendersOnlyNewVersionFile(t *testing.T) {
	service := &fakeModuleService{publication: &barnpublish.Result{
		Kind: barnpublish.NewVersion,
		Module: &registryspec.ModuleManifest{
			Source: registryspec.Source{Repository: "https://example.com/acme/widget"},
		},
		Version: &registryspec.VersionRecord{Version: "1.2.4", Tag: "v1.2.4", Commit: "def"},
		Files: []barnpublish.File{{
			Path: "registry/modules/acme/widget/versions/v1.2.4.json", Content: []byte("{\"version\":\"1.2.4\"}\n"),
		}},
	}}

	output, err := executeModCommand(t, service, "publish")
	if err != nil {
		t.Fatal(err)
	}
	if service.publishOptions.Tag != "" || strings.Contains(output, "manifest.json") || !strings.Contains(output, service.publication.Files[0].Path) || !strings.Contains(output, "without modifying published records") {
		t.Fatalf("unexpected new-version output:\n%s", output)
	}
}

func TestModCommandExposesOnlyRenamedLifecycleSubcommands(t *testing.T) {
	command := New(newTestStore(t, t.TempDir()), &fakeModuleService{})
	if command.Name() != "mod" || len(command.Aliases) != 0 {
		t.Fatalf("unexpected command name or aliases: %q %#v", command.Name(), command.Aliases)
	}

	want := map[string]bool{"search": true, "info": true, "init": true, "publish": true}
	for _, child := range command.Commands() {
		delete(want, child.Name())
		if len(child.Aliases) != 0 {
			t.Fatalf("unexpected aliases for %q: %#v", child.Name(), child.Aliases)
		}
		if child.Name() == "add" || child.Name() == "create" || child.Name() == "install" || child.Name() == "update" {
			t.Fatalf("unexpected dependency command %q", child.Name())
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing subcommands: %#v", want)
	}

	command.SetArgs([]string{"create", "db/sqlite"})
	if err := command.Execute(); err == nil {
		t.Fatal("expected removed create command to fail")
	}
}

func TestModCommandPropagatesServiceErrors(t *testing.T) {
	want := errors.New("registry unavailable")
	_, err := executeModCommand(t, &fakeModuleService{err: want}, "search")
	if !errors.Is(err, want) {
		t.Fatalf("expected service error, got %v", err)
	}
}

func executeModCommand(t *testing.T, service Service, args ...string) (string, error) {
	t.Helper()

	command := New(newTestStore(t, t.TempDir()), service)
	output := new(bytes.Buffer)
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs(args)

	err := command.Execute()
	return output.String(), err
}

func newTestStore(t *testing.T, home string) *config.Store {
	t.Helper()

	t.Setenv("HOME", home)
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	store, err := config.NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	return store
}
