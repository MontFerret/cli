package migration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMigratorStdlibStandaloneModes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "query.fql")
	before := "// keep location\nreturn [abs(-2), keys(obj, true)]"
	want := "// keep location\nreturn [math::abs(-2), keys(obj, true)]"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}

	migrator, runner := newFixtureMigrator()
	var actions []ManualAction
	for _, mode := range []Mode{ModeDryRun, ModePrint, ModeApply} {
		result, err := migrator.Migrate(context.Background(), Options{Path: path, Mode: mode})
		if err != nil {
			t.Fatal(err)
		}

		if result.Applied != (mode == ModeApply) || result.ScannedFQLFiles != 1 || result.MigratedFQLFiles != 1 ||
			len(result.Changes) != 1 || len(result.ManualActions) != 1 || result.DependenciesChanged {
			t.Fatalf("unexpected mode %d result: %#v", mode, result)
		}

		if string(result.Changes[0].After) != want || result.ManualActions[0].Line != 2 || result.ManualActions[0].Path != "query.fql" {
			t.Fatalf("unexpected migration contents: %#v", result)
		}

		if actions != nil && !reflect.DeepEqual(actions, result.ManualActions) {
			t.Fatalf("manual reports changed between modes: %#v", result.ManualActions)
		}

		actions = result.ManualActions
		expected := before
		if mode == ModeApply {
			expected = want
		}

		if got := readMigrationFixture(t, path); got != expected {
			t.Fatalf("mode %d contents = %q, want %q", mode, got, expected)
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}

	second, err := migrator.Migrate(context.Background(), Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}

	if second.Applied || len(second.Changes) != 0 || !reflect.DeepEqual(second.ManualActions, actions) {
		t.Fatalf("second run changed source or lost manual actions: %#v", second)
	}

	if runner.runCalls != 0 || runner.getCalls != 0 {
		t.Fatalf("FQL-only migration used Go tooling: %#v", runner)
	}
}

func TestMigratorStdlibDirectoryReportingAndExclusions(t *testing.T) {
	root := writeMigrationFixture(t, "module example.com/app\n\ngo 1.26.0\n", map[string]string{
		"main.go":                "package app\n",
		"go.sum":                 "preserve dependencies\n",
		"a_mixed.fql":            "let obj = json_parse(body)\nreturn [date_diff(a, b, unit), average(xs)]",
		"b_manual.fql":           "RETURN join(a, b)",
		"c_broken.fql":           "return abs(1) + (",
		"d_valid.fql":            "return sha1(body)",
		"e_comments.fql":         "return sha1 /* preserve me */ (body)",
		"vendor/query.fql":       "return abs(1)",
		"testdata/query.fql":     "return abs(1)",
		"node_modules/query.fql": "return abs(1)",
		".hidden/query.fql":      "return abs(1)",
		"_generated/query.fql":   "return abs(1)",
		"upper.FQL":              "return abs(1)",
		"nested/go.mod":          "module example.com/nested\n",
		"nested/query.fql":       "return abs(1)",
	})
	before := snapshotMigrationFixture(t, root)
	migrator, runner := newFixtureMigrator()
	result, err := migrator.Migrate(context.Background(), Options{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Applied || result.ScannedFQLFiles != 5 || result.MigratedFQLFiles != 2 ||
		len(result.ManualActions) != 5 || result.DependenciesChanged || runner.getCalls != 0 {
		t.Fatalf("unexpected directory migration: %#v", result)
	}

	wantActions := []struct {
		path, detail string
		line         int
	}{
		{"a_mixed.fql", "average(...)", 2},
		{"a_mixed.fql", "date_diff(...)", 2},
		{"b_manual.fql", "join(...)", 1},
		{"c_broken.fql", "", 1},
		{"e_comments.fql", "format migrated Ferret source: formatter did not preserve comments", 1},
	}
	for i, want := range wantActions {
		got := result.ManualActions[i]
		if got.Path != want.path || got.Line != want.line || (want.detail != "" && got.Detail != want.detail) || got.Reason == "" {
			t.Fatalf("manual action %d = %#v, want %#v", i, got, want)
		}
	}

	for path, contents := range before {
		if path == "a_mixed.fql" || path == "d_valid.fql" {
			continue
		}

		if got := readMigrationFixture(t, filepath.Join(root, path)); got != string(contents) {
			t.Fatalf("unrelated, excluded, manual-only or malformed file %s changed", path)
		}
	}
}

func TestMigratorStdlibRollsBackOnCommitFailure(t *testing.T) {
	root := t.TempDir()
	writeMigrationTargetFile(t, root, "a.fql", "return abs(-1)")
	writeMigrationTargetFile(t, root, "b.fql", "return sha1(body)")
	before := snapshotMigrationFixture(t, root)
	originalRename := renameMigrationFile
	calls := 0
	renameMigrationFile = func(oldPath, newPath string) error {
		calls++
		if calls == 2 {
			return errors.New("stdlib commit failed")
		}

		return os.Rename(oldPath, newPath)
	}
	t.Cleanup(func() { renameMigrationFile = originalRename })

	migrator, _ := newFixtureMigrator()
	_, err := migrator.Migrate(context.Background(), Options{Path: root})
	if err == nil || !strings.Contains(err.Error(), "stdlib commit failed") {
		t.Fatalf("unexpected commit error: %v", err)
	}

	if after := snapshotMigrationFixture(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("migration did not restore original files: %#v", after)
	}
}

func TestCheckFQLCompatibilityDoesNotIncludeStdlibRules(t *testing.T) {
	root := t.TempDir()
	writeMigrationTargetFile(t, root, "query.fql", "return [abs(-1), average(xs), join(a, b)]")
	result, err := new(Migrator).CheckCompatibility(context.Background(), CompatibilityOptions{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	if result.ScannedFiles != 1 || len(result.Diagnostics) != 0 {
		t.Fatalf("stdlib migration expanded check scope: %#v", result)
	}
}

func BenchmarkPlanFQLStdlibSourceChanges(b *testing.B) {
	for _, canonical := range []bool{false, true} {
		name := "legacy"
		content := "let obj = json_parse(body)\nlet digest = sha1(json_stringify(obj))\nreturn [abs(-2), keys(obj), base(path), date_year(date(text))]"
		if canonical {
			name = "canonical"
			content = "let obj = encoding::json_parse(body)\nlet digest = crypto::sha1(encoding::json_stringify(obj))\nreturn [math::abs(-2), object::keys(obj), path::base(path), datetime::year(datetime::parse(text))]"
		}

		b.Run(name, func(b *testing.B) {
			root := b.TempDir()
			files := make([]string, 100)
			for i := range files {
				files[i] = filepath.Join(root, fmt.Sprintf("query_%03d.fql", i))
				if err := os.WriteFile(files[i], []byte(content), 0o644); err != nil {
					b.Fatal(err)
				}
			}

			project := &migrationProject{Root: root, FQLFiles: files}
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				result, err := planFQLSourceChanges(context.Background(), project)
				if err != nil {
					b.Fatal(err)
				}

				want := len(files)
				if canonical {
					want = 0
				}

				if result.MigratedFiles != want {
					b.Fatalf("migrated files = %d, want %d", result.MigratedFiles, want)
				}
			}
		})
	}
}
