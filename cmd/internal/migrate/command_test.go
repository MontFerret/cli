package migrate

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"

	"github.com/MontFerret/cli/v2/internal/migration"
	"github.com/MontFerret/cli/v2/pkg/config"
)

func TestMigrateCommandAppliesAndRendersMigration(t *testing.T) {
	service := &fakeMigrationService{result: &migration.Result{
		ScannedFiles:        4,
		ScannedFQLFiles:     2,
		UpdatedImports:      2,
		FormattedFiles:      1,
		MigratedFQLFiles:    1,
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
		"✓ Scanned 2 FQL files",
		"  go.mod",
		"  main.go",
		"✓ Updated 2 Ferret imports",
		"✓ Updated Go module dependencies",
		"✓ Formatted 1 Go file",
		"✓ Migrated and formatted 1 FQL file",
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
		ScannedFiles:     3,
		ScannedFQLFiles:  1,
		MigratedFQLFiles: 1,
		Changes: []migration.Change{
			{Path: "go.mod"},
			{Path: "internal/app.go"},
		},
	}}

	stdout, stderr, err := executeMigrateCommand(t, service, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{
		"✓ Scanned 1 FQL file",
		"✓ Found 1 supported FQL source migration",
		"Would update:",
		"  go.mod",
		"  internal/app.go",
		"No files changed.",
	} {
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
		ScannedFQLFiles:  1,
		MigratedFQLFiles: 1,
		Changes: []migration.Change{
			{
				Path:         "query.fql",
				BeforeExists: true,
				Before:       []byte("FOR x IN 1..3\n    RETURN x\n"),
				After:        []byte("return for x in 1..3 {\n    return x\n}"),
			},
		},
	}}

	stdout, stderr, err := executeMigrateCommand(t, service, "--print")
	if err != nil {
		t.Fatal(err)
	}

	want := "--- a/query.fql\n" +
		"+++ b/query.fql\n" +
		"@@ -1,3 +1,3 @@\n" +
		"-FOR x IN 1..3\n" +
		"-    RETURN x\n" +
		"-\n" +
		"+return for x in 1..3 {\n" +
		"+    return x\n" +
		"+}\n"
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
				Path:   "generated.go",
				Line:   4,
				Detail: "github.com/MontFerret/ferret/pkg/runtime",
				Reason: "generated file was not modified",
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

func TestMigrateCheckReportsCompatibilityIssues(t *testing.T) {
	service := &fakeMigrationService{checkResult: &migration.CompatibilityResult{
		ScannedFiles: 12,
		Diagnostics: []migration.CompatibilityDiagnostic{
			{
				Path:    "1_hackernews.fql",
				Line:    2,
				Column:  1,
				Message: "Final collecting FOR no longer becomes the script result in Ferret v2.",
				Help:    "Add `return` before this loop.",
				Kind:    migration.CompatibilityDiagnosticIssue,
			},
		},
	}}

	stdout, stderr, err := executeMigrateCommand(t, service, "check", "--from", "v1", ".")
	if err == nil || err.Error() != "Found 1 v1 compatibility issue in 1 of 12 FQL files." {
		t.Fatalf("unexpected check error: %v", err)
	}

	if stdout != "" {
		t.Fatalf("unexpected stdout: %s", stdout)
	}

	wantStderr := "1_hackernews.fql:2:1: Final collecting FOR no longer becomes the script result in Ferret v2.\n" +
		"  help: Add `return` before this loop.\n\n"
	if stderr != wantStderr {
		t.Fatalf("unexpected stderr:\nwant:\n%s\ngot:\n%s", wantStderr, stderr)
	}

	if service.checkCalls != 1 || service.checkOptions.Path != "." || service.checkOptions.From != "v1" {
		t.Fatalf("unexpected compatibility check call: calls=%d options=%#v", service.checkCalls, service.checkOptions)
	}
}

func TestMigrateCheckDefaultsToV1AndCurrentDirectory(t *testing.T) {
	service := &fakeMigrationService{checkResult: &migration.CompatibilityResult{ScannedFiles: 1}}

	stdout, stderr, err := executeMigrateCommand(t, service, "check")
	if err != nil {
		t.Fatal(err)
	}

	if stdout != "✓ No v1 compatibility issues found in 1 FQL file.\n" || stderr != "" {
		t.Fatalf("unexpected output:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	if service.checkOptions != (migration.CompatibilityOptions{Path: ".", From: "v1"}) {
		t.Fatalf("unexpected default options: %#v", service.checkOptions)
	}
}

func TestMigrateCheckReportsEmptyDirectory(t *testing.T) {
	service := &fakeMigrationService{checkResult: new(migration.CompatibilityResult)}

	stdout, stderr, err := executeMigrateCommand(t, service, "check", "scripts")
	if err != nil {
		t.Fatal(err)
	}

	if stdout != "✓ No FQL files found at scripts.\n" || stderr != "" {
		t.Fatalf("unexpected output:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestMigrateCheckReportsUncheckableSourceAndFails(t *testing.T) {
	service := &fakeMigrationService{checkResult: &migration.CompatibilityResult{
		ScannedFiles: 2,
		Diagnostics: []migration.CompatibilityDiagnostic{
			{
				Path:    "broken.fql",
				Line:    3,
				Column:  5,
				Message: "Could not check v1 compatibility: unexpected end of input",
				Help:    "Fix this FQL source and rerun the check.",
				Kind:    migration.CompatibilityDiagnosticFailure,
			},
		},
	}}

	stdout, stderr, err := executeMigrateCommand(t, service, "check", ".")
	if err == nil || err.Error() != "Could not check 1 of 2 FQL files for v1 compatibility." {
		t.Fatalf("unexpected check error: %v", err)
	}

	if stdout != "" || !strings.Contains(stderr, "broken.fql:3:5: Could not check v1 compatibility") {
		t.Fatalf("unexpected output:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestMigrateCheckRejectsUnsupportedVersionAndExtraArguments(t *testing.T) {
	service := new(fakeMigrationService)

	if _, _, err := executeMigrateCommand(t, service, "check", "--from", "v2", "."); err == nil || !strings.Contains(err.Error(), "expected v1") {
		t.Fatalf("unexpected source-version error: %v", err)
	}

	if service.checkCalls != 0 {
		t.Fatalf("compatibility checker was called %d times", service.checkCalls)
	}

	if _, _, err := executeMigrateCommand(t, service, "check", "one", "two"); err == nil {
		t.Fatal("expected positional argument error")
	}
}

func TestMigrateCheckHelpDocumentsReadOnlyScanBoundaries(t *testing.T) {
	service := new(fakeMigrationService)

	stdout, stderr, err := executeMigrateCommand(t, service, "check", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{
		"migrate check [path]",
		"does not require a Go module and never modifies source files",
		"include testdata, hidden and underscore-prefixed directories, and nested Go modules",
		"skip .git, .hg, .svn, vendor, and node_modules",
		"currently only v1",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected help to contain %q:\n%s", expected, stdout)
		}
	}

	if stderr != "" || service.checkCalls != 0 {
		t.Fatalf("unexpected help side effects: stderr=%q calls=%d", stderr, service.checkCalls)
	}
}

func TestMigrateCheckChecksStandaloneFQLWithoutGoModule(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "query.fql")
	if err := os.WriteFile(path, []byte("let label = \"привет\"\nfor value in 1..3 return value"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := executeMigrateCommand(t, migration.New(nil), "check", path)
	if err == nil || err.Error() != "Found 1 v1 compatibility issue in 1 of 1 FQL file." {
		t.Fatalf("unexpected check error: %v", err)
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	displayPath, err := filepath.Rel(workingDirectory, path)
	if err != nil {
		t.Fatal(err)
	}

	wantDiagnostic := filepath.ToSlash(displayPath) +
		":2:1: Final collecting FOR no longer becomes the script result in Ferret v2.\n" +
		"  help: Add `return` before this loop.\n\n"
	if stdout != "" || stderr != wantDiagnostic {
		t.Fatalf("unexpected output:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestCompatibilityCheckErrorSummarizesMultipleFindingsAndFailures(t *testing.T) {
	result := &migration.CompatibilityResult{
		ScannedFiles: 4,
		Diagnostics: []migration.CompatibilityDiagnostic{
			{Path: "first.fql", Kind: migration.CompatibilityDiagnosticIssue},
			{Path: "first.fql", Kind: migration.CompatibilityDiagnosticIssue},
			{Path: "second.fql", Kind: migration.CompatibilityDiagnosticIssue},
			{Path: "broken.fql", Kind: migration.CompatibilityDiagnosticFailure},
		},
	}

	err := newCompatibilityCheckError(result)
	want := "Found 3 v1 compatibility issues in 2 of 4 FQL files; could not check 1 FQL file."
	if err == nil || err.Error() != want {
		t.Fatalf("unexpected summary: %v", err)
	}
}

type fakeMigrationService struct {
	result       *migration.Result
	err          error
	options      migration.Options
	calls        int
	checkResult  *migration.CompatibilityResult
	checkErr     error
	checkOptions migration.CompatibilityOptions
	checkCalls   int
}

func (service *fakeMigrationService) CheckCompatibility(
	_ context.Context,
	options migration.CompatibilityOptions,
) (*migration.CompatibilityResult, error) {
	service.checkCalls++
	service.checkOptions = options

	if service.checkResult == nil && service.checkErr == nil {
		service.checkResult = new(migration.CompatibilityResult)
	}

	return service.checkResult, service.checkErr
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
	command.SilenceErrors = true
	command.SilenceUsage = true
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
