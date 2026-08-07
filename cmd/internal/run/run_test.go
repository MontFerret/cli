package run

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/build"
	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/logger"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
	"github.com/MontFerret/ferret/v2/pkg/compiler"
	ferrethttp "github.com/MontFerret/ferret/v2/pkg/net/http"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestExecuteRun_ArtifactRemoteRuntimeRejected(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	artifactPath := filepath.Join(dir, "query.fqlc")

	testutil.WriteQuery(t, input, "RETURN 42")

	if err := build.WriteArtifact(compiler.New(), source.New(input, "RETURN 42"), artifactPath); err != nil {
		t.Fatalf("build artifact: %v", err)
	}

	_, err := testutil.CaptureStdout(t, func() error {
		return execute(
			testutil.NewCommand(),
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

	testutil.WriteQuery(t, input, "RETURN 42")

	if err := build.WriteArtifact(compiler.New(), source.New(input, "RETURN 42"), artifactPath); err != nil {
		t.Fatalf("build artifact: %v", err)
	}

	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}

	testutil.WithStdinBytes(t, data, func() {
		err := execute(
			testutil.NewCommand(),
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

func TestExecuteRun_HTTPPolicyRemoteRuntimeRejectedBeforeBrowserStartup(t *testing.T) {
	opts := cliruntime.NewDefaultOptions()
	opts.Type = "https://worker.example"
	opts.WithBrowser = true
	opts.BrowserAddress = "://invalid"
	opts.HTTPPolicy = []ferrethttp.PolicyOption{ferrethttp.WithAllowLocalhost(true)}

	err := execute(
		testutil.NewCommand(),
		opts,
		browser.Options{},
		nil,
		"RETURN 1",
		nil,
	)
	if !errors.Is(err, cliruntime.ErrHTTPPolicyRequiresBuiltinRuntime) {
		t.Fatalf("expected builtin runtime policy error, got %v", err)
	}
}

func TestExecuteRun_FSPolicyRemoteRuntimeRejectedBeforeBrowserStartup(t *testing.T) {
	opts := cliruntime.NewDefaultOptions()
	opts.Type = "https://worker.example"
	opts.WithBrowser = true
	opts.BrowserAddress = "://invalid"
	opts.FSPolicy = &cliruntime.FileSystemPolicy{ReadOnly: true}

	err := execute(
		testutil.NewCommand(),
		opts,
		browser.Options{},
		nil,
		"RETURN 1",
		nil,
	)
	if !errors.Is(err, cliruntime.ErrFSPolicyRequiresBuiltinRuntime) {
		t.Fatalf("expected builtin runtime policy error, got %v", err)
	}
}

func TestRunCommand_RejectsMultiplePositionalArgs(t *testing.T) {
	cmd := New(new(config.Store))

	if err := cmd.Args(cmd, []string{"one.fql", "two.fql"}); err == nil {
		t.Fatal("expected argument validation error")
	}
}

func TestRunCommand_RejectsEvalWithFileArgs(t *testing.T) {
	cmd := New(new(config.Store))
	cmd.SetContext(config.With(context.Background(), new(config.Store)))
	cmd.Flags().Set("eval", "RETURN 1")

	err := cmd.RunE(cmd, []string{"query.fql"})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecuteRun_NoInputShowsHelp(t *testing.T) {
	cmd := testutil.NewCommand()
	testutil.WithDevNullStdin(t, func() {
		if err := execute(cmd, cliruntime.NewDefaultOptions(), browser.Options{}, nil, "", nil); err != nil {
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

	stderr, runErr := testutil.CaptureStderr(t, func() error {
		stdout, err = testutil.CaptureStdout(t, func() error {
			return execute(
				testutil.NewCommand(),
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

	stdout, err := testutil.CaptureStdout(t, func() error {
		return execute(
			testutil.NewCommand(),
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

	err := execute(
		testutil.NewCommand(),
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

	err := execute(
		testutil.NewCommand(),
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

	err := execute(
		testutil.NewCommand(),
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
