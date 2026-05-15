package runtime

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/MontFerret/cli/v2/pkg/logger"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestBuiltinRunWritesExecutionLogsToFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "ferret.log")
	opts := NewDefaultOptions()
	opts.Logger.Level = zerolog.DebugLevel
	opts.Logger.LogOutput = logger.OutputFile
	opts.Logger.LogFilename = logPath

	out, err := Run(
		context.Background(),
		opts,
		source.NewAnonymous("LET printed = PRINT(\"hello\") RETURN 42"),
		nil,
	)

	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	assertRuntimeOutput(t, out, "42")

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(logData), "hello") {
		t.Fatalf("expected log file to contain execution log, got %q", string(logData))
	}
}

func TestBuiltinRunNoneLogOutputDiscardsExecutionLogs(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "ferret.log")
	opts := NewDefaultOptions()
	opts.Logger.Level = zerolog.DebugLevel
	opts.Logger.LogOutput = logger.OutputNone
	opts.Logger.LogFilename = logPath

	out, err := Run(
		context.Background(),
		opts,
		source.NewAnonymous("LET printed = PRINT(\"hello\") RETURN 42"),
		nil,
	)

	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	assertRuntimeOutput(t, out, "42")

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("expected no log file, got stat error %v", err)
	}
}

func assertRuntimeOutput(t *testing.T, output io.ReadCloser, expected string) {
	t.Helper()

	defer output.Close()

	data, err := io.ReadAll(output)
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(string(data)) != expected {
		t.Fatalf("expected %s, got %q", expected, string(data))
	}
}
