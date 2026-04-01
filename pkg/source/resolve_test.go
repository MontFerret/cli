package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MontFerret/cli/pkg/source"
)

func TestResolve_Eval(t *testing.T) {
	input := source.Input{
		Eval: "RETURN 1",
	}

	sources, err := source.Resolve(input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	if sources[0].Name() != "<eval>" {
		t.Errorf("expected name '<eval>', got %q", sources[0].Name())
	}
}

func TestResolve_FileArgs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.fql")

	if err := os.WriteFile(path, []byte("RETURN 42"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := source.Input{
		Args: []string{path},
	}

	sources, err := source.Resolve(input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	if sources[0].Name() != path {
		t.Errorf("expected name %q, got %q", path, sources[0].Name())
	}
}

func TestResolve_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	paths := make([]string, 3)

	for i := range paths {
		p := filepath.Join(dir, "test"+string(rune('0'+i))+".fql")
		if err := os.WriteFile(p, []byte("RETURN 1"), 0o644); err != nil {
			t.Fatal(err)
		}

		paths[i] = p
	}

	sources, err := source.Resolve(source.Input{Args: paths})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}
}

func TestResolve_FileNotFound(t *testing.T) {
	input := source.Input{
		Args: []string{"/nonexistent/source.fql"},
	}

	_, err := source.Resolve(input)

	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestResolve_NoInput(t *testing.T) {
	// When stdin is a terminal (not piped), and no args/eval, returns nil
	// We can only reliably test this in a terminal context.
	// In test context, stdin is typically not a character device,
	// so this test verifies the stdin-pipe path doesn't error.
	input := source.Input{}

	sources, err := source.Resolve(input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In test context, stdin behavior varies - just verify no crash
	_ = sources
}
