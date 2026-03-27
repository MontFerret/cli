package cmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func writeQuery(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
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

func captureStderr(t *testing.T, fn func() error) (string, error) {
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
