package migration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MontFerret/cli/v2/internal/goproject"
)

func TestMigratorAppliesStandaloneFQLWithoutGoToolchain(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "query.fql")
	before := "FOR x IN 1..3 RETURN x"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}

	migrator, runner := newFixtureMigrator()
	for _, mode := range []Mode{ModeDryRun, ModePrint} {
		preview, err := migrator.Migrate(context.Background(), Options{Path: path, Mode: mode})
		if err != nil {
			t.Fatal(err)
		}

		if preview.Applied || preview.ScannedFiles != 0 || preview.ScannedFQLFiles != 1 ||
			preview.MigratedFQLFiles != 1 || len(preview.Changes) != 1 || preview.GoModPath != "" {
			t.Fatalf("unexpected standalone preview for mode %d: %#v", mode, preview)
		}

		if preview.Changes[0].Path != "query.fql" || readMigrationFixture(t, path) != before {
			t.Fatalf("standalone preview changed source or reported the wrong path: %#v", preview.Changes)
		}
	}

	result, err := migrator.Migrate(context.Background(), Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Applied || result.MigratedFQLFiles != 1 {
		t.Fatalf("unexpected standalone migration: %#v", result)
	}

	if got := readMigrationFixture(t, path); got != "return for x in 1..3 {\n    return x\n}" {
		t.Fatalf("unexpected migrated FQL:\n%s", got)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("standalone FQL mode = %o, want 600", info.Mode().Perm())
	}

	second, err := migrator.Migrate(context.Background(), Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}

	if second.Applied || second.MigratedFQLFiles != 0 || len(second.Changes) != 0 {
		t.Fatalf("standalone migration was not idempotent: %#v", second)
	}

	if runner.runCalls != 0 || runner.getCalls != 0 {
		t.Fatalf("standalone FQL migration invoked Go tooling: runs=%d gets=%d", runner.runCalls, runner.getCalls)
	}
}

func TestMigratorAppliesFQLOnlyDirectoryWithoutGoToolchain(t *testing.T) {
	tests := []struct {
		name  string
		goMod string
	}{
		{name: "without_go_mod"},
		{name: "with_go_mod", goMod: "module example.com/fql-only\n\ngo 1.26.0\n"},
		{name: "with_malformed_go_mod", goMod: "module [\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.goMod != "" {
				if err := os.WriteFile(
					filepath.Join(root, "go.mod"),
					[]byte(tt.goMod),
					0o644,
				); err != nil {
					t.Fatal(err)
				}
			}

			writeMigrationTargetFile(t, root, "query.fql", "FOR x IN 1..3 RETURN x")
			writeMigrationTargetFile(t, root, "testdata/fixture.fql", "FOR x IN 1..3 RETURN x")
			writeMigrationTargetFile(t, root, ".hidden/query.fql", "FOR x IN 1..3 RETURN x")
			writeMigrationTargetFile(t, root, "_generated/query.fql", "FOR x IN 1..3 RETURN x")
			writeMigrationTargetFile(t, root, "vendor/dependency/query.fql", "FOR x IN 1..3 RETURN x")
			writeMigrationTargetFile(t, root, "node_modules/dependency/query.fql", "FOR x IN 1..3 RETURN x")
			writeMigrationTargetFile(t, root, "nested/go.mod", "module example.com/nested\n")
			writeMigrationTargetFile(t, root, "nested/query.fql", "FOR x IN 1..3 RETURN x")
			writeMigrationTargetFile(t, root, "testdata/helper.go", "package fixture\n")

			before := snapshotMigrationFixture(t, root)
			migrator, runner := newFixtureMigrator()
			result, err := migrator.Migrate(context.Background(), Options{Path: root})
			if err != nil {
				t.Fatal(err)
			}

			if !result.Applied || result.ScannedFiles != 0 || result.ScannedFQLFiles != 1 ||
				result.MigratedFQLFiles != 1 || result.GoModPath != "" || result.DependenciesChanged ||
				result.VendorDetected {
				t.Fatalf("unexpected FQL-only migration: %#v", result)
			}

			if runner.runCalls != 0 || runner.getCalls != 0 {
				t.Fatalf("FQL-only directory invoked Go tooling: runs=%d gets=%d", runner.runCalls, runner.getCalls)
			}

			if got := readMigrationFixture(t, filepath.Join(root, "query.fql")); !strings.HasPrefix(got, "return for") {
				t.Fatalf("included FQL was not migrated:\n%s", got)
			}

			for path, content := range before {
				if path == "query.fql" {
					continue
				}

				if got := []byte(readMigrationFixture(t, filepath.Join(root, filepath.FromSlash(path)))); !bytes.Equal(got, content) {
					t.Fatalf("excluded or metadata file %s changed:\nbefore:\n%s\nafter:\n%s", path, content, got)
				}
			}
		})
	}
}

func TestMigratorDoesNotFollowDirectorySymlinks(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	writeMigrationTargetFile(t, root, "query.fql", "FOR x IN 1..3 RETURN x")
	writeMigrationTargetFile(t, external, "linked.fql", "FOR x IN 1..3 RETURN x")
	externalBefore := readMigrationFixture(t, filepath.Join(external, "linked.fql"))

	if err := os.Symlink(external, filepath.Join(root, "linked")); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}

	migrator, runner := newFixtureMigrator()
	result, err := migrator.Migrate(context.Background(), Options{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Applied || result.ScannedFQLFiles != 1 || result.MigratedFQLFiles != 1 {
		t.Fatalf("unexpected symlink-directory result: %#v", result)
	}

	if got := readMigrationFixture(t, filepath.Join(external, "linked.fql")); got != externalBefore {
		t.Fatalf("FQL under a directory symlink changed:\n%s", got)
	}

	if runner.runCalls != 0 {
		t.Fatalf("FQL-only symlink scan invoked Go tooling %d times", runner.runCalls)
	}
}

func TestMigratorAcceptsEmptyFQLOnlyDirectory(t *testing.T) {
	migrator, runner := newFixtureMigrator()
	result, err := migrator.Migrate(context.Background(), Options{Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	if result.Applied || result.ScannedFiles != 0 || result.ScannedFQLFiles != 0 || len(result.Changes) != 0 {
		t.Fatalf("unexpected empty-directory result: %#v", result)
	}

	if runner.runCalls != 0 {
		t.Fatalf("empty FQL-only directory invoked Go tooling %d times", runner.runCalls)
	}
}

func TestMigratorRejectsGoSourceWithoutModuleBeforeFQLMutation(t *testing.T) {
	root := t.TempDir()
	writeMigrationTargetFile(t, root, "main.go", `package app
import "github.com/MontFerret/ferret"
`)
	writeMigrationTargetFile(t, root, "query.fql", "FOR x IN 1..3 RETURN x")
	before := snapshotMigrationFixture(t, root)

	migrator, runner := newFixtureMigrator()
	_, err := migrator.Migrate(context.Background(), Options{Path: root})
	if !errors.Is(err, goproject.ErrNoModule) || !strings.Contains(err.Error(), "Go source was found") {
		t.Fatalf("unexpected missing-module error: %v", err)
	}

	assertMigrationFixtureUnchanged(t, root, before)
	if runner.runCalls != 0 {
		t.Fatalf("module-less Go source invoked Go tooling %d times", runner.runCalls)
	}
}

func TestMigratorReportsMalformedStandaloneFQL(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "broken.fql")
	before := "FOR x IN RETURN x"
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}

	migrator, runner := newFixtureMigrator()
	result, err := migrator.Migrate(context.Background(), Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}

	if result.Applied || result.ScannedFQLFiles != 1 || len(result.Changes) != 0 || len(result.ManualActions) != 1 {
		t.Fatalf("unexpected malformed standalone result: %#v", result)
	}

	if result.ManualActions[0].Path != "broken.fql" || readMigrationFixture(t, path) != before {
		t.Fatalf("malformed standalone source changed or reported the wrong path: %#v", result.ManualActions)
	}

	if runner.runCalls != 0 {
		t.Fatalf("malformed standalone FQL invoked Go tooling %d times", runner.runCalls)
	}
}

func TestMigratorDirectoryInsideModuleTargetsContainingModule(t *testing.T) {
	root := writeMigrationFixture(t, `module example.com/app

go 1.26.0
`, map[string]string{
		"main.go":           "package app\n",
		"root.fql":          "FOR x IN 1..3 RETURN x",
		"scripts/query.fql": "FOR x IN 1..3 RETURN x",
	})

	migrator, _ := newFixtureMigrator()
	result, err := migrator.Migrate(context.Background(), Options{Path: filepath.Join(root, "scripts")})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Applied || result.ScannedFiles != 1 || result.ScannedFQLFiles != 2 || result.MigratedFQLFiles != 2 {
		t.Fatalf("unexpected containing-module migration: %#v", result)
	}

	for _, path := range []string{"root.fql", "scripts/query.fql"} {
		if got := readMigrationFixture(t, filepath.Join(root, filepath.FromSlash(path))); !strings.HasPrefix(got, "return for") {
			t.Fatalf("module FQL %s was not migrated:\n%s", path, got)
		}
	}
}

func TestMigratorRejectsInvalidAndSymlinkTargets(t *testing.T) {
	root := t.TempDir()
	migrator, _ := newFixtureMigrator()

	missing := filepath.Join(root, "missing")
	if _, err := migrator.Migrate(context.Background(), Options{Path: missing}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected missing-target error: %v", err)
	}

	upper := filepath.Join(root, "query.FQL")
	if err := os.WriteFile(upper, []byte("return 1"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := migrator.Migrate(context.Background(), Options{Path: upper}); err == nil || !strings.Contains(err.Error(), "must use the .fql extension") {
		t.Fatalf("unexpected extension error: %v", err)
	}

	source := filepath.Join(root, "query.fql")
	if err := os.WriteFile(source, []byte("return 1"), 0o644); err != nil {
		t.Fatal(err)
	}

	linkedFile := filepath.Join(root, "linked.fql")
	if err := os.Symlink(source, linkedFile); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	if _, err := migrator.Migrate(context.Background(), Options{Path: linkedFile}); err == nil || !strings.Contains(err.Error(), "symlink target") {
		t.Fatalf("unexpected file-symlink error: %v", err)
	}

	directory := filepath.Join(root, "source")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}

	linkedDirectory := filepath.Join(root, "linked-directory")
	if err := os.Symlink(directory, linkedDirectory); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}

	if _, err := migrator.Migrate(context.Background(), Options{Path: linkedDirectory}); err == nil || !strings.Contains(err.Error(), "symlink target") {
		t.Fatalf("unexpected directory-symlink error: %v", err)
	}
}

func TestMigratorHonorsCancellationForFQLOnlyTarget(t *testing.T) {
	root := t.TempDir()
	writeMigrationTargetFile(t, root, "query.fql", "FOR x IN 1..3 RETURN x")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	migrator, runner := newFixtureMigrator()
	_, err := migrator.Migrate(ctx, Options{Path: root})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}

	if runner.runCalls != 0 {
		t.Fatalf("canceled FQL-only migration invoked Go tooling %d times", runner.runCalls)
	}
}

func BenchmarkDiscoverFQLOnlyMigrationProject(b *testing.B) {
	root := b.TempDir()
	for index := range 100 {
		path := filepath.Join(root, fmt.Sprintf("query_%03d.fql", index))
		if err := os.WriteFile(path, []byte("RETURN 1\n"), 0o644); err != nil {
			b.Fatal(err)
		}
	}

	runner := new(fixtureGoRunner)
	b.ResetTimer()

	for range b.N {
		project, err := discoverMigrationProject(context.Background(), runner, root)
		if err != nil {
			b.Fatal(err)
		}

		if len(project.GoFiles) != 0 || len(project.FQLFiles) != 100 {
			b.Fatalf("unexpected discovered project: %#v", project)
		}
	}

	if runner.runCalls != 0 {
		b.Fatalf("FQL-only discovery invoked Go tooling %d times", runner.runCalls)
	}
}

func writeMigrationTargetFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
