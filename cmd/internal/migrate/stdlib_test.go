package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MontFerret/cli/v2/internal/migration"
)

func TestMigrateRunStdlibPreviewAndApply(t *testing.T) {
	path := filepath.Join(t.TempDir(), "query.fql")
	before := "return [abs(-2), average(xs)]"
	want := "return [math::abs(-2), average(xs)]"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}

	service := migration.New(nil)
	for _, mode := range []string{"--dry-run", "--print", "apply"} {
		args := []string{"run", path}
		if mode != "apply" {
			args = append(args, mode)
		}

		stdout, stderr, err := executeMigrateCommand(t, service, args...)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(stderr, "query.fql:1: average(...)") || !strings.Contains(stderr, "heterogeneous") {
			t.Fatalf("missing manual warning in mode %s: %s", mode, stderr)
		}

		if mode == "--print" {
			if !strings.HasPrefix(stdout, "--- a/query.fql\n+++ b/query.fql\n") ||
				!strings.Contains(stdout, "+"+want) || strings.Contains(stdout, "Manual follow-up") ||
				strings.Contains(stdout, "Scanned") || !strings.Contains(stderr, "Scanned 1 FQL file") {
				t.Fatalf("print mixed diff and diagnostics: stdout=%q stderr=%q", stdout, stderr)
			}
		} else if !strings.Contains(stdout, "query.fql") || strings.Contains(stdout, "average(...)") {
			t.Fatalf("unexpected human output: %s", stdout)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		expected := before
		if mode == "apply" {
			expected = want
		}

		if string(contents) != expected {
			t.Fatalf("mode %s contents = %q, want %q", mode, contents, expected)
		}
	}

	stdout, stderr, err := executeMigrateCommand(t, service, "run", "--print", path)
	if err != nil {
		t.Fatal(err)
	}

	if stdout != "" || !strings.Contains(stderr, "average(...)") || !strings.Contains(stderr, "No safe automatic changes available") {
		t.Fatalf("manual-only second run: stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestMigrateRunStdlibHelp(t *testing.T) {
	stdout, _, err := executeMigrateCommand(t, new(fakeMigrationService), "run", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, text := range []string{"safe legacy stdlib", "encoding, crypto, path, object, datetime, and math", "local function or use alias", "Arrays and rand/range"} {
		if !strings.Contains(stdout, text) {
			t.Fatalf("help does not contain %q: %s", text, stdout)
		}
	}
}
