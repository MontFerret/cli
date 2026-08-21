package migration

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCheckFQLCompatibilityReportsIssuesAndContinuesAfterMalformedSource(t *testing.T) {
	root := writeCompatibilityFixture(t, map[string]string{
		"a_clean.fql": "return 42",
		"b_broken.fql": `let ok = 1
for value in
    return value`,
		"c_unicode.fql": `let label = "привет"
for value in 1..3
    return { label, value }`,
	})
	before := compatibilityFixtureContents(t, root)

	result, err := new(Migrator).CheckCompatibility(context.Background(), CompatibilityOptions{
		Path: root,
		From: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.ScannedFiles != 3 || len(result.Diagnostics) != 2 {
		t.Fatalf("unexpected compatibility result: %#v", result)
	}

	broken := result.Diagnostics[0]
	if broken.Kind != CompatibilityDiagnosticFailure ||
		broken.Path != compatibilityTestDisplayPath(t, filepath.Join(root, "b_broken.fql")) ||
		broken.Line < 2 || broken.Column < 1 ||
		!strings.HasPrefix(broken.Message, "Could not check v1 compatibility:") ||
		broken.Help != checkFailureHelp {
		t.Fatalf("unexpected malformed-source diagnostic: %#v", broken)
	}

	issue := result.Diagnostics[1]
	if issue.Kind != CompatibilityDiagnosticIssue ||
		issue.Path != compatibilityTestDisplayPath(t, filepath.Join(root, "c_unicode.fql")) ||
		issue.Line != 2 || issue.Column != 1 ||
		issue.Message != finalForMessage || issue.Help != finalForHelp {
		t.Fatalf("unexpected compatibility issue: %#v", issue)
	}

	after := compatibilityFixtureContents(t, root)
	if !maps.Equal(before, after) {
		t.Fatalf("compatibility check modified source:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestCheckFQLCompatibilityScansAllUserFQL(t *testing.T) {
	issue := "for value in 1..3 return value"
	root := writeCompatibilityFixture(t, map[string]string{
		"query.fql":                         issue,
		"testdata/fixture.fql":              issue,
		".hidden/query.fql":                 issue,
		"_generated/query.fql":              issue,
		"nested/go.mod":                     "module example.com/nested\n",
		"nested/query.fql":                  issue,
		"vendor/dependency/query.fql":       issue,
		"node_modules/dependency/query.fql": issue,
		".git/query.fql":                    issue,
		".hg/query.fql":                     issue,
		".svn/query.fql":                    issue,
		"upper.FQL":                         issue,
	})

	result, err := new(Migrator).CheckCompatibility(context.Background(), CompatibilityOptions{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	if result.ScannedFiles != 5 || len(result.Diagnostics) != 5 {
		t.Fatalf("unexpected recursive scan result: %#v", result)
	}

	wantPaths := []string{
		compatibilityTestDisplayPath(t, filepath.Join(root, ".hidden", "query.fql")),
		compatibilityTestDisplayPath(t, filepath.Join(root, "_generated", "query.fql")),
		compatibilityTestDisplayPath(t, filepath.Join(root, "nested", "query.fql")),
		compatibilityTestDisplayPath(t, filepath.Join(root, "query.fql")),
		compatibilityTestDisplayPath(t, filepath.Join(root, "testdata", "fixture.fql")),
	}
	slices.Sort(wantPaths)

	gotPaths := make([]string, len(result.Diagnostics))
	for index, diagnostic := range result.Diagnostics {
		gotPaths[index] = diagnostic.Path
	}

	if !slices.Equal(gotPaths, wantPaths) {
		t.Fatalf("checked paths = %#v, want %#v", gotPaths, wantPaths)
	}
}

func TestCheckFQLCompatibilityAcceptsOneFileWithoutGoModule(t *testing.T) {
	root := writeCompatibilityFixture(t, map[string]string{
		"query.fql": "for value in 1..3 return value",
	})
	path := filepath.Join(root, "query.fql")

	result, err := new(Migrator).CheckCompatibility(context.Background(), CompatibilityOptions{Path: path})
	if err != nil {
		t.Fatal(err)
	}

	if result.ScannedFiles != 1 || len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Path != compatibilityTestDisplayPath(t, path) {
		t.Fatalf("unexpected single-file result: %#v", result)
	}
}

func TestCheckFQLCompatibilityFollowsFileSymlinksButNotDirectorySymlinks(t *testing.T) {
	root := writeCompatibilityFixture(t, map[string]string{
		"source/query.fql": "for value in 1..3 return value",
	})
	if err := os.Symlink(filepath.Join(root, "source", "query.fql"), filepath.Join(root, "linked.fql")); err != nil {
		t.Skipf("file symlinks are unavailable: %v", err)
	}

	if err := os.Symlink(filepath.Join(root, "source"), filepath.Join(root, "linked-source")); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}

	result, err := new(Migrator).CheckCompatibility(context.Background(), CompatibilityOptions{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	if result.ScannedFiles != 2 || len(result.Diagnostics) != 2 {
		t.Fatalf("unexpected symlink scan result: %#v", result)
	}

	wantPaths := []string{
		compatibilityTestDisplayPath(t, filepath.Join(root, "linked.fql")),
		compatibilityTestDisplayPath(t, filepath.Join(root, "source", "query.fql")),
	}
	for index, diagnostic := range result.Diagnostics {
		if diagnostic.Path != wantPaths[index] {
			t.Fatalf("diagnostic %d path = %q, want %q", index, diagnostic.Path, wantPaths[index])
		}
	}
}

func TestCheckFQLCompatibilityHandlesEmptyAndInvalidTargets(t *testing.T) {
	root := t.TempDir()

	result, err := new(Migrator).CheckCompatibility(context.Background(), CompatibilityOptions{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	if result.ScannedFiles != 0 || len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected empty-directory result: %#v", result)
	}

	nonFQL := filepath.Join(root, "query.FQL")
	if err := os.WriteFile(nonFQL, []byte("return 1"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := new(Migrator).CheckCompatibility(
		context.Background(),
		CompatibilityOptions{Path: nonFQL},
	); err == nil || !strings.Contains(err.Error(), "must use the .fql extension") {
		t.Fatalf("unexpected non-FQL error: %v", err)
	}

	if _, err := new(Migrator).CheckCompatibility(
		context.Background(),
		CompatibilityOptions{Path: filepath.Join(root, "missing")},
	); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected missing-target error: %v", err)
	}

	if _, err := new(Migrator).CheckCompatibility(
		context.Background(),
		CompatibilityOptions{Path: root, From: "v2"},
	); err == nil || !strings.Contains(err.Error(), "expected v1") {
		t.Fatalf("unexpected source-version error: %v", err)
	}
}

func TestCheckFQLCompatibilityHonorsCancellation(t *testing.T) {
	root := writeCompatibilityFixture(t, map[string]string{"query.fql": "return 1"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := new(Migrator).CheckCompatibility(ctx, CompatibilityOptions{Path: root})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func BenchmarkCheckFQLCompatibility(b *testing.B) {
	root := b.TempDir()
	for index := range 100 {
		path := filepath.Join(root, fmt.Sprintf("query_%03d.fql", index))
		content := fmt.Sprintf(`let seed = %d
for value in 1..100 {
    filter value > seed
    return value * 2
}`, index)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for range b.N {
		result, err := new(Migrator).CheckCompatibility(context.Background(), CompatibilityOptions{Path: root})
		if err != nil {
			b.Fatal(err)
		}

		if result.ScannedFiles != 100 || len(result.Diagnostics) != 100 {
			b.Fatalf("unexpected benchmark result: %#v", result)
		}
	}
}

func writeCompatibilityFixture(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func compatibilityFixtureContents(t *testing.T, root string) map[string]string {
	t.Helper()

	result := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		result[path] = string(data)

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func compatibilityTestDisplayPath(t *testing.T, filename string) string {
	t.Helper()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	path, err := compatibilityDisplayPath(workingDirectory, filename)
	if err != nil {
		t.Fatal(err)
	}

	return path
}
