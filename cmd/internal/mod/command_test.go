package mod

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
	registryspec "github.com/MontFerret/specs/pkg/registry"

	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/module/discovery"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

func TestModCommandSearchRendersTableAndEmptyResult(t *testing.T) {
	service := &fakeModuleService{searchResults: []discovery.SearchResult{{
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
	service := &fakeModuleService{info: &discovery.ModuleInfo{
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

func TestModCommandInstallPassesReferenceAndRendersChanges(t *testing.T) {
	service := &fakeModuleService{install: &install.Result{
		ID:                  "montferret/archive",
		Version:             "1.0.0-rc.3",
		PackagePath:         "github.com/MontFerret/contrib/modules/archive",
		FerretConstraint:    ">=2.0.0-alpha.43 <3.0.0",
		ProjectFerret:       "v2.0.0-alpha.44",
		EditedFile:          "main.go",
		Changed:             true,
		SourceChanged:       true,
		DependenciesChanged: true,
	}}

	output, err := executeModCommand(t, service, "install", "montferret/archive@1.0.0-rc.3")
	if err != nil {
		t.Fatal(err)
	}
	if service.installOptions.Reference != "montferret/archive@1.0.0-rc.3" || service.installOptions.Directory != "." {
		t.Fatalf("unexpected install options: %#v", service.installOptions)
	}
	for _, expected := range []string{
		"Resolving montferret/archive@1.0.0-rc.3...",
		"Resolved montferret/archive@1.0.0-rc.3",
		"Compatible with project Ferret v2.0.0-alpha.44",
		"Package: github.com/MontFerret/contrib/modules/archive",
		"Updated Go module dependencies",
		"Registered module in main.go",
		"Validated owning package build",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected %q in output:\n%s", expected, output)
		}
	}
}

func TestModCommandInstallRendersIdempotentResult(t *testing.T) {
	service := &fakeModuleService{install: &install.Result{
		ID: "montferret/archive", Version: "1.0.0-rc.3", PackagePath: "example.com/archive",
		FerretConstraint: ">=2.0.0 <3.0.0", ProjectFerret: "v2.1.0",
	}}

	output, err := executeModCommand(t, service, "install", "montferret/archive")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "montferret/archive@1.0.0-rc.3 is already installed") || strings.Contains(output, "Validated owning package build") {
		t.Fatalf("unexpected idempotent output:\n%s", output)
	}
}

func TestModCommandInstallApprovesCombinedProjectSetupInteractively(t *testing.T) {
	missing := errors.Join(
		&install.MissingDependencyError{
			Path:    "github.com/MontFerret/ferret/v2",
			Version: "v2.0.0-alpha.44",
		},
		&install.MissingCompositionError{File: "ferret.go", Package: "xproject"},
	)
	service := &fakeModuleService{installSequence: []fakeInstallResponse{
		{err: missing},
		{result: &install.Result{
			ID:                    "montferret/postgres",
			Version:               "1.0.0",
			PackagePath:           "example.com/postgres",
			FerretConstraint:      ">=2.0.0-alpha.43 <3.0.0",
			ProjectFerret:         "v2.0.0-alpha.44",
			EditedFile:            "ferret.go",
			Changed:               true,
			SourceChanged:         true,
			DependenciesChanged:   true,
			FerretDependencyAdded: true,
			CompositionScaffolded: true,
		}},
	}}

	stdout, stderr, err := executeInteractiveModCommand(t, service, "\n", "install", "montferret/postgres")
	if err != nil {
		t.Fatal(err)
	}
	if service.installCalls != 2 || len(service.installHistory) != 2 {
		t.Fatalf("unexpected install calls: %d %#v", service.installCalls, service.installHistory)
	}
	if service.installHistory[0].InstallMissingDependencies ||
		service.installHistory[0].ScaffoldMissingComposition ||
		!service.installHistory[1].InstallMissingDependencies ||
		!service.installHistory[1].ScaffoldMissingComposition {
		t.Fatalf("unexpected approval sequence: %#v", service.installHistory)
	}
	for _, expected := range []string{
		"Project setup required",
		"Add dependency: github.com/MontFerret/ferret/v2@v2.0.0-alpha.44",
		"Create composition helper: ferret.go (package xproject)",
		"Apply project setup? [Y/n]:",
	} {
		if !strings.Contains(stderr, expected) {
			t.Fatalf("expected %q in stderr:\n%s", expected, stderr)
		}
	}
	if !strings.Contains(stdout, "Resolving montferret/postgres...") ||
		!strings.Contains(stdout, "Added github.com/MontFerret/ferret/v2 v2.0.0-alpha.44") ||
		!strings.Contains(stdout, "Created Ferret composition helper in ferret.go") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestModCommandInstallApprovesMissingCompositionOnly(t *testing.T) {
	service := &fakeModuleService{installSequence: []fakeInstallResponse{
		{err: &install.MissingCompositionError{File: "internal/runtime/ferret.go", Package: "runtime"}},
		{result: &install.Result{
			ID: "montferret/postgres", Version: "1.0.0", PackagePath: "example.com/postgres",
			FerretConstraint: ">=2.0.0 <3.0.0", ProjectFerret: "v2.0.0",
			EditedFile: "internal/runtime/ferret.go", Changed: true, SourceChanged: true, CompositionScaffolded: true,
		}},
	}}

	stdout, stderr, err := executeInteractiveModCommand(t, service, "yes\n", "install", "montferret/postgres")
	if err != nil {
		t.Fatal(err)
	}
	if service.installCalls != 2 || service.installHistory[1].InstallMissingDependencies || !service.installHistory[1].ScaffoldMissingComposition {
		t.Fatalf("unexpected scaffold approval: %#v", service.installHistory)
	}
	if strings.Contains(stderr, "Add dependency:") || !strings.Contains(stderr, "Create composition helper: internal/runtime/ferret.go (package runtime)") {
		t.Fatalf("unexpected stderr:\n%s", stderr)
	}
	if !strings.Contains(stdout, "Created Ferret composition helper in internal/runtime/ferret.go") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestModCommandInstallYesFlagsApproveWithoutPrompt(t *testing.T) {
	for _, flag := range []string{"-y", "--yes"} {
		t.Run(flag, func(t *testing.T) {
			service := &fakeModuleService{install: &install.Result{
				ID: "montferret/postgres", Version: "1.0.0", PackagePath: "example.com/postgres",
				FerretConstraint: ">=2.0.0 <3.0.0", ProjectFerret: "v2.0.0",
			}}

			if _, err := executeModCommand(t, service, "install", "montferret/postgres", flag); err != nil {
				t.Fatal(err)
			}
			if service.installCalls != 1 || !service.installOptions.InstallMissingDependencies || !service.installOptions.ScaffoldMissingComposition {
				t.Fatalf("unexpected install request: calls=%d options=%#v", service.installCalls, service.installOptions)
			}
		})
	}
}

func TestModCommandInstallRetriesInvalidDependencyAnswer(t *testing.T) {
	missing := &install.MissingDependencyError{Path: "github.com/MontFerret/ferret/v2", Version: "v2.0.0"}
	service := &fakeModuleService{installSequence: []fakeInstallResponse{
		{err: missing},
		{result: &install.Result{
			ID: "montferret/postgres", Version: "1.0.0", PackagePath: "example.com/postgres",
			FerretConstraint: ">=2.0.0 <3.0.0", ProjectFerret: "v2.0.0",
		}},
	}}

	_, stderr, err := executeInteractiveModCommand(t, service, "maybe\nyes\n", "install", "montferret/postgres")
	if err != nil {
		t.Fatal(err)
	}
	if service.installCalls != 2 || !strings.Contains(stderr, "Please answer yes or no.") {
		t.Fatalf("unexpected retry: calls=%d stderr=%q", service.installCalls, stderr)
	}
}

func TestModCommandInstallCancellationDoesNotRetry(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
	}{
		{name: "decline", input: "no\n"},
		{name: "eof"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeModuleService{installSequence: []fakeInstallResponse{{err: &install.MissingDependencyError{
				Path: "github.com/MontFerret/ferret/v2", Version: "v2.0.0",
			}}}}

			stdout, stderr, err := executeInteractiveModCommand(t, service, test.input, "install", "montferret/postgres")
			if err != nil {
				t.Fatal(err)
			}
			if service.installCalls != 1 || !strings.Contains(stderr, "Module installation canceled.") {
				t.Fatalf("unexpected cancellation: calls=%d stderr=%q", service.installCalls, stderr)
			}
			if stdout != "Resolving montferret/postgres...\n" {
				t.Fatalf("unexpected stdout: %q", stdout)
			}
		})
	}
}

func TestModCommandInstallNonTerminalRequiresApproval(t *testing.T) {
	service := &fakeModuleService{installSequence: []fakeInstallResponse{{err: &install.MissingDependencyError{
		Path: "github.com/MontFerret/ferret/v2", Version: "v2.0.0-alpha.44",
	}}}}

	_, err := executeModCommand(t, service, "install", "montferret/postgres")
	if err == nil || !strings.Contains(err.Error(), "stdin is not a terminal") || !strings.Contains(err.Error(), "--yes") || !strings.Contains(err.Error(), "go get github.com/MontFerret/ferret/v2@v2.0.0-alpha.44") {
		t.Fatalf("unexpected non-terminal error: %v", err)
	}
	if service.installCalls != 1 {
		t.Fatalf("unexpected install calls: %d", service.installCalls)
	}
}

func TestModCommandInstallNonTerminalReportsCombinedSetup(t *testing.T) {
	service := &fakeModuleService{installSequence: []fakeInstallResponse{{err: errors.Join(
		&install.MissingDependencyError{Path: "github.com/MontFerret/ferret/v2", Version: "v2.0.0-alpha.44"},
		&install.MissingCompositionError{File: "ferret.go", Package: "xproject"},
	)}}}

	_, err := executeModCommand(t, service, "install", "montferret/postgres")
	if err == nil ||
		!strings.Contains(err.Error(), "go get github.com/MontFerret/ferret/v2@v2.0.0-alpha.44") ||
		!strings.Contains(err.Error(), "create ferret.go with NewFerret in package xproject") {
		t.Fatalf("unexpected combined setup error: %v", err)
	}
	if service.installCalls != 1 {
		t.Fatalf("unexpected install calls: %d", service.installCalls)
	}
}

func TestModCommandInstallDoesNotPromptWhenDependenciesExist(t *testing.T) {
	service := &fakeModuleService{install: &install.Result{
		ID: "montferret/postgres", Version: "1.0.0", PackagePath: "example.com/postgres",
		FerretConstraint: ">=2.0.0 <3.0.0", ProjectFerret: "v2.0.0",
	}}

	stdout, stderr, err := executeInteractiveModCommand(t, service, "", "install", "montferret/postgres")
	if err != nil {
		t.Fatal(err)
	}
	if service.installCalls != 1 || stderr != "" || !strings.Contains(stdout, "Resolved montferret/postgres@1.0.0") {
		t.Fatalf("unexpected install behavior: calls=%d stdout=%q stderr=%q", service.installCalls, stdout, stderr)
	}
}

func TestModCommandInitRequiresAndPassesFlags(t *testing.T) {
	service := &fakeModuleService{create: &scaffold.Result{Directory: "/tmp/sqlite", Namespace: "DB::SQLITE"}}
	if _, err := executeModCommand(t, service, "init", "db/sqlite"); err == nil || !strings.Contains(err.Error(), "--go-module") || !strings.Contains(err.Error(), "not a terminal") {
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

func TestModCommandInitGuidesFullInteractiveFlow(t *testing.T) {
	service := &fakeModuleService{create: &scaffold.Result{Directory: "/tmp/sqlite", Namespace: "DB::SQLITE"}}
	stdout, stderr, err := executeInteractiveModCommand(
		t,
		service,
		"acme/sqlite\n\n\nDB::SQLITE\n\n",
		"init",
	)
	if err != nil {
		t.Fatal(err)
	}

	want := scaffold.Options{
		Name:      "acme/sqlite",
		GoModule:  "github.com/acme/ferret-sqlite",
		Directory: "sqlite",
		Namespace: "DB::SQLITE",
	}
	if service.createOptions != want || service.createCalls != 1 {
		t.Fatalf("unexpected create request: options=%#v calls=%d", service.createOptions, service.createCalls)
	}
	for _, expected := range []string{
		"Ferret module name",
		"Registry identity used for distribution and discovery",
		"Go module path (--go-module)",
		"Go module [github.com/acme/ferret-sqlite]",
		"Destination directory (--dir)",
		"Directory [sqlite]",
		"Runtime namespace (--namespace)",
		"Namespace [sqlite]",
		"Module configuration:",
		"Namespace: DB::SQLITE",
		"Create module? [Y/n]",
	} {
		if !strings.Contains(stderr, expected) {
			t.Fatalf("expected %q in prompt output:\n%s", expected, stderr)
		}
	}
	if strings.Contains(stdout, "Go module path") || !strings.Contains(stdout, "Created Ferret module") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestModCommandInitPromptsOnlyForOmittedValues(t *testing.T) {
	service := &fakeModuleService{create: &scaffold.Result{Directory: "/tmp/custom", Namespace: "DB::SQLITE"}}
	_, stderr, err := executeInteractiveModCommand(
		t,
		service,
		"\ncustom\n\n",
		"init",
		"acme/sqlite",
		"--namespace",
		"DB::SQLITE",
	)
	if err != nil {
		t.Fatal(err)
	}

	if service.createOptions.GoModule != "github.com/acme/ferret-sqlite" || service.createOptions.Directory != "custom" || service.createOptions.Namespace != "DB::SQLITE" {
		t.Fatalf("unexpected create options: %#v", service.createOptions)
	}
	if strings.Contains(stderr, "Ferret module name") || strings.Contains(stderr, "Runtime namespace (--namespace)") {
		t.Fatalf("explicit values were prompted again:\n%s", stderr)
	}
}

func TestModCommandInitRetriesInvalidInteractiveAnswers(t *testing.T) {
	service := &fakeModuleService{create: &scaffold.Result{Directory: "/tmp/sqlite", Namespace: "sqlite"}}
	_, stderr, err := executeInteractiveModCommand(
		t,
		service,
		"invalid\nacme/sqlite\nnot-a-module\n\n\n\n\n",
		"init",
	)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Count(stderr, "Invalid value:") != 2 {
		t.Fatalf("expected two validation retries:\n%s", stderr)
	}
	if service.createCalls != 1 || service.createOptions.GoModule != "github.com/acme/ferret-sqlite" {
		t.Fatalf("unexpected create request: options=%#v calls=%d", service.createOptions, service.createCalls)
	}
}

func TestModCommandInitCancellationDoesNotCreate(t *testing.T) {
	t.Run("declined", func(t *testing.T) {
		service := &fakeModuleService{}
		stdout, stderr, err := executeInteractiveModCommand(t, service, "\n\n\nn\n", "init", "acme/sqlite")
		if err != nil {
			t.Fatal(err)
		}
		if service.createCalls != 0 || stdout != "" || !strings.Contains(stderr, "Module initialization canceled.") {
			t.Fatalf("unexpected cancellation: calls=%d stdout=%q stderr=%q", service.createCalls, stdout, stderr)
		}
	})

	t.Run("end of input", func(t *testing.T) {
		service := &fakeModuleService{}
		stdout, stderr, err := executeInteractiveModCommand(t, service, "", "init")
		if err != nil {
			t.Fatal(err)
		}
		if service.createCalls != 0 || stdout != "" || !strings.Contains(stderr, "Module initialization canceled.") {
			t.Fatalf("unexpected cancellation: calls=%d stdout=%q stderr=%q", service.createCalls, stdout, stderr)
		}
	})
}

func TestModCommandInitNonInteractiveUsesSafeDefaults(t *testing.T) {
	service := &fakeModuleService{create: &scaffold.Result{Directory: "/tmp/sqlite", Namespace: "sqlite"}}
	_, err := executeModCommand(t, service, "init", "acme/sqlite", "--go-module", "example.com/acme/sqlite")
	if err != nil {
		t.Fatal(err)
	}

	want := scaffold.Options{
		Name:      "acme/sqlite",
		GoModule:  "example.com/acme/sqlite",
		Directory: "sqlite",
		Namespace: "sqlite",
	}
	if service.createOptions != want {
		t.Fatalf("unexpected non-interactive defaults: %#v", service.createOptions)
	}
}

func TestModCommandInitNonInteractiveRequiresName(t *testing.T) {
	service := &fakeModuleService{}
	_, err := executeModCommand(t, service, "init", "--go-module", "example.com/acme/sqlite")
	if err == nil || !strings.Contains(err.Error(), "<name>") || !strings.Contains(err.Error(), "not a terminal") {
		t.Fatalf("unexpected missing name error: %v", err)
	}
	if service.createCalls != 0 {
		t.Fatalf("service called for incomplete input: %d", service.createCalls)
	}
}

func TestModCommandInitRejectsInvalidExplicitInputBeforePrompting(t *testing.T) {
	service := &fakeModuleService{}
	_, stderr, err := executeInteractiveModCommand(
		t,
		service,
		"",
		"init",
		"acme/sqlite",
		"--go-module",
		"not-a-module",
	)
	if err == nil || !strings.Contains(err.Error(), "invalid Go module path") {
		t.Fatalf("unexpected explicit input error: %v", err)
	}
	if strings.Contains(stderr, "Go module path (--go-module)") || service.createCalls != 0 {
		t.Fatalf("invalid explicit input prompted or created: calls=%d stderr=%q", service.createCalls, stderr)
	}
}

func TestModCommandInitPropagatesContextCancellation(t *testing.T) {
	service := &fakeModuleService{}
	command := newCommand(
		newTestStore(t, t.TempDir()),
		service,
		func() bool { return true },
		newScriptedPrompt,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command.SetContext(ctx)
	command.SetIn(strings.NewReader(""))
	command.SetOut(new(bytes.Buffer))
	command.SetErr(new(bytes.Buffer))
	command.SetArgs([]string{"init"})

	if err := command.Execute(); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if service.createCalls != 0 {
		t.Fatalf("service called after cancellation: %d", service.createCalls)
	}
}

func TestModCommandInitFullySpecifiedBypassesInteractivePrompt(t *testing.T) {
	service := &fakeModuleService{create: &scaffold.Result{Directory: "/tmp/custom", Namespace: "DB::SQLITE"}}
	_, stderr, err := executeInteractiveModCommand(
		t,
		service,
		"",
		"init",
		"acme/sqlite",
		"--go-module",
		"example.com/acme/sqlite",
		"--dir",
		"custom",
		"--namespace",
		"DB::SQLITE",
	)
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" || service.createCalls != 1 {
		t.Fatalf("fully specified init prompted unexpectedly: calls=%d stderr=%q", service.createCalls, stderr)
	}
}

func TestModCommandPublishSubmitsAndRendersProgress(t *testing.T) {
	prepared := &barnpublish.Result{
		Kind: barnpublish.NewModule,
		Module: &registryspec.ModuleManifest{
			Owner: "acme", Name: "widget",
			Source: registryspec.Source{Repository: "https://example.com/acme/widget", Path: "modules/widget"},
		},
		Version: &registryspec.VersionRecord{Version: "1.2.3", Tag: "release-1.2.3", Commit: "abcdef0123456789"},
		Files: []barnpublish.File{
			{Path: "registry/modules/acme/widget/manifest.json", Content: []byte("{}\n")},
			{Path: "registry/modules/acme/widget/versions/v1.2.3.json", Content: []byte("{}\n")},
		},
	}
	service := &fakeModuleService{publication: &modulepublish.Result{
		Status: modulepublish.StatusSubmitted, Module: "acme/widget", Version: "1.2.3",
		Tag: "release-1.2.3", Prepared: prepared, PullRequestURL: "https://github.com/MontFerret/barn/pull/123",
	}}
	output, err := executeModCommand(t, service, "publish", "--tag", "release-1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if service.publishOptions.Tag != "release-1.2.3" || service.publishOptions.Mode != modulepublish.ModeSubmit ||
		!strings.Contains(output, "✓ Resolved release-1.2.3 → abcdef0") ||
		!strings.Contains(output, "✓ Submitted to Ferret Registry") ||
		!strings.Contains(output, service.publication.PullRequestURL) || strings.Contains(output, "registry/modules/") {
		t.Fatalf("unexpected publication output:\n%s", output)
	}
}

func TestModCommandPublishReportsExistingSubmission(t *testing.T) {
	service := &fakeModuleService{publication: &modulepublish.Result{
		Status: modulepublish.StatusExistingSubmission, Module: "acme/widget", Version: "1.2.3",
		Tag: "v1.2.3", Prepared: &barnpublish.Result{
			Kind:    barnpublish.NewVersion,
			Module:  &registryspec.ModuleManifest{Owner: "acme", Name: "widget"},
			Version: &registryspec.VersionRecord{Version: "1.2.3", Tag: "v1.2.3", Commit: "abcdef0123456789"},
		},
		PullRequestURL: "https://github.com/MontFerret/barn/pull/123",
	}}

	output, err := executeModCommand(t, service, "publish")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "✓ Found existing Registry submission") || !strings.Contains(output, service.publication.PullRequestURL) {
		t.Fatalf("unexpected existing-submission output:\n%s", output)
	}
}

func TestModCommandPublishPrintsDeterministicJSON(t *testing.T) {
	prepared := &barnpublish.Result{
		Kind: barnpublish.NewModule,
		Module: &registryspec.ModuleManifest{
			Owner: "acme", Name: "widget",
			Source: registryspec.Source{Repository: "https://example.com/acme/widget"},
		},
		Version: &registryspec.VersionRecord{Version: "1.2.4", Tag: "v1.2.4", Commit: "def0123456789abc"},
		Files: []barnpublish.File{
			{Path: "registry/modules/acme/widget/versions/v1.2.4.json", Content: []byte("{\"version\":\"1.2.4\"}\n")},
			{Path: "registry/modules/acme/widget/manifest.json", Content: []byte("{\"name\":\"widget\"}\n")},
		},
	}
	service := &fakeModuleService{publication: &modulepublish.Result{
		Status: modulepublish.StatusReady, Module: "acme/widget", Version: "1.2.4", Tag: "v1.2.4", Prepared: prepared,
	}}

	output, err := executeModCommand(t, service, "publish", "--print")
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	var document publicationDocument
	if decodeErr := json.Unmarshal([]byte(output), &fields); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if decodeErr := json.Unmarshal([]byte(output), &document); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if service.publishOptions.Mode != modulepublish.ModePrint || strings.Contains(output, "✓") || !strings.HasSuffix(output, "\n") ||
		len(fields) != 8 || document.SchemaVersion != 1 || document.Status != modulepublish.StatusReady ||
		document.Module != "acme/widget" || document.Version != "1.2.4" || document.Tag != "v1.2.4" ||
		document.Kind != string(barnpublish.NewModule) || document.Commit != "def0123456789abc" || len(document.Records) != 2 ||
		document.Records[0].Path != "registry/modules/acme/widget/manifest.json" || document.Records[0].Content != "{\"name\":\"widget\"}\n" ||
		document.Records[1].Path != "registry/modules/acme/widget/versions/v1.2.4.json" || document.Records[1].Content != "{\"version\":\"1.2.4\"}\n" {
		t.Fatalf("unexpected publication JSON:\n%s", output)
	}
}

func TestModCommandPublishDryRunAndAlreadyPublished(t *testing.T) {
	prepared := &barnpublish.Result{
		Kind:    barnpublish.NewVersion,
		Module:  &registryspec.ModuleManifest{Owner: "acme", Name: "widget"},
		Version: &registryspec.VersionRecord{Version: "1.2.4", Tag: "v1.2.4", Commit: "def0123456789abc"},
	}
	service := &fakeModuleService{publication: &modulepublish.Result{
		Status: modulepublish.StatusReady, Module: "acme/widget", Version: "1.2.4", Tag: "v1.2.4", Prepared: prepared,
	}}
	output, err := executeModCommand(t, service, "publish", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if service.publishOptions.Mode != modulepublish.ModeDryRun || !strings.Contains(output, "Ready to publish.") {
		t.Fatalf("unexpected dry-run output:\n%s", output)
	}

	service.publication = &modulepublish.Result{
		Status: modulepublish.StatusAlreadyPublished, Module: "acme/widget", Version: "1.2.4", Tag: "v1.2.4",
	}
	output, err = executeModCommand(t, service, "publish")
	if err != nil {
		t.Fatal(err)
	}
	if output != "✓ Validated ferret.yaml\nacme/widget@1.2.4 is already published.\n" {
		t.Fatalf("unexpected already-published output: %q", output)
	}
}

func TestModCommandPublishRejectsConflictingModesBeforeService(t *testing.T) {
	service := &fakeModuleService{}
	_, err := executeModCommand(t, service, "publish", "--dry-run", "--print")
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("unexpected flag error: %v", err)
	}
	if service.publishCalls != 0 {
		t.Fatalf("service called for conflicting modes: %d", service.publishCalls)
	}
}

func TestModCommandPublishRendersPreparedProgressBeforeSubmissionError(t *testing.T) {
	want := errors.New("create pull request")
	prepared := &barnpublish.Result{
		Kind:    barnpublish.NewVersion,
		Module:  &registryspec.ModuleManifest{Owner: "acme", Name: "widget"},
		Version: &registryspec.VersionRecord{Version: "1.2.4", Tag: "v1.2.4", Commit: "def0123456789abc"},
	}
	service := &fakeModuleService{
		publication: &modulepublish.Result{
			Status: modulepublish.StatusReady, Module: "acme/widget", Version: "1.2.4", Tag: "v1.2.4", Prepared: prepared,
		},
		err: want,
	}

	output, err := executeModCommand(t, service, "publish")
	if !errors.Is(err, want) {
		t.Fatalf("expected submission error, got %v", err)
	}
	if !strings.Contains(output, "✓ Prepared acme/widget@1.2.4") || strings.Contains(output, "Submitted") || strings.Contains(output, "Ready to publish") {
		t.Fatalf("unexpected partial publication output:\n%s", output)
	}
}

func TestModCommandPublishPrintsAlreadyPublishedEnvelope(t *testing.T) {
	service := &fakeModuleService{publication: &modulepublish.Result{
		Status: modulepublish.StatusAlreadyPublished, Module: "acme/widget", Version: "1.2.4", Tag: "v1.2.4",
	}}

	output, err := executeModCommand(t, service, "publish", "--print")
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	var document publicationDocument
	if decodeErr := json.Unmarshal([]byte(output), &fields); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if decodeErr := json.Unmarshal([]byte(output), &document); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if len(fields) != 6 || document.Status != modulepublish.StatusAlreadyPublished || len(document.Records) != 0 ||
		document.Kind != "" || document.Commit != "" || strings.Contains(output, "✓") || !strings.HasSuffix(output, "\n") {
		t.Fatalf("unexpected already-published JSON:\n%s", output)
	}
}

func TestModCommandExposesOnlyRenamedLifecycleSubcommands(t *testing.T) {
	command := New(newTestStore(t, t.TempDir()), &fakeModuleService{})
	if command.Name() != "mod" || len(command.Aliases) != 0 {
		t.Fatalf("unexpected command name or aliases: %q %#v", command.Name(), command.Aliases)
	}

	want := map[string]bool{"search": true, "info": true, "install": true, "init": true, "publish": true}
	for _, child := range command.Commands() {
		delete(want, child.Name())
		if len(child.Aliases) != 0 {
			t.Fatalf("unexpected aliases for %q: %#v", child.Name(), child.Aliases)
		}
		if child.Name() == "add" || child.Name() == "create" || child.Name() == "update" {
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

	command := newCommand(
		newTestStore(t, t.TempDir()),
		service,
		func() bool { return false },
		func(io.Reader, io.Writer) (prompt, error) {
			t.Fatal("non-interactive command attempted to prompt")
			return nil, nil
		},
	)
	output := new(bytes.Buffer)
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs(args)

	err := command.Execute()
	return output.String(), err
}

func executeInteractiveModCommand(t *testing.T, service Service, input string, args ...string) (string, string, error) {
	t.Helper()

	command := newCommand(
		newTestStore(t, t.TempDir()),
		service,
		func() bool { return true },
		newScriptedPrompt,
	)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command.SetIn(strings.NewReader(input))
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	err := command.Execute()
	return stdout.String(), stderr.String(), err
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
