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
	before := "return [shift(union(a, b)), append(a, v, true)]"
	want := "return [arrays::slice(arrays::concat(a, b), 1), append(a, v, true)]"
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

		if !strings.Contains(stderr, "query.fql:1: append(...)") || !strings.Contains(stderr, "existing duplicates") {
			t.Fatalf("missing manual warning in mode %s: %s", mode, stderr)
		}

		if mode == "--print" {
			if !strings.HasPrefix(stdout, "--- a/query.fql\n+++ b/query.fql\n") ||
				!strings.Contains(stdout, "+"+want) || strings.Contains(stdout, "Manual follow-up") ||
				strings.Contains(stdout, "Scanned") || !strings.Contains(stderr, "Scanned 1 FQL file") {
				t.Fatalf("print mixed diff and diagnostics: stdout=%q stderr=%q", stdout, stderr)
			}
		} else if !strings.Contains(stdout, "query.fql") || strings.Contains(stdout, "append(...)") {
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

	if stdout != "" || !strings.Contains(stderr, "append(...)") || !strings.Contains(stderr, "No safe automatic changes available") {
		t.Fatalf("manual-only second run: stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestMigrateRunStdlibHelp(t *testing.T) {
	stdout, _, err := executeMigrateCommand(t, new(fakeMigrationService), "run", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, text := range []string{"safe legacy stdlib", "encoding, crypto, path, arrays, object, datetime, math, and random", "local function or use alias", "Range and zero-argument rand"} {
		if !strings.Contains(stdout, text) {
			t.Fatalf("help does not contain %q: %s", text, stdout)
		}
	}
}

func TestMigrateCheckStdlibReportsFindingsWithoutWriting(t *testing.T) {
	tests := []struct {
		name, input, diagnostic string
	}{
		{
			name:  "has replacement",
			input: `return has({ foo: "bar" }, "baz")`,
			diagnostic: ":1:8: Legacy stdlib call `has` should use `object::has_key`.\n" +
				"  help: Preview automatic replacements with `ferret migrate run --print`.\n\n",
		},
		{
			name:  "manual review only",
			input: "return average(xs)",
			diagnostic: ":1:8: Stdlib call `average` needs manual review.\n" +
				"  help: legacy aggregate behavior is permissive for heterogeneous collections; " +
				"the strict math API requires a separate semantic migration\n\n",
		},
		{
			name:  "composed replacement",
			input: "return keys(obj, true)",
			diagnostic: ":1:8: Legacy stdlib call `keys` should use `arrays::sorted(object::keys(...))`.\n" +
				"  help: Preview automatic replacements with `ferret migrate run --print`. The final literal mode argument is removed.\n\n",
		},
		{
			name:  "parameterized rand remains manual",
			input: "return rand(10)",
			diagnostic: ":1:8: Stdlib call `rand` needs manual review.\n" +
				"  help: parameterized legacy rand uses a historical rounded/floored range calculation (max/2 to max*2 for one argument) and has no behavior-preserving canonical call\n\n",
		},
		{
			name:  "canonical source",
			input: `return object::has_key({ foo: "bar" }, "baz")`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "query.fql")
			if err := os.WriteFile(path, []byte(test.input), 0o600); err != nil {
				t.Fatal(err)
			}

			workingDirectory, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}

			displayPath, err := filepath.Rel(workingDirectory, path)
			if err != nil {
				t.Fatal(err)
			}

			for range 2 {
				stdout, stderr, err := executeMigrateCommand(t, migration.New(nil), "check", path)
				if test.diagnostic == "" {
					if err != nil || stderr != "" || stdout != "✓ No v1 compatibility issues found in 1 FQL file.\n" {
						t.Fatalf("unexpected clean check: stdout=%q stderr=%q err=%v", stdout, stderr, err)
					}
				} else if err == nil || err.Error() != "Found 1 v1 compatibility issue in 1 of 1 FQL file." ||
					stdout != "" || stderr != filepath.ToSlash(displayPath)+test.diagnostic {
					t.Fatalf("unexpected check finding: stdout=%q stderr=%q err=%v", stdout, stderr, err)
				}

				contents, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}

				if string(contents) != test.input {
					t.Fatalf("check changed source: %q", contents)
				}
			}
		})
	}
}
