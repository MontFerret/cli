package module

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testModulePackage = "example.com/ferret/archive"

func TestRewriteCompositionAddsModuleOptionAndImport(t *testing.T) {
	target := parseTestComposition(t, `package main

import "github.com/MontFerret/ferret/v2"

func main() {
	_, _ = ferret.New()
}
`)

	rewrite, err := rewriteComposition(target, "acme/archive", testModulePackage, map[string]struct{}{testModulePackage: {}})
	if err != nil {
		t.Fatal(err)
	}
	if !rewrite.Changed || rewrite.Registered {
		t.Fatalf("unexpected rewrite result: %#v", rewrite)
	}

	output := string(rewrite.Source)
	if !strings.Contains(output, `"example.com/ferret/archive"`) || !strings.Contains(output, "ferret.WithModules(ferretmod_acme_archive_") || !strings.Contains(output, ".New())") {
		t.Fatalf("unexpected rewrite:\n%s", output)
	}
}

func TestRewriteCompositionSupportsNamedFerretImport(t *testing.T) {
	target := parseTestComposition(t, `package main

import core "github.com/MontFerret/ferret/v2"

func main() {
	_, _ = core.New(core.WithFSRoot("."))
}
`)

	rewrite, err := rewriteComposition(target, "acme/archive", testModulePackage, map[string]struct{}{testModulePackage: {}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rewrite.Source), "core.WithModules(") {
		t.Fatalf("unexpected rewrite:\n%s", rewrite.Source)
	}
}

func TestRewriteCompositionCopiesSpreadOptions(t *testing.T) {
	target := parseTestComposition(t, `package main

import "github.com/MontFerret/ferret/v2"

func build(options []ferret.Option) {
	_, _ = ferret.New(options...)
}
`)

	rewrite, err := rewriteComposition(target, "acme/archive", testModulePackage, map[string]struct{}{testModulePackage: {}})
	if err != nil {
		t.Fatal(err)
	}

	output := string(rewrite.Source)
	if !strings.Contains(output, "append([]ferret.Option(nil), options...)") || !strings.Contains(output, "ferret.WithModules(") || !strings.Contains(output, "ferret.New(append(") {
		t.Fatalf("unexpected spread rewrite:\n%s", output)
	}
}

func TestRewriteCompositionIsIdempotentForRegisteredPackage(t *testing.T) {
	target := parseTestComposition(t, `package main

import (
	"github.com/MontFerret/ferret/v2"
	archive "example.com/ferret/archive"
)

func main() {
	_, _ = ferret.New(ferret.WithModules(archive.New()))
}
`)

	rewrite, err := rewriteComposition(target, "acme/archive", testModulePackage, map[string]struct{}{testModulePackage: {}})
	if err != nil {
		t.Fatal(err)
	}
	if rewrite.Changed || !rewrite.Registered || string(rewrite.Source) != string(target.Source) {
		t.Fatalf("unexpected idempotent rewrite: %#v", rewrite)
	}
}

func TestRewriteCompositionRejectsHistoricalPackagePath(t *testing.T) {
	target := parseTestComposition(t, `package main

import (
	"github.com/MontFerret/ferret/v2"
	archive "example.com/ferret/archive-old"
)

func main() {
	_, _ = ferret.New(ferret.WithModules(archive.New()))
}
`)

	_, err := rewriteComposition(target, "acme/archive", testModulePackage, map[string]struct{}{
		testModulePackage:                {},
		"example.com/ferret/archive-old": {},
	})
	if err == nil || !strings.Contains(err.Error(), "migrate the existing registration manually") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRewriteCompositionRejectsDeterministicAliasCollision(t *testing.T) {
	target := parseTestComposition(t, `package main

import "github.com/MontFerret/ferret/v2"

var ferretmod_acme_archive_2594ef6375cf = 1

func main() {
	_, _ = ferret.New()
}
`)

	_, err := rewriteComposition(target, "acme/archive", testModulePackage, map[string]struct{}{testModulePackage: {}})
	if err == nil || !strings.Contains(err.Error(), "collides with a declaration") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInspectCompositionFileReportsFerretDotImport(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "main.go")
	if err := os.WriteFile(filename, []byte(`package main
import . "github.com/MontFerret/ferret/v2"
func main() { New() }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	matches, dotImport, err := inspectCompositionFile(filename, "example.com/app")
	if err != nil {
		t.Fatal(err)
	}
	if !dotImport || len(matches) != 0 {
		t.Fatalf("unexpected dot-import discovery: dot=%v matches=%d", dotImport, len(matches))
	}
}

func TestDiscoverCompositionRejectsZeroAndMultipleCalls(t *testing.T) {
	// File-level inspection is sufficient to prove call identity; project enumeration is covered by installer fixtures.
	directory := t.TempDir()
	filename := filepath.Join(directory, "main.go")
	if err := os.WriteFile(filename, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	matches, _, err := inspectCompositionFile(filename, "example.com/app")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no matches, got %d", len(matches))
	}

	if err := os.WriteFile(filename, []byte(`package main
import "github.com/MontFerret/ferret/v2"
func one() { ferret.New() }
func two() { ferret.New() }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	matches, _, err = inspectCompositionFile(filename, "example.com/app")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected two matches, got %d", len(matches))
	}
}

func BenchmarkRewriteComposition(b *testing.B) {
	source := `package main
import "github.com/MontFerret/ferret/v2"
func build(options []ferret.Option) { ferret.New(options...) }
`

	for b.Loop() {
		target := parseBenchmarkComposition(b, source)
		if _, err := rewriteComposition(target, "acme/archive", testModulePackage, map[string]struct{}{testModulePackage: {}}); err != nil {
			b.Fatal(err)
		}
	}
}

func parseTestComposition(t *testing.T, source string) *composition {
	t.Helper()
	directory := t.TempDir()
	filename := filepath.Join(directory, "main.go")
	if err := os.WriteFile(filename, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	matches, dotImport, err := inspectCompositionFile(filename, "example.com/app")
	if err != nil {
		t.Fatal(err)
	}
	if dotImport || len(matches) != 1 {
		t.Fatalf("expected one normal Ferret composition, dot=%v matches=%d", dotImport, len(matches))
	}

	return matches[0]
}

func parseBenchmarkComposition(b *testing.B, source string) *composition {
	b.Helper()
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "main.go", source, parser.ParseComments)
	if err != nil {
		b.Fatal(err)
	}

	var call *ast.CallExpr
	ast.Inspect(file, func(node ast.Node) bool {
		candidate, ok := node.(*ast.CallExpr)
		if ok && isSelectorCall(candidate, "ferret", "New") {
			call = candidate
		}
		return true
	})
	if call == nil {
		b.Fatal("composition call not found")
	}

	return &composition{
		Filename: "main.go", Directory: ".", Package: "example.com/app", Source: []byte(source),
		Mode: 0o644, File: file, FileSet: fileSet, Call: call, CoreAlias: "ferret",
	}
}
