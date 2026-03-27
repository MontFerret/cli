package run

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2/pkg/compiler"
	"github.com/MontFerret/ferret/v2/pkg/file"

	"github.com/MontFerret/cli/pkg/build"
	cliruntime "github.com/MontFerret/cli/pkg/runtime"
)

func TestResolveInput_SourceFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")

	writeQuery(t, input, "RETURN 42")

	resolved, err := ResolveInput("", []string{input})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved == nil || resolved.Source == nil {
		t.Fatal("expected source input")
	}

	if resolved.Source.Content() != "RETURN 42" {
		t.Fatalf("unexpected source content: %q", resolved.Source.Content())
	}
}

func TestResolveInput_RejectsMultipleArgs(t *testing.T) {
	resolved, err := ResolveInput("", []string{"first.fql", "second.fql"})

	if err == nil {
		t.Fatal("expected error")
	}

	if resolved != nil {
		t.Fatalf("expected nil input, got %#v", resolved)
	}

	if !strings.Contains(err.Error(), "at most one file argument") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveInput_StdinSource(t *testing.T) {
	withStdinBytes(t, []byte("RETURN 42\n"), func() {
		resolved, err := ResolveInput("", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resolved == nil || resolved.Source == nil {
			t.Fatal("expected source input")
		}

		if resolved.Source.Name() != "stdin" {
			t.Fatalf("unexpected source name: %q", resolved.Source.Name())
		}

		if resolved.Source.Content() != "RETURN 42\n" {
			t.Fatalf("unexpected source content: %q", resolved.Source.Content())
		}
	})
}

func TestResolveInput_StdinArtifact(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	outputPath := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 42")
	buildArtifact(t, input, outputPath)
	data := readFile(t, outputPath)

	withStdinBytes(t, data, func() {
		resolved, err := ResolveInput("", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resolved == nil || len(resolved.Artifact) == 0 {
			t.Fatal("expected artifact input")
		}
	})
}

func TestExecute_CompiledArtifact(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")

	writeQuery(t, input, "RETURN 42")
	buildArtifact(t, input, filepath.Join(dir, "query.fqlc"))

	resolved, err := ResolveInput("", []string{filepath.Join(dir, "query.fqlc")})

	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}

	output, err := Execute(context.Background(), cliruntime.NewDefaultOptions(), nil, resolved)

	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}

	assertOutput(t, output, "42")
}

func TestExecute_CompiledArtifactCustomName(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	outputPath := filepath.Join(dir, "compiled.bin")

	writeQuery(t, input, "RETURN 42")
	buildArtifact(t, input, outputPath)

	resolved, err := ResolveInput("", []string{outputPath})

	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}

	output, err := Execute(context.Background(), cliruntime.NewDefaultOptions(), nil, resolved)

	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}

	assertOutput(t, output, "42")
}

func TestExecute_CompiledArtifactWithParams(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	outputPath := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN @value")
	buildArtifact(t, input, outputPath)

	resolved, err := ResolveInput("", []string{outputPath})

	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}

	output, err := Execute(context.Background(), cliruntime.NewDefaultOptions(), map[string]any{"value": float64(99)}, resolved)

	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}

	assertOutput(t, output, "99")
}

func TestExecute_CompiledArtifactFromStdin(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	outputPath := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 42")
	buildArtifact(t, input, outputPath)
	data := readFile(t, outputPath)

	withStdinBytes(t, data, func() {
		resolved, err := ResolveInput("", nil)

		if err != nil {
			t.Fatalf("unexpected resolve error: %v", err)
		}

		output, err := Execute(context.Background(), cliruntime.NewDefaultOptions(), nil, resolved)

		if err != nil {
			t.Fatalf("unexpected execute error: %v", err)
		}

		assertOutput(t, output, "42")
	})
}

func TestResolveInput_PlainTextFQLCIsSource(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 7")

	resolved, err := ResolveInput("", []string{input})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved == nil || resolved.Source == nil {
		t.Fatal("expected source input")
	}

	if resolved.Source.Content() != "RETURN 7" {
		t.Fatalf("unexpected source content: %q", resolved.Source.Content())
	}
}

func TestExecute_CorruptArtifactReturnsLoadError(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "broken.bin")

	if err := os.WriteFile(input, []byte("FBC2"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveInput("", []string{input})

	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}

	_, err = Execute(context.Background(), cliruntime.NewDefaultOptions(), nil, resolved)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "bytecode artifact") {
		t.Fatalf("expected artifact load error, got %v", err)
	}
}

func TestExecute_CorruptArtifactFromStdinReturnsLoadError(t *testing.T) {
	withStdinBytes(t, []byte("FBC2"), func() {
		resolved, err := ResolveInput("", nil)

		if err != nil {
			t.Fatalf("unexpected resolve error: %v", err)
		}

		_, err = Execute(context.Background(), cliruntime.NewDefaultOptions(), nil, resolved)

		if err == nil {
			t.Fatal("expected error")
		}

		if !strings.Contains(err.Error(), "bytecode artifact") {
			t.Fatalf("expected artifact load error, got %v", err)
		}
	})
}

func TestResolveInput_NoStdinReturnsNil(t *testing.T) {
	withDevNullStdin(t, func() {
		resolved, err := ResolveInput("", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resolved != nil {
			t.Fatalf("expected nil input, got %#v", resolved)
		}
	})
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

func withStdinBytes(t *testing.T, data []byte, fn func()) {
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

func withDevNullStdin(t *testing.T, fn func()) {
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

func buildArtifact(t *testing.T, inputPath, outputPath string) {
	t.Helper()

	src := readSource(t, inputPath)

	if err := build.WriteArtifact(nilCompiler(), src, outputPath); err != nil {
		t.Fatalf("build artifact: %v", err)
	}
}

func readSource(t *testing.T, path string) *file.Source {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return file.NewSource(path, string(data))
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func assertOutput(t *testing.T, output io.ReadCloser, expected string) {
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

func nilCompiler() *compiler.Compiler {
	return compiler.New()
}
