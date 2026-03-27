package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/pkg/browser"
	"github.com/MontFerret/cli/pkg/config"
	cliruntime "github.com/MontFerret/cli/pkg/runtime"
)

func TestExecuteRun_SourceFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")

	writeQuery(t, input, "RETURN 42")

	stdout, err := captureStdout(t, func() error {
		return executeRun(newTestCommand(), cliruntime.NewDefaultOptions(), browser.Options{}, nil, "", []string{input})
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(stdout) != "42" {
		t.Fatalf("expected 42, got %q", stdout)
	}
}

func TestExecuteRun_CompiledArtifact(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")

	writeQuery(t, input, "RETURN 42")

	if _, err := captureStderr(t, func() error {
		return runBuild([]string{input}, "")
	}); err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	stdout, err := captureStdout(t, func() error {
		return executeRun(newTestCommand(), cliruntime.NewDefaultOptions(), browser.Options{}, nil, "", []string{siblingArtifactPath(input)})
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(stdout) != "42" {
		t.Fatalf("expected 42, got %q", stdout)
	}
}

func TestExecuteRun_CompiledArtifactCustomName(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "compiled.bin")

	writeQuery(t, input, "RETURN 42")

	if _, err := captureStderr(t, func() error {
		return runBuild([]string{input}, output)
	}); err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	stdout, err := captureStdout(t, func() error {
		return executeRun(newTestCommand(), cliruntime.NewDefaultOptions(), browser.Options{}, nil, "", []string{output})
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(stdout) != "42" {
		t.Fatalf("expected 42, got %q", stdout)
	}
}

func TestExecuteRun_CompiledArtifactWithParams(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")

	writeQuery(t, input, "RETURN @value")

	if _, err := captureStderr(t, func() error {
		return runBuild([]string{input}, "")
	}); err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	stdout, err := captureStdout(t, func() error {
		return executeRun(
			newTestCommand(),
			cliruntime.NewDefaultOptions(),
			browser.Options{},
			map[string]interface{}{"value": float64(99)},
			"",
			[]string{siblingArtifactPath(input)},
		)
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(stdout) != "99" {
		t.Fatalf("expected 99, got %q", stdout)
	}
}

func TestExecuteRun_PlainTextFQLCIsSource(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 7")

	stdout, err := captureStdout(t, func() error {
		return executeRun(newTestCommand(), cliruntime.NewDefaultOptions(), browser.Options{}, nil, "", []string{input})
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(stdout) != "7" {
		t.Fatalf("expected 7, got %q", stdout)
	}
}

func TestExecuteRun_CorruptArtifactReturnsLoadError(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "broken.bin")

	if err := os.WriteFile(input, []byte("FBC2"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := captureStderr(t, func() error {
		return executeRun(newTestCommand(), cliruntime.NewDefaultOptions(), browser.Options{}, nil, "", []string{input})
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "bytecode artifact") {
		t.Fatalf("expected artifact load error, got %v", err)
	}
}

func TestExecuteRun_ArtifactRemoteRuntimeRejected(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")

	writeQuery(t, input, "RETURN 42")

	if _, err := captureStderr(t, func() error {
		return runBuild([]string{input}, "")
	}); err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	_, err := captureStdout(t, func() error {
		return executeRun(
			newTestCommand(),
			cliruntime.Options{Type: "https://worker.example"},
			browser.Options{},
			nil,
			"",
			[]string{siblingArtifactPath(input)},
		)
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "compiled artifacts require the builtin runtime") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCommand_RejectsMultiplePositionalArgs(t *testing.T) {
	cmd := RunCommand(new(config.Store))

	if err := cmd.Args(cmd, []string{"one.fql", "two.fql"}); err == nil {
		t.Fatal("expected argument validation error")
	}
}

func newTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	return cmd
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = writer

	runErr := fn()

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	os.Stdout = original

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if closeErr := reader.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	return string(data), runErr
}
