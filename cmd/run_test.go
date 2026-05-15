package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/rs/zerolog"

	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/build"
	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/logger"
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

func TestExecuteRun_ExecutionLogsGoToStderrNotStdout(t *testing.T) {
	opts := cliruntime.NewDefaultOptions()
	opts.Logger.Level = zerolog.DebugLevel
	opts.Logger.LogOutput = logger.OutputStderr

	var stdout string
	var err error

	stderr, runErr := captureStderr(t, func() error {
		stdout, err = captureStdout(t, func() error {
			return executeRun(
				newTestCommand(),
				opts,
				browser.Options{},
				nil,
				"LET printed = PRINT(\"hello\") RETURN 42",
				nil,
			)
		})

		return err
	})

	if runErr != nil {
		t.Fatalf("unexpected run error: %v", runErr)
	}

	if strings.TrimSpace(stdout) != "42" {
		t.Fatalf("expected stdout result 42, got %q", stdout)
	}

	if strings.Contains(stdout, "hello") {
		t.Fatalf("expected stdout not to contain logs, got %q", stdout)
	}

	if !strings.Contains(stderr, "hello") {
		t.Fatalf("expected stderr log, got %q", stderr)
	}
}

func TestExecuteRun_ExecutionLogsGoToFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "ferret.log")
	opts := cliruntime.NewDefaultOptions()
	opts.Logger.Level = zerolog.DebugLevel
	opts.Logger.LogOutput = logger.OutputFile
	opts.Logger.LogFilename = logPath

	stdout, err := captureStdout(t, func() error {
		return executeRun(
			newTestCommand(),
			opts,
			browser.Options{},
			nil,
			"LET printed = PRINT(\"hello\") RETURN 42",
			nil,
		)
	})

	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	if strings.TrimSpace(stdout) != "42" {
		t.Fatalf("expected stdout result 42, got %q", stdout)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(logData), "hello") {
		t.Fatalf("expected log file to contain execution log, got %q", string(logData))
	}
}

func TestExecuteRun_RejectsInvalidLogOutput(t *testing.T) {
	opts := cliruntime.NewDefaultOptions()
	opts.Logger.LogOutput = "stdout"

	err := executeRun(
		newTestCommand(),
		opts,
		browser.Options{},
		nil,
		"RETURN 42",
		nil,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "invalid log output") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteRun_RejectsEmptyLogOutput(t *testing.T) {
	opts := cliruntime.NewDefaultOptions()
	opts.Logger.LogOutput = ""
	opts.Logger.LogOutputSet = true

	err := executeRun(
		newTestCommand(),
		opts,
		browser.Options{},
		nil,
		"RETURN 42",
		nil,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "invalid log output") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteRun_RejectsEmptyLogFile(t *testing.T) {
	opts := cliruntime.NewDefaultOptions()
	opts.Logger.LogOutput = logger.OutputFile
	opts.Logger.LogFilename = ""

	err := executeRun(
		newTestCommand(),
		opts,
		browser.Options{},
		nil,
		"RETURN 42",
		nil,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "log file cannot be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	return cmd
}
