package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2/pkg/bytecode/artifact"
)

func TestRunBuild_DefaultOutputPath(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")

	writeQuery(t, input, "RETURN 42")

	stderr, err := captureStderr(t, func() error {
		return runBuild([]string{input}, "")
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stderr != "" {
		t.Fatalf("expected no stderr output, got %q", stderr)
	}

	output := filepath.Join(dir, "query.fqlc")
	assertArtifactSource(t, output, "RETURN 42")
}

func TestRunBuild_SingleFileExplicitOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "compiled.bin")

	writeQuery(t, input, "RETURN 42")

	if _, err := captureStderr(t, func() error {
		return runBuild([]string{input}, output)
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 42")

	if _, err := os.Stat(filepath.Join(dir, "query.fqlc")); !os.IsNotExist(err) {
		t.Fatalf("expected sibling artifact to be absent, stat err=%v", err)
	}
}

func TestRunBuild_SingleFileOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	outputDir := filepath.Join(dir, "dist")

	writeQuery(t, input, "RETURN 42")

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := captureStderr(t, func() error {
		return runBuild([]string{input}, outputDir)
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, filepath.Join(outputDir, "query.fqlc"), "RETURN 42")
}

func TestRunBuild_MultiFileOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "first.fql")
	inputB := filepath.Join(dir, "second")
	outputDir := filepath.Join(dir, "dist")

	writeQuery(t, inputA, "RETURN 1")
	writeQuery(t, inputB, "RETURN 2")

	if _, err := captureStderr(t, func() error {
		return runBuild([]string{inputA, inputB}, outputDir)
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, filepath.Join(outputDir, "first.fqlc"), "RETURN 1")
	assertArtifactSource(t, filepath.Join(outputDir, "second.fqlc"), "RETURN 2")
}

func TestRunBuild_MultiFileOutputMustBeDirectory(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "first.fql")
	inputB := filepath.Join(dir, "second.fql")
	output := filepath.Join(dir, "artifact.fqlc")

	writeQuery(t, inputA, "RETURN 1")
	writeQuery(t, inputB, "RETURN 2")
	writeQuery(t, output, "not a directory")

	_, err := captureStderr(t, func() error {
		return runBuild([]string{inputA, inputB}, output)
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "--output must be a directory when building multiple files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBuild_MultiFileOutputCollision(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "one", "query.fql")
	inputB := filepath.Join(dir, "two", "query.fql")
	outputDir := filepath.Join(dir, "dist")

	writeQuery(t, inputA, "RETURN 1")
	writeQuery(t, inputB, "RETURN 2")

	_, err := captureStderr(t, func() error {
		return runBuild([]string{inputA, inputB}, outputDir)
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "output collision") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBuild_RejectsOverwritingSource(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	query := "RETURN 42"

	writeQuery(t, input, query)

	stderr, err := captureStderr(t, func() error {
		return runBuild([]string{input}, input)
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "1 of 1 scripts failed to build") {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stderr, "would overwrite source file") {
		t.Fatalf("expected overwrite message, got %q", stderr)
	}

	content, readErr := os.ReadFile(input)
	if readErr != nil {
		t.Fatal(readErr)
	}

	if string(content) != query {
		t.Fatalf("expected source file to remain unchanged, got %q", string(content))
	}
}

func TestRunBuild_InvalidQueryDoesNotCreateArtifact(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "broken.fql")
	output := filepath.Join(dir, "broken.fqlc")

	writeQuery(t, input, "FOR item IN")

	_, err := captureStderr(t, func() error {
		return runBuild([]string{input}, "")
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "1 of 1 scripts failed to build") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("expected artifact to be absent, stat err=%v", statErr)
	}
}

func TestRunBuild_MixedMultiFileBuildContinues(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.fql")
	invalid := filepath.Join(dir, "invalid.fql")
	outputDir := filepath.Join(dir, "dist")

	writeQuery(t, valid, "RETURN 1")
	writeQuery(t, invalid, "FOR item IN")

	_, err := captureStderr(t, func() error {
		return runBuild([]string{valid, invalid}, outputDir)
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "1 of 2 scripts failed to build") {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, filepath.Join(outputDir, "valid.fqlc"), "RETURN 1")

	if _, statErr := os.Stat(filepath.Join(outputDir, "invalid.fqlc")); !os.IsNotExist(statErr) {
		t.Fatalf("expected invalid artifact to be absent, stat err=%v", statErr)
	}
}

func TestRunBuild_ReplacesExistingDestinationFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 1")

	if _, err := captureStderr(t, func() error {
		return runBuild([]string{input}, "")
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 1")

	writeQuery(t, input, "RETURN 2")

	if _, err := captureStderr(t, func() error {
		return runBuild([]string{input}, "")
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 2")
}

func writeQuery(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertArtifactSource(t *testing.T, path, expected string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	program, err := artifact.Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal artifact: %v", err)
	}

	if program.Source == nil {
		t.Fatal("expected serialized source")
	}

	if program.Source.Content() != expected {
		t.Fatalf("expected source %q, got %q", expected, program.Source.Content())
	}
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
