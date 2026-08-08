package testutil

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func WriteQuery(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func CaptureStdout(t *testing.T, fn func() error) (string, error) {
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

func CaptureStderr(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	original := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stderr = writer

	runErr := fn()

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	os.Stderr = original

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if closeErr := reader.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	return string(data), runErr
}

func WithStdinBytes(t *testing.T, data []byte, fn func()) {
	t.Helper()

	original := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	os.Stdin = reader
	defer func() {
		os.Stdin = original
		reader.Close()
	}()

	fn()
}

func WithDevNullStdin(t *testing.T, fn func()) {
	t.Helper()

	original := os.Stdin
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}

	os.Stdin = stdin
	defer func() {
		os.Stdin = original
		stdin.Close()
	}()

	fn()
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	return cmd
}
