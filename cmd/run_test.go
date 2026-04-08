package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/build"
	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
	"github.com/MontFerret/ferret/v2/pkg/compiler"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestExecuteRun_ArtifactRemoteRuntimeRejected(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	artifactPath := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 42")

	if err := build.WriteArtifact(compiler.New(), source.New(input, "RETURN 42"), artifactPath); err != nil {
		t.Fatalf("build artifact: %v", err)
	}

	_, err := captureStdout(t, func() error {
		return executeRun(
			newTestCommand(),
			cliruntime.Options{Type: "https://worker.example"},
			browser.Options{},
			nil,
			"",
			[]string{artifactPath},
		)
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, cliruntime.ErrArtifactRequiresBuiltinRuntime) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteRun_ArtifactStdinRemoteRuntimeRejected(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	artifactPath := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 42")

	if err := build.WriteArtifact(compiler.New(), source.New(input, "RETURN 42"), artifactPath); err != nil {
		t.Fatalf("build artifact: %v", err)
	}

	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}

	withStdinBytes(t, data, func() {
		err := executeRun(
			newTestCommand(),
			cliruntime.Options{
				Type:           "https://worker.example",
				WithBrowser:    true,
				BrowserAddress: "://invalid",
			},
			browser.Options{},
			nil,
			"",
			nil,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cliruntime.ErrArtifactRequiresBuiltinRuntime) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRunCommand_RejectsMultiplePositionalArgs(t *testing.T) {
	cmd := RunCommand(new(config.Store))

	if err := cmd.Args(cmd, []string{"one.fql", "two.fql"}); err == nil {
		t.Fatal("expected argument validation error")
	}
}

func TestRunCommand_RejectsEvalWithFileArgs(t *testing.T) {
	cmd := RunCommand(new(config.Store))
	cmd.SetContext(config.With(context.Background(), new(config.Store)))
	cmd.Flags().Set("eval", "RETURN 1")

	err := cmd.RunE(cmd, []string{"query.fql"})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecuteRun_NoInputShowsHelp(t *testing.T) {
	cmd := newTestCommand()
	withDevNullStdin(t, func() {
		if err := executeRun(cmd, cliruntime.NewDefaultOptions(), browser.Options{}, nil, "", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func newTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	return cmd
}
