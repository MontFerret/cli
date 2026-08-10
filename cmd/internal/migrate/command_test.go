package migrate

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"

	"github.com/MontFerret/cli/v2/internal/migration"
	"github.com/MontFerret/cli/v2/pkg/config"
)

func TestMigrateCommandAppliesAndRendersMigration(t *testing.T) {
	service := &fakeMigrationService{result: &migration.Result{
		ScannedFiles:        4,
		UpdatedImports:      2,
		FormattedFiles:      1,
		DependenciesChanged: true,
		Applied:             true,
		Changes: []migration.Change{
			{Path: "go.mod"},
			{Path: "main.go"},
		},
	}}

	stdout, stderr, err := executeMigrateCommand(t, service)
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{
		"Ferret v1 → v2 compatibility migration",
		"✓ Scanned 4 Go files",
		"  go.mod",
		"  main.go",
		"✓ Updated 2 Ferret imports",
		"✓ Updated Go module dependencies",
		"✓ Formatted 1 Go file",
		"Migration completed.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected stdout to contain %q:\n%s", expected, stdout)
		}
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if service.calls != 1 || service.options.Mode != migration.ModeApply || service.options.Directory != "." {
		t.Fatalf("unexpected service call: calls=%d options=%#v", service.calls, service.options)
	}
}

func TestMigrateCommandDryRunListsFilesWithoutApplying(t *testing.T) {
	service := &fakeMigrationService{result: &migration.Result{
		ScannedFiles: 3,
		Changes: []migration.Change{
			{Path: "go.mod"},
			{Path: "internal/app.go"},
		},
	}}

	stdout, stderr, err := executeMigrateCommand(t, service, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{"Would update:", "  go.mod", "  internal/app.go", "No files changed."} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected stdout to contain %q:\n%s", expected, stdout)
		}
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if service.options.Mode != migration.ModeDryRun {
		t.Fatalf("unexpected mode: %v", service.options.Mode)
	}
}

func TestMigrateCommandPrintsDeterministicUnifiedDiffOnlyOnStdout(t *testing.T) {
	service := &fakeMigrationService{result: &migration.Result{
		ScannedFiles: 1,
		Changes: []migration.Change{
			{
				Path:         "main.go",
				BeforeExists: true,
				Before:       []byte("package main\n\nvar version = 1\n"),
				After:        []byte("package main\n\nvar version = 2\n"),
			},
		},
	}}

	stdout, stderr, err := executeMigrateCommand(t, service, "--print")
	if err != nil {
		t.Fatal(err)
	}

	want := "--- a/main.go\n" +
		"+++ b/main.go\n" +
		"@@ -1,4 +1,4 @@\n" +
		" package main\n" +
		" \n" +
		"-var version = 1\n" +
		"+var version = 2\n" +
		" \n"
	if stdout != want {
		t.Fatalf("unexpected diff:\n%s", stdout)
	}
	if !strings.Contains(stderr, "Printed a unified diff for 1 file.") ||
		!strings.Contains(stderr, "No files changed.") ||
		strings.Contains(stdout, "Ferret v1") {
		t.Fatalf("unexpected status streams:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if service.options.Mode != migration.ModePrint {
		t.Fatalf("unexpected mode: %v", service.options.Mode)
	}
}

func TestMigrateCommandRendersManualAndVendorWarnings(t *testing.T) {
	service := &fakeMigrationService{result: &migration.Result{
		ScannedFiles:   2,
		VendorDetected: true,
		Changes:        []migration.Change{{Path: "main.go"}},
		ManualActions: []migration.ManualAction{
			{
				Path:       "generated.go",
				Line:       4,
				ImportPath: "github.com/MontFerret/ferret/pkg/runtime",
				Reason:     "generated file was not modified",
			},
		},
	}}

	stdout, stderr, err := executeMigrateCommand(t, service)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(stdout, "Manual follow-up is still required") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	for _, expected := range []string{
		"Manual follow-up:",
		"generated.go:4: github.com/MontFerret/ferret/pkg/runtime (generated file was not modified)",
		"Vendor directory was not modified; run go mod vendor",
	} {
		if !strings.Contains(stderr, expected) {
			t.Fatalf("expected stderr to contain %q:\n%s", expected, stderr)
		}
	}
}

func TestMigrateCommandRejectsConflictingModesBeforeService(t *testing.T) {
	service := new(fakeMigrationService)
	_, _, err := executeMigrateCommand(t, service, "--dry-run", "--print")
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("unexpected error: %v", err)
	}
	if service.calls != 0 {
		t.Fatalf("service was called %d times", service.calls)
	}
}

func TestMigrateCommandRejectsArgumentsAndPropagatesServiceErrors(t *testing.T) {
	service := new(fakeMigrationService)
	if _, _, err := executeMigrateCommand(t, service, "v1"); err == nil {
		t.Fatal("expected positional argument error")
	}
	if service.calls != 0 {
		t.Fatalf("service was called %d times", service.calls)
	}

	want := errors.New("migration failed")
	service.err = want
	_, _, err := executeMigrateCommand(t, service)
	if !errors.Is(err, want) {
		t.Fatalf("expected service error, got %v", err)
	}
}

type fakeMigrationService struct {
	result  *migration.Result
	err     error
	options migration.Options
	calls   int
}

func (service *fakeMigrationService) Migrate(_ context.Context, options migration.Options) (*migration.Result, error) {
	service.calls++
	service.options = options

	if service.result == nil && service.err == nil {
		service.result = new(migration.Result)
	}

	return service.result, service.err
}

func executeMigrateCommand(t *testing.T, service Service, args ...string) (string, string, error) {
	t.Helper()

	command := New(newMigrateTestStore(t), service)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	err := command.Execute()

	return stdout.String(), stderr.String(), err
}

func newMigrateTestStore(t *testing.T) *config.Store {
	t.Helper()

	t.Setenv("HOME", t.TempDir())
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	store, err := config.NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	return store
}
