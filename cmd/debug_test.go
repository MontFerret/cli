package cmd

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/MontFerret/ferret/v2/pkg/compiler"
	"github.com/MontFerret/ferret/v2/pkg/source"

	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/build"
	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestDebugCommandRequiresExactlyOneScript(t *testing.T) {
	command := DebugCommand(new(config.Store))

	if err := command.Args(command, nil); err == nil {
		t.Fatal("expected missing argument error")
	}
	if err := command.Args(command, []string{"one.fql", "two.fql"}); err == nil {
		t.Fatal("expected too many arguments error")
	}
	if err := command.Args(command, []string{"one.fql"}); err != nil {
		t.Fatalf("unexpected argument error: %v", err)
	}
}

func TestExecuteDebugRejectsRemoteRuntimeBeforeStarting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "query.fql")
	writeQuery(t, path, "RETURN 1")

	err := executeDebug(
		newTestCommand(),
		cliruntime.Options{Type: "https://worker.example"},
		browser.Options{},
		nil,
		[]string{path},
	)
	if !errors.Is(err, cliruntime.ErrDebugRequiresBuiltinRuntime) {
		t.Fatalf("expected builtin runtime error, got %v", err)
	}
	if got := err.Error(); got != "debug currently supports only the builtin runtime" {
		t.Fatalf("unexpected builtin runtime error: %q", got)
	}
}

func TestExecuteDebugRejectsArtifact(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "query.fql")
	artifactPath := filepath.Join(dir, "query.fqlc")
	writeQuery(t, sourcePath, "RETURN 1")
	if err := build.WriteArtifact(compiler.New(), source.New(sourcePath, "RETURN 1"), artifactPath); err != nil {
		t.Fatal(err)
	}

	err := executeDebug(
		newTestCommand(),
		cliruntime.NewDefaultOptions(),
		browser.Options{},
		nil,
		[]string{artifactPath},
	)
	if err == nil || err.Error() != "debugging compiled artifacts is not supported yet; run debug with the original .fql source file" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDebugCommandUsesSharedParamFlags(t *testing.T) {
	command := DebugCommand(new(config.Store))
	command.SetContext(config.With(context.Background(), new(config.Store)))

	if err := command.Flags().Set(paramFlag, "limit=2"); err != nil {
		t.Fatal(err)
	}
	values, err := command.Flags().GetStringArray(paramFlag)
	if err != nil {
		t.Fatal(err)
	}
	params, err := parseParams(values)
	if err != nil {
		t.Fatal(err)
	}
	if params["limit"] != float64(2) {
		t.Fatalf("unexpected params: %#v", params)
	}
}
